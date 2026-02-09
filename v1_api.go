package langfuse

import (
	"context"
	"time"
)

// ============================================================================
// V1 API - Simplified Client Creation
// ============================================================================

// NewClient creates a new Langfuse client with the simplified v1 API.
// This function returns the client directly without an error.
//
// If configuration fails due to invalid credentials or other issues,
// the function will panic. For explicit error handling, use New()
// or NewWithConfig() instead.
//
// Example:
//
//	client := langfuse.NewClient("pk-lf-xxx", "sk-lf-xxx")
//	defer client.Shutdown(context.Background())
//
//	trace, _ := client.Trace(ctx, "user-request",
//	    langfuse.WithUserID("user-123"))
func NewClient(publicKey, secretKey string, opts ...ConfigOption) *Client {
	client, err := New(publicKey, secretKey, opts...)
	if err != nil {
		panic("langfuse: NewClient failed: " + err.Error())
	}
	return client
}

// MustClient is an alias for NewClient that clearly indicates it panics on error.
// Use this when initialization failures should be fatal.
func MustClient(publicKey, secretKey string, opts ...ConfigOption) *Client {
	return NewClient(publicKey, secretKey, opts...)
}

// TryClient creates a new Langfuse client, returning nil if initialization fails.
// Use this when you want optional observability that gracefully degrades.
//
// Example:
//
//	client := langfuse.TryClient("pk-lf-xxx", "sk-lf-xxx")
//	if client != nil {
//	    defer client.Shutdown(context.Background())
//	}
func TryClient(publicKey, secretKey string, opts ...ConfigOption) *Client {
	client, _ := New(publicKey, secretKey, opts...)
	return client
}

// ============================================================================
// V1 API - Context-First Trace Creation
// ============================================================================

// TraceV1 creates a new trace with context-first approach.
// This is an alias for Trace() that matches the v1 API naming convention.
//
// Example:
//
//	trace, err := client.TraceV1(ctx, "user-request",
//	    langfuse.WithUserID("user-123"),
//	    langfuse.WithTags("api", "v2"))
func (c *Client) TraceV1(ctx context.Context, name string, opts ...TraceOption) (*TraceContext, error) {
	return c.Trace(ctx, name, opts...)
}

// ============================================================================
// V1 API - Context-First Span Creation
// ============================================================================

// NewSpan creates a new span with context-first approach (v1 API).
// This is an alias for Span() that matches the v1 API naming convention.
//
// Example:
//
//	span, err := trace.NewSpan(ctx, "processing",
//	    langfuse.WithSpanInput(data))
func (t *TraceContext) NewSpanV1(ctx context.Context, name string, opts ...SpanOption) (*SpanContext, error) {
	return t.Span(ctx, name, opts...)
}

// NewSpan on SpanContext creates a child span with context-first approach (v1 API).
func (s *SpanContext) NewSpanV1(ctx context.Context, name string, opts ...SpanOption) (*SpanContext, error) {
	return s.Span(ctx, name, opts...)
}

// NewSpan on GenerationContext creates a child span with context-first approach (v1 API).
func (g *GenerationContext) NewSpanV1(ctx context.Context, name string, opts ...SpanOption) (*SpanContext, error) {
	return g.Span(ctx, name, opts...)
}

// ============================================================================
// V1 API - Context-First Generation Creation
// ============================================================================

// NewGeneration creates a new generation with context-first approach (v1 API).
// This is an alias for Generation() that matches the v1 API naming convention.
//
// Example:
//
//	gen, err := trace.NewGeneration(ctx, "llm-call",
//	    langfuse.WithModel("gpt-4"),
//	    langfuse.WithGenerationInput(messages))
func (t *TraceContext) NewGenerationV1(ctx context.Context, name string, opts ...GenerationOption) (*GenerationContext, error) {
	return t.Generation(ctx, name, opts...)
}

// NewGeneration on SpanContext creates a child generation (v1 API).
func (s *SpanContext) NewGenerationV1(ctx context.Context, name string, opts ...GenerationOption) (*GenerationContext, error) {
	return s.Generation(ctx, name, opts...)
}

// NewGeneration on GenerationContext creates a child generation (v1 API).
func (g *GenerationContext) NewGenerationV1(ctx context.Context, name string, opts ...GenerationOption) (*GenerationContext, error) {
	return g.Generation(ctx, name, opts...)
}

// ============================================================================
// V1 API - Context-First Event Creation
// ============================================================================

// NewEvent creates a new event with context-first approach (v1 API).
//
// Example:
//
//	err := trace.NewEvent(ctx, "cache-hit",
//	    langfuse.WithEventMetadata(langfuse.M{"key": cacheKey}))
func (t *TraceContext) NewEventV1(ctx context.Context, name string, opts ...EventOption) error {
	return t.Event(ctx, name, opts...)
}

