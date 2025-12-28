# AGENTS.md - AI Coding Assistant Guide

This document provides guidance for AI coding assistants working with the Langfuse Go SDK.

## Project Overview

This is a pure Go SDK for [Langfuse](https://langfuse.com/), an open-source LLM observability platform. The SDK provides:

- Zero external dependencies (stdlib only)
- Type-safe API with fluent builders
- Automatic event batching and async flushing
- Thread-safe concurrent operations
- Full API coverage for traces, observations, scores, prompts, datasets, and more

**Module**: `github.com/jdziat/langfuse-go`
**Go Version**: 1.23+
**License**: MIT

## Repository Structure

```
langfuse-go/
├── *.go                    # Core SDK source (single package)
├── *_test.go               # Unit tests for each module
├── examples/
│   ├── basic/main.go       # Simple usage example
│   └── advanced/main.go    # Complex workflows
├── docs/proposals/         # Future enhancement proposals
├── .github/workflows/      # CI/CD pipelines
├── .goreleaser.yaml        # Release automation
└── README.md               # User documentation
```

## Key Source Files

| File | Purpose |
|------|---------|
| `client.go` | Main client, trace builders, event queue management |
| `config.go` | Configuration options and defaults |
| `types.go` | Core data types (Trace, Observation, Score, etc.) |
| `ingestion.go` | Event builders (Span, Generation, Event, Score) |
| `http.go` | HTTP client with retry logic |
| `errors.go` | Error types and sentinel errors |
| `traces.go` | Traces API sub-client |
| `observations.go` | Observations API sub-client |
| `scores.go` | Scores API sub-client |
| `prompts.go` | Prompts API sub-client |
| `datasets.go` | Datasets API sub-client |
| `sessions.go` | Sessions API sub-client |
| `models.go` | Models API sub-client |

## Coding Patterns

### Fluent Builder Pattern

All entity creation uses fluent builders:

```go
trace, err := client.NewTrace().
    Name("operation").
    UserID("user-123").
    Input(data).
    Tags([]string{"prod"}).
    Create()
```

- Builders return `*BuilderType` for method chaining
- `.Create()` or `.Apply()` finalizes and queues the event
- Builders wrap internal event structs

### Context Hierarchy

Observations nest via embedded contexts:

```
TraceContext
├── SpanContext (embeds TraceContext)
│   └── Child spans, generations, events
├── GenerationContext
└── EventContext
```

### Sub-Client Pattern

Each API area has a dedicated sub-client:

```go
client.Traces()      // *TracesClient
client.Scores()      // *ScoresClient
client.Prompts()     // *PromptsClient
// etc.
```

Sub-clients use the main client's HTTP client and event queue.

### Error Handling

- Return explicit errors from all public methods
- Use sentinel errors for configuration validation (`ErrMissingPublicKey`, etc.)
- Use `errors.Is()` for checking specific error types
- `APIError` wraps HTTP responses with helper methods (`IsRetryable()`, etc.)

### JSON Serialization

- Use `json:"fieldName,omitempty"` tags
- Custom `Time` type handles RFC3339Nano format
- Nullable fields use pointers
- Omitempty prevents sending empty values

## Testing Requirements

### Before Committing

Always run:

```bash
go test -v -race ./...
go vet ./...
go fmt ./...
```

### Test Patterns

- **Mock HTTP**: Use `httptest.NewServer()` for API tests
- **Table-driven tests**: For validation and configuration scenarios
- **Race detection**: All tests run with `-race` flag in CI

### Test File Naming

Each source file has a corresponding `*_test.go`:
- `client.go` → `client_test.go`
- `ingestion.go` → `ingestion_test.go`

## Thread Safety

The client uses `sync.Mutex` to protect:
- `pendingEvents` slice
- `flushTimer`
- `closed` flag

All public API methods must acquire locks when accessing shared state.

## Event Batching

Events are batched automatically:

1. Events queued via `queueEvent()` to `pendingEvents[]`
2. Flush triggers:
   - Batch reaches `BatchSize` (default 100) → async flush
   - `FlushInterval` timer fires (default 5s) → background flush
   - Explicit `client.Flush(ctx)` → synchronous flush
   - `client.Shutdown(ctx)` → final synchronous flush

## Common Tasks

### Adding a New Builder Type

1. Define the event struct in `ingestion.go`
2. Create the builder struct with fluent methods
3. Add the context struct for observation management
4. Wire to parent context (TraceContext or SpanContext)
5. Add tests with httptest mock server

### Adding a New API Sub-Client

1. Create new file (e.g., `newfeature.go`)
2. Define the sub-client struct holding `*Client`
3. Implement CRUD methods using `client.http`
4. Add accessor method to main Client
5. Create corresponding `*_test.go`
6. Update README.md with examples

### Adding New Configuration Options

1. Add field to `Config` struct in `config.go`
2. Create `ConfigOption` function (e.g., `WithNewOption`)
3. Set sensible default in `NewClient()`
4. Add validation if needed
5. Document in README.md

## Do's and Don'ts

### Do

- Follow the fluent builder pattern for new entity types
- Use `context.Context` for all HTTP operations
- Return errors explicitly (no panics)
- Use table-driven tests
- Protect shared state with mutex
- Use `json:",omitempty"` for optional fields
- Add doc comments on exported types and methods
- Run `go fmt` before committing

### Don't

- Add external dependencies (keep stdlib only)
- Block on network I/O in builder methods
- Expose internal event structs
- Use global state
- Skip the race detector in tests
- Forget to update README.md for new features

## Commit Message Convention

Follow conventional commits for automated changelog generation:

- `feat:` - New features
- `fix:` - Bug fixes
- `perf:` - Performance improvements
- `docs:` - Documentation changes
- `test:` - Test additions/changes
- `refactor:` - Code refactoring

## CI/CD

### CI Pipeline (`.github/workflows/ci.yml`)

- Tests on Go 1.23 and 1.24
- Race detector enabled
- Coverage uploaded to Codecov
- Linting: `go vet`, `gofmt`, `staticcheck`

### Release Pipeline (`.github/workflows/release.yml`)

- Triggered by version tags (`v*`)
- Uses GoReleaser for semantic versioning
- Auto-generates changelog from commits

## Configuration Defaults

| Setting | Default | Description |
|---------|---------|-------------|
| `BatchSize` | 100 | Events per batch |
| `FlushInterval` | 5s | Auto-flush interval |
| `Timeout` | 30s | HTTP request timeout |
| `MaxRetries` | 3 | Retry attempts |
| `RetryDelay` | 1s | Initial retry delay |

## Regions

```go
RegionUS   = "us"      // https://us.cloud.langfuse.com
RegionEU   = "eu"      // https://cloud.langfuse.com
RegionHIPAA = "hipaa"  // https://hipaa.cloud.langfuse.com
```

## Useful Commands

```bash
# Run all tests with race detection
go test -v -race ./...

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Check formatting
gofmt -l .

# Static analysis
go vet ./...
staticcheck ./...

# Build examples
go build -o /dev/null ./examples/basic
go build -o /dev/null ./examples/advanced

# Tidy dependencies
go mod tidy
```

---

## Technical Debt & Remediation Guide

This section documents known technical debt and provides actionable remediation steps. **AI agents should prioritize these issues when making changes to the codebase.**

### Priority Matrix

| Issue | Severity | Effort | Priority |
|-------|----------|--------|----------|
| Dead code: `endpoints` unused | High | Low | **P0** |
| Dead code: `types_generic.go` unused | High | Medium | **P0** |
| Dead code: event type aliases unused | Medium | Low | **P0** |
| Test coverage gaps (critical paths) | High | Medium | **P1** |
| Circuit breaker partial integration | High | Medium | **P1** |
| Error helper naming (`Is*` vs `As*`) | Low | Low | **P2** |
| Error helper redundancy | Medium | Low | **P2** |
| `IsNetwork()` uses deprecated API | Medium | Low | **P2** |
| `Metadata` type lacks utility methods | Low | Medium | **P3** |

---

### Issue 1: `endpoints` Constant Not Used (P0)

**Location**: `api.go` defines `endpoints` struct, but all API calls use hardcoded strings.

**Problem**:
```go
// api.go - Defined but ignored
var endpoints = struct {
    Ingestion string
    Prompts   string
    // ...
}{
    Ingestion: "/ingestion",
    Prompts:   apiV2 + "/prompts",
}

// client.go:498 - Hardcoded!
c.http.post(ctx, "/ingestion", req, &result)
```

**Remediation**:
1. Replace ALL hardcoded paths with `endpoints.*` references
2. Files to update:
   - `client.go`: Replace `"/ingestion"` with `endpoints.Ingestion`
   - `prompts.go`: Replace `"/v2/prompts"` with `endpoints.Prompts`
   - `datasets.go`: Replace `"/v2/datasets"` with `endpoints.Datasets`
   - `traces.go`: Replace `"/traces"` with `endpoints.Traces`
   - `observations.go`: Replace `"/observations"` with `endpoints.Observations`
   - `scores.go`: Replace `"/scores"` with `endpoints.Scores`
   - `sessions.go`: Replace `"/sessions"` with `endpoints.Sessions`
   - `models.go`: Replace `"/models"` with `endpoints.Models`

**Verification**:
```bash
# Should return ONLY api.go after fix
grep -rn '"/v2/\|"/ingestion"\|"/traces"\|"/observations"' --include="*.go" *.go
```

---

### Issue 2: Generic Types Not Used (P0)

**Location**: `types_generic.go` defines `ObservationBase`, `TraceBase`, `ScoreBase`, `GenerationFields`, `ScoreValue` but none are used.

**Problem**: These types were created for "consolidation" but never integrated.

**Remediation Option A (Embed in event structs)**:
```go
// ingestion.go - Use embedding
type observationEvent struct {
    ObservationBase              // Embed instead of duplicating fields
    Model               string   `json:"model,omitempty"`
    // ... generation-specific only
}
```

**Remediation Option B (Delete if not needed)**:
If embedding causes JSON serialization issues or complexity, delete the unused types entirely. Dead code is worse than no abstraction.

**Verification**:
```bash
# Should return matches in files OTHER than types_generic.go
grep -rn "ObservationBase\|TraceBase\|ScoreBase" --include="*.go" *.go | grep -v types_generic.go
```

---

### Issue 3: Event Type Aliases Unused (P0)

**Location**: `ingestion.go` lines 97-106

**Problem**:
```go
type (
    createTraceEvent      = traceEvent       // Never referenced
    updateTraceEvent      = traceEvent       // Never referenced
    createSpanEvent       = observationEvent // Never referenced
    // ... 5 more unused aliases
)
```

**Remediation**: Delete all type aliases. They provide no value.

```go
// DELETE this entire block from ingestion.go
type (
    createTraceEvent      = traceEvent
    updateTraceEvent      = traceEvent
    createSpanEvent       = observationEvent
    updateSpanEvent       = observationEvent
    createGenerationEvent = observationEvent
    updateGenerationEvent = observationEvent
    createEventEvent      = observationEvent
    createScoreEvent      = scoreEvent
)
```

---

### Issue 4: Critical Test Coverage Gaps (P1)

**Functions with 0% coverage that MUST be tested**:

| Function | File | Why Critical |
|----------|------|--------------|
| `handleError()` | client.go:213 | Async error handling - silent failures if broken |
| `handleQueueFull()` | client.go:463 | Queue overflow - data loss if broken |
| `log()` | client.go:240 | Debug output |
| `logError()` | client.go:258 | Error logging |
| `CircuitBreakerState()` | client.go:303 | New public API method |

**Required Tests**:

```go
// client_test.go - Add these test cases

func TestHandleError(t *testing.T) {
    t.Run("calls ErrorHandler when set", func(t *testing.T) {
        var captured error
        client, _ := New("pk-test", "sk-test",
            WithErrorHandler(func(err error) { captured = err }),
        )
        defer client.Shutdown(context.Background())

        // Trigger handleError via a failed batch
        // Assert captured != nil
    })

    t.Run("logs to stderr when no handler", func(t *testing.T) {
        // Capture stderr, trigger error, verify output
    })
}

func TestHandleQueueFull(t *testing.T) {
    // Create client with BatchQueueSize: 1
    // Fill the queue
    // Verify overflow handled without panic
}

func TestCircuitBreakerState(t *testing.T) {
    t.Run("returns Closed when no breaker configured", func(t *testing.T) {
        client, _ := New("pk-test", "sk-test")
        defer client.Shutdown(context.Background())

        if client.CircuitBreakerState() != CircuitClosed {
            t.Error("expected CircuitClosed")
        }
    })
}
```

---

### Issue 5: Circuit Breaker Only Protects Ingestion (P1)

**Problem**: Circuit breaker wraps only `sendBatch()`. All read operations bypass it.

**Current State**:
```go
// client.go - Only ingestion protected
func (c *Client) sendBatch(...) {
    if c.circuitBreaker != nil {
        err = c.circuitBreaker.Execute(...)
    }
}

// prompts.go - NOT protected
func (c *PromptsClient) Get(...) {
    return c.client.http.get(...)  // No circuit breaker!
}
```

**Remediation**: Add circuit breaker wrapper to `httpClient`:

```go
// http.go - Add method
func (h *httpClient) doWithCircuitBreaker(ctx context.Context, cb *CircuitBreaker, req *request) error {
    if cb == nil {
        return h.do(ctx, req)
    }
    return cb.Execute(func() error {
        return h.do(ctx, req)
    })
}

// Then update all sub-clients to use:
func (c *PromptsClient) Get(ctx context.Context, name string, params *GetPromptParams) (*Prompt, error) {
    // ...
    err := c.client.http.doWithCircuitBreaker(ctx, c.client.circuitBreaker, &request{...})
}
```

**Alternative**: Pass circuit breaker to `httpClient` at construction and wrap all calls internally.

---

### Issue 6: Error Helper Naming Convention (P2)

**Problem**: Functions named `Is*` return `(*T, bool)` which is the `As*` pattern.

**Current**:
```go
func IsAPIError(err error) (*APIError, bool)      // Should be AsAPIError
func IsValidationError(err error) (*ValidationError, bool)  // Should be AsValidationError
```

**Go Convention**:
- `errors.Is(err, target)` → returns `bool`
- `errors.As(err, &target)` → extracts into pointer

**Remediation**: Add correctly-named functions, deprecate old ones:

```go
// errors.go - Add new functions
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
```

---

### Issue 7: Redundant Error Helpers (P2)

**Problem**: Two ways to check every error condition.

```go
// Package-level functions
func IsNotFound(err error) bool
func IsUnauthorized(err error) bool

// PLUS methods on APIError
func (e *APIError) IsNotFound() bool
func (e *APIError) IsUnauthorized() bool

// PLUS errors.Is support
errors.Is(err, ErrNotFound)
```

**Remediation**: Keep only `errors.Is()` pattern and methods on `APIError`:

```go
// DELETE these package-level functions:
// - IsNotFound()
// - IsUnauthorized()
// - IsForbidden()
// - IsServerError()
// - IsRateLimit()

// KEEP:
// - errors.Is(err, ErrNotFound) - standard Go
// - apiErr.IsNotFound() - after extraction
```

Update documentation to show:
```go
if errors.Is(err, langfuse.ErrNotFound) {
    // Handle 404
}

// Or for detailed inspection:
if apiErr, ok := langfuse.AsAPIError(err); ok {
    if apiErr.IsNotFound() { ... }
}
```

---

### Issue 8: `IsNetwork()` Uses Deprecated Interface (P2)

**Location**: `errors.go:275-290`

**Problem**:
```go
var tempErr interface{ Temporary() bool }
if errors.As(err, &tempErr) && tempErr.Temporary() {
    return true
}
```

The `Temporary()` method is deprecated in Go's `net` package and returns unreliable values.

**Remediation**:
```go
func IsNetwork(err error) bool {
    if err == nil {
        return false
    }

    // Check for timeout
    var netErr interface{ Timeout() bool }
    if errors.As(err, &netErr) && netErr.Timeout() {
        return true
    }

    // Check for common network error types
    var opErr *net.OpError
    if errors.As(err, &opErr) {
        return true
    }

    var dnsErr *net.DNSError
    if errors.As(err, &dnsErr) {
        return true
    }

    // Check error string as last resort
    errStr := err.Error()
    return strings.Contains(errStr, "connection refused") ||
           strings.Contains(errStr, "no such host") ||
           strings.Contains(errStr, "network is unreachable")
}
```

---

### Issue 9: `Metadata` Type Lacks Utility (P3)

**Problem**: `type Metadata map[string]any` is just an alias with no added value.

**Remediation**: Add utility methods:

```go
// types_generic.go

type Metadata map[string]any

// Get returns a value with type assertion.
func (m Metadata) Get(key string) (any, bool) {
    v, ok := m[key]
    return v, ok
}

// GetString returns a string value or empty string.
func (m Metadata) GetString(key string) string {
    if v, ok := m[key].(string); ok {
        return v
    }
    return ""
}

// GetInt returns an int value or 0.
func (m Metadata) GetInt(key string) int {
    switch v := m[key].(type) {
    case int:
        return v
    case float64:
        return int(v)
    }
    return 0
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
```

---

### Validation Checklist

Before any PR, verify:

```bash
# 1. No hardcoded API paths (except api.go)
grep -rn '"/v2/\|"/ingestion"\|"/traces"' --include="*.go" *.go | grep -v api.go | grep -v _test.go
# Expected: No output

# 2. No unused type definitions
grep -rn "ObservationBase\|TraceBase" --include="*.go" *.go | grep -v types_generic.go
# Expected: Matches showing actual usage

# 3. No interface{} (use any)
grep -r "interface{}" --include="*.go" *.go
# Expected: No output

# 4. Test coverage > 80%
go test -coverprofile=cov.out ./... && go tool cover -func=cov.out | grep total
# Expected: > 80%

# 5. Critical functions tested
go test -v -run "TestHandleError\|TestHandleQueueFull\|TestCircuitBreakerState" ./...
# Expected: All pass
```

---

### When Adding New Features

1. **Use `endpoints.*`** for all API paths
2. **Compose with existing types** from `types_generic.go` if applicable
3. **Add circuit breaker support** for any new HTTP operations
4. **Test error paths** - not just happy paths
5. **Use `Metadata`** instead of raw `map[string]any` in public APIs
6. **Follow `As*` naming** for error extraction functions
