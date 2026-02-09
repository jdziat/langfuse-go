package evaluation

import (
	"context"
	"fmt"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
)

// RAGWorkflow provides a specialized builder for RAG (Retrieval-Augmented Generation)
// workflows that are optimized for evaluation.
//
// Example:
//
//	rag := evaluation.NewRAGWorkflow(client, "document-qa").
//	    UserID("user-123").
//	    Query("What are Go's concurrency features?")
//
//	// Record retrieval
//	docs, err := rag.Retrieve(ctx, func() ([]string, error) {
//	    return vectorDB.Search(query)
//	})
//
//	// Record generation
//	response, err := rag.Generate(ctx, "gpt-4", func(prompt string) (string, int, int, error) {
//	    resp := openai.Complete(prompt)
//	    return resp.Content, resp.InputTokens, resp.OutputTokens, nil
//	})
//
//	// Complete
//	result, err := rag.Complete(ctx)
type RAGWorkflow struct {
	*WorkflowBuilder

	// RAG-specific state
	retrievedDocs   []string
	retrievalScores []float64
	citations       []string
	generatedAnswer string
	confidence      float64
}

// NewRAGWorkflow creates a new RAG workflow builder.
func NewRAGWorkflow(client *langfuse.Client, name string) *RAGWorkflow {
	return &RAGWorkflow{
		WorkflowBuilder: NewWorkflow(client, langfuse.WorkflowRAG).Name(name),
	}
}

// UserID sets the user ID.
func (r *RAGWorkflow) UserID(userID string) *RAGWorkflow {
	r.WorkflowBuilder.UserID(userID)
	return r
}

// SessionID sets the session ID.
func (r *RAGWorkflow) SessionID(sessionID string) *RAGWorkflow {
	r.WorkflowBuilder.SessionID(sessionID)
	return r
}

// Tags sets the trace tags.
func (r *RAGWorkflow) Tags(tags ...string) *RAGWorkflow {
	r.WorkflowBuilder.Tags(tags...)
	return r
}

// Metadata sets additional metadata.
func (r *RAGWorkflow) Metadata(metadata langfuse.Metadata) *RAGWorkflow {
	r.WorkflowBuilder.Metadata(metadata)
	return r
}

// Environment sets the environment.
func (r *RAGWorkflow) Environment(env string) *RAGWorkflow {
	r.WorkflowBuilder.Environment(env)
	return r
}

// Query sets the user's question.
func (r *RAGWorkflow) Query(query string) *RAGWorkflow {
	r.WorkflowBuilder.WithQuery(query)
	return r
}

// GroundTruth sets the expected correct answer for evaluation.
func (r *RAGWorkflow) GroundTruth(groundTruth string) *RAGWorkflow {
	r.WorkflowBuilder.WithGroundTruth(groundTruth)
	return r
}

// RetrieveFunc is a function that performs document retrieval.
type RetrieveFunc func() ([]string, error)

// RetrieveWithScoresFunc is a function that performs retrieval and returns scores.
type RetrieveWithScoresFunc func() ([]string, []float64, error)

// Retrieve executes a retrieval function and records it as a span.
func (r *RAGWorkflow) Retrieve(ctx context.Context, retrieveFunc RetrieveFunc) ([]string, error) {
	if err := r.Start(ctx); err != nil {
		return nil, err
	}

	// Create retrieval span
	span, err := r.trace.NewRetrievalSpan().
		Name("document-retrieval").
		WithQuery(r.query).
		Create(ctx)
	if err != nil {
		return nil, err
	}

	startTime := time.Now()

	// Execute retrieval
	docs, err := retrieveFunc()
	if err != nil {
		// End span with error
		span.Update().
			Level(langfuse.ObservationLevelError).
			StatusMessage(err.Error()).
			EndTime(time.Now()).
			Apply(ctx)
		return nil, fmt.Errorf("retrieval failed: %w", err)
	}

	// Store retrieved docs
	r.retrievedDocs = docs
	r.WithContext(docs...)

	// End span with context
	output := &langfuse.RetrievalOutput{
		Documents:    docs,
		NumDocuments: len(docs),
		Metadata: map[string]any{
			"duration_ms": time.Since(startTime).Milliseconds(),
		},
	}

	span.Update().Output(output).EndTime(time.Now()).Apply(ctx)

	return docs, nil
}

// RetrieveWithScores executes a retrieval function that also returns relevance scores.
func (r *RAGWorkflow) RetrieveWithScores(ctx context.Context, retrieveFunc RetrieveWithScoresFunc) ([]string, []float64, error) {
	if err := r.Start(ctx); err != nil {
		return nil, nil, err
	}

	// Create retrieval span
	span, err := r.trace.NewRetrievalSpan().
		Name("document-retrieval").
		WithQuery(r.query).
		Create(ctx)
	if err != nil {
		return nil, nil, err
	}

	startTime := time.Now()

	// Execute retrieval
	docs, scores, err := retrieveFunc()
	if err != nil {
		span.Update().
			Level(langfuse.ObservationLevelError).
			StatusMessage(err.Error()).
			EndTime(time.Now()).
			Apply(ctx)
		return nil, nil, fmt.Errorf("retrieval failed: %w", err)
	}

	// Store retrieved docs
	r.retrievedDocs = docs
	r.retrievalScores = scores
	r.WithContext(docs...)

	// End span with context and scores
	output := &langfuse.RetrievalOutput{
		Documents:    docs,
		NumDocuments: len(docs),
		Scores:       scores,
		Metadata: map[string]any{
			"duration_ms": time.Since(startTime).Milliseconds(),
		},
	}

	span.Update().Output(output).EndTime(time.Now()).Apply(ctx)

	return docs, scores, nil
}

