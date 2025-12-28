//go:build v1_example

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jdziat/langfuse-go"
)

func main() {
	// Example 1: Simple v1 API Usage
	simpleExample()

	// Example 2: Advanced v1 API Usage
	advancedExample()

	// Example 3: Error Handling Patterns
	errorHandlingExample()
}

// simpleExample demonstrates the basic v1 API
func simpleExample() {
	fmt.Println("=== Simple v1 API Example ===")

	// Create client with simplified configuration - only essential parameters
	client := langfuse.NewClient("pk-lf-test-key", "sk-lf-test-key")
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	// Create trace with context-first approach using Trace()
	trace, err := client.Trace(ctx, "user-request")
	if err != nil {
		log.Fatalf("Failed to create trace: %v", err)
	}

	// Create nested span - much simpler using Span()!
	span, err := trace.Span(ctx, "processing")
	if err != nil {
		log.Fatalf("Failed to create span: %v", err)
	}

	// Create generation within span using Generation()
	generation, err := span.Generation(ctx, "gpt-4",
		langfuse.WithModel("gpt-4"),
		langfuse.WithGenerationInput([]map[string]string{
			{"role": "user", "content": "Hello!"},
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create generation: %v", err)
	}

	// End generation with output and token usage - consistent pattern using EndV1!
	gen, err := generation.EndV1(ctx,
		langfuse.WithEndOutput("Hello! How can I help you?"),
		langfuse.WithUsage(10, 12),
	)
	if err != nil {
		log.Fatalf("Failed to end generation: %v", err)
	}
	fmt.Printf("Generation ended: %s\n", gen.GenerationID())

	// End span using EndV1
	span, err = span.EndV1(ctx, langfuse.WithEndOutput("Processing complete"))
	if err != nil {
		log.Fatalf("Failed to end span: %v", err)
	}
	fmt.Printf("Span ended: %s\n", span.ID())

	// Update trace with final output using UpdateV1!
	trace, err = trace.UpdateV1(ctx, langfuse.WithUpdateOutput(map[string]interface{}{
		"response": "Request completed successfully",
	}))
	if err != nil {
		log.Fatalf("Failed to update trace: %v", err)
	}
	fmt.Printf("Trace updated: %s\n", trace.ID())
}

// advancedExample demonstrates more complex v1 API patterns
func advancedExample() {
	fmt.Println("\n=== Advanced v1 API Example ===")

	// Create client with options but still much simpler
	client := langfuse.NewClient("pk-lf-test-key", "sk-lf-test-key",
		langfuse.WithRegion(langfuse.RegionUS),
		langfuse.WithDebug(true),
	)
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	// Create trace with comprehensive options using Trace()
	trace, err := client.Trace(ctx, "complex-request",
		langfuse.WithUserID("user-123"),
		langfuse.WithSessionID("session-456"),
		langfuse.WithTags("api", "v2", "production"),
		langfuse.WithEnvironment("production"),
		langfuse.WithMetadata(map[string]interface{}{
			"endpoint":     "/api/v2/chat",
			"method":       "POST",
			"request_size": 156,
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create trace: %v", err)
	}

	// Create span with level and metadata using Span()
	span, err := trace.Span(ctx, "request-processing",
		langfuse.WithSpanLevel(langfuse.ObservationLevelDebug),
		langfuse.WithSpanMetadata(map[string]interface{}{
			"component": "chat-processor",
			"version":   "2.1.0",
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create span: %v", err)
	}

	// Create event for logging using Event()
	err = span.Event(ctx, "cache-check",
		langfuse.WithEventMetadata(map[string]interface{}{
			"cache_key": "conversation:123",
			"hit":       true,
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create event: %v", err)
	}

	// Create generation with all options using Generation()
	generation, err := span.Generation(ctx, "llm-response",
		langfuse.WithModel("gpt-4-turbo"),
		langfuse.WithModelParameters(map[string]interface{}{
			"temperature": 0.7,
			"max_tokens":  500,
		}),
		langfuse.WithGenerationInput([]map[string]string{
			{"role": "system", "content": "You are a helpful assistant"},
			{"role": "user", "content": "Tell me about Langfuse"},
		}),
		langfuse.WithPromptName("chat-template"),
		langfuse.WithPromptVersion(2),
	)
	if err != nil {
		log.Fatalf("Failed to create generation: %v", err)
	}

	// Add scores - simplified using Score()!
	err = generation.Score(ctx, "response_quality", 0.92,
		langfuse.WithScoreComment("Comprehensive and accurate"),
		langfuse.WithScoreDataType(langfuse.ScoreDataTypeNumeric),
	)
	if err != nil {
		log.Fatalf("Failed to add score: %v", err)
	}

	// Add another score using Score helper method
	err = generation.Score(ctx, "response_time", 0.85,
		langfuse.WithScoreComment("Average response time"),
	)
	if err != nil {
		log.Fatalf("Failed to add speed score: %v", err)
	}

	// End generation with comprehensive usage data using EndV1
	gen, err := generation.EndV1(ctx,
		langfuse.WithEndOutput("Langfuse is an open-source LLM observability platform..."),
		langfuse.WithEndDuration(800*time.Millisecond),
	)
	if err != nil {
		log.Fatalf("Failed to end generation: %v", err)
	}
	fmt.Printf("Generation ended: %s\n", gen.GenerationID())

	// End span using EndV1
	span, err = span.EndV1(ctx, langfuse.WithEndOutput("Chat processing complete"))
	if err != nil {
		log.Fatalf("Failed to end span: %v", err)
	}

	// Final trace update using UpdateV1
	trace, err = trace.UpdateV1(ctx,
		langfuse.WithUpdateOutput(map[string]interface{}{
			"status":      "success",
			"tokens_used": 22,
			"duration_ms": 800,
		}),
		langfuse.WithUpdateTags("completed"),
	)
	if err != nil {
		log.Fatalf("Failed to update trace: %v", err)
	}
	fmt.Printf("Final trace: %s\n", trace.ID())

	// Show client stats
	stats := client.Stats()
	fmt.Printf("Client stats: %+v\n", stats)
}

// errorHandlingExample demonstrates proper error handling patterns
func errorHandlingExample() {
	fmt.Println("\n=== Error Handling Example ===")

	// Use TryClient for optional observability
	client := langfuse.TryClient("pk-lf-invalid", "sk-lf-invalid")
	if client == nil {
		fmt.Println("Client creation failed (expected with invalid keys)")
		return
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = client.Shutdown(ctx)
	}()

	ctx := context.Background()

	// Handle trace creation errors
	trace, err := client.Trace(ctx, "error-test")
	if err != nil {
		if apiErr, ok := langfuse.AsAPIError(err); ok && apiErr.IsUnauthorized() {
			fmt.Printf("Authentication failed: %v\n", apiErr)
		} else {
			fmt.Printf("Other error creating trace: %v\n", err)
		}
		return
	}

	// Handle span creation errors
	span, err := trace.Span(ctx, "error-span")
	if err != nil {
		fmt.Printf("Failed to create span: %v\n", err)
		return
	}

	// Handle generation errors
	generation, err := span.Generation(ctx, "error-generation")
	if err != nil {
		fmt.Printf("Failed to create generation: %v\n", err)
		return
	}

	// Handle update errors
	trace, err = trace.UpdateV1(ctx, langfuse.WithUpdateOutput("test"))
	if err != nil {
		fmt.Printf("Failed to update trace: %v\n", err)
		return
	}

	// Handle score errors
	err = generation.Score(ctx, "test-score", 0.5)
	if err != nil {
		fmt.Printf("Failed to add score: %v\n", err)
		return
	}

	// Handle end errors using EndV1
	_, err = span.EndV1(ctx)
	if err != nil {
		fmt.Printf("Failed to end span: %v\n", err)
		return
	}

	fmt.Println("All operations completed successfully")
}

// demonstrateContextPropagation shows context usage patterns
func demonstrateContextPropagation() {
	fmt.Println("\n=== Context Propagation Example ===")

	client := langfuse.NewClient(os.Getenv("LANGFUSE_PUBLIC_KEY"), os.Getenv("LANGFUSE_SECRET_KEY"))
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	// Create trace and store in context
	trace, err := client.Trace(ctx, "context-demo")
	if err != nil {
		log.Fatalf("Failed to create trace: %v", err)
	}

	ctx = langfuse.ContextWithTrace(ctx, trace)

	// Use trace from context in different function
	processWithContext(ctx)

	// Use MustTraceFromContext for guaranteed presence
	trace = langfuse.MustTraceFromContext(ctx)
	fmt.Printf("Trace from context: %s\n", trace.ID())
}

func processWithContext(ctx context.Context) {
	trace, ok := langfuse.TraceFromContext(ctx)
	if !ok {
		log.Println("No trace in context")
		return
	}

	// Create observations using context trace with Span()
	span, err := trace.Span(ctx, "context-processing")
	if err != nil {
		log.Printf("Failed to create span: %v", err)
		return
	}

	// Nested function can also use context
	nestedProcessing(ctx, span)

	// End span using EndV1
	_, err = span.EndV1(ctx)
	if err != nil {
		log.Printf("Failed to end span: %v", err)
		return
	}
}

func nestedProcessing(ctx context.Context, parentSpan *langfuse.SpanContext) {
	// Can still access the original trace from context
	trace, ok := langfuse.TraceFromContext(ctx)
	if !ok {
		log.Println("No trace context")
		return
	}

	// Create nested span using Span()
	span, err := parentSpan.Span(ctx, "nested-work")
	if err != nil {
		log.Printf("Failed to create nested span: %v", err)
		return
	}

	// Create event using Event()
	err = span.Event(ctx, "nested-event",
		langfuse.WithEventMetadata(map[string]interface{}{
			"nested": true,
		}),
	)
	if err != nil {
		log.Printf("Failed to create event: %v", err)
		return
	}

	// End span using EndV1
	_, err = span.EndV1(ctx)
	if err != nil {
		log.Printf("Failed to end nested span: %v", err)
		return
	}

	fmt.Printf("Nested processing completed for trace: %s\n", trace.ID())
}
