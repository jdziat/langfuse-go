package langfuse

import (
	"context"
	"time"
)

// GenerationBuilder provides a fluent interface for creating generations.
//
// GenerationBuilder is NOT safe for concurrent use. Each builder instance
// should be created, configured, and used within a single goroutine. All
// setter methods modify the builder in place and return the same pointer
// for method chaining.
//
// Example:
//
//	gen, err := trace.Generation().
//	    Name("llm-call").
//	    Model("gpt-4").
//	    Input(prompt).
//	    Create()
type GenerationBuilder struct {
	ctx *TraceContext
	gen *createGenerationEvent
}

// ID sets the generation ID.
func (b *GenerationBuilder) ID(id string) *GenerationBuilder {
	b.gen.ID = id
	return b
}

// Name sets the generation name.
func (b *GenerationBuilder) Name(name string) *GenerationBuilder {
	b.gen.Name = name
	return b
}

// StartTime sets the start time.
func (b *GenerationBuilder) StartTime(t time.Time) *GenerationBuilder {
	b.gen.StartTime = Time{Time: t}
	return b
}

// EndTime sets the end time.
func (b *GenerationBuilder) EndTime(t time.Time) *GenerationBuilder {
	b.gen.EndTime = Time{Time: t}
	return b
}

// CompletionStartTime sets when the completion started.
func (b *GenerationBuilder) CompletionStartTime(t time.Time) *GenerationBuilder {
	b.gen.CompletionStartTime = Time{Time: t}
	return b
}

// Input sets the input.
func (b *GenerationBuilder) Input(input any) *GenerationBuilder {
	b.gen.Input = input
	return b
}

// Output sets the output.
func (b *GenerationBuilder) Output(output any) *GenerationBuilder {
	b.gen.Output = output
	return b
}

// Metadata sets the metadata.
func (b *GenerationBuilder) Metadata(metadata Metadata) *GenerationBuilder {
	b.gen.Metadata = metadata
	return b
}

// Level sets the observation level.
func (b *GenerationBuilder) Level(level ObservationLevel) *GenerationBuilder {
	b.gen.Level = level
	return b
}

// StatusMessage sets the status message.
func (b *GenerationBuilder) StatusMessage(msg string) *GenerationBuilder {
	b.gen.StatusMessage = msg
	return b
}

// ParentObservationID sets the parent observation ID.
func (b *GenerationBuilder) ParentObservationID(id string) *GenerationBuilder {
	b.gen.ParentObservationID = id
	return b
}

// ParentID is an alias for ParentObservationID.
// It sets the parent observation ID for this generation.
func (b *GenerationBuilder) ParentID(id string) *GenerationBuilder {
	return b.ParentObservationID(id)
}

// Version sets the version.
func (b *GenerationBuilder) Version(version string) *GenerationBuilder {
	b.gen.Version = version
	return b
}

// Model sets the model name.
func (b *GenerationBuilder) Model(model string) *GenerationBuilder {
	b.gen.Model = model
	return b
}

// ModelParameters sets the model parameters.
func (b *GenerationBuilder) ModelParameters(params Metadata) *GenerationBuilder {
	b.gen.ModelParameters = params
	return b
}

// Usage sets the token usage.
func (b *GenerationBuilder) Usage(usage *Usage) *GenerationBuilder {
	b.gen.Usage = usage
	return b
}

// UsageTokens sets token counts.
func (b *GenerationBuilder) UsageTokens(input, output int) *GenerationBuilder {
	b.gen.Usage = &Usage{
		Input:  input,
		Output: output,
		Total:  input + output,
	}
	return b
}

// PromptName sets the prompt name.
func (b *GenerationBuilder) PromptName(name string) *GenerationBuilder {
	b.gen.PromptName = name
	return b
}

// PromptVersion sets the prompt version.
func (b *GenerationBuilder) PromptVersion(version int) *GenerationBuilder {
	b.gen.PromptVersion = version
	return b
}

// Environment sets the environment.
func (b *GenerationBuilder) Environment(env string) *GenerationBuilder {
	b.gen.Environment = env
	return b
}

