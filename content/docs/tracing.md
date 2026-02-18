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

## Traces

A **Trace** represents a complete execution flow, such as a user request or a batch job.

### Creating a Trace

```go
trace := client.Trace(langfuse.TraceParams{
    Name:   "chat-completion",
    UserID: "user-123",
    SessionID: "session-456",
    Input: map[string]any{
        "messages": []map[string]string{
            {"role": "user", "content": "Hello!"},
        },
    },
    Metadata: map[string]any{
        "environment": "production",
        "version": "1.0.0",
    },
})
```

### Trace Parameters

```go
type TraceParams struct {
    // Required
    Name string // Name of the trace

    // Optional
    ID        string         // Custom trace ID (auto-generated if not provided)
    UserID    string         // User identifier
    SessionID string         // Session identifier
    Input     map[string]any // Input data
    Output    map[string]any // Output data
    Metadata  map[string]any // Additional metadata
    Tags      []string       // Tags for filtering
    Public    bool           // Make trace public
}
```

### Updating a Trace

```go
trace.Update(langfuse.TraceParams{
    Output: map[string]any{
        "response": "Hello! How can I help you?",
    },
    Metadata: map[string]any{
        "duration_ms": 150,
    },
})
```

### Trace Properties

```go
traceID := trace.ID // Get the trace ID
```

## Generations

A **Generation** represents an LLM call, including prompts, completions, and token usage.

### Creating a Generation

```go
generation := trace.Generation(langfuse.GenerationParams{
    Name:  "openai-chat",
    Model: "gpt-4",
    Input: map[string]any{
        "messages": []map[string]string{
            {"role": "user", "content": "What is 2+2?"},
        },
        "temperature": 0.7,
    },
})
```

### Generation Parameters

```go
type GenerationParams struct {
    // Required
    Name  string // Name of the generation
    Model string // Model name (e.g., "gpt-4")

    // Optional
    ID              string         // Custom generation ID
    Input           map[string]any // Input (prompt)
    Output          map[string]any // Output (completion)
    Metadata        map[string]any // Additional metadata
    ModelParameters map[string]any // Model parameters (temp, top_p, etc.)
    Usage           *Usage         // Token usage
    PromptName      string         // Associated prompt name
    PromptVersion   int            // Prompt version
}

type Usage struct {
    PromptTokens     int
    CompletionTokens int
    TotalTokens      int
}
```

### Updating a Generation

```go
generation.Update(langfuse.GenerationParams{
    Output: map[string]any{
        "choices": []map[string]any{
            {"message": map[string]string{
                "role": "assistant",
                "content": "2+2 equals 4.",
            }},
        },
    },
    Usage: &langfuse.Usage{
        PromptTokens:     12,
        CompletionTokens: 6,
        TotalTokens:      18,
    },
})
```

### Complete Generation Example

```go
// Create generation
generation := trace.Generation(langfuse.GenerationParams{
    Name:  "openai-completion",
    Model: "gpt-4",
    Input: map[string]any{
        "messages": []map[string]string{
            {"role": "system", "content": "You are a helpful assistant."},
            {"role": "user", "content": "Explain quantum computing"},
        },
    },
    ModelParameters: map[string]any{
        "temperature": 0.7,
        "max_tokens":  500,
    },
})

// Call OpenAI...
response := callOpenAI()

// Update with response
generation.Update(langfuse.GenerationParams{
    Output: response,
    Usage: &langfuse.Usage{
        PromptTokens:     25,
        CompletionTokens: 150,
        TotalTokens:      175,
    },
})
```

## Spans

A **Span** represents a logical step or operation in your application.

### Creating a Span

```go
span := trace.Span(langfuse.SpanParams{
    Name: "retrieve-documents",
    Input: map[string]any{
        "query": "quantum computing",
        "limit": 5,
    },
})
```

### Span Parameters

```go
type SpanParams struct {
    // Required
    Name string // Name of the span

    // Optional
    ID       string         // Custom span ID
    Input    map[string]any // Input data
    Output   map[string]any // Output data
    Metadata map[string]any // Additional metadata
}
```

