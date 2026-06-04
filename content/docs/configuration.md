---
title: Configuration
weight: 2
---

The Langfuse Go SDK provides flexible configuration options to customize its behavior for your needs.

## Configuration Methods

There are three ways to configure the SDK:

1. **Functional options** (recommended)
2. **Environment variables**
3. **Default values**

## Basic Configuration

### API Keys

The public and secret keys are required arguments to `New`. They are passed
positionally, not as options:

```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...")
```

Or read them from environment variables and pass them along:

```bash
export LANGFUSE_PUBLIC_KEY="pk-lf-..."
export LANGFUSE_SECRET_KEY="sk-lf-..."
```

```go
client, err := langfuse.New(
    os.Getenv("LANGFUSE_PUBLIC_KEY"),
    os.Getenv("LANGFUSE_SECRET_KEY"),
)
```

All additional configuration is supplied as `ConfigOption` values after the two
keys.

## Region Configuration

Langfuse supports multiple deployment regions.

### Cloud EU (Default)

```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...",
    langfuse.WithRegion(langfuse.RegionEU),
)
```

### Cloud US

```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...",
    langfuse.WithRegion(langfuse.RegionUS),
)
```

### Self-Hosted

For self-hosted deployments, use a custom base URL:

```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...",
    langfuse.WithBaseURL("https://langfuse.yourcompany.com"),
)
```

Or via environment variable:

```bash
export LANGFUSE_BASE_URL="https://langfuse.yourcompany.com"
```

## Batching Configuration

The SDK batches events for efficient network usage. Customize batching behavior:

```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...",
    langfuse.WithBatchSize(50),                 // Max events per batch (default: 100)
    langfuse.WithFlushInterval(5*time.Second),  // Flush interval (default: 5s)
    langfuse.WithMaxRetries(5),                 // Max retry attempts (default: 3)
)
```

### Batch Size

Controls how many events are sent in a single HTTP request:

- **Larger values**: Fewer HTTP requests, better throughput
- **Smaller values**: More frequent updates, lower latency
- **Default**: 100 events

```go
langfuse.WithBatchSize(50)
```

### Flush Interval

How often to send batched events:

- **Shorter intervals**: More real-time updates
- **Longer intervals**: Better batching efficiency
- **Default**: 5 seconds

```go
langfuse.WithFlushInterval(5 * time.Second)
```

### Queue Size

The internal event queue has a fixed capacity. Increase it for high-throughput
workloads:

```go
langfuse.WithBatchQueueSize(2000)
```

## Timeout Configuration

### HTTP Timeout

Set the timeout for HTTP requests:

```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...",
    langfuse.WithTimeout(30*time.Second), // Default: 30s
)
```

### Context-Based Timeouts

For shutdown and flush operations:

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

client.Shutdown(ctx)
```

## Retry Configuration

Configure retry behavior for failed requests:

```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...",
    langfuse.WithMaxRetries(5), // Default: 3
)
```

The SDK uses exponential backoff for retries.

## Debug Logging

Enable verbose logging while developing:

```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...",
    langfuse.WithDebug(true),
)
```

## HTTP Client Customization

Use a custom HTTP client for advanced scenarios:

```go
httpClient := &http.Client{
    Timeout: 30 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}

client, err := langfuse.New("pk-lf-...", "sk-lf-...",
    langfuse.WithHTTPClient(httpClient),
)
```

## Complete Configuration Example

```go
package main

import (
    "context"
    "log"
    "net/http"
    "time"

    langfuse "github.com/jdziat/langfuse-go"
)

func main() {
    // Custom HTTP client
    httpClient := &http.Client{
        Timeout: 30 * time.Second,
    }

    // Initialize with all options
    client, err := langfuse.New("pk-lf-...", "sk-lf-...",
        // Region / Base URL
        langfuse.WithRegion(langfuse.RegionUS),

        // Batching
        langfuse.WithBatchSize(75),
        langfuse.WithFlushInterval(8*time.Second),

        // Retry
        langfuse.WithMaxRetries(5),

        // HTTP
        langfuse.WithTimeout(20*time.Second),
        langfuse.WithHTTPClient(httpClient),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(context.Background())

    // Your application code
}
```

## Environment Variables Reference

Common configuration options can be set via environment variables:

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `LANGFUSE_PUBLIC_KEY` | string | - | Public API key |
| `LANGFUSE_SECRET_KEY` | string | - | Secret API key |
| `LANGFUSE_BASE_URL` | string | EU Cloud | Base URL for API |

## Configuration Options Reference

### WithRegion

Set the Langfuse cloud region.

```go
langfuse.WithRegion(langfuse.RegionUS) // or langfuse.RegionEU
```

### WithBaseURL

Set a custom base URL (for self-hosted deployments).

```go
langfuse.WithBaseURL("https://langfuse.yourcompany.com")
```

### WithBatchSize

Set the maximum number of events per batch.

```go
langfuse.WithBatchSize(50)
```

### WithFlushInterval

Set how often batched events are sent.

```go
langfuse.WithFlushInterval(5 * time.Second)
```

### WithBatchQueueSize

Set the capacity of the internal event queue.

```go
langfuse.WithBatchQueueSize(2000)
```

### WithMaxRetries

Set the maximum number of retry attempts.

```go
langfuse.WithMaxRetries(5)
```

### WithTimeout

Set the HTTP request timeout.

```go
langfuse.WithTimeout(30 * time.Second)
```

### WithDebug

Enable debug logging.

```go
langfuse.WithDebug(true)
```

### WithHTTPClient

Use a custom HTTP client.

```go
langfuse.WithHTTPClient(&http.Client{Timeout: 30 * time.Second})
```

## Best Practices

### Development

For development, use smaller batch sizes and shorter intervals for faster feedback:

```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...",
    langfuse.WithBatchSize(10),
    langfuse.WithFlushInterval(2*time.Second),
    langfuse.WithDebug(true),
)
```

### Production

For production, optimize for throughput:

```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...",
    langfuse.WithBatchSize(100),
    langfuse.WithFlushInterval(10*time.Second),
    langfuse.WithMaxRetries(3),
)
```

### High-Volume Applications

For applications generating many events:

```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...",
    langfuse.WithBatchSize(200),
    langfuse.WithBatchQueueSize(5000),
    langfuse.WithFlushInterval(15*time.Second),
    langfuse.WithTimeout(30*time.Second),
)
```

### Graceful Shutdown

Always ensure graceful shutdown in production. Give the shutdown a bounded
timeout so queued events get a chance to flush:

```go
// Graceful shutdown with timeout
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

if err := client.Shutdown(ctx); err != nil {
    log.Printf("Shutdown error: %v", err)
}
```

## Configuration Validation

The SDK validates configuration at initialization:

```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...",
    langfuse.WithBatchSize(-1), // Invalid
)
if err != nil {
    // Handle configuration error
    log.Fatal(err)
}
```

Common validation errors:

- Invalid or missing API keys
- Negative batch size
- Negative flush interval
- Invalid base URL format

## Next Steps

- [Tracing Guide](../tracing/) - Learn about traces, spans, and generations
- [API Reference](../api-reference/) - Complete type reference
- [Getting Started](../getting-started/) - Basic setup guide
