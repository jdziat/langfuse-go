package langfuse

import (
	"context"
	"encoding/json"
	"time"

	pkgeval "github.com/jdziat/langfuse-go/pkg/evaluation"
)

// ============================================================================
// Evaluation Types - Re-exported from pkg/evaluation
// ============================================================================

// EvaluationMode controls how traces are structured for LLM-as-a-Judge evaluation.
type EvaluationMode = pkgeval.Mode

const (
	// EvaluationModeOff disables automatic evaluation structuring (default).
	EvaluationModeOff = pkgeval.ModeOff

	// EvaluationModeAuto automatically structures data for common evaluators.
	EvaluationModeAuto = pkgeval.ModeAuto

	// EvaluationModeRAGAS structures data specifically for RAGAS metrics.
	EvaluationModeRAGAS = pkgeval.ModeRAGAS

	// EvaluationModeLangfuse structures data for Langfuse managed evaluators.
	EvaluationModeLangfuse = pkgeval.ModeLangfuse
)

// WorkflowType represents the type of LLM workflow being traced.
type WorkflowType = pkgeval.WorkflowType

const (
	WorkflowRAG            = pkgeval.WorkflowRAG
	WorkflowQA             = pkgeval.WorkflowQA
	WorkflowChatCompletion = pkgeval.WorkflowChatCompletion
	WorkflowAgentTask      = pkgeval.WorkflowAgentTask
	WorkflowChainOfThought = pkgeval.WorkflowChainOfThought
	WorkflowSummarization  = pkgeval.WorkflowSummarization
	WorkflowClassification = pkgeval.WorkflowClassification
)

// EvaluatorType represents built-in evaluator types.
type EvaluatorType = pkgeval.EvaluatorType

const (
	EvaluatorFaithfulness     = pkgeval.EvaluatorFaithfulness
	EvaluatorAnswerRelevance  = pkgeval.EvaluatorAnswerRelevance
	EvaluatorContextPrecision = pkgeval.EvaluatorContextPrecision
	EvaluatorContextRecall    = pkgeval.EvaluatorContextRecall
	EvaluatorHallucination    = pkgeval.EvaluatorHallucination
	EvaluatorToxicity         = pkgeval.EvaluatorToxicity
	EvaluatorCorrectness      = pkgeval.EvaluatorCorrectness
)

// EvaluationConfig holds configuration for evaluation mode.
type EvaluationConfig = pkgeval.Config

// DefaultEvaluationConfig returns a sensible default evaluation configuration.
var DefaultEvaluationConfig = pkgeval.DefaultConfig

// RAGASEvaluationConfig returns configuration optimized for RAGAS evaluators.
var RAGASEvaluationConfig = pkgeval.RAGASConfig

// EvalMetadata contains evaluation-specific metadata added to traces.
type EvalMetadata = pkgeval.Metadata

// EvalMetadataKey is the metadata key used for evaluation metadata.
const EvalMetadataKey = pkgeval.MetadataKey

// EvalMetadataVersion is the current version of the evaluation metadata schema.
const EvalMetadataVersion = pkgeval.MetadataVersion

// Evaluation tag constants.
const (
	EvalTagPrefix      = pkgeval.TagPrefix
	EvalTagReady       = pkgeval.TagReady
	EvalTagNotReady    = pkgeval.TagNotReady
	EvalTagGroundTruth = pkgeval.TagGroundTruth
)

// EvalTagForWorkflow returns the evaluation tag for a workflow type.
var EvalTagForWorkflow = pkgeval.TagForWorkflow

// EvalTagForEvaluator returns the evaluation tag for an evaluator type.
var EvalTagForEvaluator = pkgeval.TagForEvaluator

// EvalSpanType represents the type of evaluation span.
type EvalSpanType = pkgeval.SpanType

const (
	EvalSpanRetrieval  = pkgeval.SpanRetrieval
	EvalSpanProcessing = pkgeval.SpanProcessing
	EvalSpanToolCall   = pkgeval.SpanToolCall
	EvalSpanReasoning  = pkgeval.SpanReasoning
)

