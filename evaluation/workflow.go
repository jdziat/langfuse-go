package evaluation

import (
	"context"
	"fmt"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
)

// WorkflowBuilder provides a high-level interface for creating evaluation-ready
// traces that guide users through the correct data structure for LLM-as-a-Judge.
//
// Example:
//
//	workflow := evaluation.NewWorkflow(client, langfuse.WorkflowRAG).
//	    Name("document-qa").
//	    UserID("user-123")
//
//	// Record the user query
//	workflow.WithQuery("What are Go's concurrency features?")
//
//	// Record retrieval step
//	docs := vectorDB.Search(query)
//	workflow.AddRetrieval(ctx, "vector-search", docs)
//
//	// Record generation
//	response := llm.Complete(prompt)
//	workflow.AddGeneration(ctx, "gpt-4", response, usage)
//
//	// Complete the workflow
//	trace, err := workflow.Complete(ctx)
type WorkflowBuilder struct {
	client       *langfuse.Client
	workflowType langfuse.WorkflowType
	name         string
	userID       string
	sessionID    string
	tags         []string
	metadata     langfuse.Metadata
	environment  string

	// Evaluation data
	query       string
	context     []string
	groundTruth string
	output      string

	// Internal state
	trace     *langfuse.TraceContext
	evalState *langfuse.EvalState
	startTime time.Time
}

// NewWorkflow creates a new workflow builder for the specified workflow type.
func NewWorkflow(client *langfuse.Client, workflowType langfuse.WorkflowType) *WorkflowBuilder {
	return &WorkflowBuilder{
		client:       client,
		workflowType: workflowType,
		evalState:    langfuse.NewEvalState(),
		startTime:    time.Now(),
	}
}

// Name sets the workflow/trace name.
func (w *WorkflowBuilder) Name(name string) *WorkflowBuilder {
	w.name = name
	return w
}

// UserID sets the user ID.
func (w *WorkflowBuilder) UserID(userID string) *WorkflowBuilder {
	w.userID = userID
	return w
}

// SessionID sets the session ID.
func (w *WorkflowBuilder) SessionID(sessionID string) *WorkflowBuilder {
	w.sessionID = sessionID
	return w
}

// Tags sets the trace tags.
func (w *WorkflowBuilder) Tags(tags ...string) *WorkflowBuilder {
	w.tags = append(w.tags, tags...)
	return w
}

// Metadata sets additional metadata.
func (w *WorkflowBuilder) Metadata(metadata langfuse.Metadata) *WorkflowBuilder {
	w.metadata = metadata
	return w
}

// Environment sets the environment.
func (w *WorkflowBuilder) Environment(env string) *WorkflowBuilder {
	w.environment = env
	return w
}

// WithQuery sets the user query/question.
func (w *WorkflowBuilder) WithQuery(query string) *WorkflowBuilder {
	w.query = query
	w.evalState.InputFields["query"] = true
	w.evalState.InputFields["input"] = true
	return w
}

// WithContext sets the retrieved context/documents.
func (w *WorkflowBuilder) WithContext(documents ...string) *WorkflowBuilder {
	w.context = documents
	w.evalState.HasContext = true
	w.evalState.InputFields["context"] = true
	return w
}

// WithGroundTruth sets the expected correct answer for evaluation.
func (w *WorkflowBuilder) WithGroundTruth(groundTruth string) *WorkflowBuilder {
	w.groundTruth = groundTruth
	w.evalState.HasGroundTruth = true
	w.evalState.InputFields["ground_truth"] = true
	return w
}

// WithOutput sets the final output (if already available).
func (w *WorkflowBuilder) WithOutput(output string) *WorkflowBuilder {
	w.output = output
	w.evalState.HasOutput = true
	w.evalState.OutputFields["output"] = true
	return w
}

// Start creates the trace and prepares for step recording.
// This is called automatically by Complete() if not called earlier.
func (w *WorkflowBuilder) Start(ctx context.Context) error {
	if w.trace != nil {
		return nil // Already started
	}

	// Build input structure
	input := w.buildInput()

	// Create trace
	builder := w.client.NewTrace().Name(w.name)

	if w.userID != "" {
		builder.UserID(w.userID)
	}
	if w.sessionID != "" {
		builder.SessionID(w.sessionID)
	}
	if w.environment != "" {
		builder.Environment(w.environment)
	}

	// Merge tags
	allTags := append(w.tags, langfuse.EvalTagForWorkflow(w.workflowType))
	builder.Tags(allTags)

	// Merge metadata with eval metadata
	w.evalState.WorkflowType = w.workflowType
	evalMeta := w.evalState.BuildMetadata().BuildAsMap()
	allMetadata := langfuse.Metadata{}
	for k, v := range w.metadata {
		allMetadata[k] = v
	}
	for k, v := range evalMeta {
		allMetadata[k] = v
	}
	builder.Metadata(allMetadata)

	builder.Input(input)

	trace, err := builder.Create(ctx)
	if err != nil {
		return err
	}

	w.trace = trace
	return nil
}

// buildInput builds the structured input based on current state.
func (w *WorkflowBuilder) buildInput() map[string]any {
	input := make(map[string]any)

	if w.query != "" {
		input["query"] = w.query
		input["input"] = w.query // Alias for compatibility
	}
	if len(w.context) > 0 {
		input["context"] = w.context
	}
	if w.groundTruth != "" {
		input["ground_truth"] = w.groundTruth
	}

	return input
}

