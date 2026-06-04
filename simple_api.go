package langfuse

import (
	"context"
	"time"
)

// ============================================================================
// Type Aliases for Convenience
// ============================================================================

// M is a convenient alias for Metadata.
// Use this for inline metadata construction.
//
// Example:
//
//	trace, _ := client.Trace(ctx, "request", WithMetadata(langfuse.M{
//	    "endpoint": "/api/chat",
//	    "method":   "POST",
//	}))
type M = Metadata

// Tags is a convenience function to create a slice of tags.
//
// Example:
//
//	trace, _ := client.Trace(ctx, "request", WithTags(langfuse.Tags("api", "v2", "production")...))
func Tags(tags ...string) []string {
	return tags
}

// ============================================================================
// Trace Options
// ============================================================================

// traceConfig holds the configuration for creating a trace via Simple API.
type traceConfig struct {
	id          string
	userID      string
	sessionID   string
	input       any
	output      any
	metadata    Metadata
	tags        []string
	release     string
	version     string
	public      bool
	hasPublic   bool
	environment string
}

// TraceOption configures a trace creation.
type TraceOption func(*traceConfig)

// WithTraceID sets a custom trace ID.
func WithTraceID(id string) TraceOption {
	return func(c *traceConfig) {
		c.id = id
	}
}

// WithUserID sets the user ID for a trace.
func WithUserID(userID string) TraceOption {
	return func(c *traceConfig) {
		c.userID = userID
	}
}

// WithSessionID sets the session ID for a trace.
func WithSessionID(sessionID string) TraceOption {
	return func(c *traceConfig) {
		c.sessionID = sessionID
	}
}

// WithInput sets the input for a trace, span, generation, or event.
func WithInput(input any) TraceOption {
	return func(c *traceConfig) {
		c.input = input
	}
}

// WithTraceOutput sets the output for a trace.
func WithTraceOutput(output any) TraceOption {
	return func(c *traceConfig) {
		c.output = output
	}
}

// WithMetadata sets metadata for a trace.
func WithMetadata(metadata Metadata) TraceOption {
	return func(c *traceConfig) {
		c.metadata = metadata
	}
}

// WithTags sets tags for a trace.
func WithTags(tags ...string) TraceOption {
	return func(c *traceConfig) {
		c.tags = tags
	}
}

// WithRelease sets the release version.
func WithRelease(release string) TraceOption {
	return func(c *traceConfig) {
		c.release = release
	}
}

// WithVersion sets the version.
func WithVersion(version string) TraceOption {
	return func(c *traceConfig) {
		c.version = version
	}
}

// WithPublic sets whether the trace is public.
func WithPublic(public bool) TraceOption {
	return func(c *traceConfig) {
		c.public = public
		c.hasPublic = true
	}
}

// WithEnvironment sets the environment for a trace.
func WithEnvironment(env string) TraceOption {
	return func(c *traceConfig) {
		c.environment = env
	}
}

// ============================================================================
// Span Options
// ============================================================================

// spanConfig holds the configuration for creating a span via Simple API.
type spanConfig struct {
	id            string
	input         any
	output        any
	metadata      Metadata
	level         ObservationLevel
	hasLevel      bool
	statusMessage string
	version       string
	environment   string
	startTime     time.Time
	hasStartTime  bool
	endTime       time.Time
	hasEndTime    bool
}

// SpanOption configures a span creation.
// This interface allows both function-based options and unified observation options.
type SpanOption interface {
	applySpan(*spanConfig)
}

// spanOptionFunc allows using functions as SpanOptions.
type spanOptionFunc func(*spanConfig)

func (f spanOptionFunc) applySpan(c *spanConfig) { f(c) }

// WithSpanID sets a custom span ID.
func WithSpanID(id string) SpanOption {
	return spanOptionFunc(func(c *spanConfig) {
		c.id = id
	})
}

// WithSpanInput sets the input for a span.
func WithSpanInput(input any) SpanOption {
	return spanOptionFunc(func(c *spanConfig) {
		c.input = input
	})
}

// WithSpanOutput sets the output for a span (useful for pre-completed spans).
func WithSpanOutput(output any) SpanOption {
	return spanOptionFunc(func(c *spanConfig) {
		c.output = output
	})
}

// WithSpanMetadata sets metadata for a span.
func WithSpanMetadata(metadata Metadata) SpanOption {
	return spanOptionFunc(func(c *spanConfig) {
		c.metadata = metadata
	})
}

// WithLevel sets the observation level.
func WithLevel(level ObservationLevel) SpanOption {
	return spanOptionFunc(func(c *spanConfig) {
		c.level = level
		c.hasLevel = true
	})
}

// WithSpanStatusMessage sets the status message for a span.
func WithSpanStatusMessage(msg string) SpanOption {
	return spanOptionFunc(func(c *spanConfig) {
		c.statusMessage = msg
	})
}

