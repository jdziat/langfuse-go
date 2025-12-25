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
		"pk-test",
		"sk-test",
		WithRegion(RegionUS),
		WithBatchSize(50),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	if client.config.PublicKey != "pk-test" {
		t.Errorf("PublicKey = %v, want pk-test", client.config.PublicKey)
	}
	if client.config.SecretKey != "sk-test" {
		t.Errorf("SecretKey = %v, want sk-test", client.config.SecretKey)
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
			secretKey: "sk-test",
			wantError: ErrMissingPublicKey,
		},
		{
			name:      "missing secret key",
			publicKey: "pk-test",
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
		"pk-test",
		"sk-test",
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
	client, err := New("pk-test", "sk-test")
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
		"pk-test",
		"sk-test",
		WithBaseURL(server.URL),
		WithBatchSize(100),
		WithFlushInterval(1*time.Hour), // Prevent auto flush
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	// Create a trace
	trace, err := client.NewTrace().Name("test-trace").Create()
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
		"pk-test",
		"sk-test",
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
	_, err = client.NewTrace().Create()
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
		"pk-test",
		"sk-test",
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
		"pk-test",
		"sk-test",
		WithBaseURL(server.URL),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	trace, err := client.NewTrace().
		ID("custom-id").
		Name("test-trace").
		UserID("user-123").
		SessionID("session-456").
		Input(map[string]string{"key": "value"}).
		Output("output").
		Metadata(map[string]interface{}{"meta": "data"}).
		Tags([]string{"tag1", "tag2"}).
		Release("v1.0.0").
		Version("1").
		Public(true).
		Environment("test").
		Create()

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
		"pk-test",
		"sk-test",
		WithBaseURL(server.URL),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	trace, err := client.NewTrace().Name("test").Create()
	if err != nil {
		t.Fatalf("Create trace failed: %v", err)
	}

	// Test creating a span
	span, err := trace.Span().Name("test-span").Create()
	if err != nil {
		t.Fatalf("Create span failed: %v", err)
	}
	if span.SpanID() == "" {
		t.Error("Span ID should not be empty")
	}

	// Test creating a generation
	gen, err := trace.Generation().Name("test-gen").Create()
	if err != nil {
		t.Fatalf("Create generation failed: %v", err)
	}
	if gen.GenerationID() == "" {
		t.Error("Generation ID should not be empty")
	}

	// Test creating an event
	err = trace.Event().Name("test-event").Create()
	if err != nil {
		t.Fatalf("Create event failed: %v", err)
	}

	// Test creating a score
	err = trace.Score().Name("test-score").NumericValue(0.9).Create()
	if err != nil {
		t.Fatalf("Create score failed: %v", err)
	}

	// Test updating trace
	err = trace.Update().Output("final output").Apply()
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
		"pk-test",
		"sk-test",
		WithBaseURL(server.URL),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	trace, err := client.NewTrace().Name("test").Create()
	if err != nil {
		t.Fatalf("Create trace failed: %v", err)
	}

	err = trace.Update().
		Name("updated-name").
		UserID("user-456").
		SessionID("session-789").
		Input("new input").
		Output("new output").
		Metadata(map[string]interface{}{"key": "value"}).
		Tags([]string{"new-tag"}).
		Public(true).
		Apply()

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
		"pk-test",
		"sk-test",
		WithBaseURL(server.URL),
		WithBatchSize(5),
		WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	// Create 6 events (should trigger flush at 5)
	for i := 0; i < 6; i++ {
		client.NewTrace().Name("test").Create()
	}

	// Wait for async flush
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()
	if flushCount < 1 {
		t.Errorf("Expected at least 1 flush, got %d", flushCount)
	}
}