// buildOutput builds the structured output based on current state.
func (w *WorkflowBuilder) buildOutput() map[string]any {
	output := make(map[string]any)

	if w.output != "" {
		output["output"] = w.output
	}

	return output
}

// AddRetrieval adds a retrieval step to the workflow.
func (w *WorkflowBuilder) AddRetrieval(ctx context.Context, name string, documents []string) error {
	if err := w.Start(ctx); err != nil {
		return err
	}

	// Update context
	w.context = documents
	w.evalState.HasContext = true
	w.evalState.InputFields["context"] = true

	// Create retrieval span
	span, err := w.trace.NewRetrievalSpan().
		Name(name).
		WithQuery(w.query).
		Create(ctx)
	if err != nil {
		return err
	}

	// End with context
	return span.EndWithContext(ctx, documents...)
}

// AddGeneration adds an LLM generation step to the workflow.
func (w *WorkflowBuilder) AddGeneration(ctx context.Context, model, output string, inputTokens, outputTokens int) error {
	if err := w.Start(ctx); err != nil {
		return err
	}

	// Update output
	w.output = output
	w.evalState.HasOutput = true
	w.evalState.OutputFields["output"] = true

	// Create generation
	gen, err := w.trace.NewEvalGeneration().
		Name("llm-response").
		Model(model).
		WithQuery(w.query).
		Create(ctx)
	if err != nil {
		return err
	}

	// Complete with evaluation result
	result := &langfuse.EvalGenerationResult{
		Output:       output,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Model:        model,
	}

	endResult := gen.CompleteWithEvaluation(ctx, result)
	return endResult.Error
}

// AddGenerationResult adds an LLM generation step with full result data.
func (w *WorkflowBuilder) AddGenerationResult(ctx context.Context, model string, result *langfuse.EvalGenerationResult) error {
	if err := w.Start(ctx); err != nil {
		return err
	}

	// Update output
	w.output = result.Output
	w.evalState.HasOutput = true
	w.evalState.OutputFields["output"] = true

	// Create generation
	gen, err := w.trace.NewEvalGeneration().
		Name("llm-response").
		Model(model).
		WithQuery(w.query).
		Create(ctx)
	if err != nil {
		return err
	}

	// Add context if available
	if len(w.context) > 0 {
		gen.GetEvalState().HasContext = true
	}

	endResult := gen.CompleteWithEvaluation(ctx, result)
	return endResult.Error
}

// Validate checks if the workflow has all required fields.
func (w *WorkflowBuilder) Validate() error {
	required := w.workflowType.GetRequiredFields()
	allFields := w.evalState.AllFields()

	var missing []string
	for _, f := range required {
		if !allFields[f] && !allFields[fieldAlias(f)] {
			missing = append(missing, f)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("workflow %s missing required fields: %v", w.workflowType, missing)
	}

	return nil
}

// IsReady returns true if the workflow is ready for evaluation.
func (w *WorkflowBuilder) IsReady() bool {
	return w.Validate() == nil
}

// GetCompatibleEvaluators returns evaluators compatible with current data.
func (w *WorkflowBuilder) GetCompatibleEvaluators() []langfuse.EvaluatorType {
	return w.evalState.GetCompatibleEvaluators()
}

// Complete finalizes the workflow and updates the trace with output.
func (w *WorkflowBuilder) Complete(ctx context.Context) (*WorkflowResult, error) {
	if err := w.Start(ctx); err != nil {
		return nil, err
	}

	// Build output
	output := w.buildOutput()

	// Update eval state for final metadata
	w.evalState.WorkflowType = w.workflowType

	// Update trace with output and final metadata
	evalMeta := w.evalState.BuildMetadata()
	evalTags := evalMeta.GenerateEvalTags()

	err := w.trace.Update().
		Output(output).
		Metadata(evalMeta.BuildAsMap()).
		Tags(langfuse.NewTags().Add(evalTags...).Build()).
		Apply(ctx)

	if err != nil {
		return nil, err
	}

	return &WorkflowResult{
		Trace:                w.trace,
		IsReady:              w.evalState.IsReady(),
		CompatibleEvaluators: w.evalState.GetCompatibleEvaluators(),
		MissingFields:        w.evalState.GetMissingFields(),
		Duration:             time.Since(w.startTime),
	}, nil
}

// WorkflowResult contains the result of completing a workflow.
type WorkflowResult struct {
	// Trace is the completed trace context.
	Trace *langfuse.TraceContext

	// IsReady indicates if the trace is ready for evaluation.
	IsReady bool

	// CompatibleEvaluators lists evaluators that can be used.
	CompatibleEvaluators []langfuse.EvaluatorType

	// MissingFields lists fields that are still missing.
	MissingFields []string

	// Duration is the total workflow duration.
	Duration time.Duration
}

// ID returns the trace ID.
func (r *WorkflowResult) ID() string {
	return r.Trace.ID()
}

// fieldAlias returns an alias for a field name.
func fieldAlias(field string) string {
	aliases := map[string]string{
		"query":              "input",
		"input":              "query",
		"output":             "response",
		"response":           "output",
		"context":            "retrieved_contexts",
		"retrieved_contexts": "context",
	}
	return aliases[field]
}