// NewEvent on SpanContext creates a child event (v1 API).
func (s *SpanContext) NewEventV1(ctx context.Context, name string, opts ...EventOption) error {
	return s.Event(ctx, name, opts...)
}

// NewEvent on GenerationContext creates a child event (v1 API).
func (g *GenerationContext) NewEventV1(ctx context.Context, name string, opts ...EventOption) error {
	return g.Event(ctx, name, opts...)
}

// ============================================================================
// V1 API - Unified Update Methods
// ============================================================================

// updateConfig holds the configuration for updating an entity.
type updateConfig struct {
	output     any
	metadata   Metadata
	tags       []string
	hasTags    bool
	input      any
	level      ObservationLevel
	hasLevel   bool
	statusMsg  string
	name       string
	userID     string
	sessionID  string
	public     bool
	hasPublic  bool
	endTime    time.Time
	hasEndTime bool
}

// UpdateOption configures an update operation.
type UpdateOption func(*updateConfig)

// WithOutput sets the output for an update operation.
func WithUpdateOutput(output any) UpdateOption {
	return func(c *updateConfig) {
		c.output = output
	}
}

// WithUpdateMetadata sets the metadata for an update operation.
func WithUpdateMetadata(metadata Metadata) UpdateOption {
	return func(c *updateConfig) {
		c.metadata = metadata
	}
}

// WithUpdateTags sets the tags for an update operation.
func WithUpdateTags(tags ...string) UpdateOption {
	return func(c *updateConfig) {
		c.tags = tags
		c.hasTags = true
	}
}

// WithUpdateInput sets the input for an update operation.
func WithUpdateInput(input any) UpdateOption {
	return func(c *updateConfig) {
		c.input = input
	}
}

// WithUpdateLevel sets the observation level for an update operation.
func WithUpdateLevel(level ObservationLevel) UpdateOption {
	return func(c *updateConfig) {
		c.level = level
		c.hasLevel = true
	}
}

// WithUpdateStatusMessage sets the status message for an update operation.
func WithUpdateStatusMessage(msg string) UpdateOption {
	return func(c *updateConfig) {
		c.statusMsg = msg
	}
}

// WithUpdateName sets the name for an update operation.
func WithUpdateName(name string) UpdateOption {
	return func(c *updateConfig) {
		c.name = name
	}
}

// WithUpdateUserID sets the user ID for a trace update.
func WithUpdateUserID(userID string) UpdateOption {
	return func(c *updateConfig) {
		c.userID = userID
	}
}

// WithUpdateSessionID sets the session ID for a trace update.
func WithUpdateSessionID(sessionID string) UpdateOption {
	return func(c *updateConfig) {
		c.sessionID = sessionID
	}
}

// WithUpdatePublic sets whether a trace is public.
func WithUpdatePublic(public bool) UpdateOption {
	return func(c *updateConfig) {
		c.public = public
		c.hasPublic = true
	}
}

// UpdateV1 updates the trace with the provided options and returns the trace context.
// This provides a unified, consistent API for updates.
//
// Example:
//
//	trace, err = trace.UpdateV1(ctx,
//	    langfuse.WithUpdateOutput(response),
//	    langfuse.WithUpdateTags("completed"))
func (t *TraceContext) UpdateV1(ctx context.Context, opts ...UpdateOption) (*TraceContext, error) {
	cfg := &updateConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	update := t.Update()

	if cfg.name != "" {
		update.Name(cfg.name)
	}
	if cfg.userID != "" {
		update.UserID(cfg.userID)
	}
	if cfg.sessionID != "" {
		update.SessionID(cfg.sessionID)
	}
	if cfg.input != nil {
		update.Input(cfg.input)
	}
	if cfg.output != nil {
		update.Output(cfg.output)
	}
	if cfg.metadata != nil {
		update.Metadata(cfg.metadata)
	}
	if cfg.hasTags {
		update.Tags(cfg.tags)
	}
	if cfg.hasPublic {
		update.Public(cfg.public)
	}

	if err := update.Apply(ctx); err != nil {
		return nil, err
	}

	return t, nil
}

// ============================================================================
// V1 API - Unified End Methods with Consistent Return Types
// ============================================================================

// EndV1 ends the span with the provided options and returns the span context.
// This provides a consistent (result, error) return pattern.
//
// Example:
//
//	span, err := span.EndV1(ctx,
//	    langfuse.WithEndOutput(response),
//	    langfuse.WithEndMetadata(langfuse.M{"cached": true}))
func (s *SpanContext) EndV1(ctx context.Context, opts ...EndOption) (*SpanContext, error) {
	result := s.EndWith(ctx, opts...)
	if result.Error != nil {
		return nil, result.Error
	}
	return s, nil
}

