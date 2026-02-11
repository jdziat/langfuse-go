package langfuse

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jdziat/langfuse-go/pkg/builders"
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

// ============================================================================
// Helper Builders - Re-exported from pkg/builders
// ============================================================================

// MetadataBuilder provides a type-safe way to build metadata with
// typed setter methods. Re-exported from pkg/builders.
type MetadataBuilder = builders.MetadataBuilder

// BuildMetadata creates a new MetadataBuilder with typed setter methods.
// For simple metadata construction, you can also use NewMetadata().Set() directly.
var BuildMetadata = builders.BuildMetadata

// TagsBuilder provides a type-safe way to build tags.
// Re-exported from pkg/builders.
type TagsBuilder = builders.TagsBuilder

// NewTags creates a new TagsBuilder.
var NewTags = builders.NewTags

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
// Re-exported from pkg/builders.
type UsageBuilder = builders.UsageBuilder

// NewUsage creates a new UsageBuilder.
var NewUsage = builders.NewUsage

// ModelParametersBuilder provides a type-safe way to build model parameters.
// Re-exported from pkg/builders.
type ModelParametersBuilder = builders.ModelParametersBuilder

// NewModelParameters creates a new ModelParametersBuilder.
var NewModelParameters = builders.NewModelParameters

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

// ============================================================================
// Validation Types and Functions (from validation.go)
// ============================================================================

// ValidationMode controls how validation errors are handled.
type ValidationMode int

const (
	// ValidationModeDeferred collects errors and reports them at Create() time.
	// This is the default mode that maintains the fluent API.
	ValidationModeDeferred ValidationMode = iota

	// ValidationModeImmediate causes setters to store errors immediately.
	// Errors must be checked with HasErrors() before calling Create().
	ValidationModeImmediate
)

// Validator provides validation methods for builder types.
// Builders can embed this to gain validation capabilities.
type Validator struct {
	errors []error
}

// AddError adds a validation error.
func (v *Validator) AddError(err error) {
	v.errors = append(v.errors, err)
}

// AddFieldError adds a validation error for a specific field.
func (v *Validator) AddFieldError(field, message string) {
	v.errors = append(v.errors, NewValidationError(field, message))
}

// HasErrors returns true if there are any validation errors.
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Errors returns all accumulated validation errors.
func (v *Validator) Errors() []error {
	return v.errors
}

// ClearErrors clears all validation errors.
func (v *Validator) ClearErrors() {
	v.errors = nil
}

// CombinedError returns a single error combining all validation errors,
// or nil if there are no errors.
func (v *Validator) CombinedError() error {
	if len(v.errors) == 0 {
		return nil
	}
	if len(v.errors) == 1 {
		return v.errors[0]
	}

	msgs := make([]string, len(v.errors))
	for i, err := range v.errors {
		msgs[i] = err.Error()
	}
	return fmt.Errorf("langfuse: multiple validation errors: %s", strings.Join(msgs, "; "))
}

// Validation rules

// ValidateID validates an ID field.
// IDs must be non-empty and either valid UUIDs or custom identifiers.
func ValidateID(field, value string) error {
	if value == "" {
		return NewValidationError(field, "cannot be empty")
	}
	// Allow any non-empty string as ID (UUIDs and custom IDs are both valid)
	return nil
}

// ValidateIDFormat validates that an ID is a valid UUID format.
// Use this when strict UUID format is required.
func ValidateIDFormat(field, value string) error {
	if value == "" {
		return NewValidationError(field, "cannot be empty")
	}
	if !IsValidUUID(value) {
		return NewValidationError(field, "must be a valid UUID format")
	}
	return nil
}

// ValidateName validates a name field.
// Names must be non-empty and within reasonable length.
func ValidateName(field, value string, maxLength int) error {
	if value == "" {
		return nil // Names are optional
	}
	if maxLength > 0 && utf8.RuneCountInString(value) > maxLength {
		return NewValidationError(field, fmt.Sprintf("exceeds maximum length of %d characters", maxLength))
	}
	return nil
}

