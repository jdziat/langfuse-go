# API Reference

Complete reference for the Langfuse Go SDK.

## Client

### New

Creates a new Langfuse client.

```go
func New(opts ...Option) (*Client, error)
```

**Parameters:**
- `opts`: Configuration options (see [Configuration Options](#configuration-options))

**Returns:**
- `*Client`: Initialized client
- `error`: Configuration error, if any

**Example:**
```go
client, err := langfuse.New(
    langfuse.WithPublicKey("pk_lf_..."),
    langfuse.WithSecretKey("sk_lf_..."),
)
```

### Client Methods

#### Trace

Creates a new trace.

```go
func (c *Client) Trace(params TraceParams) *Trace
```

**Parameters:**
- `params`: Trace configuration (see [TraceParams](#traceparams))

**Returns:**
- `*Trace`: New trace object

**Example:**
```go
trace := client.Trace(langfuse.TraceParams{
    Name: "chat-completion",
    UserID: "user-123",
})
```

#### Flush

Flushes all pending events to Langfuse.

```go
func (c *Client) Flush(ctx context.Context) error
```

**Parameters:**
- `ctx`: Context for timeout/cancellation

**Returns:**
- `error`: Flush error, if any

**Example:**
```go
if err := client.Flush(ctx); err != nil {
    log.Printf("Flush failed: %v", err)
}
```

#### Shutdown

Flushes pending events and closes the client.

```go
func (c *Client) Shutdown(ctx context.Context) error
```

**Parameters:**
- `ctx`: Context for timeout/cancellation

**Returns:**
- `error`: Shutdown error, if any

**Example:**
```go
defer client.Shutdown(context.Background())
```

#### Evaluator

Creates a new LLM-as-a-Judge evaluator.

```go
func (c *Client) Evaluator(params EvaluatorParams) *Evaluator
```

**Parameters:**
- `params`: Evaluator configuration (see [EvaluatorParams](#evaluatorparams))

**Returns:**
- `*Evaluator`: New evaluator object

**Example:**
```go
evaluator := client.Evaluator(langfuse.EvaluatorParams{
    Name: "quality",
    Model: "gpt-4",
    Prompt: "Rate quality from 0.0 to 1.0...",
})
```

## Configuration Options

Configuration options for `New()`.

### WithPublicKey

```go
func WithPublicKey(key string) Option
```

Sets the Langfuse public API key.

**Environment Variable:** `LANGFUSE_PUBLIC_KEY`

### WithSecretKey

```go
func WithSecretKey(key string) Option
```

Sets the Langfuse secret API key.

**Environment Variable:** `LANGFUSE_SECRET_KEY`

### WithBaseURL

```go
func WithBaseURL(url string) Option
```

Sets a custom base URL (for self-hosted deployments).

**Environment Variable:** `LANGFUSE_BASE_URL`

### WithRegion

```go
func WithRegion(region Region) Option
```

Sets the Langfuse cloud region.

**Options:**
- `RegionUS`: US cloud (default)
- `RegionEU`: EU cloud

### WithBatchSize

```go
func WithBatchSize(size int) Option
```

Sets the maximum number of events per batch.

**Default:** 100

**Environment Variable:** `LANGFUSE_BATCH_SIZE`

### WithFlushInterval

```go
func WithFlushInterval(interval time.Duration) Option
```

Sets how often batched events are sent.

**Default:** 10 seconds

**Environment Variable:** `LANGFUSE_FLUSH_INTERVAL`

### WithMaxRetries

```go
func WithMaxRetries(retries int) Option
```

Sets the maximum number of retry attempts.

**Default:** 3

**Environment Variable:** `LANGFUSE_MAX_RETRIES`

### WithTimeout

```go
func WithTimeout(timeout time.Duration) Option
```

Sets the HTTP request timeout.

**Default:** 10 seconds

**Environment Variable:** `LANGFUSE_TIMEOUT`

### WithHTTPClient

```go
func WithHTTPClient(client *http.Client) Option
```

Uses a custom HTTP client.

## Trace

### Trace Methods

#### Update

Updates the trace with new parameters.

```go
func (t *Trace) Update(params TraceParams)
```

**Parameters:**
- `params`: Updated trace parameters

**Example:**
```go
trace.Update(langfuse.TraceParams{
    Output: map[string]any{"result": "success"},
})
```

#### Generation

Creates a generation within the trace.

```go
func (t *Trace) Generation(params GenerationParams) *Generation
```

**Parameters:**
- `params`: Generation configuration

**Returns:**
- `*Generation`: New generation object

**Example:**
```go
generation := trace.Generation(langfuse.GenerationParams{
    Name: "openai-chat",
    Model: "gpt-4",
})
```

#### Span

Creates a span within the trace.

```go
func (t *Trace) Span(params SpanParams) *Span
```

**Parameters:**
- `params`: Span configuration

**Returns:**
- `*Span`: New span object

**Example:**
```go
span := trace.Span(langfuse.SpanParams{
    Name: "retrieve-documents",
})
```

#### Event

Creates an event within the trace.

```go
func (t *Trace) Event(params EventParams) *Event
```

**Parameters:**
- `params`: Event configuration

**Returns:**
- `*Event`: New event object

**Example:**
```go
event := trace.Event(langfuse.EventParams{
    Name: "cache-hit",
})
```

#### Score

Adds a score to the trace.

```go
func (t *Trace) Score(params ScoreParams)
```

**Parameters:**
- `params`: Score configuration

**Example:**
```go
trace.Score(langfuse.ScoreParams{
    Name: "user-rating",
    Value: 5.0,
})
```

### TraceParams

Configuration for traces.

```go
type TraceParams struct {
    Name      string         // Trace name
    ID        string         // Custom trace ID (optional)
    UserID    string         // User identifier (optional)
    SessionID string         // Session identifier (optional)
    Input     map[string]any // Input data (optional)
    Output    map[string]any // Output data (optional)
    Metadata  map[string]any // Additional metadata (optional)
    Tags      []string       // Tags for filtering (optional)
    Public    bool           // Make trace public (optional)
}
```

## Generation

### Generation Methods

#### Update

Updates the generation with new parameters.

```go
func (g *Generation) Update(params GenerationParams)
```

**Parameters:**
- `params`: Updated generation parameters

**Example:**
```go
generation.Update(langfuse.GenerationParams{
    Output: responseData,
    Usage: &langfuse.Usage{
        PromptTokens: 10,
        CompletionTokens: 20,
        TotalTokens: 30,
    },
})
```

#### Score

Adds a score to the generation.

```go
func (g *Generation) Score(params ScoreParams)
```

**Parameters:**
- `params`: Score configuration

**Example:**
```go
generation.Score(langfuse.ScoreParams{
    Name: "quality",
    Value: 0.95,
})
```

### GenerationParams

Configuration for generations.

```go
type GenerationParams struct {
    Name            string         // Generation name
    Model           string         // Model name (e.g., "gpt-4")
    ID              string         // Custom generation ID (optional)
    Input           map[string]any // Input/prompt (optional)
    Output          map[string]any // Output/completion (optional)
    Metadata        map[string]any // Additional metadata (optional)
    ModelParameters map[string]any // Model parameters (optional)
    Usage           *Usage         // Token usage (optional)
    PromptName      string         // Associated prompt name (optional)
    PromptVersion   int            // Prompt version (optional)
}
```

### Usage

Token usage information.

```go
type Usage struct {
    PromptTokens     int // Tokens in prompt
    CompletionTokens int // Tokens in completion
    TotalTokens      int // Total tokens
}
```

## Span

### Span Methods

#### Update

Updates the span with new parameters.

```go
func (s *Span) Update(params SpanParams)
```

**Parameters:**
- `params`: Updated span parameters

**Example:**
```go
span.Update(langfuse.SpanParams{
    Output: map[string]any{"results": []string{"doc1", "doc2"}},
})
```

#### Generation

Creates a generation within the span.

```go
func (s *Span) Generation(params GenerationParams) *Generation
```

**Parameters:**
- `params`: Generation configuration

**Returns:**
- `*Generation`: New generation object

#### Span

Creates a nested span within this span.

```go
func (s *Span) Span(params SpanParams) *Span
```

**Parameters:**
- `params`: Span configuration

**Returns:**
- `*Span`: New span object

#### Event

Creates an event within the span.

```go
func (s *Span) Event(params EventParams) *Event
```

**Parameters:**
- `params`: Event configuration

**Returns:**
- `*Event`: New event object

#### Score

Adds a score to the span.

```go
func (s *Span) Score(params ScoreParams)
```

**Parameters:**
- `params`: Score configuration

### SpanParams

Configuration for spans.

```go
type SpanParams struct {
    Name     string         // Span name
    ID       string         // Custom span ID (optional)
    Input    map[string]any // Input data (optional)
    Output   map[string]any // Output data (optional)
    Metadata map[string]any // Additional metadata (optional)
}
```

## Event

### EventParams

Configuration for events.

```go
type EventParams struct {
    Name     string         // Event name
    ID       string         // Custom event ID (optional)
    Input    map[string]any // Event data (optional)
    Metadata map[string]any // Additional metadata (optional)
}
```

## Score

### ScoreParams

Configuration for scores.

```go
type ScoreParams struct {
    Name    string  // Score name
    Value   float64 // Score value
    Comment string  // Optional comment (optional)
}
```

## Evaluator

### Evaluator Methods

#### Evaluate

Evaluates an observation using LLM-as-a-Judge.

```go
func (e *Evaluator) Evaluate(ctx context.Context, params EvaluateParams) (*Score, error)
```

**Parameters:**
- `ctx`: Context for timeout/cancellation
- `params`: Evaluation parameters

**Returns:**
- `*Score`: Evaluation score
- `error`: Evaluation error, if any

**Example:**
```go
score, err := evaluator.Evaluate(ctx, langfuse.EvaluateParams{
    ObservationID: generation.ID,
    Input: inputData,
    Output: outputData,
})
```

### EvaluatorParams

Configuration for evaluators.

```go
type EvaluatorParams struct {
    Name   string // Evaluator name
    Model  string // Model to use (e.g., "gpt-4")
    Prompt string // Evaluation prompt template
}
```

### EvaluateParams

Parameters for evaluation.

```go
type EvaluateParams struct {
    ObservationID string         // ID of observation to evaluate
    Input         map[string]any // Input data
    Output        map[string]any // Output data
}
```

## Types

### Region

Langfuse cloud regions.

```go
const (
    RegionUS Region = "US" // US cloud (default)
    RegionEU Region = "EU" // EU cloud
)
```

### Error Types

Common error types returned by the SDK.

```go
var (
    ErrInvalidConfig    = errors.New("invalid configuration")
    ErrInvalidAPIKey    = errors.New("invalid API key")
    ErrFlushTimeout     = errors.New("flush timeout")
    ErrShutdownTimeout  = errors.New("shutdown timeout")
    ErrEvaluationFailed = errors.New("evaluation failed")
)
```

## Examples

### Complete Example

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
        langfuse.WithPublicKey("pk_lf_..."),
        langfuse.WithSecretKey("sk_lf_..."),
        langfuse.WithRegion(langfuse.RegionUS),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(context.Background())

    // Create trace
    trace := client.Trace(langfuse.TraceParams{
        Name: "chat-completion",
        UserID: "user-123",
        Input: map[string]any{
            "messages": []map[string]string{
                {"role": "user", "content": "Hello!"},
            },
        },
    })

    // Add generation
    generation := trace.Generation(langfuse.GenerationParams{
        Name: "openai-chat",
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
            "response": "Hi! How can I help?",
        },
        Usage: &langfuse.Usage{
            PromptTokens: 10,
            CompletionTokens: 8,
            TotalTokens: 18,
        },
    })

    // Add score
    generation.Score(langfuse.ScoreParams{
        Name: "quality",
        Value: 0.95,
    })

    // Flush
    if err := client.Flush(context.Background()); err != nil {
        log.Fatal(err)
    }
}
```

## Next Steps

- [Getting Started](getting-started.md) - Basic setup guide
- [Tracing Guide](tracing.md) - Complete tracing documentation
- [Evaluation Guide](evaluation.md) - LLM-as-a-Judge workflows
- [Configuration](configuration.md) - Configuration reference
