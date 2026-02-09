package langfuse_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
)

func setupHelpersTestServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.IngestionResult{
			Successes: []langfuse.IngestionSuccess{{ID: "test", Status: 200}},
		})
	}))
}

func setupHelpersTestClient(t *testing.T, serverURL string) *langfuse.Client {
	client, err := langfuse.New("pk-lf-test-key", "sk-lf-test-key",
		langfuse.WithBaseURL(serverURL),
		langfuse.WithFlushInterval(1*time.Hour),
	)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}
	return client
}

func TestTraceGeneration(t *testing.T) {
	server := setupHelpersTestServer(t)
	defer server.Close()

	client := setupHelpersTestClient(t, server.URL)
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	result, err := langfuse.TraceGeneration(ctx, client, langfuse.GenerationParams{
		Name:      "test-generation",
		Model:     "gpt-4",
		Input:     "Hello, world!",
		UserID:    "user-123",
		SessionID: "session-456",
		Tags:      []string{"test"},
	}, func() (langfuse.GenerationResult, error) {
		return langfuse.GenerationResult{
			Output: "Hello! How can I help you?",
			Usage:  langfuse.Usage{Input: 10, Output: 20},
		}, nil
	})

	if err != nil {
		t.Fatalf("TraceGeneration failed: %v", err)
	}

	if result.Output != "Hello! How can I help you?" {
		t.Errorf("Unexpected output: %v", result.Output)
	}
	if result.Usage.Input != 10 || result.Usage.Output != 20 {
		t.Errorf("Unexpected usage: %+v", result.Usage)
	}
}