// ValidateRequired validates that a required field is not empty.
func ValidateRequired(field, value string) error {
	if value == "" {
		return NewValidationError(field, "is required")
	}
	return nil
}

// ValidatePositive validates that a numeric field is positive.
func ValidatePositive(field string, value int) error {
	if value < 0 {
		return NewValidationError(field, "must be non-negative")
	}
	return nil
}

// ValidateRange validates that a numeric field is within a range.
func ValidateRange(field string, value, min, max int) error {
	if value < min || value > max {
		return NewValidationError(field, fmt.Sprintf("must be between %d and %d", min, max))
	}
	return nil
}

// ValidateMetadata validates metadata fields.
// Checks for nil keys or values that can't be serialized.
func ValidateMetadata(field string, metadata Metadata) error {
	if metadata == nil {
		return nil // nil metadata is valid
	}
	for key := range metadata {
		if key == "" {
			return NewValidationError(field, "metadata keys cannot be empty")
		}
	}
	return nil
}

// ValidateTags validates a tags slice.
func ValidateTags(field string, tags []string) error {
	if tags == nil {
		return nil // nil tags is valid
	}
	for i, tag := range tags {
		if tag == "" {
			return NewValidationError(field, fmt.Sprintf("tag at index %d cannot be empty", i))
		}
	}
	return nil
}

// ValidateLevel validates an observation level.
func ValidateLevel(field string, level ObservationLevel) error {
	switch level {
	case "", ObservationLevelDebug, ObservationLevelDefault, ObservationLevelWarning, ObservationLevelError:
		return nil
	default:
		return NewValidationError(field, fmt.Sprintf("invalid level: %s", level))
	}
}

// ValidateScoreValue validates a score value is within expected range.
func ValidateScoreValue(field string, value float64, min, max float64) error {
	if value < min || value > max {
		return NewValidationError(field, fmt.Sprintf("must be between %.2f and %.2f", min, max))
	}
	return nil
}

// ValidateDataType validates a score data type.
func ValidateDataType(field string, dataType ScoreDataType) error {
	switch dataType {
	case "", ScoreDataTypeNumeric, ScoreDataTypeCategorical, ScoreDataTypeBoolean:
		return nil
	default:
		return NewValidationError(field, fmt.Sprintf("invalid data type: %s", dataType))
	}
}

// MaxNameLength is the maximum allowed length for name fields.
const MaxNameLength = 500

// MaxTagLength is the maximum allowed length for individual tags.
const MaxTagLength = 100

// MaxTagCount is the maximum number of tags allowed.
const MaxTagCount = 50

// ============================================================================
// BuildResult and Validated Builders (from validated_builder.go)
// ============================================================================

// BuildResult wraps a result with its validation state.
// This pattern forces callers to handle validation by requiring
// explicit unwrapping of the result.
type BuildResult[T any] struct {
	value T
	err   error
}

// Unwrap returns the value and error, forcing error handling.
// This is the recommended way to use BuildResult.
//
// Example:
//
//	trace, err := builder.Create(ctx).Unwrap()
//	if err != nil {
//	    log.Printf("Failed to create trace: %v", err)
//	    return
//	}
func (r BuildResult[T]) Unwrap() (T, error) {
	return r.value, r.err
}

// Must returns the value or panics if there's an error.
// Use only in tests or when validation is guaranteed.
//
// Example:
//
//	// Only use in tests!
//	trace := builder.Create(ctx).Must()
func (r BuildResult[T]) Must() T {
	if r.err != nil {
		panic(r.err)
	}
	return r.value
}

// Ok returns true if there was no error.
func (r BuildResult[T]) Ok() bool {
	return r.err == nil
}

// Err returns the error, if any.
func (r BuildResult[T]) Err() error {
	return r.err
}

// Value returns the value without checking for errors.
// Prefer Unwrap() for safe access.
func (r BuildResult[T]) Value() T {
	return r.value
}

// NewBuildResult creates a new BuildResult with a value.
func NewBuildResult[T any](value T, err error) BuildResult[T] {
	return BuildResult[T]{value: value, err: err}
}

