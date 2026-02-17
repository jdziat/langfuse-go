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

func TestMetadataBuilder(t *testing.T) {
	t.Run("builds metadata with various types", func(t *testing.T) {
		now := time.Now()
		metadata := langfuse.BuildMetadata().
			String("user_id", "123").
			Int("count", 5).
			Int64("big_count", 9223372036854775807).
			Float("score", 0.95).
			Bool("premium", true).
			Time("timestamp", now).
			Duration("elapsed", 5*time.Second).
			DurationMs("latency", 100*time.Millisecond).
			Strings("tags", []string{"a", "b"}).
			JSON("nested", map[string]any{"key": "value"}).
			Build()

		if metadata["user_id"] != "123" {
			t.Errorf("user_id = %v, want 123", metadata["user_id"])
		}
		if metadata["count"] != 5 {
			t.Errorf("count = %v, want 5", metadata["count"])
		}
		if metadata["big_count"] != int64(9223372036854775807) {
			t.Errorf("big_count = %v, want 9223372036854775807", metadata["big_count"])
		}
		if metadata["score"] != 0.95 {
			t.Errorf("score = %v, want 0.95", metadata["score"])
		}
		if metadata["premium"] != true {
			t.Errorf("premium = %v, want true", metadata["premium"])
		}
		if metadata["latency"] != int64(100) {
			t.Errorf("latency = %v, want 100", metadata["latency"])
		}
	})

	t.Run("merge overwrites existing keys", func(t *testing.T) {
		metadata := langfuse.BuildMetadata().
			String("key", "original").
			Merge(langfuse.Metadata{"key": "overwritten"}).
			Build()

		if metadata["key"] != "overwritten" {
			t.Errorf("key = %v, want overwritten", metadata["key"])
		}
	})

	t.Run("map adds nested map", func(t *testing.T) {
		metadata := langfuse.BuildMetadata().
			Map("nested", map[string]any{"a": 1, "b": 2}).
			Build()

		nested, ok := metadata["nested"].(map[string]any)
		if !ok {
			t.Fatalf("nested is not a map")
		}
		if nested["a"] != 1 {
			t.Errorf("nested.a = %v, want 1", nested["a"])
		}
	})
}

func TestTagsBuilder(t *testing.T) {
	t.Run("builds tags", func(t *testing.T) {
		tags := langfuse.NewTags().
			Add("production", "api").
			Add("v2").
			Build()

		if len(tags) != 3 {
			t.Errorf("len(tags) = %d, want 3", len(tags))
		}
		if tags[0] != "production" {
			t.Errorf("tags[0] = %s, want production", tags[0])
		}
	})

	t.Run("conditional add", func(t *testing.T) {
		tags := langfuse.NewTags().
			Add("always").
			AddIf(true, "included").
			AddIf(false, "excluded").
			Build()

		if len(tags) != 2 {
			t.Errorf("len(tags) = %d, want 2", len(tags))
		}
	})

	t.Run("add if not empty", func(t *testing.T) {
		tags := langfuse.NewTags().
			AddIfNotEmpty("included").
			AddIfNotEmpty("").
			Build()

		if len(tags) != 1 {
			t.Errorf("len(tags) = %d, want 1", len(tags))
		}
	})

	t.Run("environment and version", func(t *testing.T) {
		tags := langfuse.NewTags().
			Environment("production").
			Version("1.2.3").
			Build()

		if tags[0] != "env:production" {
			t.Errorf("tags[0] = %s, want env:production", tags[0])
		}
		if tags[1] != "version:1.2.3" {
			t.Errorf("tags[1] = %s, want version:1.2.3", tags[1])
		}
	})
}

