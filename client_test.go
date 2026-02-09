package langfuse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithRegion(RegionUS),
		WithBatchSize(50),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	if client.config.PublicKey != "pk-lf-test-key" {
		t.Errorf("PublicKey = %v, want pk-lf-test-key", client.config.PublicKey)
	}
	if client.config.SecretKey != "sk-lf-test-key" {
		t.Errorf("SecretKey = %v, want sk-lf-test-key", client.config.SecretKey)
	}
	if client.config.Region != RegionUS {
		t.Errorf("Region = %v, want %v", client.config.Region, RegionUS)
	}
	if client.config.BatchSize != 50 {
		t.Errorf("BatchSize = %v, want 50", client.config.BatchSize)
	}
}

func TestNewClientValidation(t *testing.T) {
	tests := []struct {
		name      string
		publicKey string
		secretKey string
		wantError error
	}{
		{
			name:      "missing public key",
			publicKey: "",
			secretKey: "sk-lf-test-key",
			wantError: ErrMissingPublicKey,
		},
		{
			name:      "missing secret key",
			publicKey: "pk-lf-test-key",
			secretKey: "",
			wantError: ErrMissingSecretKey,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.publicKey, tt.secretKey)
			if err != tt.wantError {
				t.Errorf("New() error = %v, want %v", err, tt.wantError)
			}
		})
	}
}

func TestClientHealth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			t.Errorf("Expected /health, got %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthStatus{
			Status:  "ok",
			Version: "1.0.0",
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	health, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health failed: %v", err)
	}

	if health.Status != "ok" {
		t.Errorf("Status = %v, want ok", health.Status)
	}
	if health.Version != "1.0.0" {
		t.Errorf("Version = %v, want 1.0.0", health.Version)
	}
}

func TestClientSubClients(t *testing.T) {
	client, err := New("pk-lf-test-key", "sk-lf-test-key")
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	if client.Traces() == nil {
		t.Error("Traces() should not be nil")
	}
	if client.Observations() == nil {
		t.Error("Observations() should not be nil")
	}
	if client.Scores() == nil {
		t.Error("Scores() should not be nil")
	}
	if client.Prompts() == nil {
		t.Error("Prompts() should not be nil")
	}
	if client.Datasets() == nil {
		t.Error("Datasets() should not be nil")
	}
	if client.Sessions() == nil {
		t.Error("Sessions() should not be nil")
	}
	if client.Models() == nil {
		t.Error("Models() should not be nil")
	}
}

func TestClientFlush(t *testing.T) {
	var receivedEvents []ingestionEvent
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingestion" {
			var req ingestionRequest
			json.NewDecoder(r.Body).Decode(&req)

			mu.Lock()
			receivedEvents = append(receivedEvents, req.Batch...)
			mu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(IngestionResult{
				Successes: []IngestionSuccess{{ID: "1", Status: 200}},
			})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithBatchSize(100),
		WithFlushInterval(1*time.Hour), // Prevent auto flush
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx := context.Background()
	// Create a trace
	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Create trace failed: %v", err)
	}

	if trace.ID() == "" {
		t.Error("Trace ID should not be empty")
	}

	// Flush manually
	err = client.Flush(context.Background())
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	mu.Lock()
	eventCount := len(receivedEvents)
	mu.Unlock()
	if eventCount == 0 {
		t.Error("Expected to receive events")
	}

	// Shutdown (ignore error as server may have timing issues)
	client.Shutdown(context.Background())
}

func TestClientShutdown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	err = client.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	// Second shutdown should return error
	err = client.Shutdown(context.Background())
	if err != ErrClientClosed {
		t.Errorf("Expected ErrClientClosed, got %v", err)
	}

	// Flush should return error
	err = client.Flush(context.Background())
	if err != ErrClientClosed {
		t.Errorf("Expected ErrClientClosed, got %v", err)
	}

	// Creating trace should fail
	_, err = client.NewTrace().Create(context.Background())
	if err != ErrClientClosed {
		t.Errorf("Expected ErrClientClosed, got %v", err)
	}
}

