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

func TestNewClient(t *testing.T) {
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
		client := langfuse.MustNew("pk-lf-test-key", "sk-lf-test-key",
			langfuse.WithBaseURL(server.URL),
			langfuse.WithTimeout(5*time.Second),
			langfuse.WithShutdownTimeout(10*time.Second),
		)
		if client == nil {
			t.Fatal("MustNew returned nil")
		}
		defer client.Shutdown(context.Background())

		if !client.IsActive() {
			t.Error("client should be active")
		}
	})

	t.Run("MustNew panics with empty credentials", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustNew should panic with empty credentials")
			}
		}()
		_ = langfuse.MustNew("", "", langfuse.WithBaseURL(server.URL))
	})

	t.Run("New returns an error with empty credentials", func(t *testing.T) {
		client, err := langfuse.New("", "", langfuse.WithBaseURL(server.URL))
		if err == nil {
			t.Error("New should return an error with empty credentials")
			if client != nil {
				client.Shutdown(context.Background())
			}
		}
		if client != nil {
			t.Error("New should return a nil client on error")
		}
	})
}

func TestScorerInterface(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/public/ingestion" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "1", Status: 200}},
			})
		}
	}))
	defer server.Close()

	client := langfuse.MustNew("pk-lf-test-key", "sk-lf-test-key",
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

	client := langfuse.MustNew("pk-lf-test-key", "sk-lf-test-key",
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

	client := langfuse.MustNew("pk-lf-test-key", "sk-lf-test-key",
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
