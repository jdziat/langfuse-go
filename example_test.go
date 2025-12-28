package langfuse_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jdziat/langfuse-go"
)

// This example demonstrates creating a new Langfuse client with basic configuration.
func ExampleNew() {
	client, err := langfuse.New("pk-lf-...", "sk-lf-...")
	if err != nil {
		fmt.Println("Error creating client:", err)
		return
	}
	defer client.Close(context.Background())

	fmt.Println("Client created successfully")
	// Output: Client created successfully
}

// This example shows how to configure the client with custom options.
func ExampleNew_withOptions() {
	client, err := langfuse.New("pk-lf-...", "sk-lf-...",
		langfuse.WithBaseURL("https://custom.langfuse.com"),
		langfuse.WithBatchSize(50),
		langfuse.WithFlushInterval(5*time.Second),
		langfuse.WithRegion(langfuse.RegionUS),
	)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer client.Close(context.Background())

	fmt.Println("Client with custom options created")
	// Output: Client with custom options created
}

// This example demonstrates creating a trace with the fluent builder API.
func ExampleClient_NewTrace() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Close(context.Background())

	// Create a trace using the fluent builder
	trace, err := client.NewTrace().
		Name("chat-completion").
		UserID("user-123").
		SessionID("session-456").
		Metadata(langfuse.Metadata{
			"model":   "gpt-4",
			"version": "1.0",
		}).
		Tags([]string{"production", "chat"}).
		Create(context.Background())

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	fmt.Println("Trace ID:", trace.ID() != "")
	// Output: Trace ID: true
}

// This example shows how to create a generation within a trace.
func ExampleTraceContext_Generation() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Close(context.Background())

	trace, _ := client.NewTrace().Name("llm-call").Create(context.Background())

	// Create a generation for an LLM call
	gen, err := trace.NewGeneration().
		Name("gpt-4-completion").
		Model("gpt-4").
		ModelParameters(langfuse.Metadata{
			"temperature": 0.7,
			"max_tokens":  1000,
		}).
		Input([]map[string]string{
			{"role": "user", "content": "Hello, how are you?"},
		}).
		Create(context.Background())

	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	// Later, update the generation with the output
	_ = gen.Update().
		Output(map[string]string{
			"role":    "assistant",
			"content": "I'm doing well, thank you!",
		}).
		UsageTokens(10, 20).
		EndTime(time.Now()).
		Apply(context.Background())

	fmt.Println("Generation created and updated")
	// Output: Generation created and updated
}

// This example shows how to create nested spans within a trace.
func ExampleTraceContext_Span() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Close(context.Background())

	trace, _ := client.NewTrace().Name("document-processing").Create(context.Background())

	// Create a parent span
	parentSpan, _ := trace.NewSpan().
		Name("parse-document").
		Input(map[string]string{"file": "document.pdf"}).
		Create(context.Background())

	// Create a child span
	childSpan, _ := parentSpan.NewSpan().
		Name("extract-text").
		Create(context.Background())

	// Complete the child span
	_ = childSpan.EndWithOutput(context.Background(), "Extracted text content...")

	// Complete the parent span
	_ = parentSpan.End(context.Background())

	fmt.Println("Spans created successfully")
	// Output: Spans created successfully
}

// This example demonstrates adding scores to generations.
func ExampleGenerationContext_Score() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Close(context.Background())

	trace, _ := client.NewTrace().Name("scored-generation").Create(context.Background())
	gen, _ := trace.NewGeneration().Name("completion").Create(context.Background())

	// Add a numeric score
	_ = gen.ScoreNumeric(context.Background(), "accuracy", 0.95)

	// Add a categorical score
	_ = gen.ScoreCategorical(context.Background(), "sentiment", "positive")

	// Add a boolean score
	_ = gen.ScoreBoolean(context.Background(), "factual", true)

	// Or use the full builder for more control
	_ = gen.NewScore().
		Name("custom-score").
		NumericValue(0.87).
		Comment("High quality response").
		Create(context.Background())

	fmt.Println("Scores added")
	// Output: Scores added
}

// This example shows how to use the Clone method for creating multiple similar traces.
func ExampleTraceBuilder_Clone() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Close(context.Background())

	// Create a template builder
	template := client.NewTrace().
		Metadata(langfuse.Metadata{"environment": "production"}).
		Tags([]string{"batch-process"})

	// Clone and customize for each trace
	trace1, _ := template.Clone().Name("process-batch-1").Create(context.Background())
	trace2, _ := template.Clone().Name("process-batch-2").Create(context.Background())

	fmt.Println("Traces created:", trace1.ID() != trace2.ID())
	// Output: Traces created: true
}

