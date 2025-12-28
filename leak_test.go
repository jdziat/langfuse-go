package langfuse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/goleak"
)

// TestMain runs goleak verification for all tests in the package.
// This catches goroutine leaks that individual tests might miss.
func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m,
		// Ignore known background goroutines from test infrastructure
		goleak.IgnoreTopFunction("testing.(*T).Run"),
		goleak.IgnoreTopFunction("testing.(*T).Parallel"),
		// Ignore HTTP/2 transport goroutines from stdlib (connection pooling)
		goleak.IgnoreTopFunction("net/http.(*http2ClientConn).readLoop"),
		goleak.IgnoreTopFunction("net/http.(*http2clientConnReadLoop).run"),
		goleak.IgnoreTopFunction("internal/poll.runtime_pollWait"),
	)
}

// TestClientShutdown_NoLeaks verifies that shutting down a client
// properly cleans up all goroutines.
func TestClientShutdown_NoLeaks(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("testing.(*T).Run"),
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(IngestionResult{
				Successes: []IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithBatchSize(10),
		WithFlushInterval(100*time.Millisecond),
		WithTimeout(5*time.Second),
		WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx := context.Background()

	// Create traces with spans and generations
	for i := 0; i < 50; i++ {
		trace, err := client.NewTrace().Name("leak-test").Create(ctx)
		if err != nil {
			continue
		}
		span, err := trace.NewSpan().Name("span").Create(ctx)
		if err != nil {
			continue
		}
		span.End(ctx)
	}

	// Flush and shutdown
	if err := client.Flush(ctx); err != nil {
		t.Logf("Flush warning: %v", err)
	}

	shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Give goroutines time to exit
	time.Sleep(100 * time.Millisecond)
}

// TestClientShutdown_WithPendingEvents verifies no leaks when shutting down
// with events still pending.
func TestClientShutdown_WithPendingEvents(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("testing.(*T).Run"),
	)

	// Slow server to cause pending events
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingestion" {
			time.Sleep(50 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(IngestionResult{
				Successes: []IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithBatchSize(5),
		WithFlushInterval(1*time.Hour), // Don't auto-flush
		WithTimeout(5*time.Second),
		WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx := context.Background()

	// Queue events without flushing
	for i := 0; i < 20; i++ {
		client.NewTrace().Name("pending-test").Create(ctx)
	}

	// Shutdown should drain pending events
	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := client.Shutdown(shutdownCtx); err != nil {
		t.Logf("Shutdown warning: %v", err)
	}

	// Give goroutines time to exit
	time.Sleep(100 * time.Millisecond)
}

// TestClientShutdown_Timeout verifies no leaks when shutdown times out.
func TestClientShutdown_Timeout(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("testing.(*T).Run"),
	)

	// Very slow server to force timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingestion" {
			time.Sleep(5 * time.Second) // Longer than shutdown timeout
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(IngestionResult{
				Successes: []IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithBatchSize(1),
		WithFlushInterval(1*time.Hour),
		WithTimeout(1*time.Second),
		WithShutdownTimeout(2*time.Second),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx := context.Background()

	// Queue events
	for i := 0; i < 5; i++ {
		client.NewTrace().Name("timeout-test").Create(ctx)
	}

	// Trigger a flush to start slow requests
	go client.Flush(ctx)
	time.Sleep(50 * time.Millisecond)

	// Shutdown with short timeout - should timeout but not leak
	shutdownCtx, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
	defer cancel()

	err = client.Shutdown(shutdownCtx)
	// We expect a timeout error here
	if err != nil {
		t.Logf("Expected timeout error: %v", err)
	}

	// Give goroutines time to exit after context cancellation
	time.Sleep(200 * time.Millisecond)
}

// TestMultipleClients_NoLeaks verifies that creating and shutting down
// multiple clients doesn't leak goroutines.
func TestMultipleClients_NoLeaks(t *testing.T) {
	defer goleak.VerifyNone(t,
		goleak.IgnoreTopFunction("testing.(*T).Run"),
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(IngestionResult{
				Successes: []IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	ctx := context.Background()

	// Create and shutdown multiple clients
	for i := 0; i < 5; i++ {
		client, err := New(
			"pk-lf-test-key",
			"sk-lf-test-key",
			WithBaseURL(server.URL),
			WithTimeout(5*time.Second),
			WithShutdownTimeout(10*time.Second),
		)
		if err != nil {
			t.Fatalf("New failed: %v", err)
		}

		// Do some work
		for j := 0; j < 10; j++ {
			client.NewTrace().Name("multi-client-test").Create(ctx)
		}

		// Shutdown
		shutdownCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		if err := client.Shutdown(shutdownCtx); err != nil {
			t.Logf("Shutdown %d warning: %v", i, err)
		}
		cancel()
	}

	// Give goroutines time to exit
	time.Sleep(100 * time.Millisecond)
}
