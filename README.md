# Langfuse Go SDK

[![Go Reference](https://pkg.go.dev/badge/github.com/jdziat/langfuse-go.svg)](https://pkg.go.dev/github.com/jdziat/langfuse-go)
[![Go Version](https://img.shields.io/github/go-mod/go-version/jdziat/langfuse-go)](https://golang.org/dl/)
[![CI](https://github.com/jdziat/langfuse-go/workflows/CI/badge.svg)](https://github.com/jdziat/langfuse-go/actions)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

Go SDK for [Langfuse](https://langfuse.com) - the open-source LLM observability platform. Track traces, spans, generations, and scores for your LLM applications with zero external dependencies.

## Features

- **Zero Dependencies**: Pure Go implementation with no external dependencies
- **Type-Safe API**: Strongly typed interfaces for all Langfuse entities
- **Automatic Batching**: Efficient event batching with configurable flush intervals
- **Concurrent-Safe**: Thread-safe operations for high-performance applications
- **Fluent Builder API**: Intuitive and chainable method calls
- **Full API Coverage**: Support for traces, spans, generations, events, scores, prompts, datasets, and more
- **Enhanced Metadata**: Rich utility methods for type-safe metadata operations
- **Go-Conventional Errors**: Standard error handling with As* helpers

## Installation

```bash
go get github.com/jdziat/langfuse-go
```

## Requirements

- Go 1.23 or later

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    "os"

    langfuse "github.com/jdziat/langfuse-go"
)

func main() {
    ctx := context.Background()

    // Create a new Langfuse client
    client, err := langfuse.New(
        os.Getenv("LANGFUSE_PUBLIC_KEY"),
        os.Getenv("LANGFUSE_SECRET_KEY"),
        langfuse.WithRegion(langfuse.RegionUS),
    )
    if err != nil {
        log.Fatalf("Failed to create client: %v", err)
    }
    defer client.Shutdown(ctx)

    // Create a trace for your LLM interaction
    trace, err := client.NewTrace().
        Name("chat-completion").
        UserID("user-123").
        Input(map[string]interface{}{
            "message": "What is the capital of France?",
        }).
        Tags([]string{"production", "chat"}).
        Create(ctx)
    if err != nil {
        log.Fatalf("Failed to create trace: %v", err)
    }

    // Add a generation (LLM call) to the trace
    generation, err := trace.Generation().
        Name("gpt-4-completion").
        Model("gpt-4").
        ModelParameters(map[string]interface{}{
            "temperature": 0.7,
            "max_tokens":  150,
        }).
        Input([]map[string]string{
            {"role": "user", "content": "What is the capital of France?"},
        }).
        Create(ctx)
    if err != nil {
        log.Fatalf("Failed to create generation: %v", err)
    }

    // End the generation with output and token usage
    err = generation.EndWithUsage(ctx,
        "The capital of France is Paris.",
        10, // input tokens
        8,  // output tokens
    )
    if err != nil {
        log.Printf("Failed to end generation: %v", err)
    }

    // Add a score to evaluate the generation
    err = generation.Score().
        Name("quality").
        NumericValue(0.95).
        Comment("Accurate and concise response").
        Create(ctx)
    if err != nil {
        log.Printf("Failed to create score: %v", err)
    }

    // Update trace with final output
    err = trace.Update().
        Output(map[string]interface{}{
            "response": "The capital of France is Paris.",
        }).
        Apply(ctx)
    if err != nil {
        log.Printf("Failed to update trace: %v", err)
    }

    // Flush pending events before shutdown
    if err := client.Flush(ctx); err != nil {
        log.Printf("Failed to flush: %v", err)
    }
}
```

### Working with Spans

Spans represent units of work within a trace:

```go
ctx := context.Background()

// Create metadata with utility methods
meta := langfuse.NewMetadata().
    Set("step", "preprocessing").
    Set("version", "2.0")

// Create a span for preprocessing
span, err := trace.Span().
    Name("preprocess-input").
    Input("raw user input").
    Metadata(meta).
    Create(ctx)
if err != nil {
    log.Fatalf("Failed to create span: %v", err)
}

// Perform your work...

// End the span with output
err = span.EndWithOutput(ctx, "processed input")
if err != nil {
    log.Printf("Failed to end span: %v", err)
}
```

### Nested Observations

Create parent-child relationships between observations:

```go
ctx := context.Background()

// Create a parent span
parentSpan, err := trace.Span().
    Name("parent-operation").
    Create(ctx)
if err != nil {
    log.Fatalf("Failed to create parent span: %v", err)
}

// Create a child span under the parent
childSpan, err := parentSpan.Span().
    Name("child-operation").
    Create(ctx)
if err != nil {
    log.Fatalf("Failed to create child span: %v", err)
}

// End observations
childSpan.End(ctx)
parentSpan.End(ctx)
```

### Configuration Options

Configure the client with various options:

```go
client, err := langfuse.New(
    publicKey,
    secretKey,
    langfuse.WithRegion(langfuse.RegionUS),       // or RegionEU
    langfuse.WithBatchSize(50),                   // events per batch
    langfuse.WithFlushInterval(5*time.Second),    // auto-flush interval
    langfuse.WithDebug(true),                     // enable debug logging
    langfuse.WithRelease("v1.0.0"),               // default release version
    langfuse.WithEnvironment("production"),       // default environment
)
```

### Working with Prompts

Retrieve and use prompts from Langfuse:

```go
// Get a prompt by name
prompt, err := client.Prompts().Get(ctx, "chat-template", nil)
if err != nil {
    log.Fatalf("Failed to get prompt: %v", err)
}

// Use the prompt in your generation
generation, err := trace.Generation().
    Name("chat-completion").
    Model("gpt-4").
    PromptName(prompt.Name).
    PromptVersion(prompt.Version).
    Create()
```

### Datasets and Evaluation

Work with datasets for testing and evaluation:

```go
// Create a dataset
dataset, err := client.Datasets().Create(ctx, &langfuse.Dataset{
    Name:        "qa-dataset",
    Description: "Question-answering evaluation set",
})

// Add items to the dataset
item, err := client.Datasets().CreateItem(ctx, &langfuse.DatasetItem{
    DatasetName:    "qa-dataset",
    Input:          map[string]interface{}{"question": "What is 2+2?"},
    ExpectedOutput: map[string]interface{}{"answer": "4"},
})

// Create a dataset run for evaluation
run, err := client.Datasets().CreateRun(ctx, &langfuse.DatasetRun{
    Name:        "evaluation-run-1",
    DatasetName: "qa-dataset",
})
```

## Package Structure

The SDK is organized into focused modules for maintainability:

```
langfuse-go/
├── client.go          # Main client and API entry point
├── lifecycle.go       # Client lifecycle (initialization, shutdown)
├── batching.go        # Event batching logic
├── queue.go           # Async event queue management
├── errors_api.go      # API error types (APIError)
├── errors_async.go    # Async/batch error types (IngestionError, ShutdownError)
├── errors_validation.go  # Validation error types
├── errors_helpers.go  # Go-conventional As* error helpers
├── helpers.go         # Metadata utilities and tracing helpers
├── pkg/config/        # Layered configuration types
└── ...                # Sub-clients, builders, and more
```

## API Reference

### Core Components

- **Client**: Main entry point for the SDK
- **Traces**: Top-level container for tracking an execution flow
- **Observations**: Individual operations within a trace
  - **Spans**: Generic operations or code blocks
  - **Generations**: LLM completions
  - **Events**: Point-in-time occurrences
- **Scores**: Evaluation metrics for traces or observations
- **Prompts**: Versioned prompt templates
- **Datasets**: Test and evaluation datasets

### Client Methods

```go
client.NewTrace()              // Create a new trace
client.Traces()                // Access traces client
client.Observations()          // Access observations client
client.Scores()                // Access scores client
client.Prompts()               // Access prompts client
client.Datasets()              // Access datasets client
client.Sessions()              // Access sessions client
client.Models()                // Access models client
client.Health(ctx)             // Check API health
client.Flush(ctx)              // Force flush pending events
client.Shutdown(ctx)           // Flush and close client

// Configured sub-clients (see "Configured Sub-clients" section)
client.PromptsWithOptions(...)
client.TracesWithOptions(...)
client.DatasetsWithOptions(...)
client.ScoresWithOptions(...)
client.SessionsWithOptions(...)
client.ModelsWithOptions(...)
```

### Configuration Constants

```go
// Regions
langfuse.RegionEU              // EU region (default)
langfuse.RegionUS              // US region

// Observation Levels
langfuse.ObservationLevelDebug
langfuse.ObservationLevelDefault
langfuse.ObservationLevelWarning
langfuse.ObservationLevelError

// Score Data Types
langfuse.ScoreDataTypeNumeric
langfuse.ScoreDataTypeCategorical
langfuse.ScoreDataTypeBoolean
```

## Error Handling

The SDK provides Go-conventional error handling with type extraction helpers:

### Using As* Helper Functions

The SDK provides `As*` helper functions that follow Go's `errors.As()` convention:

```go
trace, err := client.NewTrace().Name("example").Create(ctx)
if err != nil {
    // Check for API errors with AsAPIError
    if apiErr, ok := langfuse.AsAPIError(err); ok {
        if apiErr.IsRateLimited() {
            // Handle rate limiting
            delay := apiErr.RetryAfter
            log.Printf("Rate limited, retry after %v", delay)
        }
        log.Printf("API error %d: %s", apiErr.StatusCode, apiErr.Message)
    }

    // Check for validation errors
    if valErr, ok := langfuse.AsValidationError(err); ok {
        log.Printf("Validation failed: %v", valErr.Fields)
    }

    // Check for async/batch errors
    if ingErr, ok := langfuse.AsIngestionError(err); ok {
        log.Printf("Ingestion failed: %s", ingErr.Reason)
    }
}
```

### Available Error Helpers

```go
langfuse.AsAPIError(err)         // Extract *APIError
langfuse.AsValidationError(err)  // Extract *ValidationError
langfuse.AsIngestionError(err)   // Extract *IngestionError
langfuse.AsShutdownError(err)    // Extract *ShutdownError
langfuse.IsRetryable(err)        // Check if error is retryable
langfuse.RetryAfter(err)         // Get suggested retry delay
```

### Using Standard errors.Is/As

You can also use Go's standard error functions:

```go
// Check sentinel errors
if errors.Is(err, langfuse.ErrClientClosed) {
    // Client has been closed
}

// Type assertion with errors.As
var apiErr *langfuse.APIError
if errors.As(err, &apiErr) {
    if apiErr.IsRateLimited() {
        time.Sleep(apiErr.RetryAfter)
        // Retry operation
    }
}
```

## Metadata Utilities

The `Metadata` type provides rich utility methods for type-safe metadata operations:

### Basic Operations

```go
// Create and set values
meta := langfuse.NewMetadata()
meta.Set("user", "alice").Set("version", "1.0")

// Get values with type checking
if user, ok := meta.GetString("user"); ok {
    log.Printf("User: %s", user)
}

if count, ok := meta.GetInt("count"); ok {
    log.Printf("Count: %d", count)
}

// Check existence
if meta.Has("version") {
    version, _ := meta.GetString("version")
    // Use version
}
```

### Advanced Operations

```go
// Merge metadata
defaults := langfuse.Metadata{"env": "prod", "region": "us"}
custom := langfuse.Metadata{"region": "eu", "tier": "premium"}
merged := defaults.Clone().Merge(custom)
// Result: {"env": "prod", "region": "eu", "tier": "premium"}

// Filter specific keys
filtered := meta.Filter("user", "session")

// Get all keys
keys := meta.Keys()

// Check if empty
if meta.IsEmpty() {
    log.Println("No metadata")
}
```

### Available Methods

```go
meta.Set(key, value)          // Set a value
meta.Get(key)                 // Get any value
meta.GetString(key)           // Get string with type check
meta.GetInt(key)              // Get int with type check
meta.GetFloat(key)            // Get float64 with type check
meta.GetBool(key)             // Get bool with type check
meta.Has(key)                 // Check if key exists
meta.Delete(key)              // Remove a key
meta.Merge(other)             // Merge another metadata
meta.Clone()                  // Create a shallow copy
meta.Filter(keys...)          // Filter to specific keys
meta.Keys()                   // Get all keys
meta.Len()                    // Get number of entries
meta.IsEmpty()                // Check if empty
```

## Configured Sub-clients

Configure sub-clients with default options for repeated operations:

### Prompts with Default Options

```go
// Create a configured prompts client with default label
prompts := client.PromptsWithOptions(
    langfuse.WithPromptsLabel("production"),
)

// All operations use the default label
prompt, err := prompts.Get(ctx, "chat-template", nil)
```

### Sessions with Default Pagination

```go
// Configure sessions client with pagination defaults
sessions := client.SessionsWithOptions(
    langfuse.WithSessionsPage(1),
    langfuse.WithSessionsLimit(50),
)

// List sessions using configured pagination
result, err := sessions.List(ctx)
```

### Models with Filters

```go
// Configure models client with filters
models := client.ModelsWithOptions(
    langfuse.WithModelsPage(1),
    langfuse.WithModelsLimit(100),
)

result, err := models.List(ctx)
```

### Available WithOptions Methods

All major sub-clients support the WithOptions pattern:

```go
client.PromptsWithOptions(opts...)   // Configure prompts client
client.TracesWithOptions(opts...)    // Configure traces client
client.DatasetsWithOptions(opts...)  // Configure datasets client
client.ScoresWithOptions(opts...)    // Configure scores client
client.SessionsWithOptions(opts...)  // Configure sessions client
client.ModelsWithOptions(opts...)    // Configure models client
```

## Best Practices

1. **Always defer Shutdown**: Ensure pending events are flushed
   ```go
   defer client.Shutdown(context.Background())
   ```

2. **Use context for timeouts**: Pass appropriate contexts to all API calls
   ```go
   ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
   defer cancel()
   trace, err := client.NewTrace().Name("example").Create(ctx)
   ```

3. **Batch configuration**: Tune batch size and flush interval for your workload
   ```go
   langfuse.WithBatchSize(100),
   langfuse.WithFlushInterval(10*time.Second),
   ```

4. **Error handling**: Always check errors from Create(), Apply(), and End() methods

5. **Resource cleanup**: Always end observations with context
   ```go
   generation.End(ctx) // or EndWithOutput(ctx, output) or EndWithUsage(ctx, output, in, out)
   ```

## Examples

See the [examples](examples/) directory for complete working examples:

- [Basic Example](examples/basic/main.go): Simple trace with generation and scoring
- [Advanced Example](examples/advanced/main.go): Complex workflows with nested spans and evaluations

## Documentation

For more information about Langfuse and its features, visit:

- [Langfuse Documentation](https://langfuse.com/docs)
- [API Reference](https://api.reference.langfuse.com)
- [Go Package Documentation](https://pkg.go.dev/github.com/jdziat/langfuse-go)

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

This SDK is an unofficial Go client for [Langfuse](https://langfuse.com), the open-source LLM observability platform.
