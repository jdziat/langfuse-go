package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
)

func main() {
	// Create a new Langfuse client
	client, err := langfuse.New(
		os.Getenv("LANGFUSE_PUBLIC_KEY"),
		os.Getenv("LANGFUSE_SECRET_KEY"),
		langfuse.WithRegion(langfuse.RegionUS),
		langfuse.WithDebug(true),
	)
	if err != nil {
		log.Fatalf("Failed to create Langfuse client: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	// Check API health
	health, err := client.Health(ctx)
	if err != nil {
		log.Printf("Health check failed: %v", err)
	} else {
		fmt.Printf("API Status: %s\n", health.Status)
	}

	// Create a trace for an LLM interaction
	trace, err := client.NewTrace().
		Name("chat-completion").
		UserID("user-123").
		SessionID("session-456").
		Input(map[string]any{
			"message": "Hello, how can I help you today?",
		}).
		Tags([]string{"production", "chat"}).
		Metadata(map[string]any{
			"source": "web",
		}).
		Create(ctx)
	if err != nil {
		log.Fatalf("Failed to create trace: %v", err)
	}
	fmt.Printf("Created trace: %s\n", trace.ID())

	// Create a span for preprocessing
	preprocessSpan, err := trace.NewSpan().
		Name("preprocess-input").
		Input("Hello, how can I help you today?").
		Create(ctx)
	if err != nil {
		log.Fatalf("Failed to create span: %v", err)
	}

	// Simulate preprocessing work
	time.Sleep(50 * time.Millisecond)

	// End the span with output
	if err := preprocessSpan.EndWithOutput(ctx, "preprocessed: hello how can i help you today"); err != nil {
		log.Printf("Failed to end span: %v", err)
	}

	// Create a generation for the LLM call
	generation, err := trace.NewGeneration().
		Name("gpt-4-completion").
		Model("gpt-4").
		ModelParameters(map[string]any{
			"temperature": 0.7,
			"max_tokens":  150,
		}).
		Input([]map[string]string{
			{"role": "user", "content": "Hello, how can I help you today?"},
		}).
		Create(ctx)
	if err != nil {
		log.Fatalf("Failed to create generation: %v", err)
	}
	fmt.Printf("Created generation: %s\n", generation.GenerationID())

	// Simulate LLM call
	time.Sleep(100 * time.Millisecond)

	// End generation with output and usage
	if err := generation.EndWithUsage(
		ctx,
		"I'm an AI assistant. I can help you with various tasks like answering questions, writing, coding, and more. What would you like help with?",
		25, // input tokens
		42, // output tokens
	); err != nil {
		log.Printf("Failed to end generation: %v", err)
	}

	// Create a score for the generation
	if err := generation.NewScore().
		Name("quality").
		NumericValue(0.95).
		Comment("High quality response").
		Create(ctx); err != nil {
		log.Printf("Failed to create score: %v", err)
	}

	// Create an event for logging
	if err := trace.NewEvent().
		Name("response-sent").
		Level(langfuse.ObservationLevelDefault).
		Output("Response delivered to user").
		Create(ctx); err != nil {
		log.Printf("Failed to create event: %v", err)
	}

	// Update the trace with the final output
	if err := trace.Update().
		Output(map[string]any{
			"response": "I'm an AI assistant...",
			"tokens":   67,
		}).
		Apply(ctx); err != nil {
		log.Printf("Failed to update trace: %v", err)
	}

	// Flush all pending events
	if err := client.Flush(ctx); err != nil {
		log.Printf("Failed to flush: %v", err)
	}

	fmt.Println("Example completed successfully!")
}
