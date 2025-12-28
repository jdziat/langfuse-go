package langfuse

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

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
