# Audit Enhancements

## New Items to Add

### Section 2 (API Design & Ergonomics) - Additional Items

#### 2.13 No Request/Response Interceptor Support
**Location:** `http.go`
**Issue:** No way to intercept requests/responses for custom logging, metrics, or modification.
**Priority:** P1

```go
// ADD interceptor interface
type RequestInterceptor interface {
    BeforeRequest(ctx context.Context, req *http.Request) error
    AfterResponse(ctx context.Context, req *http.Request, resp *http.Response, err error) error
}

type httpClient struct {
    client       *http.Client
    baseURL      string
    retryStrategy RetryStrategy
    debug        bool
    interceptors []RequestInterceptor  // NEW
}

func (h *httpClient) doOnce(ctx context.Context, req *request) error {
    // Build HTTP request
    httpReq, err := h.buildRequest(ctx, req)
    if err != nil {
        return err
    }

    // Execute interceptors before request
    for _, interceptor := range h.interceptors {
        if err := interceptor.BeforeRequest(ctx, httpReq); err != nil {
            return fmt.Errorf("interceptor before request failed: %w", err)
        }
    }

    // Execute request
    resp, err := h.client.Do(httpReq)

    // Execute interceptors after response
    for _, interceptor := range h.interceptors {
        if interceptErr := interceptor.AfterResponse(ctx, httpReq, resp, err); interceptErr != nil {
            // Log but don't fail
            if h.debug {
                log.Printf("interceptor after response failed: %v", interceptErr)
            }
        }
    }

    // ... rest of handling
}

// Config option
func WithRequestInterceptor(interceptor RequestInterceptor) ConfigOption {
    return func(c *Config) {
        c.Interceptors = append(c.Interceptors, interceptor)
    }
}

// Example: Correlation ID interceptor
type CorrelationIDInterceptor struct {
    headerName string
}

func (ci *CorrelationIDInterceptor) BeforeRequest(ctx context.Context, req *http.Request) error {
    // Extract correlation ID from context
    if correlationID, ok := ctx.Value("correlation_id").(string); ok {
        req.Header.Set(ci.headerName, correlationID)
    } else {
        // Generate new correlation ID
        req.Header.Set(ci.headerName, generateID())
    }
    return nil
}

func (ci *CorrelationIDInterceptor) AfterResponse(ctx context.Context, req *http.Request, resp *http.Response, err error) error {
    // Log correlation ID with response
    if resp != nil {
        correlationID := req.Header.Get(ci.headerName)
        log.Printf("Request %s completed with status %d", correlationID, resp.StatusCode)
    }
    return nil
}
```

**Impact:** HIGH - Essential for observability and custom behavior
**Effort:** 12 hours
**Dependencies:** None
**Risk:** LOW - Additive change, doesn't affect existing code
**Affects:** http.go, config.go

---

#### 2.14 No Middleware/Plugin System
**Location:** Global
**Issue:** No extensibility mechanism for users to add custom behavior.
**Priority:** P1

```go
// ADD middleware system
type Middleware interface {
    Name() string
    BeforeEvent(ctx context.Context, event ingestionEvent) (ingestionEvent, error)
    AfterEvent(ctx context.Context, event ingestionEvent, err error) error
}

type Client struct {
    // ... existing fields
    middlewares []Middleware
}

func (c *Client) queueEvent(event ingestionEvent) error {
    ctx := c.ctx

    // Execute before-event middlewares
    modifiedEvent := event
    for _, mw := range c.middlewares {
        var err error
        modifiedEvent, err = mw.BeforeEvent(ctx, modifiedEvent)
        if err != nil {
            return fmt.Errorf("middleware %s before-event failed: %w", mw.Name(), err)
        }
    }

    // Queue the modified event
    c.mu.Lock()
    if c.closed.Load() {
        c.mu.Unlock()
        return ErrClientClosed
    }
    c.pendingEvents = append(c.pendingEvents, modifiedEvent)
    c.mu.Unlock()

    // Execute after-event middlewares
    for _, mw := range c.middlewares {
        if err := mw.AfterEvent(ctx, modifiedEvent, nil); err != nil {
            // Log but don't fail
            c.log("middleware %s after-event failed: %v", mw.Name(), err)
        }
    }

    return nil
}

// Example: Metadata enrichment middleware
type MetadataEnrichmentMiddleware struct {
    additionalMetadata map[string]interface{}
}

func (m *MetadataEnrichmentMiddleware) Name() string {
    return "metadata-enrichment"
}

func (m *MetadataEnrichmentMiddleware) BeforeEvent(ctx context.Context, event ingestionEvent) (ingestionEvent, error) {
    // Add additional metadata to all events
    switch e := event.(type) {
    case *createTraceEvent:
        if e.Metadata == nil {
            e.Metadata = make(map[string]interface{})
        }
        for k, v := range m.additionalMetadata {
            e.Metadata[k] = v
        }
    case *createObservationEvent:
        if e.Metadata == nil {
            e.Metadata = make(map[string]interface{})
        }
        for k, v := range m.additionalMetadata {
            e.Metadata[k] = v
        }
    }
    return event, nil
}

func (m *MetadataEnrichmentMiddleware) AfterEvent(ctx context.Context, event ingestionEvent, err error) error {
    return nil
}

// Config option
func WithMiddleware(middleware Middleware) ConfigOption {
    return func(c *Config) {
        c.Middlewares = append(c.Middlewares, middleware)
    }
}
```

