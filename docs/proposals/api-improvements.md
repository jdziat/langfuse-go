# API Accessibility and Go Idioms Proposal

## Executive Summary

This proposal addresses opportunities to make the langfuse-go SDK more accessible to Go developers and better aligned with Go idioms. The focus is on reducing cognitive load, improving discoverability, and following established Go conventions.

---

## Current State Analysis

### Strengths
- Well-implemented fluent builder pattern
- Good separation of concerns with subpackages
- Comprehensive feature coverage
- Proper concurrency handling

### Areas for Improvement

| Issue | Impact | Priority |
|-------|--------|----------|
| Large config.go (711 lines) | Hard to navigate | Medium |
| 30+ ConfigOption functions | Overwhelming API surface | High |
| Deprecated methods not removed | Confusing for new users | Medium |
| Inconsistent naming conventions | Cognitive load | Medium |
| Silent error handling in ingestion | Debugging difficulty | High |
| Custom Time type confusion | Unexpected behavior | Medium |
| Missing interface definitions | Hard to mock | High |

---

## Proposal 1: Simplify Configuration

### Problem
The current API has 30+ `ConfigOption` functions, making it overwhelming for new users. Most users only need 2-3 options.

### Current API
```go
client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithRegion(langfuse.RegionUS),
    langfuse.WithBatchSize(100),
    langfuse.WithFlushInterval(5*time.Second),
    langfuse.WithTimeout(30*time.Second),
    langfuse.WithMaxRetries(3),
    langfuse.WithRetryDelay(time.Second),
    langfuse.WithDebug(true),
    langfuse.WithLogger(logger),
    // ... 20+ more options
)
```

### Proposed Solution: Tiered Configuration

**Tier 1: Simple API (90% of users)**
```go
// Most users just need this
client, err := langfuse.New(publicKey, secretKey)

// Or with region
client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithRegion(langfuse.RegionEU))
```

**Tier 2: Config Struct (power users)**
```go
// For users who need multiple options
client, err := langfuse.NewWithConfig(langfuse.Config{
    PublicKey:     publicKey,
    SecretKey:     secretKey,
    Region:        langfuse.RegionEU,
    BatchSize:     200,
    FlushInterval: 10 * time.Second,
    Debug:         true,
})
```

**Tier 3: Full Control (advanced users)**
```go
// Keep existing functional options for maximum flexibility
client, err := langfuse.New(publicKey, secretKey,
    langfuse.WithHTTPClient(customClient),
    langfuse.WithRetryStrategy(customStrategy),
)
```

### Implementation

1. Export the `Config` struct (currently internal)
2. Add `NewWithConfig(Config) (*Client, error)` constructor
3. Group related options into sub-structs:

```go
type Config struct {
    // Required
    PublicKey string
    SecretKey string

    // Common options (with sensible defaults)
    Region      Region        // Default: RegionUS
    Environment string        // Default: "default"

    // Batching (optional, has defaults)
    Batching BatchConfig

    // Advanced (optional)
    HTTP     HTTPConfig
    Retry    RetryConfig
    Logging  LogConfig
}

type BatchConfig struct {
    Size          int           // Default: 100
    FlushInterval time.Duration // Default: 5s
    QueueSize     int           // Default: 10000
}
```

### Benefits
- Clear documentation of all options in one place
- IDE autocomplete shows available fields
- Easier to understand default behavior
- Reduces API surface cognitive load

---

## Proposal 2: Define Core Interfaces

### Problem
The SDK lacks explicit interface definitions, making it hard to:
- Mock the client in tests
- Understand the core API contract
- Extend or wrap functionality

### Proposed Interfaces

```go
// Tracer is the core interface for creating traces.
type Tracer interface {
    NewTrace() *TraceBuilder
    Flush(ctx context.Context) error
    Shutdown(ctx context.Context) error
}

// TraceContext represents an active trace.
type TraceContextInterface interface {
    ID() string
    Span() *SpanBuilder
    Generation() *GenerationBuilder
    Event() *EventBuilder
    Score() *ScoreBuilder
    Update() *TraceUpdateBuilder
}

// ObservationContext represents an active observation (span or generation).
type ObservationContext interface {
    ID() string
    TraceID() string
    End() ObservationEndBuilder
    Span() *SpanBuilder
    Generation() *GenerationBuilder
    Event() *EventBuilder
    Score() *ScoreBuilder
}
```