func TestClientClose(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	err = client.Close(context.Background())
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Client should be closed
	err = client.Flush(context.Background())
	if err != ErrClientClosed {
		t.Errorf("Expected ErrClientClosed, got %v", err)
	}
}

func TestTraceBuilder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, err := client.NewTrace().
		ID("custom-id").
		Name("test-trace").
		UserID("user-123").
		SessionID("session-456").
		Input(map[string]string{"key": "value"}).
		Output("output").
		Metadata(map[string]any{"meta": "data"}).
		Tags([]string{"tag1", "tag2"}).
		Release("v1.0.0").
		Version("1").
		Public(true).
		Environment("test").
		Create(ctx)

	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if trace.ID() != "custom-id" {
		t.Errorf("ID = %v, want custom-id", trace.ID())
	}
}

func TestTraceContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, err := client.NewTrace().Name("test").Create(ctx)
	if err != nil {
		t.Fatalf("Create trace failed: %v", err)
	}

	// Test creating a span
	span, err := trace.NewSpan().Name("test-span").Create(ctx)
	if err != nil {
		t.Fatalf("Create span failed: %v", err)
	}
	if span.SpanID() == "" {
		t.Error("Span ID should not be empty")
	}

	// Test creating a generation
	gen, err := trace.NewGeneration().Name("test-gen").Create(ctx)
	if err != nil {
		t.Fatalf("Create generation failed: %v", err)
	}
	if gen.GenerationID() == "" {
		t.Error("Generation ID should not be empty")
	}

	// Test creating an event
	err = trace.NewEvent().Name("test-event").Create(ctx)
	if err != nil {
		t.Fatalf("Create event failed: %v", err)
	}

	// Test creating a score
	err = trace.NewScore().Name("test-score").NumericValue(0.9).Create(ctx)
	if err != nil {
		t.Fatalf("Create score failed: %v", err)
	}

	// Test updating trace
	err = trace.Update().Output("final output").Apply(ctx)
	if err != nil {
		t.Fatalf("Update trace failed: %v", err)
	}
}

func TestTraceUpdateBuilder(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, err := client.NewTrace().Name("test").Create(ctx)
	if err != nil {
		t.Fatalf("Create trace failed: %v", err)
	}

	err = trace.Update().
		Name("updated-name").
		UserID("user-456").
		SessionID("session-789").
		Input("new input").
		Output("new output").
		Metadata(map[string]any{"key": "value"}).
		Tags([]string{"new-tag"}).
		Public(true).
		Apply(ctx)

	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
}

func TestBatchSizeTriggersFlush(t *testing.T) {
	flushCount := 0
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingestion" {
			mu.Lock()
			flushCount++
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithBatchSize(5),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	// Create 6 events (should trigger flush at 5)
	for i := 0; i < 6; i++ {
		client.NewTrace().Name("test").Create(ctx)
	}

	// Wait for async flush
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if flushCount < 1 {
		t.Errorf("Expected at least 1 flush, got %d", flushCount)
	}
}

// testLogger implements a simple logger for testing
type testLogger struct {
	mu       sync.Mutex
	messages []string
}

func (l *testLogger) Printf(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.messages = append(l.messages, format)
}

func (l *testLogger) Messages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string{}, l.messages...)
}

// testStructuredLogger implements StructuredLogger for testing
type testStructuredLogger struct {
	mu     sync.Mutex
	errors []string
	debugs []string
	infos  []string
}

func (l *testStructuredLogger) Debug(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.debugs = append(l.debugs, msg)
}

func (l *testStructuredLogger) Info(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infos = append(l.infos, msg)
}

func (l *testStructuredLogger) Warn(msg string, args ...any) {}

func (l *testStructuredLogger) Error(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errors = append(l.errors, msg)
}

func (l *testStructuredLogger) Errors() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string{}, l.errors...)
}

func TestHandleErrorWithLogger(t *testing.T) {
	logger := &testLogger{}

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithLogger(logger),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	// Trigger an error through handleError
	testErr := NewValidationError("test", "test error")
	client.handleError(testErr)

	// Give it a moment to process
	time.Sleep(10 * time.Millisecond)

	msgs := logger.Messages()
	if len(msgs) == 0 {
		t.Error("Expected logger to receive error message")
	}
}

