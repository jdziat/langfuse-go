# V1 API Roadmap: Dual-Tier Design

## Overview

The v1.0 API provides **two complementary tiers**:

| Tier | Target | Pattern | Use Case |
|------|--------|---------|----------|
| **Simple API** | 90% of users | One-liner functions | Quick tracing, common patterns |
| **Advanced API** | Power users | Builder pattern | Full control, complex workflows |

Both tiers are consistent, type-safe, and production-ready. The Simple API is syntactic sugar over the Advanced API.

---

## Tier 1: Simple API

### Design Principles

1. **One function call per operation**
2. **Context-first parameters**
3. **Required params as arguments, optional as variadic options**
4. **Returns `(result, error)` consistently**

### Client Creation

```go
// Simple: Minimal configuration
client, err := langfuse.New("pk-xxx", "sk-xxx")

// Simple: With region
client, err := langfuse.New("pk-xxx", "sk-xxx",
    langfuse.WithRegion(langfuse.RegionUS))

// Simple: From environment
client, err := langfuse.NewFromEnv()
```

### Trace Operations

```go
// Create trace with name (most common)
trace, err := client.Trace(ctx, "user-request")

// Create trace with options
trace, err := client.Trace(ctx, "user-request",
    langfuse.WithUserID("user-123"),
    langfuse.WithSessionID("session-456"),
    langfuse.WithTags("api", "v2"),
    langfuse.WithMetadata(langfuse.M{"endpoint": "/chat"}))

// Update trace
err = trace.SetOutput(ctx, response)

// End trace (optional - traces auto-complete)
err = trace.Complete(ctx)
```

### Span Operations

```go
// Create span
span, err := trace.Span(ctx, "preprocessing")

// Create span with options
span, err := trace.Span(ctx, "preprocessing",
    langfuse.WithInput(rawData),
    langfuse.WithMetadata(langfuse.M{"step": 1}))

// End span with output
err = span.End(ctx, processedData)

// End span without output
err = span.End(ctx, nil)
```

### Generation Operations

```go
// Create and end generation in one call (most common LLM pattern)
gen, err := trace.Generation(ctx, "gpt-4-call",
    langfuse.WithModel("gpt-4"),
    langfuse.WithInput(messages),
    langfuse.WithOutput(response),
    langfuse.WithUsage(100, 50))

// Create generation, end later
gen, err := trace.Generation(ctx, "gpt-4-call",
    langfuse.WithModel("gpt-4"),
    langfuse.WithInput(messages))

// ... make LLM call ...

err = gen.End(ctx, response,
    langfuse.WithUsage(inputTokens, outputTokens))
```

### Scoring

```go
// Score a generation
err = gen.Score(ctx, "quality", 0.95)
err = gen.Score(ctx, "quality", 0.95, langfuse.WithComment("excellent"))

// Boolean score
err = gen.ScoreBool(ctx, "passed", true)

// Categorical score
err = gen.ScoreCategory(ctx, "rating", "excellent")
```

### Event Operations

```go
// Log an event
err = trace.Event(ctx, "cache-hit",
    langfuse.WithMetadata(langfuse.M{"key": cacheKey}))
```

### Complete Example (Simple API)

```go
func HandleChat(ctx context.Context, client *langfuse.Client, userID, message string) (string, error) {
    // Create trace
    trace, err := client.Trace(ctx, "chat-request",
        langfuse.WithUserID(userID),
        langfuse.WithInput(message))
    if err != nil {
        return "", err
    }
    defer trace.Complete(ctx)

    // Preprocess
    span, _ := trace.Span(ctx, "preprocess")
    processed := preprocess(message)
    span.End(ctx, processed)

    // LLM call
    response, tokens, _ := callLLM(processed)

    gen, _ := trace.Generation(ctx, "llm-call",
        langfuse.WithModel("gpt-4"),
        langfuse.WithInput(processed),
        langfuse.WithOutput(response),
        langfuse.WithUsage(tokens.In, tokens.Out))

    // Score
    gen.Score(ctx, "latency", tokens.LatencyMs/1000.0)

    // Set final output
    trace.SetOutput(ctx, response)

    return response, nil
}
```

---

## Tier 2: Advanced API (Builder Pattern)

### Design Principles

1. **Fluent builder pattern with method chaining**
2. **Create() or Apply() to finalize**
3. **Full access to all fields**
4. **Clone() for templates**
5. **Strict validation mode available**

### Client Creation

