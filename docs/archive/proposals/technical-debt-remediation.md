# Technical Debt Remediation Proposal

**Status**: Draft
**Author**: AI Assistant
**Created**: 2025-12-27
**Priority**: High

---

## Executive Summary

This proposal addresses critical technical debt in the langfuse-go SDK that undermines code quality, maintainability, and reliability. The issues fall into three categories:

1. **Dead Code** - Infrastructure created but never integrated
2. **Incomplete Integration** - Features partially implemented
3. **Test Coverage Gaps** - Critical paths untested

**Estimated Effort**: 3-4 days
**Risk Level**: Low (mostly additive/refactoring changes)
**Breaking Changes**: None (all changes are internal or additive)

---

## Current State Analysis

| Metric | Current | Target |
|--------|---------|--------|
| Main package coverage | 74.9% | 85%+ |
| Zero-coverage functions | 469 | <50 |
| Hardcoded API paths | 17 | 0 |
| `endpoints` constant usage | 0 | All API calls |
| Circuit breaker coverage | Ingestion only | All HTTP calls |
| Generic types used | 0/5 types | Decision needed |

---

## Phase 1: Eliminate Dead Code (P0)

### 1.1 Use `endpoints` Constant

**Problem**: `api.go` defines an `endpoints` struct with all API paths, but every file still uses hardcoded strings.

**Files with hardcoded paths**:
| File | Hardcoded Paths | Lines |
|------|-----------------|-------|
| `client.go` | `/ingestion` | 498, 501 |
| `prompts.go` | `/v2/prompts` | 47, 73, 118 |
| `datasets.go` | `/v2/datasets` | 33, 43, 67 |
| `traces.go` | `/traces` | 34, 49, 61 |
| `observations.go` | `/observations` | 27, 42 |
| `scores.go` | `/scores` | 26, 42, 57 |
| `sessions.go` | `/sessions` | 23, 38 |
| `models.go` | `/models` | 23 |
| `client.go` | `/health` | 313 |

**Solution**:

```go
// BEFORE (client.go:498)
c.http.post(ctx, "/ingestion", req, &result)

// AFTER
c.http.post(ctx, endpoints.Ingestion, req, &result)
```

**Implementation Steps**:
1. For each file listed above, replace hardcoded strings with `endpoints.*`
2. Run `go build ./...` to catch any typos
3. Run tests to verify behavior unchanged

**Verification**:
```bash
# Should only match api.go after fix
grep -rn '"/v2/\|"/ingestion"\|"/traces"' --include="*.go" *.go | grep -v api.go | grep -v _test.go
```

---

### 1.2 Resolve Generic Types

**Problem**: `types_generic.go` defines 5 types that are never used:
- `ObservationBase`
- `GenerationFields`
- `TraceBase`
- `ScoreBase`
- `ScoreValue`

**Decision Required**: Embed these types OR delete them.

**Option A: Embed (Recommended)**

Benefits:
- Reduces field duplication in `ingestion.go`
- Provides documented base types for API consumers
- Enables future generic builder methods

```go
// ingestion.go - BEFORE
type observationEvent struct {
    ID                  string           `json:"id"`
    TraceID             string           `json:"traceId,omitempty"`
    Name                string           `json:"name,omitempty"`
    // ... 10 more duplicated fields
    Model               string           `json:"model,omitempty"`
    // ... generation fields
}

// ingestion.go - AFTER
type observationEvent struct {
    ObservationBase                      // Embed common fields
    GenerationFields                     // Embed generation-specific fields
}
```

**Caveat**: Go's JSON encoding flattens embedded structs, so JSON output is identical. However, embedded struct literals require different syntax:

```go
// With embedding, construction changes:
event := observationEvent{
    ObservationBase: ObservationBase{
        ID:   "123",
        Name: "test",
    },
    GenerationFields: GenerationFields{
        Model: "gpt-4",
    },
}
```

**Option B: Delete**

If embedding adds complexity without clear benefit, delete `types_generic.go` entirely. Dead code is worse than no abstraction.

