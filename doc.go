// Package langfuse provides a Go SDK for the Langfuse LLM observability platform.
//
// Langfuse enables you to trace, monitor, and analyze LLM applications. This SDK
// provides a fluent builder pattern for creating traces, spans, generations, and events.
//
// # Quick Start
//
// Create a client and start tracing:
//
//	client, err := langfuse.New(
//	    os.Getenv("LANGFUSE_PUBLIC_KEY"),
//	    os.Getenv("LANGFUSE_SECRET_KEY"),
//	)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Shutdown(context.Background())
//
//	// Create a trace
//	trace, err := client.NewTrace().
//	    Name("my-llm-call").
//	    UserID("user-123").
//	    Create()
//
//	// Record an LLM generation
//	gen, err := trace.Generation().
//	    Name("openai-completion").
//	    Model("gpt-4").
//	    Input("What is Go?").
//	    Create()
//
//	// ... make your LLM call ...
//
//	gen.End().
//	    Output("Go is a programming language...").
//	    Usage(1500, 100).
//	    Apply()
//
// # Configuration
//
// The client can be configured with various options:
//
//	client, err := langfuse.New(publicKey, secretKey,
//	    langfuse.WithRegion(langfuse.RegionUS),
//	    langfuse.WithBatchSize(100),
//	    langfuse.WithFlushInterval(5 * time.Second),
//	    langfuse.WithDebug(true),
//	)
//
// # Thread Safety
//
// The Client is safe for concurrent use. TraceContext and observation contexts
// (SpanContext, GenerationContext) are also safe for concurrent use within a
// single trace. However, individual builder instances (TraceBuilder, SpanBuilder,
// etc.) should only be used from a single goroutine.
//
// # Event Delivery Guarantees
//
// The SDK provides best-effort delivery of events to the Langfuse API. Events are
// queued locally and sent in batches to minimize network overhead and improve
// throughput. Understanding the delivery semantics helps you design your
// application appropriately.
//
// # Delivery Semantics
//
// Events are delivered asynchronously with best-effort guarantees:
//
//   - Events are queued in an internal buffer and sent in batches
//   - Failed batches are retried with exponential backoff (up to 3 attempts by default)
//   - Events may be lost if: the process crashes before flushing, the queue fills up,
//     or all retry attempts are exhausted
//   - No duplicate detection is performed; events may be delivered more than once
//     if a network error occurs after the server accepts the batch but before
//     the client receives confirmation
//
// # Graceful Shutdown
//
// Always call [Client.Shutdown] before your application exits to ensure queued
// events are flushed:
//
//	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
//	defer cancel()
//
//	if err := client.Shutdown(ctx); err != nil {
//	    log.Printf("shutdown warning: %v", err)
//	}
//
// The shutdown process follows this sequence:
//
//  1. Stop accepting new events (methods return ErrClientClosed)
//  2. Stop the automatic flush timer
//  3. Signal the batch processor to drain all queued events
//  4. Wait for drain to complete (respects context timeout)
//  5. Cancel internal context to stop remaining goroutines
//  6. Wait for all goroutines to exit
//
// If the context times out during shutdown, remaining events may be lost. Configure
// [WithShutdownTimeout] appropriately for your workload. For critical telemetry,
// consider calling [Client.Flush] periodically to reduce the window of potential
// data loss.
//
// # Queue Behavior
//
// The internal queue has a fixed capacity (configurable via [WithQueueSize]).
// When the queue is full, new events may be dropped. Monitor for dropped events
// by checking the error returned from trace/span/generation creation methods.
//
// For high-throughput applications:
//
//   - Increase queue size with [WithQueueSize]
//   - Increase batch size with [WithBatchSize]
//   - Consider calling [Client.Flush] at natural breakpoints (e.g., request completion)
//
// # Subpackages
//
// The SDK provides additional functionality through subpackages:
//
//   - [github.com/jdziat/langfuse-go/evaluation]: Evaluation-ready tracing with
//     typed inputs/outputs for RAG, Q&A, summarization, and classification workflows.
//
//   - [github.com/jdziat/langfuse-go/langfusetest]: Test utilities including mock
//     servers and test clients for unit testing code that uses Langfuse.
//
//   - [github.com/jdziat/langfuse-go/otel]: OpenTelemetry bridge for integrating
//     Langfuse with existing OpenTelemetry instrumentation.
//
// # Examples
//
// See the examples directory for complete working examples:
//   - examples/basic: Simple trace and generation
//   - examples/advanced: Complex traces with spans and nested generations
//   - examples/evaluation: RAG and Q&A evaluation workflows
package langfuse

// Version is the current SDK version.
// This is used in User-Agent headers and for debugging.
const Version = "1.0.0"
