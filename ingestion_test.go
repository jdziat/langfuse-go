package langfuse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()

	if id1 == "" {
		t.Error("ID should not be empty")
	}

	if len(id1) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("ID length = %d, want 32", len(id1))
	}

	if id1 == id2 {
		t.Error("IDs should be unique")
	}
}

func TestSpanBuilder(t *testing.T) {
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

	now := time.Now()
	span, err := trace.Span().
		ID("custom-span-id").
		Name("test-span").
		StartTime(now).
		EndTime(now.Add(time.Second)).
		Input("span input").
		Output("span output").
		Metadata(map[string]interface{}{"key": "value"}).
		Level(ObservationLevelDefault).
		StatusMessage("success").
		ParentObservationID("parent-id").
		Version("1.0").
		Environment("test").
		Create()

	if err != nil {
		t.Fatalf("Create span failed: %v", err)
	}

	if span.SpanID() != "custom-span-id" {
		t.Errorf("SpanID = %v, want custom-span-id", span.SpanID())
	}
}

func TestSpanContextEnd(t *testing.T) {
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

	trace, _ := client.NewTrace().Name("test").Create()
	span, _ := trace.Span().Name("test-span").Create()

	err = span.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}
}

func TestSpanContextEndWithOutput(t *testing.T) {
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

	trace, _ := client.NewTrace().Name("test").Create()
	span, _ := trace.Span().Name("test-span").Create()

	err = span.EndWithOutput("final output")
	if err != nil {
		t.Fatalf("EndWithOutput failed: %v", err)
	}
}

func TestSpanContextChildObservations(t *testing.T) {
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

	trace, _ := client.NewTrace().Name("test").Create()
	parentSpan, _ := trace.Span().Name("parent").Create()

	// Create child span
	childSpan, err := parentSpan.Span().Name("child-span").Create()
	if err != nil {
		t.Fatalf("Create child span failed: %v", err)
	}
	if childSpan.SpanID() == "" {
		t.Error("Child span ID should not be empty")
	}

	// Create child generation
	gen, err := parentSpan.Generation().Name("child-gen").Create()
	if err != nil {
		t.Fatalf("Create child generation failed: %v", err)
	}
	if gen.GenerationID() == "" {
		t.Error("Child generation ID should not be empty")
	}

	// Create child event
	err = parentSpan.Event().Name("child-event").Create()
	if err != nil {
		t.Fatalf("Create child event failed: %v", err)
	}

	// Create score on span
	err = parentSpan.Score().Name("span-score").NumericValue(0.9).Create()
	if err != nil {
		t.Fatalf("Create span score failed: %v", err)
	}
}

func TestSpanUpdateBuilder(t *testing.T) {
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

	trace, _ := client.NewTrace().Name("test").Create()
	span, _ := trace.Span().Name("test-span").Create()

	err = span.Update().
		Name("updated-span").
		EndTime(time.Now()).
		Input("updated input").
		Output("updated output").
		Metadata(map[string]interface{}{"updated": true}).
		Level(ObservationLevelWarning).
		StatusMessage("updated status").
		Version("2.0").
		Apply()

	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
}

func TestGenerationBuilder(t *testing.T) {
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

	trace, _ := client.NewTrace().Name("test").Create()

	now := time.Now()
	gen, err := trace.Generation().
		ID("custom-gen-id").
		Name("test-generation").
		StartTime(now).
		EndTime(now.Add(time.Second)).
		CompletionStartTime(now.Add(100 * time.Millisecond)).
		Input("generation input").
		Output("generation output").
		Metadata(map[string]interface{}{"key": "value"}).
		Level(ObservationLevelDefault).
		StatusMessage("success").
		ParentObservationID("parent-id").
		Version("1.0").
		Model("gpt-4").
		ModelParameters(map[string]interface{}{"temperature": 0.7}).
		Usage(&Usage{Input: 100, Output: 50, Total: 150}).
		PromptName("my-prompt").
		PromptVersion(1).
		Environment("production").
		Create()

	if err != nil {
		t.Fatalf("Create generation failed: %v", err)
	}

	if gen.GenerationID() != "custom-gen-id" {
		t.Errorf("GenerationID = %v, want custom-gen-id", gen.GenerationID())
	}
}

func TestGenerationBuilderUsageTokens(t *testing.T) {
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

	trace, _ := client.NewTrace().Name("test").Create()

	gen, err := trace.Generation().
		Name("test-generation").
		UsageTokens(100, 50).
		Create()

	if err != nil {
		t.Fatalf("Create generation failed: %v", err)
	}

	if gen.GenerationID() == "" {
		t.Error("GenerationID should not be empty")
	}
}