```go
// Advanced: Full configuration
client, err := langfuse.NewWithConfig(&langfuse.Config{
    PublicKey:     "pk-xxx",
    SecretKey:     "sk-xxx",
    Region:        langfuse.RegionUS,
    BatchSize:     200,
    FlushInterval: 10 * time.Second,
    CircuitBreaker: &langfuse.CircuitBreakerConfig{
        FailureThreshold: 10,
        Timeout:          time.Minute,
    },
    OnBatchFlushed: func(result langfuse.BatchResult) {
        log.Printf("Batch: %d events, success=%t", result.EventCount, result.Success)
    },
})
```

### Trace Operations

```go
// Create trace with full control
trace, err := client.NewTrace().
    ID("custom-trace-id").
    Name("user-request").
    UserID("user-123").
    SessionID("session-456").
    Input(map[string]any{"message": "hello"}).
    Metadata(langfuse.Metadata{"version": "2.0"}).
    Tags([]string{"api", "chat"}).
    Release("v1.2.3").
    Environment("production").
    Create(ctx)

// Update with builder
err = trace.Update().
    Output(response).
    Metadata(langfuse.Metadata{"tokens": 150}).
    Apply(ctx)

// Clone for templates
template := client.NewTrace().
    Environment("production").
    Release(version)

trace1, _ := template.Clone().Name("request-1").Create(ctx)
trace2, _ := template.Clone().Name("request-2").Create(ctx)
```

### Span Operations

```go
// Create span with full control
span, err := trace.Span().
    ID("custom-span-id").
    Name("data-processing").
    Input(rawData).
    Metadata(langfuse.Metadata{"step": "preprocessing"}).
    Level(langfuse.ObservationLevelDebug).
    Create(ctx)

// End with full control
err = span.Update().
    Output(result).
    EndTime(time.Now()).
    StatusMessage("completed successfully").
    Apply(ctx)

// Or use EndWith for flexibility
result := span.EndWith(ctx,
    langfuse.WithOutput(data),
    langfuse.WithLevel(langfuse.ObservationLevelWarning),
    langfuse.WithStatusMessage("completed with warnings"))
if !result.Ok() {
    log.Printf("Failed: %v", result.Error)
}
```

### Generation Operations

```go
// Create generation with full control
gen, err := trace.Generation().
    ID("custom-gen-id").
    Name("gpt-4-completion").
    Model("gpt-4-turbo").
    ModelParameters(langfuse.Metadata{
        "temperature": 0.7,
        "max_tokens":  1000,
        "top_p":       0.9,
    }).
    Input([]map[string]string{
        {"role": "system", "content": systemPrompt},
        {"role": "user", "content": userMessage},
    }).
    PromptName("chat-v2").
    PromptVersion(3).
    Create(ctx)

// End with usage details
err = gen.Update().
    Output(response).
    Usage(&langfuse.Usage{
        Input:      promptTokens,
        Output:     completionTokens,
        Total:      promptTokens + completionTokens,
        InputCost:  0.01,
        OutputCost: 0.03,
        TotalCost:  0.04,
    }).
    CompletionStartTime(firstTokenTime).
    EndTime(time.Now()).
    Apply(ctx)
```

### Scoring

```go
// Score with full control
err = gen.Score().
    ID("custom-score-id").
    Name("quality").
    NumericValue(0.95).
    Comment("Excellent response with accurate information").
    Source("automated").
    ConfigID("quality-rubric-v2").
    Create(ctx)

// Multiple score types
err = trace.Score().Name("passed").BooleanValue(true).Create(ctx)
err = trace.Score().Name("rating").CategoricalValue("excellent").Create(ctx)
```

### Strict Validation Mode

```go
// Enable strict validation
client, _ := langfuse.NewWithConfig(&langfuse.Config{
    PublicKey: pk,
    SecretKey: sk,
    StrictValidation: &langfuse.StrictValidationConfig{
        Enabled:  true,
        FailFast: false,
    },
})

// Use validated builders
result := client.NewTraceStrict().
    Name("").  // Error: empty name
    UserID("user-123").
    Create(ctx)

trace, err := result.Unwrap()
if err != nil {
    // err contains all validation errors
    log.Printf("Validation failed: %v", err)
}
```

### Complete Example (Advanced API)

