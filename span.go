package langfuse

import (
	"context"
	"time"
)

// spanContextKey is the context key for SpanContext.
type spanContextKey struct{}

// SpanFromContext returns the SpanContext from ctx, if present.
// This allows retrieving a span that was previously stored in the context
// for use in nested function calls without explicitly passing the span.
//
// Example:
//
//	func processItem(ctx context.Context) {
//	    span, ok := langfuse.SpanFromContext(ctx)
//	    if ok {
//	        childSpan, _ := span.Span(ctx, "item-processing")
//	        defer childSpan.End(ctx)
//	    }
//	}
func SpanFromContext(ctx context.Context) (*SpanContext, bool) {
	sc, ok := ctx.Value(spanContextKey{}).(*SpanContext)
	return sc, ok
}

// ContextWithSpan returns a new context with the SpanContext stored.
// This allows passing span context through function calls without
// explicitly threading the span through all parameters.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process")
//	ctx = langfuse.ContextWithSpan(ctx, span)
//	doWork(ctx) // Can retrieve span via SpanFromContext
func ContextWithSpan(ctx context.Context, sc *SpanContext) context.Context {
	return context.WithValue(ctx, spanContextKey{}, sc)
}

// MustSpanFromContext returns the SpanContext from ctx.
// It panics if no SpanContext is present.
// Use this only when you're certain the context contains a span.
func MustSpanFromContext(ctx context.Context) *SpanContext {
	sc, ok := SpanFromContext(ctx)
	if !ok {
		panic("langfuse: no SpanContext in context")
	}
	return sc
}

// SpanBuilder provides a fluent interface for creating spans.
//
// SpanBuilder is NOT safe for concurrent use. Each builder instance should
// be created, configured, and used within a single goroutine. All setter
// methods modify the builder in place and return the same pointer for method
// chaining.
//
// Example:
//
//	span, err := trace.Span().
//	    Name("process-data").
//	    Input(data).
//	    Create()
type SpanBuilder struct {
	ctx  *TraceContext
	span *createSpanEvent
}

// ID sets the span ID.
func (b *SpanBuilder) ID(id string) *SpanBuilder {
	b.span.ID = id
	return b
}

// Name sets the span name.
func (b *SpanBuilder) Name(name string) *SpanBuilder {
	b.span.Name = name
	return b
}

// StartTime sets the start time.
func (b *SpanBuilder) StartTime(t time.Time) *SpanBuilder {
	b.span.StartTime = Time{Time: t}
	return b
}

// EndTime sets the end time.
func (b *SpanBuilder) EndTime(t time.Time) *SpanBuilder {
	b.span.EndTime = Time{Time: t}
	return b
}

// Input sets the input.
func (b *SpanBuilder) Input(input any) *SpanBuilder {
	b.span.Input = input
	return b
}

// Output sets the output.
func (b *SpanBuilder) Output(output any) *SpanBuilder {
	b.span.Output = output
	return b
}

// Metadata sets the metadata.
func (b *SpanBuilder) Metadata(metadata Metadata) *SpanBuilder {
	b.span.Metadata = metadata
	return b
}

// Level sets the observation level.
func (b *SpanBuilder) Level(level ObservationLevel) *SpanBuilder {
	b.span.Level = level
	return b
}

// StatusMessage sets the status message.
func (b *SpanBuilder) StatusMessage(msg string) *SpanBuilder {
	b.span.StatusMessage = msg
	return b
}

// ParentObservationID sets the parent observation ID.
func (b *SpanBuilder) ParentObservationID(id string) *SpanBuilder {
	b.span.ParentObservationID = id
	return b
}

// ParentID is an alias for ParentObservationID.
// It sets the parent observation ID for this span.
func (b *SpanBuilder) ParentID(id string) *SpanBuilder {
	return b.ParentObservationID(id)
}

// Version sets the version.
func (b *SpanBuilder) Version(version string) *SpanBuilder {
	b.span.Version = version
	return b
}

// Environment sets the environment.
func (b *SpanBuilder) Environment(env string) *SpanBuilder {
	b.span.Environment = env
	return b
}

// Clone creates a deep copy of the SpanBuilder with a new ID and timestamp.
// This is useful for creating multiple similar spans from a template.
//
// Example:
//
//	template := trace.Span().
//	    Level(langfuse.ObservationLevelDefault).
//	    Environment("production")
//
//	span1, _ := template.Clone().Name("step-1").Create(ctx)
//	span2, _ := template.Clone().Name("step-2").Create(ctx)
func (b *SpanBuilder) Clone() *SpanBuilder {
	// Deep copy metadata map
	var metadata Metadata
	if b.span.Metadata != nil {
		metadata = make(Metadata, len(b.span.Metadata))
		for k, v := range b.span.Metadata {
			metadata[k] = v
		}
	}

	return &SpanBuilder{
		ctx: b.ctx,
		span: &createSpanEvent{
			ID:                  generateID(), // New ID for the clone
			TraceID:             b.span.TraceID,
			Name:                b.span.Name,
			StartTime:           Now(), // Fresh timestamp
			EndTime:             b.span.EndTime,
			Metadata:            metadata,
			Level:               b.span.Level,
			StatusMessage:       b.span.StatusMessage,
			ParentObservationID: b.span.ParentObservationID,
			Version:             b.span.Version,
			Input:               b.span.Input,
			Output:              b.span.Output,
			Environment:         b.span.Environment,
		},
	}
}

