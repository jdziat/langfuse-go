package evaluation

import "time"

// GenerationResult contains the result of an LLM generation for evaluation.
type GenerationResult struct {
	// Output is the generated response text.
	Output string

	// InputTokens is the number of input tokens used.
	InputTokens int

	// OutputTokens is the number of output tokens used.
	OutputTokens int

	// Model is the model that was actually used (may differ from requested).
	Model string

	// CompletionTime is when the model started generating (for TTFT).
	CompletionTime time.Time

	// Confidence is the model's confidence in the response.
	Confidence float64

	// Citations are source documents referenced.
	Citations []string

	// ToolCalls are any tool calls made during generation.
	ToolCalls []map[string]any

	// Reasoning is chain-of-thought reasoning (if available).
	Reasoning string
}

// ToStandardOutput converts the result to a StandardOutput.
func (r *GenerationResult) ToStandardOutput() *StandardOutput {
	return &StandardOutput{
		Output:     r.Output,
		Citations:  r.Citations,
		Confidence: r.Confidence,
		Reasoning:  r.Reasoning,
		ToolCalls:  r.ToolCalls,
	}
}

// EvalFields implements Output for GenerationResult.
func (r *GenerationResult) EvalFields() map[string]any {
	result := map[string]any{
		"output": r.Output,
	}
	if len(r.Citations) > 0 {
		result["citations"] = r.Citations
	}
	if r.Confidence > 0 {
		result["confidence"] = r.Confidence
	}
	if r.Reasoning != "" {
		result["reasoning"] = r.Reasoning
	}
	if len(r.ToolCalls) > 0 {
		result["tool_calls"] = r.ToolCalls
	}
	return result
}

// RetrievalOutput represents the output of a retrieval operation.
type RetrievalOutput struct {
	// Documents are the retrieved document chunks.
	Documents []string `json:"documents"`

	// NumDocuments is the number of documents retrieved.
	NumDocuments int `json:"num_documents"`

	// Query is the query used (if different from input).
	Query string `json:"query,omitempty"`

	// Scores are the relevance scores for each document.
	Scores []float64 `json:"scores,omitempty"`

	// Source indicates where documents came from.
	Source string `json:"source,omitempty"`

	// Metadata contains additional retrieval metadata.
	Metadata map[string]any `json:"metadata,omitempty"`
}

// EvalFields implements Output for RetrievalOutput.
func (r *RetrievalOutput) EvalFields() map[string]any {
	result := map[string]any{
		"context":       r.Documents,
		"num_documents": r.NumDocuments,
	}
	if r.Query != "" {
		result["query"] = r.Query
	}
	if len(r.Scores) > 0 {
		result["scores"] = r.Scores
	}
	if r.Source != "" {
		result["source"] = r.Source
	}
	return result
}

// ToolCallResult represents the result of a tool call.
type ToolCallResult struct {
	// ToolName is the name of the tool called.
	ToolName string `json:"tool_name"`

	// Input is the input provided to the tool.
	Input any `json:"input"`

	// Output is the output returned by the tool.
	Output any `json:"output"`

	// Success indicates if the tool call succeeded.
	Success bool `json:"success"`

	// Error contains any error message.
	Error string `json:"error,omitempty"`

	// Duration is how long the tool call took.
	Duration time.Duration `json:"duration,omitempty"`
}

// EvalFields implements Output for ToolCallResult.
func (t *ToolCallResult) EvalFields() map[string]any {
	result := map[string]any{
		"tool_name": t.ToolName,
		"success":   t.Success,
	}
	if t.Input != nil {
		result["tool_input"] = t.Input
	}
	if t.Output != nil {
		result["tool_output"] = t.Output
	}
	if t.Error != "" {
		result["error"] = t.Error
	}
	if t.Duration > 0 {
		result["duration_ms"] = t.Duration.Milliseconds()
	}
	return result
}