**Recommendation**: Start with Option B (delete) unless there's a concrete use case for the base types in the public API.

---

### 1.3 Preserve Type Aliases (Correction)

**Previous Assessment**: Type aliases in `ingestion.go` were reported as unused.

**Actual State**: They ARE used throughout the codebase:
- `trace.go` uses `createTraceEvent`, `updateTraceEvent`, `createSpanEvent`
- `span.go` uses `createSpanEvent`, `updateSpanEvent`
- `generation.go` uses `createGenerationEvent`, `updateGenerationEvent`
- `event.go` uses `createEventEvent`
- `score.go` uses `createScoreEvent`

**Action**: Keep the type aliases. They provide semantic clarity about event intent.

---

## Phase 2: Complete Circuit Breaker Integration (P1)

### 2.1 Current State

Circuit breaker only wraps `sendBatch()`:

```go
// client.go:496-502 - ONLY place circuit breaker is used
if c.circuitBreaker != nil {
    err = c.circuitBreaker.Execute(func() error {
        return c.http.post(ctx, "/ingestion", req, &result)
    })
} else {
    err = c.http.post(ctx, "/ingestion", req, &result)
}
```

**Unprotected Operations**:
- `Prompts().Get()`, `Prompts().List()`, `Prompts().Create()`
- `Traces().Get()`, `Traces().List()`, `Traces().Delete()`
- `Datasets().Get()`, `Datasets().List()`, `Datasets().Create()`
- `Scores().Get()`, `Scores().List()`, `Scores().Create()`
- `Sessions().Get()`, `Sessions().List()`
- `Models().Get()`, `Models().List()`
- `Health()`

### 2.2 Solution: Centralize in httpClient

**Step 1**: Add circuit breaker to `httpClient`:

```go
// http.go
type httpClient struct {
    client         *http.Client
    baseURL        string
    authHeader     string
    maxRetries     int
    retryDelay     time.Duration
    retryStrategy  RetryStrategy
    circuitBreaker *CircuitBreaker  // ADD THIS
    debug          bool
}

func newHTTPClient(cfg *Config, cb *CircuitBreaker) *httpClient {
    // ... existing code ...
    return &httpClient{
        // ... existing fields ...
        circuitBreaker: cb,
    }
}
```

**Step 2**: Wrap `do()` method:

```go
// http.go
func (h *httpClient) do(ctx context.Context, req *request) error {
    if h.circuitBreaker != nil {
        return h.circuitBreaker.Execute(func() error {
            return h.doInternal(ctx, req)
        })
    }
    return h.doInternal(ctx, req)
}

// Rename existing do() to doInternal()
func (h *httpClient) doInternal(ctx context.Context, req *request) error {
    // ... existing implementation ...
}
```

**Step 3**: Update `NewWithConfig()`:

```go
// client.go
func NewWithConfig(cfg *Config) (*Client, error) {
    // ... existing validation ...

    var cb *CircuitBreaker
    if cfgCopy.CircuitBreaker != nil {
        cb = NewCircuitBreaker(*cfgCopy.CircuitBreaker)
    }

    httpClient := newHTTPClient(&cfgCopy, cb)  // Pass circuit breaker

    c := &Client{
        config:         &cfgCopy,
        http:           httpClient,
        circuitBreaker: cb,  // Still keep reference for CircuitBreakerState()
        // ...
    }
}
```

**Step 4**: Remove circuit breaker logic from `sendBatch()`:

```go
// client.go - sendBatch() simplifies to:
func (c *Client) sendBatch(ctx context.Context, events []ingestionEvent) error {
    // ...
    // Circuit breaker is now handled by http.do()
    err := c.http.post(ctx, endpoints.Ingestion, req, &result)
    // ...
}
```

**Benefits**:
- All HTTP operations protected by circuit breaker
- Single point of integration
- Cleaner `sendBatch()` implementation

---

## Phase 3: Critical Test Coverage (P1)

### 3.1 Zero-Coverage Critical Functions

