package langfuse

import (
	"context"
	"errors"
	"fmt"
	"time"
)

// EndResult contains the result of ending an observation (span or generation).
// It provides a consistent return type for all End operations.
//
// Example:
//
//	result := span.EndWith(ctx, WithOutput(response))
//	if result.Error != nil {
//	    log.Printf("Failed to end span: %v", result.Error)
//	}
//	log.Printf("Span duration: %v", result.Duration)
type EndResult struct {
	// Duration is the time elapsed since the observation started.
	// This is only set if the observation had a start time.
	Duration time.Duration

	// Error is the error that occurred during the end operation, if any.
	Error error
}

// Ok returns true if the end operation succeeded.
func (r EndResult) Ok() bool {
	return r.Error == nil
}

// endConfig holds the configuration for ending an observation.
type endConfig struct {
	output              any
	inputTokens         int
	outputTokens        int
	hasUsage            bool
	metadata            Metadata
	level               ObservationLevel
	hasLevel            bool
	statusMessage       string
	completionStartTime time.Time
	hasCompletionStart  bool
	endTime             time.Time
	hasEndTime          bool
	observationError    error
}

// EndOption configures how an observation is ended.
type EndOption func(*endConfig)

// WithOutput sets the output for the observation.
//
// Example:
//
//	span.EndWith(ctx, WithOutput(result))
func WithOutput(output any) EndOption {
	return func(c *endConfig) {
		c.output = output
	}
}

// WithUsage sets the token usage for a generation.
// This option only applies to generations.
//
// Example:
//
//	gen.EndWith(ctx, WithUsage(100, 50))
func WithUsage(inputTokens, outputTokens int) EndOption {
	return func(c *endConfig) {
		c.inputTokens = inputTokens
		c.outputTokens = outputTokens
		c.hasUsage = true
	}
}

// WithEndMetadata sets metadata when ending an observation.
//
// Example:
//
//	span.EndWith(ctx, WithEndMetadata(Metadata{"latency_ms": 42}))
func WithEndMetadata(metadata Metadata) EndOption {
	return func(c *endConfig) {
		c.metadata = metadata
	}
}

// WithEndLevel sets the observation level when ending.
//
// Example:
//
//	span.EndWith(ctx, WithEndLevel(ObservationLevelError))
func WithEndLevel(level ObservationLevel) EndOption {
	return func(c *endConfig) {
		c.level = level
		c.hasLevel = true
	}
}

// WithStatusMessage sets a status message when ending.
//
// Example:
//
//	span.EndWith(ctx, WithStatusMessage("completed successfully"))
func WithStatusMessage(msg string) EndOption {
	return func(c *endConfig) {
		c.statusMessage = msg
	}
}

// WithCompletionStart sets when the completion started (for generations).
// This is useful for tracking time-to-first-token.
//
// Example:
//
//	gen.EndWith(ctx, WithCompletionStart(completionStartedAt))
func WithCompletionStart(t time.Time) EndOption {
	return func(c *endConfig) {
		c.completionStartTime = t
		c.hasCompletionStart = true
	}
}

// WithEndTime sets a specific end time instead of using time.Now().
//
// Example:
//
//	span.EndWith(ctx, WithEndTime(recordedEndTime))
func WithEndTime(t time.Time) EndOption {
	return func(c *endConfig) {
		c.endTime = t
		c.hasEndTime = true
	}
}

// WithError marks the observation as having an error.
// This sets the level to ERROR and includes the error message.
//
// Example:
//
//	span.EndWith(ctx, WithError(err))
func WithError(err error) EndOption {
	return func(c *endConfig) {
		c.observationError = err
		c.level = ObservationLevelError
		c.hasLevel = true
		if err != nil {
			c.statusMessage = err.Error()
		}
	}
}

// MetadataBuilder provides a type-safe way to build metadata with
// typed setter methods. This complements the Metadata type's Set() method
// by providing methods like String(), Int(), Float(), etc.
//
// Example:
//
//	metadata := BuildMetadata().
//	    String("user_id", "123").
//	    Int("request_count", 5).
//	    Bool("is_premium", true).
//	    Float("score", 0.95).
//	    Build()
//
//	trace.Metadata(metadata).Create(ctx)
type MetadataBuilder struct {
	data Metadata
}