// EvalInput represents a structured input that can be flattened for evaluation.
type EvalInput = pkgeval.Input

// EvalOutput represents a structured output that can be flattened for evaluation.
type EvalOutput = pkgeval.Output

// InputFlattener flattens structured inputs for evaluation.
type InputFlattener = pkgeval.InputFlattener

// NewInputFlattener creates a new input flattener for the given mode.
var NewInputFlattener = pkgeval.NewInputFlattener

// FlattenedInput wraps flattened input data with metadata.
type FlattenedInput = pkgeval.FlattenedInput

// FlattenedOutput wraps flattened output data with metadata.
type FlattenedOutput = pkgeval.FlattenedOutput

// StandardEvalInput provides a standard evaluation input structure.
type StandardEvalInput = pkgeval.StandardInput

// StandardEvalOutput provides a standard evaluation output structure.
type StandardEvalOutput = pkgeval.StandardOutput

// EvalState tracks the current evaluation state for a trace.
type EvalState = pkgeval.State

// NewEvalState creates a new evaluation state tracker.
var NewEvalState = pkgeval.NewState

// EvalMetadataBuilder helps build evaluation metadata.
type EvalMetadataBuilder = pkgeval.MetadataBuilder

// NewEvalMetadataBuilder creates a new evaluation metadata builder.
var NewEvalMetadataBuilder = pkgeval.NewMetadataBuilder

// ValidateForEvaluator checks if data has all required fields for an evaluator.
var ValidateForEvaluator = pkgeval.ValidateForEvaluator

// ValidateForWorkflow checks if data has all required fields for a workflow.
var ValidateForWorkflow = pkgeval.ValidateForWorkflow

// FieldAlias returns an alias for a field name (for compatibility).
var FieldAlias = pkgeval.FieldAlias

// ExtractFieldPresence checks which evaluation fields are present in data.
var ExtractFieldPresence = pkgeval.ExtractFieldPresence

// MergeFieldPresence merges two presence maps.
var MergeFieldPresence = pkgeval.MergeFieldPresence

// EvalGenerationResult contains the result of an LLM generation for evaluation.
type EvalGenerationResult = pkgeval.GenerationResult

// RetrievalOutput represents the output of a retrieval operation.
type RetrievalOutput = pkgeval.RetrievalOutput

// ToolCallResult represents the result of a tool call.
type ToolCallResult = pkgeval.ToolCallResult

// EventPersistence handles saving and loading events to/from disk.
type EventPersistence = pkgeval.EventPersistence

// NewEventPersistence creates a new event persistence handler.
var NewEventPersistence = pkgeval.NewEventPersistence

// PersistedEvent represents an event saved to disk.
type PersistedEvent = pkgeval.PersistedEvent

// PersistedBatch represents a batch of events saved to disk.
type PersistedBatch = pkgeval.PersistedBatch

// PersistenceConfig configures the event persistence behavior.
type PersistenceConfig = pkgeval.PersistenceConfig

// DefaultPersistenceConfig returns a PersistenceConfig with sensible defaults.
var DefaultPersistenceConfig = pkgeval.DefaultPersistenceConfig

// ManagedPersistence wraps EventPersistence with automatic cleanup.
type ManagedPersistence = pkgeval.ManagedPersistence

// NewManagedPersistence creates a new managed persistence handler.
var NewManagedPersistence = pkgeval.NewManagedPersistence

// ============================================================================
// Internal Helper Functions
// ============================================================================

// prepareInputForEval prepares input data for evaluation based on config.
func prepareInputForEval(data any, config *EvaluationConfig) any {
	return pkgeval.PrepareInput(data, config)
}

// prepareOutputForEval prepares output data for evaluation based on config.
func prepareOutputForEval(data any, config *EvaluationConfig) any {
	return pkgeval.PrepareOutput(data, config)
}

