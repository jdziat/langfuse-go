package evaluation_test

import (
	"context"
	"fmt"

	langfuse "github.com/jdziat/langfuse-go"
	"github.com/jdziat/langfuse-go/evaluation"
)

// ExampleNewRAGTrace shows the canonical evaluation flow for a RAG
// (retrieval-augmented generation) pipeline: capture the query and retrieved
// context, record the generated answer, and attach an evaluation score.
//
// Events are queued locally and flushed in the background, so the calls below
// succeed without a live network connection.
func ExampleNewRAGTrace() {
	client, err := langfuse.New("pk-lf-...", "sk-lf-...")
	if err != nil {
		fmt.Println("error creating client:", err)
		return
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	rag, err := evaluation.NewRAGTrace(client, "rag-query").
		Query("What is Go's concurrency model?").
		Context(
			"Go uses goroutines for lightweight concurrency.",
			"Channels coordinate communication between goroutines.",
		).
		GroundTruth("Go uses goroutines and channels.").
		UserID("user-123").
		Create(ctx)
	if err != nil {
		fmt.Println("error creating RAG trace:", err)
		return
	}

	// Record the generated answer with its citations.
	if err := rag.UpdateOutput(ctx,
		"Go provides goroutines and channels for concurrency.",
		"golang-docs.txt",
	); err != nil {
		fmt.Println("error updating output:", err)
		return
	}

	// Attach an evaluation score to the trace.
	_ = rag.Score(ctx, "faithfulness", 0.95)

	fmt.Println("RAG trace recorded")
	// Output: RAG trace recorded
}

// ExampleNewQATrace shows the question-answering evaluation flow: capture the
// query and ground truth, record the answer with a confidence, and score it.
func ExampleNewQATrace() {
	client, _ := langfuse.New("pk-lf-...", "sk-lf-...")
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	qa, err := evaluation.NewQATrace(client, "qa-query").
		Query("What is the capital of France?").
		GroundTruth("Paris").
		Create(ctx)
	if err != nil {
		fmt.Println("error creating QA trace:", err)
		return
	}

	// Record the answer with a confidence score.
	if err := qa.UpdateOutput(ctx, "Paris", 0.99); err != nil {
		fmt.Println("error updating output:", err)
		return
	}

	_ = qa.Score(ctx, "correctness", 1.0)

	fmt.Println("QA trace recorded")
	// Output: QA trace recorded
}