// Validate validates the span builder configuration.
func (b *SpanBuilder) Validate() error {
	if b.span.ID == "" {
		return NewValidationError("id", "span ID cannot be empty")
	}
	if b.span.TraceID == "" {
		return NewValidationError("traceId", "trace ID cannot be empty")
	}
	return nil
}

// Create creates the span and returns a SpanContext.
func (b *SpanBuilder) Create(ctx context.Context) (*SpanContext, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}

	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeSpanCreate,
		Timestamp: Now(),
		Body:      b.span,
	}

	if err := b.ctx.client.queueEvent(ctx, event); err != nil {
		return nil, err
	}

	return &SpanContext{
		TraceContext: b.ctx,
		spanID:       b.span.ID,
	}, nil
}

// SpanContext provides context for a span.
//
// SpanContext is safe for concurrent use. You can create child spans,
// generations, and events concurrently. However, individual builders
// created from SpanContext are NOT safe for concurrent use.
type SpanContext struct {
	*TraceContext
	spanID string
}

// SpanID returns the span ID.
func (s *SpanContext) SpanID() string {
	return s.spanID
}

// ID returns the span ID. This method satisfies the Observer interface.
func (s *SpanContext) ID() string {
	return s.spanID
}

// Update updates the span.
func (s *SpanContext) Update() *SpanUpdateBuilder {
	return &SpanUpdateBuilder{
		ctx: s,
		update: &updateSpanEvent{
			ID:      s.spanID,
			TraceID: s.traceID,
		},
	}
}

// End ends the span with the current time.
func (s *SpanContext) End(ctx context.Context) error {
	return s.Update().EndTime(time.Now()).Apply(ctx)
}

// EndWithOutput ends the span with output and the current time.
func (s *SpanContext) EndWithOutput(ctx context.Context, output any) error {
	return s.Update().Output(output).EndTime(time.Now()).Apply(ctx)
}

// EndWith ends the span with the provided options.
// This provides a consistent, flexible API for ending observations.
//
// Example:
//
//	result := span.EndWith(ctx,
//	    WithOutput(response),
//	    WithEndMetadata(Metadata{"cache_hit": true}),
//	)
//	if !result.Ok() {
//	    log.Printf("Failed to end span: %v", result.Error)
//	}
//
//	// With error handling:
//	result := span.EndWith(ctx, WithError(err))
func (s *SpanContext) EndWith(ctx context.Context, opts ...EndOption) EndResult {
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
	update := s.Update().EndTime(endTime)

	if cfg.output != nil {
		update.Output(cfg.output)
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

	err := update.Apply(ctx)

	return EndResult{
		Error: err,
	}
}

// NewSpan creates a child span builder (Advanced API).
// For the Simple API, use Span(ctx, name, ...opts).
func (s *SpanContext) NewSpan() *SpanBuilder {
	builder := s.TraceContext.NewSpan()
	builder.span.ParentObservationID = s.spanID
	return builder
}

// NewGeneration creates a child generation builder (Advanced API).
// For the Simple API, use Generation(ctx, name, ...opts).
func (s *SpanContext) NewGeneration() *GenerationBuilder {
	builder := s.TraceContext.NewGeneration()
	builder.gen.ParentObservationID = s.spanID
	return builder
}

// NewEvent creates a child event builder (Advanced API).
// For the Simple API, use Event(ctx, name, ...opts).
func (s *SpanContext) NewEvent() *EventBuilder {
	builder := s.TraceContext.NewEvent()
	builder.event.ParentObservationID = s.spanID
	return builder
}

// NewScore creates a score builder for this span (Advanced API).
// For the Simple API, use Score(ctx, name, value, ...opts).
func (s *SpanContext) NewScore() *ScoreBuilder {
	builder := s.TraceContext.NewScore()
	builder.score.ObservationID = s.spanID
	return builder
}

// ScoreNumeric adds a numeric score to this span.
// This is a convenience method for the common case of adding a simple numeric score.
func (s *SpanContext) ScoreNumeric(ctx context.Context, name string, value float64) error {
	return s.NewScore().Name(name).NumericValue(value).Create(ctx)
}

// ScoreCategorical adds a categorical score to this span.
// This is a convenience method for the common case of adding a simple categorical score.
func (s *SpanContext) ScoreCategorical(ctx context.Context, name string, value string) error {
	return s.NewScore().Name(name).CategoricalValue(value).Create(ctx)
}

// ScoreBoolean adds a boolean score to this span.
// This is a convenience method for the common case of adding a simple boolean score.
func (s *SpanContext) ScoreBoolean(ctx context.Context, name string, value bool) error {
	return s.NewScore().Name(name).BooleanValue(value).Create(ctx)
}

// SpanUpdateBuilder provides a fluent interface for updating spans.
//
// SpanUpdateBuilder is NOT safe for concurrent use. Each builder instance
// should be created, configured, and used within a single goroutine.
type SpanUpdateBuilder struct {
	ctx    *SpanContext
	update *updateSpanEvent
}

// Name sets the span name.
func (b *SpanUpdateBuilder) Name(name string) *SpanUpdateBuilder {
	b.update.Name = name
	return b
}

// EndTime sets the end time.
func (b *SpanUpdateBuilder) EndTime(t time.Time) *SpanUpdateBuilder {
	b.update.EndTime = Time{Time: t}
	return b
}

// Input sets the input.
func (b *SpanUpdateBuilder) Input(input any) *SpanUpdateBuilder {
	b.update.Input = input
	return b
}

// Output sets the output.
func (b *SpanUpdateBuilder) Output(output any) *SpanUpdateBuilder {
	b.update.Output = output
	return b
}

// Metadata sets the metadata.
func (b *SpanUpdateBuilder) Metadata(metadata Metadata) *SpanUpdateBuilder {
	b.update.Metadata = metadata
	return b
}

// Level sets the observation level.
func (b *SpanUpdateBuilder) Level(level ObservationLevel) *SpanUpdateBuilder {
	b.update.Level = level
	return b
}

// StatusMessage sets the status message.
func (b *SpanUpdateBuilder) StatusMessage(msg string) *SpanUpdateBuilder {
	b.update.StatusMessage = msg
	return b
}

// Version sets the version.
func (b *SpanUpdateBuilder) Version(version string) *SpanUpdateBuilder {
	b.update.Version = version
	return b
}

// Apply applies the update.
func (b *SpanUpdateBuilder) Apply(ctx context.Context) error {
	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeSpanUpdate,
		Timestamp: Now(),
		Body:      b.update,
	}

	return b.ctx.client.queueEvent(ctx, event)
}

