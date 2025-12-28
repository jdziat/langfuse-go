//go:build integration

// Package langfuse_test contains integration tests for the Langfuse SDK.
//
// These tests require a running Langfuse instance and valid credentials.
// Set the following environment variables:
//   - LANGFUSE_PUBLIC_KEY: Your Langfuse public key
//   - LANGFUSE_SECRET_KEY: Your Langfuse secret key
//   - LANGFUSE_BASE_URL: (Optional) Base URL for self-hosted instances
//
// Run integration tests with:
//
//	go test -tags=integration -v ./...
package langfuse_test

import (
	"context"
	"os"
	"testing"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
)

func getTestCredentials(t *testing.T) (publicKey, secretKey string) {
	t.Helper()

	publicKey = os.Getenv("LANGFUSE_PUBLIC_KEY")
	secretKey = os.Getenv("LANGFUSE_SECRET_KEY")

	if publicKey == "" || secretKey == "" {
		t.Skip("LANGFUSE_PUBLIC_KEY and LANGFUSE_SECRET_KEY must be set for integration tests")
	}

	return publicKey, secretKey
}

func getTestClient(t *testing.T) *langfuse.Client {
	t.Helper()

	publicKey, secretKey := getTestCredentials(t)

	opts := []langfuse.ConfigOption{
		langfuse.WithDebug(true),
		langfuse.WithFlushInterval(1 * time.Second),
		langfuse.WithBatchSize(10),
	}

	// Use custom base URL if provided
	if baseURL := os.Getenv("LANGFUSE_BASE_URL"); baseURL != "" {
		opts = append(opts, langfuse.WithBaseURL(baseURL))
	}

	client, err := langfuse.New(publicKey, secretKey, opts...)
	if err != nil {
		t.Fatalf("failed to create Langfuse client: %v", err)
	}

	return client
}

func TestIntegration_HealthCheck(t *testing.T) {
	client := getTestClient(t)
	defer client.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	health, err := client.Health(ctx)
	if err != nil {
		t.Fatalf("Health() error = %v", err)
	}

	if health == nil {
		t.Fatal("Health() returned nil")
	}

	t.Logf("Health check passed: %+v", health)
}

func TestIntegration_CreateTrace(t *testing.T) {
	client := getTestClient(t)
	defer client.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	trace, err := client.NewTrace().
		Name("integration-test-trace").
		UserID("test-user").
		SessionID("test-session").
		Input(map[string]any{
			"message": "Hello from integration test",
		}).
		Tags([]string{"integration-test", "automated"}).
		Metadata(map[string]any{
			"test_run": time.Now().Unix(),
		}).
		Create(ctx)

	if err != nil {
		t.Fatalf("NewTrace().Create() error = %v", err)
	}

	if trace == nil {
		t.Fatal("trace is nil")
	}

	if trace.ID() == "" {
		t.Error("trace ID should not be empty")
	}

	t.Logf("Created trace: %s", trace.ID())

	// Flush to ensure events are sent
	if err := client.Flush(ctx); err != nil {
		t.Errorf("Flush() error = %v", err)
	}
}

func TestIntegration_CreateTraceWithSpan(t *testing.T) {
	client := getTestClient(t)
	defer client.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	trace, err := client.NewTrace().
		Name("integration-test-with-span").
		UserID("test-user").
		Create(ctx)

	if err != nil {
		t.Fatalf("NewTrace().Create() error = %v", err)
	}

	span, err := trace.NewSpan().
		Name("test-span").
		Input("span input").
		Create(ctx)

	if err != nil {
		t.Fatalf("Span().Create() error = %v", err)
	}

	if span == nil {
		t.Fatal("span is nil")
	}

	// End the span
	if err := span.EndWithOutput("span output"); err != nil {
		t.Errorf("EndWithOutput() error = %v", err)
	}

	t.Logf("Created span: %s", span.ObservationID())

	if err := client.Flush(ctx); err != nil {
		t.Errorf("Flush() error = %v", err)
	}
}

func TestIntegration_CreateTraceWithGeneration(t *testing.T) {
	client := getTestClient(t)
	defer client.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	trace, err := client.NewTrace().
		Name("integration-test-with-generation").
		UserID("test-user").
		Create(ctx)

	if err != nil {
		t.Fatalf("NewTrace().Create() error = %v", err)
	}

	gen, err := trace.NewGeneration().
		Name("test-generation").
		Model("gpt-4").
		ModelParameters(map[string]any{
			"temperature": 0.7,
		}).
		Input([]map[string]string{
			{"role": "user", "content": "Hello"},
		}).
		Create(ctx)

	if err != nil {
		t.Fatalf("Generation().Create() error = %v", err)
	}

	if gen == nil {
		t.Fatal("generation is nil")
	}

	// End with usage
	if err := gen.EndWithUsage("Hello! How can I help?", 10, 20); err != nil {
		t.Errorf("EndWithUsage() error = %v", err)
	}

	t.Logf("Created generation: %s", gen.ObservationID())

	if err := client.Flush(ctx); err != nil {
		t.Errorf("Flush() error = %v", err)
	}
}

