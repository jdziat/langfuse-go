package langfuse

import "context"

// Tracer defines the core tracing interface.
// This interface is implemented by *Client and can be used for
// dependency injection and testing.
//
// Example:
//
//	func ProcessRequest(ctx context.Context, tracer langfuse.Tracer) error {
//	    trace, err := tracer.NewTrace().Name("process-request").Create(ctx)
//	    if err != nil {
//	        return err
//	    }
//	    // ... do work ...
//	    return nil
//	}
type Tracer interface {
	// NewTrace creates a new trace builder.
	NewTrace() *TraceBuilder

	// Flush sends all pending events to Langfuse.
	// It blocks until all events are sent or the context is cancelled.
	Flush(ctx context.Context) error

	// Shutdown gracefully shuts down the client, flushing any pending events.
	// After Shutdown returns, the client should not be used.
	Shutdown(ctx context.Context) error
}

// Ensure Client implements Tracer at compile time.
var _ Tracer = (*Client)(nil)

// Observer defines the interface for creating observations within a trace.
// This interface is implemented by TraceContext, SpanContext, and GenerationContext.
//
// The Advanced API uses builder methods (NewSpan, NewGeneration, etc.) for full control.
// The Simple API uses convenience methods (Span, Generation, etc.) for common use cases.
type Observer interface {
	// ID returns the ID of this trace or observation.
	ID() string

	// TraceID returns the trace ID that this observation belongs to.
	TraceID() string

	// NewSpan creates a new span builder as a child of this context (Advanced API).
	NewSpan() *SpanBuilder

	// NewGeneration creates a new generation builder as a child of this context (Advanced API).
	NewGeneration() *GenerationBuilder

	// NewEvent creates a new event builder as a child of this context (Advanced API).
	NewEvent() *EventBuilder

	// NewScore creates a new score builder for this context (Advanced API).
	NewScore() *ScoreBuilder
}

// Ensure contexts implement Observer at compile time.
var (
	_ Observer = (*TraceContext)(nil)
	_ Observer = (*SpanContext)(nil)
	_ Observer = (*GenerationContext)(nil)
)

// Flusher defines the interface for types that can flush pending data.
type Flusher interface {
	Flush(ctx context.Context) error
}

// Ensure Client implements Flusher at compile time.
var _ Flusher = (*Client)(nil)

// HealthChecker defines the interface for health checking.
type HealthChecker interface {
	Health(ctx context.Context) (*HealthStatus, error)
}

// Ensure Client implements HealthChecker at compile time.
var _ HealthChecker = (*Client)(nil)