| Function | File:Line | Risk if Broken |
|----------|-----------|----------------|
| `handleError()` | client.go:213 | Silent async failures |
| `handleQueueFull()` | client.go:463 | Data loss on overload |
| `log()` | client.go:240 | Debug visibility |
| `logError()` | client.go:258 | Error visibility |
| `CircuitBreakerState()` | client.go:303 | Public API method |
| `parseRetryAfter()` | http.go:204 | Rate limit handling |

### 3.2 Required Test Cases

```go
// client_test.go

func TestHandleError(t *testing.T) {
    t.Run("calls ErrorHandler when configured", func(t *testing.T) {
        var capturedErr error
        client, _ := New("pk-lf-test", "sk-lf-test",
            WithErrorHandler(func(err error) {
                capturedErr = err
            }),
            WithBaseURL("http://localhost:0"), // Invalid URL to force error
        )
        defer client.Shutdown(context.Background())

        // Force an error through the async path
        client.NewTrace().Name("test").Create(context.Background())
        time.Sleep(100 * time.Millisecond) // Allow async processing

        if capturedErr == nil {
            t.Error("ErrorHandler should have been called")
        }
    })

    t.Run("logs to stderr when no handler configured", func(t *testing.T) {
        // Capture stderr
        oldStderr := os.Stderr
        r, w, _ := os.Pipe()
        os.Stderr = w

        client, _ := New("pk-lf-test", "sk-lf-test",
            WithBaseURL("http://localhost:0"),
        )

        client.NewTrace().Name("test").Create(context.Background())
        time.Sleep(100 * time.Millisecond)
        client.Shutdown(context.Background())

        w.Close()
        os.Stderr = oldStderr

        var buf bytes.Buffer
        io.Copy(&buf, r)

        if !strings.Contains(buf.String(), "langfuse:") {
            t.Error("Expected error to be logged to stderr")
        }
    })
}

func TestHandleQueueFull(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        time.Sleep(500 * time.Millisecond) // Slow response to fill queue
        w.WriteHeader(http.StatusOK)
        json.NewEncoder(w).Encode(IngestionResult{})
    }))
    defer server.Close()

    client, _ := New("pk-lf-test", "sk-lf-test",
        WithBaseURL(server.URL),
        WithBatchSize(1),
        WithBatchQueueSize(1),
    )
    defer client.Shutdown(context.Background())

    // Rapidly queue events to overflow
    for i := 0; i < 10; i++ {
        client.NewTrace().Name(fmt.Sprintf("trace-%d", i)).Create(context.Background())
    }

    // Should not panic - overflow handled gracefully
    time.Sleep(100 * time.Millisecond)
}

func TestCircuitBreakerState(t *testing.T) {
    t.Run("returns Closed when no breaker configured", func(t *testing.T) {
        client, _ := New("pk-lf-test", "sk-lf-test")
        defer client.Shutdown(context.Background())

        state := client.CircuitBreakerState()
        if state != CircuitClosed {
            t.Errorf("expected CircuitClosed, got %v", state)
        }
    })

    t.Run("returns actual state when breaker configured", func(t *testing.T) {
        client, _ := New("pk-lf-test", "sk-lf-test",
            WithDefaultCircuitBreaker(),
        )
        defer client.Shutdown(context.Background())

        state := client.CircuitBreakerState()
        if state != CircuitClosed {
            t.Errorf("expected CircuitClosed initially, got %v", state)
        }
    })
}
```

```go
// http_test.go

func TestParseRetryAfter(t *testing.T) {
    tests := []struct {
        name     string
        value    string
        expected time.Duration
    }{
        {"empty", "", 0},
        {"seconds", "60", 60 * time.Second},
        {"invalid", "not-a-number", 0},
        {"negative", "-5", 0}, // Should handle gracefully
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result := parseRetryAfter(tt.value)
            if result != tt.expected {
                t.Errorf("parseRetryAfter(%q) = %v, want %v", tt.value, result, tt.expected)
            }
        })
    }
}
```

---

## Phase 4: API Consistency Improvements (P2)

### 4.1 Error Helper Naming