// WithSpanVersion sets the version for a span.
func WithSpanVersion(version string) SpanOption {
	return spanOptionFunc(func(c *spanConfig) {
		c.version = version
	})
}

// WithSpanEnvironment sets the environment for a span.
func WithSpanEnvironment(env string) SpanOption {
	return spanOptionFunc(func(c *spanConfig) {
		c.environment = env
	})
}

// WithStartTime sets a custom start time.
func WithStartTime(t time.Time) SpanOption {
	return spanOptionFunc(func(c *spanConfig) {
		c.startTime = t
		c.hasStartTime = true
	})
}

// WithSpanEndTime sets the end time for a pre-completed span.
func WithSpanEndTime(t time.Time) SpanOption {
	return spanOptionFunc(func(c *spanConfig) {
		c.endTime = t
		c.hasEndTime = true
	})
}

// ============================================================================
// Generation Options
// ============================================================================

// generationConfig holds the configuration for creating a generation via Simple API.
type generationConfig struct {
	id                  string
	model               string
	modelParameters     Metadata
	input               any
	output              any
	metadata            Metadata
	level               ObservationLevel
	hasLevel            bool
	statusMessage       string
	version             string
	environment         string
	usage               *Usage
	promptName          string
	promptVersion       int
	hasPromptVersion    bool
	startTime           time.Time
	hasStartTime        bool
	endTime             time.Time
	hasEndTime          bool
	completionStartTime time.Time
	hasCompletionStart  bool
}

// GenerationOption configures a generation creation.
// This interface allows both function-based options and unified observation options.
type GenerationOption interface {
	applyGeneration(*generationConfig)
}

// generationOptionFunc allows using functions as GenerationOptions.
type generationOptionFunc func(*generationConfig)

func (f generationOptionFunc) applyGeneration(c *generationConfig) { f(c) }

// WithGenerationID sets a custom generation ID.
func WithGenerationID(id string) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.id = id
	})
}

// WithModel sets the model name for a generation.
func WithModel(model string) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.model = model
	})
}

// WithModelParameters sets the model parameters.
func WithModelParameters(params Metadata) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.modelParameters = params
	})
}

// WithGenerationInput sets the input for a generation.
func WithGenerationInput(input any) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.input = input
	})
}

// WithGenerationOutput sets the output for a generation.
func WithGenerationOutput(output any) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.output = output
	})
}

// WithGenerationMetadata sets metadata for a generation.
func WithGenerationMetadata(metadata Metadata) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.metadata = metadata
	})
}

// WithGenerationLevel sets the observation level.
func WithGenerationLevel(level ObservationLevel) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.level = level
		c.hasLevel = true
	})
}

// WithGenerationStatusMessage sets the status message.
func WithGenerationStatusMessage(msg string) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.statusMessage = msg
	})
}

// WithGenerationVersion sets the version.
func WithGenerationVersion(version string) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.version = version
	})
}

// WithGenerationEnvironment sets the environment.
func WithGenerationEnvironment(env string) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.environment = env
	})
}

// WithTokenUsage sets token usage for a generation.
func WithTokenUsage(inputTokens, outputTokens int) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.usage = &Usage{
			Input:  inputTokens,
			Output: outputTokens,
			Total:  inputTokens + outputTokens,
		}
	})
}

// WithFullUsage sets complete usage information.
func WithFullUsage(usage *Usage) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.usage = usage
	})
}

// WithPromptName sets the prompt name.
func WithPromptName(name string) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.promptName = name
	})
}

// WithPromptVersion sets the prompt version.
func WithPromptVersion(version int) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.promptVersion = version
		c.hasPromptVersion = true
	})
}

// WithGenerationStartTime sets a custom start time.
func WithGenerationStartTime(t time.Time) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.startTime = t
		c.hasStartTime = true
	})
}

// WithGenerationEndTime sets the end time.
func WithGenerationEndTime(t time.Time) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.endTime = t
		c.hasEndTime = true
	})
}

// WithCompletionStartTime sets the completion start time (time to first token).
func WithCompletionStartTime(t time.Time) GenerationOption {
	return generationOptionFunc(func(c *generationConfig) {
		c.completionStartTime = t
		c.hasCompletionStart = true
	})
}

// ============================================================================
// Event Options
// ============================================================================

// eventConfig holds the configuration for creating an event via Simple API.
type eventConfig struct {
	id            string
	input         any
	output        any
	metadata      Metadata
	level         ObservationLevel
	hasLevel      bool
	statusMessage string
	version       string
	environment   string
	startTime     time.Time
	hasStartTime  bool
}

// EventOption configures an event creation.
// This interface allows both function-based options and unified observation options.
type EventOption interface {
	applyEvent(*eventConfig)
}

