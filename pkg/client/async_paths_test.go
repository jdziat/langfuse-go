package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	pkgerrors "github.com/jdziat/langfuse-go/pkg/errors"
	pkghttp "github.com/jdziat/langfuse-go/pkg/http"
)

// These tests close previously-untested critical paths in the batching layer:
// the background batch-send error bridge (handleError), the queue-overflow drop
// path (handleQueueFull / ErrBatchDropped), and the debug/info/error logging
// helpers. They run in the internal `client` package so they can drive the
// unexported paths directly and deterministically, without relying on goroutine
// timing.

// captureLogger records messages routed through the Logger interface.
type captureLogger struct {
	mu       sync.Mutex
	messages []string
}

func (l *captureLogger) Printf(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	// Store the raw format so callers can assert on the template even when no
	// args are supplied (matching how the client invokes Printf).
	l.messages = append(l.messages, format)
}

func (l *captureLogger) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.messages...)
}

// captureStructuredLogger records messages per level.
type captureStructuredLogger struct {
	mu     sync.Mutex
	debug  []string
	info   []string
	warn   []string
	errors []string
}

func (l *captureStructuredLogger) Debug(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.debug = append(l.debug, msg)
}

func (l *captureStructuredLogger) Info(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.info = append(l.info, msg)
}

func (l *captureStructuredLogger) Warn(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warn = append(l.warn, msg)
}

func (l *captureStructuredLogger) Error(msg string, _ ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors = append(l.errors, msg)
}

func (l *captureStructuredLogger) snapshot() (dbg, info, warn, errs []string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.debug...),
		append([]string(nil), l.info...),
		append([]string(nil), l.warn...),
		append([]string(nil), l.errors...)
}

// captureMetrics records counter increments and gauge sets.
type captureMetrics struct {
	mu       sync.Mutex
	counters map[string]int64
	gauges   map[string]float64
}

func newCaptureMetrics() *captureMetrics {
	return &captureMetrics{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

func (m *captureMetrics) IncrementCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

func (m *captureMetrics) RecordDuration(string, time.Duration) {}

func (m *captureMetrics) SetGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *captureMetrics) counter(name string) (int64, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	v, ok := m.counters[name]
	return v, ok
}

// failingServer returns an httptest server that always responds with the given
// status, plus a counter of how many ingestion requests it received.
func failingServer(t *testing.T, status int) (*httptest.Server, func() int) {
	t.Helper()
	var (
		mu    sync.Mutex
		count int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		count++
		mu.Unlock()
		w.WriteHeader(status)
		_, _ = w.Write([]byte(`{"message":"boom"}`))
	}))
	t.Cleanup(srv.Close)
	return srv, func() int {
		mu.Lock()
		defer mu.Unlock()
		return count
	}
}

// newTestClient builds a client aimed at the given base URL with retries
// disabled (so a failing response surfaces immediately, keeping the async-error
// tests fast and deterministic) and the supplied options applied last.
//
// Note: WithMaxRetries(0) alone is insufficient - ApplyDefaults rewrites a zero
// MaxRetries back to the default of 3 - so a NoRetry strategy is injected to
// truly disable retries.
func newTestClient(t *testing.T, baseURL string, opts ...ConfigOption) *Client {
	t.Helper()
	base := []ConfigOption{
		WithBaseURL(baseURL),
		WithRetryStrategy(&pkghttp.NoRetry{}),
		WithFlushInterval(time.Hour), // keep the flush loop out of the way
	}
	c, err := New("pk-lf-test-key", "sk-lf-test-key", append(base, opts...)...)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	return c
}

func sampleEvents(n int) []IngestionEvent {
	events := make([]IngestionEvent, n)
	for i := range events {
		events[i] = IngestionEvent{
			ID:        "evt",
			Type:      EventTypeEventCreate,
			Timestamp: Now(),
			Body:      map[string]any{"i": i},
		}
	}
	return events
}

