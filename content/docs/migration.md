---
title: Migration Guide
weight: 6
---

This guide helps you adopt the canonical Langfuse Go SDK API. Earlier drafts and
some third-party examples used a params-struct style (for example
`client.Trace(TraceParams{...})`) that the SDK does not provide. The supported
API is a context-threaded fluent builder, and this guide shows how to move to
it.

## Overview

The canonical API has these properties:

- A single import at the **module root** (no `/langfuse` subdirectory).
- Positional `publicKey, secretKey` arguments to `New`, with behavior tuned by
  `ConfigOption` functions.
- Fluent builders that chain setters and finish with `Create(ctx)`,
  `Apply(ctx)`, or one of the `End*` methods.
- Every network-touching call takes a `context.Context`.

## Import Path

The package lives at the module root. Import it directly:

```text
import langfuse "github.com/jdziat/langfuse-go"
```

There is no `github.com/jdziat/langfuse-go/langfuse` package. The optional
evaluation helpers live in a subpackage:

```text
import "github.com/jdziat/langfuse-go/evaluation"
```

## Client Construction

The keys are positional arguments, not options. There are no `WithPublicKey` or
`WithSecretKey` functions.

**Incorrect:**
```text
client, err := langfuse.New(
    langfuse.WithPublicKey("pk-lf-..."),
    langfuse.WithSecretKey("sk-lf-..."),
)
```

**Correct:**
```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...",
    langfuse.WithRegion(langfuse.RegionUS),
)
```

## Creating Traces

There are no `TraceParams`, `GenerationParams`, `SpanParams`, `EventParams`, or
`ScoreParams` types. Use the fluent builder and finish with `Create(ctx)`.

**Incorrect:**
```text
trace := client.Trace(langfuse.TraceParams{
    Name:   "chat-completion",
    UserID: "user-123",
})
```

**Correct:**
```go
ctx := context.Background()

trace, err := client.NewTrace().
    Name("chat-completion").
    UserID("user-123").
    Create(ctx)
if err != nil {
    log.Fatal(err)
}
```

For one-line creation, `Client.Trace` takes a context, a name, and
`TraceOption` functions:

```go
ctx := context.Background()

trace, err := client.Trace(ctx, "chat-completion",
    langfuse.WithUserID("user-123"),
)
if err != nil {
    log.Fatal(err)
}
_ = trace
```

## Creating Generations

Generations are created from a trace (or observation) with `NewGeneration()` and
finished with `EndWithUsage` (or `End` / `EndWithOutput`).

**Incorrect:**
```text
generation := trace.Generation(langfuse.GenerationParams{
    Name:  "openai-chat",
    Model: "gpt-4",
})
generation.Update(langfuse.GenerationParams{
    Output: map[string]any{"response": "Hello!"},
    Usage:  &langfuse.Usage{PromptTokens: 10, CompletionTokens: 5},
})
```

**Correct:**
```go
ctx := context.Background()

generation, err := trace.NewGeneration().
    Name("openai-chat").
    Model("gpt-4").
    Create(ctx)
if err != nil {
    log.Fatal(err)
}

// EndWithUsage records the output plus input and output token counts.
if err := generation.EndWithUsage(ctx, "Hello!", 10, 5); err != nil {
    log.Printf("end failed: %v", err)
}
```

Note that the `Usage` type uses `Input`, `Output`, and `Total` fields (not
`PromptTokens`/`CompletionTokens`):

```go
usage := &langfuse.Usage{Input: 10, Output: 5, Total: 15}
_ = usage
```

## Creating Spans and Events

Spans use `NewSpan()` + `Create(ctx)` and finish with `End(ctx)` or
`EndWithOutput(ctx, output)`. Events use `NewEvent()` + `Create(ctx)`.

**Incorrect:**
```text
span := trace.Span(langfuse.SpanParams{Name: "retrieve-documents"})
event := trace.Event(langfuse.EventParams{Name: "cache-hit"})
```

**Correct:**
```go
ctx := context.Background()

span, err := trace.NewSpan().Name("retrieve-documents").Create(ctx)
if err != nil {
    log.Fatal(err)
}
if err := span.End(ctx); err != nil {
    log.Printf("end failed: %v", err)
}

if err := trace.NewEvent().Name("cache-hit").Create(ctx); err != nil {
    log.Printf("event failed: %v", err)
}
```