// mergeMetadata merges evaluation metadata into existing metadata.
func mergeMetadata(existing Metadata, evalMeta map[string]any) Metadata {
	if existing == nil {
		existing = make(Metadata)
	}
	for k, v := range evalMeta {
		existing[k] = v
	}
	return existing
}

// joinStrings joins strings with a separator.
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

// EventsToPersistedEvents converts ingestion events to persisted format.
func EventsToPersistedEvents(events []ingestionEvent) []PersistedEvent {
	result := make([]PersistedEvent, 0, len(events))

	for _, e := range events {
		var bodyMap map[string]any
		if data, err := json.Marshal(e.Body); err == nil {
			if err := json.Unmarshal(data, &bodyMap); err != nil {
				bodyMap = map[string]any{
					"_conversionError": err.Error(),
				}
			}
		}

		result = append(result, PersistedEvent{
			ID:        e.ID,
			Type:      e.Type,
			Timestamp: e.Timestamp.Time,
			Body:      bodyMap,
		})
	}

	return result
}

// ============================================================================
// Eval Generation Builder (Client-dependent)
// ============================================================================

// EvalGenerationBuilder provides a fluent interface for creating evaluation-aware generations.
type EvalGenerationBuilder struct {
	*GenerationBuilder
	evalState  *EvalState
	evalConfig *EvaluationConfig
}

// NewEvalGeneration creates a new evaluation-aware generation builder.
func (t *TraceContext) NewEvalGeneration() *EvalGenerationBuilder {
	return &EvalGenerationBuilder{
		GenerationBuilder: t.NewGeneration(),
		evalState:         NewEvalState(),
		evalConfig:        t.client.rootConfig.EvaluationConfig,
	}
}

// ForEvaluator specifies which evaluator(s) this generation should be optimized for.
func (b *EvalGenerationBuilder) ForEvaluator(evaluators ...EvaluatorType) *EvalGenerationBuilder {
	b.evalState.TargetEvaluators = append(b.evalState.TargetEvaluators, evaluators...)
	return b
}

// ForWorkflow specifies the workflow type for this generation.
func (b *EvalGenerationBuilder) ForWorkflow(workflow WorkflowType) *EvalGenerationBuilder {
	b.evalState.WorkflowType = workflow
	return b
}

// WithQuery sets the user query/input for evaluation.
func (b *EvalGenerationBuilder) WithQuery(query string) *EvalGenerationBuilder {
	b.evalState.InputFields["query"] = true
	b.evalState.InputFields["input"] = true

	if b.gen.Input == nil {
		b.gen.Input = &StandardEvalInput{Query: query}
	} else if sei, ok := b.gen.Input.(*StandardEvalInput); ok {
		sei.Query = query
	} else {
		b.gen.Input = map[string]any{
			"query":    query,
			"input":    query,
			"_wrapped": b.gen.Input,
		}
	}
	return b
}

// WithContext sets the context/retrieved documents for evaluation.
func (b *EvalGenerationBuilder) WithContext(context ...string) *EvalGenerationBuilder {
	b.evalState.HasContext = true
	b.evalState.InputFields["context"] = true

	if b.gen.Input == nil {
		b.gen.Input = &StandardEvalInput{Context: context}
	} else if sei, ok := b.gen.Input.(*StandardEvalInput); ok {
		sei.Context = context
	} else if m, ok := b.gen.Input.(map[string]any); ok {
		m["context"] = context
	}
	return b
}

// WithGroundTruth sets the expected answer for evaluation.
func (b *EvalGenerationBuilder) WithGroundTruth(groundTruth string) *EvalGenerationBuilder {
	b.evalState.HasGroundTruth = true
	b.evalState.InputFields["ground_truth"] = true

	if b.gen.Input == nil {
		b.gen.Input = &StandardEvalInput{GroundTruth: groundTruth}
	} else if sei, ok := b.gen.Input.(*StandardEvalInput); ok {
		sei.GroundTruth = groundTruth
	} else if m, ok := b.gen.Input.(map[string]any); ok {
		m["ground_truth"] = groundTruth
	}
	return b
}

