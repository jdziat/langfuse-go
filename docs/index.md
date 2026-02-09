# Langfuse Go SDK

The official Go SDK for [Langfuse](https://langfuse.com), the open-source LLM engineering platform.

## Overview

Langfuse helps you trace, evaluate, and monitor your LLM applications. This Go SDK provides a type-safe, idiomatic way to integrate Langfuse into your Go applications.

## Features

- **Type-Safe Tracing**: Strongly-typed API for creating traces, spans, generations, and events
- **Automatic Batching**: Efficient background processing of telemetry data
- **LLM-as-a-Judge**: Built-in support for AI-powered evaluation workflows
- **Flexible Configuration**: Support for all Langfuse regions and custom configurations
- **Production Ready**: Comprehensive error handling and graceful shutdown
- **Zero Dependencies**: Minimal external dependencies for easy integration

## Quick Start

```go
package main

import (
    "context"
    "log"

    "github.com/jdziat/langfuse-go/langfuse"
)

func main() {
    // Initialize client
    client, err := langfuse.New(
        langfuse.WithPublicKey("pk_..."),
        langfuse.WithSecretKey("sk_..."),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(context.Background())

    // Create a trace
    trace := client.Trace(langfuse.TraceParams{
        Name: "chat-completion",
        Input: map[string]any{
            "messages": []map[string]string{
                {"role": "user", "content": "Hello!"},
            },
        },
    })

    // Add a generation
    generation := trace.Generation(langfuse.GenerationParams{
        Name:  "openai-chat",
        Model: "gpt-4",
        Input: map[string]any{
            "messages": []map[string]string{
                {"role": "user", "content": "Hello!"},
            },
        },
    })

    // Update with response
    generation.Update(langfuse.GenerationParams{
        Output: map[string]any{
            "choices": []map[string]any{
                {"message": map[string]string{
                    "role": "assistant",
                    "content": "Hi! How can I help you?",
                }},
            },
        },
        Usage: &langfuse.Usage{
            PromptTokens:     10,
            CompletionTokens: 8,
            TotalTokens:      18,
        },
    })

    // Flush and wait
    if err := client.Flush(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

## Installation

```bash
go get github.com/jdziat/langfuse-go
```

## Next Steps

- [Getting Started Guide](getting-started.md) - Detailed setup and first trace
- [Configuration Reference](configuration.md) - All configuration options
- [Tracing Documentation](tracing.md) - Complete tracing guide
- [Evaluation Workflows](evaluation.md) - LLM-as-a-Judge integration
- [API Reference](api-reference.md) - Quick reference for all types
- [Migration Guide](migration.md) - Upgrading from previous versions

## Support

- **GitHub Issues**: [Report bugs or request features](https://github.com/jdziat/langfuse-go/issues)
- **Documentation**: [Langfuse Docs](https://langfuse.com/docs)
- **Community**: [Join the Langfuse Discord](https://langfuse.com/discord)

## License

MIT License - see [LICENSE](https://github.com/jdziat/langfuse-go/blob/main/LICENSE) for details.