func TestGenerationContextEnd(t *testing.T) {
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

	trace, _ := client.NewTrace().Name("test").Create()
	gen, _ := trace.Generation().Name("test-gen").Create()

	err = gen.End()
	if err != nil {
		t.Fatalf("End failed: %v", err)
	}
}

func TestGenerationContextEndWithOutput(t *testing.T) {
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

	trace, _ := client.NewTrace().Name("test").Create()
	gen, _ := trace.Generation().Name("test-gen").Create()

	err = gen.EndWithOutput("AI response")
	if err != nil {
		t.Fatalf("EndWithOutput failed: %v", err)
	}
}

func TestGenerationContextEndWithUsage(t *testing.T) {
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

	trace, _ := client.NewTrace().Name("test").Create()
	gen, _ := trace.Generation().Name("test-gen").Create()

	err = gen.EndWithUsage("AI response", 100, 50)
	if err != nil {
		t.Fatalf("EndWithUsage failed: %v", err)
	}
}

func TestGenerationContextScore(t *testing.T) {
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

	trace, _ := client.NewTrace().Name("test").Create()
	gen, _ := trace.Generation().Name("test-gen").Create()

	err = gen.Score().Name("quality").NumericValue(0.95).Create()
	if err != nil {
		t.Fatalf("Create score failed: %v", err)
	}
}

func TestGenerationUpdateBuilder(t *testing.T) {
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

	trace, _ := client.NewTrace().Name("test").Create()
	gen, _ := trace.Generation().Name("test-gen").Create()

	now := time.Now()
	err = gen.Update().
		Name("updated-gen").
		EndTime(now).
		CompletionStartTime(now.Add(-100 * time.Millisecond)).
		Input("updated input").
		Output("updated output").
		Metadata(map[string]interface{}{"updated": true}).
		Level(ObservationLevelDefault).
		StatusMessage("updated status").
		Model("gpt-4-turbo").
		ModelParameters(map[string]interface{}{"temperature": 0.5}).
		Usage(&Usage{Input: 200, Output: 100}).
		UsageTokens(200, 100).
		Apply()

	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
}

func TestEventBuilder(t *testing.T) {
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

	trace, _ := client.NewTrace().Name("test").Create()

	err = trace.Event().
		ID("custom-event-id").
		Name("test-event").
		StartTime(time.Now()).
		Input("event input").
		Output("event output").
		Metadata(map[string]interface{}{"key": "value"}).
		Level(ObservationLevelWarning).
		StatusMessage("warning occurred").
		ParentObservationID("parent-id").
		Version("1.0").
		Environment("test").
		Create()

	if err != nil {
		t.Fatalf("Create event failed: %v", err)
	}
}

func TestScoreBuilder(t *testing.T) {
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

	trace, _ := client.NewTrace().Name("test").Create()

	// Test numeric score
	err = trace.Score().
		ID("custom-score-id").
		Name("numeric-score").
		NumericValue(0.95).
		Comment("High quality").
		ConfigID("config-123").
		Environment("production").
		Metadata(map[string]interface{}{"key": "value"}).
		Create()

	if err != nil {
		t.Fatalf("Create numeric score failed: %v", err)
	}

	// Test categorical score
	err = trace.Score().
		Name("category-score").
		CategoricalValue("excellent").
		Create()

	if err != nil {
		t.Fatalf("Create categorical score failed: %v", err)
	}

	// Test boolean score (true)
	err = trace.Score().
		Name("boolean-score-true").
		BooleanValue(true).
		Create()

	if err != nil {
		t.Fatalf("Create boolean score (true) failed: %v", err)
	}

	// Test boolean score (false)
	err = trace.Score().
		Name("boolean-score-false").
		BooleanValue(false).
		Create()

	if err != nil {
		t.Fatalf("Create boolean score (false) failed: %v", err)
	}

	// Test generic value
	err = trace.Score().
		Name("generic-score").
		Value(42).
		ObservationID("obs-123").
		Create()

	if err != nil {
		t.Fatalf("Create generic score failed: %v", err)
	}
}

func TestScoreBuilderValidation(t *testing.T) {
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

	trace, _ := client.NewTrace().Name("test").Create()

	// Score without name should fail
	err = trace.Score().NumericValue(0.9).Create()
	if err == nil {
		t.Error("Expected validation error for missing name")
	}

	validErr, ok := err.(*ValidationError)
	if !ok {
		t.Errorf("Expected ValidationError, got %T", err)
	}
	if validErr.Field != "name" {
		t.Errorf("Expected field 'name', got %s", validErr.Field)
	}
}
