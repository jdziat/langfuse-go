package langfuse_test

import (
	"context"
	"errors"
	"fmt"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
)

// Example shows the canonical end-to-end flow: construct a client, start a
// trace, record an LLM generation, and finish the generation with output and
// token usage. This is the path new code should follow.
//
// Events are queued locally and flushed in the background, so the calls below
// succeed without a live network connection.
func Example() {
	client, err := langfuse.New("pk-lf-...", "sk-lf-...")
	if err != nil {
		fmt.Println("error creating client:", err)
		return
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	trace, err := client.NewTrace().
		Name("my-llm-call").
		UserID("user-123").
		Create(ctx)
	if err != nil {
		fmt.Println("error creating trace:", err)
		return
	}

	gen, err := trace.NewGeneration().
		Name("openai-completion").
		Model("gpt-4").
		Input("What is Go?").
		Create(ctx)
	if err != nil {
		fmt.Println("error creating generation:", err)
		return
	}

	// ... make your LLM call here ...

	if err := gen.EndWithUsage(ctx, "Go is a programming language.", 1500, 100); err != nil {
		fmt.Println("error ending generation:", err)
		return
	}

	fmt.Println("trace recorded")
	// Output: trace recorded
}

// ExampleClient_NewTrace demonstrates starting a trace with the canonical
// fluent builder. Chain setters, then call Create(ctx) to enqueue the trace.
func ExampleClient_NewTrace() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Shutdown(context.Background())

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
		fmt.Println("error:", err)
		return
	}

	fmt.Println("trace has ID:", trace.ID() != "")
	// Output: trace has ID: true
}

// ExampleTraceContext_NewGeneration records an LLM generation on a trace using
// the fluent builder, then finishes it with the generated output.
func ExampleTraceContext_NewGeneration() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, _ := client.NewTrace().Name("llm-call").Create(ctx)

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
		Create(ctx)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	_ = gen.EndWithOutput(ctx, map[string]string{
		"role":    "assistant",
		"content": "I'm doing well, thank you!",
	})

	fmt.Println("generation recorded")
	// Output: generation recorded
}

// ExampleTraceContext_NewSpan creates nested spans to model units of work
// within a trace.
func ExampleTraceContext_NewSpan() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, _ := client.NewTrace().Name("document-processing").Create(ctx)

	parent, _ := trace.NewSpan().
		Name("parse-document").
		Input(map[string]string{"file": "document.pdf"}).
		Create(ctx)

	child, _ := parent.NewSpan().
		Name("extract-text").
		Create(ctx)

	_ = child.EndWithOutput(ctx, "Extracted text content...")
	_ = parent.End(ctx)

	fmt.Println("spans recorded")
	// Output: spans recorded
}

// ExampleGenerationContext_EndWithUsage finishes a generation with both its
// output and the input/output token counts in a single call. This is the
// canonical way to close out an LLM generation.
func ExampleGenerationContext_EndWithUsage() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, _ := client.NewTrace().Name("summarize").Create(ctx)
	gen, _ := trace.NewGeneration().
		Name("gpt-4-summary").
		Model("gpt-4").
		Input("Summarize the following article...").
		Create(ctx)

	// inputTokens = 1500, outputTokens = 100
	if err := gen.EndWithUsage(ctx, "A concise summary.", 1500, 100); err != nil {
		fmt.Println("error:", err)
		return
	}

	fmt.Println("generation ended with usage")
	// Output: generation ended with usage
}

// ExampleGenerationContext_EndWith shows the option-based end variant, which
// returns an [langfuse.EndResult] carrying the measured Duration and any error.
func ExampleGenerationContext_EndWith() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, _ := client.NewTrace().Name("llm-call").Create(ctx)
	gen, _ := trace.NewGeneration().Name("completion").Model("gpt-4").Create(ctx)

	result := gen.EndWith(ctx,
		langfuse.WithUsage(1500, 100),
		langfuse.WithEndMetadata(langfuse.Metadata{"cached": false}),
	)

	fmt.Println("ended ok:", result.Ok())
	fmt.Println("measured duration:", result.Duration >= 0)
	// Output:
	// ended ok: true
	// measured duration: true
}