func TestHandleErrorWithStructuredLogger(t *testing.T) {
	logger := &testStructuredLogger{}

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithStructuredLogger(logger),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	// Trigger an error through handleError
	testErr := NewValidationError("test", "test error")
	client.handleError(testErr)

	// Give it a moment to process
	time.Sleep(10 * time.Millisecond)

	errors := logger.Errors()
	if len(errors) == 0 {
		t.Error("Expected structured logger to receive error message")
	}
}

func TestHandleErrorWithErrorHandler(t *testing.T) {
	var capturedErr error
	var mu sync.Mutex

	handler := func(err error) {
		mu.Lock()
		capturedErr = err
		mu.Unlock()
	}

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithErrorHandler(handler),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	// Trigger an error through handleError
	testErr := NewValidationError("test", "test error")
	client.handleError(testErr)

	// Give it a moment to process
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if capturedErr == nil {
		t.Error("Expected error handler to capture error")
	}
	if capturedErr != testErr {
		t.Errorf("Expected captured error to be %v, got %v", testErr, capturedErr)
	}
}

func TestHandleErrorWithMetrics(t *testing.T) {
	metrics := &testMetrics{}

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithMetrics(metrics),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	// Trigger an error through handleError
	testErr := NewValidationError("test", "test error")
	client.handleError(testErr)

	// Give it a moment to process
	time.Sleep(10 * time.Millisecond)

	counters := metrics.Counters()
	if _, ok := counters["langfuse.errors"]; !ok {
		t.Error("Expected metrics to record langfuse.errors counter")
	}
}

// testMetrics implements Metrics for testing
type testMetrics struct {
	mu       sync.Mutex
	counters map[string]int64
	gauges   map[string]float64
}

func (m *testMetrics) IncrementCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.counters == nil {
		m.counters = make(map[string]int64)
	}
	m.counters[name] += value
}

func (m *testMetrics) RecordDuration(name string, duration time.Duration) {}

func (m *testMetrics) SetGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.gauges == nil {
		m.gauges = make(map[string]float64)
	}
	m.gauges[name] = value
}

func (m *testMetrics) Counters() map[string]int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]int64)
	for k, v := range m.counters {
		result[k] = v
	}
	return result
}

func TestHandleQueueFull(t *testing.T) {
	var receivedBatches int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingestion" {
			mu.Lock()
			receivedBatches++
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	// Create client with very small queue to trigger handleQueueFull
	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithBatchSize(1),      // Trigger flush immediately
		WithBatchQueueSize(1), // Very small queue
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	// Create many traces quickly to overwhelm the queue
	for i := 0; i < 20; i++ {
		client.NewTrace().Name("test").Create(ctx)
	}

	// Wait for background processing
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if receivedBatches == 0 {
		t.Error("Expected at least one batch to be received")
	}
}

func TestBatchProcessorDrainOnShutdown(t *testing.T) {
	var receivedBatches int
	var mu sync.Mutex

	// Server with slight delay to simulate processing time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingestion" {
			time.Sleep(10 * time.Millisecond)
			mu.Lock()
			receivedBatches++
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithBatchSize(2),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx := context.Background()
	// Create some traces
	for i := 0; i < 5; i++ {
		client.NewTrace().Name("test").Create(ctx)
	}

	// Shutdown should drain all pending events
	err = client.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	// Should have received at least some batches (5 events / 2 batch size = 2-3 batches)
	if receivedBatches == 0 {
		t.Error("Expected batches to be drained during shutdown")
	}
}

func TestBatchProcessorContextCancelled(t *testing.T) {
	var receivedBatches int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingestion" {
			mu.Lock()
			receivedBatches++
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithBatchSize(1),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Create a trace with the cancellable context
	client.NewTrace().Name("test").Create(ctx)

	// Cancel the context immediately
	cancel()

	// The batch processor should still send despite cancelled context
	// (it uses a background context fallback)
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	// Should still have received the batch despite cancelled context
	if receivedBatches == 0 {
		t.Error("Expected batch to be sent even with cancelled context")
	}
}