// GenerateFunc is a function that generates a response.
// Returns: output, input_tokens, output_tokens, error
type GenerateFunc func(prompt string) (string, int, int, error)

// GenerateWithPromptFunc receives the full prompt including context.
type GenerateWithPromptFunc func(query string, context []string) (string, int, int, error)

// Generate executes a generation function and records it.
func (r *RAGWorkflow) Generate(ctx context.Context, model string, generateFunc GenerateWithPromptFunc) (string, error) {
	if err := r.Start(ctx); err != nil {
		return "", err
	}

	// Create generation
	gen, err := r.trace.NewEvalGeneration().
		Name("llm-response").
		Model(model).
		WithQuery(r.query).
		WithContext(r.retrievedDocs...).
		Create(ctx)
	if err != nil {
		return "", err
	}

	startTime := time.Now()
	completionStart := time.Now()

	// Execute generation
	output, inputTokens, outputTokens, err := generateFunc(r.query, r.retrievedDocs)
	if err != nil {
		gen.Update().
			Level(langfuse.ObservationLevelError).
			StatusMessage(err.Error()).
			EndTime(time.Now()).
			Apply(ctx)
		return "", fmt.Errorf("generation failed: %w", err)
	}

	// Store generated answer
	r.generatedAnswer = output
	r.WithOutput(output)

	// Complete generation
	result := &langfuse.EvalGenerationResult{
		Output:         output,
		InputTokens:    inputTokens,
		OutputTokens:   outputTokens,
		Model:          model,
		CompletionTime: completionStart,
	}

	endResult := gen.CompleteWithEvaluation(ctx, result)
	if endResult.Error != nil {
		return "", endResult.Error
	}

	// Log generation duration in metadata if needed
	_ = time.Since(startTime)

	return output, nil
}

// GenerateWithCitations executes generation and extracts citations.
type GenerateWithCitationsFunc func(query string, context []string) (output string, citations []string, inputTokens, outputTokens int, err error)

// GenerateWithCitations executes a generation that also returns citations.
func (r *RAGWorkflow) GenerateWithCitations(ctx context.Context, model string, generateFunc GenerateWithCitationsFunc) (string, []string, error) {
	if err := r.Start(ctx); err != nil {
		return "", nil, err
	}

	// Create generation
	gen, err := r.trace.NewEvalGeneration().
		Name("llm-response").
		Model(model).
		WithQuery(r.query).
		WithContext(r.retrievedDocs...).
		Create(ctx)
	if err != nil {
		return "", nil, err
	}

	completionStart := time.Now()

	// Execute generation
	output, citations, inputTokens, outputTokens, err := generateFunc(r.query, r.retrievedDocs)
	if err != nil {
		gen.Update().
			Level(langfuse.ObservationLevelError).
			StatusMessage(err.Error()).
			EndTime(time.Now()).
			Apply(ctx)
		return "", nil, fmt.Errorf("generation failed: %w", err)
	}

	// Store results
	r.generatedAnswer = output
	r.citations = citations
	r.WithOutput(output)

	// Complete generation
	result := &langfuse.EvalGenerationResult{
		Output:         output,
		Citations:      citations,
		InputTokens:    inputTokens,
		OutputTokens:   outputTokens,
		Model:          model,
		CompletionTime: completionStart,
	}

	endResult := gen.CompleteWithEvaluation(ctx, result)
	if endResult.Error != nil {
		return "", nil, endResult.Error
	}

	return output, citations, nil
}

// SetConfidence sets the confidence score for the generated answer.
func (r *RAGWorkflow) SetConfidence(confidence float64) *RAGWorkflow {
	r.confidence = confidence
	return r
}

// Complete finalizes the RAG workflow.
func (r *RAGWorkflow) Complete(ctx context.Context) (*RAGWorkflowResult, error) {
	baseResult, err := r.WorkflowBuilder.Complete(ctx)
	if err != nil {
		return nil, err
	}

	return &RAGWorkflowResult{
		WorkflowResult:  baseResult,
		Query:           r.query,
		RetrievedDocs:   r.retrievedDocs,
		RetrievalScores: r.retrievalScores,
		GeneratedAnswer: r.generatedAnswer,
		Citations:       r.citations,
		Confidence:      r.confidence,
	}, nil
}

// RAGWorkflowResult contains the complete RAG workflow result.
type RAGWorkflowResult struct {
	*WorkflowResult

	// Query is the user's question.
	Query string

	// RetrievedDocs are the retrieved context documents.
	RetrievedDocs []string

	// RetrievalScores are the relevance scores for retrieved docs.
	RetrievalScores []float64

	// GeneratedAnswer is the LLM's response.
	GeneratedAnswer string

	// Citations are source documents referenced in the answer.
	Citations []string

	// Confidence is the confidence score.
	Confidence float64
}

// GetRAGEvaluators returns the list of RAGAS evaluators this result is compatible with.
func (r *RAGWorkflowResult) GetRAGEvaluators() []langfuse.EvaluatorType {
	evaluators := []langfuse.EvaluatorType{}

	hasContext := len(r.RetrievedDocs) > 0
	hasOutput := r.GeneratedAnswer != ""
	hasQuery := r.Query != ""

	// Faithfulness: context + output
	if hasContext && hasOutput {
		evaluators = append(evaluators, langfuse.EvaluatorFaithfulness)
	}

	// Answer Relevance: query + output
	if hasQuery && hasOutput {
		evaluators = append(evaluators, langfuse.EvaluatorAnswerRelevance)
	}

	// Hallucination: query + context + output
	if hasQuery && hasContext && hasOutput {
		evaluators = append(evaluators, langfuse.EvaluatorHallucination)
	}

	return evaluators
}