// Clone creates a deep copy of the GenerationBuilder with a new ID and timestamp.
// This is useful for creating multiple similar generations from a template.
//
// Example:
//
//	template := trace.Generation().
//	    Model("gpt-4").
//	    ModelParameters(langfuse.Metadata{"temperature": 0.7}).
//	    Environment("production")
//
//	gen1, _ := template.Clone().Name("query-1").Input(prompt1).Create(ctx)
//	gen2, _ := template.Clone().Name("query-2").Input(prompt2).Create(ctx)
func (b *GenerationBuilder) Clone() *GenerationBuilder {
	// Deep copy metadata map
	var metadata Metadata
	if b.gen.Metadata != nil {
		metadata = make(Metadata, len(b.gen.Metadata))
		for k, v := range b.gen.Metadata {
			metadata[k] = v
		}
	}

	// Deep copy model parameters
	var modelParams Metadata
	if b.gen.ModelParameters != nil {
		modelParams = make(Metadata, len(b.gen.ModelParameters))
		for k, v := range b.gen.ModelParameters {
			modelParams[k] = v
		}
	}

	// Deep copy usage if present
	var usage *Usage
	if b.gen.Usage != nil {
		usage = &Usage{
			Input:      b.gen.Usage.Input,
			Output:     b.gen.Usage.Output,
			Total:      b.gen.Usage.Total,
			Unit:       b.gen.Usage.Unit,
			InputCost:  b.gen.Usage.InputCost,
			OutputCost: b.gen.Usage.OutputCost,
			TotalCost:  b.gen.Usage.TotalCost,
		}
	}

	return &GenerationBuilder{
		ctx: b.ctx,
		gen: &createGenerationEvent{
			ID:                  generateID(), // New ID for the clone
			TraceID:             b.gen.TraceID,
			Name:                b.gen.Name,
			StartTime:           Now(), // Fresh timestamp
			EndTime:             b.gen.EndTime,
			CompletionStartTime: b.gen.CompletionStartTime,
			Metadata:            metadata,
			Level:               b.gen.Level,
			StatusMessage:       b.gen.StatusMessage,
			ParentObservationID: b.gen.ParentObservationID,
			Version:             b.gen.Version,
			Input:               b.gen.Input,
			Output:              b.gen.Output,
			Environment:         b.gen.Environment,
			Model:               b.gen.Model,
			ModelParameters:     modelParams,
			Usage:               usage,
			PromptName:          b.gen.PromptName,
			PromptVersion:       b.gen.PromptVersion,
		},
	}
}

// Validate validates the generation builder configuration.
func (b *GenerationBuilder) Validate() error {
	if b.gen.ID == "" {
		return NewValidationError("id", "generation ID cannot be empty")
	}
	if b.gen.TraceID == "" {
		return NewValidationError("traceId", "trace ID cannot be empty")
	}
	return nil
}

// Create creates the generation and returns a GenerationContext.
func (b *GenerationBuilder) Create(ctx context.Context) (*GenerationContext, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}

	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeGenerationCreate,
		Timestamp: Now(),
		Body:      b.gen,
	}

	if err := b.ctx.client.queueEvent(ctx, event); err != nil {
		return nil, err
	}

	return &GenerationContext{
		TraceContext: b.ctx,
		genID:        b.gen.ID,
	}, nil
}

// GenerationContext provides context for a generation.
//
// GenerationContext is safe for concurrent use. You can create scores and
// access properties concurrently. However, update builders created from
// GenerationContext are NOT safe for concurrent use.
type GenerationContext struct {
	*TraceContext
	genID string
}

// GenerationID returns the generation ID.
func (g *GenerationContext) GenerationID() string {
	return g.genID
}

// ID returns the generation ID. This method satisfies the Observer interface.
func (g *GenerationContext) ID() string {
	return g.genID
}

// Update updates the generation.
func (g *GenerationContext) Update() *GenerationUpdateBuilder {
	return &GenerationUpdateBuilder{
		ctx: g,
		update: &updateGenerationEvent{
			ID:      g.genID,
			TraceID: g.traceID,
		},
	}
}

// End ends the generation with the current time.
func (g *GenerationContext) End(ctx context.Context) error {
	return g.Update().EndTime(time.Now()).Apply(ctx)
}

// EndWithOutput ends the generation with output and the current time.
func (g *GenerationContext) EndWithOutput(ctx context.Context, output any) error {
	return g.Update().Output(output).EndTime(time.Now()).Apply(ctx)
}

// EndWithUsage ends the generation with output, usage, and the current time.
func (g *GenerationContext) EndWithUsage(ctx context.Context, output any, inputTokens, outputTokens int) error {
	return g.Update().
		Output(output).
		UsageTokens(inputTokens, outputTokens).
		EndTime(time.Now()).
		Apply(ctx)
}

// EndWith ends the generation with the provided options.
// This provides a consistent, flexible API for ending observations.
//
// Example:
//
//	result := gen.EndWith(ctx,
//	    WithOutput(response),
//	    WithUsage(100, 50),
//	    WithCompletionStart(completionStartedAt),
//	)
//	if !result.Ok() {
//	    log.Printf("Failed to end generation: %v", result.Error)
//	}
//
//	// With error handling:
//	result := gen.EndWith(ctx, WithError(err))
func (g *GenerationContext) EndWith(ctx context.Context, opts ...EndOption) EndResult {
	cfg := &endConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	// Determine end time
	endTime := time.Now()
	if cfg.hasEndTime {
		endTime = cfg.endTime
	}

	// Build the update
	update := g.Update().EndTime(endTime)

	if cfg.output != nil {
		update.Output(cfg.output)
	}
	if cfg.hasUsage {
		update.UsageTokens(cfg.inputTokens, cfg.outputTokens)
	}
	if cfg.metadata != nil {
		update.Metadata(cfg.metadata)
	}
	if cfg.hasLevel {
		update.Level(cfg.level)
	}
	if cfg.statusMessage != "" {
		update.StatusMessage(cfg.statusMessage)
	}
	if cfg.hasCompletionStart {
		update.CompletionStartTime(cfg.completionStartTime)
	}

	err := update.Apply(ctx)

	return EndResult{
		Error: err,
	}
}