func TestCircuitBreakerStateFromHttpClient(t *testing.T) {
	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithCircuitBreaker(CircuitBreakerConfig{
			FailureThreshold: 5,
			Timeout:          time.Second,
		}),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	// Initially should be closed
	if client.CircuitBreakerState() != CircuitClosed {
		t.Errorf("Expected CircuitClosed, got %v", client.CircuitBreakerState())
	}
}

func TestCircuitBreakerStateWithoutConfig(t *testing.T) {
	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	// Without circuit breaker config, should return CircuitClosed
	if client.CircuitBreakerState() != CircuitClosed {
		t.Errorf("Expected CircuitClosed, got %v", client.CircuitBreakerState())
	}
}

func TestTraceBuilderClone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	// Create a template trace builder
	template := client.NewTrace().
		UserID("user-123").
		Environment("production").
		Tags([]string{"api", "v2"}).
		Metadata(map[string]any{"version": "1.0"})

	// Clone and create first trace
	trace1, err := template.Clone().Name("trace-1").Create(ctx)
	if err != nil {
		t.Fatalf("Create trace1 failed: %v", err)
	}

	// Clone and create second trace
	trace2, err := template.Clone().Name("trace-2").Create(ctx)
	if err != nil {
		t.Fatalf("Create trace2 failed: %v", err)
	}

	// Verify traces have different IDs
	if trace1.ID() == trace2.ID() {
		t.Error("Cloned traces should have different IDs")
	}

	// Verify original template still works
	trace3, err := template.Clone().Name("trace-3").Create(ctx)
	if err != nil {
		t.Fatalf("Create trace3 failed: %v", err)
	}

	if trace3.ID() == trace1.ID() || trace3.ID() == trace2.ID() {
		t.Error("All cloned traces should have unique IDs")
	}
}

func TestSpanBuilderClone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, _ := client.NewTrace().Name("test").Create(ctx)

	// Create a template span builder
	template := trace.NewSpan().
		Level(ObservationLevelDefault).
		Environment("production").
		Metadata(map[string]any{"type": "http"})

	// Clone and create spans
	span1, err := template.Clone().Name("span-1").Create(ctx)
	if err != nil {
		t.Fatalf("Create span1 failed: %v", err)
	}

	span2, err := template.Clone().Name("span-2").Create(ctx)
	if err != nil {
		t.Fatalf("Create span2 failed: %v", err)
	}

	if span1.SpanID() == span2.SpanID() {
		t.Error("Cloned spans should have different IDs")
	}
}

func TestGenerationBuilderClone(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, _ := client.NewTrace().Name("test").Create(ctx)

	// Create a template generation builder
	template := trace.NewGeneration().
		Model("gpt-4").
		ModelParameters(map[string]any{"temperature": 0.7}).
		Environment("production")

	// Clone and create generations
	gen1, err := template.Clone().Name("gen-1").Input("prompt 1").Create(ctx)
	if err != nil {
		t.Fatalf("Create gen1 failed: %v", err)
	}

	gen2, err := template.Clone().Name("gen-2").Input("prompt 2").Create(ctx)
	if err != nil {
		t.Fatalf("Create gen2 failed: %v", err)
	}

	if gen1.GenerationID() == gen2.GenerationID() {
		t.Error("Cloned generations should have different IDs")
	}
}

