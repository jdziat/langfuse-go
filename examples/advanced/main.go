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
	ctx := context.Background()

	// Create client with custom configuration
	client, err := langfuse.New(
		os.Getenv("LANGFUSE_PUBLIC_KEY"),
		os.Getenv("LANGFUSE_SECRET_KEY"),
		langfuse.WithRegion(langfuse.RegionUS),
		langfuse.WithBatchSize(50),
		langfuse.WithFlushInterval(10*time.Second),
		langfuse.WithTimeout(60*time.Second),
		langfuse.WithMaxRetries(5),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Shutdown(ctx)

	// Example 1: Working with Prompts
	fmt.Println("\n=== Prompts Example ===")
	promptsExample(ctx, client)

	// Example 2: Working with Datasets
	fmt.Println("\n=== Datasets Example ===")
	datasetsExample(ctx, client)

	// Example 3: Complex trace with nested spans
	fmt.Println("\n=== Complex Trace Example ===")
	complexTraceExample(ctx, client)

	// Example 4: Query traces and observations
	fmt.Println("\n=== Query Example ===")
	queryExample(ctx, client)

	// Flush all pending events
	if err := client.Flush(ctx); err != nil {
		log.Printf("Failed to flush: %v", err)
	}

	fmt.Println("\nAll examples completed!")
}

func promptsExample(ctx context.Context, client *langfuse.Client) {
	// Create a text prompt
	prompt, err := client.Prompts().CreateTextPrompt(
		ctx,
		"greeting-prompt",
		"Hello {{name}}! Welcome to {{service}}. How can I assist you today?",
		[]string{"production"},
	)
	if err != nil {
		log.Printf("Failed to create prompt: %v", err)
		return
	}
	fmt.Printf("Created prompt: %s (version %d)\n", prompt.Name, prompt.Version)

	// Get the latest prompt
	latestPrompt, err := client.Prompts().GetLatest(ctx, "greeting-prompt")
	if err != nil {
		log.Printf("Failed to get prompt: %v", err)
		return
	}

	// Compile the prompt with variables
	compiled, err := latestPrompt.Compile(map[string]string{
		"name":    "John",
		"service": "AI Assistant",
	})
	if err != nil {
		log.Printf("Failed to compile prompt: %v", err)
		return
	}
	fmt.Printf("Compiled prompt: %s\n", compiled)

	// Create a chat prompt
	chatPrompt, err := client.Prompts().CreateChatPrompt(
		ctx,
		"chat-prompt",
		[]langfuse.ChatMessage{
			{Role: "system", Content: "You are a helpful assistant for {{company}}."},
			{Role: "user", Content: "{{user_message}}"},
		},
		[]string{"development"},
	)
	if err != nil {
		log.Printf("Failed to create chat prompt: %v", err)
		return
	}
	fmt.Printf("Created chat prompt: %s\n", chatPrompt.Name)
}

func datasetsExample(ctx context.Context, client *langfuse.Client) {
	// Create a dataset
	dataset, err := client.Datasets().Create(ctx, &langfuse.CreateDatasetRequest{
		Name:        "qa-evaluation-set",
		Description: "Question-answering evaluation dataset",
		Metadata: map[string]interface{}{
			"category": "qa",
			"version":  1,
		},
	})
	if err != nil {
		log.Printf("Failed to create dataset: %v", err)
		return
	}
	fmt.Printf("Created dataset: %s\n", dataset.Name)

	// Add items to the dataset
	item, err := client.Datasets().CreateItem(ctx, &langfuse.CreateDatasetItemRequest{
		DatasetName: "qa-evaluation-set",
		Input: map[string]interface{}{
			"question": "What is the capital of France?",
		},
		ExpectedOutput: map[string]interface{}{
			"answer": "Paris",
		},
		Metadata: map[string]interface{}{
			"difficulty": "easy",
			"category":   "geography",
		},
	})
	if err != nil {
		log.Printf("Failed to create dataset item: %v", err)
		return
	}
	fmt.Printf("Created dataset item: %s\n", item.ID)

	// List datasets
	datasets, err := client.Datasets().List(ctx, nil)
	if err != nil {
		log.Printf("Failed to list datasets: %v", err)
		return
	}
	fmt.Printf("Found %d datasets\n", len(datasets.Data))
}