// BuildMetadata creates a new MetadataBuilder with typed setter methods.
// For simple metadata construction, you can also use NewMetadata().Set() directly.
func BuildMetadata() *MetadataBuilder {
	return &MetadataBuilder{data: make(Metadata)}
}

// String adds a string value to the metadata.
func (m *MetadataBuilder) String(key, value string) *MetadataBuilder {
	m.data[key] = value
	return m
}

// Int adds an integer value to the metadata.
func (m *MetadataBuilder) Int(key string, value int) *MetadataBuilder {
	m.data[key] = value
	return m
}

// Int64 adds an int64 value to the metadata.
func (m *MetadataBuilder) Int64(key string, value int64) *MetadataBuilder {
	m.data[key] = value
	return m
}

// Float adds a float64 value to the metadata.
func (m *MetadataBuilder) Float(key string, value float64) *MetadataBuilder {
	m.data[key] = value
	return m
}

// Bool adds a boolean value to the metadata.
func (m *MetadataBuilder) Bool(key string, value bool) *MetadataBuilder {
	m.data[key] = value
	return m
}

// Time adds a time value to the metadata (formatted as RFC3339).
func (m *MetadataBuilder) Time(key string, value time.Time) *MetadataBuilder {
	m.data[key] = value.Format(time.RFC3339Nano)
	return m
}

// Duration adds a duration value to the metadata (as string).
func (m *MetadataBuilder) Duration(key string, value time.Duration) *MetadataBuilder {
	m.data[key] = value.String()
	return m
}

// DurationMs adds a duration value as milliseconds.
func (m *MetadataBuilder) DurationMs(key string, value time.Duration) *MetadataBuilder {
	m.data[key] = value.Milliseconds()
	return m
}

// JSON adds an arbitrary JSON-serializable value to the metadata.
func (m *MetadataBuilder) JSON(key string, value any) *MetadataBuilder {
	m.data[key] = value
	return m
}

// Strings adds a string slice to the metadata.
func (m *MetadataBuilder) Strings(key string, values []string) *MetadataBuilder {
	m.data[key] = values
	return m
}

// Map adds a nested map to the metadata.
func (m *MetadataBuilder) Map(key string, value map[string]any) *MetadataBuilder {
	m.data[key] = value
	return m
}

// Merge merges another metadata map into this builder.
// Existing keys are overwritten.
func (m *MetadataBuilder) Merge(other Metadata) *MetadataBuilder {
	for k, v := range other {
		m.data[k] = v
	}
	return m
}

// Build returns the constructed metadata map.
func (m *MetadataBuilder) Build() Metadata {
	return m.data
}

// TagsBuilder provides a type-safe way to build tags.
//
// Example:
//
//	tags := NewTags().
//	    Add("production").
//	    Add("api", "v2").
//	    AddIf(isPremium, "premium").
//	    Build()
//
//	trace.Tags(tags).Create(ctx)
type TagsBuilder struct {
	tags []string
}

// NewTags creates a new TagsBuilder.
func NewTags() *TagsBuilder {
	return &TagsBuilder{tags: make([]string, 0)}
}

// Add adds one or more tags.
func (t *TagsBuilder) Add(tags ...string) *TagsBuilder {
	t.tags = append(t.tags, tags...)
	return t
}

// AddIf conditionally adds a tag.
func (t *TagsBuilder) AddIf(condition bool, tag string) *TagsBuilder {
	if condition {
		t.tags = append(t.tags, tag)
	}
	return t
}

// AddIfNotEmpty adds a tag only if it's not empty.
func (t *TagsBuilder) AddIfNotEmpty(tag string) *TagsBuilder {
	if tag != "" {
		t.tags = append(t.tags, tag)
	}
	return t
}

