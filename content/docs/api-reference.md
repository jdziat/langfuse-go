---
title: API Reference
weight: 5
---

Complete reference for the Langfuse Go SDK. For the full, always-current
reference see the [Go package documentation](https://pkg.go.dev/github.com/jdziat/langfuse-go).

## Client

### New

Creates a new Langfuse client. The public and secret keys are required
positional arguments; behavior is tuned with `ConfigOption` functions.

```text
func New(publicKey, secretKey string, opts ...ConfigOption) (*Client, error)
```

**Parameters:**
- `publicKey`: Langfuse public API key (`pk-lf-...`)
- `secretKey`: Langfuse secret API key (`sk-lf-...`)
- `opts`: Configuration options (see [Configuration Options](#configuration-options))

**Returns:**
- `*Client`: Initialized client
- `error`: Configuration error, if any

**Example:**
```go
client, err := langfuse.New("pk-lf-...", "sk-lf-...",
    langfuse.WithRegion(langfuse.RegionUS),
)
if err != nil {
    log.Fatal(err)
}
defer client.Shutdown(context.Background())
```

### Client Methods

#### NewTrace

Returns a fluent `TraceBuilder`. Chain setters and finish with `Create(ctx)`.

```text
func (c *Client) NewTrace() *TraceBuilder
```

**Example:**
```go
ctx := context.Background()

trace, err := client.NewTrace().
    Name("chat-completion").
    UserID("user-123").
    Create(ctx)
if err != nil {
    log.Fatal(err)
}
_ = trace
```

#### Trace

Creates a trace in a single call using `TraceOption` functions.

```text
func (c *Client) Trace(ctx context.Context, name string, opts ...TraceOption) (*TraceContext, error)
```

**Example:**
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

#### Flush

Flushes all pending events to Langfuse.

```text
func (c *Client) Flush(ctx context.Context) error
```

**Example:**
```go
ctx := context.Background()

if err := client.Flush(ctx); err != nil {
    log.Printf("Flush failed: %v", err)
}
```

#### Shutdown

Flushes pending events and closes the client.

```text
func (c *Client) Shutdown(ctx context.Context) error
```

**Example:**
```go
defer client.Shutdown(context.Background())
```

#### Sub-client Accessors

The client exposes sub-clients for the various Langfuse resources:

```text
func (c *Client) Traces() *TracesClient
func (c *Client) Observations() *ObservationsClient
func (c *Client) Scores() *ScoresClient
func (c *Client) Prompts() *PromptsClient
func (c *Client) Datasets() *DatasetsClient
func (c *Client) Sessions() *SessionsClient
func (c *Client) Models() *ModelsClient
```

## Configuration Options

Configuration options for `New()`. Each returns a `ConfigOption`.

### WithRegion

```text
func WithRegion(region Region) ConfigOption
```

Sets the Langfuse cloud region (`RegionEU` default, or `RegionUS`).

### WithBaseURL

```text
func WithBaseURL(baseURL string) ConfigOption
```

Sets a custom base URL (for self-hosted deployments).

**Environment Variable:** `LANGFUSE_BASE_URL`

### WithBatchSize

```text
func WithBatchSize(size int) ConfigOption
```

Sets the maximum number of events per batch. **Default:** 100

### WithBatchQueueSize

```text
func WithBatchQueueSize(size int) ConfigOption
```

Sets the capacity of the internal event queue.

### WithFlushInterval

```text
func WithFlushInterval(interval time.Duration) ConfigOption
```

Sets how often batched events are sent. **Default:** 10 seconds

### WithMaxRetries

```text
func WithMaxRetries(maxRetries int) ConfigOption
```

Sets the maximum number of retry attempts. **Default:** 3

### WithTimeout

```text
func WithTimeout(timeout time.Duration) ConfigOption
```

Sets the HTTP request timeout. **Default:** 10 seconds

### WithDebug

```text
func WithDebug(debug bool) ConfigOption
```

Enables debug logging.

### WithHTTPClient

```text
func WithHTTPClient(client *http.Client) ConfigOption
```

Uses a custom HTTP client.

## Trace

`client.NewTrace()` returns a `*TraceBuilder`. `Create(ctx)` returns a
`*TraceContext`.

### TraceBuilder

```text
func (b *TraceBuilder) ID(id string) *TraceBuilder
func (b *TraceBuilder) Name(name string) *TraceBuilder
func (b *TraceBuilder) UserID(userID string) *TraceBuilder
func (b *TraceBuilder) SessionID(sessionID string) *TraceBuilder
func (b *TraceBuilder) Input(input any) *TraceBuilder
func (b *TraceBuilder) Output(output any) *TraceBuilder
func (b *TraceBuilder) Metadata(metadata Metadata) *TraceBuilder
func (b *TraceBuilder) Tags(tags []string) *TraceBuilder
func (b *TraceBuilder) Release(release string) *TraceBuilder
func (b *TraceBuilder) Version(version string) *TraceBuilder
func (b *TraceBuilder) Environment(env string) *TraceBuilder
func (b *TraceBuilder) Public(public bool) *TraceBuilder
func (b *TraceBuilder) Create(ctx context.Context) (*TraceContext, error)
```

### TraceContext

```text
func (t *TraceContext) ID() string
func (t *TraceContext) Update() *TraceUpdateBuilder
func (t *TraceContext) NewSpan() *SpanBuilder
func (t *TraceContext) NewGeneration() *GenerationBuilder
func (t *TraceContext) NewEvent() *EventBuilder
func (t *TraceContext) NewScore() *ScoreBuilder
```

`TraceUpdateBuilder` mirrors the trace setters and is finished with
`Apply(ctx) error`.

**Example:**
```go
ctx := context.Background()

trace, err := client.NewTrace().Name("chat-completion").Create(ctx)
if err != nil {
    log.Fatal(err)
}

if err := trace.Update().
    Output(map[string]any{"result": "success"}).
    Apply(ctx); err != nil {
    log.Printf("update failed: %v", err)
}
```

## Generation

`trace.NewGeneration()` returns a `*GenerationBuilder`. `Create(ctx)` returns a
`*GenerationContext`.

### GenerationBuilder

```text
func (b *GenerationBuilder) Name(name string) *GenerationBuilder
func (b *GenerationBuilder) Model(model string) *GenerationBuilder
func (b *GenerationBuilder) ModelParameters(params Metadata) *GenerationBuilder
func (b *GenerationBuilder) Input(input any) *GenerationBuilder
func (b *GenerationBuilder) Output(output any) *GenerationBuilder
func (b *GenerationBuilder) Metadata(metadata Metadata) *GenerationBuilder
func (b *GenerationBuilder) Usage(usage *Usage) *GenerationBuilder
func (b *GenerationBuilder) UsageTokens(input, output int) *GenerationBuilder
func (b *GenerationBuilder) PromptName(name string) *GenerationBuilder
func (b *GenerationBuilder) PromptVersion(version int) *GenerationBuilder
func (b *GenerationBuilder) Create(ctx context.Context) (*GenerationContext, error)
```

### GenerationContext

```text
func (g *GenerationContext) ID() string
func (g *GenerationContext) Update() *GenerationUpdateBuilder
func (g *GenerationContext) End(ctx context.Context) error
func (g *GenerationContext) EndWithOutput(ctx context.Context, output any) error
func (g *GenerationContext) EndWithUsage(ctx context.Context, output any, inputTokens, outputTokens int) error
func (g *GenerationContext) NewScore() *ScoreBuilder
func (g *GenerationContext) NewSpan() *SpanBuilder
func (g *GenerationContext) NewGeneration() *GenerationBuilder
func (g *GenerationContext) NewEvent() *EventBuilder
```

**Example:**
```go
ctx := context.Background()

trace, err := client.NewTrace().Name("llm-call").Create(ctx)
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

// output, input tokens, output tokens
if err := generation.EndWithUsage(ctx, "Hi! How can I help?", 10, 8); err != nil {
    log.Printf("end failed: %v", err)
}
```

### Usage

Token usage information.

```text
type Usage struct {
    Input      int     // input tokens
    Output     int     // output tokens
    Total      int     // total tokens
    Unit       string  // usage unit
    InputCost  float64 // cost of input tokens
    OutputCost float64 // cost of output tokens
    TotalCost  float64 // total cost
}
```

## Span

`trace.NewSpan()` returns a `*SpanBuilder`. `Create(ctx)` returns a
`*SpanContext`. Spans can also be created from a `SpanContext` or
`GenerationContext` to nest observations.

### SpanBuilder

```text
func (b *SpanBuilder) ID(id string) *SpanBuilder
func (b *SpanBuilder) Name(name string) *SpanBuilder
func (b *SpanBuilder) Input(input any) *SpanBuilder
func (b *SpanBuilder) Output(output any) *SpanBuilder
func (b *SpanBuilder) Metadata(metadata Metadata) *SpanBuilder
func (b *SpanBuilder) Level(level ObservationLevel) *SpanBuilder
func (b *SpanBuilder) Create(ctx context.Context) (*SpanContext, error)
```

### SpanContext

```text
func (s *SpanContext) ID() string
func (s *SpanContext) Update() *SpanUpdateBuilder
func (s *SpanContext) End(ctx context.Context) error
func (s *SpanContext) EndWithOutput(ctx context.Context, output any) error
func (s *SpanContext) NewSpan() *SpanBuilder
func (s *SpanContext) NewGeneration() *GenerationBuilder
func (s *SpanContext) NewEvent() *EventBuilder
func (s *SpanContext) NewScore() *ScoreBuilder
```

**Example:**
```go
ctx := context.Background()

trace, err := client.NewTrace().Name("pipeline").Create(ctx)
if err != nil {
    log.Fatal(err)
}

span, err := trace.NewSpan().
    Name("retrieve-documents").
    Create(ctx)
if err != nil {
    log.Fatal(err)
}

if err := span.EndWithOutput(ctx, map[string]any{
    "results": []string{"doc1", "doc2"},
}); err != nil {
    log.Printf("end failed: %v", err)
}
```

## Event

`trace.NewEvent()` returns an `*EventBuilder`. Events are point-in-time and are
sent with `Create(ctx)`.

### EventBuilder

```text
func (b *EventBuilder) ID(id string) *EventBuilder
func (b *EventBuilder) Name(name string) *EventBuilder
func (b *EventBuilder) Input(input any) *EventBuilder
func (b *EventBuilder) Output(output any) *EventBuilder
func (b *EventBuilder) Metadata(metadata Metadata) *EventBuilder
func (b *EventBuilder) Level(level ObservationLevel) *EventBuilder
func (b *EventBuilder) Create(ctx context.Context) error
```

**Example:**
```go
ctx := context.Background()

trace, err := client.NewTrace().Name("pipeline").Create(ctx)
if err != nil {
    log.Fatal(err)
}

if err := trace.NewEvent().Name("cache-hit").Create(ctx); err != nil {
    log.Printf("event failed: %v", err)
}
```

## Score

`NewScore()` (on a trace, span, or generation context) returns a
`*ScoreBuilder`.

### ScoreBuilder

```text
func (b *ScoreBuilder) Name(name string) *ScoreBuilder
func (b *ScoreBuilder) Value(value any) *ScoreBuilder
func (b *ScoreBuilder) NumericValue(value float64) *ScoreBuilder
func (b *ScoreBuilder) CategoricalValue(value string) *ScoreBuilder
func (b *ScoreBuilder) BooleanValue(value bool) *ScoreBuilder
func (b *ScoreBuilder) Comment(comment string) *ScoreBuilder
func (b *ScoreBuilder) ObservationID(id string) *ScoreBuilder
func (b *ScoreBuilder) ConfigID(id string) *ScoreBuilder
func (b *ScoreBuilder) Metadata(metadata Metadata) *ScoreBuilder
func (b *ScoreBuilder) Create(ctx context.Context) error
```

**Example:**
```go
ctx := context.Background()

trace, err := client.NewTrace().Name("pipeline").Create(ctx)
if err != nil {
    log.Fatal(err)
}

if err := trace.NewScore().
    Name("user-rating").
    NumericValue(5.0).
    Create(ctx); err != nil {
    log.Printf("score failed: %v", err)
}
```

## Prompts

```text
func (c *PromptsClient) Get(ctx context.Context, name string, params *GetPromptParams) (*Prompt, error)
func (c *PromptsClient) GetLatest(ctx context.Context, name string) (*Prompt, error)
func (c *PromptsClient) GetByVersion(ctx context.Context, name string, version int) (*Prompt, error)
func (c *PromptsClient) GetByLabel(ctx context.Context, name string, label string) (*Prompt, error)
```

**Example:**
```go
ctx := context.Background()

prompt, err := client.Prompts().Get(ctx, "chat-template", nil)
if err != nil {
    log.Fatal(err)
}
_ = prompt
```

## Datasets

```text
func (c *DatasetsClient) Create(ctx context.Context, req *CreateDatasetRequest) (*Dataset, error)
func (c *DatasetsClient) CreateItem(ctx context.Context, req *CreateDatasetItemRequest) (*DatasetItem, error)
func (c *DatasetsClient) CreateRunItem(ctx context.Context, req *CreateDatasetRunItemRequest) (*DatasetRunItem, error)
```

**Example:**
```go
ctx := context.Background()

dataset, err := client.Datasets().Create(ctx, &langfuse.CreateDatasetRequest{
    Name:        "qa-dataset",
    Description: "Question-answering evaluation set",
})
if err != nil {
    log.Fatal(err)
}
_ = dataset
```

## Types and Constants

### Region

```text
RegionEU // EU cloud (default)
RegionUS // US cloud
```

### Observation Levels

```text
ObservationLevelDebug
ObservationLevelDefault
ObservationLevelWarning
ObservationLevelError
```

### Score Data Types

```text
ScoreDataTypeNumeric
ScoreDataTypeCategorical
ScoreDataTypeBoolean
```

### Metadata

`Metadata` is a `map[string]any` with helper methods such as `Set`, `Get`,
`Has`, `Merge`, `Clone`, and `Filter`. Create one with `NewMetadata()` or a
composite literal:

```go
meta := langfuse.Metadata{"environment": "production"}
_ = meta
```

## Complete Example

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
    client, err := langfuse.New("pk-lf-...", "sk-lf-...",
        langfuse.WithRegion(langfuse.RegionUS),
    )
    if err != nil {
        log.Fatal(err)
    }
    defer client.Shutdown(ctx)

    // Create trace
    trace, err := client.NewTrace().
        Name("chat-completion").
        UserID("user-123").
        Input(map[string]any{
            "messages": []map[string]string{
                {"role": "user", "content": "Hello!"},
            },
        }).
        Create(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Add generation
    generation, err := trace.NewGeneration().
        Name("openai-chat").
        Model("gpt-4").
        Input(map[string]any{
            "messages": []map[string]string{
                {"role": "user", "content": "Hello!"},
            },
        }).
        Create(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Finish with output and token usage
    if err := generation.EndWithUsage(ctx, "Hi! How can I help?", 10, 8); err != nil {
        log.Printf("end failed: %v", err)
    }

    // Add a score
    if err := generation.NewScore().
        Name("quality").
        NumericValue(0.95).
        Create(ctx); err != nil {
        log.Printf("score failed: %v", err)
    }

    // Flush
    if err := client.Flush(ctx); err != nil {
        log.Fatal(err)
    }
}
```

## Next Steps

- [Getting Started](../getting-started/) - Basic setup guide
- [Tracing Guide](../tracing/) - Complete tracing documentation
- [Evaluation Guide](../evaluation/) - Scoring and evaluation-ready tracing
- [Configuration](../configuration/) - Configuration reference