### File: `interfaces.go`
```go
package langfuse

// Tracer defines the core tracing interface.
// This interface is implemented by *Client and can be used
// for dependency injection and testing.
type Tracer interface {
    // NewTrace creates a new trace builder.
    NewTrace() *TraceBuilder

    // Flush sends all pending events to Langfuse.
    Flush(ctx context.Context) error

    // Shutdown gracefully shuts down the client.
    Shutdown(ctx context.Context) error
}

// Ensure Client implements Tracer
var _ Tracer = (*Client)(nil)
```

### Benefits
- Users can depend on interfaces, not concrete types
- Easier to create test doubles
- Clear API contract documentation
- Enables wrapper implementations (caching, logging, etc.)

---

## Proposal 3: Improve Error Handling

### Problem 1: Silent Ingestion Errors
Currently, ingestion errors are logged but not returned to the caller. Users may not realize their traces aren't being recorded.

### Current Behavior
```go
trace, err := client.NewTrace().Name("test").Create()
// err is nil even if the trace fails to be queued
// Actual errors only visible via error handler callback
```

### Proposed Solution: Error Channels

```go
// Option 1: Error channel (non-blocking)
client, err := langfuse.New(pk, sk)
go func() {
    for err := range client.Errors() {
        log.Printf("Langfuse error: %v", err)
    }
}()

// Option 2: Synchronous mode for critical paths
client, err := langfuse.New(pk, sk, langfuse.WithSyncMode(true))
trace, err := client.NewTrace().Name("test").Create()
if err != nil {
    // Error returned immediately
}
```

### Problem 2: Error Type Assertions
Users must type-assert to get error details.

### Proposed Solution: Error Helper Functions

```go
// Add to errors.go
func IsRetryable(err error) bool
func IsRateLimit(err error) bool
func IsValidation(err error) bool
func IsNetwork(err error) bool

// Usage
if langfuse.IsRateLimit(err) {
    time.Sleep(langfuse.RetryAfter(err))
}
```

---

## Proposal 4: Naming Consistency

### Issues Found

| Current | Issue | Proposed |
|---------|-------|----------|
| `WithFlushInterval(time.Duration)` | Takes nanoseconds as int64 in some places | Consistently use `time.Duration` |
| `NumericValue` / `CategoricalValue` | Inconsistent with `Value` suffix | Keep as-is (clear intent) |
| `CreateContext` / `ApplyContext` | Verbose, Go prefers shorter names | Keep deprecated versions, add `Create(ctx)` |
| `ObservationID` field | Should be `ParentID` in builder context | Add alias `ParentID()` |

### Proposed Changes

**1. Duration Consistency**
```go
// Current (confusing - uses nanoseconds)
WithFlushInterval(5 * 1e9)

// Proposed (use time.Duration everywhere)
WithFlushInterval(5 * time.Second)
```

**2. Context Method Naming**
```go
// Current
trace, err := builder.CreateContext(ctx)

// Add shorter alias (Go convention)
trace, err := builder.Create(ctx)

// Keep CreateContext as alias for backwards compatibility
```

**3. Parent Reference Clarity**
```go
// Current - confusing name
span.ParentObservationID("obs-123")

// Add clearer alias
span.ParentID("obs-123")
```

---

## Proposal 5: Remove Deprecated APIs

### Current Deprecated Methods
```go
// In trace.go
func (b *TraceBuilder) Create() (*TraceContext, error) // Deprecated

// In span.go, generation.go
func (b *SpanBuilder) Create() (*SpanContext, error) // Deprecated
```

### Proposed Approach

**Phase 1: Add migration helpers (this release)**
```go
// Add to doc.go
//
// # Migration from v0.x
//
// The following changes were made in v1.0:
// - Create() now requires a context parameter: Create(ctx)
// - Use context.Background() if you don't need cancellation
```