// Environment adds an environment tag (e.g., "env:production").
func (t *TagsBuilder) Environment(env string) *TagsBuilder {
	if env != "" {
		t.tags = append(t.tags, "env:"+env)
	}
	return t
}

// Version adds a version tag (e.g., "version:1.2.3").
func (t *TagsBuilder) Version(version string) *TagsBuilder {
	if version != "" {
		t.tags = append(t.tags, "version:"+version)
	}
	return t
}

// Build returns the constructed tags slice.
func (t *TagsBuilder) Build() []string {
	return t.tags
}

// BatchTraceBuilder provides a way to create multiple traces efficiently.
//
// Example:
//
//	batch := client.BatchTraces()
//	batch.Add("trace-1").UserID("user-1").Tags("api", "v2")
//	batch.Add("trace-2").UserID("user-2").Tags("api", "v2")
//	batch.Add("trace-3").UserID("user-3").Tags("api", "v2")
//
//	traces, err := batch.Create(ctx)
//	if err != nil {
//	    // Handle partial failures
//	    var batchErr *BatchError
//	    if errors.As(err, &batchErr) {
//	        log.Printf("Created %d/%d traces", batchErr.Succeeded, batchErr.Total)
//	    }
//	}
type BatchTraceBuilder struct {
	client  *Client
	traces  []*TraceBuilder
	options batchOptions
}

// batchOptions configures batch behavior.
type batchOptions struct {
	stopOnError bool
}

// BatchError is returned when batch operations partially fail.
type BatchError struct {
	// Total is the total number of items attempted.
	Total int

	// Succeeded is the number of items that succeeded.
	Succeeded int

	// Errors contains the individual errors, indexed by position.
	Errors map[int]error
}

// Error implements the error interface.
func (e *BatchError) Error() string {
	return fmt.Sprintf("batch operation: %d/%d succeeded, %d failed",
		e.Succeeded, e.Total, len(e.Errors))
}

// FirstError returns the first error encountered.
func (e *BatchError) FirstError() error {
	for i := 0; i < e.Total; i++ {
		if err, ok := e.Errors[i]; ok {
			return err
		}
	}
	return nil
}

// BatchTraces creates a new batch trace builder.
//
// Example:
//
//	batch := client.BatchTraces()
//	batch.Add("request-1").UserID("user-1")
//	batch.Add("request-2").UserID("user-2")
//	traces, err := batch.Create(ctx)
func (c *Client) BatchTraces() *BatchTraceBuilder {
	return &BatchTraceBuilder{
		client: c,
		traces: make([]*TraceBuilder, 0),
	}
}

// Add adds a new trace to the batch and returns its builder for configuration.
func (b *BatchTraceBuilder) Add(name string) *TraceBuilder {
	tb := b.client.NewTrace().Name(name)
	b.traces = append(b.traces, tb)
	return tb
}

// AddBuilder adds an existing trace builder to the batch.
func (b *BatchTraceBuilder) AddBuilder(tb *TraceBuilder) *BatchTraceBuilder {
	b.traces = append(b.traces, tb)
	return b
}

// StopOnError configures the batch to stop on the first error.
// By default, the batch continues and collects all errors.
func (b *BatchTraceBuilder) StopOnError() *BatchTraceBuilder {
	b.options.stopOnError = true
	return b
}

// Len returns the number of traces in the batch.
func (b *BatchTraceBuilder) Len() int {
	return len(b.traces)
}

// Create creates all traces in the batch.
// Returns a slice of TraceContexts (nil for failed traces) and any error.
// If some traces fail, a BatchError is returned containing details.
func (b *BatchTraceBuilder) Create(ctx context.Context) ([]*TraceContext, error) {
	if len(b.traces) == 0 {
		return nil, nil
	}

	results := make([]*TraceContext, len(b.traces))
	errs := make(map[int]error)
	succeeded := 0

	for i, tb := range b.traces {
		tc, err := tb.Create(ctx)
		if err != nil {
			errs[i] = err
			if b.options.stopOnError {
				return results, &BatchError{
					Total:     len(b.traces),
					Succeeded: succeeded,
					Errors:    errs,
				}
			}
		} else {
			results[i] = tc
			succeeded++
		}
	}

	if len(errs) > 0 {
		return results, &BatchError{
			Total:     len(b.traces),
			Succeeded: succeeded,
			Errors:    errs,
		}
	}

	return results, nil
}