Nested observations are created from their parent context:

```go
ctx := context.Background()

parent, err := trace.NewSpan().Name("parent").Create(ctx)
if err != nil {
    log.Fatal(err)
}

child, err := parent.NewSpan().Name("child").Create(ctx)
if err != nil {
    log.Fatal(err)
}

_ = child.End(ctx)
_ = parent.End(ctx)
```

## Adding Scores

There is no `client.Evaluator()` and no `ScoreParams`. Score any observation with
`NewScore()`:

**Incorrect:**
```text
trace.Score(langfuse.ScoreParams{Name: "user-rating", Value: 5.0})
```

**Correct:**
```go
ctx := context.Background()

if err := trace.NewScore().
    Name("user-rating").
    NumericValue(5.0).
    Create(ctx); err != nil {
    log.Printf("score failed: %v", err)
}
```

## Updating Observations

Updates use an update builder finished with `Apply(ctx)`.

**Incorrect:**
```text
trace.Update(langfuse.TraceParams{Output: map[string]any{"result": "ok"}})
```

**Correct:**
```go
ctx := context.Background()

if err := trace.Update().
    Output(map[string]any{"result": "ok"}).
    Apply(ctx); err != nil {
    log.Printf("update failed: %v", err)
}
```

## Flush and Shutdown

Both `Flush` and `Shutdown` take a context and return an error:

```go
ctx := context.Background()

if err := client.Flush(ctx); err != nil {
    log.Printf("flush failed: %v", err)
}

if err := client.Shutdown(ctx); err != nil {
    log.Printf("shutdown failed: %v", err)
}
```

## Complete Before/After Example

### Before (params-struct style — not supported)

```text
package main

import (
    "log"

    "github.com/jdziat/langfuse-go/langfuse"
)

func main() {
    client, err := langfuse.New(
        langfuse.WithPublicKey("pk-lf-..."),
        langfuse.WithSecretKey("sk-lf-..."),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown()

    trace := client.Trace(langfuse.TraceParams{Name: "chat-completion"})

    generation := trace.Generation(langfuse.GenerationParams{
        Name:  "openai-chat",
        Model: "gpt-4",
    })

    generation.Update(langfuse.GenerationParams{
        Output: map[string]any{"response": "Hello!"},
    })

    client.Flush()
}
```

### After (canonical API)

```go
package main

import (
    "context"
    "log"

    langfuse "github.com/jdziat/langfuse-go"
)

func main() {
    ctx := context.Background()

    client, err := langfuse.New("pk-lf-...", "sk-lf-...")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(ctx)

    trace, err := client.NewTrace().Name("chat-completion").Create(ctx)
    if err != nil {
        log.Fatal(err)
    }

    generation, err := trace.NewGeneration().
        Name("openai-chat").
        Model("gpt-4").
        Create(ctx)
    if err != nil {
        log.Fatal(err)
    }

    if err := generation.EndWithUsage(ctx, "Hello!", 10, 5); err != nil {
        log.Printf("end failed: %v", err)
    }

    if err := client.Flush(ctx); err != nil {
        log.Printf("Flush failed: %v", err)
    }
}
```

## Migration Checklist

- [ ] Import from `github.com/jdziat/langfuse-go` (module root)
- [ ] Pass `publicKey, secretKey` positionally to `New`
- [ ] Replace `*Params` structs with fluent builders (`NewTrace`,
      `NewGeneration`, `NewSpan`, `NewEvent`, `NewScore`)
- [ ] Finish builders with `Create(ctx)` / `Apply(ctx)` / `End*`
- [ ] Use `Usage{Input, Output, Total}` field names
- [ ] Add `context.Context` to `Flush` and `Shutdown`
- [ ] Check returned errors on every call

## Getting Help

If you encounter issues during migration:

1. **Check the documentation**: [Getting Started Guide](../getting-started/)
2. **Review the reference**: [API Reference](../api-reference/)
3. **Open an issue**: [GitHub Issues](https://github.com/jdziat/langfuse-go/issues)

## Next Steps

- [Getting Started](../getting-started/) - Learn the canonical API
- [Configuration](../configuration/) - Optimize your setup
- [Tracing Guide](../tracing/) - Master tracing patterns
- [API Reference](../api-reference/) - Complete type reference