// BuildResultError creates a BuildResult with only an error.
func BuildResultError[T any](err error) BuildResult[T] {
	var zero T
	return BuildResult[T]{value: zero, err: err}
}

// BuildResultOk creates a BuildResult with only a value.
func BuildResultOk[T any](value T) BuildResult[T] {
	return BuildResult[T]{value: value, err: nil}
}

// ValidatedTraceBuilder wraps TraceBuilder with compile-time validation enforcement.
// All setter methods immediately validate input and accumulate errors.
// The Create method returns a BuildResult that must be unwrapped.
type ValidatedTraceBuilder struct {
	builder *TraceBuilder
	errors  []error
	client  *Client
}

// NewValidatedTraceBuilder creates a new validated trace builder.
// Use client.NewTraceStrict() instead of calling this directly.
func NewValidatedTraceBuilder(client *Client) *ValidatedTraceBuilder {
	return &ValidatedTraceBuilder{
		builder: client.NewTrace(),
		errors:  make([]error, 0),
		client:  client,
	}
}

// ID sets the trace ID with immediate validation.
func (b *ValidatedTraceBuilder) ID(id string) *ValidatedTraceBuilder {
	if id == "" {
		b.errors = append(b.errors, NewValidationError("id", "cannot be empty"))
	} else {
		b.builder.ID(id)
	}
	return b
}

// Name sets the trace name with immediate validation.
func (b *ValidatedTraceBuilder) Name(name string) *ValidatedTraceBuilder {
	if err := ValidateName("name", name, MaxNameLength); err != nil {
		b.errors = append(b.errors, err)
	} else {
		b.builder.Name(name)
	}
	return b
}

// UserID sets the user ID.
func (b *ValidatedTraceBuilder) UserID(userID string) *ValidatedTraceBuilder {
	b.builder.UserID(userID)
	return b
}

// SessionID sets the session ID.
func (b *ValidatedTraceBuilder) SessionID(sessionID string) *ValidatedTraceBuilder {
	b.builder.SessionID(sessionID)
	return b
}

// Input sets the trace input.
func (b *ValidatedTraceBuilder) Input(input any) *ValidatedTraceBuilder {
	b.builder.Input(input)
	return b
}

// Output sets the trace output.
func (b *ValidatedTraceBuilder) Output(output any) *ValidatedTraceBuilder {
	b.builder.Output(output)
	return b
}

// Metadata sets the trace metadata with validation.
func (b *ValidatedTraceBuilder) Metadata(metadata map[string]any) *ValidatedTraceBuilder {
	if err := ValidateMetadata("metadata", metadata); err != nil {
		b.errors = append(b.errors, err)
	} else {
		b.builder.Metadata(metadata)
	}
	return b
}

// Tags sets the trace tags with immediate validation.
func (b *ValidatedTraceBuilder) Tags(tags []string) *ValidatedTraceBuilder {
	if len(tags) > MaxTagCount {
		b.errors = append(b.errors, NewValidationError("tags",
			fmt.Sprintf("exceeds maximum count of %d", MaxTagCount)))
	}
	if err := ValidateTags("tags", tags); err != nil {
		b.errors = append(b.errors, err)
	}
	if len(b.errors) == 0 || (len(tags) <= MaxTagCount && ValidateTags("tags", tags) == nil) {
		b.builder.Tags(tags)
	}
	return b
}

// Version sets the trace version.
func (b *ValidatedTraceBuilder) Version(version string) *ValidatedTraceBuilder {
	b.builder.Version(version)
	return b
}

// Release sets the trace release.
func (b *ValidatedTraceBuilder) Release(release string) *ValidatedTraceBuilder {
	b.builder.Release(release)
	return b
}

// Public sets whether the trace is publicly accessible.
func (b *ValidatedTraceBuilder) Public(public bool) *ValidatedTraceBuilder {
	b.builder.Public(public)
	return b
}

// HasErrors returns true if any validation errors have been accumulated.
func (b *ValidatedTraceBuilder) HasErrors() bool {
	return len(b.errors) > 0
}

// Errors returns all accumulated validation errors.
func (b *ValidatedTraceBuilder) Errors() []error {
	return b.errors
}

