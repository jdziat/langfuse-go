---
title: Tracing
weight: 3
---

Langfuse provides comprehensive tracing capabilities for LLM applications. This guide covers all tracing concepts and how to use them effectively.

## Overview

Tracing in Langfuse follows a hierarchical structure:

```
Trace (root observation)
├── Span (logical step)
│   ├── Generation (LLM call)
│   └── Event (discrete event)
├── Generation (LLM call)
└── Span (another step)
    └── Event (discrete event)
```

Every observation is created with a fluent builder. Chain setters, then call
`Create(ctx)` to send the event. The setup for the snippets below is:

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

    trace, err := client.NewTrace().Name("example").Create(ctx)
    if err != nil {
        log.Fatal(err)
    }
    _ = trace
}
```

## Traces

A **Trace** represents a complete execution flow, such as a user request or a batch job.

### Creating a Trace

```go
ctx := context.Background()

trace, err := client.NewTrace().
    Name("chat-completion").
    UserID("user-123").
    SessionID("session-456").
    Input(map[string]any{
        "messages": []map[string]string{
            {"role": "user", "content": "Hello!"},
        },
    }).
    Metadata(langfuse.Metadata{
        "environment": "production",
        "version":     "1.0.0",
    }).
    Create(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Trace Builder Methods

The `TraceBuilder` returned by `client.NewTrace()` supports the following
setters (all return the builder for chaining):

```go
b := client.NewTrace()
b.ID("custom-id")                 // custom trace ID (auto-generated if omitted)
b.Name("chat-completion")         // trace name
b.UserID("user-123")              // user identifier
b.SessionID("session-456")        // session identifier
b.Input(map[string]any{})         // input data (any value)
b.Output(map[string]any{})        // output data (any value)
b.Metadata(langfuse.Metadata{})   // additional metadata
b.Tags([]string{"production"})    // tags for filtering
b.Release("v1.0.0")               // release version
b.Version("1")                    // trace version
b.Environment("production")       // deployment environment
b.Public(true)                    // make trace public
```

### Updating a Trace

Use `Update()` to get an update builder, then `Apply(ctx)`:

```go
ctx := context.Background()

err := trace.Update().
    Output(map[string]any{
        "response": "Hello! How can I help you?",
    }).
    Metadata(langfuse.Metadata{
        "duration_ms": 150,
    }).
    Apply(ctx)
if err != nil {
    log.Printf("update failed: %v", err)
}
```

### Trace Properties

```go
traceID := trace.ID() // Get the trace ID
```

## Generations

A **Generation** represents an LLM call, including prompts, completions, and token usage.

### Creating a Generation

```go
ctx := context.Background()

generation, err := trace.NewGeneration().
    Name("openai-chat").
    Model("gpt-4").
    ModelParameters(langfuse.Metadata{
        "temperature": 0.7,
    }).
    Input([]map[string]string{
        {"role": "user", "content": "What is 2+2?"},
    }).
    Create(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Generation Builder Methods

```go
g := trace.NewGeneration()
g.Name("openai-chat")                  // generation name
g.Model("gpt-4")                       // model name
g.ModelParameters(langfuse.Metadata{}) // model parameters (temp, top_p, etc.)
g.Input("prompt")                      // input/prompt (any value)
g.Output("completion")                 // output/completion (any value)
g.Metadata(langfuse.Metadata{})        // additional metadata
g.Usage(&langfuse.Usage{})             // token usage
g.UsageTokens(12, 6)                   // shorthand: input, output tokens
g.PromptName("chat-template")          // associated prompt name
g.PromptVersion(1)                     // prompt version
```

### Finishing a Generation

Finish a generation with `End`, `EndWithOutput`, or `EndWithUsage`. The
`EndWithUsage` form records the output plus input and output token counts:

```go
ctx := context.Background()

// Call OpenAI...
response := "2+2 equals 4."

// Record output and token usage (input tokens, output tokens)
if err := generation.EndWithUsage(ctx, response, 12, 6); err != nil {
    log.Printf("end failed: %v", err)
}
```

You can also update a generation incrementally before ending it:

```go
ctx := context.Background()

err := generation.Update().
    Output(map[string]any{"answer": "4"}).
    Usage(&langfuse.Usage{
        Input:  12,
        Output: 6,
        Total:  18,
    }).
    Apply(ctx)
if err != nil {
    log.Printf("update failed: %v", err)
}
```

## Spans

A **Span** represents a logical step or operation in your application.

### Creating a Span

```go
ctx := context.Background()

span, err := trace.NewSpan().
    Name("retrieve-documents").
    Input(map[string]any{
        "query": "quantum computing",
        "limit": 5,
    }).
    Create(ctx)
if err != nil {
    log.Fatal(err)
}
```

### Span Builder Methods

```go
s := trace.NewSpan()
s.ID("custom-id")               // custom span ID
s.Name("retrieve-documents")    // span name
s.Input(map[string]any{})       // input data (any value)
s.Output(map[string]any{})      // output data (any value)
s.Metadata(langfuse.Metadata{}) // additional metadata
s.Level(langfuse.ObservationLevelDefault)
```

### Finishing a Span

```go
ctx := context.Background()

// End with an output payload
err := span.EndWithOutput(ctx, map[string]any{
    "documents": []string{"doc1", "doc2", "doc3"},
    "count":     3,
})
if err != nil {
    log.Printf("end failed: %v", err)
}

// Or end without output
// err := span.End(ctx)
```

### Nested Spans

Spans can be nested to represent sub-operations. Call `NewSpan()` on a span
context to create a child:

```go
ctx := context.Background()

// Parent span
retrievalSpan, err := trace.NewSpan().Name("retrieval-pipeline").Create(ctx)
if err != nil {
    log.Fatal(err)
}

// Child span
embeddingSpan, err := retrievalSpan.NewSpan().
    Name("generate-embeddings").
    Input(map[string]any{"text": "quantum computing"}).
    Create(ctx)
if err != nil {
    log.Fatal(err)
}

if err := embeddingSpan.EndWithOutput(ctx, map[string]any{
    "embedding": []float64{0.1, 0.2, 0.3},
}); err != nil {
    log.Printf("end failed: %v", err)
}

if err := retrievalSpan.End(ctx); err != nil {
    log.Printf("end failed: %v", err)
}
```

## Events

An **Event** represents a discrete occurrence or log entry. Events are
point-in-time, so they are created and sent in a single `Create(ctx)` call.

### Creating an Event

```go
ctx := context.Background()

err := trace.NewEvent().
    Name("user-feedback").
    Input(map[string]any{
        "rating":  5,
        "comment": "Great response!",
    }).
    Create(ctx)
if err != nil {
    log.Printf("event failed: %v", err)
}
```

### Event Builder Methods

```go
e := trace.NewEvent()
e.ID("custom-id")               // custom event ID
e.Name("cache-hit")             // event name
e.Input(map[string]any{})       // event data (any value)
e.Output(map[string]any{})      // output data (any value)
e.Metadata(langfuse.Metadata{}) // additional metadata
e.Level(langfuse.ObservationLevelDefault)
```

### Common Event Use Cases

```go
ctx := context.Background()

// Logging
_ = trace.NewEvent().
    Name("cache-hit").
    Input(map[string]any{"cache_key": "user:123:profile"}).
    Create(ctx)

// System events
_ = trace.NewEvent().
    Name("rate-limit-exceeded").
    Metadata(langfuse.Metadata{
        "user_id": "user-123",
        "limit":   100,
        "current": 105,
    }).
    Create(ctx)
```

## Scores

Scores let you rate observations (traces, generations, spans). Build a score
with `NewScore()` and finalize it with `Create(ctx)`.

### Adding a Score

```go
ctx := context.Background()

// Score a trace
err := trace.NewScore().
    Name("user-rating").
    NumericValue(5.0).
    Comment("Excellent response").
    Create(ctx)
if err != nil {
    log.Printf("score failed: %v", err)
}

// Score a generation
err = generation.NewScore().
    Name("quality").
    NumericValue(0.95).
    Create(ctx)
if err != nil {
    log.Printf("score failed: %v", err)
}
```

### Score Value Types

Scores support numeric, categorical, and boolean values:

```go
ctx := context.Background()

_ = trace.NewScore().Name("accuracy").NumericValue(0.95).Create(ctx)
_ = trace.NewScore().Name("sentiment").CategoricalValue("positive").Create(ctx)
_ = trace.NewScore().Name("passed").BooleanValue(true).Create(ctx)
```

## Complete Example: RAG Application

Here's a complete example tracing a RAG (Retrieval-Augmented Generation) application:

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

    // Create main trace
    trace, err := client.NewTrace().
        Name("rag-query").
        UserID("user-123").
        Input(map[string]any{"query": "What is quantum computing?"}).
        Create(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Step 1: Query embedding
    embeddingSpan, err := trace.NewSpan().
        Name("generate-query-embedding").
        Input(map[string]any{"text": "What is quantum computing?"}).
        Create(ctx)
    if err != nil {
        log.Fatal(err)
    }

    embedding := generateEmbedding("What is quantum computing?")
    if err := embeddingSpan.EndWithOutput(ctx, map[string]any{"embedding": embedding}); err != nil {
        log.Printf("end failed: %v", err)
    }

    // Step 2: Document retrieval
    retrievalSpan, err := trace.NewSpan().
        Name("retrieve-documents").
        Input(map[string]any{"embedding": embedding, "top_k": 3}).
        Create(ctx)
    if err != nil {
        log.Fatal(err)
    }

    documents := retrieveDocuments(embedding)
    if err := retrievalSpan.EndWithOutput(ctx, map[string]any{
        "documents": documents,
        "count":     len(documents),
    }); err != nil {
        log.Printf("end failed: %v", err)
    }

    // Step 3: LLM generation
    generation, err := trace.NewGeneration().
        Name("openai-chat").
        Model("gpt-4").
        Input(map[string]any{
            "messages": []map[string]string{
                {"role": "system", "content": "Answer based on context."},
                {"role": "user", "content": formatPrompt(documents, "What is quantum computing?")},
            },
        }).
        Create(ctx)
    if err != nil {
        log.Fatal(err)
    }

    response := callLLM(documents, "What is quantum computing?")
    if err := generation.EndWithUsage(ctx, response, 200, 100); err != nil {
        log.Printf("end failed: %v", err)
    }

    // Step 4: Log result
    if err := trace.NewEvent().
        Name("response-generated").
        Input(map[string]any{"success": true, "latency_ms": 1500}).
        Create(ctx); err != nil {
        log.Printf("event failed: %v", err)
    }

    // Update trace with final output
    if err := trace.Update().
        Output(map[string]any{"answer": response}).
        Apply(ctx); err != nil {
        log.Printf("update failed: %v", err)
    }

    // Flush
    if err := client.Flush(ctx); err != nil {
        log.Fatal(err)
    }
}

func generateEmbedding(text string) []float64 {
    return []float64{0.1, 0.2, 0.3}
}

func retrieveDocuments(embedding []float64) []string {
    return []string{"doc1", "doc2", "doc3"}
}

func formatPrompt(docs []string, query string) string {
    return "Context: " + docs[0] + "\n\nQuestion: " + query
}

func callLLM(docs []string, query string) map[string]any {
    return map[string]any{
        "answer": "Quantum computing uses quantum mechanics...",
    }
}
```

## Best Practices

### 1. Use Meaningful Names

Use descriptive names that indicate what each observation does:

```go
ctx := context.Background()

// Good
_, _ = client.NewTrace().Name("user-chat-completion").Create(ctx)

// Bad
_, _ = client.NewTrace().Name("trace1").Create(ctx)
```

### 2. Include Relevant Metadata

Add metadata that helps with debugging and analysis:

```go
ctx := context.Background()

_, _ = client.NewTrace().
    Name("chat-completion").
    Metadata(langfuse.Metadata{
        "environment": "production",
        "version":     "1.2.3",
        "region":      "us-east-1",
    }).
    Create(ctx)
```

### 3. Track Token Usage

Always include usage information for generations via `EndWithUsage`:

```go
ctx := context.Background()

if err := generation.EndWithUsage(ctx, "response", 100, 50); err != nil {
    log.Printf("end failed: %v", err)
}
```

### 4. Add User Context

Include user information for better analysis:

```go
ctx := context.Background()

_, _ = client.NewTrace().
    Name("chat-completion").
    UserID("user-123").
    SessionID("session-456").
    Create(ctx)
```

## Next Steps

- [Evaluation Guide](../evaluation/) - Learn about scoring and evaluation
- [Configuration](../configuration/) - Customize SDK behavior
- [API Reference](../api-reference/) - Complete type reference
