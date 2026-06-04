---
title: Getting Started
weight: 1
---

This guide will help you get up and running with the Langfuse Go SDK.

## Prerequisites

- Go 1.23 or later
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
3. Copy your Public Key (starts with `pk-lf-`) and Secret Key (starts with `sk-lf-`)

{{< callout type="warning" >}}
Keep your Secret Key secure. Never commit it to version control or expose it in client-side code.
{{< /callout >}}

## Basic Setup

### Initialize the Client

Create a new Langfuse client with your API keys. The public and secret keys are
passed positionally; additional behavior is configured with `ConfigOption`
functions:

```go
package main

import (
    "context"
    "log"

    langfuse "github.com/jdziat/langfuse-go"
)

func main() {
    client, err := langfuse.New("pk-lf-...", "sk-lf-...")
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
export LANGFUSE_PUBLIC_KEY="pk-lf-..."
export LANGFUSE_SECRET_KEY="sk-lf-..."
```

Read the keys from the environment yourself and pass them to `New`:

```go
package main

import (
    "context"
    "log"
    "os"

    langfuse "github.com/jdziat/langfuse-go"
)

func main() {
    client, err := langfuse.New(
        os.Getenv("LANGFUSE_PUBLIC_KEY"),
        os.Getenv("LANGFUSE_SECRET_KEY"),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(context.Background())
}
```

## Your First Trace

Let's create a simple trace to verify everything is working. Traces are built
with the fluent builder and finalized with `Create(ctx)`:

```go
package main

import (
    "context"
    "log"

    langfuse "github.com/jdziat/langfuse-go"
)

func main() {
    ctx := context.Background()

    // Initialize client
    client, err := langfuse.New("pk-lf-...", "sk-lf-...")
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(ctx)

    // Create a trace
    trace, err := client.NewTrace().
        Name("hello-world").
        UserID("user-123").
        Input(map[string]any{
            "query": "Hello, Langfuse!",
        }).
        Create(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Update the trace with output
    err = trace.Update().
        Output(map[string]any{
            "response": "Welcome to Langfuse Go SDK!",
        }).
        Apply(ctx)
    if err != nil {
        log.Printf("Failed to update trace: %v", err)
    }

    // Flush to ensure data is sent
    if err := client.Flush(ctx); err != nil {
        log.Fatal(err)
    }

    log.Println("Trace created successfully!")
    log.Printf("View in dashboard: https://cloud.langfuse.com/trace/%s", trace.ID())
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

### The Simple Option Form

If you don't need the full builder chain, `Client.Trace` creates a trace in a
single call using `TraceOption` functions:

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

    trace, err := client.Trace(ctx, "hello-world",
        langfuse.WithUserID("user-123"),
        langfuse.WithInput(map[string]any{"query": "Hello, Langfuse!"}),
        langfuse.WithTags("production", "demo"),
    )
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Created trace %s", trace.ID())
}
```

## Adding a Generation

Generations represent LLM calls. Create one on a trace, then finish it with
output and token usage:

```go
ctx := context.Background()

// Create a trace
trace, err := client.NewTrace().Name("chat-example").Create(ctx)
if err != nil {
    log.Fatal(err)
}

// Add a generation
generation, err := trace.NewGeneration().
    Name("openai-completion").
    Model("gpt-4").
    Input([]map[string]string{
        {"role": "user", "content": "What is 2+2?"},
    }).
    Create(ctx)
if err != nil {
    log.Fatal(err)
}

// End the generation with output and token usage (input, output tokens)
if err := generation.EndWithUsage(ctx, "2+2 equals 4.", 12, 6); err != nil {
    log.Printf("Failed to end generation: %v", err)
}
```

## Adding Spans

Spans represent logical steps in your application. End them with `End(ctx)` or
`EndWithOutput(ctx, output)`:

```go
ctx := context.Background()

trace, err := client.NewTrace().Name("document-processing").Create(ctx)
if err != nil {
    log.Fatal(err)
}

// Add a span for data retrieval
retrievalSpan, err := trace.NewSpan().
    Name("retrieve-documents").
    Input(map[string]any{"query": "machine learning"}).
    Create(ctx)
if err != nil {
    log.Fatal(err)
}

// Simulate work
documents := []string{"doc1", "doc2", "doc3"}

if err := retrievalSpan.EndWithOutput(ctx, map[string]any{
    "documents": documents,
}); err != nil {
    log.Printf("Failed to end span: %v", err)
}
```

## Error Handling

Always handle errors appropriately:

```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...")
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
- [Evaluation Workflows](../evaluation/) - Evaluation-ready tracing helpers
- [API Reference](../api-reference/) - Complete type reference

## Common Issues

### "Invalid API keys"

- Double-check your Public Key starts with `pk-lf-`
- Ensure your Secret Key starts with `sk-lf-`
- Verify you're using keys from the correct Langfuse project

### "Connection timeout"

- Check your network connection
- Verify you can reach `https://cloud.langfuse.com`
- If using self-hosted, ensure the correct base URL is set

### "Data not appearing in dashboard"

- Ensure you call `Flush()` or `Shutdown()` before program exit
- Check the program runs without errors
- Wait a few seconds for data to appear in the UI