func TestUsageBuilder(t *testing.T) {
	t.Run("builds usage with tokens", func(t *testing.T) {
		usage := langfuse.NewUsage().
			Input(100).
			Output(50).
			Build()

		if usage.Input != 100 {
			t.Errorf("Input = %d, want 100", usage.Input)
		}
		if usage.Output != 50 {
			t.Errorf("Output = %d, want 50", usage.Output)
		}
		if usage.Total != 150 {
			t.Errorf("Total = %d, want 150", usage.Total)
		}
	})

	t.Run("builds usage with costs", func(t *testing.T) {
		usage := langfuse.NewUsage().
			Tokens(100, 50).
			InputCost(0.001).
			OutputCost(0.002).
			Unit("TOKENS").
			Build()

		if usage.TotalCost != 0.003 {
			t.Errorf("TotalCost = %f, want 0.003", usage.TotalCost)
		}
		if usage.Unit != "TOKENS" {
			t.Errorf("Unit = %s, want TOKENS", usage.Unit)
		}
	})

	t.Run("explicit total overrides calculated", func(t *testing.T) {
		usage := langfuse.NewUsage().
			Input(100).
			Output(50).
			Total(200). // Override
			Build()

		if usage.Total != 200 {
			t.Errorf("Total = %d, want 200", usage.Total)
		}
	})
}

func TestModelParametersBuilder(t *testing.T) {
	t.Run("builds common parameters", func(t *testing.T) {
		params := langfuse.NewModelParameters().
			Temperature(0.7).
			MaxTokens(150).
			TopP(0.9).
			TopK(40).
			FrequencyPenalty(0.5).
			PresencePenalty(0.5).
			Stop("END", "STOP").
			Seed(42).
			Build()

		if params["temperature"] != 0.7 {
			t.Errorf("temperature = %v, want 0.7", params["temperature"])
		}
		if params["max_tokens"] != 150 {
			t.Errorf("max_tokens = %v, want 150", params["max_tokens"])
		}
		if params["seed"] != 42 {
			t.Errorf("seed = %v, want 42", params["seed"])
		}
	})

	t.Run("response format", func(t *testing.T) {
		params := langfuse.NewModelParameters().
			ResponseFormat("json_object").
			Build()

		format, ok := params["response_format"].(map[string]string)
		if !ok {
			t.Fatalf("response_format is not a map")
		}
		if format["type"] != "json_object" {
			t.Errorf("response_format.type = %s, want json_object", format["type"])
		}
	})

	t.Run("custom parameters", func(t *testing.T) {
		params := langfuse.NewModelParameters().
			Set("custom_param", "custom_value").
			Merge(langfuse.Metadata{"another": 123}).
			Build()

		if params["custom_param"] != "custom_value" {
			t.Errorf("custom_param = %v, want custom_value", params["custom_param"])
		}
		if params["another"] != 123 {
			t.Errorf("another = %v, want 123", params["another"])
		}
	})
}

func TestEndResult(t *testing.T) {
	t.Run("Ok returns true when no error", func(t *testing.T) {
		result := langfuse.EndResult{Error: nil}
		if !result.Ok() {
			t.Error("Ok() should return true when Error is nil")
		}
	})

	t.Run("Ok returns false when error", func(t *testing.T) {
		result := langfuse.EndResult{Error: errors.New("test error")}
		if result.Ok() {
			t.Error("Ok() should return false when Error is not nil")
		}
	})
}

// NOTE: TestEndOptions was removed because it tests the unexported endConfig type
// which is not accessible from external test packages.

