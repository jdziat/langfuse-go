package langfuse

import (
	"context"
)

// traceContextKey is the context key for TraceContext.
type traceContextKey struct{}

// TraceFromContext returns the TraceContext from ctx, if present.
// This allows retrieving a trace that was previously stored in the context
// for use in nested function calls without explicitly passing the trace.
//
// Example:
//
//	func handleRequest(ctx context.Context) {
//	    trace, ok := langfuse.TraceFromContext(ctx)
//	    if ok {
//	        span, _ := trace.NewSpan().Name("sub-operation").Create(ctx)
//	        defer span.End(ctx)
//	    }
//	}
func TraceFromContext(ctx context.Context) (*TraceContext, bool) {
	tc, ok := ctx.Value(traceContextKey{}).(*TraceContext)
	return tc, ok
}

// ContextWithTrace returns a new context with the TraceContext stored.
// This allows passing trace context through function calls without
// explicitly threading the trace through all parameters.
//
// Example:
//
//	trace, _ := client.NewTrace().Name("request").Create(ctx)
//	ctx = langfuse.ContextWithTrace(ctx, trace)
//	processRequest(ctx) // Can retrieve trace via TraceFromContext
func ContextWithTrace(ctx context.Context, tc *TraceContext) context.Context {
	return context.WithValue(ctx, traceContextKey{}, tc)
}

// MustTraceFromContext returns the TraceContext from ctx.
// It panics if no TraceContext is present.
// Use this only when you're certain the context contains a trace.
func MustTraceFromContext(ctx context.Context) *TraceContext {
	tc, ok := TraceFromContext(ctx)
	if !ok {
		panic("langfuse: no TraceContext in context")
	}
	return tc
}

// TraceBuilder provides a fluent interface for creating traces.
//
// TraceBuilder is NOT safe for concurrent use. Each builder instance should
// be created, configured, and used within a single goroutine. All setter
// methods modify the builder in place and return the same pointer for method
// chaining.
//
// Validation is performed both on set (for early error detection) and at
// Create() time. Use HasErrors() to check for validation errors before
// calling Create(), or let Create() return the combined errors.
//
// Example:
//
//	trace, err := client.NewTrace().
//	    Name("my-trace").
//	    UserID("user-123").
//	    Create()
type TraceBuilder struct {
	client    *Client
	trace     *createTraceEvent
	validator Validator
}

// NewTrace creates a new trace builder.
func (c *Client) NewTrace() *TraceBuilder {
	return &TraceBuilder{
		client: c,
		trace: &createTraceEvent{
			ID:        generateID(),
			Timestamp: Now(),
		},
	}
}

// ID sets the trace ID.
func (b *TraceBuilder) ID(id string) *TraceBuilder {
	b.trace.ID = id
	return b
}

// Name sets the trace name.
func (b *TraceBuilder) Name(name string) *TraceBuilder {
	b.trace.Name = name
	return b
}

// UserID sets the user ID.
func (b *TraceBuilder) UserID(userID string) *TraceBuilder {
	b.trace.UserID = userID
	return b
}

// SessionID sets the session ID.
func (b *TraceBuilder) SessionID(sessionID string) *TraceBuilder {
	b.trace.SessionID = sessionID
	return b
}

// Input sets the trace input.
func (b *TraceBuilder) Input(input any) *TraceBuilder {
	b.trace.Input = input
	return b
}

// Output sets the trace output.
func (b *TraceBuilder) Output(output any) *TraceBuilder {
	b.trace.Output = output
	return b
}

// Metadata sets the trace metadata.
// Validates that metadata keys are not empty.
func (b *TraceBuilder) Metadata(metadata Metadata) *TraceBuilder {
	if err := ValidateMetadata("metadata", metadata); err != nil {
		b.validator.AddError(err)
	}
	b.trace.Metadata = metadata
	return b
}

// Tags sets the trace tags.
// Validates that individual tags are not empty.
func (b *TraceBuilder) Tags(tags []string) *TraceBuilder {
	if err := ValidateTags("tags", tags); err != nil {
		b.validator.AddError(err)
	}
	if len(tags) > MaxTagCount {
		b.validator.AddFieldError("tags", "exceeds maximum tag count")
	}
	b.trace.Tags = tags
	return b
}