// eventOptionFunc allows using functions as EventOptions.
type eventOptionFunc func(*eventConfig)

func (f eventOptionFunc) applyEvent(c *eventConfig) { f(c) }

// WithEventID sets a custom event ID.
func WithEventID(id string) EventOption {
	return eventOptionFunc(func(c *eventConfig) {
		c.id = id
	})
}

// WithEventInput sets the input for an event.
func WithEventInput(input any) EventOption {
	return eventOptionFunc(func(c *eventConfig) {
		c.input = input
	})
}

// WithEventOutput sets the output for an event.
func WithEventOutput(output any) EventOption {
	return eventOptionFunc(func(c *eventConfig) {
		c.output = output
	})
}

// WithEventMetadata sets metadata for an event.
func WithEventMetadata(metadata Metadata) EventOption {
	return eventOptionFunc(func(c *eventConfig) {
		c.metadata = metadata
	})
}

// WithEventLevel sets the observation level.
func WithEventLevel(level ObservationLevel) EventOption {
	return eventOptionFunc(func(c *eventConfig) {
		c.level = level
		c.hasLevel = true
	})
}

// WithEventStatusMessage sets the status message.
func WithEventStatusMessage(msg string) EventOption {
	return eventOptionFunc(func(c *eventConfig) {
		c.statusMessage = msg
	})
}

// WithEventVersion sets the version.
func WithEventVersion(version string) EventOption {
	return eventOptionFunc(func(c *eventConfig) {
		c.version = version
	})
}

// WithEventEnvironment sets the environment.
func WithEventEnvironment(env string) EventOption {
	return eventOptionFunc(func(c *eventConfig) {
		c.environment = env
	})
}

// WithEventStartTime sets a custom start time.
func WithEventStartTime(t time.Time) EventOption {
	return eventOptionFunc(func(c *eventConfig) {
		c.startTime = t
		c.hasStartTime = true
	})
}

// ============================================================================
// Score Options
// ============================================================================

// scoreConfig holds the configuration for creating a score via Simple API.
type scoreConfig struct {
	id       string
	comment  string
	source   string
	configID string
}

// ScoreOption configures a score creation.
type ScoreOption func(*scoreConfig)

// WithScoreID sets a custom score ID.
func WithScoreID(id string) ScoreOption {
	return func(c *scoreConfig) {
		c.id = id
	}
}

// WithComment sets a comment for the score.
func WithComment(comment string) ScoreOption {
	return func(c *scoreConfig) {
		c.comment = comment
	}
}

// WithSource sets the source of the score.
func WithSource(source string) ScoreOption {
	return func(c *scoreConfig) {
		c.source = source
	}
}

// WithConfigID sets the config ID for the score.
func WithConfigID(configID string) ScoreOption {
	return func(c *scoreConfig) {
		c.configID = configID
	}
}

// ============================================================================
// Shared apply helpers
//
// The Simple API resolves each WithXxx option into a *Config struct and then
// applies it to the corresponding fluent builder. Every Span/Generation/Event/
// Score entry point (on Client, TraceContext, SpanContext, and GenerationContext)
// shares the same apply logic, so it lives in one place here. The per-context
// methods are thin wrappers that pick the right builder constructor and delegate
// to these helpers. This keeps the Simple API a strict convenience layer over the
// canonical fluent builders with a single internal apply path.
// ============================================================================

// applyTraceConfig copies a resolved traceConfig onto a TraceBuilder.
func applyTraceConfig(builder *TraceBuilder, cfg *traceConfig) {
	if cfg.id != "" {
		builder.ID(cfg.id)
	}
	if cfg.userID != "" {
		builder.UserID(cfg.userID)
	}
	if cfg.sessionID != "" {
		builder.SessionID(cfg.sessionID)
	}
	if cfg.input != nil {
		builder.Input(cfg.input)
	}
	if cfg.output != nil {
		builder.Output(cfg.output)
	}
	if cfg.metadata != nil {
		builder.Metadata(cfg.metadata)
	}
	if cfg.tags != nil {
		builder.Tags(cfg.tags)
	}
	if cfg.release != "" {
		builder.Release(cfg.release)
	}
	if cfg.version != "" {
		builder.Version(cfg.version)
	}
	if cfg.hasPublic {
		builder.Public(cfg.public)
	}
	if cfg.environment != "" {
		builder.Environment(cfg.environment)
	}
}