**Phase 2: Remove deprecated methods (next major version)**

Since this is a "big bang" reorganization with no users yet, we can:
1. Remove deprecated `Create()` methods
2. Rename `CreateContext()` to `Create()`
3. Update all examples

---

## Proposal 6: Improve Builder Ergonomics

### Problem: Verbose End Patterns
```go
// Current - verbose
gen.End().Output(result).Usage(in, out).Apply()

// If user forgets Apply(), nothing happens (silent failure)
```

### Proposed Solution: Simplified End

```go
// Option 1: End with result directly
gen.EndWith(result, inputTokens, outputTokens)

// Option 2: Functional options for End
gen.End(
    langfuse.WithOutput(result),
    langfuse.WithUsage(in, out),
)

// Option 3: Keep builder but make Apply automatic via defer
defer gen.End().Apply()
// ... do work, set output later
gen.SetOutput(result)
```

### Problem: Score Builder Complexity
```go
// Current - must choose one value type
trace.Score().Name("quality").NumericValue(0.95).Create(ctx)
trace.Score().Name("passed").BooleanValue(true).Create(ctx)
trace.Score().Name("grade").CategoricalValue("A").Create(ctx)
```

### Proposed Solution: Type-Safe Score Helpers
```go
// Add convenience methods on TraceContext
trace.ScoreNumeric(ctx, "quality", 0.95)
trace.ScoreBoolean(ctx, "passed", true)
trace.ScoreCategory(ctx, "grade", "A")

// Or with options for additional fields
trace.ScoreNumeric(ctx, "quality", 0.95,
    langfuse.WithComment("Excellent response"),
)
```

---

## Proposal 7: Split config.go (711 lines)

### Current Structure
```
config.go (711 lines)
├── Config struct
├── ConfigOption type
├── 30+ With* functions
├── Default values
├── Validation logic
├── Region constants
└── Environment loading
```

### Proposed Structure
```
config.go (200 lines)
├── Config struct
├── ConfigOption type
├── NewConfig() with defaults
├── Validate() method
└── Common With* functions (5-6 most used)

options.go (200 lines)
├── Batching options
├── HTTP options
├── Retry options
└── Logging options

regions.go (50 lines)
├── Region type
├── Region constants
└── Region URL mapping

env.go (100 lines)
├── NewFromEnv()
├── Environment variable parsing
└── Default environment detection
```

---

## Proposal 8: Improve Type Safety with Generics

### Problem: Interface{} Usage
```go
// Current
func (b *TraceBuilder) Input(input interface{}) *TraceBuilder
func (b *TraceBuilder) Metadata(metadata map[string]interface{}) *TraceBuilder
```

### Proposed Solution: Type Aliases
```go
// Add type aliases for clarity (Go 1.18+)
type JSON = any
type JSONObject = map[string]any

// Usage remains the same but intent is clearer
func (b *TraceBuilder) Input(input JSON) *TraceBuilder
func (b *TraceBuilder) Metadata(metadata JSONObject) *TraceBuilder
```

### Consideration
Keep `interface{}` in Go 1.18+ codebases for compatibility, but document that any JSON-serializable value is accepted.

---

## Proposal 9: Add Common Workflow Helpers

### Problem: Boilerplate for Common Patterns

```go
// Current - verbose for simple LLM call tracing
trace, _ := client.NewTrace().Name("chat").UserID(userID).Create(ctx)
gen, _ := trace.Generation().Name("gpt-4").Model("gpt-4").Input(prompt).Create(ctx)
// ... call LLM ...
gen.End().Output(response).Usage(tokens.Input, tokens.Output).Apply(ctx)
trace.Update().Output(response).Apply(ctx)
```

### Proposed Solution: High-Level Helpers