func TestIntegration_CreateScore(t *testing.T) {
	client := getTestClient(t)
	defer client.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	trace, err := client.NewTrace().
		Name("integration-test-with-score").
		UserID("test-user").
		Create(ctx)

	if err != nil {
		t.Fatalf("NewTrace().Create() error = %v", err)
	}

	err = trace.NewScore().
		Name("quality").
		NumericValue(0.95).
		Comment("Great response").
		Create(ctx)

	if err != nil {
		t.Fatalf("Score().Create() error = %v", err)
	}

	t.Log("Created score for trace")

	if err := client.Flush(ctx); err != nil {
		t.Errorf("Flush() error = %v", err)
	}
}

func TestIntegration_CreateEvent(t *testing.T) {
	client := getTestClient(t)
	defer client.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	trace, err := client.NewTrace().
		Name("integration-test-with-event").
		UserID("test-user").
		Create(ctx)

	if err != nil {
		t.Fatalf("NewTrace().Create() error = %v", err)
	}

	err = trace.NewEvent().
		Name("test-event").
		Input(map[string]any{
			"action": "button_click",
		}).
		Create(ctx)

	if err != nil {
		t.Fatalf("Event().Create() error = %v", err)
	}

	t.Log("Created event for trace")

	if err := client.Flush(ctx); err != nil {
		t.Errorf("Flush() error = %v", err)
	}
}

func TestIntegration_BatchingPerformance(t *testing.T) {
	client := getTestClient(t)
	defer client.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	const numTraces = 50

	start := time.Now()

	for i := 0; i < numTraces; i++ {
		trace, err := client.NewTrace().
			Name("batch-test-trace").
			UserID("batch-user").
			Metadata(map[string]any{
				"index": i,
			}).
			Create(ctx)

		if err != nil {
			t.Fatalf("Failed to create trace %d: %v", i, err)
		}

		// Add a span to each trace
		_, err = trace.NewSpan().
			Name("batch-span").
			Create(ctx)

		if err != nil {
			t.Errorf("Failed to create span for trace %d: %v", i, err)
		}
	}

	creationDuration := time.Since(start)

	// Flush all events
	flushStart := time.Now()
	if err := client.Flush(ctx); err != nil {
		t.Errorf("Flush() error = %v", err)
	}
	flushDuration := time.Since(flushStart)

	t.Logf("Created %d traces with spans in %v", numTraces, creationDuration)
	t.Logf("Flushed all events in %v", flushDuration)
}

func TestIntegration_NestedSpans(t *testing.T) {
	client := getTestClient(t)
	defer client.Shutdown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	trace, err := client.NewTrace().
		Name("integration-test-nested-spans").
		UserID("test-user").
		Create(ctx)

	if err != nil {
		t.Fatalf("NewTrace().Create() error = %v", err)
	}

	// Create parent span
	parentSpan, err := trace.NewSpan().
		Name("parent-span").
		Create(ctx)

	if err != nil {
		t.Fatalf("Parent Span().Create() error = %v", err)
	}

	// Create child span
	childSpan, err := trace.NewSpan().
		Name("child-span").
		ParentObservationID(parentSpan.ObservationID()).
		Create(ctx)

	if err != nil {
		t.Fatalf("Child Span().Create() error = %v", err)
	}

	// Create grandchild span
	_, err = trace.NewSpan().
		Name("grandchild-span").
		ParentObservationID(childSpan.ObservationID()).
		Create(ctx)

	if err != nil {
		t.Fatalf("Grandchild Span().Create() error = %v", err)
	}

	// End spans
	childSpan.End()
	parentSpan.End()

	t.Logf("Created nested spans under trace: %s", trace.ID())

	if err := client.Flush(ctx); err != nil {
		t.Errorf("Flush() error = %v", err)
	}
}

func TestIntegration_GracefulShutdown(t *testing.T) {
	client := getTestClient(t)

	ctx := context.Background()

	// Create some events
	for i := 0; i < 10; i++ {
		_, err := client.NewTrace().
			Name("shutdown-test-trace").
			Create(ctx)

		if err != nil {
			t.Errorf("Failed to create trace: %v", err)
		}
	}

	// Shutdown should flush all pending events
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := client.Shutdown(shutdownCtx); err != nil {
		t.Errorf("Shutdown() error = %v", err)
	}

	t.Log("Graceful shutdown completed")
}