// Release sets the release version.
func (b *TraceBuilder) Release(release string) *TraceBuilder {
	b.trace.Release = release
	return b
}

// Version sets the version.
func (b *TraceBuilder) Version(version string) *TraceBuilder {
	b.trace.Version = version
	return b
}

// Public sets whether the trace is public.
func (b *TraceBuilder) Public(public bool) *TraceBuilder {
	b.trace.Public = public
	return b
}

// Environment sets the environment.
func (b *TraceBuilder) Environment(env string) *TraceBuilder {
	b.trace.Environment = env
	return b
}

// Clone creates a deep copy of the TraceBuilder with a new ID and timestamp.
// This is useful for creating multiple similar traces from a template.
//
// Example:
//
//	template := client.NewTrace().
//	    UserID("user-123").
//	    Environment("production").
//	    Tags([]string{"api"})
//
//	trace1, _ := template.Clone().Name("request-1").Create(ctx)
//	trace2, _ := template.Clone().Name("request-2").Create(ctx)
func (b *TraceBuilder) Clone() *TraceBuilder {
	// Deep copy tags slice
	var tags []string
	if b.trace.Tags != nil {
		tags = make([]string, len(b.trace.Tags))
		copy(tags, b.trace.Tags)
	}

	// Deep copy metadata map
	var metadata Metadata
	if b.trace.Metadata != nil {
		metadata = make(Metadata, len(b.trace.Metadata))
		for k, v := range b.trace.Metadata {
			metadata[k] = v
		}
	}

	return &TraceBuilder{
		client: b.client,
		trace: &createTraceEvent{
			ID:          generateID(), // New ID for the clone
			Timestamp:   Now(),        // Fresh timestamp
			Name:        b.trace.Name,
			UserID:      b.trace.UserID,
			SessionID:   b.trace.SessionID,
			Input:       b.trace.Input,
			Output:      b.trace.Output,
			Metadata:    metadata,
			Tags:        tags,
			Release:     b.trace.Release,
			Version:     b.trace.Version,
			Public:      b.trace.Public,
			Environment: b.trace.Environment,
		},
	}
}

// HasErrors returns true if there are any validation errors.
func (b *TraceBuilder) HasErrors() bool {
	return b.validator.HasErrors()
}

// Errors returns all accumulated validation errors.
func (b *TraceBuilder) Errors() []error {
	return b.validator.Errors()
}

// Validate validates the trace builder configuration.
// It returns any errors accumulated during setting plus final validation checks.
func (b *TraceBuilder) Validate() error {
	// Check accumulated errors from setters
	if b.validator.HasErrors() {
		return b.validator.CombinedError()
	}

	// Final validation checks
	if b.trace.ID == "" {
		return NewValidationError("id", "trace ID cannot be empty")
	}
	return nil
}

// Create creates the trace and returns a TraceContext for adding observations.
func (b *TraceBuilder) Create(ctx context.Context) (*TraceContext, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}

	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeTraceCreate,
		Timestamp: Now(),
		Body:      b.trace,
	}

	if err := b.client.queueEvent(ctx, event); err != nil {
		return nil, err
	}

	return &TraceContext{
		client:  b.client,
		traceID: b.trace.ID,
	}, nil
}

// TraceContext provides context for a trace and allows adding observations.
//
// TraceContext is safe for concurrent use within a single trace. You can
// create multiple spans, generations, and events from the same TraceContext
// concurrently. However, individual builders created from TraceContext
// (via Span(), Generation(), etc.) are NOT safe for concurrent use.
type TraceContext struct {
	client  *Client
	traceID string
}

// ID returns the trace ID.
func (t *TraceContext) ID() string {
	return t.traceID
}

// TraceID returns the trace ID. This method satisfies the Observer interface.
// For TraceContext, this returns the same value as ID().
func (t *TraceContext) TraceID() string {
	return t.traceID
}

// Update updates the trace.
func (t *TraceContext) Update() *TraceUpdateBuilder {
	return &TraceUpdateBuilder{
		ctx: t,
		update: &updateTraceEvent{
			ID: t.traceID,
		},
	}
}

// NewSpan creates a new span builder in this trace (Advanced API).
// For the Simple API, use Span(ctx, name, ...opts).
func (t *TraceContext) NewSpan() *SpanBuilder {
	return &SpanBuilder{
		ctx: t,
		span: &createSpanEvent{
			ID:        generateID(),
			TraceID:   t.traceID,
			StartTime: Now(),
		},
	}
}