// WithSystemPrompt sets the system prompt for the generation.
func (b *EvalGenerationBuilder) WithSystemPrompt(systemPrompt string) *EvalGenerationBuilder {
	b.evalState.InputFields["system_prompt"] = true

	if b.gen.Input == nil {
		b.gen.Input = &StandardEvalInput{SystemPrompt: systemPrompt}
	} else if sei, ok := b.gen.Input.(*StandardEvalInput); ok {
		sei.SystemPrompt = systemPrompt
	} else if m, ok := b.gen.Input.(map[string]any); ok {
		m["system_prompt"] = systemPrompt
	}
	return b
}

// WithMessages sets the chat messages for the generation.
func (b *EvalGenerationBuilder) WithMessages(messages []map[string]string) *EvalGenerationBuilder {
	b.evalState.InputFields["messages"] = true

	if b.gen.Input == nil {
		b.gen.Input = &StandardEvalInput{Messages: messages}
	} else if sei, ok := b.gen.Input.(*StandardEvalInput); ok {
		sei.Messages = messages
	} else if m, ok := b.gen.Input.(map[string]any); ok {
		m["messages"] = messages
	}
	return b
}

// ID sets the generation ID.
func (b *EvalGenerationBuilder) ID(id string) *EvalGenerationBuilder {
	b.GenerationBuilder.ID(id)
	return b
}

// Name sets the generation name.
func (b *EvalGenerationBuilder) Name(name string) *EvalGenerationBuilder {
	b.GenerationBuilder.Name(name)
	return b
}

// Model sets the model name.
func (b *EvalGenerationBuilder) Model(model string) *EvalGenerationBuilder {
	b.GenerationBuilder.Model(model)
	return b
}

// ModelParameters sets the model parameters.
func (b *EvalGenerationBuilder) ModelParameters(params Metadata) *EvalGenerationBuilder {
	b.GenerationBuilder.ModelParameters(params)
	return b
}

// Input sets the generation input.
func (b *EvalGenerationBuilder) Input(input any) *EvalGenerationBuilder {
	b.GenerationBuilder.Input(input)
	b.evalState.UpdateFromInput(input)
	return b
}

// Output sets the generation output.
func (b *EvalGenerationBuilder) Output(output any) *EvalGenerationBuilder {
	b.GenerationBuilder.Output(output)
	b.evalState.UpdateFromOutput(output)
	return b
}

// Metadata sets the generation metadata.
func (b *EvalGenerationBuilder) Metadata(metadata Metadata) *EvalGenerationBuilder {
	b.GenerationBuilder.Metadata(metadata)
	return b
}

// Level sets the observation level.
func (b *EvalGenerationBuilder) Level(level ObservationLevel) *EvalGenerationBuilder {
	b.GenerationBuilder.Level(level)
	return b
}

// PromptName sets the prompt name.
func (b *EvalGenerationBuilder) PromptName(name string) *EvalGenerationBuilder {
	b.GenerationBuilder.PromptName(name)
	return b
}

// PromptVersion sets the prompt version.
func (b *EvalGenerationBuilder) PromptVersion(version int) *EvalGenerationBuilder {
	b.GenerationBuilder.PromptVersion(version)
	return b
}

// Environment sets the environment.
func (b *EvalGenerationBuilder) Environment(env string) *EvalGenerationBuilder {
	b.GenerationBuilder.Environment(env)
	return b
}

// Create creates the evaluation-aware generation.
func (b *EvalGenerationBuilder) Create(ctx context.Context) (*EvalGenerationContext, error) {
	if b.evalConfig != nil && b.evalConfig.Mode != EvaluationModeOff {
		b.applyEvalTransformations()
	}

	genCtx, err := b.GenerationBuilder.Create(ctx)
	if err != nil {
		return nil, err
	}

	return &EvalGenerationContext{
		GenerationContext: genCtx,
		evalState:         b.evalState,
		evalConfig:        b.evalConfig,
	}, nil
}

