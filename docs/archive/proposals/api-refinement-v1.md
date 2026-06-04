# Langfuse Go API Refinement Proposal for v1.0

## Executive Summary

This proposal outlines a comprehensive API refinement for the Langfuse Go SDK ahead of the v1.0 release. The current API, while feature-complete, suffers from inconsistency and complexity that harms developer experience. We propose a unified, simplified API that maintains all current functionality while dramatically improving usability and understandability.

## Current State Analysis

### identified Issues

#### 1. Inconsistent Creation Patterns
```go
// Current mixed approaches create confusion
client.NewTrace().Name("test").Create(ctx)           // Builder + Create()
trace.Update().Output(data).Apply(ctx)               // Separate Apply()
client.Prompts().Get(ctx, "name")                    // Direct method
trace.Score().Name("quality").NumericValue(0.9).Create(ctx)
```

#### 2. Over-Complex Configuration
```go
type Config struct {
    // 25+ interdependent fields
    PublicKey string
    SecretKey string
    HTTPClient *http.Client
    MaxRetries int
    RetryDelay time.Duration
    BatchSize int
    FlushInterval time.Duration
    Debug bool
    ErrorHandler func(error)
    Logger Logger
    StructuredLogger StructuredLogger
    Metrics Metrics
    // ... 15 more fields
}
```

#### 3. Mixed Return Patterns
```go
trace, err := client.NewTrace().Name("test").Create(ctx)  // (result, error)
err := trace.Update().Output(data).Apply(ctx)            // (error only)
err := generation.End(ctx)                                // (error only)
```

#### 4. Context Handling Variations
```go
trace.Create(ctx)              // Context required here
generation.End(ctx)            // Context required here
trace.Update().Apply(ctx)      // Context required at end
client.Prompts().Get(ctx, "name")  // Context as first param
```

## Proposed API Design

### Core Principles

1. **One Primary Pattern**: Builder pattern with single finalization
2. **Consistent Return Types**: Always `(result, error)` for consistency
3. **Context-First**: Context is always the first parameter
4. **Fluent Chaining**: All methods return the builder for chaining
5. **Zero Learning**: Intuitive API that reads like prose

### New API Structure

#### 1. Simplified Client Creation
```go
// Original: Complex with 25+ options
client, err := langfuse.NewWithConfig(&langfuse.Config{
    PublicKey: "pk-xxx",
    SecretKey: "sk-xxx",
    Region: langfuse.RegionUS,
    BatchSize: 100,
    FlushInterval: 5 * time.Second,
    // ... 20 more fields
})

// Proposed: Simplified with sane defaults
client := langfuse.NewClient("pk-xxx", "sk-xxx",
    langfuse.WithRegion(langfuse.RegionUS),
    langfuse.WithEnvironment("production"))
```

#### 2. Unified Entity Creation
```go
// Proposed: Single pattern for all entities
trace, err := client.NewTrace(ctx, "user-request",
    langfuse.WithUserID("user-123"),
    langfuse.WithTags([]string{"api", "v1"}),
    langfuse.WithMetadata(map[string]interface{}{
        "endpoint": "/api/chat",
    }))

span, err := trace.NewSpan(ctx, "preprocessing",
    langfuse.WithInput("raw message"),
    langfuse.WithMetadata(map[string]interface{}{
        "steps": 3,
    }))

generation, err := span.NewGeneration(ctx, "gpt-4",
    langfuse.WithModel("gpt-4"),
    langfuse.WithInput([]map[string]string{
        {"role": "user", "content": "Hello"},
    }),
    langfuse.WithModelParameters(map[string]interface{}{
        "temperature": 0.7,
    }))
```