// applySpanConfig copies a resolved spanConfig onto a SpanBuilder.
func applySpanConfig(builder *SpanBuilder, cfg *spanConfig) {
	if cfg.id != "" {
		builder.ID(cfg.id)
	}
	if cfg.input != nil {
		builder.Input(cfg.input)
	}
	if cfg.output != nil {
		builder.Output(cfg.output)
	}
	if cfg.metadata != nil {
		builder.Metadata(cfg.metadata)
	}
	if cfg.hasLevel {
		builder.Level(cfg.level)
	}
	if cfg.statusMessage != "" {
		builder.StatusMessage(cfg.statusMessage)
	}
	if cfg.version != "" {
		builder.Version(cfg.version)
	}
	if cfg.environment != "" {
		builder.Environment(cfg.environment)
	}
	if cfg.hasStartTime {
		builder.StartTime(cfg.startTime)
	}
	if cfg.hasEndTime {
		builder.EndTime(cfg.endTime)
	}
}

// applyGenerationConfig copies a resolved generationConfig onto a GenerationBuilder.
func applyGenerationConfig(builder *GenerationBuilder, cfg *generationConfig) {
	if cfg.id != "" {
		builder.ID(cfg.id)
	}
	if cfg.model != "" {
		builder.Model(cfg.model)
	}
	if cfg.modelParameters != nil {
		builder.ModelParameters(cfg.modelParameters)
	}
	if cfg.input != nil {
		builder.Input(cfg.input)
	}
	if cfg.output != nil {
		builder.Output(cfg.output)
	}
	if cfg.metadata != nil {
		builder.Metadata(cfg.metadata)
	}
	if cfg.hasLevel {
		builder.Level(cfg.level)
	}
	if cfg.statusMessage != "" {
		builder.StatusMessage(cfg.statusMessage)
	}
	if cfg.version != "" {
		builder.Version(cfg.version)
	}
	if cfg.environment != "" {
		builder.Environment(cfg.environment)
	}
	if cfg.usage != nil {
		builder.Usage(cfg.usage)
	}
	if cfg.promptName != "" {
		builder.PromptName(cfg.promptName)
	}
	if cfg.hasPromptVersion {
		builder.PromptVersion(cfg.promptVersion)
	}
	if cfg.hasStartTime {
		builder.StartTime(cfg.startTime)
	}
	if cfg.hasEndTime {
		builder.EndTime(cfg.endTime)
	}
	if cfg.hasCompletionStart {
		builder.CompletionStartTime(cfg.completionStartTime)
	}
}

// applyEventConfig copies a resolved eventConfig onto an EventBuilder.
func applyEventConfig(builder *EventBuilder, cfg *eventConfig) {
	if cfg.id != "" {
		builder.ID(cfg.id)
	}
	if cfg.input != nil {
		builder.Input(cfg.input)
	}
	if cfg.output != nil {
		builder.Output(cfg.output)
	}
	if cfg.metadata != nil {
		builder.Metadata(cfg.metadata)
	}
	if cfg.hasLevel {
		builder.Level(cfg.level)
	}
	if cfg.statusMessage != "" {
		builder.StatusMessage(cfg.statusMessage)
	}
	if cfg.version != "" {
		builder.Version(cfg.version)
	}
	if cfg.environment != "" {
		builder.Environment(cfg.environment)
	}
	if cfg.hasStartTime {
		builder.StartTime(cfg.startTime)
	}
}

// applyScoreOpts resolves ScoreOptions and applies the shared id/comment fields to
// a ScoreBuilder. The caller selects the value type (numeric/boolean/categorical)
// before calling this. It returns the resolved scoreConfig in case the caller needs
// other fields.
func applyScoreOpts(builder *ScoreBuilder, opts ...ScoreOption) *scoreConfig {
	cfg := &scoreConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	if cfg.id != "" {
		builder.ID(cfg.id)
	}
	if cfg.comment != "" {
		builder.Comment(cfg.comment)
	}
	return cfg
}

// ============================================================================
// Simple API on Client
// ============================================================================

// Trace creates a new trace with the given name.
//
// Trace is a convenience wrapper over the canonical fluent builder
// ([Client.NewTrace]); it resolves the [TraceOption] functions and delegates to
// the builder. For new code, prefer the fluent builders directly.
//
// Example:
//
//	trace, err := client.Trace(ctx, "user-request",
//	    langfuse.WithUserID("user-123"),
//	    langfuse.WithTags("api", "v2"))
//	if err != nil {
//	    return err
//	}
//	defer client.Flush(ctx)
func (c *Client) Trace(ctx context.Context, name string, opts ...TraceOption) (*TraceContext, error) {
	cfg := &traceConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	builder := c.NewTrace().Name(name)
	applyTraceConfig(builder, cfg)

	return builder.Create(ctx)
}

// ============================================================================
// Simple API on TraceContext
// ============================================================================