// This example demonstrates handling API errors.
func ExampleAsAPIError() {
	// Simulated API error
	var err error = &langfuse.APIError{
		StatusCode: 429,
		Message:    "Rate limit exceeded",
	}

	// Check if it's an API error and handle accordingly
	if apiErr, ok := langfuse.AsAPIError(err); ok {
		if apiErr.IsRateLimited() {
			fmt.Printf("Rate limited: %s\n", apiErr.Message)
		} else if apiErr.IsUnauthorized() {
			fmt.Println("Check your API keys")
		}
	}
	// Output: Rate limited: Rate limit exceeded
}

// This example shows how to check error codes for categorization.
func ExampleErrorCodeOf() {
	// Different error types return appropriate codes
	configErr := langfuse.ErrMissingPublicKey
	code := langfuse.ErrorCodeOf(configErr)
	fmt.Println("Config error code:", code)

	apiErr := &langfuse.APIError{StatusCode: 401}
	code = langfuse.ErrorCodeOf(apiErr)
	fmt.Println("Auth error code:", code)

	// Output:
	// Config error code: CONFIG
	// Auth error code: AUTH
}

// This example demonstrates wrapping errors with context.
func ExampleWrapError() {
	originalErr := errors.New("connection refused")
	wrapped := langfuse.WrapError(originalErr, "failed to connect to Langfuse")

	fmt.Println(wrapped)
	// Output: langfuse: failed to connect to Langfuse: connection refused
}

// This example shows how to validate UUIDs.
func ExampleIsValidUUID() {
	// Standard UUID format (36 chars with hyphens)
	fmt.Println("Standard UUID:", langfuse.IsValidUUID("550e8400-e29b-41d4-a716-446655440000"))

	// Compact UUID format (32 chars without hyphens)
	fmt.Println("Compact UUID:", langfuse.IsValidUUID("550e8400e29b41d4a716446655440000"))

	// Invalid format
	fmt.Println("Invalid:", langfuse.IsValidUUID("not-a-uuid"))

	// Output:
	// Standard UUID: true
	// Compact UUID: true
	// Invalid: false
}

// This example demonstrates masking credentials for logging.
func ExampleMaskCredential() {
	// API keys are masked but preserve their prefix for identification
	publicKey := "pk-lf-1234567890abcdef"
	secretKey := "sk-lf-secretkey12345678"

	fmt.Println("Public:", langfuse.MaskCredential(publicKey))
	fmt.Println("Secret:", langfuse.MaskCredential(secretKey))

	// Output:
	// Public: pk-lf-************cdef
	// Secret: sk-lf-*************5678
}

// This example shows proper client shutdown with flush.
// Note: In production, Flush and Close would succeed with valid credentials.
func ExampleClient_Close() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")

	// Create some traces (events are queued locally)
	_, _ = client.NewTrace().Name("trace-1").Create(context.Background())
	_, _ = client.NewTrace().Name("trace-2").Create(context.Background())

	// Close flushes pending events and shuts down the client.
	// The ctx timeout controls how long to wait for pending events.
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Close handles the shutdown - errors during flush are logged, not returned
	_ = client.Close(ctx)

	fmt.Println("Client shutdown initiated")
	// Output: Client shutdown initiated
}

// This example shows using trace context through Go context.
func ExampleContextWithTrace() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Close(context.Background())

	// Create a trace and store it in context
	trace, _ := client.NewTrace().Name("request-handler").Create(context.Background())
	ctx := langfuse.ContextWithTrace(context.Background(), trace)

	// Later, retrieve the trace from context
	if traceFromCtx, ok := langfuse.TraceFromContext(ctx); ok {
		_, _ = traceFromCtx.NewSpan().Name("sub-operation").Create(ctx)
		fmt.Println("Retrieved trace from context")
	}
	// Output: Retrieved trace from context
}

// This example demonstrates using the Config struct directly.
func ExampleNewWithConfig() {
	cfg := &langfuse.Config{
		PublicKey:     "pk-lf-...",
		SecretKey:     "sk-lf-...",
		Region:        langfuse.RegionUS,
		BatchSize:     100,
		FlushInterval: 10 * time.Second,
		Debug:         true,
	}

	client, err := langfuse.NewWithConfig(cfg)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer client.Close(context.Background())

	fmt.Println("Client from config created")
	// Output: Client from config created
}
