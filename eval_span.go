package langfuse

import (
	"context"
	"time"
)

// EvalSpanType represents the type of evaluation span.
type EvalSpanType string

const (
	// EvalSpanRetrieval is a retrieval/search span.
	EvalSpanRetrieval EvalSpanType = "retrieval"

	// EvalSpanProcessing is a data processing span.
	EvalSpanProcessing EvalSpanType = "processing"

	// EvalSpanToolCall is a tool call span.
	EvalSpanToolCall EvalSpanType = "tool_call"

	// EvalSpanReasoning is a reasoning/thinking span.
	EvalSpanReasoning EvalSpanType = "reasoning"
)

// EvalSpanBuilder provides a fluent interface for creating evaluation-aware spans.
// It extends SpanBuilder with evaluation-specific methods for retrieval,
// tool calls, and other observable operations.
//
// Example:
//
//	retrieval, err := trace.NewRetrievalSpan().
//	    Name("vector-search").
//	    WithQuery(userQuery).
//	    Create(ctx)
//
//	docs := vectorDB.Search(query)
//	retrieval.WithContext(docs...).End(ctx)
type EvalSpanBuilder struct {
	*SpanBuilder
	spanType   EvalSpanType
	evalState  *EvalState
	evalConfig *EvaluationConfig
}

// NewEvalSpan creates a new evaluation-aware span builder.
func (t *TraceContext) NewEvalSpan() *EvalSpanBuilder {
	return &EvalSpanBuilder{
		SpanBuilder: t.NewSpan(),
		evalState:   NewEvalState(),
		evalConfig:  t.client.config.EvaluationConfig,
	}
}

// NewRetrievalSpan creates a span specifically for retrieval operations.
// This automatically tags the span for context retrieval evaluation.
func (t *TraceContext) NewRetrievalSpan() *EvalSpanBuilder {
	builder := t.NewEvalSpan()
	builder.spanType = EvalSpanRetrieval
	return builder
}

// NewToolCallSpan creates a span specifically for tool call operations.
func (t *TraceContext) NewToolCallSpan() *EvalSpanBuilder {
	builder := t.NewEvalSpan()
	builder.spanType = EvalSpanToolCall
	return builder
}

// Type sets the evaluation span type.
func (b *EvalSpanBuilder) Type(spanType EvalSpanType) *EvalSpanBuilder {
	b.spanType = spanType
	return b
}

// WithQuery sets the query/input for this span.
func (b *EvalSpanBuilder) WithQuery(query string) *EvalSpanBuilder {
	b.evalState.InputFields["query"] = true

	if b.span.Input == nil {
		b.span.Input = map[string]any{"query": query}
	} else if m, ok := b.span.Input.(map[string]any); ok {
		m["query"] = query
	}
	return b
}

// WithContext sets the context/retrieved documents.
func (b *EvalSpanBuilder) WithContext(context ...string) *EvalSpanBuilder {
	b.evalState.HasContext = true
	b.evalState.InputFields["context"] = true

	if b.span.Input == nil {
		b.span.Input = map[string]any{"context": context}
	} else if m, ok := b.span.Input.(map[string]any); ok {
		m["context"] = context
	}
	return b
}

// ID sets the span ID.
func (b *EvalSpanBuilder) ID(id string) *EvalSpanBuilder {
	b.SpanBuilder.ID(id)
	return b
}

// Name sets the span name.
func (b *EvalSpanBuilder) Name(name string) *EvalSpanBuilder {
	b.SpanBuilder.Name(name)
	return b
}

// Input sets the span input.
func (b *EvalSpanBuilder) Input(input any) *EvalSpanBuilder {
	b.SpanBuilder.Input(input)
	b.evalState.UpdateFromInput(input)
	return b
}

// Output sets the span output.
func (b *EvalSpanBuilder) Output(output any) *EvalSpanBuilder {
	b.SpanBuilder.Output(output)
	b.evalState.UpdateFromOutput(output)
	return b
}

// Metadata sets the span metadata.
func (b *EvalSpanBuilder) Metadata(metadata Metadata) *EvalSpanBuilder {
	b.SpanBuilder.Metadata(metadata)
	return b
}

// Level sets the observation level.
func (b *EvalSpanBuilder) Level(level ObservationLevel) *EvalSpanBuilder {
	b.SpanBuilder.Level(level)
	return b
}

// Environment sets the environment.
func (b *EvalSpanBuilder) Environment(env string) *EvalSpanBuilder {
	b.SpanBuilder.Environment(env)
	return b
}