#### 3. Consistent Operation Methods
```go
// All operations follow: (result, error) pattern
err := span.End(ctx, langfuse.WithOutput("processed text"))
err := generation.End(ctx,
    langfuse.WithOutput("response text"),
    langfuse.WithTokenUsage(10, 8))

scores, err := generation.Scores(ctx,
    langfuse.NewScore("quality", 0.95,
        langfuse.WithComment("Excellent response")),
    langfuse.NewScore("speed", 0.8))

// Updates follow same pattern
updated, err := trace.Update(ctx,
    langfuse.WithOutput(map[string]interface{}{
        "response": "final result",
    }))
```

#### 4. Simplified Scoping and Context
```go
// Natural nesting with context propagation
func handleRequest(ctx context.Context, client *langfuse.Client) error {
    trace, err := client.NewTrace(ctx, "request",
        langfuse.WithUserID("user-123"))
    if err != nil {
        return err
    }

    // Pass trace context to nested functions
    return processData(ctx, trace)
}

func processData(ctx context.Context, tracer langfuse.Tracer) error {
    // Create nested spans automatically
    span, err := tracer.NewSpan(ctx, "processing")
    if err != nil {
        return err
    }

    // Additional work...
    return span.End(ctx)
}
```

## Proposed Interface Changes

### Core Simplified Interfaces
```go
// Main client interface
type Client interface {
    NewTrace(ctx context.Context, name string, opts ...TraceOption) (*Trace, error)

    // Legacy methods available via sub-clients
    Traces() *TracesClient
    Prompts() *PromptsClient
    // ... but recommended to use top-level methods

    // Simplified configuration
    Shutdown(ctx context.Context) error
    Stats() ClientStats
}

// Unified interface for all observation creators
type Observer interface {
    NewSpan(ctx context.Context, name string, opts ...SpanOption) (*Span, error)
    NewGeneration(ctx context.Context, name string, opts ...GenerationOption) (*Generation, error)
    NewEvent(ctx context.Context, name string, opts ...EventOption) (*Event, error)
}

// Simplified scoring
type Scorer interface {
    AddScores(ctx context.Context, scores ...*Score) error
    Score(ctx context.Context, name string, value float64, opts ...ScoreOption) error
}
```

### Type-Safe Option Pattern
```go
// Consistent option functions for all entities
type TraceOption func(*traceConfig)
type SpanOption func(*spanConfig)
type GenerationOption func(*generationConfig)

// Example option constructors
func WithUserID(id string) TraceOption { ... }
func WithTags(tags []string) TraceOption { ... }
func WithMetadata(meta map[string]interface{}) TraceOption { ... }

func WithInput(input interface{}) SpanOption { ... }
func WithOutput(output interface{}) SpanOption { ... }

func WithModel(model string) GenerationOption { ... }
func WithTokenUsage(input, output int) GenerationOption { ... }
def WithModelParameters(params map[string]interface{}) GenerationOption { ... }
```

## Migration Strategy

### Phase 1: Backward Compatibility (v1.0-alpha)
```go
// Package provides both APIs
package langfuse

// New simplified API (recommended)
client := langfuse.NewClient("pk-xxx", "sk-xxx", opts...)

// Legacy API (deprecated but supported)
client, err := langfuse.NewWithConfig(&langfuse.Config{...})
```

### Phase 2: Gradual Migration (v1.0-beta)
- Deprecation warnings for legacy API
- Migration guide and tools
- Automated codemod scripts

### Phase 3: Clean API (v1.0)
- Legacy methods moved to `legacy/` package
- Clean, focused v1 API surface
- All tests pass with new API only

## Specific API Improvements

### 1. Context Handling
```go
// Current: Inconsistent
client.NewTrace().Name("test").Create(ctx)  // Context at end
trace.Update().Output(data).Apply(ctx)      // Context at end

// Proposed: Context-first always
client.NewTrace(ctx, "test", opts...)
trace.Update(ctx, opts...)
```

### 2. Error Handling
```go
// Current: Mixed patterns
trace, err := client.NewTrace().Name("test").Create(ctx)     // (T, error)
err := generation.End(ctx)                                    // (error only)

// Proposed: Consistent
trace, err := client.NewTrace(ctx, "test", opts...)          // (T, error)
updated, err := generation.End(ctx, opts...)                  // (T, error)
```