```go
func HandleComplexWorkflow(ctx context.Context, client *langfuse.Client, req *Request) (*Response, error) {
    // Create trace with template
    trace, err := client.NewTrace().
        Name("complex-workflow").
        UserID(req.UserID).
        SessionID(req.SessionID).
        Input(req).
        Metadata(langfuse.Metadata{
            "version":  "2.0",
            "workflow": "multi-step",
        }).
        Tags([]string{"production", "workflow"}).
        Create(ctx)
    if err != nil {
        return nil, err
    }

    // Store in context for nested calls
    ctx = langfuse.ContextWithTrace(ctx, trace)

    // Step 1: Preprocessing span
    preprocessSpan, _ := trace.Span().
        Name("preprocess").
        Input(req.Data).
        Create(ctx)

    processed, err := preprocess(req.Data)
    if err != nil {
        preprocessSpan.Update().
            Level(langfuse.ObservationLevelError).
            StatusMessage(err.Error()).
            Apply(ctx)
        return nil, err
    }
    preprocessSpan.EndWithOutput(ctx, processed)

    // Step 2: LLM Generation
    startTime := time.Now()
    gen, _ := trace.Generation().
        Name("main-llm-call").
        Model("gpt-4-turbo").
        ModelParameters(langfuse.Metadata{"temperature": 0.7}).
        Input(processed).
        Create(ctx)

    response, usage, err := callLLM(ctx, processed)

    gen.Update().
        Output(response).
        Usage(&langfuse.Usage{
            Input:  usage.PromptTokens,
            Output: usage.CompletionTokens,
        }).
        CompletionStartTime(usage.FirstTokenAt).
        EndTime(time.Now()).
        Apply(ctx)

    // Score the generation
    gen.Score().
        Name("latency").
        NumericValue(time.Since(startTime).Seconds()).
        Create(ctx)

    gen.Score().
        Name("quality").
        NumericValue(evaluateQuality(response)).
        Comment("automated evaluation").
        Create(ctx)

    // Final trace output
    trace.Update().
        Output(response).
        Metadata(langfuse.Metadata{
            "total_tokens": usage.TotalTokens,
            "duration_ms":  time.Since(startTime).Milliseconds(),
        }).
        Apply(ctx)

    return response, nil
}
```

---

## API Consistency Rules

### Both Tiers Follow

| Rule | Simple API | Advanced API |
|------|------------|--------------|
| Context first | `Trace(ctx, name)` | `Create(ctx)` at end |
| Required params explicit | `Trace(ctx, name)` | `.Name(name)` required |
| Options are variadic | `...Option` | `.Method()` chaining |
| Returns `(T, error)` | Yes | Yes |
| Supports all features | Via options | Via methods |

### Type Aliases for Convenience

```go
// Simple API uses short names
type M = Metadata  // langfuse.M{"key": "value"}

// Helper constructors
func Tags(tags ...string) []string { return tags }
```

### Shared Options

Options work in both APIs:

```go
// These options work everywhere
langfuse.WithInput(data)
langfuse.WithOutput(data)
langfuse.WithMetadata(langfuse.M{"key": "value"})
langfuse.WithLevel(langfuse.ObservationLevelWarning)

// Usage in Simple API
span, _ := trace.Span(ctx, "name", langfuse.WithInput(data))

// Same option concept in Advanced API
span, _ := trace.Span().Name("name").Input(data).Create(ctx)
```

---

## Implementation Plan

### Phase 1: Core Refactoring (Week 1-2)

**Goal**: Consolidate existing code, prepare for Simple API layer

- [ ] Consolidate 76 files to ~35 files
- [ ] Standardize all option types
- [ ] Ensure Advanced API is fully consistent
- [ ] Add missing `EndWith()` methods to all observation types
- [ ] Ensure all contexts implement `Observer` interface

**File Consolidation Plan**:

| Current | Target | Action |
|---------|--------|--------|
| trace.go, span.go, generation.go, event.go | observations.go | Merge |
| score.go, scores.go | scores.go | Merge |
| builders.go, validated_builder.go | builders.go | Keep separate |
| helpers.go, types_generic.go | helpers.go | Merge |
| async_errors.go, errors.go | errors.go | Merge |
| lifecycle.go, client.go | client.go | Merge |

### Phase 2: Simple API Layer (Week 3-4)

**Goal**: Implement Simple API as thin layer over Advanced API

- [ ] Implement `client.Trace(ctx, name, ...opts)`
- [ ] Implement `trace.Span(ctx, name, ...opts)`
- [ ] Implement `trace.Generation(ctx, name, ...opts)`
- [ ] Implement `trace.Event(ctx, name, ...opts)`
- [ ] Implement `observation.Score(ctx, name, value, ...opts)`
- [ ] Implement `observation.End(ctx, output, ...opts)`
- [ ] Add convenience methods: `SetOutput()`, `Complete()`

**Implementation Pattern**:

```go
// Simple API wraps Advanced API
func (c *Client) Trace(ctx context.Context, name string, opts ...TraceOption) (*Trace, error) {
    builder := c.NewTrace().Name(name)
    for _, opt := range opts {
        opt.applyTrace(builder)
    }
    return builder.Create(ctx)
}
```

### Phase 3: Unified Options (Week 5)

**Goal**: Create unified option system that works across both APIs