**Impact:** HIGH - Major extensibility feature
**Effort:** 16 hours
**Dependencies:** None
**Risk:** MEDIUM - Needs careful design to avoid performance impact
**Affects:** client.go, config.go, ingestion.go

---

#### 2.15 No Bulk Operation Support
**Location:** All client methods
**Issue:** No way to create multiple resources in a single call.
**Priority:** P1

```go
// ADD bulk operations
type BulkCreateTracesRequest struct {
    Traces []struct {
        ID        string
        Name      string
        UserID    string
        Metadata  map[string]interface{}
        Tags      []string
        Timestamp *Time
    }
}

type BulkCreateTracesResponse struct {
    Created []string  // IDs of successfully created traces
    Failed  []struct {
        Index int
        ID    string
        Error string
    }
}

func (c *TracesClient) BulkCreate(ctx context.Context, req *BulkCreateTracesRequest) (*BulkCreateTracesResponse, error) {
    if req == nil || len(req.Traces) == 0 {
        return nil, NewValidationError("traces", "at least one trace required")
    }

    if len(req.Traces) > 100 {
        return nil, NewValidationError("traces", "maximum 100 traces per bulk request")
    }

    // Convert to batch of ingestion events
    events := make([]ingestionEvent, 0, len(req.Traces))
    for i, trace := range req.Traces {
        if trace.ID == "" {
            trace.ID = generateID()
            req.Traces[i].ID = trace.ID
        }

        event := &createTraceEvent{
            ID:        trace.ID,
            Name:      trace.Name,
            UserID:    trace.UserID,
            Metadata:  trace.Metadata,
            Tags:      trace.Tags,
            Timestamp: trace.Timestamp,
        }

        if err := event.Validate(); err != nil {
            return nil, fmt.Errorf("trace at index %d invalid: %w", i, err)
        }

        events = append(events, event)
    }

    // Send batch directly (bypass normal queueing)
    if err := c.client.sendBatch(ctx, events); err != nil {
        return nil, err
    }

    // Build response
    response := &BulkCreateTracesResponse{
        Created: make([]string, len(req.Traces)),
    }
    for i, trace := range req.Traces {
        response.Created[i] = trace.ID
    }

    return response, nil
}

// Similar methods for bulk observations, scores, etc.
```

**Impact:** MEDIUM - Performance improvement for batch scenarios
**Effort:** 20 hours (across all resource types)
**Dependencies:** None
**Risk:** LOW - New functionality
**Affects:** traces.go, observations.go, scores.go, events.go

---

### Section 4 (Reliability & Resilience) - Additional Items

#### 4.8 No Client-Side Rate Limiting
**Location:** `client.go`, `http.go`
**Issue:** No protection against overwhelming the API with requests.
**Priority:** P0

```go
// ADD rate limiter
import "golang.org/x/time/rate"

type Config struct {
    // ... existing fields
    RateLimit     float64       // Requests per second (0 = unlimited)
    RateLimitBurst int          // Burst size
}

type Client struct {
    // ... existing fields
    rateLimiter *rate.Limiter
}

func New(publicKey, secretKey string, opts ...ConfigOption) (*Client, error) {
    // ... existing code

    // Initialize rate limiter if configured
    if config.RateLimit > 0 {
        burst := config.RateLimitBurst
        if burst == 0 {
            burst = int(config.RateLimit) // Default burst = limit
        }
        client.rateLimiter = rate.NewLimiter(rate.Limit(config.RateLimit), burst)
    }

    return client, nil
}

func (h *httpClient) do(ctx context.Context, req *request) error {
    // Wait for rate limiter if configured
    if h.rateLimiter != nil {
        if err := h.rateLimiter.Wait(ctx); err != nil {
            return fmt.Errorf("rate limit wait failed: %w", err)
        }
    }

    // ... existing retry loop
}

// Expose rate limiter stats
type RateLimitStats struct {
    Limit      float64
    Burst      int
    TokensAvailable int
}

func (c *Client) GetRateLimitStats() *RateLimitStats {
    if c.rateLimiter == nil {
        return nil
    }

    return &RateLimitStats{
        Limit: float64(c.rateLimiter.Limit()),
        Burst: c.rateLimiter.Burst(),
        TokensAvailable: c.rateLimiter.Tokens(),
    }
}
```

**Impact:** CRITICAL - Prevents API abuse and throttling
**Effort:** 6 hours
**Dependencies:** golang.org/x/time/rate
**Risk:** LOW - Well-tested library
**Affects:** client.go, http.go, config.go

---

#### 4.9 No Connection Draining on Shutdown
**Location:** `client.go:231-277`
**Issue:** Active HTTP connections not gracefully closed during shutdown.
**Priority:** P1

