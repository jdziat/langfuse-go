package langfuse_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jdziat/langfuse-go"
)

// TestHighThroughput tests the SDK under high-throughput conditions.
// This test is skipped in short mode.
// Note: This test does not verify exact event counts because ingestionRequest
// is unexported and cannot be decoded in external tests.
func TestHighThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	var requestCount atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			requestCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client, err := langfuse.New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		langfuse.WithBaseURL(server.URL),
		langfuse.WithBatchSize(100),
		langfuse.WithFlushInterval(100*time.Millisecond),
		langfuse.WithTimeout(10*time.Second),
		langfuse.WithShutdownTimeout(30*time.Second),
		langfuse.WithBatchQueueSize(500),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	const (
		numGoroutines      = 50
		eventsPerGoroutine = 500
	)

	var (
		wg      sync.WaitGroup
		created atomic.Int64
		errors  atomic.Int64
	)

	ctx := context.Background()
	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				_, err := client.NewTrace().
					Name("stress-test").
					Metadata(map[string]any{
						"goroutine": goroutineID,
						"event":     j,
					}).
					Create(ctx)
				if err != nil {
					errors.Add(1)
				} else {
					created.Add(1)
				}
			}
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Created %d events in %v (%.0f events/sec)",
		created.Load(), elapsed, float64(created.Load())/elapsed.Seconds())
	t.Logf("Errors: %d (%.2f%%)",
		errors.Load(), float64(errors.Load())/float64(numGoroutines*eventsPerGoroutine)*100)

	// Flush remaining events
	if err := client.Flush(ctx); err != nil {
		t.Logf("Flush warning: %v", err)
	}

	// Shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := client.Shutdown(shutdownCtx); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Allow time for final events to be counted
	time.Sleep(100 * time.Millisecond)

	createdCount := created.Load()
	requests := requestCount.Load()

	t.Logf("Total requests made: %d, events created: %d", requests, createdCount)

	// We expect requests to have been made
	if requests == 0 {
		t.Errorf("Expected requests to be made, got 0")
	}
}

// TestHighThroughputWithBackpressure tests the SDK with backpressure enabled.
// Note: This test does not verify exact event counts because ingestionRequest
// is unexported and cannot be decoded in external tests.
func TestHighThroughputWithBackpressure(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	var requestCount atomic.Int64

	// Slow server to trigger backpressure
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			time.Sleep(10 * time.Millisecond) // Slow processing
			requestCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	var backpressureCallbackCount atomic.Int64

	client, err := langfuse.New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		langfuse.WithBaseURL(server.URL),
		langfuse.WithBatchSize(50),
		langfuse.WithFlushInterval(100*time.Millisecond),
		langfuse.WithTimeout(10*time.Second),
		langfuse.WithShutdownTimeout(30*time.Second),
		langfuse.WithBatchQueueSize(100),
		langfuse.WithOnBackpressure(func(state langfuse.QueueState) {
			backpressureCallbackCount.Add(1)
		}),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	const (
		numGoroutines      = 20
		eventsPerGoroutine = 200
	)

	var (
		wg      sync.WaitGroup
		created atomic.Int64
	)

	ctx := context.Background()
	start := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				_, err := client.NewTrace().Name("backpressure-stress").Create(ctx)
				if err == nil {
					created.Add(1)
				}
			}
		}()
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Created %d events in %v (%.0f events/sec)",
		created.Load(), elapsed, float64(created.Load())/elapsed.Seconds())
	t.Logf("Backpressure callbacks: %d", backpressureCallbackCount.Load())

	// Check backpressure status
	status := client.BackpressureStatus()
	t.Logf("Backpressure stats: dropped=%d, blocked=%d, level=%v",
		status.DroppedCount, status.BlockedCount, status.MonitorStats.CurrentLevel)

	// Shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := client.Shutdown(shutdownCtx); err != nil {
		t.Logf("Shutdown warning: %v", err)
	}
}

