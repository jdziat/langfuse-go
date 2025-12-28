// Package evaluation provides evaluation-ready tracing for Langfuse.
//
// This package contains specialized trace builders and validation utilities
// for common LLM evaluation scenarios like RAG, Q&A, summarization, and classification.
//
// # Basic Usage
//
// Create evaluation-ready traces using the specialized builders:
//
//	client, _ := langfuse.New(publicKey, secretKey)
//	defer client.Shutdown(ctx)
//
//	// Create a RAG trace
//	trace, _ := evaluation.NewRAGTrace(client, "document-qa").
//	    Query("What are Go's concurrency features?").
//	    Context("Go has goroutines...", "Channels enable...").
//	    Create()
//
//	// Update with output
//	trace.UpdateOutput("Go provides goroutines and channels...", "docs.md")
//
//	// Validate before running evaluators
//	if err := trace.ValidateForEvaluation(); err != nil {
//	    log.Printf("Not ready: %v", err)
//	}
//
// # Validation
//
// Use validation utilities to check traces against evaluator requirements:
//
//	input := &evaluation.RAGInput{Query: "...", Context: []string{"..."}}
//	output := &evaluation.RAGOutput{Output: "..."}
//
//	if err := evaluation.ValidateFor(input, output, evaluation.RAGEvaluator); err != nil {
//	    log.Printf("Missing fields: %v", err)
//	}
//
// # Evaluator Requirements
//
// Pre-defined evaluator requirements are available:
//   - RAGEvaluator: Requires query, context, output
//   - QAEvaluator: Requires query, output
//   - SummarizationEvaluator: Requires input, output
//   - ClassificationEvaluator: Requires input, output
//   - And more...
package evaluation