// applyEvalTransformations applies evaluation-specific transformations.
func (b *EvalGenerationBuilder) applyEvalTransformations() {
	config := b.evalConfig

	if config.FlattenInput && b.gen.Input != nil {
		b.gen.Input = prepareInputForEval(b.gen.Input, config)
	}

	if config.FlattenOutput && b.gen.Output != nil {
		b.gen.Output = prepareOutputForEval(b.gen.Output, config)
	}

	if config.IncludeMetadata {
		evalMeta := b.evalState.BuildMetadata().BuildAsMap()
		b.gen.Metadata = mergeMetadata(b.gen.Metadata, evalMeta)
	}
}

// EvalGenerationContext provides context for an evaluation-aware generation.
type EvalGenerationContext struct {
	*GenerationContext
	evalState  *EvalState
	evalConfig *EvaluationConfig
}

// GetEvalState returns the current evaluation state.
func (g *EvalGenerationContext) GetEvalState() *EvalState {
	return g.evalState
}

// IsEvalReady returns true if the generation is ready for evaluation.
func (g *EvalGenerationContext) IsEvalReady() bool {
	return g.evalState.IsReady()
}

// GetCompatibleEvaluators returns evaluators compatible with current data.
func (g *EvalGenerationContext) GetCompatibleEvaluators() []EvaluatorType {
	return g.evalState.GetCompatibleEvaluators()
}

// GetMissingFields returns fields still required for evaluation.
func (g *EvalGenerationContext) GetMissingFields() []string {
	return g.evalState.GetMissingFields()
}

// ValidateForEvaluator validates that this generation has required fields.
func (g *EvalGenerationContext) ValidateForEvaluator(evaluator EvaluatorType) error {
	allFields := g.evalState.AllFields()
	required := evaluator.GetRequiredFields()

	var missing []string
	for _, f := range required {
		if !allFields[f] && !allFields[pkgeval.FieldAlias(f)] {
			missing = append(missing, f)
		}
	}

	if len(missing) > 0 {
		return NewValidationError("evaluation",
			"missing required fields for "+string(evaluator)+": "+joinStrings(missing, ", "))
	}

	return nil
}

// CompleteWithEvaluation ends the generation with evaluation-structured output.
func (g *EvalGenerationContext) CompleteWithEvaluation(ctx context.Context, result *EvalGenerationResult) EndResult {
	g.evalState.UpdateFromOutput(result)

	update := g.Update()

	if result.Output != "" {
		outputData := prepareOutputForEval(result.ToStandardOutput(), g.evalConfig)
		update.Output(outputData)
	}

	if result.InputTokens > 0 || result.OutputTokens > 0 {
		update.UsageTokens(result.InputTokens, result.OutputTokens)
	}

	if !result.CompletionTime.IsZero() {
		update.CompletionStartTime(result.CompletionTime)
	}

	if g.evalConfig != nil && g.evalConfig.IncludeMetadata {
		evalMeta := g.evalState.BuildMetadata().BuildAsMap()
		update.Metadata(evalMeta)
	}

	update.EndTime(time.Now())

	err := update.Apply(ctx)
	return EndResult{Error: err}
}

// ============================================================================
// Eval Span Builder (Client-dependent)
// ============================================================================

// EvalSpanBuilder provides a fluent interface for creating evaluation-aware spans.
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
		evalConfig:  t.client.rootConfig.EvaluationConfig,
	}
}

// NewRetrievalSpan creates a span specifically for retrieval operations.
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

	if b.span.Metadata == nil {
		b.span.Metadata = make(Metadata)
	}
	if b.spanType != "" {
		b.span.Metadata["_eval_span_type"] = string(b.spanType)
	}

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
func (s *EvalSpanContext) WithContext(ctx context.Context, documents ...string) *EvalSpanContext {
	s.evalState.HasContext = true
	s.evalState.OutputFields["context"] = true

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
