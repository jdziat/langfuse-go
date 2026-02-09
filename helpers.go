package langfuse

import (
	"context"
	"time"
)

// ============================================================================
// Metadata Type
// ============================================================================

// Metadata provides type-safe metadata storage with JSON serialization.
// This replaces raw map[string]any for better type clarity.
type Metadata map[string]any

// NewMetadata creates a new empty Metadata instance.
func NewMetadata() Metadata {
	return make(Metadata)
}

// Set sets a key-value pair in the metadata.
// Returns the Metadata for method chaining.
func (m Metadata) Set(key string, value any) Metadata {
	m[key] = value
	return m
}

// Get retrieves a value from the metadata.
// Returns the value and true if found, nil and false otherwise.
func (m Metadata) Get(key string) (any, bool) {
	v, ok := m[key]
	return v, ok
}

// GetString retrieves a string value from the metadata.
// Returns the string and true if found and is a string, empty string and false otherwise.
func (m Metadata) GetString(key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// GetInt retrieves an int value from the metadata.
// Returns the int and true if found and is an int, 0 and false otherwise.
// Also handles float64 values (common from JSON unmarshaling).
func (m Metadata) GetInt(key string) (int, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

// GetFloat retrieves a float64 value from the metadata.
// Returns the float64 and true if found and is numeric, 0 and false otherwise.
func (m Metadata) GetFloat(key string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// GetBool retrieves a bool value from the metadata.
// Returns the bool and true if found and is a bool, false and false otherwise.
func (m Metadata) GetBool(key string) (bool, bool) {
	v, ok := m[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// Has returns true if the key exists in the metadata.
func (m Metadata) Has(key string) bool {
	_, ok := m[key]
	return ok
}

// Delete removes a key from the metadata.
// Returns the Metadata for method chaining.
func (m Metadata) Delete(key string) Metadata {
	delete(m, key)
	return m
}

// Merge merges another Metadata into this one.
// Values from other will overwrite values in m for duplicate keys.
// Returns the Metadata for method chaining.
func (m Metadata) Merge(other Metadata) Metadata {
	for k, v := range other {
		m[k] = v
	}
	return m
}

// Clone creates a shallow copy of the metadata.
func (m Metadata) Clone() Metadata {
	clone := make(Metadata, len(m))
	for k, v := range m {
		clone[k] = v
	}
	return clone
}

// Keys returns all keys in the metadata.
func (m Metadata) Keys() []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Len returns the number of entries in the metadata.
func (m Metadata) Len() int {
	return len(m)
}

// IsEmpty returns true if the metadata has no entries.
func (m Metadata) IsEmpty() bool {
	return len(m) == 0
}

// Filter returns a new Metadata containing only the specified keys.
// Keys that don't exist in the source metadata are ignored.
func (m Metadata) Filter(keys ...string) Metadata {
	result := make(Metadata, len(keys))
	for _, k := range keys {
		if v, ok := m[k]; ok {
			result[k] = v
		}
	}
	return result
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