```go
// ENHANCE shutdown process
func (c *Client) Shutdown(ctx context.Context) error {
    if !c.closed.CompareAndSwap(false, true) {
        return ErrClientClosed
    }

    // Signal shutdown
    close(c.stopFlush)
    c.cancel()

    // Wait for background goroutines with timeout
    done := make(chan struct{})
    go func() {
        c.wg.Wait()
        close(done)
    }()

    select {
    case <-done:
        // All goroutines stopped
    case <-ctx.Done():
        return fmt.Errorf("shutdown timeout: %w", ctx.Err())
    }

    // Flush remaining events
    c.mu.Lock()
    events := c.pendingEvents
    c.pendingEvents = nil
    c.mu.Unlock()

    if len(events) > 0 {
        if err := c.sendBatch(ctx, events); err != nil {
            return fmt.Errorf("final flush failed: %w", err)
        }
    }

    // Close HTTP connection pool
    if transport, ok := c.http.client.Transport.(*http.Transport); ok {
        transport.CloseIdleConnections()

        // Wait for connections to drain (max 5 seconds)
        drainCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
        defer cancel()

        ticker := time.NewTicker(100 * time.Millisecond)
        defer ticker.Stop()

        for {
            select {
            case <-drainCtx.Done():
                // Force close remaining connections
                return nil
            case <-ticker.C:
                // Check if connections are drained
                // This is heuristic - http.Transport doesn't expose connection count
                time.Sleep(100 * time.Millisecond)
                return nil
            }
        }
    }

    return nil
}
```

**Impact:** MEDIUM - Cleaner shutdown, prevents connection leaks
**Effort:** 4 hours
**Dependencies:** Requires 4.3 (shutdown timeout)
**Risk:** LOW
**Affects:** client.go

---

#### 4.10 No Graceful Degradation Strategy
**Location:** `client.go`
**Issue:** No fallback behavior when API is unavailable.
**Priority:** P2

```go
// ADD degradation modes
type DegradationMode int

const (
    DegradationModeNone DegradationMode = iota
    DegradationModeLogOnly              // Log events but don't send
    DegradationModeLocalQueue           // Queue to disk/memory
    DegradationModeDrop                 // Silently drop events
)

type Config struct {
    // ... existing fields
    DegradationMode    DegradationMode
    DegradationThreshold int // Number of failures before degrading
}

type Client struct {
    // ... existing fields
    failureCount     atomic.Int64
    degraded         atomic.Bool
    degradationMode  DegradationMode
}

func (c *Client) handleError(err error) {
    c.failureCount.Add(1)

    if c.failureCount.Load() >= int64(c.config.DegradationThreshold) {
        if c.degraded.CompareAndSwap(false, true) {
            c.log("WARN: Entering degradation mode after %d failures", c.failureCount.Load())

            if c.config.Metrics != nil {
                c.config.Metrics.IncrementCounter("langfuse.degradation_entered", 1)
            }
        }
    }

    // ... existing error handling
}

func (c *Client) queueEvent(event ingestionEvent) error {
    // Check if degraded
    if c.degraded.Load() {
        switch c.degradationMode {
        case DegradationModeLogOnly:
            c.log("DEGRADED: Would send event %s (type: %s)", event.GetID(), event.GetType())
            return nil

        case DegradationModeDrop:
            if c.config.Metrics != nil {
                c.config.Metrics.IncrementCounter("langfuse.events_dropped", 1)
            }
            return nil

        case DegradationModeLocalQueue:
            // Queue to persistent storage
            return c.queueToLocalStorage(event)

        default:
            // Continue normal operation even when degraded
        }
    }

    // ... normal queueing logic
}

// Recovery mechanism
func (c *Client) checkRecovery() {
    // Called after successful batch send
    if c.degraded.Load() {
        c.failureCount.Store(0)

        if c.degraded.CompareAndSwap(true, false) {
            c.log("INFO: Recovered from degradation mode")

            if c.config.Metrics != nil {
                c.config.Metrics.IncrementCounter("langfuse.degradation_recovered", 1)
            }
        }
    }
}
```

**Impact:** MEDIUM - Better resilience
**Effort:** 10 hours
**Dependencies:** None
**Risk:** MEDIUM - Needs careful testing
**Affects:** client.go, config.go

---

### Section 5 (Observability & Debugging) - ENHANCED OpenTelemetry

#### 5.3 Enhanced OpenTelemetry Integration (EXPANDED)
**Location:** Multiple files
**Issue:** Need comprehensive OTel integration with traces, metrics, and logs.
**Priority:** P1