// NewGeneration creates a new generation builder in this trace (Advanced API).
// For the Simple API, use Generation(ctx, name, ...opts).
func (t *TraceContext) NewGeneration() *GenerationBuilder {
	return &GenerationBuilder{
		ctx: t,
		gen: &createGenerationEvent{
			ID:        generateID(),
			TraceID:   t.traceID,
			StartTime: Now(),
		},
	}
}

// NewEvent creates a new event builder in this trace (Advanced API).
// For the Simple API, use Event(ctx, name, ...opts).
func (t *TraceContext) NewEvent() *EventBuilder {
	return &EventBuilder{
		ctx: t,
		event: &createEventEvent{
			ID:        generateID(),
			TraceID:   t.traceID,
			StartTime: Now(),
		},
	}
}

// NewScore creates a new score builder for this trace (Advanced API).
// For the Simple API, use Score(ctx, name, value, ...opts).
func (t *TraceContext) NewScore() *ScoreBuilder {
	return &ScoreBuilder{
		ctx: t,
		score: &createScoreEvent{
			TraceID: t.traceID,
		},
	}
}

// ScoreNumeric adds a numeric score to this trace.
// This is a convenience method for the common case of adding a simple numeric score.
//
// Example:
//
//	trace.ScoreNumeric(ctx, "quality", 0.95)
func (t *TraceContext) ScoreNumeric(ctx context.Context, name string, value float64) error {
	return t.NewScore().Name(name).NumericValue(value).Create(ctx)
}

// ScoreCategorical adds a categorical score to this trace.
// This is a convenience method for the common case of adding a simple categorical score.
//
// Example:
//
//	trace.ScoreCategorical(ctx, "sentiment", "positive")
func (t *TraceContext) ScoreCategorical(ctx context.Context, name string, value string) error {
	return t.NewScore().Name(name).CategoricalValue(value).Create(ctx)
}

// ScoreBoolean adds a boolean score to this trace.
// This is a convenience method for the common case of adding a simple boolean score.
//
// Example:
//
//	trace.ScoreBoolean(ctx, "correct", true)
func (t *TraceContext) ScoreBoolean(ctx context.Context, name string, value bool) error {
	return t.NewScore().Name(name).BooleanValue(value).Create(ctx)
}

// TraceUpdateBuilder provides a fluent interface for updating traces.
//
// TraceUpdateBuilder is NOT safe for concurrent use. Each builder instance
// should be created, configured, and used within a single goroutine.
type TraceUpdateBuilder struct {
	ctx    *TraceContext
	update *updateTraceEvent
}

// Name sets the trace name.
func (b *TraceUpdateBuilder) Name(name string) *TraceUpdateBuilder {
	b.update.Name = name
	return b
}

// UserID sets the user ID.
func (b *TraceUpdateBuilder) UserID(userID string) *TraceUpdateBuilder {
	b.update.UserID = userID
	return b
}

// SessionID sets the session ID.
func (b *TraceUpdateBuilder) SessionID(sessionID string) *TraceUpdateBuilder {
	b.update.SessionID = sessionID
	return b
}

// Input sets the trace input.
func (b *TraceUpdateBuilder) Input(input any) *TraceUpdateBuilder {
	b.update.Input = input
	return b
}

// Output sets the trace output.
func (b *TraceUpdateBuilder) Output(output any) *TraceUpdateBuilder {
	b.update.Output = output
	return b
}

// Metadata sets the trace metadata.
func (b *TraceUpdateBuilder) Metadata(metadata Metadata) *TraceUpdateBuilder {
	b.update.Metadata = metadata
	return b
}

// Tags sets the trace tags.
func (b *TraceUpdateBuilder) Tags(tags []string) *TraceUpdateBuilder {
	b.update.Tags = tags
	return b
}

// Public sets whether the trace is public.
func (b *TraceUpdateBuilder) Public(public bool) *TraceUpdateBuilder {
	b.update.Public = public
	return b
}

// Apply applies the update.
func (b *TraceUpdateBuilder) Apply(ctx context.Context) error {
	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeTraceUpdate,
		Timestamp: Now(),
		Body:      b.update,
	}

	return b.ctx.client.queueEvent(ctx, event)
}