// ============================================================================
// Event Builder
// ============================================================================

// EventBuilder provides a fluent interface for creating events.
//
// EventBuilder is NOT safe for concurrent use. Each builder instance should
// be created, configured, and used within a single goroutine. All setter
// methods modify the builder in place and return the same pointer for method
// chaining.
//
// Example:
//
//	err := trace.Event().
//	    Name("user-action").
//	    Input(actionData).
//	    Create()
type EventBuilder struct {
	ctx   *TraceContext
	event *createEventEvent
}

// ID sets the event ID.
func (b *EventBuilder) ID(id string) *EventBuilder {
	b.event.ID = id
	return b
}

// Name sets the event name.
func (b *EventBuilder) Name(name string) *EventBuilder {
	b.event.Name = name
	return b
}

// StartTime sets the start time.
func (b *EventBuilder) StartTime(t time.Time) *EventBuilder {
	b.event.StartTime = Time{Time: t}
	return b
}

// Input sets the input.
func (b *EventBuilder) Input(input any) *EventBuilder {
	b.event.Input = input
	return b
}

// Output sets the output.
func (b *EventBuilder) Output(output any) *EventBuilder {
	b.event.Output = output
	return b
}

// Metadata sets the metadata.
func (b *EventBuilder) Metadata(metadata Metadata) *EventBuilder {
	b.event.Metadata = metadata
	return b
}

// Level sets the observation level.
func (b *EventBuilder) Level(level ObservationLevel) *EventBuilder {
	b.event.Level = level
	return b
}

// StatusMessage sets the status message.
func (b *EventBuilder) StatusMessage(msg string) *EventBuilder {
	b.event.StatusMessage = msg
	return b
}

// ParentObservationID sets the parent observation ID.
func (b *EventBuilder) ParentObservationID(id string) *EventBuilder {
	b.event.ParentObservationID = id
	return b
}

// ParentID is an alias for ParentObservationID.
// It sets the parent observation ID for this event.
func (b *EventBuilder) ParentID(id string) *EventBuilder {
	return b.ParentObservationID(id)
}

// Version sets the version.
func (b *EventBuilder) Version(version string) *EventBuilder {
	b.event.Version = version
	return b
}

// Environment sets the environment.
func (b *EventBuilder) Environment(env string) *EventBuilder {
	b.event.Environment = env
	return b
}

// Validate validates the event builder configuration.
func (b *EventBuilder) Validate() error {
	if b.event.ID == "" {
		return NewValidationError("id", "event ID cannot be empty")
	}
	if b.event.TraceID == "" {
		return NewValidationError("traceId", "trace ID cannot be empty")
	}
	return nil
}

// Create creates the event.
func (b *EventBuilder) Create(ctx context.Context) error {
	if err := b.Validate(); err != nil {
		return err
	}

	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeEventCreate,
		Timestamp: Now(),
		Body:      b.event,
	}

	return b.ctx.client.queueEvent(ctx, event)
}