// Create creates the evaluation-aware span.
func (b *EvalSpanBuilder) Create(ctx context.Context) (*EvalSpanContext, error) {
	// Apply evaluation transformations
	if b.evalConfig != nil && b.evalConfig.Mode != EvaluationModeOff {
		b.applyEvalTransformations()
	}

	spanCtx, err := b.SpanBuilder.Create(ctx)
	if err != nil {
		return nil, err
	}

	return &EvalSpanContext{
		SpanContext: spanCtx,
		spanType:    b.spanType,
		evalState:   b.evalState,
		evalConfig:  b.evalConfig,
	}, nil
}

// applyEvalTransformations applies evaluation-specific transformations.
func (b *EvalSpanBuilder) applyEvalTransformations() {
	config := b.evalConfig

	// Add span type to metadata
	if b.span.Metadata == nil {
		b.span.Metadata = make(Metadata)
	}
	if b.spanType != "" {
		b.span.Metadata["_eval_span_type"] = string(b.spanType)
	}

	// Flatten input if configured
	if config.FlattenInput && b.span.Input != nil {
		b.span.Input = prepareInputForEval(b.span.Input, config)
	}
}

// EvalSpanContext provides context for an evaluation-aware span.
type EvalSpanContext struct {
	*SpanContext
	spanType   EvalSpanType
	evalState  *EvalState
	evalConfig *EvaluationConfig
}

// GetSpanType returns the evaluation span type.
func (s *EvalSpanContext) GetSpanType() EvalSpanType {
	return s.spanType
}

// GetEvalState returns the current evaluation state.
func (s *EvalSpanContext) GetEvalState() *EvalState {
	return s.evalState
}

// WithContext sets the retrieved context and updates eval state.
// This is typically called after a retrieval operation completes.
func (s *EvalSpanContext) WithContext(ctx context.Context, documents ...string) *EvalSpanContext {
	s.evalState.HasContext = true
	s.evalState.OutputFields["context"] = true

	// Build output structure
	output := &RetrievalOutput{
		Documents:    documents,
		NumDocuments: len(documents),
	}

	s.evalState.UpdateFromOutput(output)
	return s
}

// EndWithContext ends the span with the retrieved context.
func (s *EvalSpanContext) EndWithContext(ctx context.Context, documents ...string) error {
	s.WithContext(ctx, documents...)

	output := &RetrievalOutput{
		Documents:    documents,
		NumDocuments: len(documents),
	}

	// Apply output transformation
	var finalOutput any = output
	if s.evalConfig != nil && s.evalConfig.Mode != EvaluationModeOff && s.evalConfig.FlattenOutput {
		finalOutput = prepareOutputForEval(output, s.evalConfig)
	}

	return s.Update().Output(finalOutput).EndTime(time.Now()).Apply(ctx)
}

// EndWithToolResult ends the span with a tool call result.
func (s *EvalSpanContext) EndWithToolResult(ctx context.Context, result *ToolCallResult) error {
	s.evalState.OutputFields["tool_result"] = true
	s.evalState.UpdateFromOutput(result)

	var finalOutput any = result
	if s.evalConfig != nil && s.evalConfig.Mode != EvaluationModeOff && s.evalConfig.FlattenOutput {
		finalOutput = prepareOutputForEval(result, s.evalConfig)
	}

	return s.Update().Output(finalOutput).EndTime(time.Now()).Apply(ctx)
}

// NewEvalGeneration creates a child evaluation-aware generation.
func (s *EvalSpanContext) NewEvalGeneration() *EvalGenerationBuilder {
	builder := s.TraceContext.NewEvalGeneration()
	builder.gen.ParentObservationID = s.spanID
	// Inherit context from parent span
	if s.evalState.HasContext {
		builder.evalState.HasContext = true
		for k, v := range s.evalState.InputFields {
			if k == "context" || k == "retrieved_contexts" {
				builder.evalState.InputFields[k] = v
			}
		}
	}
	return builder
}

// NewRetrievalSpan creates a child retrieval span.
func (s *EvalSpanContext) NewRetrievalSpan() *EvalSpanBuilder {
	builder := s.TraceContext.NewRetrievalSpan()
	builder.span.ParentObservationID = s.spanID
	return builder
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

// EvalFields implements EvalOutput for RetrievalOutput.
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

// EvalFields implements EvalOutput for ToolCallResult.
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