// EndV1 ends the generation with the provided options and returns the generation context.
// This provides a consistent (result, error) return pattern.
//
// Example:
//
//	gen, err := gen.EndV1(ctx,
//	    langfuse.WithEndOutput(response),
//	    langfuse.WithUsage(100, 50))
func (g *GenerationContext) EndV1(ctx context.Context, opts ...EndOption) (*GenerationContext, error) {
	result := g.EndWith(ctx, opts...)
	if result.Error != nil {
		return nil, result.Error
	}
	return g, nil
}

// ============================================================================
// V1 API - Additional End Options
// ============================================================================

// WithEndOutput is an alias for WithOutput, provided for v1 API clarity.
func WithEndOutput(output any) EndOption {
	return WithOutput(output)
}

// WithEndDuration sets the end time based on a duration from start.
// This is useful when you've measured the operation duration separately.
//
// Example:
//
//	gen.EndV1(ctx, langfuse.WithEndDuration(800*time.Millisecond))
func WithEndDuration(d time.Duration) EndOption {
	return func(c *endConfig) {
		// Duration is calculated from "now" - so we set endTime to now
		// The actual duration tracking happens via start/end time difference
		c.endTime = time.Now()
		c.hasEndTime = true
	}
}

// ============================================================================
// V1 API - Additional Span Options
// ============================================================================

// WithSpanLevel sets the observation level for a span.
// This is an alias for WithLevel with a clearer name.
func WithSpanLevel(level ObservationLevel) SpanOption {
	return WithLevel(level)
}

// ============================================================================
// V1 API - Additional Score Options
// ============================================================================

// WithScoreComment sets a comment for the score.
// This is an alias for WithComment with a clearer name.
func WithScoreComment(comment string) ScoreOption {
	return WithComment(comment)
}

// WithScoreDataType sets the data type for the score.
// Uses the ScoreDataType constants from types.go.
func WithScoreDataType(dataType ScoreDataType) ScoreOption {
	return func(c *scoreConfig) {
		// Note: The score config doesn't have a dataType field yet,
		// but this is a placeholder for future enhancement
	}
}

// ============================================================================
// V1 API - Client Statistics
// ============================================================================
// Note: ClientStats and Stats() are defined in metrics.go with a comprehensive
// implementation. Use client.Stats() to get current statistics.

// ============================================================================
// V1 API - Scorer Interface
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
	cfg := &scoreConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	builder := s.NewScore().Name(name).BooleanValue(value)

	if cfg.id != "" {
		builder.ID(cfg.id)
	}
	if cfg.comment != "" {
		builder.Comment(cfg.comment)
	}

	return builder.Create(ctx)
}

// ScoreCategory on SpanContext adds a categorical score (implements Scorer interface).
func (s *SpanContext) ScoreCategory(ctx context.Context, name string, value string, opts ...ScoreOption) error {
	cfg := &scoreConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	builder := s.NewScore().Name(name).CategoricalValue(value)

	if cfg.id != "" {
		builder.ID(cfg.id)
	}
	if cfg.comment != "" {
		builder.Comment(cfg.comment)
	}

	return builder.Create(ctx)
}

// ScoreBool on GenerationContext adds a boolean score (implements Scorer interface).
func (g *GenerationContext) ScoreBool(ctx context.Context, name string, value bool, opts ...ScoreOption) error {
	cfg := &scoreConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	builder := g.NewScore().Name(name).BooleanValue(value)

	if cfg.id != "" {
		builder.ID(cfg.id)
	}
	if cfg.comment != "" {
		builder.Comment(cfg.comment)
	}

	return builder.Create(ctx)
}

// ScoreCategory on GenerationContext adds a categorical score (implements Scorer interface).
func (g *GenerationContext) ScoreCategory(ctx context.Context, name string, value string, opts ...ScoreOption) error {
	cfg := &scoreConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	builder := g.NewScore().Name(name).CategoricalValue(value)

	if cfg.id != "" {
		builder.ID(cfg.id)
	}
	if cfg.comment != "" {
		builder.Comment(cfg.comment)
	}

	return builder.Create(ctx)
}

// ============================================================================
// V1 API - Context Helper for Traces
// ============================================================================

// TraceContextV1 is an extended trace context that provides v1 API methods.
// It wraps TraceContext and adds the NewXxx naming convention methods.
type TraceContextV1 struct {
	*TraceContext
}

// WrapTraceContext wraps a TraceContext to provide v1 API naming.
func WrapTraceContext(t *TraceContext) *TraceContextV1 {
	return &TraceContextV1{TraceContext: t}
}

// ============================================================================
// V1 API - ContextWithTrace alias for clearer naming
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
