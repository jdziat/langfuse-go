# Proposal: Langfuse Go SDK Production Readiness

**Author:** Engineering Team
**Status:** Draft
**Created:** 2025-12-27
**Target Version:** v2.0.0

---

## Executive Summary

This proposal addresses critical issues identified during a comprehensive code review of the Langfuse Go SDK. The current implementation, while functionally sound, has several architectural gaps that could cause production incidents:

1. **Shutdown loses events** due to context cancellation ordering
2. **Backpressure system exists but isn't connected** to the main client
3. **Five different error types** with inconsistent interfaces
4. **No failure mode testing** for high-load scenarios
5. **Dead code accumulation** indicating incomplete features

This proposal outlines a phased approach to address these issues while maintaining backward compatibility where possible.

---

## Table of Contents

1. [Phase 1: Critical Fixes (Breaking)](#phase-1-critical-fixes)
2. [Phase 2: Integration & Consolidation](#phase-2-integration--consolidation)
3. [Phase 3: Testing & Observability](#phase-3-testing--observability)
4. [Phase 4: API Improvements](#phase-4-api-improvements)
5. [Phase 5: Documentation & Production Guide](#phase-5-documentation--production-guide)
6. [Migration Guide](#migration-guide)
7. [Timeline & Milestones](#timeline--milestones)

---

## Phase 1: Critical Fixes

**Priority:** P0 - Must fix before any production deployment
**Breaking Changes:** Yes

### 1.1 Fix Shutdown Race Condition

**Problem:** Current shutdown cancels context before draining events, causing event loss.

**Current Implementation:**
```go
func (c *Client) Shutdown(ctx context.Context) error {
    c.markClosed()           // 1. Stop accepting new events
    c.cancel()               // 2. Cancel context - GOROUTINES DIE HERE
    c.wg.Wait()              // 3. Wait for dead goroutines
    c.drainPendingEvents()   // 4. Try to drain with cancelled context - FAILS
}
```

**Proposed Implementation:**
```go
func (c *Client) Shutdown(ctx context.Context) error {
    // 1. Stop accepting new events
    c.markClosed()

    // 2. Signal batch processor to drain (don't cancel yet)
    close(c.drainSignal)

    // 3. Wait for batch processor to finish with timeout
    drainCtx, drainCancel := context.WithTimeout(ctx, c.config.ShutdownTimeout)
    defer drainCancel()

    select {
    case <-c.drainComplete:
        // Batch processor finished gracefully
    case <-drainCtx.Done():
        // Timeout - force cancel remaining operations
        c.cancel()
    }

    // 4. Wait for all goroutines
    c.wg.Wait()

    // 5. Report any lost events
    return c.reportShutdownStatus()
}
```

**New Fields Required:**
```go
type Client struct {
    // ... existing fields ...

    drainSignal   chan struct{}  // Signal to start draining
    drainComplete chan struct{}  // Signal drain finished
}
```

**Batch Processor Changes:**
```go
func (c *Client) batchProcessor() {
    defer close(c.drainComplete)

    for {
        select {
        case <-c.drainSignal:
            // Drain mode: process remaining queue items
            c.drainRemainingBatches()
            return

        case req := <-c.batchQueue:
            c.processBatchRequest(req)

        case <-c.flushTicker.C:
            c.flushCurrentBatch()
        }
    }
}

func (c *Client) drainRemainingBatches() {
    // Process all remaining items in queue
    for {
        select {
        case req := <-c.batchQueue:
            c.processBatchRequest(req)
        default:
            // Queue empty, flush final batch
            c.flushCurrentBatch()
            return
        }
    }
}
```

### 1.2 Event Delivery Guarantees

**Problem:** No clear contract on event delivery semantics.

**Proposed Contract:**
```go
// EventDeliveryMode defines the delivery guarantee for events.
type EventDeliveryMode int

const (
    // DeliveryBestEffort: Events may be lost under high load or failures.
    // Fastest option, no persistence, no retries beyond immediate failures.
    DeliveryBestEffort EventDeliveryMode = iota

    // DeliveryAtLeastOnce: Events are persisted and retried until acknowledged.
    // May result in duplicate events. Requires persistence configuration.
    DeliveryAtLeastOnce
)

// Config additions
type Config struct {
    // ... existing fields ...

    // DeliveryMode sets the event delivery guarantee. Default: DeliveryBestEffort
    DeliveryMode EventDeliveryMode

    // PersistencePath is required when DeliveryMode is DeliveryAtLeastOnce.
    // Events are persisted here before sending and removed after acknowledgment.
    PersistencePath string
}
```

**Documentation Update:**
```go
// Package langfuse provides a Go SDK for Langfuse observability.
//
// # Event Delivery Guarantees
//
// The SDK supports two delivery modes:
//
// Best Effort (default):
//   - Events are queued in memory and sent in batches
//   - Events may be lost if: queue overflows, shutdown times out, or process crashes
//   - Suitable for: development, non-critical telemetry, high-throughput scenarios
//
// At Least Once:
//   - Events are persisted to disk before acknowledgment
//   - Events survive process restarts and are retried on failure
//   - May result in duplicate events (idempotency required server-side)
//   - Suitable for: production, audit logs, billing events
```

---

## Phase 2: Integration & Consolidation

**Priority:** P1 - Required for production stability
**Breaking Changes:** Minimal

### 2.1 Wire Up Backpressure System

**Problem:** `BackpressureMonitor`, `QueueDepthMonitor`, `AdaptiveSampler` exist but aren't used.

**Proposed Integration:**

```go
type Client struct {
    // ... existing fields ...

    backpressure *BackpressureController
}

func NewClient(opts ...Option) (*Client, error) {
    // ... existing initialization ...

    // Initialize backpressure controller
    c.backpressure = NewBackpressureController(BackpressureConfig{
        QueueCapacity:    c.config.BatchSize * 10,
        HighWaterMark:    0.8,  // 80% triggers backpressure
        LowWaterMark:     0.5,  // 50% releases backpressure
        SamplingStrategy: AdaptiveSampling,
    })

    return c, nil
}

func (c *Client) queueEvent(ctx context.Context, event ingestionEvent) error {
    // Check backpressure before queuing
    if c.backpressure.ShouldReject() {
        c.metrics.IncrementCounter("langfuse.events.rejected.backpressure", 1)
        return ErrBackpressure
    }

    // Apply adaptive sampling if under pressure
    if c.backpressure.ShouldSample() && !c.backpressure.Sample() {
        c.metrics.IncrementCounter("langfuse.events.sampled", 1)
        return nil // Silently drop sampled event
    }

    // ... existing queue logic ...
}
```

**New Public API:**
```go
// BackpressureStatus represents the current backpressure state.
type BackpressureStatus struct {
    QueueDepth      int
    QueueCapacity   int
    Utilization     float64
    IsUnderPressure bool
    SamplingRate    float64
}

// BackpressureStatus returns the current backpressure state.
// Useful for monitoring and adaptive client behavior.
func (c *Client) BackpressureStatus() BackpressureStatus {
    return c.backpressure.Status()
}

// OnBackpressure registers a callback for backpressure state changes.
func (c *Client) OnBackpressure(fn func(status BackpressureStatus)) {
    c.backpressure.OnStateChange(fn)
}
```

### 2.2 Consolidate Error Types

**Problem:** Five error types with inconsistent interfaces.

**Proposed Unified Error Interface:**

```go
// LangfuseError is the common interface for all SDK errors.
type LangfuseError interface {
    error

    // Code returns a machine-readable error code.
    Code() ErrorCode

    // IsRetryable returns true if the operation can be retried.
    IsRetryable() bool

    // RequestID returns the server request ID, if available.
    RequestID() string

    // Unwrap returns the underlying error, if any.
    Unwrap() error
}

// ErrorCode represents categorized error types.
type ErrorCode string

const (
    ErrCodeValidation   ErrorCode = "VALIDATION_ERROR"
    ErrCodeAPI          ErrorCode = "API_ERROR"
    ErrCodeNetwork      ErrorCode = "NETWORK_ERROR"
    ErrCodeTimeout      ErrorCode = "TIMEOUT_ERROR"
    ErrCodeBackpressure ErrorCode = "BACKPRESSURE_ERROR"
    ErrCodeShutdown     ErrorCode = "SHUTDOWN_ERROR"
    ErrCodeInternal     ErrorCode = "INTERNAL_ERROR"
)
```

**Unified Error Implementation:**

```go
// Error is the unified error type for the SDK.
type Error struct {
    code       ErrorCode
    message    string
    requestID  string
    retryable  bool
    cause      error
    details    map[string]any
}

func (e *Error) Error() string {
    if e.cause != nil {
        return fmt.Sprintf("%s: %s: %v", e.code, e.message, e.cause)
    }
    return fmt.Sprintf("%s: %s", e.code, e.message)
}

func (e *Error) Code() ErrorCode       { return e.code }
func (e *Error) IsRetryable() bool     { return e.retryable }
func (e *Error) RequestID() string     { return e.requestID }
func (e *Error) Unwrap() error         { return e.cause }
func (e *Error) Details() map[string]any { return e.details }

// Constructors for common error types
func NewValidationError(field, message string) *Error {
    return &Error{
        code:      ErrCodeValidation,
        message:   message,
        retryable: false,
        details:   map[string]any{"field": field},
    }
}

func NewAPIError(statusCode int, message, requestID string) *Error {
    return &Error{
        code:      ErrCodeAPI,
        message:   message,
        requestID: requestID,
        retryable: statusCode == 429 || statusCode >= 500,
        details:   map[string]any{"status_code": statusCode},
    }
}
```

**Migration Path:**
```go
// Deprecated: Use Error with Code() == ErrCodeAPI instead.
type APIError = Error

// Deprecated: Use Error with Code() == ErrCodeValidation instead.
type ValidationError = Error

// Helper for migration
func IsValidationError(err error) bool {
    var e *Error
    return errors.As(err, &e) && e.Code() == ErrCodeValidation
}

func IsAPIError(err error) bool {
    var e *Error
    return errors.As(err, &e) && e.Code() == ErrCodeAPI
}
```

### 2.3 Consolidate Logging

**Problem:** Multiple logging interfaces, unused implementations.

**Proposed Single Interface:**

```go
// Logger is the logging interface for the SDK.
// Compatible with log/slog, zap, zerolog, and logrus.
type Logger interface {
    // Debug logs a debug message with optional key-value pairs.
    Debug(msg string, args ...any)

    // Info logs an info message with optional key-value pairs.
    Info(msg string, args ...any)

    // Warn logs a warning message with optional key-value pairs.
    Warn(msg string, args ...any)

    // Error logs an error message with optional key-value pairs.
    Error(msg string, args ...any)
}

// NopLogger is a no-op logger that discards all messages.
var NopLogger Logger = nopLogger{}

// StdLogger wraps the standard library logger.
func StdLogger(l *log.Logger) Logger {
    return &stdLogger{l: l}
}

// SlogLogger wraps a slog.Logger.
func SlogLogger(l *slog.Logger) Logger {
    return &slogLogger{l: l}
}
```

**Remove:**
- `StructuredLogger` interface (merged into `Logger`)
- `defaultLogger` (already removed)
- `loggerAdapter` (already removed)
- `formatArgs` function (no longer needed)

---

## Phase 3: Testing & Observability

**Priority:** P1 - Required for production confidence
**Breaking Changes:** None

### 3.1 Add Goroutine Leak Detection

```go
// leak_test.go
package langfuse

import (
    "testing"
    "go.uber.org/goleak"
)

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}

func TestClientShutdown_NoLeaks(t *testing.T) {
    defer goleak.VerifyNone(t)

    client, _ := NewClient(
        WithPublicKey("test"),
        WithSecretKey("test"),
    )

    // Create traces
    for i := 0; i < 100; i++ {
        trace, _ := client.NewTrace().Name("test").Create(context.Background())
        trace.Span().Name("span").Create(context.Background())
    }

    // Shutdown should clean up all goroutines
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    client.Shutdown(ctx)
}
```

### 3.2 Add Stress Tests

```go
// stress_test.go
package langfuse

import (
    "testing"
    "sync"
    "sync/atomic"
)

func TestHighThroughput(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping stress test in short mode")
    }

    server := newMockServer(t)
    client, _ := NewClient(
        WithBaseURL(server.URL),
        WithBatchSize(100),
        WithFlushInterval(100 * time.Millisecond),
    )
    defer client.Shutdown(context.Background())

    const (
        numGoroutines = 100
        eventsPerGoroutine = 1000
    )

    var (
        wg      sync.WaitGroup
        created atomic.Int64
        errors  atomic.Int64
    )

    start := time.Now()

    for i := 0; i < numGoroutines; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for j := 0; j < eventsPerGoroutine; j++ {
                _, err := client.NewTrace().Name("stress-test").Create(context.Background())
                if err != nil {
                    errors.Add(1)
                } else {
                    created.Add(1)
                }
            }
        }()
    }

    wg.Wait()
    elapsed := time.Since(start)

    t.Logf("Created %d events in %v (%.0f events/sec)",
        created.Load(), elapsed, float64(created.Load())/elapsed.Seconds())
    t.Logf("Errors: %d (%.2f%%)",
        errors.Load(), float64(errors.Load())/float64(numGoroutines*eventsPerGoroutine)*100)

    // Verify all events were received
    client.Flush(context.Background())

    if received := server.ReceivedEvents(); received < int(created.Load())*0.99 {
        t.Errorf("Expected at least 99%% delivery, got %d/%d", received, created.Load())
    }
}

func TestShutdownUnderLoad(t *testing.T) {
    server := newMockServer(t)
    client, _ := NewClient(WithBaseURL(server.URL))

    // Start continuous load
    ctx, cancel := context.WithCancel(context.Background())
    var wg sync.WaitGroup

    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for ctx.Err() == nil {
                client.NewTrace().Name("load").Create(context.Background())
            }
        }()
    }

    // Let it run for a bit
    time.Sleep(500 * time.Millisecond)

    // Initiate shutdown while load is running
    shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 2*time.Second)
    defer shutdownCancel()

    err := client.Shutdown(shutdownCtx)
    cancel() // Stop load generators
    wg.Wait()

    if err != nil {
        t.Logf("Shutdown completed with error: %v", err)
    }

    // Verify no goroutine leaks
    // (handled by TestMain with goleak)
}
```

### 3.3 Add Circuit Breaker Tests

```go
// circuitbreaker_test.go additions

func TestCircuitBreakerStateTransitions(t *testing.T) {
    cb := NewCircuitBreaker(CircuitBreakerConfig{
        FailureThreshold: 3,
        SuccessThreshold: 2,
        Timeout:          100 * time.Millisecond,
    })

    // Initial state: Closed
    assert.Equal(t, CircuitClosed, cb.State())

    // Fail 3 times -> Open
    for i := 0; i < 3; i++ {
        cb.RecordFailure()
    }
    assert.Equal(t, CircuitOpen, cb.State())

    // Cannot execute while open
    assert.False(t, cb.Allow())

    // Wait for timeout -> Half-Open
    time.Sleep(150 * time.Millisecond)
    assert.Equal(t, CircuitHalfOpen, cb.State())
    assert.True(t, cb.Allow()) // Allow one request

    // Success in half-open
    cb.RecordSuccess()
    assert.Equal(t, CircuitHalfOpen, cb.State()) // Still half-open

    cb.RecordSuccess()
    assert.Equal(t, CircuitClosed, cb.State()) // Now closed

    // Failure in half-open -> Open
    cb = NewCircuitBreaker(CircuitBreakerConfig{
        FailureThreshold: 3,
        SuccessThreshold: 2,
        Timeout:          100 * time.Millisecond,
    })
    for i := 0; i < 3; i++ {
        cb.RecordFailure()
    }
    time.Sleep(150 * time.Millisecond)
    cb.RecordFailure() // Fail in half-open
    assert.Equal(t, CircuitOpen, cb.State())
}
```

### 3.4 Expose Metrics Endpoint

```go
// metrics.go additions

// MetricsHandler returns an http.Handler that serves Prometheus metrics.
// This includes all SDK internal metrics.
//
// Example:
//
//     http.Handle("/metrics", client.MetricsHandler())
//     http.ListenAndServe(":9090", nil)
//
func (c *Client) MetricsHandler() http.Handler {
    if c.promRegistry == nil {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            w.WriteHeader(http.StatusNotImplemented)
            w.Write([]byte("Metrics not enabled. Use WithPrometheusMetrics() option."))
        })
    }
    return promhttp.HandlerFor(c.promRegistry, promhttp.HandlerOpts{})
}

// PrometheusMetrics returns a Metrics implementation backed by Prometheus.
// The registry can be used with promhttp.Handler() to expose metrics.
func PrometheusMetrics() (Metrics, *prometheus.Registry) {
    registry := prometheus.NewRegistry()

    counters := make(map[string]prometheus.Counter)
    histograms := make(map[string]prometheus.Histogram)
    gauges := make(map[string]prometheus.Gauge)

    return &prometheusMetrics{
        registry:   registry,
        counters:   counters,
        histograms: histograms,
        gauges:     gauges,
    }, registry
}
```

---

## Phase 4: API Improvements

**Priority:** P2 - Quality of life improvements
**Breaking Changes:** Additive only

### 4.1 Consistent Method Signatures

```go
// Current inconsistency:
trace.Create(ctx)                          // Returns (*TraceContext, error)
gen.EndWithUsage(ctx, out, in, out)        // Returns error
trace.Update().Apply(ctx)                  // Returns error

// Proposed additions (keep existing for compatibility):

// EndResult contains the result of ending an observation.
type EndResult struct {
    Duration time.Duration
    Error    error
}

// End ends the generation and returns the result.
func (g *GenerationContext) End(ctx context.Context) EndResult {
    // ... implementation
}

// EndWith ends the generation with output and usage.
func (g *GenerationContext) EndWith(ctx context.Context, opts ...EndOption) EndResult {
    // ... implementation
}

// EndOption configures how an observation is ended.
type EndOption func(*endConfig)

func WithOutput(output any) EndOption {
    return func(c *endConfig) { c.output = output }
}

func WithUsage(input, output int) EndOption {
    return func(c *endConfig) { c.inputTokens = input; c.outputTokens = output }
}

func WithError(err error) EndOption {
    return func(c *endConfig) { c.err = err }
}

// Example usage:
gen.EndWith(ctx,
    WithOutput(response),
    WithUsage(100, 50),
)
```

### 4.2 Type-Safe Builders

```go
// Current: accepts any
trace.Metadata(map[string]any{"key": value})

// Proposed: type-safe alternatives
type MetadataBuilder struct {
    data map[string]any
}

func NewMetadata() *MetadataBuilder {
    return &MetadataBuilder{data: make(map[string]any)}
}

func (m *MetadataBuilder) String(key, value string) *MetadataBuilder {
    m.data[key] = value
    return m
}

func (m *MetadataBuilder) Int(key string, value int) *MetadataBuilder {
    m.data[key] = value
    return m
}

func (m *MetadataBuilder) Float(key string, value float64) *MetadataBuilder {
    m.data[key] = value
    return m
}

func (m *MetadataBuilder) Bool(key string, value bool) *MetadataBuilder {
    m.data[key] = value
    return m
}

func (m *MetadataBuilder) JSON(key string, value any) *MetadataBuilder {
    m.data[key] = value
    return m
}

func (m *MetadataBuilder) Build() map[string]any {
    return m.data
}

// Usage:
trace.Metadata(NewMetadata().
    String("user_id", "123").
    Int("request_count", 5).
    Bool("is_premium", true).
    Build())
```

### 4.3 Batch Operations

```go
// BatchTraceBuilder creates multiple traces efficiently.
type BatchTraceBuilder struct {
    client  *Client
    traces  []*TraceBuilder
    options []BatchOption
}

func (c *Client) BatchTraces() *BatchTraceBuilder {
    return &BatchTraceBuilder{client: c}
}

func (b *BatchTraceBuilder) Add(name string) *TraceBuilder {
    tb := b.client.NewTrace().Name(name)
    b.traces = append(b.traces, tb)
    return tb
}

func (b *BatchTraceBuilder) Create(ctx context.Context) ([]*TraceContext, error) {
    results := make([]*TraceContext, len(b.traces))
    errs := make([]error, 0)

    for i, tb := range b.traces {
        tc, err := tb.Create(ctx)
        if err != nil {
            errs = append(errs, fmt.Errorf("trace %d: %w", i, err))
        }
        results[i] = tc
    }

    if len(errs) > 0 {
        return results, errors.Join(errs...)
    }
    return results, nil
}

// Usage:
batch := client.BatchTraces()
batch.Add("trace-1").UserID("user-1").Tags("api", "v2")
batch.Add("trace-2").UserID("user-2").Tags("api", "v2")
batch.Add("trace-3").UserID("user-3").Tags("api", "v2")

traces, err := batch.Create(ctx)
```

---

## Phase 5: Documentation & Production Guide

**Priority:** P2 - Required for enterprise adoption
**Breaking Changes:** None

### 5.1 Production Deployment Guide

Create `docs/PRODUCTION.md`:

```markdown
# Production Deployment Guide

## Resource Requirements

### Memory
- Base: ~10MB for client and buffers
- Per 1000 events/sec: +5MB for queue buffers
- Persistence mode: +50MB for disk I/O buffers

### CPU
- Minimal CPU usage under normal load
- Batch serialization: ~1ms per 100 events
- Compression (if enabled): ~5ms per 100 events

### Network
- Average event size: 500 bytes - 2KB
- Batch request size: 50KB - 500KB (depending on batch size)
- Recommended bandwidth: 1Mbps per 1000 events/sec

## Recommended Configuration

### Low Volume (< 100 events/sec)
```go
client, _ := langfuse.NewClient(
    langfuse.WithBatchSize(50),
    langfuse.WithFlushInterval(5 * time.Second),
    langfuse.WithMaxRetries(3),
)
```

### Medium Volume (100-1000 events/sec)
```go
client, _ := langfuse.NewClient(
    langfuse.WithBatchSize(100),
    langfuse.WithFlushInterval(1 * time.Second),
    langfuse.WithMaxRetries(5),
    langfuse.WithBackpressure(langfuse.BackpressureConfig{
        HighWaterMark: 0.8,
        Strategy:      langfuse.BackpressureAdaptiveSampling,
    }),
)
```

### High Volume (1000+ events/sec)
```go
client, _ := langfuse.NewClient(
    langfuse.WithBatchSize(500),
    langfuse.WithFlushInterval(500 * time.Millisecond),
    langfuse.WithMaxRetries(3),
    langfuse.WithDeliveryMode(langfuse.DeliveryBestEffort),
    langfuse.WithBackpressure(langfuse.BackpressureConfig{
        HighWaterMark: 0.7,
        Strategy:      langfuse.BackpressureReject,
    }),
    langfuse.WithCircuitBreaker(langfuse.CircuitBreakerConfig{
        FailureThreshold: 5,
        Timeout:          30 * time.Second,
    }),
)
```

## Monitoring

### Key Metrics to Watch
| Metric | Warning | Critical |
|--------|---------|----------|
| `langfuse.queue.utilization` | > 0.7 | > 0.9 |
| `langfuse.batch.errors` | > 1/min | > 10/min |
| `langfuse.events.dropped` | > 0 | > 100/min |
| `langfuse.circuit.state` | half-open | open |

### Prometheus Alerts
```yaml
groups:
  - name: langfuse
    rules:
      - alert: LangfuseQueueBackpressure
        expr: langfuse_queue_utilization > 0.8
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Langfuse queue under pressure"

      - alert: LangfuseCircuitOpen
        expr: langfuse_circuit_state == 2
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Langfuse circuit breaker open"
```

## Failure Scenarios

### API Unreachable
- Events queue locally (up to queue capacity)
- Circuit breaker opens after threshold failures
- Events dropped when queue full (logged to async error handler)
- Recovery: automatic when API returns

### Graceful Shutdown
- New events rejected immediately
- Existing queue drained (up to ShutdownTimeout)
- Pending events reported in ShutdownError
- Recovery: restart application

### High Memory Pressure
- Backpressure triggers at high water mark
- Adaptive sampling reduces event rate
- Events rejected at capacity
- Recovery: reduce event volume or scale horizontally
```

### 5.2 Troubleshooting Guide

Create `docs/TROUBLESHOOTING.md`:

```markdown
# Troubleshooting Guide

## Events Not Appearing in Langfuse

### Check 1: Client Configuration
```go
// Verify credentials
client, err := langfuse.NewClient(
    langfuse.WithPublicKey(os.Getenv("LANGFUSE_PUBLIC_KEY")),
    langfuse.WithSecretKey(os.Getenv("LANGFUSE_SECRET_KEY")),
)
if err != nil {
    log.Fatal("Client creation failed:", err)
}
```

### Check 2: Flush Before Exit
```go
// Always flush before program exit
defer func() {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    if err := client.Shutdown(ctx); err != nil {
        log.Println("Shutdown error:", err)
    }
}()
```

### Check 3: Enable Debug Logging
```go
client, _ := langfuse.NewClient(
    langfuse.WithLogger(slog.Default()),
    langfuse.WithDebug(true),
)
```

### Check 4: Monitor Async Errors
```go
client.OnAsyncError(func(err langfuse.AsyncError) {
    log.Printf("Async error: %v (operation: %s, events: %v)",
        err.Error, err.Operation, err.EventIDs)
})
```

## High Memory Usage

### Symptoms
- Memory grows continuously
- OOM kills in production

### Causes
1. Queue not draining (API issues)
2. High event volume without backpressure
3. Large event payloads

### Solutions
```go
// Enable backpressure
langfuse.WithBackpressure(langfuse.BackpressureConfig{
    HighWaterMark: 0.7,
    Strategy:      langfuse.BackpressureReject,
})

// Reduce batch size
langfuse.WithBatchSize(50)

// Enable compression
langfuse.WithCompression(true)
```

## Circuit Breaker Stays Open

### Check Circuit State
```go
status := client.CircuitBreakerStatus()
log.Printf("Circuit: state=%s, failures=%d, lastFailure=%v",
    status.State, status.Failures, status.LastFailure)
```

### Manual Reset (use with caution)
```go
client.ResetCircuitBreaker()
```
```

---

## Migration Guide

### From v1.x to v2.x

#### Error Handling Changes

```go
// v1.x
var apiErr *langfuse.APIError
if errors.As(err, &apiErr) {
    if apiErr.IsRetryable() {
        // retry
    }
}

// v2.x
var langfuseErr *langfuse.Error
if errors.As(err, &langfuseErr) {
    if langfuseErr.IsRetryable() {
        // retry
    }
    // New: access error code
    switch langfuseErr.Code() {
    case langfuse.ErrCodeValidation:
        // handle validation
    case langfuse.ErrCodeAPI:
        // handle API error
    }
}

// v2.x helper functions (recommended)
if langfuse.IsRetryable(err) {
    // retry
}
```

#### Shutdown Changes

```go
// v1.x - events could be lost
client.Shutdown(ctx)

// v2.x - explicit delivery guarantee
err := client.Shutdown(ctx)
if shutdownErr, ok := err.(*langfuse.ShutdownError); ok {
    log.Printf("Lost %d events", shutdownErr.PendingEvents)
}
```

#### Logger Interface

```go
// v1.x - two interfaces
type Logger interface {
    Printf(format string, v ...any)
}
type StructuredLogger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    // ...
}

// v2.x - single interface
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
}

// Migration helper
langfuse.WithLogger(langfuse.StdLogger(log.Default()))
```

---

## Timeline & Milestones

| Phase | Description | Duration | Dependencies |
|-------|-------------|----------|--------------|
| **Phase 1** | Critical Fixes | 2 weeks | None |
| 1.1 | Shutdown race condition | 3 days | |
| 1.2 | Event delivery guarantees | 4 days | |
| 1.3 | Testing & validation | 3 days | 1.1, 1.2 |
| **Phase 2** | Integration | 2 weeks | Phase 1 |
| 2.1 | Backpressure integration | 4 days | |
| 2.2 | Error consolidation | 3 days | |
| 2.3 | Logging consolidation | 2 days | |
| 2.4 | Migration testing | 3 days | 2.1-2.3 |
| **Phase 3** | Testing & Observability | 1 week | Phase 2 |
| 3.1 | Goroutine leak tests | 1 day | |
| 3.2 | Stress tests | 2 days | |
| 3.3 | Circuit breaker tests | 1 day | |
| 3.4 | Metrics endpoint | 1 day | |
| **Phase 4** | API Improvements | 2 weeks | Phase 2 |
| 4.1 | Consistent signatures | 3 days | |
| 4.2 | Type-safe builders | 3 days | |
| 4.3 | Batch operations | 4 days | |
| **Phase 5** | Documentation | 1 week | Phase 1-4 |
| 5.1 | Production guide | 2 days | |
| 5.2 | Troubleshooting guide | 2 days | |
| 5.3 | API documentation | 1 day | |

**Total Estimated Duration:** 8 weeks

---

## Success Criteria

### Phase 1 Complete
- [ ] Zero event loss during graceful shutdown (verified by tests)
- [ ] Documented delivery guarantees
- [ ] All shutdown race conditions eliminated

### Phase 2 Complete
- [ ] Backpressure activated under load (stress test)
- [ ] Single `Error` type with consistent interface
- [ ] Single `Logger` interface

### Phase 3 Complete
- [ ] Zero goroutine leaks (goleak passing)
- [ ] 10K events/sec sustained for 60 seconds
- [ ] 100% circuit breaker state coverage

### Phase 4 Complete
- [ ] All new APIs have consistent signatures
- [ ] Type-safe alternatives for common operations
- [ ] Batch operations working

### Phase 5 Complete
- [ ] Production guide reviewed by ops team
- [ ] Troubleshooting guide covers top 10 issues
- [ ] All public APIs documented

---

## Appendix: Files to Modify

### Phase 1
- `client.go` - Shutdown rewrite
- `ingestion.go` - Drain logic
- `doc.go` - Delivery guarantees

### Phase 2
- `backpressure.go` - Integration points
- `client.go` - Backpressure hooks
- `errors.go` - Unified error type
- `logging.go` - Single interface

### Phase 3
- `*_test.go` - New test files
- `Makefile` - Stress test targets

### Phase 4
- `trace.go`, `span.go`, `generation.go` - New methods
- `builders.go` - Type-safe builders (new file)
- `batch.go` - Batch operations (new file)

### Phase 5
- `docs/PRODUCTION.md` (new)
- `docs/TROUBLESHOOTING.md` (new)
- `docs/MIGRATION.md` (new)