```go
// COMPREHENSIVE OpenTelemetry integration
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/trace"
    "go.opentelemetry.io/otel/metric"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/codes"
)

type Config struct {
    // ... existing fields
    OTelEnabled     bool
    OTelTracerName  string
    OTelMeterName   string
}

type Client struct {
    // ... existing fields
    tracer          trace.Tracer
    meter           metric.Meter

    // Metrics instruments
    eventsQueued    metric.Int64Counter
    eventsSent      metric.Int64Counter
    eventsFailed    metric.Int64Counter
    batchSize       metric.Int64Histogram
    batchDuration   metric.Float64Histogram
    queueDepth      metric.Int64ObservableGauge
}

func (c *Client) initOpenTelemetry() error {
    if !c.config.OTelEnabled {
        return nil
    }

    tracerName := c.config.OTelTracerName
    if tracerName == "" {
        tracerName = "langfuse-go"
    }

    meterName := c.config.OTelMeterName
    if meterName == "" {
        meterName = "langfuse-go"
    }

    c.tracer = otel.Tracer(tracerName)
    c.meter = otel.Meter(meterName)

    // Initialize metric instruments
    var err error

    c.eventsQueued, err = c.meter.Int64Counter(
        "langfuse.events.queued",
        metric.WithDescription("Number of events queued"),
        metric.WithUnit("{event}"),
    )
    if err != nil {
        return fmt.Errorf("failed to create eventsQueued counter: %w", err)
    }

    c.eventsSent, err = c.meter.Int64Counter(
        "langfuse.events.sent",
        metric.WithDescription("Number of events successfully sent"),
        metric.WithUnit("{event}"),
    )
    if err != nil {
        return fmt.Errorf("failed to create eventsSent counter: %w", err)
    }

    c.eventsFailed, err = c.meter.Int64Counter(
        "langfuse.events.failed",
        metric.WithDescription("Number of events that failed to send"),
        metric.WithUnit("{event}"),
    )
    if err != nil {
        return fmt.Errorf("failed to create eventsFailed counter: %w", err)
    }

    c.batchSize, err = c.meter.Int64Histogram(
        "langfuse.batch.size",
        metric.WithDescription("Size of batches sent"),
        metric.WithUnit("{event}"),
    )
    if err != nil {
        return fmt.Errorf("failed to create batchSize histogram: %w", err)
    }

    c.batchDuration, err = c.meter.Float64Histogram(
        "langfuse.batch.duration",
        metric.WithDescription("Duration of batch send operations"),
        metric.WithUnit("ms"),
    )
    if err != nil {
        return fmt.Errorf("failed to create batchDuration histogram: %w", err)
    }

    c.queueDepth, err = c.meter.Int64ObservableGauge(
        "langfuse.queue.depth",
        metric.WithDescription("Current depth of event queue"),
        metric.WithUnit("{event}"),
        metric.WithInt64Callback(func(ctx context.Context, obs metric.Int64Observer) error {
            c.mu.Lock()
            depth := len(c.pendingEvents)
            c.mu.Unlock()
            obs.Observe(int64(depth))
            return nil
        }),
    )
    if err != nil {
        return fmt.Errorf("failed to create queueDepth gauge: %w", err)
    }

    return nil
}

func (c *Client) queueEvent(event ingestionEvent) error {
    ctx := c.ctx

    // Start OTel span
    if c.tracer != nil {
        var span trace.Span
        ctx, span = c.tracer.Start(ctx, "langfuse.queueEvent",
            trace.WithAttributes(
                attribute.String("event.id", event.GetID()),
                attribute.String("event.type", string(event.GetType())),
            ),
        )
        defer span.End()
    }

    // ... existing queueing logic

    // Record metric
    if c.eventsQueued != nil {
        c.eventsQueued.Add(ctx, 1,
            metric.WithAttributes(attribute.String("event.type", string(event.GetType()))),
        )
    }

    return nil
}

func (c *Client) sendBatch(ctx context.Context, events []ingestionEvent) error {
    // Start OTel span
    if c.tracer != nil {
        var span trace.Span
        ctx, span = c.tracer.Start(ctx, "langfuse.sendBatch",
            trace.WithAttributes(
                attribute.Int("batch.size", len(events)),
            ),
        )
        defer span.End()

        start := time.Now()
        err := c.sendBatchImpl(ctx, events)
        duration := time.Since(start)

        // Record metrics
        if c.batchSize != nil {
            c.batchSize.Record(ctx, int64(len(events)))
        }

        if c.batchDuration != nil {
            c.batchDuration.Record(ctx, float64(duration.Milliseconds()))
        }

        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())

            if c.eventsFailed != nil {
                c.eventsFailed.Add(ctx, int64(len(events)))
            }

            return err
        }

        span.SetStatus(codes.Ok, "")

        if c.eventsSent != nil {
            c.eventsSent.Add(ctx, int64(len(events)))
        }

        return nil
    }

    return c.sendBatchImpl(ctx, events)
}

func (c *Client) sendBatchImpl(ctx context.Context, events []ingestionEvent) error {
    // ... existing implementation
}

// HTTP client OTel instrumentation
func (h *httpClient) doOnce(ctx context.Context, req *request) error {
    if h.tracer != nil {
        var span trace.Span
        ctx, span = h.tracer.Start(ctx, "langfuse.http.request",
            trace.WithAttributes(
                attribute.String("http.method", req.method),
                attribute.String("http.path", req.path),
            ),
        )
        defer span.End()

        err := h.doOnceImpl(ctx, req)

        if err != nil {
            span.RecordError(err)
            span.SetStatus(codes.Error, err.Error())
        } else {
            span.SetStatus(codes.Ok, "")
        }

        return err
    }

    return h.doOnceImpl(ctx, req)
}
```

**Impact:** CRITICAL - Essential for production observability
**Effort:** 24 hours
**Dependencies:** go.opentelemetry.io/otel
**Risk:** LOW - Well-established library
**Affects:** client.go, http.go, config.go, all operation methods

---

#### 5.8 Correlation ID and Request ID Support
**Location:** `http.go`, `client.go`
**Issue:** No built-in support for correlation IDs across distributed traces.
**Priority:** P1