### Updating a Span

```go
span.Update(langfuse.SpanParams{
    Output: map[string]any{
        "documents": []string{"doc1", "doc2", "doc3"},
        "count": 3,
    },
})
```

### Nested Spans

Spans can be nested to represent sub-operations:

```go
// Parent span
retrievalSpan := trace.Span(langfuse.SpanParams{
    Name: "retrieval-pipeline",
})

// Child span
embeddingSpan := retrievalSpan.Span(langfuse.SpanParams{
    Name: "generate-embeddings",
    Input: map[string]any{
        "text": "quantum computing",
    },
})

embeddingSpan.Update(langfuse.SpanParams{
    Output: map[string]any{
        "embedding": []float64{0.1, 0.2, 0.3},
    },
})

// Another child span
searchSpan := retrievalSpan.Span(langfuse.SpanParams{
    Name: "vector-search",
    Input: map[string]any{
        "embedding": []float64{0.1, 0.2, 0.3},
    },
})

searchSpan.Update(langfuse.SpanParams{
    Output: map[string]any{
        "results": []string{"doc1", "doc2"},
    },
})
```

## Events

An **Event** represents a discrete occurrence or log entry.

### Creating an Event

```go
event := trace.Event(langfuse.EventParams{
    Name: "user-feedback",
    Input: map[string]any{
        "rating": 5,
        "comment": "Great response!",
    },
})
```

### Event Parameters

```go
type EventParams struct {
    // Required
    Name string // Name of the event

    // Optional
    ID       string         // Custom event ID
    Input    map[string]any // Event data
    Metadata map[string]any // Additional metadata
}
```

### Common Event Use Cases

#### Logging

```go
trace.Event(langfuse.EventParams{
    Name: "cache-hit",
    Input: map[string]any{
        "cache_key": "user:123:profile",
    },
})
```

#### User Actions

```go
trace.Event(langfuse.EventParams{
    Name: "button-click",
    Input: map[string]any{
        "button": "submit",
        "page": "checkout",
    },
})
```

#### System Events

```go
trace.Event(langfuse.EventParams{
    Name: "rate-limit-exceeded",
    Metadata: map[string]any{
        "user_id": "user-123",
        "limit": 100,
        "current": 105,
    },
})
```

## Scores

Scores allow you to rate observations (traces, generations, spans).

### Adding a Score

```go
// Score a trace
trace.Score(langfuse.ScoreParams{
    Name:  "user-rating",
    Value: 5.0,
    Comment: "Excellent response",
})

// Score a generation
generation.Score(langfuse.ScoreParams{
    Name:  "quality",
    Value: 0.95,
})
```

### Score Parameters

```go
type ScoreParams struct {
    Name    string  // Score name
    Value   float64 // Score value
    Comment string  // Optional comment
}
```

## Complete Example: RAG Application

Here's a complete example tracing a RAG (Retrieval-Augmented Generation) application:

