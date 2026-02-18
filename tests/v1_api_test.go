package langfuse_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
)

func TestNewClientV1(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	t.Run("creates client with valid credentials", func(t *testing.T) {
		client := langfuse.NewClient("pk-lf-test-key", "sk-lf-test-key",
			langfuse.WithBaseURL(server.URL),
			langfuse.WithTimeout(5*time.Second),
			langfuse.WithShutdownTimeout(10*time.Second),
		)
		if client == nil {
			t.Fatal("NewClient returned nil")
		}
		defer client.Shutdown(context.Background())

		if !client.IsActive() {
			t.Error("client should be active")
		}
	})

	t.Run("panics with empty credentials", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("NewClient should panic with empty credentials")
			}
		}()
		_ = langfuse.NewClient("", "", langfuse.WithBaseURL(server.URL))
	})

	t.Run("TryClient returns nil with empty credentials", func(t *testing.T) {
		client := langfuse.TryClient("", "", langfuse.WithBaseURL(server.URL))
		if client != nil {
			t.Error("TryClient should return nil with empty credentials")
			client.Shutdown(context.Background())
		}
	})
}

func TestV1TraceCreation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client := langfuse.NewClient("pk-lf-test-key", "sk-lf-test-key",
		langfuse.WithBaseURL(server.URL),
		langfuse.WithTimeout(5*time.Second),
		langfuse.WithShutdownTimeout(10*time.Second),
	)
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	t.Run("Trace creates trace with options (simple API)", func(t *testing.T) {
		trace, err := client.Trace(ctx, "test-trace",
			langfuse.WithUserID("user-123"),
			langfuse.WithTags("api", "v1"),
		)
		if err != nil {
			t.Fatalf("Trace failed: %v", err)
		}
		if trace == nil {
			t.Fatal("trace should not be nil")
		}
		if trace.ID() == "" {
			t.Error("trace ID should not be empty")
		}
	})

	t.Run("TraceV1 creates trace with options", func(t *testing.T) {
		trace, err := client.TraceV1(ctx, "test-trace-v1",
			langfuse.WithUserID("user-456"),
		)
		if err != nil {
			t.Fatalf("TraceV1 failed: %v", err)
		}
		if trace == nil {
			t.Fatal("trace should not be nil")
		}
	})
}

func TestV1SpanCreation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client := langfuse.NewClient("pk-lf-test-key", "sk-lf-test-key",
		langfuse.WithBaseURL(server.URL),
		langfuse.WithTimeout(5*time.Second),
		langfuse.WithShutdownTimeout(10*time.Second),
	)
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, _ := client.Trace(ctx, "test-trace")

	t.Run("NewSpanV1 creates span", func(t *testing.T) {
		span, err := trace.NewSpanV1(ctx, "test-span",
			langfuse.WithSpanInput("input data"),
			langfuse.WithSpanLevel(langfuse.ObservationLevelDebug),
		)
		if err != nil {
			t.Fatalf("NewSpanV1 failed: %v", err)
		}
		if span == nil {
			t.Fatal("span should not be nil")
		}
	})
}

func TestV1GenerationCreation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client := langfuse.NewClient("pk-lf-test-key", "sk-lf-test-key",
		langfuse.WithBaseURL(server.URL),
		langfuse.WithTimeout(5*time.Second),
		langfuse.WithShutdownTimeout(10*time.Second),
	)
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, _ := client.Trace(ctx, "test-trace")

	t.Run("NewGenerationV1 creates generation", func(t *testing.T) {
		gen, err := trace.NewGenerationV1(ctx, "test-gen",
			langfuse.WithModel("gpt-4"),
			langfuse.WithGenerationInput("test prompt"),
		)
		if err != nil {
			t.Fatalf("NewGenerationV1 failed: %v", err)
		}
		if gen == nil {
			t.Fatal("generation should not be nil")
		}
	})
}

func TestV1EndMethods(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client := langfuse.NewClient("pk-lf-test-key", "sk-lf-test-key",
		langfuse.WithBaseURL(server.URL),
		langfuse.WithTimeout(5*time.Second),
		langfuse.WithShutdownTimeout(10*time.Second),
	)
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	t.Run("span EndV1 returns (SpanContext, error)", func(t *testing.T) {
		trace, _ := client.Trace(ctx, "test-trace")
		span, _ := trace.Span(ctx, "test-span")

		resultSpan, err := span.EndV1(ctx,
			langfuse.WithEndOutput("result"),
			langfuse.WithEndMetadata(langfuse.Metadata{"key": "value"}),
		)
		if err != nil {
			t.Fatalf("EndV1 failed: %v", err)
		}
		if resultSpan == nil {
			t.Fatal("resultSpan should not be nil")
		}
		if resultSpan.ID() != span.ID() {
			t.Error("EndV1 should return the same span")
		}
	})

	t.Run("generation EndV1 returns (GenerationContext, error)", func(t *testing.T) {
		trace, _ := client.Trace(ctx, "test-trace")
		gen, _ := trace.Generation(ctx, "test-gen", langfuse.WithModel("gpt-4"))

		resultGen, err := gen.EndV1(ctx,
			langfuse.WithEndOutput("response"),
			langfuse.WithUsage(100, 50),
		)
		if err != nil {
			t.Fatalf("EndV1 failed: %v", err)
		}
		if resultGen == nil {
			t.Fatal("resultGen should not be nil")
		}
		if resultGen.ID() != gen.ID() {
			t.Error("EndV1 should return the same generation")
		}
	})
}

