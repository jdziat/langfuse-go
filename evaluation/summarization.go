package evaluation

import (
	"context"
	"fmt"

	langfuse "github.com/jdziat/langfuse-go"
)

// SummarizationTraceBuilder provides a fluent interface for creating summarization-ready traces.
type SummarizationTraceBuilder struct {
	*langfuse.TraceBuilder
	sumInput *SummarizationInput
}

// NewSummarizationTrace creates a new summarization trace builder.
func NewSummarizationTrace(client *langfuse.Client, name string) *SummarizationTraceBuilder {
	return &SummarizationTraceBuilder{
		TraceBuilder: client.NewTrace().Name(name),
		sumInput:     &SummarizationInput{},
	}
}

// Input sets the text to summarize.
func (b *SummarizationTraceBuilder) Input(text string) *SummarizationTraceBuilder {
	b.sumInput.Input = text
	return b
}

// GroundTruth sets the expected summary for evaluation.
func (b *SummarizationTraceBuilder) GroundTruth(truth string) *SummarizationTraceBuilder {
	b.sumInput.GroundTruth = truth
	return b
}

// MaxLength sets the target summary length in words.
func (b *SummarizationTraceBuilder) MaxLength(length int) *SummarizationTraceBuilder {
	b.sumInput.MaxLength = length
	return b
}

// Style sets the summary style (e.g., "bullet_points", "paragraph").
func (b *SummarizationTraceBuilder) Style(style string) *SummarizationTraceBuilder {
	b.sumInput.Style = style
	return b
}

// ID sets the trace ID.
func (b *SummarizationTraceBuilder) ID(id string) *SummarizationTraceBuilder {
	b.TraceBuilder.ID(id)
	return b
}

// UserID sets the user ID.
func (b *SummarizationTraceBuilder) UserID(userID string) *SummarizationTraceBuilder {
	b.TraceBuilder.UserID(userID)
	return b
}

// SessionID sets the session ID.
func (b *SummarizationTraceBuilder) SessionID(sessionID string) *SummarizationTraceBuilder {
	b.TraceBuilder.SessionID(sessionID)
	return b
}

// Tags sets the trace tags.
func (b *SummarizationTraceBuilder) Tags(tags []string) *SummarizationTraceBuilder {
	b.TraceBuilder.Tags(tags)
	return b
}

// Metadata sets the trace metadata.
func (b *SummarizationTraceBuilder) Metadata(metadata map[string]any) *SummarizationTraceBuilder {
	b.TraceBuilder.Metadata(metadata)
	return b
}

// Release sets the release version.
func (b *SummarizationTraceBuilder) Release(release string) *SummarizationTraceBuilder {
	b.TraceBuilder.Release(release)
	return b
}

// Version sets the version.
func (b *SummarizationTraceBuilder) Version(version string) *SummarizationTraceBuilder {
	b.TraceBuilder.Version(version)
	return b
}

// Environment sets the environment.
func (b *SummarizationTraceBuilder) Environment(env string) *SummarizationTraceBuilder {
	b.TraceBuilder.Environment(env)
	return b
}

// Public sets whether the trace is public.
func (b *SummarizationTraceBuilder) Public(public bool) *SummarizationTraceBuilder {
	b.TraceBuilder.Public(public)
	return b
}

// Validate validates the summarization trace configuration.
func (b *SummarizationTraceBuilder) Validate() error {
	if b.sumInput.Input == "" {
		return fmt.Errorf("input text is required for summarization traces")
	}
	return b.TraceBuilder.Validate()
}

// Create creates the summarization trace and returns a context for updating it.
func (b *SummarizationTraceBuilder) Create(ctx context.Context) (*SummarizationTraceContext, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}

	b.TraceBuilder.Input(b.sumInput)

	traceCtx, err := b.TraceBuilder.Create(ctx)
	if err != nil {
		return nil, err
	}

	return &SummarizationTraceContext{
		TraceContext: traceCtx,
		input:        b.sumInput,
	}, nil
}

// SummarizationTraceContext provides context for a summarization trace with typed methods.
type SummarizationTraceContext struct {
	*langfuse.TraceContext
	input  *SummarizationInput
	output *SummarizationOutput
}

// GetInput returns the summarization input.
func (s *SummarizationTraceContext) GetInput() *SummarizationInput {
	return s.input
}

// GetOutput returns the summarization output.
func (s *SummarizationTraceContext) GetOutput() *SummarizationOutput {
	return s.output
}

// UpdateOutput updates the trace with summarization output.
func (s *SummarizationTraceContext) UpdateOutput(ctx context.Context, summary string) error {
	s.output = &SummarizationOutput{
		Output: summary,
	}
	return s.Update().Output(s.output).Apply(ctx)
}

// UpdateOutputWithMetadata updates the trace with a full summarization output struct.
func (s *SummarizationTraceContext) UpdateOutputWithMetadata(ctx context.Context, output *SummarizationOutput) error {
	s.output = output
	return s.Update().Output(output).Apply(ctx)
}

// ValidateForEvaluation checks if the trace has all required fields for evaluation.
func (s *SummarizationTraceContext) ValidateForEvaluation() error {
	if s.output == nil {
		return fmt.Errorf("output is required before evaluation")
	}
	return ValidateFor(s.input, s.output, SummarizationEvaluator)
}