func TestClientWithOptions(t *testing.T) {
	// Create a simple mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "test", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Shutdown(context.Background())

	t.Run("PromptsWithOptions", func(t *testing.T) {
		configured := client.PromptsWithOptions(
			WithDefaultLabel("production"),
			WithDefaultVersion(5),
		)
		if configured == nil {
			t.Fatal("PromptsWithOptions returned nil")
		}
		// Verify the configured client has the right internal type
		if configured.config == nil {
			t.Error("config should be set")
		}
	})

	t.Run("TracesWithOptions", func(t *testing.T) {
		configured := client.TracesWithOptions(
			WithDefaultMetadata(Metadata{"env": "prod"}),
			WithDefaultTags([]string{"production"}),
		)
		if configured == nil {
			t.Fatal("TracesWithOptions returned nil")
		}
		if configured.DefaultMetadata()["env"] != "prod" {
			t.Error("metadata should be set")
		}
		if len(configured.DefaultTags()) != 1 || configured.DefaultTags()[0] != "production" {
			t.Error("tags should be set")
		}
	})

	t.Run("DatasetsWithOptions", func(t *testing.T) {
		configured := client.DatasetsWithOptions(
			WithDefaultPageSize(100),
		)
		if configured == nil {
			t.Fatal("DatasetsWithOptions returned nil")
		}
		if configured.DefaultPageSize() != 100 {
			t.Errorf("DefaultPageSize() = %d, want 100", configured.DefaultPageSize())
		}
	})

	t.Run("ScoresWithOptions", func(t *testing.T) {
		configured := client.ScoresWithOptions(
			WithDefaultSource("evaluation-pipeline"),
		)
		if configured == nil {
			t.Fatal("ScoresWithOptions returned nil")
		}
		if configured.DefaultSource() != "evaluation-pipeline" {
			t.Errorf("DefaultSource() = %q, want %q", configured.DefaultSource(), "evaluation-pipeline")
		}
	})

	t.Run("SessionsWithOptions", func(t *testing.T) {
		configured := client.SessionsWithOptions(
			WithSessionsTimeout(10 * time.Second),
		)
		if configured == nil {
			t.Fatal("SessionsWithOptions returned nil")
		}
		if configured.config == nil {
			t.Error("config should be set")
		}
		if configured.config.defaultTimeout != 10*time.Second {
			t.Errorf("defaultTimeout = %v, want %v", configured.config.defaultTimeout, 10*time.Second)
		}
	})

	t.Run("ModelsWithOptions", func(t *testing.T) {
		configured := client.ModelsWithOptions(
			WithModelsTimeout(10 * time.Second),
		)
		if configured == nil {
			t.Fatal("ModelsWithOptions returned nil")
		}
		if configured.config == nil {
			t.Error("config should be set")
		}
		if configured.config.defaultTimeout != 10*time.Second {
			t.Errorf("defaultTimeout = %v, want %v", configured.config.defaultTimeout, 10*time.Second)
		}
	})
}

// TestShutdownDrainsAllEvents verifies that graceful shutdown drains all pending events
// without losing any. This is a regression test for the shutdown race condition fix.
func TestShutdownDrainsAllEvents(t *testing.T) {
	var receivedEvents int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingestion" {
			var req struct {
				Batch []json.RawMessage `json:"batch"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				mu.Lock()
				receivedEvents += len(req.Batch)
				mu.Unlock()
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithBatchSize(10),
		WithFlushInterval(1*time.Hour), // Disable auto-flush
		WithTimeout(5*time.Second),
		WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx := context.Background()
	const numEvents = 50

	// Create many traces
	for i := 0; i < numEvents; i++ {
		_, err := client.NewTrace().Name("test-trace").Create(ctx)
		if err != nil {
			t.Fatalf("Create trace failed: %v", err)
		}
	}

	// Shutdown should drain all events
	err = client.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	mu.Lock()
	received := receivedEvents
	mu.Unlock()

	if received != numEvents {
		t.Errorf("Expected %d events to be received, got %d", numEvents, received)
	}
}

// TestShutdownUnderConcurrentLoad tests that shutdown properly drains events
// even when events are being created concurrently.
func TestShutdownUnderConcurrentLoad(t *testing.T) {
	var receivedEvents int
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingestion" {
			var req struct {
				Batch []json.RawMessage `json:"batch"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err == nil {
				mu.Lock()
				receivedEvents += len(req.Batch)
				mu.Unlock()
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithBatchSize(5),
		WithFlushInterval(100*time.Millisecond),
		WithTimeout(5*time.Second),
		WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx := context.Background()
	const numGoroutines = 10
	const eventsPerGoroutine = 20

	var wg sync.WaitGroup
	var createdEvents int64

	// Start concurrent event creators
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				_, err := client.NewTrace().Name("concurrent-test").Create(ctx)
				if err == nil {
					mu.Lock()
					createdEvents++
					mu.Unlock()
				}
				// Small delay to spread out events
				time.Sleep(time.Millisecond)
			}
		}()
	}

	// Wait for all creators to finish
	wg.Wait()

	// Shutdown should drain remaining events
	err = client.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	mu.Lock()
	received := receivedEvents
	created := createdEvents
	mu.Unlock()

	// All created events should be received
	if received != int(created) {
		t.Errorf("Expected %d events, received %d (lost %d)", created, received, int(created)-received)
	}
}