```go
// ADD correlation ID support
type ContextKey string

const (
    CorrelationIDKey ContextKey = "langfuse.correlation_id"
    RequestIDKey     ContextKey = "langfuse.request_id"
)

// Extract from context or generate
func getOrGenerateCorrelationID(ctx context.Context) string {
    if correlationID, ok := ctx.Value(CorrelationIDKey).(string); ok && correlationID != "" {
        return correlationID
    }
    return generateID()
}

func getOrGenerateRequestID(ctx context.Context) string {
    if requestID, ok := ctx.Value(RequestIDKey).(string); ok && requestID != "" {
        return requestID
    }
    return generateID()
}

// HTTP client integration
func (h *httpClient) doOnce(ctx context.Context, req *request) error {
    // ... build HTTP request

    // Add correlation and request IDs
    correlationID := getOrGenerateCorrelationID(ctx)
    requestID := getOrGenerateRequestID(ctx)

    httpReq.Header.Set("X-Correlation-ID", correlationID)
    httpReq.Header.Set("X-Request-ID", requestID)

    if h.debug {
        log.Printf("[correlation_id=%s][request_id=%s] %s %s",
            correlationID, requestID, req.method, req.path)
    }

    // ... execute request
}

// Helper functions for users
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
    return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
    return context.WithValue(ctx, RequestIDKey, requestID)
}

func GetCorrelationID(ctx context.Context) (string, bool) {
    correlationID, ok := ctx.Value(CorrelationIDKey).(string)
    return correlationID, ok
}

func GetRequestID(ctx context.Context) (string, bool) {
    requestID, ok := ctx.Value(RequestIDKey).(string)
    return requestID, ok
}
```

**Impact:** HIGH - Critical for distributed tracing
**Effort:** 6 hours
**Dependencies:** None
**Risk:** LOW
**Affects:** http.go, client.go

---

### Section 8 (Security & Safety) - Additional Items

#### 8.7 SDK Version in User-Agent
**Location:** `http.go:116`
**Issue:** User-Agent doesn't include SDK version for tracking.
**Priority:** P2

```go
// ADD version.go
package langfuse

// Version is set by build process
var Version = "dev"

// BuildInfo contains build metadata
type BuildInfo struct {
    Version   string
    GitCommit string
    BuildDate string
    GoVersion string
}

var Build = BuildInfo{
    Version:   Version,
    GitCommit: "unknown",
    BuildDate: "unknown",
    GoVersion: runtime.Version(),
}

// In http.go
func (h *httpClient) doOnce(ctx context.Context, req *request) error {
    // ... build request

    userAgent := fmt.Sprintf("langfuse-go/%s (Go %s; %s/%s)",
        Version,
        runtime.Version(),
        runtime.GOOS,
        runtime.GOARCH,
    )

    httpReq.Header.Set("User-Agent", userAgent)

    // ... rest of request
}

// Build with version injection:
// go build -ldflags "-X github.com/langfuse/langfuse-go.Version=1.0.0 -X github.com/langfuse/langfuse-go.Build.GitCommit=$(git rev-parse HEAD) -X github.com/langfuse/langfuse-go.Build.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

**Impact:** MEDIUM - Better analytics and debugging
**Effort:** 3 hours
**Dependencies:** None
**Risk:** LOW
**Affects:** http.go, new version.go file

---

### Section 9 (Compatibility & Maintenance) - New Sections

## 10. SDK Distribution & Release Management

### P0: Critical Issues

#### 10.1 No Automated Release Process
**Location:** Repository root
**Issue:** No automated release pipeline with changelog, tagging, and artifact publishing.
**Priority:** P0

```yaml
# ADD .github/workflows/release.yml
name: Release

on:
  push:
    tags:
      - 'v*'

permissions:
  contents: write

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'

      - name: Run tests
        run: go test -v -race ./...

      - name: Run goreleaser
        uses: goreleaser/goreleaser-action@v5
        with:
          version: latest
          args: release --clean
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

# ADD .goreleaser.yaml
version: 2

before:
  hooks:
    - go mod tidy
    - go test -v -race ./...

builds:
  - skip: true  # Go library, no binaries

archives:
  - format: tar.gz
    name_template: >-
      {{ .ProjectName }}_
      {{- .Version }}_
      {{- .Os }}_
      {{- .Arch }}

checksum:
  name_template: 'checksums.txt'

snapshot:
  name_template: "{{ incpatch .Version }}-next"

changelog:
  sort: asc
  use: github
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^chore:'
      - Merge pull request
      - Merge branch
  groups:
    - title: Breaking Changes
      regexp: "^.*BREAKING CHANGE:.*$"
      order: 0
    - title: Features
      regexp: "^.*feat[(\\w)]*:.*$"
      order: 1
    - title: Bug Fixes
      regexp: "^.*fix[(\\w)]*:.*$"
      order: 2
    - title: Performance
      regexp: "^.*perf[(\\w)]*:.*$"
      order: 3
    - title: Others
      order: 999

announce:
  skip: false
```

**Impact:** CRITICAL - Professional release management
**Effort:** 8 hours
**Dependencies:** GitHub repository
**Risk:** LOW
**Affects:** Repository configuration

---

#### 10.2 No Go Module Versioning Strategy
**Location:** go.mod
**Issue:** Module path doesn't follow Go versioning conventions for v2+.
**Priority:** P0

```go
// CURRENT go.mod
module github.com/langfuse/langfuse-go