// TestSendBatchFailureInvokesErrorHandler proves that a failed batch send (the
// real HTTP path, not a direct handleError call) bridges to the configured
// ErrorHandler with the underlying *APIError. This is the async-error path used
// by processBatchRequest, the background sender, and the drain/flush loops.
func TestSendBatchFailureInvokesErrorHandler(t *testing.T) {
	srv, _ := failingServer(t, http.StatusInternalServerError)

	var (
		mu       sync.Mutex
		captured []error
	)
	metrics := newCaptureMetrics()

	c := newTestClient(t, srv.URL,
		WithErrorHandler(func(err error) {
			mu.Lock()
			captured = append(captured, err)
			mu.Unlock()
		}),
		WithMetrics(metrics),
	)
	defer c.Shutdown(context.Background())

	// processBatchRequest runs synchronously here, so by the time it returns the
	// send has failed and handleError has been invoked - no sleeps required.
	c.processBatchRequest(batchRequest{events: sampleEvents(1), ctx: context.Background()})

	mu.Lock()
	defer mu.Unlock()
	if len(captured) != 1 {
		t.Fatalf("expected exactly one async error, got %d: %v", len(captured), captured)
	}

	var apiErr *pkgerrors.APIError
	if !errors.As(captured[0], &apiErr) {
		t.Fatalf("expected error to wrap *APIError, got %T: %v", captured[0], captured[0])
	}
	if apiErr.StatusCode != http.StatusInternalServerError {
		t.Errorf("APIError.StatusCode = %d, want %d", apiErr.StatusCode, http.StatusInternalServerError)
	}

	if got, ok := metrics.counter("langfuse.errors"); !ok || got != 1 {
		t.Errorf("langfuse.errors counter = %d (present=%v), want 1", got, ok)
	}
}

// TestHandleErrorUnhandledFallback ensures that when no handler, logger, or
// structured logger is configured, handleError still records the error metric
// and does not panic (the stderr fallback path).
func TestHandleErrorUnhandledFallback(t *testing.T) {
	metrics := newCaptureMetrics()
	c := newTestClient(t, "http://127.0.0.1:0", WithMetrics(metrics))
	defer c.Shutdown(context.Background())

	c.handleError(errors.New("boom"))

	if got, ok := metrics.counter("langfuse.errors"); !ok || got != 1 {
		t.Errorf("langfuse.errors counter = %d (present=%v), want 1", got, ok)
	}
}

// TestHandleQueueFullDropReturnsErrBatchDropped proves the overflow drop path:
// when every background-sender slot is occupied, handleQueueFull drops the batch,
// returns ErrBatchDropped, and increments the drop counter.
func TestHandleQueueFullDropReturnsErrBatchDropped(t *testing.T) {
	srv, _ := failingServer(t, http.StatusInternalServerError)
	metrics := newCaptureMetrics()

	// One background-sender slot, which we occupy by hand to force the drop path.
	c := newTestClient(t, srv.URL,
		WithMaxBackgroundSenders(1),
		WithMetrics(metrics),
	)
	defer c.Shutdown(context.Background())

	// Occupy the only semaphore slot so the non-blocking acquire in
	// handleQueueFull falls through to the drop branch deterministically.
	c.backgroundSendSem <- struct{}{}
	defer func() { <-c.backgroundSendSem }()

	err := c.handleQueueFull(sampleEvents(2))
	if !errors.Is(err, ErrBatchDropped) {
		t.Fatalf("handleQueueFull error = %v, want ErrBatchDropped", err)
	}

	if got, ok := metrics.counter("langfuse.batches_dropped"); !ok || got != 1 {
		t.Errorf("langfuse.batches_dropped counter = %d (present=%v), want 1", got, ok)
	}
}