// BatchSpanBuilder provides a way to create multiple spans efficiently.
type BatchSpanBuilder struct {
	trace   *TraceContext
	spans   []*SpanBuilder
	options batchOptions
}

// BatchSpans creates a new batch span builder from a trace.
//
// Example:
//
//	batch := trace.BatchSpans()
//	batch.Add("step-1").Input(input1)
//	batch.Add("step-2").Input(input2)
//	spans, err := batch.Create(ctx)
func (t *TraceContext) BatchSpans() *BatchSpanBuilder {
	return &BatchSpanBuilder{
		trace: t,
		spans: make([]*SpanBuilder, 0),
	}
}

// Add adds a new span to the batch.
func (b *BatchSpanBuilder) Add(name string) *SpanBuilder {
	sb := b.trace.NewSpan().Name(name)
	b.spans = append(b.spans, sb)
	return sb
}

// AddBuilder adds an existing span builder to the batch.
func (b *BatchSpanBuilder) AddBuilder(sb *SpanBuilder) *BatchSpanBuilder {
	b.spans = append(b.spans, sb)
	return b
}

// StopOnError configures the batch to stop on the first error.
func (b *BatchSpanBuilder) StopOnError() *BatchSpanBuilder {
	b.options.stopOnError = true
	return b
}

// Len returns the number of spans in the batch.
func (b *BatchSpanBuilder) Len() int {
	return len(b.spans)
}

// Create creates all spans in the batch.
func (b *BatchSpanBuilder) Create(ctx context.Context) ([]*SpanContext, error) {
	if len(b.spans) == 0 {
		return nil, nil
	}

	results := make([]*SpanContext, len(b.spans))
	errs := make(map[int]error)
	succeeded := 0

	for i, sb := range b.spans {
		sc, err := sb.Create(ctx)
		if err != nil {
			errs[i] = err
			if b.options.stopOnError {
				return results, &BatchError{
					Total:     len(b.spans),
					Succeeded: succeeded,
					Errors:    errs,
				}
			}
		} else {
			results[i] = sc
			succeeded++
		}
	}

	if len(errs) > 0 {
		return results, &BatchError{
			Total:     len(b.spans),
			Succeeded: succeeded,
			Errors:    errs,
		}
	}

	return results, nil
}

// UsageBuilder provides a type-safe way to build token usage.
//
// Example:
//
//	usage := NewUsage().
//	    Input(100).
//	    Output(50).
//	    InputCost(0.001).
//	    OutputCost(0.002).
//	    Build()
//
//	gen.Usage(usage).Create(ctx)
type UsageBuilder struct {
	usage Usage
}

// NewUsage creates a new UsageBuilder.
func NewUsage() *UsageBuilder {
	return &UsageBuilder{}
}

// Input sets the input token count.
func (u *UsageBuilder) Input(tokens int) *UsageBuilder {
	u.usage.Input = tokens
	u.usage.Total = u.usage.Input + u.usage.Output
	return u
}

// Output sets the output token count.
func (u *UsageBuilder) Output(tokens int) *UsageBuilder {
	u.usage.Output = tokens
	u.usage.Total = u.usage.Input + u.usage.Output
	return u
}

// Total sets the total token count explicitly.
// If not set, it's calculated as Input + Output.
func (u *UsageBuilder) Total(tokens int) *UsageBuilder {
	u.usage.Total = tokens
	return u
}

// Unit sets the usage unit (e.g., "TOKENS", "CHARACTERS").
func (u *UsageBuilder) Unit(unit string) *UsageBuilder {
	u.usage.Unit = unit
	return u
}

// InputCost sets the input cost.
func (u *UsageBuilder) InputCost(cost float64) *UsageBuilder {
	u.usage.InputCost = cost
	u.usage.TotalCost = u.usage.InputCost + u.usage.OutputCost
	return u
}