// SHOULD BE (for v2+)
module github.com/langfuse/langfuse-go/v2

// Migration plan:
// 1. For v0.x.x and v1.x.x: Current path is fine
// 2. For v2.0.0+: Must use versioned import path
//    - Create /v2 directory
//    - Update module path
//    - Update all imports

// ADD migration guide in docs/
```

**Impact:** CRITICAL - Correct Go module semantics
**Effort:** 4 hours (documentation)
**Dependencies:** None
**Risk:** HIGH if done incorrectly
**Affects:** go.mod, all import statements (for v2+)

---

#### 10.3 No Deprecation Policy
**Location:** Documentation
**Issue:** No clear policy for deprecating features.
**Priority:** P1

```markdown
# ADD docs/deprecation-policy.md

# Deprecation Policy

## Overview
This document outlines the deprecation policy for the langfuse-go SDK.

## Versioning
We follow Semantic Versioning (SemVer):
- MAJOR version: Incompatible API changes
- MINOR version: Backwards-compatible functionality additions
- PATCH version: Backwards-compatible bug fixes

## Deprecation Process

### 1. Announcement (N)
- Mark function/type with `// Deprecated:` comment
- Add deprecation notice to changelog
- Document replacement in deprecation comment
- Add deprecation date

Example:
```go
// Deprecated: Use NewClient instead. This function will be removed in v2.0.0.
// Deprecated since v1.5.0 (2024-03-01).
func NewClientV1(key string) (*Client, error) {
    return NewClient(key)
}
```

### 2. Warning Period (N+1 minor versions)
- Keep deprecated feature functional
- Log warnings when used (if Logger configured)
- Update all examples to use new API
- Minimum 6 months before removal

### 3. Removal (N+2 or next major)
- Remove deprecated code in next major version
- Document removal in migration guide
- Provide automated migration tools if possible

## Support Timeline
- **Current major version**: Full support, bug fixes, new features
- **Previous major version**: Security fixes only for 12 months
- **Older versions**: No support

## Communication Channels
- Deprecation notices in code comments
- CHANGELOG.md entries
- GitHub Discussions posts
- Release notes

## Examples

### Function Deprecation
```go
// Deprecated: Use TraceContext.CreateSpan instead.
// This method will be removed in v2.0.0.
// Deprecated since v1.3.0 (2024-02-15).
func (tc *TraceContext) NewSpan(name string) (*SpanContext, error) {
    return tc.CreateSpan(name)
}
```

### Type Deprecation
```go
// Deprecated: Use Config instead.
// This type will be removed in v2.0.0.
// Deprecated since v1.4.0 (2024-02-20).
//
// Migration:
//   Old: cfg := &ClientConfig{...}
//   New: cfg := &Config{...}
type ClientConfig = Config
```

### Constant Deprecation
```go
const (
    // Deprecated: Use DefaultBatchSize instead.
    // Deprecated since v1.2.0 (2024-01-10).
    BatchSize = 20

    DefaultBatchSize = 20
)
```
```

**Impact:** HIGH - Clear expectations for users
**Effort:** 4 hours
**Dependencies:** None
**Risk:** LOW
**Affects:** Documentation, development process

---

### P1: High Priority Issues

#### 10.4 No Backwards Compatibility Testing
**Location:** Test suite
**Issue:** No automated tests to ensure compatibility with previous versions.
**Priority:** P1

```go
// ADD compat_test.go
package langfuse_test

import (
    "testing"
    "encoding/json"
)

// Test that old JSON structures can still be unmarshaled
func TestBackwardsCompatibility_TraceJSON_v1_0(t *testing.T) {
    // JSON from v1.0.0
    oldJSON := `{
        "id": "trace-1",
        "name": "test",
        "user_id": "user-1",
        "metadata": {"key": "value"}
    }`

    var trace Trace
    err := json.Unmarshal([]byte(oldJSON), &trace)
    if err != nil {
        t.Fatalf("Failed to unmarshal v1.0.0 JSON: %v", err)
    }

    if trace.ID != "trace-1" {
        t.Errorf("Expected ID 'trace-1', got '%s'", trace.ID)
    }
}

// Test that old client initialization still works
func TestBackwardsCompatibility_ClientInit_v1_0(t *testing.T) {
    // v1.0.0 style initialization (if it changed)
    client, err := New("pk", "sk")
    if err != nil {
        t.Fatalf("v1.0.0 style init failed: %v", err)
    }

    if client == nil {
        t.Fatal("Expected non-nil client")
    }
}

// Add more tests for each major/minor version
```

**Impact:** MEDIUM - Prevents breaking changes
**Effort:** 8 hours (initial setup) + 2 hours per version
**Dependencies:** None
**Risk:** LOW
**Affects:** Test suite

---

## 11. Performance SLAs and Guarantees

### P1: High Priority Issues

#### 11.1 No Performance SLAs Documented
**Location:** Documentation
**Issue:** No documented performance guarantees for SDK operations.
**Priority:** P1

```markdown
# ADD docs/performance-slas.md

# Performance SLAs

## Operation Latency Guarantees