// Create creates the trace, returning a BuildResult that must be unwrapped.
// All accumulated validation errors are combined and returned.
func (b *ValidatedTraceBuilder) Create(ctx context.Context) BuildResult[*TraceContext] {
	if len(b.errors) > 0 {
		return BuildResultError[*TraceContext](combineValidationErrors(b.errors))
	}
	trace, err := b.builder.Create(ctx)
	return NewBuildResult(trace, err)
}

// ValidatedSpanBuilder wraps SpanBuilder with compile-time validation enforcement.
type ValidatedSpanBuilder struct {
	builder *SpanBuilder
	errors  []error
}

// NewValidatedSpanBuilder creates a new validated span builder.
func NewValidatedSpanBuilder(trace *TraceContext) *ValidatedSpanBuilder {
	return &ValidatedSpanBuilder{
		builder: trace.NewSpan(),
		errors:  make([]error, 0),
	}
}

// ID sets the span ID with validation.
func (b *ValidatedSpanBuilder) ID(id string) *ValidatedSpanBuilder {
	if id == "" {
		b.errors = append(b.errors, NewValidationError("id", "cannot be empty"))
	} else {
		b.builder.ID(id)
	}
	return b
}

// Name sets the span name with validation.
func (b *ValidatedSpanBuilder) Name(name string) *ValidatedSpanBuilder {
	if err := ValidateName("name", name, MaxNameLength); err != nil {
		b.errors = append(b.errors, err)
	} else {
		b.builder.Name(name)
	}
	return b
}

// Input sets the span input.
func (b *ValidatedSpanBuilder) Input(input any) *ValidatedSpanBuilder {
	b.builder.Input(input)
	return b
}

// Output sets the span output.
func (b *ValidatedSpanBuilder) Output(output any) *ValidatedSpanBuilder {
	b.builder.Output(output)
	return b
}

// Metadata sets the span metadata with validation.
func (b *ValidatedSpanBuilder) Metadata(metadata map[string]any) *ValidatedSpanBuilder {
	if err := ValidateMetadata("metadata", metadata); err != nil {
		b.errors = append(b.errors, err)
	} else {
		b.builder.Metadata(metadata)
	}
	return b
}

// Level sets the span level with validation.
func (b *ValidatedSpanBuilder) Level(level ObservationLevel) *ValidatedSpanBuilder {
	if err := ValidateLevel("level", level); err != nil {
		b.errors = append(b.errors, err)
	} else {
		b.builder.Level(level)
	}
	return b
}

// StatusMessage sets the span status message.
func (b *ValidatedSpanBuilder) StatusMessage(statusMessage string) *ValidatedSpanBuilder {
	b.builder.StatusMessage(statusMessage)
	return b
}

// Version sets the span version.
func (b *ValidatedSpanBuilder) Version(version string) *ValidatedSpanBuilder {
	b.builder.Version(version)
	return b
}

// HasErrors returns true if any validation errors have been accumulated.
func (b *ValidatedSpanBuilder) HasErrors() bool {
	return len(b.errors) > 0
}

// Errors returns all accumulated validation errors.
func (b *ValidatedSpanBuilder) Errors() []error {
	return b.errors
}

// Create creates the span, returning a BuildResult that must be unwrapped.
func (b *ValidatedSpanBuilder) Create(ctx context.Context) BuildResult[*SpanContext] {
	if len(b.errors) > 0 {
		return BuildResultError[*SpanContext](combineValidationErrors(b.errors))
	}
	span, err := b.builder.Create(ctx)
	return NewBuildResult(span, err)
}

// ValidatedGenerationBuilder wraps GenerationBuilder with compile-time validation enforcement.
type ValidatedGenerationBuilder struct {
	builder *GenerationBuilder
	errors  []error
}

// NewValidatedGenerationBuilder creates a new validated generation builder.
func NewValidatedGenerationBuilder(trace *TraceContext) *ValidatedGenerationBuilder {
	return &ValidatedGenerationBuilder{
		builder: trace.NewGeneration(),
		errors:  make([]error, 0),
	}
}