// TestConcurrentTracesAndSpans tests creating traces and spans concurrently.
func TestConcurrentTracesAndSpans(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client, err := langfuse.New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		langfuse.WithBaseURL(server.URL),
		langfuse.WithBatchSize(100),
		langfuse.WithFlushInterval(100*time.Millisecond),
		langfuse.WithTimeout(10*time.Second),
		langfuse.WithShutdownTimeout(30*time.Second),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	const numGoroutines = 20

	var (
		wg          sync.WaitGroup
		traceCount  atomic.Int64
		spanCount   atomic.Int64
		genCount    atomic.Int64
		traceErrors atomic.Int64
		spanErrors  atomic.Int64
		genErrors   atomic.Int64
	)

	ctx := context.Background()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				// Create trace
				trace, err := client.NewTrace().Name("concurrent-test").Create(ctx)
				if err != nil {
					traceErrors.Add(1)
					continue
				}
				traceCount.Add(1)

				// Create nested spans
				for k := 0; k < 3; k++ {
					span, err := trace.NewSpan().Name("span").Create(ctx)
					if err != nil {
						spanErrors.Add(1)
						continue
					}
					spanCount.Add(1)

					// Create generation within span
					gen, err := span.NewGeneration().Name("gen").Model("test-model").Create(ctx)
					if err != nil {
						genErrors.Add(1)
						continue
					}
					genCount.Add(1)

					// End generation
					gen.End(ctx)

					// End span
					span.End(ctx)
				}
			}
		}()
	}

	wg.Wait()

	t.Logf("Traces: %d (errors: %d)", traceCount.Load(), traceErrors.Load())
	t.Logf("Spans: %d (errors: %d)", spanCount.Load(), spanErrors.Load())
	t.Logf("Generations: %d (errors: %d)", genCount.Load(), genErrors.Load())

	// Shutdown
	shutdownCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := client.Shutdown(shutdownCtx); err != nil {
		t.Logf("Shutdown warning: %v", err)
	}

	// Verify counts
	expectedTraces := int64(numGoroutines * 50)
	expectedSpans := expectedTraces * 3

	if traceCount.Load()+traceErrors.Load() != expectedTraces {
		t.Errorf("Expected %d total trace attempts, got %d",
			expectedTraces, traceCount.Load()+traceErrors.Load())
	}
	if spanCount.Load()+spanErrors.Load() < expectedSpans/2 {
		t.Errorf("Expected at least %d span attempts, got %d",
			expectedSpans/2, spanCount.Load()+spanErrors.Load())
	}
}

// TestShutdownUnderContinuousLoad tests shutdown while events are continuously being created.
func TestShutdownUnderContinuousLoad(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client, err := langfuse.New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		langfuse.WithBaseURL(server.URL),
		langfuse.WithBatchSize(10),
		langfuse.WithFlushInterval(100*time.Millisecond),
		langfuse.WithTimeout(5*time.Second),
		langfuse.WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Start continuous load
	loadCtx, loadCancel := context.WithCancel(context.Background())
	var loadWg sync.WaitGroup
	var eventCount atomic.Int64
	var errorCount atomic.Int64

	for i := 0; i < 10; i++ {
		loadWg.Add(1)
		go func() {
			defer loadWg.Done()
			for loadCtx.Err() == nil {
				_, err := client.NewTrace().Name("continuous-load").Create(context.Background())
				if err != nil {
					errorCount.Add(1)
				} else {
					eventCount.Add(1)
				}
				time.Sleep(time.Millisecond) // Small delay to prevent tight loop
			}
		}()
	}

	// Let it run for a bit
	time.Sleep(300 * time.Millisecond)

	// Initiate shutdown while load is running
	shutdownDone := make(chan error)
	go func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		shutdownDone <- client.Shutdown(shutdownCtx)
	}()

	// Stop load generators after a short delay
	time.Sleep(100 * time.Millisecond)
	loadCancel()
	loadWg.Wait()

	// Wait for shutdown
	err = <-shutdownDone
	if err != nil {
		t.Logf("Shutdown completed with: %v", err)
	}

	t.Logf("Events created: %d, Errors (after close): %d", eventCount.Load(), errorCount.Load())

	// Verify we created a reasonable number of events
	if eventCount.Load() < 100 {
		t.Errorf("Expected at least 100 events, got %d", eventCount.Load())
	}
}

// TestRapidCreateShutdown tests rapidly creating and shutting down clients.
func TestRapidCreateShutdown(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	var wg sync.WaitGroup
	var successCount atomic.Int64
	var errorCount atomic.Int64

	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				client, err := langfuse.New(
					"pk-lf-test-key",
					"sk-lf-test-key",
					langfuse.WithBaseURL(server.URL),
					langfuse.WithTimeout(5*time.Second),
					langfuse.WithShutdownTimeout(10*time.Second),
				)
				if err != nil {
					errorCount.Add(1)
					continue
				}

				// Do minimal work
				client.NewTrace().Name("rapid-test").Create(context.Background())

				// Shutdown immediately
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				if err := client.Shutdown(ctx); err != nil {
					errorCount.Add(1)
				} else {
					successCount.Add(1)
				}
				cancel()
			}
		}()
	}

	wg.Wait()

	t.Logf("Successful create/shutdown cycles: %d, Errors: %d",
		successCount.Load(), errorCount.Load())

	// Most should succeed
	if successCount.Load() < 80 {
		t.Errorf("Expected at least 80 successful cycles, got %d", successCount.Load())
	}
}