// OutputCost sets the output cost.
func (u *UsageBuilder) OutputCost(cost float64) *UsageBuilder {
	u.usage.OutputCost = cost
	u.usage.TotalCost = u.usage.InputCost + u.usage.OutputCost
	return u
}

// TotalCost sets the total cost explicitly.
func (u *UsageBuilder) TotalCost(cost float64) *UsageBuilder {
	u.usage.TotalCost = cost
	return u
}

// Build returns the constructed Usage.
func (u *UsageBuilder) Build() *Usage {
	return &u.usage
}

// Tokens is a convenience method to set both input and output tokens.
func (u *UsageBuilder) Tokens(input, output int) *UsageBuilder {
	u.usage.Input = input
	u.usage.Output = output
	u.usage.Total = input + output
	return u
}

// ModelParametersBuilder provides a type-safe way to build model parameters.
//
// Example:
//
//	params := NewModelParameters().
//	    Temperature(0.7).
//	    MaxTokens(150).
//	    TopP(0.9).
//	    Build()
//
//	gen.ModelParameters(params).Create(ctx)
type ModelParametersBuilder struct {
	params Metadata
}

// NewModelParameters creates a new ModelParametersBuilder.
func NewModelParameters() *ModelParametersBuilder {
	return &ModelParametersBuilder{params: make(Metadata)}
}

// Temperature sets the temperature parameter.
func (m *ModelParametersBuilder) Temperature(temp float64) *ModelParametersBuilder {
	m.params["temperature"] = temp
	return m
}

// MaxTokens sets the max_tokens parameter.
func (m *ModelParametersBuilder) MaxTokens(tokens int) *ModelParametersBuilder {
	m.params["max_tokens"] = tokens
	return m
}

// TopP sets the top_p parameter.
func (m *ModelParametersBuilder) TopP(p float64) *ModelParametersBuilder {
	m.params["top_p"] = p
	return m
}

// TopK sets the top_k parameter.
func (m *ModelParametersBuilder) TopK(k int) *ModelParametersBuilder {
	m.params["top_k"] = k
	return m
}

// FrequencyPenalty sets the frequency_penalty parameter.
func (m *ModelParametersBuilder) FrequencyPenalty(penalty float64) *ModelParametersBuilder {
	m.params["frequency_penalty"] = penalty
	return m
}

// PresencePenalty sets the presence_penalty parameter.
func (m *ModelParametersBuilder) PresencePenalty(penalty float64) *ModelParametersBuilder {
	m.params["presence_penalty"] = penalty
	return m
}

// Stop sets the stop sequences.
func (m *ModelParametersBuilder) Stop(sequences ...string) *ModelParametersBuilder {
	m.params["stop"] = sequences
	return m
}

// Seed sets the seed for deterministic outputs.
func (m *ModelParametersBuilder) Seed(seed int) *ModelParametersBuilder {
	m.params["seed"] = seed
	return m
}

// ResponseFormat sets the response format.
func (m *ModelParametersBuilder) ResponseFormat(format string) *ModelParametersBuilder {
	m.params["response_format"] = map[string]string{"type": format}
	return m
}

// Set sets an arbitrary parameter.
func (m *ModelParametersBuilder) Set(key string, value any) *ModelParametersBuilder {
	m.params[key] = value
	return m
}

// Merge merges another parameters map.
func (m *ModelParametersBuilder) Merge(other Metadata) *ModelParametersBuilder {
	for k, v := range other {
		m.params[k] = v
	}
	return m
}

// Build returns the constructed parameters map.
func (m *ModelParametersBuilder) Build() Metadata {
	return m.params
}

// IsBatchError checks if an error is a BatchError.
func IsBatchError(err error) bool {
	var batchErr *BatchError
	return errors.As(err, &batchErr)
}

// AsBatchError returns the BatchError if err is one, nil otherwise.
func AsBatchError(err error) *BatchError {
	var batchErr *BatchError
	if errors.As(err, &batchErr) {
		return batchErr
	}
	return nil
}