func complexTraceExample(ctx context.Context, client *langfuse.Client) {
	// Create a trace for a RAG pipeline
	trace, err := client.NewTrace().
		Name("rag-pipeline").
		UserID("user-456").
		SessionID("session-789").
		Input(map[string]interface{}{
			"query": "What are the benefits of exercise?",
		}).
		Tags([]string{"rag", "production"}).
		Environment("production").
		Create()
	if err != nil {
		log.Fatalf("Failed to create trace: %v", err)
	}
	fmt.Printf("Created RAG trace: %s\n", trace.ID())

	// Span 1: Query embedding
	embeddingSpan, err := trace.Span().
		Name("query-embedding").
		Input("What are the benefits of exercise?").
		Create()
	if err != nil {
		log.Printf("Failed to create embedding span: %v", err)
		return
	}

	time.Sleep(30 * time.Millisecond) // Simulate embedding generation
	embeddingSpan.EndWithOutput([]float64{0.1, 0.2, 0.3, 0.4, 0.5})

	// Span 2: Vector search
	searchSpan, err := trace.Span().
		Name("vector-search").
		Input(map[string]interface{}{
			"embedding": []float64{0.1, 0.2, 0.3, 0.4, 0.5},
			"top_k":     5,
		}).
		Create()
	if err != nil {
		log.Printf("Failed to create search span: %v", err)
		return
	}

	time.Sleep(50 * time.Millisecond) // Simulate vector search
	searchSpan.EndWithOutput(map[string]interface{}{
		"results": []string{
			"Exercise improves cardiovascular health...",
			"Regular physical activity can reduce stress...",
			"Studies show exercise boosts mental health...",
		},
	})

	// Span 3: Context assembly
	contextSpan, err := trace.Span().
		Name("context-assembly").
		Create()
	if err != nil {
		log.Printf("Failed to create context span: %v", err)
		return
	}

	time.Sleep(10 * time.Millisecond)
	contextSpan.End()

	// Generation: LLM call with retrieved context
	generation, err := trace.Generation().
		Name("llm-synthesis").
		Model("gpt-4-turbo").
		ModelParameters(map[string]interface{}{
			"temperature": 0.3,
			"max_tokens":  500,
		}).
		Input(map[string]interface{}{
			"system": "You are a helpful health assistant. Answer based on the provided context.",
			"context": []string{
				"Exercise improves cardiovascular health...",
				"Regular physical activity can reduce stress...",
			},
			"question": "What are the benefits of exercise?",
		}).
		PromptName("rag-synthesis").
		PromptVersion(1).
		Create()
	if err != nil {
		log.Printf("Failed to create generation: %v", err)
		return
	}

	time.Sleep(200 * time.Millisecond) // Simulate LLM call

	generation.EndWithUsage(
		"Based on the research, exercise offers several key benefits: 1) Improved cardiovascular health, 2) Reduced stress levels, 3) Better mental health and mood. Regular physical activity is recommended for overall wellness.",
		150, // input tokens
		75,  // output tokens
	)

	// Add evaluation scores
	generation.Score().
		Name("relevance").
		NumericValue(0.92).
		Comment("Response highly relevant to query").
		Create()

	generation.Score().
		Name("groundedness").
		NumericValue(0.88).
		Comment("Well grounded in context").
		Create()

	// Event: Response delivered
	trace.Event().
		Name("response-complete").
		Level(langfuse.ObservationLevelDefault).
		Metadata(map[string]interface{}{
			"latency_ms": 290,
		}).
		Create()

	// Update trace with final output
	trace.Update().
		Output(map[string]interface{}{
			"response":     "Based on the research, exercise offers several key benefits...",
			"total_tokens": 225,
			"latency_ms":   290,
		}).
		Apply()

	fmt.Println("RAG pipeline trace completed")
}

func queryExample(ctx context.Context, client *langfuse.Client) {
	// List recent traces
	traces, err := client.Traces().List(ctx, &langfuse.TracesListParams{
		PaginationParams: langfuse.PaginationParams{
			Limit: 10,
		},
	})
	if err != nil {
		log.Printf("Failed to list traces: %v", err)
		return
	}
	fmt.Printf("Found %d traces\n", len(traces.Data))

	// List generations
	generations, err := client.Observations().ListGenerations(ctx, &langfuse.ObservationsListParams{
		PaginationParams: langfuse.PaginationParams{
			Limit: 5,
		},
	})
	if err != nil {
		log.Printf("Failed to list generations: %v", err)
		return
	}
	fmt.Printf("Found %d generations\n", len(generations.Data))

	// List scores
	scores, err := client.Scores().List(ctx, &langfuse.ScoresListParams{
		PaginationParams: langfuse.PaginationParams{
			Limit: 10,
		},
	})
	if err != nil {
		log.Printf("Failed to list scores: %v", err)
		return
	}
	fmt.Printf("Found %d scores\n", len(scores.Data))

	// List sessions
	sessions, err := client.Sessions().List(ctx, nil)
	if err != nil {
		log.Printf("Failed to list sessions: %v", err)
		return
	}
	fmt.Printf("Found %d sessions\n", len(sessions.Data))
}
