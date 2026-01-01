//go:build llm_as_judge_example

// Package main demonstrates the LLM-as-a-Judge aware evaluation API.
// This example shows how to create traces that are automatically structured
// for evaluation without requiring manual JSONPath configuration in Langfuse.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
	"github.com/jdziat/langfuse-go/evaluation"
)

func main() {
	// Example 1: Basic evaluation mode
	basicEvaluationMode()

	// Example 2: RAG workflow with automatic structuring
	ragWorkflowExample()

	// Example 3: Low-level evaluation-aware generation
	evalGenerationExample()

	// Example 4: RAGAS-optimized configuration
	ragasExample()
}

// basicEvaluationMode demonstrates enabling evaluation mode at the client level.
func basicEvaluationMode() {
	fmt.Println("\n=== Basic Evaluation Mode ===")

	// Create client with evaluation mode enabled
	// This automatically structures all traces for LLM-as-a-Judge
	client, err := langfuse.New(
		os.Getenv("LANGFUSE_PUBLIC_KEY"),
		os.Getenv("LANGFUSE_SECRET_KEY"),
		langfuse.WithRegion(langfuse.RegionUS),
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
		langfuse.WithDefaultWorkflow(langfuse.WorkflowRAG),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	// Create a trace - evaluation metadata is automatically added
	trace, err := client.NewTrace().
		Name("rag-query").
		UserID("user-123").
		Input(&langfuse.StandardEvalInput{
			Query:   "What are Go's concurrency features?",
			Context: []string{"Go has goroutines...", "Channels enable..."},
		}).
		Create(ctx)
	if err != nil {
		log.Fatalf("Failed to create trace: %v", err)
	}

	// The trace input is automatically flattened for evaluation:
	// {
	//   "query": "What are Go's concurrency features?",
	//   "input": "What are Go's concurrency features?",  // alias
	//   "context": ["Go has goroutines...", "Channels enable..."],
	//   "_langfuse_eval_metadata": {
	//     "workflow_type": "rag",
	//     "has_context": true,
	//     "compatible_evaluators": ["faithfulness", "answer_relevance", ...]
	//   }
	// }

	// Update with output
	err = trace.Update().Output(&langfuse.StandardEvalOutput{
		Output:    "Go provides goroutines for lightweight concurrency and channels for communication.",
		Citations: []string{"go-docs.md"},
	}).Apply(ctx)
	if err != nil {
		log.Printf("Failed to update trace: %v", err)
	}

	fmt.Printf("Created evaluation-ready trace: %s\n", trace.ID())
}

// ragWorkflowExample demonstrates the high-level RAG workflow builder.
func ragWorkflowExample() {
	fmt.Println("\n=== RAG Workflow Example ===")

	client, err := langfuse.New(
		os.Getenv("LANGFUSE_PUBLIC_KEY"),
		os.Getenv("LANGFUSE_SECRET_KEY"),
		langfuse.WithRegion(langfuse.RegionUS),
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	// Create a RAG workflow - this guides you through proper data structuring
	rag := evaluation.NewRAGWorkflow(client, "document-qa").
		UserID("user-456").
		SessionID("session-789").
		Tags("production", "api-v2").
		Query("What is dependency injection in Go?")

	// Simulate retrieval - automatically creates a retrieval span
	docs, err := rag.Retrieve(ctx, func() ([]string, error) {
		// In real code, this would call your vector DB
		return []string{
			"Dependency injection is a design pattern...",
			"In Go, DI is typically done through constructors...",
		}, nil
	})
	if err != nil {
		log.Fatalf("Retrieval failed: %v", err)
	}
	fmt.Printf("Retrieved %d documents\n", len(docs))

	// Simulate generation - automatically creates an evaluation-ready generation
	response, err := rag.Generate(ctx, "gpt-4", func(query string, context []string) (string, int, int, error) {
		// In real code, this would call your LLM
		return "Dependency injection in Go is typically implemented by passing dependencies as constructor parameters...",
			150, // input tokens
			50,  // output tokens
			nil
	})
	if err != nil {
		log.Fatalf("Generation failed: %v", err)
	}
	fmt.Printf("Generated response: %s...\n", response[:50])

	// Optional: Set ground truth for evaluation
	rag.GroundTruth("DI in Go uses constructor injection...")

	// Complete the workflow
	result, err := rag.Complete(ctx)
	if err != nil {
		log.Fatalf("Workflow completion failed: %v", err)
	}

	fmt.Printf("Workflow completed:\n")
	fmt.Printf("  - Trace ID: %s\n", result.ID())
	fmt.Printf("  - Ready for evaluation: %v\n", result.IsReady)
	fmt.Printf("  - Compatible evaluators: %v\n", result.CompatibleEvaluators)
	if len(result.MissingFields) > 0 {
		fmt.Printf("  - Missing fields: %v\n", result.MissingFields)
	}
}

// evalGenerationExample demonstrates low-level evaluation-aware generation.
func evalGenerationExample() {
	fmt.Println("\n=== Evaluation-Aware Generation Example ===")

	client, err := langfuse.New(
		os.Getenv("LANGFUSE_PUBLIC_KEY"),
		os.Getenv("LANGFUSE_SECRET_KEY"),
		langfuse.WithRegion(langfuse.RegionUS),
		langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	// Create trace
	trace, err := client.NewTrace().Name("eval-gen-demo").Create(ctx)
	if err != nil {
		log.Fatalf("Failed to create trace: %v", err)
	}

	// Create an evaluation-aware generation
	// This tracks evaluation fields and validates completeness
	gen, err := trace.NewEvalGeneration().
		Name("llm-response").
		Model("gpt-4").
		ForEvaluator(langfuse.EvaluatorFaithfulness, langfuse.EvaluatorHallucination).
		WithQuery("What is a goroutine?").
		WithContext("Goroutines are lightweight threads managed by Go runtime...").
		WithSystemPrompt("You are a helpful Go programming assistant.").
		Create(ctx)
	if err != nil {
		log.Fatalf("Failed to create generation: %v", err)
	}

	// Check what evaluators we can use
	fmt.Printf("Compatible evaluators: %v\n", gen.GetCompatibleEvaluators())

	// Simulate LLM call
	time.Sleep(100 * time.Millisecond)

	// Complete with structured result
	result := gen.CompleteWithEvaluation(ctx, &langfuse.EvalGenerationResult{
		Output:       "A goroutine is a lightweight thread of execution managed by the Go runtime...",
		InputTokens:  100,
		OutputTokens: 50,
		Model:        "gpt-4",
		Confidence:   0.95,
		Citations:    []string{"go-spec.md"},
	})

	if result.Error != nil {
		log.Printf("Failed to complete generation: %v", result.Error)
	}

	// Validate for specific evaluator
	if err := gen.ValidateForEvaluator(langfuse.EvaluatorFaithfulness); err != nil {
		fmt.Printf("Not ready for Faithfulness: %v\n", err)
	} else {
		fmt.Println("Ready for Faithfulness evaluation!")
	}

	fmt.Printf("Generation completed: %s\n", gen.GenerationID())
}

// ragasExample demonstrates RAGAS-optimized configuration.
func ragasExample() {
	fmt.Println("\n=== RAGAS-Optimized Example ===")

	// Create client optimized for RAGAS metrics
	// This uses RAGAS field naming conventions (user_input, retrieved_contexts, response)
	client, err := langfuse.New(
		os.Getenv("LANGFUSE_PUBLIC_KEY"),
		os.Getenv("LANGFUSE_SECRET_KEY"),
		langfuse.WithRegion(langfuse.RegionUS),
		langfuse.WithRAGASEvaluation(), // Convenience function for RAGAS setup
	)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	// With RAGAS mode, fields are automatically mapped:
	// query -> user_input
	// context -> retrieved_contexts
	// output -> response

	trace, err := client.NewTrace().
		Name("ragas-ready-trace").
		Input(map[string]any{
			"query":   "What is Go?",
			"context": []string{"Go is a programming language..."},
		}).
		Create(ctx)
	if err != nil {
		log.Fatalf("Failed to create trace: %v", err)
	}

	// The input is automatically transformed for RAGAS:
	// {
	//   "user_input": "What is Go?",
	//   "retrieved_contexts": ["Go is a programming language..."]
	// }

	err = trace.Update().Output(map[string]any{
		"output": "Go is a statically typed, compiled programming language...",
	}).Apply(ctx)
	if err != nil {
		log.Printf("Failed to update trace: %v", err)
	}

	fmt.Printf("Created RAGAS-ready trace: %s\n", trace.ID())
	fmt.Println("This trace can be evaluated with:")
	fmt.Println("  - Faithfulness")
	fmt.Println("  - Answer Relevance")
	fmt.Println("  - Context Precision")
	fmt.Println("  - Context Recall")
}