// Span creates a new span with the given name (Simple API).
//
// This is a convenience wrapper over the canonical fluent builder
// ([TraceContext.NewSpan]); prefer the builder for new code.
//
// Example:
//
//	span, err := trace.Span(ctx, "preprocessing",
//	    langfuse.WithSpanInput(data))
//	if err != nil {
//	    return err
//	}
//	defer span.End(ctx)
func (t *TraceContext) Span(ctx context.Context, name string, opts ...SpanOption) (*SpanContext, error) {
	cfg := &spanConfig{}
	for _, opt := range opts {
		opt.applySpan(cfg)
	}

	builder := t.NewSpan().Name(name)
	applySpanConfig(builder, cfg)

	return builder.Create(ctx)
}

// Generation creates a new generation with the given name (Simple API).
//
// This is a convenience wrapper over the canonical fluent builder
// ([TraceContext.NewGeneration]); prefer the builder for new code.
//
// Example:
//
//	gen, err := trace.Generation(ctx, "gpt-4-call",
//	    langfuse.WithModel("gpt-4"),
//	    langfuse.WithGenerationInput(messages),
//	    langfuse.WithGenerationOutput(response),
//	    langfuse.WithTokenUsage(100, 50))
func (t *TraceContext) Generation(ctx context.Context, name string, opts ...GenerationOption) (*GenerationContext, error) {
	cfg := &generationConfig{}
	for _, opt := range opts {
		opt.applyGeneration(cfg)
	}

	builder := t.NewGeneration().Name(name)
	applyGenerationConfig(builder, cfg)

	return builder.Create(ctx)
}

// Event creates a new event with the given name (Simple API).
//
// This is a convenience wrapper over the canonical fluent builder
// ([TraceContext.NewEvent]); prefer the builder for new code.
//
// Example:
//
//	err := trace.Event(ctx, "cache-hit",
//	    langfuse.WithEventMetadata(langfuse.M{"key": cacheKey}))
func (t *TraceContext) Event(ctx context.Context, name string, opts ...EventOption) error {
	cfg := &eventConfig{}
	for _, opt := range opts {
		opt.applyEvent(cfg)
	}

	builder := t.NewEvent().Name(name)
	applyEventConfig(builder, cfg)

	return builder.Create(ctx)
}

// SetOutput updates the trace output.
//
// Example:
//
//	err := trace.SetOutput(ctx, response)
func (t *TraceContext) SetOutput(ctx context.Context, output any) error {
	return t.Update().Output(output).Apply(ctx)
}

// Complete marks the trace as complete (no-op, but useful for defer patterns).
// Traces don't require explicit completion, but this provides a consistent API.
//
// Example:
//
//	trace, _ := client.Trace(ctx, "request")
//	defer trace.Complete(ctx)
func (t *TraceContext) Complete(ctx context.Context) error {
	// Traces don't need explicit completion in Langfuse
	// This is a no-op that exists for API consistency
	return nil
}

// ============================================================================
// Simple API Scoring
// ============================================================================

// Score adds a numeric score to this trace (Simple API).
//
// This is a convenience wrapper over the canonical fluent builder
// ([TraceContext.NewScore]); prefer the builder for new code.
//
// Example:
//
//	err := trace.Score(ctx, "quality", 0.95,
//	    langfuse.WithComment("excellent response"))
func (t *TraceContext) Score(ctx context.Context, name string, value float64, opts ...ScoreOption) error {
	builder := t.NewScore().Name(name).NumericValue(value)
	applyScoreOpts(builder, opts...)
	return builder.Create(ctx)
}

// ScoreBool adds a boolean score to this trace (Simple API).
//
// This is a convenience wrapper over the canonical fluent builder
// ([TraceContext.NewScore]); prefer the builder for new code.
//
// Example:
//
//	err := trace.ScoreBool(ctx, "passed", true)
func (t *TraceContext) ScoreBool(ctx context.Context, name string, value bool, opts ...ScoreOption) error {
	builder := t.NewScore().Name(name).BooleanValue(value)
	applyScoreOpts(builder, opts...)
	return builder.Create(ctx)
}

// ScoreCategory adds a categorical score to this trace (Simple API).
//
// This is a convenience wrapper over the canonical fluent builder
// ([TraceContext.NewScore]); prefer the builder for new code.
//
// Example:
//
//	err := trace.ScoreCategory(ctx, "rating", "excellent")
func (t *TraceContext) ScoreCategory(ctx context.Context, name string, value string, opts ...ScoreOption) error {
	builder := t.NewScore().Name(name).CategoricalValue(value)
	applyScoreOpts(builder, opts...)
	return builder.Create(ctx)
}

// ============================================================================
// Simple API on SpanContext
// ============================================================================

// Span creates a child span (Simple API).
//
// This is a convenience wrapper over the canonical fluent builder
// ([SpanContext.NewSpan]); prefer the builder for new code.
func (s *SpanContext) Span(ctx context.Context, name string, opts ...SpanOption) (*SpanContext, error) {
	cfg := &spanConfig{}
	for _, opt := range opts {
		opt.applySpan(cfg)
	}

	builder := s.NewSpan().Name(name)
	applySpanConfig(builder, cfg)

	return builder.Create(ctx)
}