func TestSpanEndWith(t *testing.T) {
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
		langfuse.WithTimeout(5*time.Second),
		langfuse.WithShutdownTimeout(10*time.Second),
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

	span, err := trace.NewSpan().Name("test-span").Create(ctx)
	if err != nil {
		t.Fatalf("Create span failed: %v", err)
	}

	t.Run("EndWith with output", func(t *testing.T) {
		result := span.EndWith(ctx, langfuse.WithOutput("result"))
		if !result.Ok() {
			t.Errorf("EndWith failed: %v", result.Error)
		}
	})

	t.Run("EndWith with error", func(t *testing.T) {
		span2, _ := trace.NewSpan().Name("error-span").Create(ctx)
		result := span2.EndWith(ctx, langfuse.WithError(errors.New("something failed")))
		if !result.Ok() {
			t.Errorf("EndWith failed: %v", result.Error)
		}
	})

	t.Run("EndWith with multiple options", func(t *testing.T) {
		span3, _ := trace.NewSpan().Name("multi-span").Create(ctx)
		result := span3.EndWith(ctx,
			langfuse.WithOutput("final result"),
			langfuse.WithEndMetadata(langfuse.Metadata{"key": "value"}),
			langfuse.WithEndLevel(langfuse.ObservationLevelWarning),
			langfuse.WithStatusMessage("completed with warnings"),
		)
		if !result.Ok() {
			t.Errorf("EndWith failed: %v", result.Error)
		}
	})
}

func TestGenerationEndWith(t *testing.T) {
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
		langfuse.WithTimeout(5*time.Second),
		langfuse.WithShutdownTimeout(10*time.Second),
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

	t.Run("EndWith with output and usage", func(t *testing.T) {
		gen, _ := trace.NewGeneration().Name("llm-call").Model("gpt-4").Create(ctx)
		result := gen.EndWith(ctx,
			langfuse.WithOutput("Generated response"),
			langfuse.WithUsage(100, 50),
		)
		if !result.Ok() {
			t.Errorf("EndWith failed: %v", result.Error)
		}
	})

	t.Run("EndWith with completion start time", func(t *testing.T) {
		gen, _ := trace.NewGeneration().Name("streaming-call").Model("gpt-4").Create(ctx)
		completionStart := time.Now().Add(-100 * time.Millisecond)
		result := gen.EndWith(ctx,
			langfuse.WithOutput("Streamed response"),
			langfuse.WithUsage(100, 50),
			langfuse.WithCompletionStart(completionStart),
		)
		if !result.Ok() {
			t.Errorf("EndWith failed: %v", result.Error)
		}
	})
}

func TestBatchTraceBuilder(t *testing.T) {
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
		langfuse.WithTimeout(5*time.Second),
		langfuse.WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	t.Run("creates multiple traces", func(t *testing.T) {
		batch := client.BatchTraces()
		batch.Add("trace-1").UserID("user-1").Tags([]string{"api"})
		batch.Add("trace-2").UserID("user-2").Tags([]string{"api"})
		batch.Add("trace-3").UserID("user-3").Tags([]string{"api"})

		if batch.Len() != 3 {
			t.Errorf("Len() = %d, want 3", batch.Len())
		}

		traces, err := batch.Create(ctx)
		if err != nil {
			t.Errorf("Create failed: %v", err)
		}
		if len(traces) != 3 {
			t.Errorf("len(traces) = %d, want 3", len(traces))
		}
		for i, tc := range traces {
			if tc == nil {
				t.Errorf("traces[%d] is nil", i)
			}
		}
	})

	t.Run("empty batch returns nil", func(t *testing.T) {
		batch := client.BatchTraces()
		traces, err := batch.Create(ctx)
		if err != nil {
			t.Errorf("Create failed: %v", err)
		}
		if traces != nil {
			t.Error("expected nil traces for empty batch")
		}
	})

	t.Run("AddBuilder adds existing builder", func(t *testing.T) {
		tb := client.NewTrace().Name("pre-built").UserID("user-x")
		batch := client.BatchTraces().AddBuilder(tb)

		if batch.Len() != 1 {
			t.Errorf("Len() = %d, want 1", batch.Len())
		}
	})
}

