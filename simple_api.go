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
	apply(*spanConfig)
}

// spanOptionFunc allows using functions as SpanOptions.
type spanOptionFunc func(*spanConfig)

func (f spanOptionFunc) apply(c *spanConfig) { f(c) }

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
	apply2(*generationConfig)
}

// generationOptionFunc allows using functions as GenerationOptions.
type generationOptionFunc func(*generationConfig)

func (f generationOptionFunc) apply2(c *generationConfig) { f(c) }

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
	apply3(*eventConfig)
}

// eventOptionFunc allows using functions as EventOptions.
type eventOptionFunc func(*eventConfig)

func (f eventOptionFunc) apply3(c *eventConfig) { f(c) }

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
// Simple API on Client
// ============================================================================

// Trace creates a new trace with the given name.
// This is the Simple API for creating traces.
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

	return builder.Create(ctx)
}

// ============================================================================
// Simple API on TraceContext
// ============================================================================

// Span creates a new span with the given name (Simple API).
// For the Advanced API builder, use NewSpan().
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
		opt.apply(cfg)
	}

	builder := t.NewSpan().Name(name)

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

	return builder.Create(ctx)
}

// Generation creates a new generation with the given name (Simple API).
// For the Advanced API builder, use NewGeneration().
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
		opt.apply2(cfg)
	}

	builder := t.NewGeneration().Name(name)

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

	return builder.Create(ctx)
}

// Event creates a new event with the given name (Simple API).
// For the Advanced API builder, use NewEvent().
//
// Example:
//
//	err := trace.Event(ctx, "cache-hit",
//	    langfuse.WithEventMetadata(langfuse.M{"key": cacheKey}))
func (t *TraceContext) Event(ctx context.Context, name string, opts ...EventOption) error {
	cfg := &eventConfig{}
	for _, opt := range opts {
		opt.apply3(cfg)
	}

	builder := t.NewEvent().Name(name)

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
// Example:
//
//	err := trace.Score(ctx, "quality", 0.95,
//	    langfuse.WithComment("excellent response"))
func (t *TraceContext) Score(ctx context.Context, name string, value float64, opts ...ScoreOption) error {
	cfg := &scoreConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	builder := t.NewScore().Name(name).NumericValue(value)

	if cfg.id != "" {
		builder.ID(cfg.id)
	}
	if cfg.comment != "" {
		builder.Comment(cfg.comment)
	}

	return builder.Create(ctx)
}

// ScoreBool adds a boolean score to this trace (Simple API).
//
// Example:
//
//	err := trace.ScoreBool(ctx, "passed", true)
func (t *TraceContext) ScoreBool(ctx context.Context, name string, value bool, opts ...ScoreOption) error {
	cfg := &scoreConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	builder := t.NewScore().Name(name).BooleanValue(value)

	if cfg.id != "" {
		builder.ID(cfg.id)
	}
	if cfg.comment != "" {
		builder.Comment(cfg.comment)
	}

	return builder.Create(ctx)
}

// ScoreCategory adds a categorical score to this trace (Simple API).
//
// Example:
//
//	err := trace.ScoreCategory(ctx, "rating", "excellent")
func (t *TraceContext) ScoreCategory(ctx context.Context, name string, value string, opts ...ScoreOption) error {
	cfg := &scoreConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	builder := t.NewScore().Name(name).CategoricalValue(value)

	if cfg.id != "" {
		builder.ID(cfg.id)
	}
	if cfg.comment != "" {
		builder.Comment(cfg.comment)
	}

	return builder.Create(ctx)
}

// ============================================================================
// Simple API on SpanContext
// ============================================================================

// Span creates a child span (Simple API).
// For the Advanced API builder, use NewSpan().
func (s *SpanContext) Span(ctx context.Context, name string, opts ...SpanOption) (*SpanContext, error) {
	cfg := &spanConfig{}
	for _, opt := range opts {
		opt.apply(cfg)
	}

	builder := s.NewSpan().Name(name)

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

	return builder.Create(ctx)
}

// Generation creates a child generation (Simple API).
// For the Advanced API builder, use NewGeneration().
func (s *SpanContext) Generation(ctx context.Context, name string, opts ...GenerationOption) (*GenerationContext, error) {
	cfg := &generationConfig{}
	for _, opt := range opts {
		opt.apply2(cfg)
	}

	builder := s.NewGeneration().Name(name)

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

	return builder.Create(ctx)
}

// Event creates a child event (Simple API).
// For the Advanced API builder, use NewEvent().
func (s *SpanContext) Event(ctx context.Context, name string, opts ...EventOption) error {
	cfg := &eventConfig{}
	for _, opt := range opts {
		opt.apply3(cfg)
	}

	builder := s.NewEvent().Name(name)

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

	return builder.Create(ctx)
}

// Score adds a numeric score to this span (Simple API).
// For the Advanced API builder, use NewScore().
func (s *SpanContext) Score(ctx context.Context, name string, value float64, opts ...ScoreOption) error {
	cfg := &scoreConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	builder := s.NewScore().Name(name).NumericValue(value)

	if cfg.id != "" {
		builder.ID(cfg.id)
	}
	if cfg.comment != "" {
		builder.Comment(cfg.comment)
	}

	return builder.Create(ctx)
}

// ============================================================================
// Simple API on GenerationContext
// ============================================================================

// Span creates a child span (Simple API).
// For the Advanced API builder, use NewSpan().
func (g *GenerationContext) Span(ctx context.Context, name string, opts ...SpanOption) (*SpanContext, error) {
	cfg := &spanConfig{}
	for _, opt := range opts {
		opt.apply(cfg)
	}

	builder := g.NewSpan().Name(name)

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

	return builder.Create(ctx)
}

// Generation creates a child generation (Simple API).
// For the Advanced API builder, use NewGeneration().
func (g *GenerationContext) Generation(ctx context.Context, name string, opts ...GenerationOption) (*GenerationContext, error) {
	cfg := &generationConfig{}
	for _, opt := range opts {
		opt.apply2(cfg)
	}

	builder := g.NewGeneration().Name(name)

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

	return builder.Create(ctx)
}

// Event creates a child event (Simple API).
// For the Advanced API builder, use NewEvent().
func (g *GenerationContext) Event(ctx context.Context, name string, opts ...EventOption) error {
	cfg := &eventConfig{}
	for _, opt := range opts {
		opt.apply3(cfg)
	}

	builder := g.NewEvent().Name(name)

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

	return builder.Create(ctx)
}

// Score adds a numeric score to this generation (Simple API).
// For the Advanced API builder, use NewScore().
func (g *GenerationContext) Score(ctx context.Context, name string, value float64, opts ...ScoreOption) error {
	cfg := &scoreConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	builder := g.NewScore().Name(name).NumericValue(value)

	if cfg.id != "" {
		builder.ID(cfg.id)
	}
	if cfg.comment != "" {
		builder.Comment(cfg.comment)
	}

	return builder.Create(ctx)
}