// Generation creates a child generation (Simple API).
//
// This is a convenience wrapper over the canonical fluent builder
// ([SpanContext.NewGeneration]); prefer the builder for new code.
func (s *SpanContext) Generation(ctx context.Context, name string, opts ...GenerationOption) (*GenerationContext, error) {
	cfg := &generationConfig{}
	for _, opt := range opts {
		opt.applyGeneration(cfg)
	}

	builder := s.NewGeneration().Name(name)
	applyGenerationConfig(builder, cfg)

	return builder.Create(ctx)
}

// Event creates a child event (Simple API).
//
// This is a convenience wrapper over the canonical fluent builder
// ([SpanContext.NewEvent]); prefer the builder for new code.
func (s *SpanContext) Event(ctx context.Context, name string, opts ...EventOption) error {
	cfg := &eventConfig{}
	for _, opt := range opts {
		opt.applyEvent(cfg)
	}

	builder := s.NewEvent().Name(name)
	applyEventConfig(builder, cfg)

	return builder.Create(ctx)
}

// Score adds a numeric score to this span (Simple API).
//
// This is a convenience wrapper over the canonical fluent builder
// ([SpanContext.NewScore]); prefer the builder for new code.
func (s *SpanContext) Score(ctx context.Context, name string, value float64, opts ...ScoreOption) error {
	builder := s.NewScore().Name(name).NumericValue(value)
	applyScoreOpts(builder, opts...)
	return builder.Create(ctx)
}

// ============================================================================
// Simple API on GenerationContext
// ============================================================================

// Span creates a child span (Simple API).
//
// This is a convenience wrapper over the canonical fluent builder
// ([GenerationContext.NewSpan]); prefer the builder for new code.
func (g *GenerationContext) Span(ctx context.Context, name string, opts ...SpanOption) (*SpanContext, error) {
	cfg := &spanConfig{}
	for _, opt := range opts {
		opt.applySpan(cfg)
	}

	builder := g.NewSpan().Name(name)
	applySpanConfig(builder, cfg)

	return builder.Create(ctx)
}

// Generation creates a child generation (Simple API).
//
// This is a convenience wrapper over the canonical fluent builder
// ([GenerationContext.NewGeneration]); prefer the builder for new code.
func (g *GenerationContext) Generation(ctx context.Context, name string, opts ...GenerationOption) (*GenerationContext, error) {
	cfg := &generationConfig{}
	for _, opt := range opts {
		opt.applyGeneration(cfg)
	}

	builder := g.NewGeneration().Name(name)
	applyGenerationConfig(builder, cfg)

	return builder.Create(ctx)
}

// Event creates a child event (Simple API).
//
// This is a convenience wrapper over the canonical fluent builder
// ([GenerationContext.NewEvent]); prefer the builder for new code.
func (g *GenerationContext) Event(ctx context.Context, name string, opts ...EventOption) error {
	cfg := &eventConfig{}
	for _, opt := range opts {
		opt.applyEvent(cfg)
	}

	builder := g.NewEvent().Name(name)
	applyEventConfig(builder, cfg)

	return builder.Create(ctx)
}

// Score adds a numeric score to this generation (Simple API).
//
// This is a convenience wrapper over the canonical fluent builder
// ([GenerationContext.NewScore]); prefer the builder for new code.
func (g *GenerationContext) Score(ctx context.Context, name string, value float64, opts ...ScoreOption) error {
	builder := g.NewScore().Name(name).NumericValue(value)
	applyScoreOpts(builder, opts...)
	return builder.Create(ctx)
}

// ============================================================================
// Tracing Helper Functions
// ============================================================================

// GenerationParams configures a traced generation.
type GenerationParams struct {
	// Name is the name of the generation (required).
	Name string
	// Model is the LLM model name (e.g., "gpt-4").
	Model string
	// Input is the prompt or input to the LLM.
	Input any
	// UserID is the user ID for the trace.
	UserID string
	// SessionID is the session ID for the trace.
	SessionID string
	// Metadata is additional metadata for the trace.
	Metadata Metadata
	// Tags are tags to apply to the trace.
	Tags []string
	// TraceName is the name of the parent trace (defaults to Name if not set).
	TraceName string
}

// GenerationResult contains the output and usage from a generation.
// NOTE: This is separate from EvalGenerationResult which is for evaluation.
type GenerationResult struct {
	// Output is the generated text or response.
	Output any
	// Usage contains token counts.
	Usage Usage
}

// GenerationFunc is called to perform the actual LLM call.
// It should return the output, usage information, and any error.
type GenerationFunc func() (GenerationResult, error)