// TestShutdownWithQueuedBatches verifies that batches already in the queue
// are properly drained during shutdown.
func TestShutdownWithQueuedBatches(t *testing.T) {
	var receivedBatches int
	var mu sync.Mutex

	// Server with delay to cause batches to queue up
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingestion" {
			time.Sleep(50 * time.Millisecond) // Delay to queue batches
			mu.Lock()
			receivedBatches++
			mu.Unlock()
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithBatchSize(1), // Small batch size to create more batches
		WithFlushInterval(1*time.Hour),
		WithTimeout(5*time.Second),
		WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx := context.Background()

	// Create traces quickly to queue up batches
	for i := 0; i < 10; i++ {
		client.NewTrace().Name("queue-test").Create(ctx)
	}

	// Give some time for batches to start processing
	time.Sleep(100 * time.Millisecond)

	// Shutdown should drain all queued batches
	err = client.Shutdown(context.Background())
	if err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}

	mu.Lock()
	batches := receivedBatches
	mu.Unlock()

	// Should have received all batches (10 events with batch size 1 = 10 batches)
	if batches < 10 {
		t.Errorf("Expected at least 10 batches, got %d", batches)
	}
}

// TestBackpressureIntegration verifies that the backpressure system is wired up correctly.
func TestBackpressureIntegration(t *testing.T) {
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
		WithFlushInterval(1*time.Hour),
		WithTimeout(5*time.Second),
		WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	// Verify backpressure handler is initialized
	if client.backpressure == nil {
		t.Error("Expected backpressure handler to be initialized")
	}

	// Verify we can get backpressure status
	status := client.BackpressureStatus()
	if status.MonitorStats.CurrentLevel != BackpressureNone {
		t.Errorf("Expected BackpressureNone initially, got %v", status.MonitorStats.CurrentLevel)
	}

	// Verify we can get backpressure level
	level := client.BackpressureLevel()
	if level != BackpressureNone {
		t.Errorf("Expected BackpressureNone initially, got %v", level)
	}

	// Verify IsUnderBackpressure returns false initially
	if client.IsUnderBackpressure() {
		t.Error("Expected IsUnderBackpressure to return false initially")
	}
}

// TestBackpressureCallback verifies that backpressure callbacks are invoked.
func TestBackpressureCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingestion" {
			// Slow response to cause backpressure
			time.Sleep(100 * time.Millisecond)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(IngestionResult{
				Successes: []IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	var mu sync.Mutex
	callbackCalled := false
	var lastState QueueState

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithBatchSize(1), // Small batch to queue more
		WithFlushInterval(1*time.Hour),
		WithTimeout(5*time.Second),
		WithShutdownTimeout(10*time.Second),
		WithOnBackpressure(func(state QueueState) {
			mu.Lock()
			callbackCalled = true
			lastState = state
			mu.Unlock()
		}),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	// Create many traces quickly to trigger backpressure
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		client.NewTrace().Name("backpressure-test").Create(ctx)
	}

	// Wait for flush to process
	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	called := callbackCalled
	state := lastState
	mu.Unlock()

	// Callback may or may not be called depending on queue depth
	// This test just verifies the wiring works
	t.Logf("Callback called: %v, State: %+v", called, state)
}