```go
package main

import (
    "context"
    "log"

    "github.com/jdziat/langfuse-go/langfuse"
)

func main() {
    client, err := langfuse.New()
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(context.Background())

    // Create main trace
    trace := client.Trace(langfuse.TraceParams{
        Name:   "rag-query",
        UserID: "user-123",
        Input: map[string]any{
            "query": "What is quantum computing?",
        },
    })

    // Step 1: Query embedding
    embeddingSpan := trace.Span(langfuse.SpanParams{
        Name: "generate-query-embedding",
        Input: map[string]any{
            "text": "What is quantum computing?",
        },
    })

    embedding := generateEmbedding("What is quantum computing?")

    embeddingSpan.Update(langfuse.SpanParams{
        Output: map[string]any{
            "embedding": embedding,
        },
    })

    // Step 2: Document retrieval
    retrievalSpan := trace.Span(langfuse.SpanParams{
        Name: "retrieve-documents",
        Input: map[string]any{
            "embedding": embedding,
            "top_k": 3,
        },
    })

    documents := retrieveDocuments(embedding)

    retrievalSpan.Update(langfuse.SpanParams{
        Output: map[string]any{
            "documents": documents,
            "count": len(documents),
        },
    })

    // Step 3: LLM generation
    generation := trace.Generation(langfuse.GenerationParams{
        Name:  "openai-chat",
        Model: "gpt-4",
        Input: map[string]any{
            "messages": []map[string]string{
                {"role": "system", "content": "Answer based on context."},
                {"role": "user", "content": formatPrompt(documents, "What is quantum computing?")},
            },
        },
    })

    response := callLLM(documents, "What is quantum computing?")

    generation.Update(langfuse.GenerationParams{
        Output: response,
        Usage: &langfuse.Usage{
            PromptTokens:     200,
            CompletionTokens: 100,
            TotalTokens:      300,
        },
    })

    // Step 4: Log result
    trace.Event(langfuse.EventParams{
        Name: "response-generated",
        Input: map[string]any{
            "success": true,
            "latency_ms": 1500,
        },
    })

    // Update trace with final output
    trace.Update(langfuse.TraceParams{
        Output: map[string]any{
            "answer": response,
        },
    })

    // Flush
    if err := client.Flush(context.Background()); err != nil {
        log.Fatal(err)
    }
}

func generateEmbedding(text string) []float64 {
    // Your embedding logic
    return []float64{0.1, 0.2, 0.3}
}

func retrieveDocuments(embedding []float64) []string {
    // Your retrieval logic
    return []string{"doc1", "doc2", "doc3"}
}

func formatPrompt(docs []string, query string) string {
    // Format prompt with documents
    return "Context: " + docs[0] + "\n\nQuestion: " + query
}

func callLLM(docs []string, query string) map[string]any {
    // Your LLM call
    return map[string]any{
        "answer": "Quantum computing uses quantum mechanics...",
    }
}
```

## Best Practices

### 1. Use Meaningful Names

Use descriptive names that indicate what each observation does:

```go
// Good
trace := client.Trace(langfuse.TraceParams{
    Name: "user-chat-completion",
})

// Bad
trace := client.Trace(langfuse.TraceParams{
    Name: "trace1",
})
```

### 2. Include Relevant Metadata

Add metadata that helps with debugging and analysis:

```go
trace := client.Trace(langfuse.TraceParams{
    Name: "chat-completion",
    Metadata: map[string]any{
        "environment": "production",
        "version": "1.2.3",
        "region": "us-east-1",
        "feature_flags": map[string]bool{
            "new_model": true,
        },
    },
})
```

### 3. Track Token Usage

Always include usage information for generations:

```go
generation.Update(langfuse.GenerationParams{
    Output: response,
    Usage: &langfuse.Usage{
        PromptTokens:     100,
        CompletionTokens: 50,
        TotalTokens:      150,
    },
})
```

### 4. Use Spans for Logical Steps

Break down complex flows into logical spans:

```go
// Clear separation of concerns
retrievalSpan := trace.Span(langfuse.SpanParams{Name: "retrieval"})
processingSpan := trace.Span(langfuse.SpanParams{Name: "processing"})
generationSpan := trace.Span(langfuse.SpanParams{Name: "generation"})
```

### 5. Add User Context

Include user information for better analysis:

```go
trace := client.Trace(langfuse.TraceParams{
    Name:      "chat-completion",
    UserID:    "user-123",
    SessionID: "session-456",
})
```

### 6. Log Important Events

Use events to track significant occurrences:

```go
trace.Event(langfuse.EventParams{
    Name: "cache-miss",
})

trace.Event(langfuse.EventParams{
    Name: "rate-limit-applied",
})
```

## Next Steps

- [Evaluation Guide](../evaluation/) - Learn about scoring and evaluation
- [Configuration](../configuration/) - Customize SDK behavior
- [API Reference](../api-reference/) - Complete type reference