### Synchronous Operations
| Operation | P50 | P95 | P99 | Max |
|-----------|-----|-----|-----|-----|
| Event queuing | <100μs | <500μs | <1ms | <10ms |
| Trace creation | <200μs | <1ms | <5ms | <20ms |
| Span creation | <150μs | <800μs | <3ms | <15ms |
| Validation | <50μs | <200μs | <500μs | <2ms |

### Asynchronous Operations
| Operation | P50 | P95 | P99 | Max |
|-----------|-----|-----|-----|-----|
| Batch send (network) | <100ms | <500ms | <1s | <5s |
| Flush | <200ms | <1s | <2s | <10s |
| Shutdown | <500ms | <2s | <5s | <30s |

## Resource Usage Guarantees

### Memory
- Per-event overhead: <1KB
- Client overhead: <100KB
- Queue overhead: <BatchSize × 1KB
- No memory leaks (validated with -race and pprof)

### CPU
- Queueing overhead: <0.1% CPU
- Background processing: <5% CPU (at 1000 events/sec)
- Zero CPU when idle

### Goroutines
- Fixed goroutine count: 2-3
- No goroutine leaks
- Clean shutdown: all goroutines terminated within shutdown timeout

### Network
- Batch compression: 70-90% reduction
- Connection pooling: max 10 idle connections
- Automatic connection reuse

## Throughput Guarantees

### Event Ingestion
- Sustained throughput: >10,000 events/sec/client
- Burst throughput: >50,000 events/sec for <10 seconds
- Batch efficiency: >95% (events successfully sent / events queued)

### API Operations
- Concurrent operations: 100+ simultaneous without degradation
- Thread-safe: all operations safe for concurrent use

## Reliability Guarantees

### Error Handling
- Retry success rate: >99% for transient failures
- Circuit breaker activation: <10ms detection
- Graceful degradation: automatic fallback on sustained failures

### Data Integrity
- Event ordering: guaranteed within single trace
- Duplicate prevention: >99.9% accuracy
- Data loss: <0.1% under normal conditions

## Monitoring and Validation

### Continuous Monitoring
```go
// Benchmark tests run on every PR
func BenchmarkEventQueuing(b *testing.B) {
    // Must complete in <1ms per operation
}

func BenchmarkBatchSend(b *testing.B) {
    // Must complete in <500ms per batch
}
```

### SLA Validation
- Automated benchmarks on every commit
- Performance regression tests
- Load testing in CI/CD
- Memory leak detection with -race

## Degradation Conditions

### When SLAs May Not Apply
1. Network latency >1s to API
2. CPU throttling >50%
3. Memory pressure (OOM conditions)
4. Disk I/O saturation (if using local queue)
5. API rate limiting (429 responses)

### Degradation Behavior
- Log warnings when SLAs violated
- Emit metrics for performance tracking
- Automatic backoff on API errors
- Circuit breaker activation

## Version Compatibility

These SLAs apply to:
- Go 1.21+
- Linux amd64/arm64
- macOS amd64/arm64
- Windows amd64

SLAs may vary on other platforms.
```

**Impact:** HIGH - Clear expectations
**Effort:** 6 hours (documentation + validation)
**Dependencies:** Benchmark suite
**Risk:** MEDIUM - Must maintain these guarantees
**Affects:** Documentation, CI/CD

---

## 12. Support Matrix

### P1: High Priority Issues

#### 12.1 No Support Matrix Documented
**Location:** Documentation
**Issue:** No clear documentation of supported environments.
**Priority:** P1

```markdown
# ADD docs/support-matrix.md

# Support Matrix

## Go Version Support

| Go Version | Support Status | Notes |
|------------|----------------|-------|
| 1.23+ | ✅ Fully Supported | Recommended |
| 1.22 | ✅ Fully Supported | |
| 1.21 | ✅ Supported | Minimum version |
| 1.20 | ⚠️ Best Effort | No guarantees |
| <1.20 | ❌ Not Supported | |

### Support Policy
- **Fully Supported**: All features, bug fixes, security updates
- **Supported**: Bug fixes and security updates only
- **Best Effort**: May work but not tested
- **Not Supported**: Will not work or not tested

We support the latest 3 Go minor versions.

## Operating System Support

| OS | Architecture | Support Status | Notes |
|----|-------------|----------------|-------|
| Linux | amd64 | ✅ Fully Supported | Primary platform |
| Linux | arm64 | ✅ Fully Supported | |
| macOS | amd64 | ✅ Fully Supported | |
| macOS | arm64 | ✅ Fully Supported | Apple Silicon |
| Windows | amd64 | ✅ Supported | |
| FreeBSD | amd64 | ⚠️ Best Effort | |
| Others | * | ❌ Not Tested | |

## Runtime Environment Support

| Environment | Support Status | Notes |
|-------------|----------------|-------|
| Standard Go Runtime | ✅ Fully Supported | |
| Docker Containers | ✅ Fully Supported | |
| Kubernetes | ✅ Fully Supported | |
| AWS Lambda | ✅ Supported | See Lambda guide |
| Google Cloud Functions | ✅ Supported | |
| Azure Functions | ⚠️ Best Effort | |
| TinyGo | ❌ Not Supported | Missing dependencies |

## Dependency Requirements

### Required Dependencies
- None (stdlib only for core functionality)