// TestHandleQueueFullSpawnsSenderWhenSlotFree proves the no-drop default
// behavior: when a background-sender slot is available, handleQueueFull returns
// nil (no data loss) and the spawned goroutine actually sends the batch.
func TestHandleQueueFullSpawnsSenderWhenSlotFree(t *testing.T) {
	var (
		mu       sync.Mutex
		received int
	)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		mu.Lock()
		received++
		mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"successes":[{"id":"evt","status":200}]}`))
	}))
	defer srv.Close()

	metrics := newCaptureMetrics()
	c := newTestClient(t, srv.URL,
		WithMaxBackgroundSenders(1),
		WithMetrics(metrics),
	)
	defer c.Shutdown(context.Background())

	if err := c.handleQueueFull(sampleEvents(1)); err != nil {
		t.Fatalf("handleQueueFull returned error with a free slot: %v", err)
	}

	// The send happens in a tracked goroutine; wait for the semaphore to drain
	// back to empty, which signals the background sender finished.
	deadline := time.Now().Add(2 * time.Second)
	for {
		if len(c.backgroundSendSem) == 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("background sender did not release its semaphore slot in time")
		}
		time.Sleep(2 * time.Millisecond)
	}

	mu.Lock()
	got := received
	mu.Unlock()
	if got != 1 {
		t.Errorf("server received %d batches, want 1", got)
	}

	if dropped, ok := metrics.counter("langfuse.batches_dropped"); ok && dropped != 0 {
		t.Errorf("langfuse.batches_dropped = %d, want 0 (no drop expected)", dropped)
	}
}

// TestQueueEventReturnsErrBatchDroppedOnOverflow exercises the drop path end to
// end through the public QueueEvent entry point: a full batch queue plus a fully
// occupied background-sender semaphore must surface ErrBatchDropped to the
// caller so data loss is observable.
func TestQueueEventReturnsErrBatchDroppedOnOverflow(t *testing.T) {
	srv, _ := failingServer(t, http.StatusInternalServerError)

	c := newTestClient(t, srv.URL,
		WithBatchSize(1),            // every event immediately forms a batch
		WithBatchQueueSize(1),       // tiny channel so it fills at once
		WithMaxBackgroundSenders(1), // single overflow slot
	)
	defer c.Shutdown(context.Background())

	// Stop the batch processor from draining the queue so the channel stays full,
	// then occupy the only background-sender slot. With both saturated, the next
	// full batch has nowhere to go and must be dropped.
	c.cancel() // halt batchProcessor/flushLoop via ctx.Done()
	time.Sleep(10 * time.Millisecond)

	c.batchQueue <- batchRequest{events: sampleEvents(1), ctx: context.Background()}
	c.backgroundSendSem <- struct{}{}
	defer func() { <-c.backgroundSendSem }()

	// BatchSize is 1, so this single event forms a batch that finds the queue
	// full and the semaphore occupied -> ErrBatchDropped.
	err := c.QueueEvent(context.Background(), sampleEvents(1)[0])
	if !errors.Is(err, ErrBatchDropped) {
		t.Fatalf("QueueEvent error = %v, want ErrBatchDropped", err)
	}
}

// TestLoggingHelpersRouteToLogger verifies the debug/info/error logging helpers
// route through the plain Logger interface.
func TestLoggingHelpersRouteToLogger(t *testing.T) {
	logger := &captureLogger{}
	c := newTestClient(t, "http://127.0.0.1:0", WithLogger(logger))
	defer c.Shutdown(context.Background())

	c.log("debug message %d", 1)
	c.logInfo("info message", "k", "v")
	c.logError("error message", "code", 500)

	msgs := logger.snapshot()
	if len(msgs) < 3 {
		t.Fatalf("expected at least 3 logged messages, got %d: %v", len(msgs), msgs)
	}

	var sawDebug, sawInfo, sawError bool
	for _, m := range msgs {
		switch {
		case m == "debug message %d":
			sawDebug = true
		case strings.Contains(m, "info message"):
			sawInfo = true
		case strings.Contains(m, "[ERROR]") && strings.Contains(m, "error message"):
			sawError = true
		}
	}
	if !sawDebug {
		t.Error("log() did not route to Logger.Printf")
	}
	if !sawInfo {
		t.Error("logInfo() did not route to Logger.Printf")
	}
	if !sawError {
		t.Error("logError() did not route to Logger.Printf with [ERROR] prefix")
	}
}

// TestLoggingHelpersRouteToStructuredLogger verifies the helpers prefer the
// structured logger and route each helper to the matching level.
func TestLoggingHelpersRouteToStructuredLogger(t *testing.T) {
	slog := &captureStructuredLogger{}
	c := newTestClient(t, "http://127.0.0.1:0", WithStructuredLogger(slog))
	defer c.Shutdown(context.Background())

	c.log("debug message")
	c.logInfo("info message")
	c.logError("error message")

	dbg, info, _, errs := slog.snapshot()
	if len(dbg) == 0 || dbg[0] != "debug message" {
		t.Errorf("Debug level = %v, want [\"debug message\"]", dbg)
	}
	if len(info) == 0 || info[0] != "info message" {
		t.Errorf("Info level = %v, want [\"info message\"]", info)
	}
	if len(errs) == 0 || errs[0] != "error message" {
		t.Errorf("Error level = %v, want [\"error message\"]", errs)
	}
}