// NewScore creates a score builder for this generation (Advanced API).
// For the Simple API, use Score(ctx, name, value, ...opts).
func (g *GenerationContext) NewScore() *ScoreBuilder {
	builder := g.TraceContext.NewScore()
	builder.score.ObservationID = g.genID
	return builder
}

// ScoreNumeric adds a numeric score to this generation.
// This is a convenience method for the common case of adding a simple numeric score.
func (g *GenerationContext) ScoreNumeric(ctx context.Context, name string, value float64) error {
	return g.NewScore().Name(name).NumericValue(value).Create(ctx)
}

// ScoreCategorical adds a categorical score to this generation.
// This is a convenience method for the common case of adding a simple categorical score.
func (g *GenerationContext) ScoreCategorical(ctx context.Context, name string, value string) error {
	return g.NewScore().Name(name).CategoricalValue(value).Create(ctx)
}

// ScoreBoolean adds a boolean score to this generation.
// This is a convenience method for the common case of adding a simple boolean score.
func (g *GenerationContext) ScoreBoolean(ctx context.Context, name string, value bool) error {
	return g.NewScore().Name(name).BooleanValue(value).Create(ctx)
}

// NewSpan creates a child span builder under this generation (Advanced API).
// For the Simple API, use Span(ctx, name, ...opts).
func (g *GenerationContext) NewSpan() *SpanBuilder {
	builder := g.TraceContext.NewSpan()
	builder.span.ParentObservationID = g.genID
	return builder
}

// NewGeneration creates a child generation builder under this generation (Advanced API).
// For the Simple API, use Generation(ctx, name, ...opts).
func (g *GenerationContext) NewGeneration() *GenerationBuilder {
	builder := g.TraceContext.NewGeneration()
	builder.gen.ParentObservationID = g.genID
	return builder
}

// NewEvent creates a child event builder under this generation (Advanced API).
// For the Simple API, use Event(ctx, name, ...opts).
func (g *GenerationContext) NewEvent() *EventBuilder {
	builder := g.TraceContext.NewEvent()
	builder.event.ParentObservationID = g.genID
	return builder
}

// GenerationUpdateBuilder provides a fluent interface for updating generations.
//
// GenerationUpdateBuilder is NOT safe for concurrent use. Each builder
// instance should be created, configured, and used within a single goroutine.
type GenerationUpdateBuilder struct {
	ctx    *GenerationContext
	update *updateGenerationEvent
}

// Name sets the generation name.
func (b *GenerationUpdateBuilder) Name(name string) *GenerationUpdateBuilder {
	b.update.Name = name
	return b
}

// EndTime sets the end time.
func (b *GenerationUpdateBuilder) EndTime(t time.Time) *GenerationUpdateBuilder {
	b.update.EndTime = Time{Time: t}
	return b
}

// CompletionStartTime sets when the completion started.
func (b *GenerationUpdateBuilder) CompletionStartTime(t time.Time) *GenerationUpdateBuilder {
	b.update.CompletionStartTime = Time{Time: t}
	return b
}

// Input sets the input.
func (b *GenerationUpdateBuilder) Input(input any) *GenerationUpdateBuilder {
	b.update.Input = input
	return b
}

// Output sets the output.
func (b *GenerationUpdateBuilder) Output(output any) *GenerationUpdateBuilder {
	b.update.Output = output
	return b
}

// Metadata sets the metadata.
func (b *GenerationUpdateBuilder) Metadata(metadata Metadata) *GenerationUpdateBuilder {
	b.update.Metadata = metadata
	return b
}

// Level sets the observation level.
func (b *GenerationUpdateBuilder) Level(level ObservationLevel) *GenerationUpdateBuilder {
	b.update.Level = level
	return b
}

// StatusMessage sets the status message.
func (b *GenerationUpdateBuilder) StatusMessage(msg string) *GenerationUpdateBuilder {
	b.update.StatusMessage = msg
	return b
}

// Model sets the model name.
func (b *GenerationUpdateBuilder) Model(model string) *GenerationUpdateBuilder {
	b.update.Model = model
	return b
}

// ModelParameters sets the model parameters.
func (b *GenerationUpdateBuilder) ModelParameters(params Metadata) *GenerationUpdateBuilder {
	b.update.ModelParameters = params
	return b
}

// Usage sets the token usage.
func (b *GenerationUpdateBuilder) Usage(usage *Usage) *GenerationUpdateBuilder {
	b.update.Usage = usage
	return b
}

// UsageTokens sets token counts.
func (b *GenerationUpdateBuilder) UsageTokens(input, output int) *GenerationUpdateBuilder {
	b.update.Usage = &Usage{
		Input:  input,
		Output: output,
		Total:  input + output,
	}
	return b
}

// Apply applies the update.
func (b *GenerationUpdateBuilder) Apply(ctx context.Context) error {
	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeGenerationUpdate,
		Timestamp: Now(),
		Body:      b.update,
	}

	return b.ctx.client.queueEvent(ctx, event)
}