### Optional Dependencies
| Dependency | Version | Purpose |
|------------|---------|---------|
| go.opentelemetry.io/otel | v1.20+ | OpenTelemetry integration |
| golang.org/x/time | latest | Rate limiting |

## Feature Support by Environment

| Feature | Linux | macOS | Windows | Lambda |
|---------|-------|-------|---------|--------|
| Basic ingestion | ✅ | ✅ | ✅ | ✅ |
| Batching | ✅ | ✅ | ✅ | ✅ |
| Background processing | ✅ | ✅ | ✅ | ⚠️ |
| Connection pooling | ✅ | ✅ | ✅ | ✅ |
| OpenTelemetry | ✅ | ✅ | ✅ | ✅ |
| File-based queue | ✅ | ✅ | ✅ | ❌ |

⚠️ Lambda: Use synchronous flush before function completion

## Known Limitations

### Windows
- File descriptor limits may affect connection pooling
- Signal handling differs from POSIX systems

### Lambda/Serverless
- Background goroutines may be frozen between invocations
- Must call `Flush()` before function returns
- Consider using `SyncMode: true` config

### ARM64
- Performance benchmarks use amd64 baseline
- May have different performance characteristics

## Testing Matrix

### CI/CD Testing
We run tests on:
- Go 1.21, 1.22, 1.23
- Linux amd64, macOS amd64, Windows amd64
- With -race detector
- With -cover for coverage

### Manual Testing
Periodically tested on:
- Apple Silicon (M1/M2)
- ARM64 Linux
- Various Docker images

## Reporting Issues

When reporting issues, please include:
- Go version (`go version`)
- OS and architecture (`go env GOOS GOARCH`)
- SDK version
- Minimal reproduction

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.
```

**Impact:** HIGH - Sets clear expectations
**Effort:** 4 hours
**Dependencies:** None
**Risk:** LOW
**Affects:** Documentation

---

## 13. Community and Contribution Guidelines

### P2: Medium Priority Issues

#### 13.1 No Contribution Guidelines
**Location:** Repository root
**Issue:** No CONTRIBUTING.md file.
**Priority:** P2

```markdown
# ADD CONTRIBUTING.md

# Contributing to langfuse-go

Thank you for your interest in contributing! This document provides guidelines for contributing to the langfuse-go SDK.

## Code of Conduct

This project follows the [Contributor Covenant Code of Conduct](CODE_OF_CONDUCT.md).

## Getting Started

### Prerequisites
- Go 1.21 or later
- Git
- Make (optional, for convenience)

### Setup Development Environment
```bash
# Clone repository
git clone https://github.com/langfuse/langfuse-go.git
cd langfuse-go

# Install dependencies
go mod download

# Run tests
go test -v ./...

# Run with race detector
go test -race ./...

# Run linters
golangci-lint run
```

## Development Workflow

### 1. Create an Issue
- Check existing issues first
- Describe the problem or feature
- Wait for maintainer feedback before starting work

### 2. Fork and Branch
```bash
git checkout -b feature/my-feature
# or
git checkout -b fix/my-bugfix
```

### 3. Make Changes
- Write tests first (TDD)
- Keep changes focused and atomic
- Follow Go conventions
- Add godoc comments
- Update CHANGELOG.md

### 4. Test Your Changes
```bash
# Unit tests
go test -v ./...

# Race detection
go test -race ./...

# Coverage
go test -cover ./...

# Benchmarks
go test -bench=. ./...
```

### 5. Submit Pull Request
- Write clear PR description
- Link related issues
- Ensure CI passes
- Request review

## Code Standards

### Go Style
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` (enforced by CI)
- Use `golangci-lint` (enforced by CI)

### Documentation
- Add godoc for all exported types/functions
- Include usage examples in godoc
- Update README.md if adding features

### Testing
- Aim for >90% coverage
- Test edge cases
- Test error paths
- Add benchmarks for performance-critical code

### Commit Messages
Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <description>

[optional body]

[optional footer]
```

Types:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation only
- `style`: Formatting changes
- `refactor`: Code restructuring
- `perf`: Performance improvement
- `test`: Adding tests
- `chore`: Maintenance tasks

Examples:
```
feat(client): add circuit breaker support

Implements circuit breaker pattern to prevent cascading failures
when API is unavailable.

Closes #123

fix(retry): prevent goroutine leak in exponential backoff

BREAKING CHANGE: RetryStrategy interface now includes Cleanup method
```

## Pull Request Process

1. Update documentation
2. Add tests
3. Update CHANGELOG.md
4. Ensure CI passes
5. Get review from maintainer
6. Address feedback
7. Squash commits if needed
8. Merge when approved

## Release Process

Maintainers only:

1. Update version in version.go
2. Update CHANGELOG.md
3. Create git tag: `git tag v1.2.3`
4. Push tag: `git push origin v1.2.3`
5. GitHub Actions will create release

## Questions?

- Open a [Discussion](https://github.com/langfuse/langfuse-go/discussions)
- Join our [Discord](https://discord.gg/langfuse)
- Email: support@langfuse.com
```

**Impact:** MEDIUM - Community engagement
**Effort:** 6 hours
**Dependencies:** CODE_OF_CONDUCT.md
**Risk:** LOW
**Affects:** Documentation, community

---