func TestTraceGenerationWithError(t *testing.T) {
	server := setupHelpersTestServer(t)
	defer server.Close()

	client := setupHelpersTestClient(t, server.URL)
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	expectedErr := errors.New("LLM error")

	_, err := langfuse.TraceGeneration(ctx, client, langfuse.GenerationParams{
		Name:  "test-generation",
		Model: "gpt-4",
		Input: "Hello, world!",
	}, func() (langfuse.GenerationResult, error) {
		return langfuse.GenerationResult{}, expectedErr
	})

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestTraceSpan(t *testing.T) {
	server := setupHelpersTestServer(t)
	defer server.Close()

	client := setupHelpersTestClient(t, server.URL)
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	spanExecuted := false
	err = langfuse.TraceSpan(ctx, trace, "test-span", func(span *langfuse.SpanContext) error {
		spanExecuted = true
		// Create a child event
		span.NewEvent().Name("test-event").Create(ctx)
		return nil
	})

	if err != nil {
		t.Fatalf("TraceSpan failed: %v", err)
	}
	if !spanExecuted {
		t.Error("Span function was not executed")
	}
}

func TestTraceSpanWithError(t *testing.T) {
	server := setupHelpersTestServer(t)
	defer server.Close()

	client := setupHelpersTestClient(t, server.URL)
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	expectedErr := errors.New("span error")
	err = langfuse.TraceSpan(ctx, trace, "test-span", func(span *langfuse.SpanContext) error {
		return expectedErr
	})

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
}

func TestTraceFunc(t *testing.T) {
	server := setupHelpersTestServer(t)
	defer server.Close()

	client := setupHelpersTestClient(t, server.URL)
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	result, err := langfuse.TraceFunc(ctx, client, "test-func", func(trace *langfuse.TraceContext) (string, error) {
		// Create some child observations
		trace.NewEvent().Name("step-1").Create(ctx)
		return "success", nil
	})

	if err != nil {
		t.Fatalf("TraceFunc failed: %v", err)
	}
	if result != "success" {
		t.Errorf("Expected 'success', got %q", result)
	}
}

func TestTraceFuncWithError(t *testing.T) {
	server := setupHelpersTestServer(t)
	defer server.Close()

	client := setupHelpersTestClient(t, server.URL)
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	expectedErr := errors.New("function error")

	result, err := langfuse.TraceFunc(ctx, client, "test-func", func(trace *langfuse.TraceContext) (string, error) {
		return "", expectedErr
	})

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if result != "" {
		t.Errorf("Expected empty result, got %q", result)
	}
}

func TestWithGeneration(t *testing.T) {
	server := setupHelpersTestServer(t)
	defer server.Close()

	client := setupHelpersTestClient(t, server.URL)
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	gen, output, err := langfuse.WithGeneration(ctx, trace, "gpt-4", "Hello!", func() (any, langfuse.Usage, error) {
		return "Hello there!", langfuse.Usage{Input: 5, Output: 10}, nil
	})

	if err != nil {
		t.Fatalf("WithGeneration failed: %v", err)
	}
	if gen == nil {
		t.Error("Generation context should not be nil")
	}
	if output != "Hello there!" {
		t.Errorf("Expected 'Hello there!', got %v", output)
	}
}

func TestWithGenerationError(t *testing.T) {
	server := setupHelpersTestServer(t)
	defer server.Close()

	client := setupHelpersTestClient(t, server.URL)
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, err := client.NewTrace().Name("test-trace").Create(ctx)
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	expectedErr := errors.New("generation error")
	gen, output, err := langfuse.WithGeneration(ctx, trace, "gpt-4", "Hello!", func() (any, langfuse.Usage, error) {
		return nil, langfuse.Usage{}, expectedErr
	})

	if err != expectedErr {
		t.Errorf("Expected error %v, got %v", expectedErr, err)
	}
	if gen == nil {
		t.Error("Generation context should not be nil even on error")
	}
	if output != nil {
		t.Errorf("Expected nil output, got %v", output)
	}
}

// ============================================================================
// Metadata Tests
// ============================================================================

func TestNewMetadata(t *testing.T) {
	m := langfuse.NewMetadata()
	if m == nil {
		t.Error("NewMetadata() should not return nil")
	}
	if len(m) != 0 {
		t.Errorf("NewMetadata() should be empty, got %d items", len(m))
	}
}

func TestMetadataSetAndGet(t *testing.T) {
	m := langfuse.NewMetadata()
	m.Set("key", "value")

	v, ok := m.Get("key")
	if !ok {
		t.Error("Get() should return true for existing key")
	}
	if v != "value" {
		t.Errorf("Get() = %v, want 'value'", v)
	}

	_, ok = m.Get("missing")
	if ok {
		t.Error("Get() should return false for missing key")
	}
}

func TestMetadataGetString(t *testing.T) {
	m := langfuse.NewMetadata()
	m.Set("str", "hello")
	m.Set("num", 42)

	s, ok := m.GetString("str")
	if !ok {
		t.Error("GetString() should return true for string value")
	}
	if s != "hello" {
		t.Errorf("GetString() = %s, want 'hello'", s)
	}

	_, ok = m.GetString("num")
	if ok {
		t.Error("GetString() should return false for non-string value")
	}

	_, ok = m.GetString("missing")
	if ok {
		t.Error("GetString() should return false for missing key")
	}
}

func TestMetadataGetInt(t *testing.T) {
	m := langfuse.NewMetadata()
	m.Set("int", 42)
	m.Set("float", 3.14)
	m.Set("int64", int64(100))
	m.Set("str", "hello")

	// Test int value
	n, ok := m.GetInt("int")
	if !ok {
		t.Error("GetInt() should return true for int value")
	}
	if n != 42 {
		t.Errorf("GetInt() = %d, want 42", n)
	}

	// Test float64 value (common from JSON)
	n, ok = m.GetInt("float")
	if !ok {
		t.Error("GetInt() should return true for float64 value")
	}
	if n != 3 {
		t.Errorf("GetInt() = %d, want 3 (truncated)", n)
	}

	// Test int64 value
	n, ok = m.GetInt("int64")
	if !ok {
		t.Error("GetInt() should return true for int64 value")
	}
	if n != 100 {
		t.Errorf("GetInt() = %d, want 100", n)
	}

	_, ok = m.GetInt("str")
	if ok {
		t.Error("GetInt() should return false for string value")
	}

	_, ok = m.GetInt("missing")
	if ok {
		t.Error("GetInt() should return false for missing key")
	}
}

func TestMetadataGetFloat(t *testing.T) {
	m := langfuse.NewMetadata()
	m.Set("float", 3.14)
	m.Set("int", 42)
	m.Set("str", "hello")

	f, ok := m.GetFloat("float")
	if !ok {
		t.Error("GetFloat() should return true for float value")
	}
	if f != 3.14 {
		t.Errorf("GetFloat() = %f, want 3.14", f)
	}

	f, ok = m.GetFloat("int")
	if !ok {
		t.Error("GetFloat() should return true for int value")
	}
	if f != 42.0 {
		t.Errorf("GetFloat() = %f, want 42.0", f)
	}

	_, ok = m.GetFloat("str")
	if ok {
		t.Error("GetFloat() should return false for string value")
	}
}

func TestMetadataGetBool(t *testing.T) {
	m := langfuse.NewMetadata()
	m.Set("true", true)
	m.Set("false", false)
	m.Set("str", "yes")

	b, ok := m.GetBool("true")
	if !ok {
		t.Error("GetBool() should return true for bool value")
	}
	if !b {
		t.Error("GetBool() = false, want true")
	}

	b, ok = m.GetBool("false")
	if !ok {
		t.Error("GetBool() should return true for bool value")
	}
	if b {
		t.Error("GetBool() = true, want false")
	}

	_, ok = m.GetBool("str")
	if ok {
		t.Error("GetBool() should return false for string value")
	}
}

func TestMetadataHas(t *testing.T) {
	m := langfuse.NewMetadata()
	m.Set("key", "value")

	if !m.Has("key") {
		t.Error("Has() should return true for existing key")
	}
	if m.Has("missing") {
		t.Error("Has() should return false for missing key")
	}
}

func TestMetadataDelete(t *testing.T) {
	m := langfuse.NewMetadata()
	m.Set("key", "value")
	m.Delete("key")

	if m.Has("key") {
		t.Error("Delete() should remove the key")
	}
}

func TestMetadataMerge(t *testing.T) {
	m1 := langfuse.NewMetadata().Set("a", 1).Set("b", 2)
	m2 := langfuse.NewMetadata().Set("b", 20).Set("c", 3)

	m1.Merge(m2)

	if v, _ := m1.GetInt("a"); v != 1 {
		t.Errorf("a = %d, want 1", v)
	}
	if v, _ := m1.GetInt("b"); v != 20 {
		t.Errorf("b = %d, want 20 (overwritten)", v)
	}
	if v, _ := m1.GetInt("c"); v != 3 {
		t.Errorf("c = %d, want 3", v)
	}
}

func TestMetadataClone(t *testing.T) {
	m := langfuse.NewMetadata().Set("key", "value")
	clone := m.Clone()

	// Verify clone has same value
	if v, _ := clone.GetString("key"); v != "value" {
		t.Errorf("Clone should have same value, got %s", v)
	}

	// Verify modifying clone doesn't affect original
	clone.Set("key", "modified")
	if v, _ := m.GetString("key"); v != "value" {
		t.Errorf("Original should be unchanged, got %s", v)
	}
}

func TestMetadataKeys(t *testing.T) {
	m := langfuse.NewMetadata().Set("a", 1).Set("b", 2).Set("c", 3)
	keys := m.Keys()

	if len(keys) != 3 {
		t.Errorf("Keys() length = %d, want 3", len(keys))
	}

	// Check all keys are present (order not guaranteed)
	keySet := make(map[string]bool)
	for _, k := range keys {
		keySet[k] = true
	}
	if !keySet["a"] || !keySet["b"] || !keySet["c"] {
		t.Errorf("Keys() missing expected keys: %v", keys)
	}
}

func TestMetadataLen(t *testing.T) {
	m := langfuse.NewMetadata()
	if m.Len() != 0 {
		t.Errorf("Len() = %d, want 0", m.Len())
	}

	m.Set("a", 1).Set("b", 2)
	if m.Len() != 2 {
		t.Errorf("Len() = %d, want 2", m.Len())
	}
}

func TestMetadataIsEmpty(t *testing.T) {
	m := langfuse.NewMetadata()
	if !m.IsEmpty() {
		t.Error("IsEmpty() should return true for empty metadata")
	}

	m.Set("key", "value")
	if m.IsEmpty() {
		t.Error("IsEmpty() should return false for non-empty metadata")
	}
}

func TestMetadataChaining(t *testing.T) {
	m := langfuse.NewMetadata().
		Set("name", "test").
		Set("count", 42).
		Set("enabled", true)

	if m.Len() != 3 {
		t.Errorf("Chained Set() should add all items, got %d", m.Len())
	}

	m.Delete("count")
	if m.Len() != 2 {
		t.Errorf("Chained Delete() should remove item, got %d", m.Len())
	}
}

func TestMetadataFilter(t *testing.T) {
	m := langfuse.NewMetadata().
		Set("name", "test").
		Set("count", 42).
		Set("enabled", true).
		Set("extra", "data")

	// Filter to keep only specific keys
	filtered := m.Filter("name", "count")
	if filtered.Len() != 2 {
		t.Errorf("Filter() should return 2 items, got %d", filtered.Len())
	}

	// Check that correct keys are present
	if name, ok := filtered.GetString("name"); !ok || name != "test" {
		t.Errorf("Filter() should include 'name', got %v", name)
	}
	if count, ok := filtered.GetInt("count"); !ok || count != 42 {
		t.Errorf("Filter() should include 'count', got %v", count)
	}

	// Check that filtered keys are not present
	if _, ok := filtered.Get("enabled"); ok {
		t.Error("Filter() should not include 'enabled'")
	}
	if _, ok := filtered.Get("extra"); ok {
		t.Error("Filter() should not include 'extra'")
	}

	// Original metadata should be unchanged
	if m.Len() != 4 {
		t.Errorf("Filter() should not modify original, got %d items", m.Len())
	}
}

func TestMetadataFilterNonExistentKeys(t *testing.T) {
	m := langfuse.NewMetadata().Set("existing", "value")

	// Filter with non-existent keys
	filtered := m.Filter("existing", "missing", "also-missing")

	// Should only include the existing key
	if filtered.Len() != 1 {
		t.Errorf("Filter() with missing keys should return 1 item, got %d", filtered.Len())
	}

	if v, ok := filtered.GetString("existing"); !ok || v != "value" {
		t.Errorf("Filter() should include 'existing', got %v", v)
	}
}

func TestMetadataFilterEmpty(t *testing.T) {
	m := langfuse.NewMetadata().Set("key", "value")

	// Filter with no keys
	filtered := m.Filter()

	if !filtered.IsEmpty() {
		t.Errorf("Filter() with no keys should return empty metadata, got %d items", filtered.Len())
	}
}
