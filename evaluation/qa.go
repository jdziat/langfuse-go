package evaluation

import (
	"context"
	"fmt"

	langfuse "github.com/jdziat/langfuse-go"
)

// QATraceBuilder provides a fluent interface for creating Q&A-ready traces.
type QATraceBuilder struct {
	*langfuse.TraceBuilder
	qaInput *QAInput
}

// NewQATrace creates a new Q&A trace builder.
func NewQATrace(client *langfuse.Client, name string) *QATraceBuilder {
	return &QATraceBuilder{
		TraceBuilder: client.NewTrace().Name(name),
		qaInput:      &QAInput{},
	}
}

// Query sets the question for the Q&A trace.
func (b *QATraceBuilder) Query(query string) *QATraceBuilder {
	b.qaInput.Query = query
	return b
}

// GroundTruth sets the expected answer for evaluation.
func (b *QATraceBuilder) GroundTruth(truth string) *QATraceBuilder {
	b.qaInput.GroundTruth = truth
	return b
}

// Context sets additional context for the question.
func (b *QATraceBuilder) Context(ctx string) *QATraceBuilder {
	b.qaInput.Context = ctx
	return b
}

// ID sets the trace ID.
func (b *QATraceBuilder) ID(id string) *QATraceBuilder {
	b.TraceBuilder.ID(id)
	return b
}

// UserID sets the user ID.
func (b *QATraceBuilder) UserID(userID string) *QATraceBuilder {
	b.TraceBuilder.UserID(userID)
	return b
}

// SessionID sets the session ID.
func (b *QATraceBuilder) SessionID(sessionID string) *QATraceBuilder {
	b.TraceBuilder.SessionID(sessionID)
	return b
}

// Tags sets the trace tags.
func (b *QATraceBuilder) Tags(tags []string) *QATraceBuilder {
	b.TraceBuilder.Tags(tags)
	return b
}

// Metadata sets the trace metadata.
func (b *QATraceBuilder) Metadata(metadata map[string]any) *QATraceBuilder {
	b.TraceBuilder.Metadata(metadata)
	return b
}

// Release sets the release version.
func (b *QATraceBuilder) Release(release string) *QATraceBuilder {
	b.TraceBuilder.Release(release)
	return b
}

// Version sets the version.
func (b *QATraceBuilder) Version(version string) *QATraceBuilder {
	b.TraceBuilder.Version(version)
	return b
}

// Environment sets the environment.
func (b *QATraceBuilder) Environment(env string) *QATraceBuilder {
	b.TraceBuilder.Environment(env)
	return b
}

// Public sets whether the trace is public.
func (b *QATraceBuilder) Public(public bool) *QATraceBuilder {
	b.TraceBuilder.Public(public)
	return b
}

// Validate validates the Q&A trace configuration.
func (b *QATraceBuilder) Validate() error {
	if b.qaInput.Query == "" {
		return fmt.Errorf("query is required for Q&A traces")
	}
	return b.TraceBuilder.Validate()
}

// Create creates the Q&A trace and returns a context for updating it.
func (b *QATraceBuilder) Create(ctx context.Context) (*QATraceContext, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}

	b.TraceBuilder.Input(b.qaInput)

	traceCtx, err := b.TraceBuilder.Create(ctx)
	if err != nil {
		return nil, err
	}

	return &QATraceContext{
		TraceContext: traceCtx,
		input:        b.qaInput,
	}, nil
}

// QATraceContext provides context for a Q&A trace with typed methods.
type QATraceContext struct {
	*langfuse.TraceContext
	input  *QAInput
	output *QAOutput
}

// GetInput returns the Q&A input.
func (q *QATraceContext) GetInput() *QAInput {
	return q.input
}

// GetOutput returns the Q&A output.
func (q *QATraceContext) GetOutput() *QAOutput {
	return q.output
}

// UpdateOutput updates the trace with Q&A output.
func (q *QATraceContext) UpdateOutput(ctx context.Context, answer string, confidence float64) error {
	q.output = &QAOutput{
		Output:     answer,
		Confidence: confidence,
	}
	return q.Update().Output(q.output).Apply(ctx)
}

// UpdateOutputWithMetadata updates the trace with a full Q&A output struct.
func (q *QATraceContext) UpdateOutputWithMetadata(ctx context.Context, output *QAOutput) error {
	q.output = output
	return q.Update().Output(output).Apply(ctx)
}

// ValidateForEvaluation checks if the trace has all required fields for evaluation.
func (q *QATraceContext) ValidateForEvaluation() error {
	if q.output == nil {
		return fmt.Errorf("output is required before evaluation")
	}
	return ValidateFor(q.input, q.output, QAEvaluator)
}
