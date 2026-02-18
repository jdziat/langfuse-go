---
title: Getting Started
weight: 1
---

This guide will help you get up and running with the Langfuse Go SDK.

## Prerequisites

- Go 1.21 or later
- A Langfuse account (sign up at [langfuse.com](https://langfuse.com))
- Your API keys from the Langfuse dashboard

## Installation

Install the SDK using `go get`:

```bash
go get github.com/jdziat/langfuse-go
```

## Getting Your API Keys

1. Log in to your Langfuse dashboard
2. Navigate to Settings > API Keys
3. Copy your Public Key (starts with `pk_`) and Secret Key (starts with `sk_`)

{{< callout type="warning" >}}
Keep your Secret Key secure. Never commit it to version control or expose it in client-side code.
{{< /callout >}}

## Basic Setup

### Initialize the Client

Create a new Langfuse client with your API keys:

```go
package main

import (
    "context"
    "log"

    "github.com/jdziat/langfuse-go/langfuse"
)

func main() {
    client, err := langfuse.New(
        langfuse.WithPublicKey("pk_lf_..."),
        langfuse.WithSecretKey("sk_lf_..."),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(context.Background())

    // Your code here
}
```

### Using Environment Variables

You can also configure the client using environment variables:

```bash
export LANGFUSE_PUBLIC_KEY="pk_lf_..."
export LANGFUSE_SECRET_KEY="sk_lf_..."
```

Then initialize without options:

```go
client, err := langfuse.New()
if err != nil {
    log.Fatal(err)
}
defer client.Shutdown(context.Background())
```

## Your First Trace

Let's create a simple trace to verify everything is working:

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/jdziat/langfuse-go/langfuse"
)

func main() {
    // Initialize client
    client, err := langfuse.New(
        langfuse.WithPublicKey("pk_lf_..."),
        langfuse.WithSecretKey("sk_lf_..."),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(context.Background())

    // Create a trace
    trace := client.Trace(langfuse.TraceParams{
        Name:   "hello-world",
        UserID: "user-123",
        Input: map[string]any{
            "query": "Hello, Langfuse!",
        },
    })

    // Simulate some work
    time.Sleep(100 * time.Millisecond)

    // Update the trace with output
    trace.Update(langfuse.TraceParams{
        Output: map[string]any{
            "response": "Welcome to Langfuse Go SDK!",
        },
    })

    // Flush to ensure data is sent
    if err := client.Flush(context.Background()); err != nil {
        log.Fatal(err)
    }

    log.Println("Trace created successfully!")
    log.Printf("View in dashboard: https://cloud.langfuse.com/trace/%s", trace.ID)
}
```

Run the program:

```bash
go run main.go
```

You should see output like:

```
Trace created successfully!
View in dashboard: https://cloud.langfuse.com/trace/abc123...
```

## Adding a Generation

Generations represent LLM calls. Here's how to add one:

```go
// Create a trace
trace := client.Trace(langfuse.TraceParams{
    Name: "chat-example",
})

// Add a generation
generation := trace.Generation(langfuse.GenerationParams{
    Name:  "openai-completion",
    Model: "gpt-4",
    Input: map[string]any{
        "messages": []map[string]string{
            {"role": "user", "content": "What is 2+2?"},
        },
    },
})

// Update with the response
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

## Adding Spans

Spans represent logical steps in your application:

```go
trace := client.Trace(langfuse.TraceParams{
    Name: "document-processing",
})

// Add a span for data retrieval
retrievalSpan := trace.Span(langfuse.SpanParams{
    Name: "retrieve-documents",
    Input: map[string]any{
        "query": "machine learning",
    },
})

// Simulate work
documents := []string{"doc1", "doc2", "doc3"}

retrievalSpan.Update(langfuse.SpanParams{
    Output: map[string]any{
        "documents": documents,
    },
})

// Add another span for processing
processingSpan := trace.Span(langfuse.SpanParams{
    Name: "process-documents",
    Input: map[string]any{
        "documents": documents,
    },
})

processingSpan.Update(langfuse.SpanParams{
    Output: map[string]any{
        "summary": "Processed 3 documents",
    },
})
```

## Error Handling

Always handle errors appropriately:

```go
client, err := langfuse.New(
    langfuse.WithPublicKey("pk_lf_..."),
    langfuse.WithSecretKey("sk_lf_..."),
)
if err != nil {
    log.Fatalf("Failed to initialize Langfuse: %v", err)
}

// Always shutdown gracefully
defer func() {
    if err := client.Shutdown(context.Background()); err != nil {
        log.Printf("Error during shutdown: %v", err)
    }
}()
```

## Graceful Shutdown

The SDK batches events for efficiency. Always call `Shutdown` or `Flush` before your application exits:

```go
// Option 1: Shutdown (flushes and closes client)
defer client.Shutdown(context.Background())

// Option 2: Explicit flush (keeps client open)
if err := client.Flush(context.Background()); err != nil {
    log.Printf("Flush failed: %v", err)
}
```

## Next Steps

Now that you have the basics working, explore:

- [Configuration Options](../configuration/) - Customize batching, timeouts, and regions
- [Tracing Guide](../tracing/) - Learn about traces, spans, generations, and events
- [Evaluation Workflows](../evaluation/) - Integrate LLM-as-a-Judge
- [API Reference](../api-reference/) - Complete type reference

## Common Issues

### "Invalid API keys"

- Double-check your Public Key starts with `pk_`
- Ensure your Secret Key starts with `sk_`
- Verify you're using keys from the correct Langfuse project

### "Connection timeout"

- Check your network connection
- Verify you can reach `https://cloud.langfuse.com`
- If using self-hosted, ensure the correct base URL is set

### "Data not appearing in dashboard"

- Ensure you call `Flush()` or `Shutdown()` before program exit
- Check the program runs without errors
- Wait a few seconds for data to appear in the UI