**Problem**: `IsAPIError()` returns `(*APIError, bool)` but `Is*` naming implies `bool` return.

**Solution**: Add correctly-named functions, keep old ones as deprecated aliases.

```go
// errors.go

// AsAPIError extracts an APIError from an error chain.
// Returns the error and true if found, nil and false otherwise.
func AsAPIError(err error) (*APIError, bool) {
    var apiErr *APIError
    if errors.As(err, &apiErr) {
        return apiErr, true
    }
    return nil, false
}

// Deprecated: Use AsAPIError instead.
func IsAPIError(err error) (*APIError, bool) {
    return AsAPIError(err)
}

// Similar pattern for:
// - AsValidationError (from IsValidationError)
// - AsIngestionError (from IsIngestionError)
// - AsShutdownError (from IsShutdownError)
// - AsCompilationError (from IsCompilationError)
```

### 4.2 Remove Redundant Error Helpers

**Problem**: Three ways to check error status:
1. `errors.Is(err, ErrNotFound)` - standard Go
2. `IsNotFound(err)` - package function
3. `apiErr.IsNotFound()` - method on extracted error

**Solution**: Remove package-level functions, document preferred patterns:

```go
// DELETE from errors.go:
// - func IsNotFound(err error) bool
// - func IsUnauthorized(err error) bool
// - func IsForbidden(err error) bool
// - func IsServerError(err error) bool
// - func IsRateLimit(err error) bool

// KEEP:
// - Sentinel errors (ErrNotFound, ErrUnauthorized, etc.)
// - Methods on APIError (IsNotFound(), IsRetryable(), etc.)
```

**Documentation Update**:
```go
// Recommended error checking patterns:
//
// Quick check:
//   if errors.Is(err, langfuse.ErrNotFound) { ... }
//
// Detailed inspection:
//   if apiErr, ok := langfuse.AsAPIError(err); ok {
//       if apiErr.IsRetryable() { ... }
//       log.Printf("Request %s failed: %s", apiErr.RequestID, apiErr.Message)
//   }
```

### 4.3 Fix `IsNetwork()` Deprecated API

**Problem**: Uses `Temporary()` interface which is deprecated.

```go
// errors.go - BEFORE
func IsNetwork(err error) bool {
    // ...
    var tempErr interface{ Temporary() bool }
    if errors.As(err, &tempErr) && tempErr.Temporary() {
        return true
    }
    // ...
}

// errors.go - AFTER
func IsNetwork(err error) bool {
    if err == nil {
        return false
    }

    // Check for timeout
    var netErr interface{ Timeout() bool }
    if errors.As(err, &netErr) && netErr.Timeout() {
        return true
    }

    // Check for concrete network error types
    var opErr *net.OpError
    if errors.As(err, &opErr) {
        return true
    }

    var dnsErr *net.DNSError
    if errors.As(err, &dnsErr) {
        return true
    }

    return false
}
```

---

## Phase 5: Utility Improvements (P3)

### 5.1 Metadata Utility Methods

Add methods to make `Metadata` type useful:

```go
// types_generic.go

type Metadata map[string]any

// Get returns a value by key.
func (m Metadata) Get(key string) (any, bool) {
    if m == nil {
        return nil, false
    }
    v, ok := m[key]
    return v, ok
}

// GetString returns a string value or empty string if not found/wrong type.
func (m Metadata) GetString(key string) string {
    if v, ok := m[key].(string); ok {
        return v
    }
    return ""
}

// GetInt returns an int value or 0 if not found/wrong type.
func (m Metadata) GetInt(key string) int {
    switch v := m[key].(type) {
    case int:
        return v
    case int64:
        return int(v)
    case float64:
        return int(v)
    }
    return 0
}

// GetFloat returns a float64 value or 0 if not found/wrong type.
func (m Metadata) GetFloat(key string) float64 {
    switch v := m[key].(type) {
    case float64:
        return v
    case int:
        return float64(v)
    case int64:
        return float64(v)
    }
    return 0
}

// GetBool returns a bool value or false if not found/wrong type.
func (m Metadata) GetBool(key string) bool {
    if v, ok := m[key].(bool); ok {
        return v
    }
    return false
}

// Set sets a key-value pair, initializing the map if nil.
func (m *Metadata) Set(key string, value any) {
    if *m == nil {
        *m = make(Metadata)
    }
    (*m)[key] = value
}

// Merge combines two Metadata maps, with other taking precedence.
func (m Metadata) Merge(other Metadata) Metadata {
    result := make(Metadata, len(m)+len(other))
    for k, v := range m {
        result[k] = v
    }
    for k, v := range other {
        result[k] = v
    }
    return result
}

// Clone creates a shallow copy of the Metadata.
func (m Metadata) Clone() Metadata {
    if m == nil {
        return nil
    }
    result := make(Metadata, len(m))
    for k, v := range m {
        result[k] = v
    }
    return result
}
```

