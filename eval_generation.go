package langfuse

import (
	"context"
	"time"
)

// EvalGenerationBuilder provides a fluent interface for creating evaluation-aware generations.
// It extends GenerationBuilder with evaluation-specific methods and automatic data structuring.
//
// Example:
//
//	gen, err := trace.NewEvalGeneration().
//	    Name("llm-response").
//	    Model("gpt-4").
//	    ForEvaluator(langfuse.EvaluatorFaithfulness).
//	    WithContext(retrievedChunks...).
//	    WithQuery(userQuery).
//	    Create(ctx)
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
		evalConfig:        t.client.config.EvaluationConfig,
	}
}

// ForEvaluator specifies which evaluator(s) this generation should be optimized for.
// The builder will track required fields and validate before creation.
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
// This is captured separately for evaluation metadata.
func (b *EvalGenerationBuilder) WithQuery(query string) *EvalGenerationBuilder {
	b.evalState.InputFields["query"] = true
	b.evalState.InputFields["input"] = true

	// Build or update input
	if b.gen.Input == nil {
		b.gen.Input = &StandardEvalInput{Query: query}
	} else if sei, ok := b.gen.Input.(*StandardEvalInput); ok {
		sei.Query = query
	} else {
		// Wrap existing input
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

	// Build or update input
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

	// Build or update input
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

// Input sets the generation input (standard method).
func (b *EvalGenerationBuilder) Input(input any) *EvalGenerationBuilder {
	b.GenerationBuilder.Input(input)
	b.evalState.UpdateFromInput(input)
	return b
}

// Output sets the generation output (standard method).
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
	// Apply evaluation transformations if config is set
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

	// Flatten input if configured
	if config.FlattenInput && b.gen.Input != nil {
		b.gen.Input = prepareInputForEval(b.gen.Input, config)
	}

	// Flatten output if configured
	if config.FlattenOutput && b.gen.Output != nil {
		b.gen.Output = prepareOutputForEval(b.gen.Output, config)
	}

	// Add evaluation metadata if configured
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
		if !allFields[f] && !allFields[fieldAlias(f)] {
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
	// Update eval state
	g.evalState.UpdateFromOutput(result)

	// Build update
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

	// Apply evaluation metadata
	if g.evalConfig != nil && g.evalConfig.IncludeMetadata {
		evalMeta := g.evalState.BuildMetadata().BuildAsMap()
		update.Metadata(evalMeta)
	}

	update.EndTime(time.Now())

	err := update.Apply(ctx)
	return EndResult{Error: err}
}

// EvalGenerationResult contains the result of an LLM generation for evaluation.
type EvalGenerationResult struct {
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

// ToStandardOutput converts the result to a StandardEvalOutput.
func (r *EvalGenerationResult) ToStandardOutput() *StandardEvalOutput {
	return &StandardEvalOutput{
		Output:     r.Output,
		Citations:  r.Citations,
		Confidence: r.Confidence,
		Reasoning:  r.Reasoning,
		ToolCalls:  r.ToolCalls,
	}
}

// EvalFields implements EvalOutput for EvalGenerationResult.
func (r *EvalGenerationResult) EvalFields() map[string]any {
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