```go
// Simple LLM call tracing
result, err := langfuse.TraceGeneration(ctx, client, langfuse.GenerationParams{
    Name:   "chat",
    Model:  "gpt-4",
    Input:  prompt,
    UserID: userID,
}, func() (string, langfuse.Usage, error) {
    // Your LLM call here
    resp, err := openai.Complete(prompt)
    return resp.Text, langfuse.Usage{Input: resp.InputTokens, Output: resp.OutputTokens}, err
})

// The helper handles:
// - Creating trace
// - Creating generation
// - Timing
// - Recording output and usage
// - Error handling
// - Proper cleanup
```

### File: `helpers.go`
```go
package langfuse

// TraceFunc traces a function execution as a span.
func TraceFunc[T any](ctx context.Context, client *Client, name string, fn func() (T, error)) (T, error)

// TraceGeneration traces an LLM generation.
func TraceGeneration(ctx context.Context, client *Client, params GenerationParams, fn GenerationFunc) (string, error)

// GenerationParams configures a traced generation.
type GenerationParams struct {
    Name      string
    Model     string
    Input     any
    UserID    string
    SessionID string
    Metadata  map[string]any
}

// GenerationFunc is called to perform the actual LLM call.
type GenerationFunc func() (output string, usage Usage, err error)
```

---

## Proposal 10: Documentation Improvements

### Current Gaps
1. No migration guide
2. Limited examples in godoc
3. No troubleshooting section
4. No performance tuning guide

### Proposed Documentation Structure

```
docs/
├── README.md                 # Quick start
├── MIGRATION.md              # Version migration guide
├── CONFIGURATION.md          # All configuration options
├── PATTERNS.md               # Common usage patterns
├── TROUBLESHOOTING.md        # Common issues and solutions
├── PERFORMANCE.md            # Tuning for high throughput
└── proposals/                # Design proposals (existing)
```

### Godoc Improvements

Add comprehensive examples to each major type:

```go
// Example usage in trace.go
func ExampleClient_NewTrace() {
    client, _ := langfuse.New("pk", "sk")
    defer client.Shutdown(context.Background())

    trace, err := client.NewTrace().
        Name("example-trace").
        UserID("user-123").
        Create(context.Background())
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Created trace:", trace.ID())
}

func ExampleTraceContext_Generation() {
    // ... complete example
}
```

---

## Implementation Priority

### Phase 1: Quick Wins (Low Risk)
1. [ ] Add core interfaces (`interfaces.go`)
2. [ ] Add error helper functions
3. [ ] Split config.go into smaller files
4. [ ] Add type aliases (JSON, JSONObject)
5. [ ] Improve godoc examples

### Phase 2: API Improvements (Medium Risk)
1. [ ] Add `NewWithConfig()` constructor
2. [ ] Add simplified score helpers
3. [ ] Add `ParentID()` alias
4. [ ] Ensure Duration consistency

### Phase 3: New Features (Higher Risk)
1. [ ] Add error channel for async errors
2. [ ] Add high-level workflow helpers
3. [ ] Add synchronous mode option

### Phase 4: Breaking Changes (Major Version)
1. [ ] Remove deprecated methods
2. [ ] Rename `CreateContext` → `Create`
3. [ ] Simplify End() pattern

---

## Compatibility Considerations

### Backward Compatible Changes
- Adding new methods/functions
- Adding new optional fields to Config
- Adding interfaces that existing types implement
- Adding type aliases

### Breaking Changes (Require Major Version)
- Removing deprecated methods
- Changing method signatures
- Changing struct field types
- Renaming exported identifiers

### Recommendation
Since the package has no users yet ("big bang" reorganization), consider implementing Phase 4 changes now to avoid future breaking changes.

---

## Success Metrics

1. **Reduced API Surface**: From 30+ options to 3 tiers
2. **Improved Testability**: Core interfaces defined
3. **Better Error Handling**: Error channel + helper functions
4. **Cleaner Code**: config.go < 300 lines
5. **Documentation**: 100% godoc coverage with examples

---

## References

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Standard Library Patterns](https://pkg.go.dev/std)
- [Google Cloud Go SDK](https://github.com/googleapis/google-cloud-go) - Good example of tiered config
- [AWS SDK for Go v2](https://github.com/aws/aws-sdk-go-v2) - Good example of options pattern