func TestBatchSpanBuilder(t *testing.T) {
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
		langfuse.WithTimeout(5*time.Second),
		langfuse.WithShutdownTimeout(10*time.Second),
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

	t.Run("creates multiple spans", func(t *testing.T) {
		batch := trace.BatchSpans()
		batch.Add("step-1").Input("input-1")
		batch.Add("step-2").Input("input-2")
		batch.Add("step-3").Input("input-3")

		if batch.Len() != 3 {
			t.Errorf("Len() = %d, want 3", batch.Len())
		}

		spans, err := batch.Create(ctx)
		if err != nil {
			t.Errorf("Create failed: %v", err)
		}
		if len(spans) != 3 {
			t.Errorf("len(spans) = %d, want 3", len(spans))
		}
	})
}

func TestBatchError(t *testing.T) {
	t.Run("Error message format", func(t *testing.T) {
		batchErr := &langfuse.BatchError{
			Total:     5,
			Succeeded: 3,
			Errors:    map[int]error{1: errors.New("e1"), 3: errors.New("e2")},
		}

		msg := batchErr.Error()
		if msg != "batch operation: 3/5 succeeded, 2 failed" {
			t.Errorf("Error() = %s, unexpected format", msg)
		}
	})

	t.Run("FirstError returns first error", func(t *testing.T) {
		e1 := errors.New("first")
		e2 := errors.New("second")
		batchErr := &langfuse.BatchError{
			Total:     3,
			Succeeded: 1,
			Errors:    map[int]error{0: e1, 2: e2},
		}

		if batchErr.FirstError() != e1 {
			t.Error("FirstError should return the error at index 0")
		}
	})

	t.Run("IsBatchError helper", func(t *testing.T) {
		batchErr := &langfuse.BatchError{Total: 1, Succeeded: 0, Errors: map[int]error{}}
		if !langfuse.IsBatchError(batchErr) {
			t.Error("IsBatchError should return true for BatchError")
		}
		if langfuse.IsBatchError(errors.New("regular error")) {
			t.Error("IsBatchError should return false for regular error")
		}
	})

	t.Run("AsBatchError helper", func(t *testing.T) {
		batchErr := &langfuse.BatchError{Total: 1, Succeeded: 0, Errors: map[int]error{}}
		result := langfuse.AsBatchError(batchErr)
		if result != batchErr {
			t.Error("AsBatchError should return the BatchError")
		}
		if langfuse.AsBatchError(errors.New("regular error")) != nil {
			t.Error("AsBatchError should return nil for regular error")
		}
	})
}

func TestBuildersIntegration(t *testing.T) {
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
		langfuse.WithTimeout(5*time.Second),
		langfuse.WithShutdownTimeout(10*time.Second),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	t.Run("full workflow with builders", func(t *testing.T) {
		// Create trace with type-safe metadata and tags
		trace, err := client.NewTrace().
			Name("api-request").
			Metadata(langfuse.BuildMetadata().
				String("user_id", "user-123").
				String("endpoint", "/api/chat").
				Int("request_size", 1024).
				Build()).
			Tags(langfuse.NewTags().
				Environment("production").
				Version("1.0.0").
				Add("api").
				Build()).
			Create(ctx)
		if err != nil {
			t.Fatalf("Create trace failed: %v", err)
		}

		// Create generation with model parameters builder
		gen, err := trace.NewGeneration().
			Name("llm-completion").
			Model("gpt-4").
			ModelParameters(langfuse.NewModelParameters().
				Temperature(0.7).
				MaxTokens(150).
				TopP(0.9).
				Build()).
			Input("What is Go?").
			Create(ctx)
		if err != nil {
			t.Fatalf("Create generation failed: %v", err)
		}

		// End generation with usage builder
		result := gen.EndWith(ctx,
			langfuse.WithOutput("Go is a programming language..."),
			langfuse.WithUsage(10, 25),
			langfuse.WithEndMetadata(langfuse.BuildMetadata().
				DurationMs("latency", 500*time.Millisecond).
				Bool("cached", false).
				Build()),
		)
		if !result.Ok() {
			t.Errorf("EndWith failed: %v", result.Error)
		}
	})
}
