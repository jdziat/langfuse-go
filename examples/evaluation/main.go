package main

import (
	"context"
	"fmt"
	"log"
	"os"

	langfuse "github.com/jdziat/langfuse-go"
	"github.com/jdziat/langfuse-go/evaluation"
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

	// ============================================================
	// Example 1: RAG (Retrieval-Augmented Generation) Trace
	// ============================================================
	fmt.Println("\n=== RAG Trace Example ===")

	ragTrace, err := evaluation.NewRAGTrace(client, "document-qa").
		Query("What are Go's main features for concurrency?").
		Context(
			"Go provides goroutines for lightweight concurrent execution.",
			"Channels in Go enable safe communication between goroutines.",
			"The select statement allows waiting on multiple channel operations.",
		).
		GroundTruth("Go has goroutines, channels, and select for concurrency").
		UserID("user-123").
		SessionID("session-456").
		Tags([]string{"rag", "golang", "docs"}).
		Create(ctx)
	if err != nil {
		log.Fatalf("Failed to create RAG trace: %v", err)
	}
	fmt.Printf("Created RAG trace: %s\n", ragTrace.ID())

	// Simulate processing and update with output
	err = ragTrace.UpdateOutput(
		ctx,
		"Go provides three main concurrency features: goroutines for lightweight threads, channels for safe communication, and the select statement for coordinating multiple channel operations.",
		"golang-docs.txt",
		"concurrency-guide.md",
	)
	if err != nil {
		log.Printf("Failed to update RAG output: %v", err)
	}

	// Validate the trace is ready for evaluation
	if err := ragTrace.ValidateForEvaluation(); err != nil {
		log.Printf("RAG trace validation failed: %v", err)
	} else {
		fmt.Println("RAG trace is ready for evaluation!")
	}

	// ============================================================
	// Example 2: Question-Answering Trace
	// ============================================================
	fmt.Println("\n=== Q&A Trace Example ===")

	qaTrace, err := evaluation.NewQATrace(client, "geography-quiz").
		Query("What is the capital of France?").
		GroundTruth("Paris").
		UserID("user-456").
		Create(ctx)
	if err != nil {
		log.Fatalf("Failed to create Q&A trace: %v", err)
	}
	fmt.Printf("Created Q&A trace: %s\n", qaTrace.ID())

	// Update with the model's answer
	err = qaTrace.UpdateOutput(ctx, "Paris", 0.99)
	if err != nil {
		log.Printf("Failed to update Q&A output: %v", err)
	}

	// Validate
	if err := qaTrace.ValidateForEvaluation(); err != nil {
		log.Printf("Q&A trace validation failed: %v", err)
	} else {
		fmt.Println("Q&A trace is ready for evaluation!")
	}

	// ============================================================
	// Example 3: Summarization Trace
	// ============================================================
	fmt.Println("\n=== Summarization Trace Example ===")

	longText := `Go is a statically typed, compiled programming language designed
at Google. It provides built-in support for concurrent programming through
goroutines and channels. Go's simplicity and performance make it ideal for
building scalable network services and distributed systems.`

	summaryTrace, err := evaluation.NewSummarizationTrace(client, "article-summarizer").
		Input(longText).
		MaxLength(50).
		Style("paragraph").
		GroundTruth("Go is a compiled language with built-in concurrency support.").
		UserID("user-789").
		Create(ctx)
	if err != nil {
		log.Fatalf("Failed to create summarization trace: %v", err)
	}
	fmt.Printf("Created Summarization trace: %s\n", summaryTrace.ID())

	// Update with generated summary
	err = summaryTrace.UpdateOutput(ctx, "Go is a compiled language designed at Google with built-in concurrency features.")
	if err != nil {
		log.Printf("Failed to update summarization output: %v", err)
	}

	// Validate
	if err := summaryTrace.ValidateForEvaluation(); err != nil {
		log.Printf("Summarization trace validation failed: %v", err)
	} else {
		fmt.Println("Summarization trace is ready for evaluation!")
	}

	// ============================================================
	// Example 4: Classification Trace
	// ============================================================
	fmt.Println("\n=== Classification Trace Example ===")

	classTrace, err := evaluation.NewClassificationTrace(client, "sentiment-analyzer").
		Input("I absolutely love this product! It exceeded all my expectations.").
		Classes([]string{"positive", "negative", "neutral"}).
		GroundTruth("positive").
		UserID("user-101").
		Create(ctx)
	if err != nil {
		log.Fatalf("Failed to create classification trace: %v", err)
	}
	fmt.Printf("Created Classification trace: %s\n", classTrace.ID())

	// Update with classification result and confidence scores
	scores := map[string]float64{
		"positive": 0.95,
		"negative": 0.02,
		"neutral":  0.03,
	}
	err = classTrace.UpdateOutputWithScores(ctx, "positive", scores)
	if err != nil {
		log.Printf("Failed to update classification output: %v", err)
	}

	// Validate
	if err := classTrace.ValidateForEvaluation(); err != nil {
		log.Printf("Classification trace validation failed: %v", err)
	} else {
		fmt.Println("Classification trace is ready for evaluation!")
	}

	// ============================================================
	// Example 5: Using Validation Utilities Directly
	// ============================================================
	fmt.Println("\n=== Direct Validation Example ===")

	// Create typed input/output structures
	input := &evaluation.RAGInput{
		Query:       "How do I use channels in Go?",
		Context:     []string{"Channels are typed conduits for sending values."},
		GroundTruth: "Use make(chan Type) to create a channel.",
	}
	output := &evaluation.RAGOutput{
		Output:     "Create a channel using make(chan Type) and send/receive with <- operator.",
		Citations:  []string{"go-docs.md"},
		Confidence: 0.92,
	}

	// Validate against different evaluators
	if err := evaluation.ValidateFor(input, output, evaluation.RAGEvaluator); err != nil {
		fmt.Printf("RAG validation failed: %v\n", err)
	} else {
		fmt.Println("Input/Output valid for RAG evaluator")
	}

	// Get detailed validation result
	result := evaluation.ValidateDetailed(input, output, evaluation.RAGEvaluator)
	fmt.Printf("Evaluator: %s\n", result.EvaluatorName)
	fmt.Printf("Valid: %v\n", result.Valid)
	fmt.Printf("Present fields: %v\n", result.PresentFields)
	if len(result.Warnings) > 0 {
		fmt.Printf("Warnings: %v\n", result.Warnings)
	}

	// ============================================================
	// Flush all events
	// ============================================================
	fmt.Println("\n=== Flushing Events ===")
	if err := client.Flush(ctx); err != nil {
		log.Printf("Failed to flush: %v", err)
	}

	fmt.Println("\nEvaluation examples completed successfully!")
}