// TraceGeneration traces an LLM generation with automatic timing and error handling.
// It creates a trace and generation, executes the provided function, and records
// the output and usage. If an error occurs, it sets the appropriate status.
//
// Example:
//
//	result, err := langfuse.TraceGeneration(ctx, client, langfuse.GenerationParams{
//	    Name:   "chat",
//	    Model:  "gpt-4",
//	    Input:  "Hello, world!",
//	    UserID: "user-123",
//	}, func() (langfuse.GenerationResult, error) {
//	    resp, err := openai.Complete(prompt)
//	    return langfuse.GenerationResult{
//	        Output: resp.Text,
//	        Usage:  langfuse.Usage{Input: resp.InputTokens, Output: resp.OutputTokens},
//	    }, err
//	})
func TraceGeneration(ctx context.Context, client *Client, params GenerationParams, fn GenerationFunc) (GenerationResult, error) {
	traceName := params.TraceName
	if traceName == "" {
		traceName = params.Name
	}

	// Create trace
	traceBuilder := client.NewTrace().Name(traceName)
	if params.UserID != "" {
		traceBuilder.UserID(params.UserID)
	}
	if params.SessionID != "" {
		traceBuilder.SessionID(params.SessionID)
	}
	if params.Metadata != nil {
		traceBuilder.Metadata(params.Metadata)
	}
	if params.Tags != nil {
		traceBuilder.Tags(params.Tags)
	}
	traceBuilder.Input(params.Input)

	trace, err := traceBuilder.Create(ctx)
	if err != nil {
		return GenerationResult{}, err
	}

	// Create generation
	genBuilder := trace.NewGeneration().Name(params.Name)
	if params.Model != "" {
		genBuilder.Model(params.Model)
	}
	if params.Input != nil {
		genBuilder.Input(params.Input)
	}

	gen, err := genBuilder.Create(ctx)
	if err != nil {
		return GenerationResult{}, err
	}

	// Execute the function
	result, fnErr := fn()

	// Record the result
	if fnErr != nil {
		// On error, still record what we have and set error status
		_ = gen.Update().
			Level(ObservationLevelError).
			StatusMessage(fnErr.Error()).
			Apply(ctx)
	} else {
		// Success - record output and usage
		_ = gen.EndWithUsage(ctx, result.Output, result.Usage.Input, result.Usage.Output)
	}

	// Update trace with output
	_ = trace.Update().Output(result.Output).Apply(ctx)

	return result, fnErr
}

// SpanFunc is called to perform work within a span.
// It receives the span context for creating child observations.
type SpanFunc func(span *SpanContext) error

// TraceSpan traces a function execution as a span with automatic timing and error handling.
// The span is automatically ended when the function returns.
//
// Example:
//
//	err := langfuse.TraceSpan(ctx, trace, "process-data", func(span *langfuse.SpanContext) error {
//	    // Do some work...
//	    span.Event().Name("step-1").Create(ctx)
//	    // Do more work...
//	    return nil
//	})
func TraceSpan(ctx context.Context, trace *TraceContext, name string, fn SpanFunc) error {
	span, err := trace.NewSpan().Name(name).Create(ctx)
	if err != nil {
		return err
	}

	// Execute the function
	startTime := time.Now()
	fnErr := fn(span)
	duration := time.Since(startTime)

	// Record the result
	update := span.Update().EndTime(time.Now())

	if fnErr != nil {
		update.Level(ObservationLevelError).StatusMessage(fnErr.Error())
	}

	// Add duration as metadata
	update.Metadata(map[string]any{
		"duration_ms": duration.Milliseconds(),
	})

	_ = update.Apply(ctx)

	return fnErr
}

// TraceFunc traces a function execution with automatic trace creation and cleanup.
// It's a convenience wrapper for simple function tracing.
//
// Example:
//
//	result, err := langfuse.TraceFunc(ctx, client, "my-function", func(trace *langfuse.TraceContext) (string, error) {
//	    // Do some work...
//	    return "result", nil
//	})
func TraceFunc[T any](ctx context.Context, client *Client, name string, fn func(trace *TraceContext) (T, error)) (T, error) {
	var zero T

	trace, err := client.NewTrace().Name(name).Create(ctx)
	if err != nil {
		return zero, err
	}

	// Execute the function
	startTime := time.Now()
	result, fnErr := fn(trace)
	duration := time.Since(startTime)

	// Record the result
	update := trace.Update()

	if fnErr != nil {
		update.Metadata(map[string]any{
			"error":       fnErr.Error(),
			"duration_ms": duration.Milliseconds(),
		})
	} else {
		update.Output(result).Metadata(map[string]any{
			"duration_ms": duration.Milliseconds(),
		})
	}

	_ = update.Apply(ctx)

	return result, fnErr
}

