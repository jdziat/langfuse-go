package langfuse

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// Event types for the ingestion API
const (
	eventTypeTraceCreate      = "trace-create"
	eventTypeTraceUpdate      = "trace-update"
	eventTypeSpanCreate       = "span-create"
	eventTypeSpanUpdate       = "span-update"
	eventTypeGenerationCreate = "generation-create"
	eventTypeGenerationUpdate = "generation-update"
	eventTypeEventCreate      = "event-create"
	eventTypeScoreCreate      = "score-create"
	eventTypeSDKLog           = "sdk-log"
)

// ingestionRequest represents a batch ingestion request.
type ingestionRequest struct {
	Batch    []ingestionEvent       `json:"batch"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ingestionEvent represents a single event in a batch.
type ingestionEvent struct {
	ID        string      `json:"id"`
	Type      string      `json:"type"`
	Timestamp Time        `json:"timestamp"`
	Body      interface{} `json:"body"`
}

// createTraceEvent represents the body of a trace-create event.
type createTraceEvent struct {
	ID          string                 `json:"id"`
	Timestamp   Time                   `json:"timestamp,omitempty"`
	Name        string                 `json:"name,omitempty"`
	UserID      string                 `json:"userId,omitempty"`
	Input       interface{}            `json:"input,omitempty"`
	Output      interface{}            `json:"output,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	SessionID   string                 `json:"sessionId,omitempty"`
	Release     string                 `json:"release,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Public      bool                   `json:"public,omitempty"`
	Environment string                 `json:"environment,omitempty"`
}

// updateTraceEvent represents the body of a trace-update event.
type updateTraceEvent struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name,omitempty"`
	UserID      string                 `json:"userId,omitempty"`
	Input       interface{}            `json:"input,omitempty"`
	Output      interface{}            `json:"output,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	SessionID   string                 `json:"sessionId,omitempty"`
	Release     string                 `json:"release,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Public      bool                   `json:"public,omitempty"`
	Environment string                 `json:"environment,omitempty"`
}

// createSpanEvent represents the body of a span-create event.
type createSpanEvent struct {
	ID                  string                 `json:"id"`
	TraceID             string                 `json:"traceId,omitempty"`
	Name                string                 `json:"name,omitempty"`
	StartTime           Time                   `json:"startTime,omitempty"`
	EndTime             Time                   `json:"endTime,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	Level               ObservationLevel       `json:"level,omitempty"`
	StatusMessage       string                 `json:"statusMessage,omitempty"`
	ParentObservationID string                 `json:"parentObservationId,omitempty"`
	Version             string                 `json:"version,omitempty"`
	Input               interface{}            `json:"input,omitempty"`
	Output              interface{}            `json:"output,omitempty"`
	Environment         string                 `json:"environment,omitempty"`
}

// updateSpanEvent represents the body of a span-update event.
type updateSpanEvent struct {
	ID            string                 `json:"id"`
	TraceID       string                 `json:"traceId,omitempty"`
	Name          string                 `json:"name,omitempty"`
	EndTime       Time                   `json:"endTime,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
	Level         ObservationLevel       `json:"level,omitempty"`
	StatusMessage string                 `json:"statusMessage,omitempty"`
	Version       string                 `json:"version,omitempty"`
	Input         interface{}            `json:"input,omitempty"`
	Output        interface{}            `json:"output,omitempty"`
	Environment   string                 `json:"environment,omitempty"`
}