func TestV1UpdateMethod(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client := langfuse.NewClient("pk-lf-test-key", "sk-lf-test-key",
		langfuse.WithBaseURL(server.URL),
		langfuse.WithTimeout(5*time.Second),
		langfuse.WithShutdownTimeout(10*time.Second),
	)
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	t.Run("trace UpdateV1 returns (TraceContext, error)", func(t *testing.T) {
		trace, _ := client.Trace(ctx, "test-trace")

		updatedTrace, err := trace.UpdateV1(ctx,
			langfuse.WithUpdateOutput(map[string]interface{}{"result": "success"}),
			langfuse.WithUpdateTags("completed"),
		)
		if err != nil {
			t.Fatalf("UpdateV1 failed: %v", err)
		}
		if updatedTrace == nil {
			t.Fatal("updatedTrace should not be nil")
		}
		if updatedTrace.ID() != trace.ID() {
			t.Error("UpdateV1 should return the same trace")
		}
	})
}

func TestV1ScorerInterface(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client := langfuse.NewClient("pk-lf-test-key", "sk-lf-test-key",
		langfuse.WithBaseURL(server.URL),
		langfuse.WithTimeout(5*time.Second),
		langfuse.WithShutdownTimeout(10*time.Second),
	)
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	t.Run("TraceContext implements Scorer", func(t *testing.T) {
		trace, _ := client.Trace(ctx, "test-trace")
		if trace == nil {
			t.Fatal("trace should not be nil")
		}
		var scorer langfuse.Scorer = trace

		err := scorer.Score(ctx, "quality", 0.95)
		if err != nil {
			t.Errorf("Score failed: %v", err)
		}

		err = scorer.ScoreBool(ctx, "passed", true)
		if err != nil {
			t.Errorf("ScoreBool failed: %v", err)
		}

		err = scorer.ScoreCategory(ctx, "sentiment", "positive")
		if err != nil {
			t.Errorf("ScoreCategory failed: %v", err)
		}
	})

	t.Run("SpanContext implements Scorer", func(t *testing.T) {
		trace, _ := client.Trace(ctx, "test-trace")
		span, _ := trace.Span(ctx, "test-span")
		if span == nil {
			t.Fatal("span should not be nil")
		}
		var scorer langfuse.Scorer = span

		err := scorer.Score(ctx, "quality", 0.8)
		if err != nil {
			t.Errorf("Score failed: %v", err)
		}

		err = scorer.ScoreBool(ctx, "passed", false)
		if err != nil {
			t.Errorf("ScoreBool failed: %v", err)
		}

		err = scorer.ScoreCategory(ctx, "sentiment", "negative")
		if err != nil {
			t.Errorf("ScoreCategory failed: %v", err)
		}
	})

	t.Run("GenerationContext implements Scorer", func(t *testing.T) {
		trace, _ := client.Trace(ctx, "test-trace")
		gen, _ := trace.Generation(ctx, "test-gen", langfuse.WithModel("gpt-4"))
		if gen == nil {
			t.Fatal("generation should not be nil")
		}
		var scorer langfuse.Scorer = gen

		err := scorer.Score(ctx, "accuracy", 0.92)
		if err != nil {
			t.Errorf("Score failed: %v", err)
		}

		err = scorer.ScoreBool(ctx, "is_correct", true)
		if err != nil {
			t.Errorf("ScoreBool failed: %v", err)
		}

		err = scorer.ScoreCategory(ctx, "rating", "excellent")
		if err != nil {
			t.Errorf("ScoreCategory failed: %v", err)
		}
	})
}

func TestClientStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client := langfuse.NewClient("pk-lf-test-key", "sk-lf-test-key",
		langfuse.WithBaseURL(server.URL),
		langfuse.WithTimeout(5*time.Second),
		langfuse.WithShutdownTimeout(10*time.Second),
	)
	defer client.Shutdown(context.Background())

	t.Run("Stats returns valid statistics", func(t *testing.T) {
		stats := client.Stats()

		if stats.State != langfuse.ClientStateActive {
			t.Errorf("State = %v, want %v", stats.State, langfuse.ClientStateActive)
		}

		if stats.UptimeNanos <= 0 {
			t.Error("UptimeNanos should be positive")
		}

		// Batch pending events should be zero for new client
		if stats.Batch.PendingEvents != 0 {
			t.Errorf("Batch.PendingEvents = %d, want 0", stats.Batch.PendingEvents)
		}

		// BackpressureInfo.Level should be None for new client
		if stats.BackpressureInfo.Level != langfuse.BackpressureNone {
			t.Errorf("BackpressureInfo.Level = %v, want %v", stats.BackpressureInfo.Level, langfuse.BackpressureNone)
		}
	})
}