// ExampleGenerationContext_Score attaches scores to a generation. Scores can be
// numeric, boolean, or categorical, and the [langfuse.ScoreBuilder] offers full
// control for advanced cases.
func ExampleGenerationContext_Score() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, _ := client.NewTrace().Name("scored-generation").Create(ctx)
	gen, _ := trace.NewGeneration().Name("completion").Model("gpt-4").Create(ctx)

	_ = gen.Score(ctx, "accuracy", 0.95)
	_ = gen.ScoreCategory(ctx, "sentiment", "positive")
	_ = gen.ScoreBool(ctx, "factual", true)

	// The fluent builder offers more control, e.g. an attached comment.
	_ = gen.NewScore().
		Name("custom-score").
		NumericValue(0.87).
		Comment("High quality response").
		Create(ctx)

	fmt.Println("scores recorded")
	// Output: scores recorded
}

// ExampleClient_Trace shows the one-line convenience over the fluent builder:
// create a trace directly from a context, a name, and [langfuse.TraceOption]
// functions. The fluent [Client.NewTrace] builder remains the canonical path.
func ExampleClient_Trace() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	trace, err := client.Trace(ctx, "request-handler")
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	gen, _ := trace.Generation(ctx, "llm-call", langfuse.WithModel("gpt-4"))
	_ = gen.EndWithUsage(ctx, "response", 10, 20)

	fmt.Println("trace recorded")
	// Output: trace recorded
}

// ExampleContextWithTrace shows propagating a trace through a Go context so that
// downstream code can attach observations without passing the trace explicitly.
func ExampleContextWithTrace() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Shutdown(context.Background())

	trace, _ := client.NewTrace().Name("request-handler").Create(context.Background())
	ctx := langfuse.ContextWithTrace(context.Background(), trace)

	if traceFromCtx, ok := langfuse.TraceFromContext(ctx); ok {
		_, _ = traceFromCtx.NewSpan().Name("sub-operation").Create(ctx)
		fmt.Println("retrieved trace from context")
	}
	// Output: retrieved trace from context
}

// ExampleNew_withOptions configures the client with functional options.
func ExampleNew_withOptions() {
	client, err := langfuse.New("pk-lf-...", "sk-lf-...",
		langfuse.WithRegion(langfuse.RegionUS),
		langfuse.WithBatchSize(100),
		langfuse.WithFlushInterval(5*time.Second),
		langfuse.WithDebug(true),
	)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer client.Shutdown(context.Background())

	fmt.Println("client configured")
	// Output: client configured
}

// ExampleNewWithConfig constructs the client from a [langfuse.Config] struct
// instead of functional options.
func ExampleNewWithConfig() {
	client, err := langfuse.NewWithConfig(&langfuse.Config{
		PublicKey:     "pk-lf-...",
		SecretKey:     "sk-lf-...",
		Region:        langfuse.RegionUS,
		BatchSize:     100,
		FlushInterval: 10 * time.Second,
	})
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	defer client.Shutdown(context.Background())

	fmt.Println("client from config created")
	// Output: client from config created
}

// ExampleClient_Shutdown flushes pending events and stops the client. Always
// call Shutdown before your application exits so queued events are delivered.
func ExampleClient_Shutdown() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")

	_, _ = client.NewTrace().Name("trace-1").Create(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_ = client.Shutdown(ctx)

	fmt.Println("shutdown complete")
	// Output: shutdown complete
}

// ExampleAsAPIError shows how to detect and branch on Langfuse API errors.
func ExampleAsAPIError() {
	var err error = &langfuse.APIError{
		StatusCode: 429,
		Message:    "Rate limit exceeded",
	}

	if apiErr, ok := langfuse.AsAPIError(err); ok {
		switch {
		case apiErr.IsRateLimited():
			fmt.Printf("rate limited: %s\n", apiErr.Message)
		case apiErr.IsUnauthorized():
			fmt.Println("check your API keys")
		}
	}
	// Output: rate limited: Rate limit exceeded
}

// ExampleWrapError shows wrapping an underlying error with package context while
// preserving the original via errors.Is/errors.As.
func ExampleWrapError() {
	original := errors.New("connection refused")
	wrapped := langfuse.WrapError(original, "failed to connect to Langfuse")

	fmt.Println(wrapped)
	// Output: langfuse: failed to connect to Langfuse: connection refused
}