// createGenerationEvent represents the body of a generation-create event.
type createGenerationEvent struct {
	ID                  string                 `json:"id"`
	TraceID             string                 `json:"traceId,omitempty"`
	Name                string                 `json:"name,omitempty"`
	StartTime           Time                   `json:"startTime,omitempty"`
	EndTime             Time                   `json:"endTime,omitempty"`
	CompletionStartTime Time                   `json:"completionStartTime,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	Level               ObservationLevel       `json:"level,omitempty"`
	StatusMessage       string                 `json:"statusMessage,omitempty"`
	ParentObservationID string                 `json:"parentObservationId,omitempty"`
	Version             string                 `json:"version,omitempty"`
	Input               interface{}            `json:"input,omitempty"`
	Output              interface{}            `json:"output,omitempty"`
	Model               string                 `json:"model,omitempty"`
	ModelParameters     map[string]interface{} `json:"modelParameters,omitempty"`
	Usage               *Usage                 `json:"usage,omitempty"`
	PromptName          string                 `json:"promptName,omitempty"`
	PromptVersion       int                    `json:"promptVersion,omitempty"`
	Environment         string                 `json:"environment,omitempty"`
}

// updateGenerationEvent represents the body of a generation-update event.
type updateGenerationEvent struct {
	ID                  string                 `json:"id"`
	TraceID             string                 `json:"traceId,omitempty"`
	Name                string                 `json:"name,omitempty"`
	EndTime             Time                   `json:"endTime,omitempty"`
	CompletionStartTime Time                   `json:"completionStartTime,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	Level               ObservationLevel       `json:"level,omitempty"`
	StatusMessage       string                 `json:"statusMessage,omitempty"`
	Version             string                 `json:"version,omitempty"`
	Input               interface{}            `json:"input,omitempty"`
	Output              interface{}            `json:"output,omitempty"`
	Model               string                 `json:"model,omitempty"`
	ModelParameters     map[string]interface{} `json:"modelParameters,omitempty"`
	Usage               *Usage                 `json:"usage,omitempty"`
	PromptName          string                 `json:"promptName,omitempty"`
	PromptVersion       int                    `json:"promptVersion,omitempty"`
	Environment         string                 `json:"environment,omitempty"`
}

