package evaluation

import (
	"context"
	"fmt"

	langfuse "github.com/jdziat/langfuse-go"
)

// RAGTraceBuilder provides a fluent interface for creating RAG-ready traces.
type RAGTraceBuilder struct {
	*langfuse.TraceBuilder
	ragInput *RAGInput
}

// NewRAGTrace creates a new RAG trace builder.
func NewRAGTrace(client *langfuse.Client, name string) *RAGTraceBuilder {
	return &RAGTraceBuilder{
		TraceBuilder: client.NewTrace().Name(name),
		ragInput:     &RAGInput{},
	}
}

// Query sets the query for the RAG trace.
func (b *RAGTraceBuilder) Query(query string) *RAGTraceBuilder {
	b.ragInput.Query = query
	return b
}

// Context sets the retrieved context chunks.
func (b *RAGTraceBuilder) Context(chunks ...string) *RAGTraceBuilder {
	b.ragInput.Context = chunks
	return b
}

// GroundTruth sets the expected answer for evaluation.
func (b *RAGTraceBuilder) GroundTruth(truth string) *RAGTraceBuilder {
	b.ragInput.GroundTruth = truth
	return b
}

// AdditionalContext sets additional context metadata.
func (b *RAGTraceBuilder) AdditionalContext(ctx map[string]any) *RAGTraceBuilder {
	b.ragInput.AdditionalContext = ctx
	return b
}

// ID sets the trace ID.
func (b *RAGTraceBuilder) ID(id string) *RAGTraceBuilder {
	b.TraceBuilder.ID(id)
	return b
}

// UserID sets the user ID.
func (b *RAGTraceBuilder) UserID(userID string) *RAGTraceBuilder {
	b.TraceBuilder.UserID(userID)
	return b
}

// SessionID sets the session ID.
func (b *RAGTraceBuilder) SessionID(sessionID string) *RAGTraceBuilder {
	b.TraceBuilder.SessionID(sessionID)
	return b
}

// Tags sets the trace tags.
func (b *RAGTraceBuilder) Tags(tags []string) *RAGTraceBuilder {
	b.TraceBuilder.Tags(tags)
	return b
}

// Metadata sets the trace metadata.
func (b *RAGTraceBuilder) Metadata(metadata map[string]any) *RAGTraceBuilder {
	b.TraceBuilder.Metadata(metadata)
	return b
}

// Release sets the release version.
func (b *RAGTraceBuilder) Release(release string) *RAGTraceBuilder {
	b.TraceBuilder.Release(release)
	return b
}

// Version sets the version.
func (b *RAGTraceBuilder) Version(version string) *RAGTraceBuilder {
	b.TraceBuilder.Version(version)
	return b
}

// Environment sets the environment.
func (b *RAGTraceBuilder) Environment(env string) *RAGTraceBuilder {
	b.TraceBuilder.Environment(env)
	return b
}

// Public sets whether the trace is public.
func (b *RAGTraceBuilder) Public(public bool) *RAGTraceBuilder {
	b.TraceBuilder.Public(public)
	return b
}

// Validate validates the RAG trace configuration.
func (b *RAGTraceBuilder) Validate() error {
	if b.ragInput.Query == "" {
		return fmt.Errorf("query is required for RAG traces")
	}
	if len(b.ragInput.Context) == 0 {
		return fmt.Errorf("at least one context chunk is required for RAG traces")
	}
	return b.TraceBuilder.Validate()
}

// Create creates the RAG trace and returns a context for updating it.
func (b *RAGTraceBuilder) Create(ctx context.Context) (*RAGTraceContext, error) {
	if err := b.Validate(); err != nil {
		return nil, err
	}

	// Set the structured input
	b.TraceBuilder.Input(b.ragInput)

	traceCtx, err := b.TraceBuilder.Create(ctx)
	if err != nil {
		return nil, err
	}

	return &RAGTraceContext{
		TraceContext: traceCtx,
		input:        b.ragInput,
	}, nil
}

// RAGTraceContext provides context for a RAG trace with typed methods.
type RAGTraceContext struct {
	*langfuse.TraceContext
	input  *RAGInput
	output *RAGOutput
}

// GetInput returns the RAG input.
func (r *RAGTraceContext) GetInput() *RAGInput {
	return r.input
}

// GetOutput returns the RAG output.
func (r *RAGTraceContext) GetOutput() *RAGOutput {
	return r.output
}

// UpdateOutput updates the trace with RAG output.
func (r *RAGTraceContext) UpdateOutput(ctx context.Context, answer string, citations ...string) error {
	r.output = &RAGOutput{
		Output:    answer,
		Citations: citations,
	}
	return r.Update().Output(r.output).Apply(ctx)
}

// UpdateOutputWithMetadata updates the trace with a full RAG output struct.
func (r *RAGTraceContext) UpdateOutputWithMetadata(ctx context.Context, output *RAGOutput) error {
	r.output = output
	return r.Update().Output(output).Apply(ctx)
}

// ValidateForEvaluation checks if the trace has all required fields for evaluation.
func (r *RAGTraceContext) ValidateForEvaluation() error {
	if r.output == nil {
		return fmt.Errorf("output is required before evaluation")
	}
	return ValidateFor(r.input, r.output, RAGEvaluator)
}