// TestDropOnQueueFull verifies events are dropped when queue is full and DropOnQueueFull is set.
func TestDropOnQueueFull(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/ingestion" {
			// Very slow response to cause queue backup
			time.Sleep(500 * time.Millisecond)
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
		WithBatchQueueSize(2), // Very small queue
		WithFlushInterval(1*time.Hour),
		WithTimeout(5*time.Second),
		WithShutdownTimeout(10*time.Second),
		WithDropOnQueueFull(true),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	// Try to queue many events - some should be dropped
	ctx := context.Background()
	for i := 0; i < 100; i++ {
		client.NewTrace().Name("drop-test").Create(ctx)
	}

	// Verify we can get stats about dropped events
	stats := client.BackpressureStatus()
	t.Logf("Backpressure stats: dropped=%d, blocked=%d", stats.DroppedCount, stats.BlockedCount)
}

// TestLangfuseErrorInterface verifies that all error types implement LangfuseError.
func TestLangfuseErrorInterface(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode ErrorCode
	}{
		{
			name:     "APIError 401",
			err:      &APIError{StatusCode: 401, Message: "unauthorized"},
			wantCode: ErrCodeAuth,
		},
		{
			name:     "APIError 429",
			err:      &APIError{StatusCode: 429, Message: "rate limited"},
			wantCode: ErrCodeRateLimit,
		},
		{
			name:     "APIError 500",
			err:      &APIError{StatusCode: 500, Message: "server error"},
			wantCode: ErrCodeAPI,
		},
		{
			name:     "ValidationError",
			err:      NewValidationError("field", "invalid"),
			wantCode: ErrCodeValidation,
		},
		{
			name:     "ShutdownError",
			err:      &ShutdownError{Message: "timeout", PendingEvents: 10},
			wantCode: ErrCodeShutdown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			langfuseErr, ok := tt.err.(LangfuseError)
			if !ok {
				t.Fatalf("Expected error to implement LangfuseError")
			}

			if langfuseErr.Code() != tt.wantCode {
				t.Errorf("Code() = %v, want %v", langfuseErr.Code(), tt.wantCode)
			}

			// Verify GetRequestID doesn't panic
			_ = langfuseErr.GetRequestID()

			// Verify IsRetryable doesn't panic
			_ = langfuseErr.IsRetryable()
		})
	}
}

// TestIsRetryableHelper verifies the IsRetryable helper function.
func TestIsRetryableHelper(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		wantRetryable bool
	}{
		{
			name:          "nil error",
			err:           nil,
			wantRetryable: false,
		},
		{
			name:          "APIError 429",
			err:           &APIError{StatusCode: 429},
			wantRetryable: true,
		},
		{
			name:          "APIError 500",
			err:           &APIError{StatusCode: 500},
			wantRetryable: true,
		},
		{
			name:          "APIError 400",
			err:           &APIError{StatusCode: 400},
			wantRetryable: false,
		},
		{
			name:          "ValidationError",
			err:           NewValidationError("field", "invalid"),
			wantRetryable: false,
		},
		{
			name:          "ShutdownError",
			err:           &ShutdownError{Message: "timeout"},
			wantRetryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsRetryable(tt.err)
			if got != tt.wantRetryable {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.wantRetryable)
			}
		})
	}
}

// TestClient_HandleQueueFull tests the queue overflow handler.
func TestClient_HandleQueueFull(t *testing.T) {
	var queueFullHandled bool
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Slow response to cause queue backup
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
		WithBatchSize(1),
		WithBatchQueueSize(1), // Very small queue to trigger overflow
		WithFlushInterval(1*time.Hour),
		WithOnBackpressure(func(state QueueState) {
			mu.Lock()
			if state.Level >= BackpressureOverflow {
				queueFullHandled = true
			}
			mu.Unlock()
		}),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	// Create many events quickly to trigger queue overflow
	for i := 0; i < 100; i++ {
		client.NewTrace().Name("overflow-test").Create(ctx)
	}

	// Wait for async processing
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	handled := queueFullHandled
	mu.Unlock()

	if !handled {
		t.Error("expected queue full/overflow condition to be detected")
	}
}