// ID sets the generation ID with validation.
func (b *ValidatedGenerationBuilder) ID(id string) *ValidatedGenerationBuilder {
	if id == "" {
		b.errors = append(b.errors, NewValidationError("id", "cannot be empty"))
	} else {
		b.builder.ID(id)
	}
	return b
}

// Name sets the generation name with validation.
func (b *ValidatedGenerationBuilder) Name(name string) *ValidatedGenerationBuilder {
	if err := ValidateName("name", name, MaxNameLength); err != nil {
		b.errors = append(b.errors, err)
	} else {
		b.builder.Name(name)
	}
	return b
}

// Model sets the generation model.
func (b *ValidatedGenerationBuilder) Model(model string) *ValidatedGenerationBuilder {
	b.builder.Model(model)
	return b
}

// ModelParameters sets the model parameters.
func (b *ValidatedGenerationBuilder) ModelParameters(params map[string]any) *ValidatedGenerationBuilder {
	b.builder.ModelParameters(params)
	return b
}

// Input sets the generation input.
func (b *ValidatedGenerationBuilder) Input(input any) *ValidatedGenerationBuilder {
	b.builder.Input(input)
	return b
}

// Output sets the generation output.
func (b *ValidatedGenerationBuilder) Output(output any) *ValidatedGenerationBuilder {
	b.builder.Output(output)
	return b
}

// Metadata sets the generation metadata with validation.
func (b *ValidatedGenerationBuilder) Metadata(metadata map[string]any) *ValidatedGenerationBuilder {
	if err := ValidateMetadata("metadata", metadata); err != nil {
		b.errors = append(b.errors, err)
	} else {
		b.builder.Metadata(metadata)
	}
	return b
}

// Level sets the generation level with validation.
func (b *ValidatedGenerationBuilder) Level(level ObservationLevel) *ValidatedGenerationBuilder {
	if err := ValidateLevel("level", level); err != nil {
		b.errors = append(b.errors, err)
	} else {
		b.builder.Level(level)
	}
	return b
}

// Usage sets the usage statistics with validation.
func (b *ValidatedGenerationBuilder) Usage(input, output int) *ValidatedGenerationBuilder {
	if input < 0 {
		b.errors = append(b.errors, NewValidationError("input", "must be non-negative"))
	}
	if output < 0 {
		b.errors = append(b.errors, NewValidationError("output", "must be non-negative"))
	}
	if input >= 0 && output >= 0 {
		b.builder.UsageTokens(input, output)
	}
	return b
}

// UsageDetails sets detailed usage statistics with validation.
func (b *ValidatedGenerationBuilder) UsageDetails(usage *Usage) *ValidatedGenerationBuilder {
	if usage != nil {
		if usage.Input < 0 {
			b.errors = append(b.errors, NewValidationError("usage.input", "must be non-negative"))
		}
		if usage.Output < 0 {
			b.errors = append(b.errors, NewValidationError("usage.output", "must be non-negative"))
		}
		if usage.Total < 0 {
			b.errors = append(b.errors, NewValidationError("usage.total", "must be non-negative"))
		}
		if usage.Input >= 0 && usage.Output >= 0 && usage.Total >= 0 {
			b.builder.Usage(usage)
		}
	}
	return b
}

// HasErrors returns true if any validation errors have been accumulated.
func (b *ValidatedGenerationBuilder) HasErrors() bool {
	return len(b.errors) > 0
}

// Errors returns all accumulated validation errors.
func (b *ValidatedGenerationBuilder) Errors() []error {
	return b.errors
}

// Create creates the generation, returning a BuildResult that must be unwrapped.
func (b *ValidatedGenerationBuilder) Create(ctx context.Context) BuildResult[*GenerationContext] {
	if len(b.errors) > 0 {
		return BuildResultError[*GenerationContext](combineValidationErrors(b.errors))
	}
	gen, err := b.builder.Create(ctx)
	return NewBuildResult(gen, err)
}

// ValidatedScoreBuilder wraps ScoreBuilder with compile-time validation enforcement.
type ValidatedScoreBuilder struct {
	builder *ScoreBuilder
	errors  []error
}

