// Package langfuse provides a Go SDK for the Langfuse LLM observability platform.
//
// Langfuse enables you to trace, monitor, and analyze LLM applications. This SDK
// provides a fluent builder pattern for creating traces, spans, generations, and events.
//
// # Start Here
//
// The SDK offers several ways to create observations, but new code should use
// the canonical, context-threaded fluent builders. These five symbols cover the
// common path:
//
//   - [New] — construct a [Client] from your public and secret keys.
//   - [Client.NewTrace] — start a trace; chain setters then Create(ctx).
//   - [TraceContext.NewSpan] — add a span (a unit of work) to a trace.
//   - [TraceContext.NewGeneration] — record an LLM generation on a trace.
//   - [GenerationContext.EndWithUsage] — finish a generation with output and
//     token usage.
//
// A complete, canonical example:
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
//	ctx := context.Background()
//
//	trace, err := client.NewTrace().
//	    Name("my-llm-call").
//	    UserID("user-123").
//	    Create(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	gen, err := trace.NewGeneration().
//	    Name("openai-completion").
//	    Model("gpt-4").
//	    Input("What is Go?").
//	    Create(ctx)
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	// ... make your LLM call ...
//
//	if err := gen.EndWithUsage(ctx, "Go is a programming language...", 1500, 100); err != nil {
//	    log.Printf("failed to end generation: %v", err)
//	}
//
// Other surfaces (the Simple API methods such as [Client.Trace], the V1-suffixed
// aliases, and the Validated* builders) remain supported but are documented as
// secondary; prefer the symbols above.
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
// # Accessing Configuration
//
// To read back the configuration after construction, use [Client.RootConfig],
// which returns the full root *[Config] (with defaults applied), including
// root-only fields such as EvaluationConfig and StrictValidation.
//
// The [Client] embeds the internal *pkgclient.Client, which promotes a Config
// method onto [Client]. That promoted method returns the internal,
// field-lossy pkgclient.Config and omits root-only fields, so prefer
// [Client.RootConfig] when you need the complete configuration:
//
//	cfg := client.RootConfig()
//	if cfg.EvaluationConfig != nil {
//	    // root-only fields are available via RootConfig
//	}
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

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// Version is the current SDK version.
// This is used in User-Agent headers and for debugging.
//
// The value is sourced from the VERSION file at the module root (embedded at
// build time) so that the version is defined in a single place and stays in
// sync with releases.
var Version = strings.TrimSpace(versionFile)