// WithGeneration is a convenience helper for creating a generation within a trace
// and automatically handling timing and output recording.
//
// Example:
//
//	gen, result, err := langfuse.WithGeneration(ctx, trace, "gpt-4", "Hello!", func() (string, Usage, error) {
//	    resp, err := openai.Complete("Hello!")
//	    return resp.Text, Usage{Input: 10, Output: 20}, err
//	})
func WithGeneration(ctx context.Context, trace *TraceContext, model string, input any, fn func() (any, Usage, error)) (*GenerationContext, any, error) {
	gen, err := trace.NewGeneration().
		Name(model).
		Model(model).
		Input(input).
		Create(ctx)
	if err != nil {
		return nil, nil, err
	}

	output, usage, fnErr := fn()

	if fnErr != nil {
		_ = gen.Update().
			Level(ObservationLevelError).
			StatusMessage(fnErr.Error()).
			Apply(ctx)
		return gen, nil, fnErr
	}

	_ = gen.EndWithUsage(ctx, output, usage.Input, usage.Output)
	return gen, output, nil
}

// ============================================================================
// Simplified Client Creation
// ============================================================================

// MustNew is a convenience constructor that wraps [New] with a panic-on-error
// signature, returning the client directly without an error.
//
// If configuration fails due to invalid credentials or other issues, MustNew
// panics. Use it only when initialization failures should be fatal (for example,
// at program startup with credentials sourced from the environment). For explicit
// error handling, use [New] or [NewWithConfig] instead.
//
// Example:
//
//	client := langfuse.MustNew("pk-lf-xxx", "sk-lf-xxx")
//	defer client.Shutdown(context.Background())
//
//	trace, _ := client.Trace(ctx, "user-request",
//	    langfuse.WithUserID("user-123"))
func MustNew(publicKey, secretKey string, opts ...ConfigOption) *Client {
	client, err := New(publicKey, secretKey, opts...)
	if err != nil {
		panic("langfuse: MustNew failed: " + err.Error())
	}
	return client
}

// ============================================================================
// Scorer Interface
// ============================================================================

// Scorer defines the interface for adding scores to observations.
type Scorer interface {
	// Score adds a numeric score.
	Score(ctx context.Context, name string, value float64, opts ...ScoreOption) error

	// ScoreBool adds a boolean score.
	ScoreBool(ctx context.Context, name string, value bool, opts ...ScoreOption) error

	// ScoreCategory adds a categorical score.
	ScoreCategory(ctx context.Context, name string, value string, opts ...ScoreOption) error
}

// Ensure contexts implement Scorer at compile time.
var (
	_ Scorer = (*TraceContext)(nil)
	_ Scorer = (*SpanContext)(nil)
	_ Scorer = (*GenerationContext)(nil)
)

// ScoreBool on SpanContext adds a boolean score (implements Scorer interface).
func (s *SpanContext) ScoreBool(ctx context.Context, name string, value bool, opts ...ScoreOption) error {
	builder := s.NewScore().Name(name).BooleanValue(value)
	applyScoreOpts(builder, opts...)
	return builder.Create(ctx)
}

// ScoreCategory on SpanContext adds a categorical score (implements Scorer interface).
func (s *SpanContext) ScoreCategory(ctx context.Context, name string, value string, opts ...ScoreOption) error {
	builder := s.NewScore().Name(name).CategoricalValue(value)
	applyScoreOpts(builder, opts...)
	return builder.Create(ctx)
}

// ScoreBool on GenerationContext adds a boolean score (implements Scorer interface).
func (g *GenerationContext) ScoreBool(ctx context.Context, name string, value bool, opts ...ScoreOption) error {
	builder := g.NewScore().Name(name).BooleanValue(value)
	applyScoreOpts(builder, opts...)
	return builder.Create(ctx)
}

// ScoreCategory on GenerationContext adds a categorical score (implements Scorer interface).
func (g *GenerationContext) ScoreCategory(ctx context.Context, name string, value string, opts ...ScoreOption) error {
	builder := g.NewScore().Name(name).CategoricalValue(value)
	applyScoreOpts(builder, opts...)
	return builder.Create(ctx)
}

// ============================================================================
// Context Helpers for Observations
// ============================================================================

// ContextWithObservation stores either a trace or span in context.
// This provides a unified way to propagate observability context.
type ContextWithObservation = context.Context

// WithObservation returns a context with the observer stored.
// Use TraceFromContext or SpanFromContext to retrieve it.
func WithObservation(ctx context.Context, obs Observer) context.Context {
	switch o := obs.(type) {
	case *TraceContext:
		return ContextWithTrace(ctx, o)
	case *SpanContext:
		return ContextWithSpan(ctx, o)
	default:
		return ctx
	}
}
