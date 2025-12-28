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

	// UUID v4 format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx (36 chars)
	if len(id1) != 36 {
		t.Errorf("ID length = %d, want 36", len(id1))
	}

	if id1 == id2 {
		t.Error("IDs should be unique")
	}
}

func TestUUID(t *testing.T) {
	uuid, err := UUID()
	if err != nil {
		t.Fatalf("UUID() error: %v", err)
	}

	// UUID v4 format: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if len(uuid) != 36 {
		t.Errorf("UUID length = %d, want 36", len(uuid))
	}

	// Verify format with dashes at correct positions
	if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
		t.Errorf("UUID format incorrect: %s", uuid)
	}

	// Verify version (4) at position 14
	if uuid[14] != '4' {
		t.Errorf("UUID version = %c, want 4", uuid[14])
	}

	// Verify variant (8, 9, a, or b) at position 19
	variant := uuid[19]
	if variant != '8' && variant != '9' && variant != 'a' && variant != 'b' {
		t.Errorf("UUID variant = %c, want 8, 9, a, or b", variant)
	}

	// Generate another and verify uniqueness
	uuid2, _ := UUID()
	if uuid == uuid2 {
		t.Error("UUIDs should be unique")
	}
}

func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Valid standard UUIDs
		{"valid standard UUID lowercase", "550e8400-e29b-41d4-a716-446655440000", true},
		{"valid standard UUID uppercase", "550E8400-E29B-41D4-A716-446655440000", true},
		{"valid standard UUID mixed case", "550e8400-E29B-41d4-A716-446655440000", true},
		{"generated UUID", generateID(), true},

		// Valid compact UUIDs (no hyphens)
		{"valid compact UUID lowercase", "550e8400e29b41d4a716446655440000", true},
		{"valid compact UUID uppercase", "550E8400E29B41D4A716446655440000", true},

		// Invalid UUIDs
		{"empty string", "", false},
		{"too short", "550e8400-e29b-41d4", false},
		{"too long", "550e8400-e29b-41d4-a716-4466554400001234", false},
		{"wrong hyphen positions", "550e-8400-e29b-41d4-a716446655440000", false},
		{"missing hyphens but wrong length", "550e8400e29b41d4a71644665544000", false},
		{"non-hex characters", "550e8400-e29b-41d4-a716-44665544zzzz", false},
		{"spaces", "550e8400 e29b 41d4 a716 446655440000", false},
		{"special characters", "550e8400-e29b-41d4-a716-44665544@@@@", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidUUID(tt.input)
			if got != tt.want {
				t.Errorf("IsValidUUID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsHexString(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"0123456789", true},
		{"abcdef", true},
		{"ABCDEF", true},
		{"0123456789abcdefABCDEF", true},
		{"", true}, // Empty string is valid hex
		{"xyz", false},
		{"12g4", false},
		{"12 34", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isHexString(tt.input)
			if got != tt.want {
				t.Errorf("isHexString(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
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

	now := time.Now()
	span, err := trace.NewSpan().
		ID("custom-span-id").
		Name("test-span").
		StartTime(now).
		EndTime(now.Add(time.Second)).
		Input("span input").
		Output("span output").
		Metadata(map[string]any{"key": "value"}).
		Level(ObservationLevelDefault).
		StatusMessage("success").
		ParentObservationID("parent-id").
		Version("1.0").
		Environment("test").
		Create(ctx)

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
	span, _ := trace.NewSpan().Name("test-span").Create(ctx)

	err = span.End(ctx)
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
	span, _ := trace.NewSpan().Name("test-span").Create(ctx)

	err = span.EndWithOutput(ctx, "final output")
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
	parentSpan, _ := trace.NewSpan().Name("parent").Create(ctx)

	// Create child span
	childSpan, err := parentSpan.NewSpan().Name("child-span").Create(ctx)
	if err != nil {
		t.Fatalf("Create child span failed: %v", err)
	}
	if childSpan.SpanID() == "" {
		t.Error("Child span ID should not be empty")
	}

	// Create child generation
	gen, err := parentSpan.NewGeneration().Name("child-gen").Create(ctx)
	if err != nil {
		t.Fatalf("Create child generation failed: %v", err)
	}
	if gen.GenerationID() == "" {
		t.Error("Child generation ID should not be empty")
	}

	// Create child event
	err = parentSpan.NewEvent().Name("child-event").Create(ctx)
	if err != nil {
		t.Fatalf("Create child event failed: %v", err)
	}

	// Create score on span
	err = parentSpan.NewScore().Name("span-score").NumericValue(0.9).Create(ctx)
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
	span, _ := trace.NewSpan().Name("test-span").Create(ctx)

	err = span.Update().
		Name("updated-span").
		EndTime(time.Now()).
		Input("updated input").
		Output("updated output").
		Metadata(map[string]any{"updated": true}).
		Level(ObservationLevelWarning).
		StatusMessage("updated status").
		Version("2.0").
		Apply(ctx)

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

	now := time.Now()
	gen, err := trace.NewGeneration().
		ID("custom-gen-id").
		Name("test-generation").
		StartTime(now).
		EndTime(now.Add(time.Second)).
		CompletionStartTime(now.Add(100 * time.Millisecond)).
		Input("generation input").
		Output("generation output").
		Metadata(map[string]any{"key": "value"}).
		Level(ObservationLevelDefault).
		StatusMessage("success").
		ParentObservationID("parent-id").
		Version("1.0").
		Model("gpt-4").
		ModelParameters(map[string]any{"temperature": 0.7}).
		Usage(&Usage{Input: 100, Output: 50, Total: 150}).
		PromptName("my-prompt").
		PromptVersion(1).
		Environment("production").
		Create(ctx)

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

	gen, err := trace.NewGeneration().
		Name("test-generation").
		UsageTokens(100, 50).
		Create(ctx)

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
	gen, _ := trace.NewGeneration().Name("test-gen").Create(ctx)

	err = gen.End(ctx)
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
	gen, _ := trace.NewGeneration().Name("test-gen").Create(ctx)

	err = gen.EndWithOutput(ctx, "AI response")
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
	gen, _ := trace.NewGeneration().Name("test-gen").Create(ctx)

	err = gen.EndWithUsage(ctx, "AI response", 100, 50)
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
	gen, _ := trace.NewGeneration().Name("test-gen").Create(ctx)

	err = gen.NewScore().Name("quality").NumericValue(0.95).Create(ctx)
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
	gen, _ := trace.NewGeneration().Name("test-gen").Create(ctx)

	now := time.Now()
	err = gen.Update().
		Name("updated-gen").
		EndTime(now).
		CompletionStartTime(now.Add(-100*time.Millisecond)).
		Input("updated input").
		Output("updated output").
		Metadata(map[string]any{"updated": true}).
		Level(ObservationLevelDefault).
		StatusMessage("updated status").
		Model("gpt-4-turbo").
		ModelParameters(map[string]any{"temperature": 0.5}).
		Usage(&Usage{Input: 200, Output: 100}).
		UsageTokens(200, 100).
		Apply(ctx)

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

	err = trace.NewEvent().
		ID("custom-event-id").
		Name("test-event").
		StartTime(time.Now()).
		Input("event input").
		Output("event output").
		Metadata(map[string]any{"key": "value"}).
		Level(ObservationLevelWarning).
		StatusMessage("warning occurred").
		ParentObservationID("parent-id").
		Version("1.0").
		Environment("test").
		Create(ctx)

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

	// Test numeric score
	err = trace.NewScore().
		ID("custom-score-id").
		Name("numeric-score").
		NumericValue(0.95).
		Comment("High quality").
		ConfigID("config-123").
		Environment("production").
		Metadata(map[string]any{"key": "value"}).
		Create(ctx)

	if err != nil {
		t.Fatalf("Create numeric score failed: %v", err)
	}

	// Test categorical score
	err = trace.NewScore().
		Name("category-score").
		CategoricalValue("excellent").
		Create(ctx)

	if err != nil {
		t.Fatalf("Create categorical score failed: %v", err)
	}

	// Test boolean score (true)
	err = trace.NewScore().
		Name("boolean-score-true").
		BooleanValue(true).
		Create(ctx)

	if err != nil {
		t.Fatalf("Create boolean score (true) failed: %v", err)
	}

	// Test boolean score (false)
	err = trace.NewScore().
		Name("boolean-score-false").
		BooleanValue(false).
		Create(ctx)

	if err != nil {
		t.Fatalf("Create boolean score (false) failed: %v", err)
	}

	// Test generic value
	err = trace.NewScore().
		Name("generic-score").
		Value(42).
		ObservationID("obs-123").
		Create(ctx)

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

	// Score without name should fail
	err = trace.NewScore().NumericValue(0.9).Create(ctx)
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