// createEventEvent represents the body of an event-create event.
type createEventEvent struct {
	ID                  string                 `json:"id"`
	TraceID             string                 `json:"traceId,omitempty"`
	Name                string                 `json:"name,omitempty"`
	StartTime           Time                   `json:"startTime,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	Level               ObservationLevel       `json:"level,omitempty"`
	StatusMessage       string                 `json:"statusMessage,omitempty"`
	ParentObservationID string                 `json:"parentObservationId,omitempty"`
	Version             string                 `json:"version,omitempty"`
	Input               interface{}            `json:"input,omitempty"`
	Output              interface{}            `json:"output,omitempty"`
	Environment         string                 `json:"environment,omitempty"`
}

// createScoreEvent represents the body of a score-create event.
type createScoreEvent struct {
	ID            string                 `json:"id,omitempty"`
	TraceID       string                 `json:"traceId"`
	ObservationID string                 `json:"observationId,omitempty"`
	Name          string                 `json:"name"`
	Value         interface{}            `json:"value"`
	StringValue   string                 `json:"stringValue,omitempty"`
	DataType      ScoreDataType          `json:"dataType,omitempty"`
	Comment       string                 `json:"comment,omitempty"`
	ConfigID      string                 `json:"configId,omitempty"`
	Environment   string                 `json:"environment,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// SpanBuilder provides a fluent interface for creating spans.
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
func (b *SpanBuilder) Input(input interface{}) *SpanBuilder {
	b.span.Input = input
	return b
}

// Output sets the output.
func (b *SpanBuilder) Output(output interface{}) *SpanBuilder {
	b.span.Output = output
	return b
}

// Metadata sets the metadata.
func (b *SpanBuilder) Metadata(metadata map[string]interface{}) *SpanBuilder {
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

// Create creates the span and returns a SpanContext.
func (b *SpanBuilder) Create() (*SpanContext, error) {
	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeSpanCreate,
		Timestamp: Now(),
		Body:      b.span,
	}

	if err := b.ctx.client.queueEvent(event); err != nil {
		return nil, err
	}

	return &SpanContext{
		TraceContext: b.ctx,
		spanID:       b.span.ID,
	}, nil
}

// SpanContext provides context for a span.
type SpanContext struct {
	*TraceContext
	spanID string
}

// SpanID returns the span ID.
func (s *SpanContext) SpanID() string {
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
func (s *SpanContext) End() error {
	return s.Update().EndTime(time.Now()).Apply()
}

// EndWithOutput ends the span with output and the current time.
func (s *SpanContext) EndWithOutput(output interface{}) error {
	return s.Update().Output(output).EndTime(time.Now()).Apply()
}

// Span creates a child span.
func (s *SpanContext) Span() *SpanBuilder {
	builder := s.TraceContext.Span()
	builder.span.ParentObservationID = s.spanID
	return builder
}

// Generation creates a child generation.
func (s *SpanContext) Generation() *GenerationBuilder {
	builder := s.TraceContext.Generation()
	builder.gen.ParentObservationID = s.spanID
	return builder
}

// Event creates a child event.
func (s *SpanContext) Event() *EventBuilder {
	builder := s.TraceContext.Event()
	builder.event.ParentObservationID = s.spanID
	return builder
}

// Score creates a score for this span.
func (s *SpanContext) Score() *ScoreBuilder {
	builder := s.TraceContext.Score()
	builder.score.ObservationID = s.spanID
	return builder
}

// SpanUpdateBuilder provides a fluent interface for updating spans.
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
func (b *SpanUpdateBuilder) Input(input interface{}) *SpanUpdateBuilder {
	b.update.Input = input
	return b
}

// Output sets the output.
func (b *SpanUpdateBuilder) Output(output interface{}) *SpanUpdateBuilder {
	b.update.Output = output
	return b
}

// Metadata sets the metadata.
func (b *SpanUpdateBuilder) Metadata(metadata map[string]interface{}) *SpanUpdateBuilder {
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
func (b *SpanUpdateBuilder) Apply() error {
	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeSpanUpdate,
		Timestamp: Now(),
		Body:      b.update,
	}

	return b.ctx.client.queueEvent(event)
}

// GenerationBuilder provides a fluent interface for creating generations.
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
func (b *GenerationBuilder) Input(input interface{}) *GenerationBuilder {
	b.gen.Input = input
	return b
}

// Output sets the output.
func (b *GenerationBuilder) Output(output interface{}) *GenerationBuilder {
	b.gen.Output = output
	return b
}

// Metadata sets the metadata.
func (b *GenerationBuilder) Metadata(metadata map[string]interface{}) *GenerationBuilder {
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
func (b *GenerationBuilder) ModelParameters(params map[string]interface{}) *GenerationBuilder {
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

// Create creates the generation and returns a GenerationContext.
func (b *GenerationBuilder) Create() (*GenerationContext, error) {
	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeGenerationCreate,
		Timestamp: Now(),
		Body:      b.gen,
	}

	if err := b.ctx.client.queueEvent(event); err != nil {
		return nil, err
	}

	return &GenerationContext{
		TraceContext: b.ctx,
		genID:        b.gen.ID,
	}, nil
}

// GenerationContext provides context for a generation.
type GenerationContext struct {
	*TraceContext
	genID string
}

// GenerationID returns the generation ID.
func (g *GenerationContext) GenerationID() string {
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
func (g *GenerationContext) End() error {
	return g.Update().EndTime(time.Now()).Apply()
}

// EndWithOutput ends the generation with output and the current time.
func (g *GenerationContext) EndWithOutput(output interface{}) error {
	return g.Update().Output(output).EndTime(time.Now()).Apply()
}

// EndWithUsage ends the generation with output, usage, and the current time.
func (g *GenerationContext) EndWithUsage(output interface{}, inputTokens, outputTokens int) error {
	return g.Update().
		Output(output).
		UsageTokens(inputTokens, outputTokens).
		EndTime(time.Now()).
		Apply()
}

// Score creates a score for this generation.
func (g *GenerationContext) Score() *ScoreBuilder {
	builder := g.TraceContext.Score()
	builder.score.ObservationID = g.genID
	return builder
}

// GenerationUpdateBuilder provides a fluent interface for updating generations.
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
func (b *GenerationUpdateBuilder) Input(input interface{}) *GenerationUpdateBuilder {
	b.update.Input = input
	return b
}

// Output sets the output.
func (b *GenerationUpdateBuilder) Output(output interface{}) *GenerationUpdateBuilder {
	b.update.Output = output
	return b
}

// Metadata sets the metadata.
func (b *GenerationUpdateBuilder) Metadata(metadata map[string]interface{}) *GenerationUpdateBuilder {
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
func (b *GenerationUpdateBuilder) ModelParameters(params map[string]interface{}) *GenerationUpdateBuilder {
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
func (b *GenerationUpdateBuilder) Apply() error {
	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeGenerationUpdate,
		Timestamp: Now(),
		Body:      b.update,
	}

	return b.ctx.client.queueEvent(event)
}

// EventBuilder provides a fluent interface for creating events.
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
func (b *EventBuilder) Input(input interface{}) *EventBuilder {
	b.event.Input = input
	return b
}

// Output sets the output.
func (b *EventBuilder) Output(output interface{}) *EventBuilder {
	b.event.Output = output
	return b
}

// Metadata sets the metadata.
func (b *EventBuilder) Metadata(metadata map[string]interface{}) *EventBuilder {
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

// Create creates the event.
func (b *EventBuilder) Create() error {
	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeEventCreate,
		Timestamp: Now(),
		Body:      b.event,
	}

	return b.ctx.client.queueEvent(event)
}

// ScoreBuilder provides a fluent interface for creating scores.
type ScoreBuilder struct {
	ctx   *TraceContext
	score *createScoreEvent
}

// ID sets the score ID.
func (b *ScoreBuilder) ID(id string) *ScoreBuilder {
	b.score.ID = id
	return b
}

// Name sets the score name.
func (b *ScoreBuilder) Name(name string) *ScoreBuilder {
	b.score.Name = name
	return b
}

// Value sets the score value.
func (b *ScoreBuilder) Value(value interface{}) *ScoreBuilder {
	b.score.Value = value
	return b
}

// NumericValue sets a numeric score value.
func (b *ScoreBuilder) NumericValue(value float64) *ScoreBuilder {
	b.score.Value = value
	b.score.DataType = ScoreDataTypeNumeric
	return b
}

// CategoricalValue sets a categorical score value.
func (b *ScoreBuilder) CategoricalValue(value string) *ScoreBuilder {
	b.score.StringValue = value
	b.score.DataType = ScoreDataTypeCategorical
	return b
}

// BooleanValue sets a boolean score value.
func (b *ScoreBuilder) BooleanValue(value bool) *ScoreBuilder {
	if value {
		b.score.Value = 1
	} else {
		b.score.Value = 0
	}
	b.score.DataType = ScoreDataTypeBoolean
	return b
}

// Comment sets the comment.
func (b *ScoreBuilder) Comment(comment string) *ScoreBuilder {
	b.score.Comment = comment
	return b
}

// ConfigID sets the score config ID.
func (b *ScoreBuilder) ConfigID(id string) *ScoreBuilder {
	b.score.ConfigID = id
	return b
}

// Environment sets the environment.
func (b *ScoreBuilder) Environment(env string) *ScoreBuilder {
	b.score.Environment = env
	return b
}

// Metadata sets the metadata.
func (b *ScoreBuilder) Metadata(metadata map[string]interface{}) *ScoreBuilder {
	b.score.Metadata = metadata
	return b
}

// ObservationID sets the observation ID.
func (b *ScoreBuilder) ObservationID(id string) *ScoreBuilder {
	b.score.ObservationID = id
	return b
}

// Create creates the score.
func (b *ScoreBuilder) Create() error {
	if b.score.Name == "" {
		return NewValidationError("name", "score name is required")
	}

	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeScoreCreate,
		Timestamp: Now(),
		Body:      b.score,
	}

	return b.ctx.client.queueEvent(event)
}

// generateID generates a random UUID-like ID.
func generateID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}