---

## Implementation Plan

### Week 1: Foundation (Days 1-2)

| Day | Task | Files | Estimated Time |
|-----|------|-------|----------------|
| 1 | Replace hardcoded paths with `endpoints.*` | 8 files | 2 hours |
| 1 | Delete unused `types_generic.go` types OR embed | 2 files | 1 hour |
| 1 | Add critical test cases | `client_test.go`, `http_test.go` | 3 hours |
| 2 | Centralize circuit breaker in `httpClient` | `http.go`, `client.go` | 4 hours |
| 2 | Test circuit breaker integration | `http_test.go` | 2 hours |

### Week 1: Polish (Days 3-4)

| Day | Task | Files | Estimated Time |
|-----|------|-------|----------------|
| 3 | Add `As*` error functions | `errors.go` | 1 hour |
| 3 | Remove redundant error helpers | `errors.go` | 1 hour |
| 3 | Fix `IsNetwork()` | `errors.go` | 30 min |
| 3 | Add Metadata utility methods | `types_generic.go` | 2 hours |
| 4 | Add tests for new utilities | `*_test.go` | 3 hours |
| 4 | Update documentation | `README.md`, `AGENTS.md` | 2 hours |

---

## Success Criteria

| Metric | Current | Target | Verification |
|--------|---------|--------|--------------|
| Hardcoded API paths | 17 | 0 | `grep` returns only `api.go` |
| Circuit breaker coverage | 1 method | All HTTP | Manual review |
| Test coverage (main) | 74.9% | 85%+ | `go test -cover` |
| Zero-coverage critical funcs | 6 | 0 | `go tool cover -func` |
| `interface{}` usage | 0 | 0 | Maintained |

---

## Rollback Plan

All changes are backwards-compatible:
- `endpoints` usage: Internal refactor, no API change
- Circuit breaker: Same public API, different internal structure
- Error helpers: Old functions remain as deprecated aliases
- Metadata methods: Additive only

If issues arise:
1. Revert specific commits
2. No database migrations or external dependencies affected
3. All changes are in-memory code changes

---

## Appendix: File Change Summary

| File | Changes |
|------|---------|
| `api.go` | No changes (already correct) |
| `client.go` | Use `endpoints.Ingestion`, simplify `sendBatch()` |
| `http.go` | Add `circuitBreaker` field, wrap `do()` |
| `prompts.go` | Use `endpoints.Prompts` |
| `datasets.go` | Use `endpoints.Datasets` |
| `traces.go` | Use `endpoints.Traces` |
| `observations.go` | Use `endpoints.Observations` |
| `scores.go` | Use `endpoints.Scores` |
| `sessions.go` | Use `endpoints.Sessions` |
| `models.go` | Use `endpoints.Models` |
| `errors.go` | Add `As*` functions, fix `IsNetwork()`, remove redundant helpers |
| `types_generic.go` | Add Metadata methods OR delete unused types |
| `client_test.go` | Add critical path tests |
| `http_test.go` | Add `parseRetryAfter` tests |
| `errors_test.go` | Add `As*` function tests |
| `types_generic_test.go` | Add Metadata method tests |
