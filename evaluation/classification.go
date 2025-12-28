package evaluation

import (
	"context"
	"fmt"

	langfuse "github.com/jdziat/langfuse-go"
)

// ClassificationTraceBuilder provides a fluent interface for creating classification-ready traces.
type ClassificationTraceBuilder struct {
	*langfuse.TraceBuilder
	classInput *ClassificationInput
}

// NewClassificationTrace creates a new classification trace builder.
func NewClassificationTrace(client *langfuse.Client, name string) *ClassificationTraceBuilder {
	return &ClassificationTraceBuilder{
		TraceBuilder: client.NewTrace().Name(name),
		classInput:   &ClassificationInput{},
	}
}

// Input sets the text to classify.
func (b *ClassificationTraceBuilder) Input(text string) *ClassificationTraceBuilder {
	b.classInput.Input = text
	return b
}

// Classes sets the possible classification categories.
func (b *ClassificationTraceBuilder) Classes(classes []string) *ClassificationTraceBuilder {
	b.classInput.Classes = classes
	return b
}

// GroundTruth sets the expected classification for evaluation.
func (b *ClassificationTraceBuilder) GroundTruth(truth string) *ClassificationTraceBuilder {
	b.classInput.GroundTruth = truth
	return b
}

// ID sets the trace ID.
func (b *ClassificationTraceBuilder) ID(id string) *ClassificationTraceBuilder {
	b.TraceBuilder.ID(id)
	return b
}

// UserID sets the user ID.
func (b *ClassificationTraceBuilder) UserID(userID string) *ClassificationTraceBuilder {
	b.TraceBuilder.UserID(userID)
	return b
}

// SessionID sets the session ID.
func (b *ClassificationTraceBuilder) SessionID(sessionID string) *ClassificationTraceBuilder {
	b.TraceBuilder.SessionID(sessionID)
	return b
}

// Tags sets the trace tags.
func (b *ClassificationTraceBuilder) Tags(tags []string) *ClassificationTraceBuilder {
	b.TraceBuilder.Tags(tags)
	return b
}

// Metadata sets the trace metadata.
func (b *ClassificationTraceBuilder) Metadata(metadata map[string]any) *ClassificationTraceBuilder {
	b.TraceBuilder.Metadata(metadata)
	return b
}

// Release sets the release version.
func (b *ClassificationTraceBuilder) Release(release string) *ClassificationTraceBuilder {
	b.TraceBuilder.Release(release)
	return b
}

// Version sets the version.
func (b *ClassificationTraceBuilder) Version(version string) *ClassificationTraceBuilder {
	b.TraceBuilder.Version(version)
	return b
}

// Environment sets the environment.
func (b *ClassificationTraceBuilder) Environment(env string) *ClassificationTraceBuilder {
	b.TraceBuilder.Environment(env)
	return b
}

// Public sets whether the trace is public.
func (b *ClassificationTraceBuilder) Public(public bool) *ClassificationTraceBuilder {
	b.TraceBuilder.Public(public)
	return b
}

// Validate validates the classification trace configuration.
func (b *ClassificationTraceBuilder) Validate() error {
	if b.classInput.Input == "" {
		return fmt.Errorf("input text is required for classification traces")
	}
	return b.TraceBuilder.Validate()
}

// Create creates the classification trace and returns a context for updating it.
func (b *ClassificationTraceBuilder) Create(ctx context.Context) (*ClassificationTraceContext, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}

	b.TraceBuilder.Input(b.classInput)

	traceCtx, err := b.TraceBuilder.Create(ctx)
	if err != nil {
		return nil, err
	}

	return &ClassificationTraceContext{
		TraceContext: traceCtx,
		input:        b.classInput,
	}, nil
}

// ClassificationTraceContext provides context for a classification trace with typed methods.
type ClassificationTraceContext struct {
	*langfuse.TraceContext
	input  *ClassificationInput
	output *ClassificationOutput
}

// GetInput returns the classification input.
func (c *ClassificationTraceContext) GetInput() *ClassificationInput {
	return c.input
}

// GetOutput returns the classification output.
func (c *ClassificationTraceContext) GetOutput() *ClassificationOutput {
	return c.output
}

// UpdateOutput updates the trace with classification output.
func (c *ClassificationTraceContext) UpdateOutput(ctx context.Context, predictedClass string, confidence float64) error {
	c.output = &ClassificationOutput{
		Output:     predictedClass,
		Confidence: confidence,
	}
	return c.Update().Output(c.output).Apply(ctx)
}

// UpdateOutputWithScores updates the trace with classification output including all class scores.
func (c *ClassificationTraceContext) UpdateOutputWithScores(ctx context.Context, predictedClass string, scores map[string]float64) error {
	c.output = &ClassificationOutput{
		Output: predictedClass,
		Scores: scores,
	}
	if conf, ok := scores[predictedClass]; ok {
		c.output.Confidence = conf
	}
	return c.Update().Output(c.output).Apply(ctx)
}

// UpdateOutputWithMetadata updates the trace with a full classification output struct.
func (c *ClassificationTraceContext) UpdateOutputWithMetadata(ctx context.Context, output *ClassificationOutput) error {
	c.output = output
	return c.Update().Output(output).Apply(ctx)
}

// ValidateForEvaluation checks if the trace has all required fields for evaluation.
func (c *ClassificationTraceContext) ValidateForEvaluation() error {
	if c.output == nil {
		return fmt.Errorf("output is required before evaluation")
	}
	return ValidateFor(c.input, c.output, ClassificationEvaluator)
}
