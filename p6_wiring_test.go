package langfuse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestWithClassifiedHooks_FiresOnRequest verifies that a ClassifiedHook configured
// via WithClassifiedHooks is actually invoked for an HTTP request. This guards the
// end-to-end wiring: the converter forwards ClassifiedHooks to the core config and
// newHTTPClient installs them on the request hook chain. Without that wiring the
// hook never fires and BeforeRequest/AfterResponse are never observed.
func TestWithClassifiedHooks_FiresOnRequest(t *testing.T) {
	var beforeCalled, afterCalled atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthStatus{Status: "ok", Version: "1.0.0"})
	}))
	defer server.Close()

	hook := NewClassifiedHook("test-observer", HTTPHookFunc{
		Before: func(ctx context.Context, req *http.Request) error {
			beforeCalled.Store(true)
			return nil
		},
		After: func(ctx context.Context, req *http.Request, resp *http.Response, d time.Duration, err error) {
			afterCalled.Store(true)
		},
	}, HookPriorityObservational)

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithClassifiedHooks(hook),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	// A synchronous Health request exercises the HTTP client (and its hook chain).
	if _, err := client.Health(context.Background()); err != nil {
		t.Fatalf("Health failed: %v", err)
	}

	if !beforeCalled.Load() {
		t.Error("classified hook BeforeRequest was not invoked for an HTTP request")
	}
	if !afterCalled.Load() {
		t.Error("classified hook AfterResponse was not invoked for an HTTP request")
	}
}

// TestWithClassifiedHooks_CriticalAbortsRequest verifies that a critical classified
// hook returning an error aborts the request. This proves the chain's priority-aware
// behavior is actually in effect (not just that some generic hook ran).
func TestWithClassifiedHooks_CriticalAbortsRequest(t *testing.T) {
	var reached atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reached.Store(true)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthStatus{Status: "ok"})
	}))
	defer server.Close()

	hook := NewClassifiedHook("test-critical", HTTPHookFunc{
		Before: func(ctx context.Context, req *http.Request) error {
			return errCriticalHookTest
		},
	}, HookPriorityCritical)

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithMaxRetries(0),
		WithClassifiedHooks(hook),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	if _, err := client.Health(context.Background()); err == nil {
		t.Error("expected Health to fail when a critical classified hook aborts the request")
	}
	if reached.Load() {
		t.Error("server was reached even though a critical hook should have aborted the request")
	}
}

// TestWithMetricsRecorder_RecordsMetric verifies that enabling the metrics recorder
// (WithMetricsRecorder + WithMetrics) results in at least one metric being recorded
// when the SDK performs activity, and that the recorder is retrievable.
func TestWithMetricsRecorder_RecordsMetric(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	rec := newMetricsTestRecorder()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithFlushInterval(1*time.Hour), // prevent auto-flush timing noise
		WithBatchSize(1),               // each event triggers an immediate background send
		WithMetrics(rec),
		WithMetricsRecorder(),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	if client.MetricsRecorder() == nil {
		t.Fatal("MetricsRecorder() returned nil when WithMetricsRecorder + WithMetrics were set")
	}
	if !client.MetricsRecorder().IsEnabled() {
		t.Error("MetricsRecorder().IsEnabled() = false, want true")
	}

	ctx := context.Background()
	if _, err := client.NewTrace().Name("metrics-trace").Create(ctx); err != nil {
		t.Fatalf("Create trace failed: %v", err)
	}

	// The background batch send records batch/event metrics. Poll until at least
	// one metric is observed.
	if !waitFor(2*time.Second, func() bool {
		return rec.getCounter("langfuse.batch.sent") > 0 ||
			rec.getCounter("langfuse.events.sent") > 0 ||
			rec.getGauge("langfuse.pending_events") > 0
	}) {
		t.Error("expected at least one metric to be recorded after SDK activity, got none")
	}
}

// TestWithMetricsRecorder_NotEnabledWithoutMetrics verifies that the recorder is not
// constructed when WithMetricsRecorder is set but no Metrics implementation is, which
// matches the documented "requires Metrics to be set" contract.
func TestWithMetricsRecorder_NotEnabledWithoutMetrics(t *testing.T) {
	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithMetricsRecorder(),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	if client.MetricsRecorder() != nil {
		t.Error("MetricsRecorder() should be nil when WithMetricsRecorder is set without WithMetrics")
	}
}

// TestWithOnAsyncError_InvokedOnBackgroundFailure verifies that a background batch
// send failure invokes the WithOnAsyncError handler with a non-nil *AsyncError.
func TestWithOnAsyncError_InvokedOnBackgroundFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Fail every ingestion request so the background send errors out.
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"boom"}`))
	}))
	defer server.Close()

	var (
		mu       sync.Mutex
		received *AsyncError
	)

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithFlushInterval(1*time.Hour),
		WithBatchSize(1),              // each event triggers an immediate background send
		WithRetryStrategy(&NoRetry{}), // fail fast instead of retrying for the test timeout
		WithStructuredLogger(NopLogger{}),
		WithOnAsyncError(func(ae *AsyncError) {
			mu.Lock()
			received = ae
			mu.Unlock()
		}),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	if _, err := client.NewTrace().Name("async-error-trace").Create(ctx); err != nil {
		t.Fatalf("Create trace failed: %v", err)
	}

	if !waitFor(3*time.Second, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return received != nil
	}) {
		t.Fatal("OnAsyncError handler was not invoked on a background batch-send failure")
	}

	mu.Lock()
	ae := received
	mu.Unlock()

	if ae == nil {
		t.Fatal("received *AsyncError is nil")
	}
	if ae.Err == nil {
		t.Error("AsyncError.Err should be non-nil for a failed batch send")
	}
	if ae.Operation != AsyncOpBatchSend {
		t.Errorf("AsyncError.Operation = %q, want %q", ae.Operation, AsyncOpBatchSend)
	}
}

// TestWithOnAsyncError_PreservesExistingErrorHandler verifies that wiring
// OnAsyncError does not drop a directly-configured ErrorHandler: both fire on a
// background failure.
func TestWithOnAsyncError_PreservesExistingErrorHandler(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"boom"}`))
	}))
	defer server.Close()

	var rawCalled, asyncCalled atomic.Bool

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithFlushInterval(1*time.Hour),
		WithBatchSize(1),
		WithRetryStrategy(&NoRetry{}),
		WithStructuredLogger(NopLogger{}),
		WithErrorHandler(func(error) { rawCalled.Store(true) }),
		WithOnAsyncError(func(*AsyncError) { asyncCalled.Store(true) }),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	if _, err := client.NewTrace().Name("preserve-handler-trace").Create(context.Background()); err != nil {
		t.Fatalf("Create trace failed: %v", err)
	}

	if !waitFor(3*time.Second, func() bool {
		return rawCalled.Load() && asyncCalled.Load()
	}) {
		t.Errorf("expected both ErrorHandler and OnAsyncError to fire; raw=%v async=%v",
			rawCalled.Load(), asyncCalled.Load())
	}
}

// errCriticalHookTest is a sentinel error returned by the critical hook in tests.
var errCriticalHookTest = &hookTestError{}

type hookTestError struct{}

func (*hookTestError) Error() string { return "critical hook test failure" }

// waitFor polls cond until it returns true or the timeout elapses.
func waitFor(timeout time.Duration, cond func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if cond() {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return cond()
}