### 3. Option Pattern Standardization
```go
// Current: Different approaches
trace.Name("test").UserID("user").Create(ctx)                // Builder methods
trace.Update().Output(data).Apply(ctx)                       // Separate Apply()

// Proposed: Unified
trace, err := client.NewTrace(ctx, "test",
    langfuse.WithUserID("user"),
    langfuse.WithTags([]string{"api"}))

updated, err := trace.Update(ctx,
    langfuse.WithOutput(data))
```

## Implementation Plan

### Week 1-2: Core Interface Design
- Define simplified interfaces
- Implement option types
- Create migration compatibility layer

### Week 3-4: Client Refactoring
- Simplify client creation
- Implement new builder pattern
- Maintain backward compatibility

### Week 5-6: Entity Unification
- Unify trace/span/generation APIs
- Implement scoring consistency
- Update all method signatures

### Week 7-8: Testing and Documentation
- Comprehensive test coverage
- Migration documentation
- Performance benchmarking

### Week 9-10: Code Generation Tools
- Automated migration scripts
- Linter rules for deprecated usage
- API validation tools

## Benefits

### 1. Developer Experience
- **Intuitive**: API reads like prose
- **Consistent**: One pattern to learn
- **Typed**: Compile-time safety
- **Discoverable**: IDE-friendly autocompletion

### 2. Maintainability
- **Simpler**: Less cognitive overhead
- **Testable**: Clear interfaces
- **Documented**: Self-documenting code
- **Refactorable**: Clean abstractions

### 3. Performance
- **Reduced Allocations**:Fewer temporary objects
- **Better Caching**: Consistent patterns
- **Optimized Paths**: Clear hot paths
- **Memory Friendly**: Cleaner lifecycles

## Risks and Mitigations

### Risk: Breaking Changes
- **Mitigation**: Gradual migration path with clear deprecation timeline
- **Fallback**: Compatibility package for legacy users
- **Tooling**: Automated migration assistance

### Risk: Performance Regression
- **Mitigation**: Comprehensive benchmarking
- **Strategy**: Performance parity commitment
- **Testing**: Regression test suite

### Risk: Ecosystem Impact
- **Mitigation**: Early community feedback
- **Strategy**: RFC process for changes
- **Support**: Extended support for v0 releases

## Success Metrics

1. **Adoption**: 90% of new projects use simplified API within 3 months
2. **Satisfaction**: Developer satisfaction survey improvement >30%
3. **Performance**: No performance regression (<5% variance)
4. **Support**: Reduced support questions about API usage
5. **Documentation**: Simplified documentation reduces support load

## Conclusion

The proposed API refinement transforms the Langfuse Go SDK from a powerful but complex tool into an elegant, developer-friendly library. By unifying patterns and simplifying interfaces, we can significantly improve the developer experience while maintaining all current functionality.

The phased migration approach ensures existing users aren't left behind, while the new API positions the SDK for long-term success and community growth.

**Recommendation: Approve and begin implementation immediately for v1.0 release.**

---

## Appendix: API Comparison Matrix

| Operation | Current API | Proposed API | Improvement |
|-----------|-------------|--------------|-------------|
| Create Trace | `client.NewTrace().Name("test").Create(ctx)` | `client.NewTrace(ctx, "test", opts...)` | Consistent, context-first |
| Add Score | `generation.Score().Name("q").NumericValue(0.9).Create(ctx)` | `generation.AddScore(ctx, "quality", 0.9)` | Simpler, clearer |
| Update Trace | `trace.Update().Output(data).Apply(ctx)` | `trace.Update(ctx, WithOutput(data))` | Unified pattern |
| End Span | `span.End(ctx)` | `span.End(ctx, WithOutput(data))` | Consistent returns |

## Detailed Migration Guide

[To be created with API approval]

## Reference Implementation

[Proof-of-concept available in branch `api-refinement-v1`]