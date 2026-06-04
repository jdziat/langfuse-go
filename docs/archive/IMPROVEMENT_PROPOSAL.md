# Langfuse Go SDK Improvement Proposal

This document proposes solutions for the issues identified in the critical review of the langfuse-go SDK.

## Table of Contents

1. [Critical Issues](#critical-issues)
2. [Design Improvements](#design-improvements)
3. [Missing Features](#missing-features)
4. [API Consistency](#api-consistency)
5. [Implementation Roadmap](#implementation-roadmap)

---

## Critical Issues

### 1. Goroutine Leak Prevention

**Problem**: If `Shutdown()` is never called, background goroutines run forever.

**Solution**: Implement a lifecycle manager with multiple safeguards.

```go
// lifecycle.go

// ClientState represents the current state of the client lifecycle.
type ClientState int

const (
    ClientStateActive ClientState = iota
    ClientStateShuttingDown
    ClientStateClosed
)

// LifecycleManager handles client lifecycle with leak prevention.
type LifecycleManager struct {
    mu           sync.RWMutex
    state        ClientState
    wg           sync.WaitGroup
    ctx          context.Context
    cancel       context.CancelFunc

    // Leak detection
    createdAt    time.Time
    lastActivity time.Time
    warningOnce  sync.Once
}

// NewLifecycleManager creates a new lifecycle manager.
// maxIdleTime triggers a warning log if no activity occurs within this duration.
func NewLifecycleManager(maxIdleTime time.Duration, logger Logger) *LifecycleManager {
    ctx, cancel := context.WithCancel(context.Background())
    lm := &LifecycleManager{
        state:        ClientStateActive,
        ctx:          ctx,
        cancel:       cancel,
        createdAt:    time.Now(),
        lastActivity: time.Now(),
    }

    // Start idle detection goroutine
    if maxIdleTime > 0 && logger != nil {
        go lm.idleDetector(maxIdleTime, logger)
    }

    return lm
}

func (lm *LifecycleManager) idleDetector(maxIdle time.Duration, logger Logger) {
    ticker := time.NewTicker(maxIdle)
    defer ticker.Stop()

    for {
        select {
        case <-lm.ctx.Done():
            return
        case <-ticker.C:
            lm.mu.RLock()
            idle := time.Since(lm.lastActivity)
            state := lm.state
            lm.mu.RUnlock()

            if state == ClientStateActive && idle > maxIdle {
                lm.warningOnce.Do(func() {
                    logger.Printf("WARNING: Langfuse client has been idle for %v without Shutdown() being called. "+
                        "This may indicate a goroutine leak. Call client.Shutdown(ctx) when done.", idle)
                })
            }
        }
    }
}

// RecordActivity updates the last activity timestamp.
func (lm *LifecycleManager) RecordActivity() {
    lm.mu.Lock()
    lm.lastActivity = time.Now()
    lm.mu.Unlock()
}

// State returns the current client state.
func (lm *LifecycleManager) State() ClientState {
    lm.mu.RLock()
    defer lm.mu.RUnlock()
    return lm.state
}
```

**Configuration Option**:
```go
// WithIdleWarning enables warnings when the client is idle without shutdown.
// This helps detect goroutine leaks in development/testing.
func WithIdleWarning(duration time.Duration) ConfigOption {
    return func(c *Config) {
        c.IdleWarningDuration = duration
    }
}
```

**Priority**: HIGH
**Effort**: Medium

---

### 2. Validated Builder Pattern

**Problem**: Validation errors in fluent builders can be silently ignored.

**Solution**: Implement a `ValidatedBuilder` that makes errors impossible to ignore.

```go
// validated_builder.go

// BuildResult wraps a result with its validation state.
// This pattern forces callers to handle validation.
type BuildResult[T any] struct {
    value T
    err   error
}

// Unwrap returns the value and error, forcing error handling.
func (r BuildResult[T]) Unwrap() (T, error) {
    return r.value, r.err
}

// Must returns the value or panics if there's an error.
// Use only in tests or when validation is guaranteed.
func (r BuildResult[T]) Must() T {
    if r.err != nil {
        panic(r.err)
    }
    return r.value
}

// ValidatedTraceBuilder wraps TraceBuilder with compile-time validation enforcement.
type ValidatedTraceBuilder struct {
    builder *TraceBuilder
    errors  []error
}

// Name sets the trace name with immediate validation.
func (b *ValidatedTraceBuilder) Name(name string) *ValidatedTraceBuilder {
    if err := ValidateName("name", name, MaxNameLength); err != nil {
        b.errors = append(b.errors, err)
    } else {
        b.builder.Name(name)
    }
    return b
}

// Tags sets the trace tags with immediate validation.
func (b *ValidatedTraceBuilder) Tags(tags []string) *ValidatedTraceBuilder {
    if len(tags) > MaxTagCount {
        b.errors = append(b.errors, NewValidationError("tags",
            fmt.Sprintf("exceeds maximum count of %d", MaxTagCount)))
    }
    if err := ValidateTags("tags", tags); err != nil {
        b.errors = append(b.errors, err)
    }
    if len(b.errors) == 0 {
        b.builder.Tags(tags)
    }
    return b
}

// Create creates the trace, returning a BuildResult that must be unwrapped.
func (b *ValidatedTraceBuilder) Create(ctx context.Context) BuildResult[*Trace] {
    if len(b.errors) > 0 {
        return BuildResult[*Trace]{err: combineErrors(b.errors)}
    }
    trace, err := b.builder.Create(ctx)
    return BuildResult[*Trace]{value: trace, err: err}
}

// Strict returns a validated builder that enforces validation.
func (c *Client) NewTraceStrict() *ValidatedTraceBuilder {
    return &ValidatedTraceBuilder{
        builder: c.NewTrace(),
        errors:  make([]error, 0),
    }
}
```

**Alternative**: Validation mode toggle
```go
// WithStrictValidation enables immediate validation errors.
// When enabled, builder setters will store errors that are checked at Create().
func WithStrictValidation() ConfigOption {
    return func(c *Config) {
        c.StrictValidation = true
    }
}
```

**Priority**: MEDIUM
**Effort**: High (affects all builders)

---

### 3. Robust UUID Generation

**Problem**: Timestamp fallback could cause ID collisions under high load.

**Solution**: Use atomic counter fallback and panic on crypto failure in production.

```go
// id.go

import (
    "crypto/rand"
    "fmt"
    "sync/atomic"
    "time"
)

var (
    // fallbackCounter provides uniqueness when combined with timestamp
    fallbackCounter uint64

    // cryptoFailures tracks crypto/rand failures for monitoring
    cryptoFailures atomic.Int64
)

// IDGenerationMode controls how IDs are generated on crypto failure.
type IDGenerationMode int

const (
    // IDModeStrict panics on crypto/rand failure (recommended for production)
    IDModeStrict IDGenerationMode = iota

    // IDModeFallback uses atomic counter fallback (suitable for testing)
    IDModeFallback
)

// IDGenerator generates unique IDs with configurable failure handling.
type IDGenerator struct {
    mode    IDGenerationMode
    metrics Metrics
}

// NewIDGenerator creates an ID generator with the specified mode.
func NewIDGenerator(mode IDGenerationMode, metrics Metrics) *IDGenerator {
    return &IDGenerator{mode: mode, metrics: metrics}
}

// Generate creates a new unique ID.
func (g *IDGenerator) Generate() (string, error) {
    id, err := g.generateUUID()
    if err == nil {
        return id, nil
    }

    // Track crypto failure
    cryptoFailures.Add(1)
    if g.metrics != nil {
        g.metrics.IncrementCounter("langfuse.id.crypto_failures", 1)
    }

    switch g.mode {
    case IDModeStrict:
        return "", fmt.Errorf("langfuse: crypto/rand failed and strict mode is enabled: %w", err)
    case IDModeFallback:
        return g.generateFallbackID(), nil
    default:
        return "", fmt.Errorf("langfuse: unknown ID generation mode")
    }
}

func (g *IDGenerator) generateUUID() (string, error) {
    b := make([]byte, 16)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    b[6] = (b[6] & 0x0f) | 0x40
    b[8] = (b[8] & 0x3f) | 0x80
    return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

func (g *IDGenerator) generateFallbackID() string {
    // Combine timestamp with atomic counter for uniqueness
    counter := atomic.AddUint64(&fallbackCounter, 1)
    now := time.Now()
    // Format: timestamp-counter-pid for maximum uniqueness
    return fmt.Sprintf("%d-%016x-%d", now.UnixNano(), counter, pid)
}

// MustGenerate generates an ID or panics. Use only when ID generation must succeed.
func (g *IDGenerator) MustGenerate() string {
    id, err := g.Generate()
    if err != nil {
        panic(err)
    }
    return id
}
```

**Configuration**:
```go
// WithIDGenerationMode sets the ID generation failure mode.
// Default is IDModeFallback for backwards compatibility.
// Production deployments should use IDModeStrict.
func WithIDGenerationMode(mode IDGenerationMode) ConfigOption {
    return func(c *Config) {
        c.IDGenerationMode = mode
    }
}
```

**Priority**: HIGH
**Effort**: Low

---

### 4. Hook Classification System

**Problem**: All hooks are treated equally; a logging hook failure shouldn't abort requests.

**Solution**: Introduce hook categories with different failure semantics.

```go
// hooks.go

// HookPriority determines how hook failures are handled.
type HookPriority int

const (
    // HookPriorityObservational - failures are logged but don't abort the request.
    // Use for logging, metrics, tracing hooks.
    HookPriorityObservational HookPriority = iota

    // HookPriorityCritical - failures abort the request.
    // Use for authentication, authorization, request signing hooks.
    HookPriorityCritical
)

// ClassifiedHook wraps an HTTPHook with priority information.
type ClassifiedHook struct {
    Hook     HTTPHook
    Priority HookPriority
    Name     string // For error messages and logging
}

// HookChain manages multiple hooks with different priorities.
type HookChain struct {
    hooks  []ClassifiedHook
    logger Logger
}

// NewHookChain creates a new hook chain.
func NewHookChain(logger Logger) *HookChain {
    return &HookChain{
        hooks:  make([]ClassifiedHook, 0),
        logger: logger,
    }
}

// Add adds a hook with the specified priority.
func (hc *HookChain) Add(name string, hook HTTPHook, priority HookPriority) {
    hc.hooks = append(hc.hooks, ClassifiedHook{
        Hook:     hook,
        Priority: priority,
        Name:     name,
    })
}

// BeforeRequest calls all hooks' BeforeRequest methods.
// Returns error only if a critical hook fails.
func (hc *HookChain) BeforeRequest(ctx context.Context, req *http.Request) error {
    for _, ch := range hc.hooks {
        if err := ch.Hook.BeforeRequest(ctx, req); err != nil {
            switch ch.Priority {
            case HookPriorityCritical:
                return fmt.Errorf("langfuse: critical hook %q failed: %w", ch.Name, err)
            case HookPriorityObservational:
                if hc.logger != nil {
                    hc.logger.Printf("langfuse: observational hook %q failed (continuing): %v", ch.Name, err)
                }
            }
        }
    }
    return nil
}

// AfterResponse calls all hooks' AfterResponse methods.
// Errors are logged but never returned (response already received).
func (hc *HookChain) AfterResponse(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error) {
    for _, ch := range hc.hooks {
        func() {
            defer func() {
                if r := recover(); r != nil {
                    if hc.logger != nil {
                        hc.logger.Printf("langfuse: hook %q panicked in AfterResponse: %v", ch.Name, r)
                    }
                }
            }()
            ch.Hook.AfterResponse(ctx, req, resp, duration, err)
        }()
    }
}

// Convenience constructors for pre-built hooks

// ObservationalLoggingHook creates a logging hook that won't abort requests.
func ObservationalLoggingHook(logger Logger) ClassifiedHook {
    return ClassifiedHook{
        Hook:     LoggingHook(logger),
        Priority: HookPriorityObservational,
        Name:     "logging",
    }
}

// CriticalAuthHook creates an auth hook that aborts on failure.
func CriticalAuthHook(authFunc func(*http.Request) error) ClassifiedHook {
    return ClassifiedHook{
        Hook: &funcHook{
            before: func(ctx context.Context, req *http.Request) error {
                return authFunc(req)
            },
        },
        Priority: HookPriorityCritical,
        Name:     "auth",
    }
}
```

**Configuration**:
```go
// WithObservationalHook adds a hook that won't abort requests on failure.
func WithObservationalHook(name string, hook HTTPHook) ConfigOption {
    return func(c *Config) {
        c.HookChain.Add(name, hook, HookPriorityObservational)
    }
}

// WithCriticalHook adds a hook that aborts requests on failure.
func WithCriticalHook(name string, hook HTTPHook) ConfigOption {
    return func(c *Config) {
        c.HookChain.Add(name, hook, HookPriorityCritical)
    }
}
```

**Priority**: MEDIUM
**Effort**: Medium

---

## Design Improvements

### 5. Structured Async Error Handling

**Problem**: Async errors have inconsistent handling and limited visibility.

**Solution**: Implement a structured error channel with typed errors.

```go
// async_errors.go

// AsyncError represents an error that occurred in background processing.
type AsyncError struct {
    Time      time.Time
    Operation string      // "batch_send", "flush", "hook", etc.
    EventIDs  []string    // IDs of affected events, if known
    Err       error
    Retryable bool
    Context   map[string]any // Additional context
}

func (e *AsyncError) Error() string {
    return fmt.Sprintf("langfuse async error [%s] at %s: %v",
        e.Operation, e.Time.Format(time.RFC3339), e.Err)
}

func (e *AsyncError) Unwrap() error {
    return e.Err
}

// AsyncErrorHandler provides structured async error handling.
type AsyncErrorHandler struct {
    // Channel for receiving async errors (buffered)
    Errors chan *AsyncError

    // Callbacks
    onError     func(*AsyncError)
    onOverflow  func(dropped int)

    // Metrics
    metrics     Metrics

    // Internal
    mu          sync.Mutex
    dropped     int64
    bufferSize  int
}

// NewAsyncErrorHandler creates a handler with the specified buffer size.
func NewAsyncErrorHandler(bufferSize int, metrics Metrics) *AsyncErrorHandler {
    return &AsyncErrorHandler{
        Errors:     make(chan *AsyncError, bufferSize),
        bufferSize: bufferSize,
        metrics:    metrics,
    }
}

// Handle processes an async error.
func (h *AsyncErrorHandler) Handle(err *AsyncError) {
    // Try to send to channel
    select {
    case h.Errors <- err:
        // Sent successfully
    default:
        // Channel full - track dropped errors
        atomic.AddInt64(&h.dropped, 1)
        if h.metrics != nil {
            h.metrics.IncrementCounter("langfuse.async_errors.dropped", 1)
        }
    }

    // Call callback if set
    if h.onError != nil {
        h.onError(err)
    }

    // Update metrics
    if h.metrics != nil {
        h.metrics.IncrementCounter("langfuse.async_errors.total", 1)
        h.metrics.IncrementCounter(
            fmt.Sprintf("langfuse.async_errors.%s", err.Operation), 1)
    }
}

// SetCallback sets the error callback.
func (h *AsyncErrorHandler) SetCallback(fn func(*AsyncError)) {
    h.mu.Lock()
    h.onError = fn
    h.mu.Unlock()
}

// DroppedCount returns the number of dropped errors.
func (h *AsyncErrorHandler) DroppedCount() int64 {
    return atomic.LoadInt64(&h.dropped)
}

// Drain returns all pending errors from the channel.
func (h *AsyncErrorHandler) Drain() []*AsyncError {
    var errors []*AsyncError
    for {
        select {
        case err := <-h.Errors:
            errors = append(errors, err)
        default:
            return errors
        }
    }
}
```

**Configuration**:
```go
// WithAsyncErrorChannel enables async error channel for programmatic handling.
// The returned channel receives all async errors (up to bufferSize).
func WithAsyncErrorChannel(bufferSize int) ConfigOption {
    return func(c *Config) {
        c.AsyncErrorHandler = NewAsyncErrorHandler(bufferSize, c.Metrics)
    }
}

// WithAsyncErrorCallback sets a callback for async errors.
func WithAsyncErrorCallback(fn func(*AsyncError)) ConfigOption {
    return func(c *Config) {
        if c.AsyncErrorHandler == nil {
            c.AsyncErrorHandler = NewAsyncErrorHandler(100, c.Metrics)
        }
        c.AsyncErrorHandler.SetCallback(fn)
    }
}
```

**Priority**: MEDIUM
**Effort**: Medium

---

### 6. Backpressure Signaling

**Problem**: No way to detect or respond to queue pressure.

**Solution**: Expose queue state and implement backpressure callbacks.

```go
// backpressure.go

// QueueState represents the current state of the event queue.
type QueueState struct {
    PendingEvents    int     // Events waiting to be batched
    QueuedBatches    int     // Batches waiting to be sent
    QueueCapacity    int     // Maximum queue capacity
    Utilization      float64 // 0.0 to 1.0
    BackpressureMode bool    // True when queue is near capacity
}

// BackpressureThreshold configures when backpressure mode activates.
type BackpressureThreshold struct {
    // ActivateAt is the utilization threshold to activate backpressure (e.g., 0.8)
    ActivateAt float64

    // DeactivateAt is the utilization threshold to deactivate (e.g., 0.5)
    // Should be lower than ActivateAt to prevent oscillation
    DeactivateAt float64
}

// DefaultBackpressureThreshold provides sensible defaults.
var DefaultBackpressureThreshold = BackpressureThreshold{
    ActivateAt:   0.8,
    DeactivateAt: 0.5,
}

// BackpressureCallback is called when backpressure state changes.
type BackpressureCallback func(state QueueState, entering bool)

// QueueMonitor monitors queue state and signals backpressure.
type QueueMonitor struct {
    threshold       BackpressureThreshold
    callback        BackpressureCallback
    metrics         Metrics

    mu              sync.RWMutex
    inBackpressure  bool
    lastState       QueueState
}

// NewQueueMonitor creates a queue monitor with the given configuration.
func NewQueueMonitor(threshold BackpressureThreshold, callback BackpressureCallback, metrics Metrics) *QueueMonitor {
    return &QueueMonitor{
        threshold: threshold,
        callback:  callback,
        metrics:   metrics,
    }
}

// Update updates the queue state and checks thresholds.
func (m *QueueMonitor) Update(pendingEvents, queuedBatches, queueCapacity int) {
    utilization := float64(queuedBatches) / float64(queueCapacity)

    state := QueueState{
        PendingEvents: pendingEvents,
        QueuedBatches: queuedBatches,
        QueueCapacity: queueCapacity,
        Utilization:   utilization,
    }

    m.mu.Lock()
    wasInBackpressure := m.inBackpressure

    // Check for state transition
    if !m.inBackpressure && utilization >= m.threshold.ActivateAt {
        m.inBackpressure = true
    } else if m.inBackpressure && utilization <= m.threshold.DeactivateAt {
        m.inBackpressure = false
    }

    state.BackpressureMode = m.inBackpressure
    m.lastState = state
    isInBackpressure := m.inBackpressure
    m.mu.Unlock()

    // Update metrics
    if m.metrics != nil {
        m.metrics.SetGauge("langfuse.queue.utilization", utilization)
        m.metrics.SetGauge("langfuse.queue.pending_events", float64(pendingEvents))
        m.metrics.SetGauge("langfuse.queue.batches", float64(queuedBatches))
        if isInBackpressure {
            m.metrics.SetGauge("langfuse.queue.backpressure", 1)
        } else {
            m.metrics.SetGauge("langfuse.queue.backpressure", 0)
        }
    }

    // Fire callback on state change
    if m.callback != nil && wasInBackpressure != isInBackpressure {
        m.callback(state, isInBackpressure)
    }
}

// State returns the current queue state.
func (m *QueueMonitor) State() QueueState {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.lastState
}

// IsInBackpressure returns true if backpressure mode is active.
func (m *QueueMonitor) IsInBackpressure() bool {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.inBackpressure
}
```

**Client Integration**:
```go
// QueueState returns the current queue state.
func (c *Client) QueueState() QueueState {
    c.mu.Lock()
    pendingEvents := len(c.pendingEvents)
    c.mu.Unlock()

    return QueueState{
        PendingEvents: pendingEvents,
        QueuedBatches: len(c.batchQueue),
        QueueCapacity: c.config.BatchQueueSize,
        Utilization:   float64(len(c.batchQueue)) / float64(c.config.BatchQueueSize),
    }
}

// IsUnderPressure returns true if the client is experiencing backpressure.
func (c *Client) IsUnderPressure() bool {
    return c.queueMonitor.IsInBackpressure()
}
```

**Configuration**:
```go
// WithBackpressureCallback sets a callback for backpressure state changes.
func WithBackpressureCallback(callback BackpressureCallback) ConfigOption {
    return func(c *Config) {
        c.BackpressureCallback = callback
    }
}

// WithBackpressureThreshold configures backpressure thresholds.
func WithBackpressureThreshold(activate, deactivate float64) ConfigOption {
    return func(c *Config) {
        c.BackpressureThreshold = BackpressureThreshold{
            ActivateAt:   activate,
            DeactivateAt: deactivate,
        }
    }
}
```

**Priority**: MEDIUM
**Effort**: Medium

---

### 7. Request-Level Timeouts

**Problem**: Timeouts are set at HTTP client level, not per-request.

**Solution**: Add per-request timeout support.

```go
// In http.go

// RequestOptions allows per-request configuration.
type RequestOptions struct {
    Timeout     time.Duration
    Priority    RequestPriority
    RetryPolicy *RetryPolicy // Override default retry for this request
}

// RequestPriority influences timeout and retry behavior.
type RequestPriority int

const (
    RequestPriorityNormal RequestPriority = iota
    RequestPriorityHigh   // Longer timeout, more retries
    RequestPriorityLow    // Shorter timeout, fewer retries
)

// doWithOptions executes a request with specific options.
func (h *httpClient) doWithOptions(ctx context.Context, req *request, opts *RequestOptions) error {
    // Apply per-request timeout if specified
    if opts != nil && opts.Timeout > 0 {
        var cancel context.CancelFunc
        ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
        defer cancel()
    }

    // Use custom retry policy if specified
    retryStrategy := h.retryStrategy
    if opts != nil && opts.RetryPolicy != nil {
        retryStrategy = opts.RetryPolicy.ToStrategy()
    }

    // Apply priority-based defaults
    if opts != nil {
        switch opts.Priority {
        case RequestPriorityHigh:
            if opts.Timeout == 0 {
                var cancel context.CancelFunc
                ctx, cancel = context.WithTimeout(ctx, h.defaultTimeout*2)
                defer cancel()
            }
        case RequestPriorityLow:
            if opts.Timeout == 0 {
                var cancel context.CancelFunc
                ctx, cancel = context.WithTimeout(ctx, h.defaultTimeout/2)
                defer cancel()
            }
        }
    }

    return h.doWithRetries(ctx, req, retryStrategy)
}
```

**Priority**: LOW
**Effort**: Low

---

## Missing Features

### 8. Enhanced SDK Observability

**Problem**: Limited visibility into SDK internals.

**Solution**: Comprehensive internal metrics.

```go
// metrics_internal.go

// InternalMetrics defines all SDK internal metrics.
type InternalMetrics struct {
    // Queue metrics
    QueueDepth           string // Gauge: current queue depth
    QueueCapacity        string // Gauge: queue capacity
    QueueUtilization     string // Gauge: queue utilization (0-1)

    // Batch metrics
    BatchSize            string // Histogram: events per batch
    BatchDuration        string // Histogram: time to send batch
    BatchRetries         string // Counter: retry attempts
    BatchSuccesses       string // Counter: successful batches
    BatchFailures        string // Counter: failed batches

    // Event metrics
    EventsQueued         string // Counter: events added to queue
    EventsSent           string // Counter: events successfully sent
    EventsDropped        string // Counter: events dropped

    // HTTP metrics
    HTTPRequestDuration  string // Histogram: request latency
    HTTPRequestRetries   string // Counter: HTTP retries
    HTTP2xx              string // Counter: 2xx responses
    HTTP4xx              string // Counter: 4xx responses
    HTTP5xx              string // Counter: 5xx responses

    // Circuit breaker metrics
    CircuitState         string // Gauge: 0=closed, 1=half-open, 2=open
    CircuitTrips         string // Counter: times circuit opened

    // Hook metrics
    HookDuration         string // Histogram: hook execution time
    HookFailures         string // Counter: hook failures

    // Lifecycle metrics
    ClientUptime         string // Gauge: seconds since creation
    ShutdownDuration     string // Histogram: shutdown time
}

// DefaultInternalMetrics returns metric names with "langfuse." prefix.
var DefaultInternalMetrics = InternalMetrics{
    QueueDepth:          "langfuse.queue.depth",
    QueueCapacity:       "langfuse.queue.capacity",
    QueueUtilization:    "langfuse.queue.utilization",
    BatchSize:           "langfuse.batch.size",
    BatchDuration:       "langfuse.batch.duration_ms",
    BatchRetries:        "langfuse.batch.retries",
    BatchSuccesses:      "langfuse.batch.successes",
    BatchFailures:       "langfuse.batch.failures",
    EventsQueued:        "langfuse.events.queued",
    EventsSent:          "langfuse.events.sent",
    EventsDropped:       "langfuse.events.dropped",
    HTTPRequestDuration: "langfuse.http.duration_ms",
    HTTPRequestRetries:  "langfuse.http.retries",
    HTTP2xx:             "langfuse.http.2xx",
    HTTP4xx:             "langfuse.http.4xx",
    HTTP5xx:             "langfuse.http.5xx",
    CircuitState:        "langfuse.circuit.state",
    CircuitTrips:        "langfuse.circuit.trips",
    HookDuration:        "langfuse.hook.duration_ms",
    HookFailures:        "langfuse.hook.failures",
    ClientUptime:        "langfuse.client.uptime_seconds",
    ShutdownDuration:    "langfuse.shutdown.duration_ms",
}

// MetricsRecorder wraps the Metrics interface with convenience methods.
type MetricsRecorder struct {
    metrics Metrics
    names   InternalMetrics
}

func NewMetricsRecorder(m Metrics) *MetricsRecorder {
    return &MetricsRecorder{
        metrics: m,
        names:   DefaultInternalMetrics,
    }
}

func (r *MetricsRecorder) RecordBatchSend(size int, duration time.Duration, success bool, retries int) {
    if r.metrics == nil {
        return
    }
    r.metrics.RecordDuration(r.names.BatchDuration, duration)
    r.metrics.IncrementCounter(r.names.EventsSent, int64(size))
    if success {
        r.metrics.IncrementCounter(r.names.BatchSuccesses, 1)
    } else {
        r.metrics.IncrementCounter(r.names.BatchFailures, 1)
    }
    if retries > 0 {
        r.metrics.IncrementCounter(r.names.BatchRetries, int64(retries))
    }
}

func (r *MetricsRecorder) RecordHTTPResponse(statusCode int, duration time.Duration) {
    if r.metrics == nil {
        return
    }
    r.metrics.RecordDuration(r.names.HTTPRequestDuration, duration)
    switch {
    case statusCode >= 200 && statusCode < 300:
        r.metrics.IncrementCounter(r.names.HTTP2xx, 1)
    case statusCode >= 400 && statusCode < 500:
        r.metrics.IncrementCounter(r.names.HTTP4xx, 1)
    case statusCode >= 500:
        r.metrics.IncrementCounter(r.names.HTTP5xx, 1)
    }
}
```

**Priority**: MEDIUM
**Effort**: Medium

---

## API Consistency

### 9. Standardized Method Naming

**Problem**: Inconsistent method names (`Create`, `Apply`, `End`).

**Solution**: Establish naming conventions.

```go
// PROPOSED NAMING CONVENTIONS
//
// Builder termination methods:
//   - Create(ctx) - Creates a new resource
//   - Update(ctx) - Modifies an existing resource (alias: Apply)
//   - Delete(ctx) - Removes a resource
//
// Observation lifecycle:
//   - End(ctx)           - End without additional data
//   - EndWithOutput(ctx) - End with output data
//   - EndWithError(ctx)  - End with error state
//
// Resource methods:
//   - Get(ctx, id)       - Retrieve single resource
//   - List(ctx, params)  - Retrieve multiple resources
//   - Delete(ctx, id)    - Remove resource
//
// Update builder methods:
//   - Apply(ctx)         - Apply updates (preferred)
//   - Update(ctx)        - Alias for Apply (deprecated in v2)

// Deprecation helper
func deprecated(oldName, newName string) {
    // Log deprecation warning once per method
}

// Example: Add Apply as alias, deprecate Update
func (b *TraceUpdateBuilder) Apply(ctx context.Context) error {
    return b.update(ctx)
}

// Update is deprecated, use Apply instead.
// Deprecated: Use Apply instead.
func (b *TraceUpdateBuilder) Update(ctx context.Context) error {
    deprecated("Update", "Apply")
    return b.update(ctx)
}
```

**Priority**: LOW
**Effort**: Low (mostly documentation and aliases)

---

### 10. Standardized Error Wrapping

**Problem**: Inconsistent error wrapping approaches.

**Solution**: Establish error wrapping guidelines and helpers.

```go
// errors_helpers.go

// Error wrapping guidelines:
// 1. Always use %w for wrappable errors
// 2. Include operation context in the message
// 3. Include relevant IDs when available
// 4. Use typed errors for categorization

// wrapHTTPError wraps an HTTP-related error with context.
func wrapHTTPError(err error, method, path string, requestID string) error {
    if err == nil {
        return nil
    }
    return fmt.Errorf("langfuse: %s %s failed (request_id=%s): %w", method, path, requestID, err)
}

// wrapValidationError wraps a validation error with field context.
func wrapValidationError(err error, operation, field string) error {
    if err == nil {
        return nil
    }
    return fmt.Errorf("langfuse: %s validation failed for %s: %w", operation, field, err)
}

// wrapQueueError wraps a queue operation error.
func wrapQueueError(err error, operation string, eventCount int) error {
    if err == nil {
        return nil
    }
    return fmt.Errorf("langfuse: %s failed (%d events affected): %w", operation, eventCount, err)
}

// ErrorChain provides fluent error building.
type ErrorChain struct {
    err     error
    context []string
}

func WrapErr(err error) *ErrorChain {
    return &ErrorChain{err: err}
}

func (e *ErrorChain) WithOp(op string) *ErrorChain {
    e.context = append(e.context, op)
    return e
}

func (e *ErrorChain) WithField(field string) *ErrorChain {
    e.context = append(e.context, "field="+field)
    return e
}

func (e *ErrorChain) WithID(id string) *ErrorChain {
    e.context = append(e.context, "id="+id)
    return e
}

func (e *ErrorChain) Error() error {
    if e.err == nil {
        return nil
    }
    ctx := strings.Join(e.context, ", ")
    return fmt.Errorf("langfuse [%s]: %w", ctx, e.err)
}
```

**Priority**: LOW
**Effort**: Medium (requires touching many files)

---

## Implementation Roadmap

### Phase 1: Critical Fixes (Week 1-2)

| Task | Priority | Effort | Files |
|------|----------|--------|-------|
| Goroutine leak prevention | HIGH | Medium | lifecycle.go, client.go |
| Robust UUID generation | HIGH | Low | id.go, ingestion.go |

### Phase 2: Design Improvements (Week 3-4)

| Task | Priority | Effort | Files |
|------|----------|--------|-------|
| Hook classification | MEDIUM | Medium | hooks.go |
| Async error handling | MEDIUM | Medium | async_errors.go, client.go |
| Backpressure signaling | MEDIUM | Medium | backpressure.go, client.go |

### Phase 3: Observability (Week 5-6)

| Task | Priority | Effort | Files |
|------|----------|--------|-------|
| Enhanced metrics | MEDIUM | Medium | metrics_internal.go |
| Validated builders | MEDIUM | High | validated_builder.go, *_builder.go |

### Phase 4: Polish (Week 7-8)

| Task | Priority | Effort | Files |
|------|----------|--------|-------|
| Request-level timeouts | LOW | Low | http.go |
| Method naming consistency | LOW | Low | *.go (deprecations) |
| Error wrapping consistency | LOW | Medium | errors_helpers.go, *.go |

---

## Migration Guide

### Breaking Changes

None of the proposed changes are breaking. All improvements are additive:

1. New configuration options (opt-in)
2. New methods alongside existing ones
3. Deprecation warnings before removal
4. Default behavior preserved

### Deprecation Timeline

| Feature | Deprecated In | Removed In |
|---------|---------------|------------|
| `Update()` method | v1.1 | v2.0 |
| `IsShutdownError()` | v1.0 | v2.0 |
| `IsCompilationError()` | v1.0 | v2.0 |

---

## Testing Strategy

### Unit Tests

Each new component should have comprehensive unit tests:

```go
func TestLifecycleManager_IdleWarning(t *testing.T) { ... }
func TestIDGenerator_Fallback(t *testing.T) { ... }
func TestHookChain_PriorityHandling(t *testing.T) { ... }
func TestBackpressure_StateTransitions(t *testing.T) { ... }
```

### Integration Tests

Add integration tests for critical paths:

```go
func TestClient_GracefulShutdown_UnderLoad(t *testing.T) { ... }
func TestClient_BackpressureRecovery(t *testing.T) { ... }
func TestClient_HookFailureIsolation(t *testing.T) { ... }
```

### Benchmark Tests

Add benchmarks for performance-sensitive code:

```go
func BenchmarkIDGeneration(b *testing.B) { ... }
func BenchmarkQueueEvent(b *testing.B) { ... }
func BenchmarkHookChain(b *testing.B) { ... }
```

---

## Conclusion

This proposal addresses the critical issues and design concerns identified in the review while maintaining backwards compatibility. The phased implementation approach allows for incremental delivery and testing.

Key principles:
1. **No breaking changes** - All improvements are additive
2. **Opt-in complexity** - Advanced features require explicit configuration
3. **Sensible defaults** - New features have production-ready defaults
4. **Observable by default** - Better metrics and error visibility

The estimated total effort is 6-8 weeks for full implementation, with critical fixes deliverable in the first 2 weeks.