- [ ] Define option interfaces for each entity type
- [ ] Implement common options: `WithInput`, `WithOutput`, `WithMetadata`, `WithLevel`
- [ ] Implement entity-specific options
- [ ] Add `M` type alias for `Metadata`
- [ ] Add `Tags()` helper function

### Phase 4: Context Integration (Week 6)

**Goal**: Improve context propagation

- [ ] Ensure `ContextWithTrace` works seamlessly
- [ ] Add `ContextWithSpan`, `ContextWithGeneration`
- [ ] Implement automatic parent detection from context
- [ ] Add middleware helpers for HTTP frameworks

### Phase 5: Documentation & Examples (Week 7-8)

**Goal**: Complete documentation for both tiers

- [ ] Update README with dual-tier examples
- [ ] Create `examples/simple/` directory
- [ ] Create `examples/advanced/` directory
- [ ] Update godoc for all public types
- [ ] Write migration guide (not needed - no users)

---

## Interface Definitions

### Core Interfaces

```go
// Client is the main entry point
type Client interface {
    // Simple API
    Trace(ctx context.Context, name string, opts ...TraceOption) (*Trace, error)

    // Advanced API
    NewTrace() *TraceBuilder

    // Lifecycle
    Flush(ctx context.Context) error
    Shutdown(ctx context.Context) error
    Health(ctx context.Context) (*HealthStatus, error)

    // Sub-clients (advanced)
    Prompts() *PromptsClient
    Datasets() *DatasetsClient
    // ...
}

// Trace represents an active trace
type Trace interface {
    Observer

    // Simple API
    Span(ctx context.Context, name string, opts ...SpanOption) (*Span, error)
    Generation(ctx context.Context, name string, opts ...GenerationOption) (*Generation, error)
    Event(ctx context.Context, name string, opts ...EventOption) error
    SetOutput(ctx context.Context, output any) error
    Complete(ctx context.Context) error

    // Advanced API
    // Span() *SpanBuilder (from Observer)
    // Generation() *GenerationBuilder (from Observer)
    Update() *TraceUpdateBuilder
}

// Span represents an active span
type Span interface {
    Observer
    Scorer

    // Simple API
    End(ctx context.Context, output any, opts ...EndOption) error

    // Advanced API
    Update() *SpanUpdateBuilder
    EndWith(ctx context.Context, opts ...EndOption) EndResult
}

// Generation represents an LLM generation
type Generation interface {
    Observer
    Scorer

    // Simple API
    End(ctx context.Context, output any, opts ...EndOption) error

    // Advanced API
    Update() *GenerationUpdateBuilder
    EndWith(ctx context.Context, opts ...EndOption) EndResult
}

// Scorer provides scoring methods
type Scorer interface {
    // Simple API
    Score(ctx context.Context, name string, value float64, opts ...ScoreOption) error
    ScoreBool(ctx context.Context, name string, value bool, opts ...ScoreOption) error
    ScoreCategory(ctx context.Context, name string, value string, opts ...ScoreOption) error

    // Advanced API
    // Score() *ScoreBuilder (from Observer interface - rename to NewScore())
}

// Observer creates child observations (Advanced API)
type Observer interface {
    ID() string
    TraceID() string
    Span() *SpanBuilder
    Generation() *GenerationBuilder
    Event() *EventBuilder
    Score() *ScoreBuilder
}
```

---

## Breaking Changes

Since there are no external users, we can make these breaking changes:

| Change | Reason |
|--------|--------|
| Rename `NewTrace()` builder method names for clarity | Consistency |
| Change `Score()` on Observer to `NewScore()` | Avoid conflict with Simple API |
| Consolidate files | Reduce cognitive load |
| Remove deprecated methods | Clean API |
| Standardize all return types to `(T, error)` | Consistency |

---

## Success Metrics

| Metric | Target |
|--------|--------|
| Root package files | 30-35 (from 76) |
| Lines of code reduction | 20% |
| Simple API coverage | All common operations |
| Test coverage | Maintain >80% |
| Example coverage | Both tiers demonstrated |

---

## Summary

The v1.0 API provides:

1. **Simple API**: One-liners for 90% of use cases
   - `client.Trace(ctx, "name")`
   - `trace.Span(ctx, "name")`
   - `gen.Score(ctx, "quality", 0.95)`

2. **Advanced API**: Full control via builders
   - `client.NewTrace().Name("name").UserID("user").Create(ctx)`
   - `trace.Span().Name("name").Input(data).Create(ctx)`
   - `gen.Score().Name("quality").NumericValue(0.95).Comment("...").Create(ctx)`

Both APIs are:
- **Consistent**: Same patterns throughout
- **Type-safe**: Compile-time checks
- **Interoperable**: Mix and match as needed
- **Production-ready**: Same underlying implementation