// NOTE: TestV1EndOptions and TestV1UpdateOptions tests were removed because they test
// unexported types (endConfig, spanConfig, updateConfig) that are not accessible
// from external test packages.

func TestScoreDataType(t *testing.T) {
	t.Run("ScoreDataType constants are defined", func(t *testing.T) {
		if langfuse.ScoreDataTypeNumeric != "NUMERIC" {
			t.Errorf("ScoreDataTypeNumeric = %s, want NUMERIC", langfuse.ScoreDataTypeNumeric)
		}
		if langfuse.ScoreDataTypeBoolean != "BOOLEAN" {
			t.Errorf("ScoreDataTypeBoolean = %s, want BOOLEAN", langfuse.ScoreDataTypeBoolean)
		}
		if langfuse.ScoreDataTypeCategorical != "CATEGORICAL" {
			t.Errorf("ScoreDataTypeCategorical = %s, want CATEGORICAL", langfuse.ScoreDataTypeCategorical)
		}
	})
}

func TestWithObservation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client := langfuse.NewClient("pk-lf-test-key", "sk-lf-test-key",
		langfuse.WithBaseURL(server.URL),
		langfuse.WithTimeout(5*time.Second),
		langfuse.WithShutdownTimeout(10*time.Second),
	)
	defer client.Shutdown(context.Background())

	ctx := context.Background()

	t.Run("WithObservation stores trace in context", func(t *testing.T) {
		trace, _ := client.Trace(ctx, "test-trace")
		ctx = langfuse.WithObservation(ctx, trace)

		recovered, ok := langfuse.TraceFromContext(ctx)
		if !ok {
			t.Fatal("trace should be in context")
		}
		if recovered.ID() != trace.ID() {
			t.Error("recovered trace should match original")
		}
	})

	t.Run("WithObservation stores span in context", func(t *testing.T) {
		trace, _ := client.Trace(ctx, "test-trace")
		span, _ := trace.Span(ctx, "test-span")
		ctx = langfuse.WithObservation(ctx, span)

		recovered, ok := langfuse.SpanFromContext(ctx)
		if !ok {
			t.Fatal("span should be in context")
		}
		if recovered.ID() != span.ID() {
			t.Error("recovered span should match original")
		}
	})
}

func TestV1FullWorkflow(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	t.Run("complete v1 API workflow", func(t *testing.T) {
		// Create client
		client := langfuse.NewClient("pk-lf-test-key", "sk-lf-test-key",
			langfuse.WithBaseURL(server.URL),
			langfuse.WithTimeout(5*time.Second),
			langfuse.WithShutdownTimeout(10*time.Second),
		)
		defer client.Shutdown(context.Background())

		ctx := context.Background()

		// Create trace using Trace() (simple API)
		trace, err := client.Trace(ctx, "user-request",
			langfuse.WithUserID("user-123"),
			langfuse.WithTags("api", "v2"),
		)
		if err != nil {
			t.Fatalf("Trace failed: %v", err)
		}

		// Create span
		span, err := trace.NewSpanV1(ctx, "processing",
			langfuse.WithSpanInput("input data"),
		)
		if err != nil {
			t.Fatalf("NewSpanV1 failed: %v", err)
		}

		// Create generation
		gen, err := span.NewGenerationV1(ctx, "llm-call",
			langfuse.WithModel("gpt-4"),
			langfuse.WithGenerationInput("prompt"),
		)
		if err != nil {
			t.Fatalf("NewGenerationV1 failed: %v", err)
		}

		// Add score
		err = gen.Score(ctx, "quality", 0.95,
			langfuse.WithScoreComment("Excellent response"),
		)
		if err != nil {
			t.Fatalf("Score failed: %v", err)
		}

		// End generation
		_, err = gen.EndV1(ctx,
			langfuse.WithEndOutput("response"),
			langfuse.WithUsage(100, 50),
		)
		if err != nil {
			t.Fatalf("EndV1 on generation failed: %v", err)
		}

		// End span
		_, err = span.EndV1(ctx,
			langfuse.WithEndOutput("processed"),
		)
		if err != nil {
			t.Fatalf("EndV1 on span failed: %v", err)
		}

		// Update trace
		_, err = trace.UpdateV1(ctx,
			langfuse.WithUpdateOutput(map[string]any{"status": "success"}),
			langfuse.WithUpdateTags("completed"),
		)
		if err != nil {
			t.Fatalf("UpdateV1 failed: %v", err)
		}

		// Check stats
		stats := client.Stats()
		if stats.State != langfuse.ClientStateActive {
			t.Errorf("State = %v, want %v", stats.State, langfuse.ClientStateActive)
		}
	})
}