// NewValidatedScoreBuilder creates a new validated score builder.
func NewValidatedScoreBuilder(trace *TraceContext) *ValidatedScoreBuilder {
	return &ValidatedScoreBuilder{
		builder: trace.NewScore(),
		errors:  make([]error, 0),
	}
}

// Name sets the score name with validation.
func (b *ValidatedScoreBuilder) Name(name string) *ValidatedScoreBuilder {
	if name == "" {
		b.errors = append(b.errors, NewValidationError("name", "is required"))
	} else if err := ValidateName("name", name, MaxNameLength); err != nil {
		b.errors = append(b.errors, err)
	} else {
		b.builder.Name(name)
	}
	return b
}

// Value sets the score value with validation (for numeric scores between 0 and 1).
func (b *ValidatedScoreBuilder) Value(value float64) *ValidatedScoreBuilder {
	if err := ValidateScoreValue("value", value, 0, 1); err != nil {
		b.errors = append(b.errors, err)
	} else {
		b.builder.NumericValue(value)
	}
	return b
}

// NumericValue sets a numeric score value with validation.
func (b *ValidatedScoreBuilder) NumericValue(value float64) *ValidatedScoreBuilder {
	b.builder.NumericValue(value)
	return b
}

// CategoricalValue sets a categorical score value with validation.
func (b *ValidatedScoreBuilder) CategoricalValue(value string) *ValidatedScoreBuilder {
	if value == "" {
		b.errors = append(b.errors, NewValidationError("value", "categorical value cannot be empty"))
	} else {
		b.builder.CategoricalValue(value)
	}
	return b
}

// BooleanValue sets a boolean score value.
func (b *ValidatedScoreBuilder) BooleanValue(value bool) *ValidatedScoreBuilder {
	b.builder.BooleanValue(value)
	return b
}

// Comment sets the score comment.
func (b *ValidatedScoreBuilder) Comment(comment string) *ValidatedScoreBuilder {
	b.builder.Comment(comment)
	return b
}

// ObservationID sets the observation ID.
func (b *ValidatedScoreBuilder) ObservationID(observationID string) *ValidatedScoreBuilder {
	b.builder.ObservationID(observationID)
	return b
}

// ConfigID sets the config ID.
func (b *ValidatedScoreBuilder) ConfigID(configID string) *ValidatedScoreBuilder {
	b.builder.ConfigID(configID)
	return b
}

// HasErrors returns true if any validation errors have been accumulated.
func (b *ValidatedScoreBuilder) HasErrors() bool {
	return len(b.errors) > 0
}

// Errors returns all accumulated validation errors.
func (b *ValidatedScoreBuilder) Errors() []error {
	return b.errors
}

// Create creates the score.
// Unlike other builders, Score creation returns an error directly since
// scores don't have a context object.
func (b *ValidatedScoreBuilder) Create(ctx context.Context) error {
	if len(b.errors) > 0 {
		return combineValidationErrors(b.errors)
	}
	return b.builder.Create(ctx)
}

// combineValidationErrors combines multiple validation errors into a single error.
func combineValidationErrors(errs []error) error {
	if len(errs) == 0 {
		return nil
	}
	if len(errs) == 1 {
		return errs[0]
	}

	var messages []string
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	return errors.New("langfuse: multiple validation errors: " + strings.Join(messages, "; "))
}

// StrictValidationConfig holds configuration for strict validation mode.
type StrictValidationConfig struct {
	// Enabled determines whether strict validation is active.
	Enabled bool

	// FailFast stops on first validation error if true.
	// If false, all errors are accumulated and returned together.
	FailFast bool
}

// DefaultStrictValidationConfig returns the default strict validation configuration.
func DefaultStrictValidationConfig() StrictValidationConfig {
	return StrictValidationConfig{
		Enabled:  true,
		FailFast: false,
	}
}

// ============================================================================
// Trace Builders (from trace.go)
// ============================================================================

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

// ============================================================================
// Span Builders (from span.go)
// ============================================================================

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

// ============================================================================
// Generation Builders (from generation.go)
// ============================================================================

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
