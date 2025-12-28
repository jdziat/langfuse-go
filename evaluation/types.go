package evaluation

// EvaluationType represents supported evaluation scenarios.
type EvaluationType string

const (
	EvaluationTypeRAG            EvaluationType = "rag"
	EvaluationTypeQA             EvaluationType = "qa"
	EvaluationTypeSummarization  EvaluationType = "summarization"
	EvaluationTypeClassification EvaluationType = "classification"
)

// RAGInput represents input for RAG (Retrieval-Augmented Generation) workflows.
// This structure matches Langfuse's RAG evaluator expectations.
//
// Example:
//
//	input := &evaluation.RAGInput{
//	    Query: "What are Go's concurrency features?",
//	    Context: []string{
//	        "Go has built-in goroutines for lightweight concurrency.",
//	        "Channels enable communication between goroutines.",
//	    },
//	    GroundTruth: "Go provides goroutines and channels for concurrency.",
//	}
type RAGInput struct {
	// Query is the user's question or search query (required for evaluation)
	Query string `json:"query"`

	// Context contains retrieved context chunks from your knowledge base (required for evaluation)
	Context []string `json:"context"`

	// GroundTruth is the expected correct answer for evaluation (optional)
	GroundTruth string `json:"ground_truth,omitempty"`

	// AdditionalContext allows passing extra metadata
	AdditionalContext map[string]any `json:"additional_context,omitempty"`
}

// RAGOutput represents output from a RAG workflow.
//
// Example:
//
//	output := &evaluation.RAGOutput{
//	    Output: "Go provides goroutines for lightweight concurrency...",
//	    Citations: []string{"golang-docs.txt", "concurrency-guide.md"},
//	    SourceChunks: []int{0, 1},
//	}
type RAGOutput struct {
	// Output is the generated response (required for evaluation)
	Output string `json:"output"`

	// Citations lists source documents used (optional)
	Citations []string `json:"citations,omitempty"`

	// SourceChunks indicates which context chunks were used (optional)
	SourceChunks []int `json:"source_chunks,omitempty"`

	// Confidence is the model's confidence in the answer (optional)
	Confidence float64 `json:"confidence,omitempty"`

	// Metadata allows passing additional metadata
	Metadata map[string]any `json:"metadata,omitempty"`
}

// QAInput represents input for question-answering workflows.
//
// Example:
//
//	input := &evaluation.QAInput{
//	    Query: "What is the capital of France?",
//	    GroundTruth: "Paris",
//	}
type QAInput struct {
	// Query is the user's question (required for evaluation)
	Query string `json:"query"`

	// GroundTruth is the expected correct answer for evaluation (optional)
	GroundTruth string `json:"ground_truth,omitempty"`

	// Context provides additional context for the question (optional)
	Context string `json:"context,omitempty"`
}

// QAOutput represents output from a question-answering workflow.
type QAOutput struct {
	// Output is the generated answer (required for evaluation)
	Output string `json:"output"`

	// Confidence is the model's confidence in the answer (optional)
	Confidence float64 `json:"confidence,omitempty"`

	// Reasoning provides explanation for the answer (optional)
	Reasoning string `json:"reasoning,omitempty"`

	// Metadata allows passing additional metadata
	Metadata map[string]any `json:"metadata,omitempty"`
}

// SummarizationInput represents input for summarization workflows.
//
// Example:
//
//	input := &evaluation.SummarizationInput{
//	    Input: longArticleText,
//	    MaxLength: 500,
//	    GroundTruth: expertSummary,
//	}
type SummarizationInput struct {
	// Input is the original text to summarize (required for evaluation)
	Input string `json:"input"`

	// GroundTruth is a reference summary for evaluation (optional)
	GroundTruth string `json:"ground_truth,omitempty"`

	// MaxLength specifies target summary length in words (optional)
	MaxLength int `json:"max_length,omitempty"`

	// Style specifies summary style (e.g., "bullet_points", "paragraph") (optional)
	Style string `json:"style,omitempty"`
}

// SummarizationOutput represents output from summarization workflows.
type SummarizationOutput struct {
	// Output is the generated summary (required for evaluation)
	Output string `json:"output"`

	// Length is the summary length in words (optional)
	Length int `json:"length,omitempty"`

	// CompressionRatio indicates input:output length ratio (optional)
	CompressionRatio float64 `json:"compression_ratio,omitempty"`

	// Metadata allows passing additional metadata
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ClassificationInput represents input for classification workflows.
type ClassificationInput struct {
	// Input is the text to classify (required for evaluation)
	Input string `json:"input"`

	// Classes lists possible classification categories (optional)
	Classes []string `json:"classes,omitempty"`

	// GroundTruth is the expected classification for evaluation (optional)
	GroundTruth string `json:"ground_truth,omitempty"`
}

// ClassificationOutput represents output from classification workflows.
type ClassificationOutput struct {
	// Output is the predicted class (required for evaluation)
	Output string `json:"output"`

	// Confidence is the prediction confidence (optional)
	Confidence float64 `json:"confidence,omitempty"`

	// Scores provides confidence scores for all classes (optional)
	Scores map[string]float64 `json:"scores,omitempty"`

	// Metadata allows passing additional metadata
	Metadata map[string]any `json:"metadata,omitempty"`
}

// ToxicityInput represents input for toxicity evaluation.
type ToxicityInput struct {
	// Input is the text to evaluate for toxicity (required)
	Input string `json:"input"`
}

// ToxicityOutput represents output from toxicity evaluation.
type ToxicityOutput struct {
	// Output is the generated text to evaluate (required)
	Output string `json:"output"`

	// ToxicityScore is the toxicity level (0-1) (optional)
	ToxicityScore float64 `json:"toxicity_score,omitempty"`

	// Categories lists detected toxicity categories (optional)
	Categories []string `json:"categories,omitempty"`

	// Metadata allows passing additional metadata
	Metadata map[string]any `json:"metadata,omitempty"`
}

// HallucinationInput represents input for hallucination detection.
type HallucinationInput struct {
	// Query is the user's question (required)
	Query string `json:"query"`

	// Context is the source of truth (required)
	Context []string `json:"context"`
}

// HallucinationOutput represents output for hallucination evaluation.
type HallucinationOutput struct {
	// Output is the generated response to evaluate (required)
	Output string `json:"output"`

	// HallucinationScore indicates degree of hallucination (0-1) (optional)
	HallucinationScore float64 `json:"hallucination_score,omitempty"`

	// Metadata allows passing additional metadata
	Metadata map[string]any `json:"metadata,omitempty"`
}
