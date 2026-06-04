package langfuse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newStrictTestServer returns an httptest server that accepts ingestion so the
// strict-OFF control paths can enqueue without background-send noise.
func newStrictTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		})
	}))
}

// newStrictTestClient builds a client pointed at server. When strict is true,
// strict validation is enabled via WithStrictValidationEnabled. A large batch
// size and long flush interval keep the test free of background-send timing.
func newStrictTestClient(t *testing.T, server *httptest.Server, strict bool) *Client {
	t.Helper()
	opts := []ConfigOption{
		WithBaseURL(server.URL),
		WithBatchSize(1000),
		WithFlushInterval(1 * time.Hour),
	}
	if strict {
		opts = append(opts, WithStrictValidationEnabled())
	}
	client, err := New("pk-lf-test-key", "sk-lf-test-key", opts...)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	return client
}

// overlongName is longer than MaxNameLength so ValidateName (used by the
// Validated* builders) rejects it, while the standard builders' Name() setters
// accept it without complaint. It is therefore input "the Validated* path
// rejects" but the standard path historically accepted.
var overlongName = strings.Repeat("x", MaxNameLength+1)

// TestStrictValidation_TraceBuilder_Create verifies that the standard
// TraceBuilder consults the client's StrictValidation config on Create: with
// strict ON an overlong name yields a validation error; with strict OFF the
// same call succeeds, proving default-off behavior is unchanged.
func TestStrictValidation_TraceBuilder_Create(t *testing.T) {
	server := newStrictTestServer(t)
	defer server.Close()

	t.Run("strict on rejects invalid name", func(t *testing.T) {
		client := newStrictTestClient(t, server, true)
		defer client.Shutdown(context.Background())

		_, err := client.NewTrace().Name(overlongName).Create(context.Background())
		if err == nil {
			t.Fatal("expected a validation error with strict validation enabled, got nil")
		}
		if _, ok := AsValidationError(err); !ok {
			t.Errorf("expected a *ValidationError, got %T: %v", err, err)
		}
	})

	t.Run("strict off accepts invalid name", func(t *testing.T) {
		client := newStrictTestClient(t, server, false)
		defer client.Shutdown(context.Background())

		if _, err := client.NewTrace().Name(overlongName).Create(context.Background()); err != nil {
			t.Errorf("expected no error with strict validation disabled, got %v", err)
		}
	})
}

// TestStrictValidation_SpanBuilder_Create verifies the same strict-on/strict-off
// contract for the standard SpanBuilder.
func TestStrictValidation_SpanBuilder_Create(t *testing.T) {
	server := newStrictTestServer(t)
	defer server.Close()

	t.Run("strict on rejects invalid name", func(t *testing.T) {
		client := newStrictTestClient(t, server, true)
		defer client.Shutdown(context.Background())

		trace, err := client.NewTrace().Name("ok").Create(context.Background())
		if err != nil {
			t.Fatalf("Create trace failed: %v", err)
		}
		if _, err := trace.NewSpan().Name(overlongName).Create(context.Background()); err == nil {
			t.Fatal("expected a validation error with strict validation enabled, got nil")
		} else if _, ok := AsValidationError(err); !ok {
			t.Errorf("expected a *ValidationError, got %T: %v", err, err)
		}
	})

	t.Run("strict off accepts invalid name", func(t *testing.T) {
		client := newStrictTestClient(t, server, false)
		defer client.Shutdown(context.Background())

		trace, err := client.NewTrace().Name("ok").Create(context.Background())
		if err != nil {
			t.Fatalf("Create trace failed: %v", err)
		}
		if _, err := trace.NewSpan().Name(overlongName).Create(context.Background()); err != nil {
			t.Errorf("expected no error with strict validation disabled, got %v", err)
		}
	})
}

// TestStrictValidation_GenerationBuilder_Create verifies the strict-on/strict-off
// contract for the standard GenerationBuilder, including a negative-usage value
// (rejected by ValidatedGenerationBuilder.UsageDetails) that the standard path
// would otherwise accept.
func TestStrictValidation_GenerationBuilder_Create(t *testing.T) {
	server := newStrictTestServer(t)
	defer server.Close()

	t.Run("strict on rejects invalid name", func(t *testing.T) {
		client := newStrictTestClient(t, server, true)
		defer client.Shutdown(context.Background())

		trace, err := client.NewTrace().Name("ok").Create(context.Background())
		if err != nil {
			t.Fatalf("Create trace failed: %v", err)
		}
		if _, err := trace.NewGeneration().Name(overlongName).Create(context.Background()); err == nil {
			t.Fatal("expected a validation error with strict validation enabled, got nil")
		} else if _, ok := AsValidationError(err); !ok {
			t.Errorf("expected a *ValidationError, got %T: %v", err, err)
		}
	})

	t.Run("strict on rejects negative usage", func(t *testing.T) {
		client := newStrictTestClient(t, server, true)
		defer client.Shutdown(context.Background())

		trace, err := client.NewTrace().Name("ok").Create(context.Background())
		if err != nil {
			t.Fatalf("Create trace failed: %v", err)
		}
		_, err = trace.NewGeneration().
			Name("gen").
			Usage(&Usage{Input: -1}).
			Create(context.Background())
		if err == nil {
			t.Fatal("expected a validation error for negative usage with strict validation enabled, got nil")
		}
		if _, ok := AsValidationError(err); !ok {
			t.Errorf("expected a *ValidationError, got %T: %v", err, err)
		}
	})

	t.Run("strict off accepts invalid name", func(t *testing.T) {
		client := newStrictTestClient(t, server, false)
		defer client.Shutdown(context.Background())

		trace, err := client.NewTrace().Name("ok").Create(context.Background())
		if err != nil {
			t.Fatalf("Create trace failed: %v", err)
		}
		if _, err := trace.NewGeneration().Name(overlongName).Create(context.Background()); err != nil {
			t.Errorf("expected no error with strict validation disabled, got %v", err)
		}
	})
}

// TestStrictValidation_FailFast verifies that the FailFast option controls how
// multiple strict-only violations are reported. The generation carries two
// violations the standard path does not catch on its own (an overlong name and
// negative usage), so both must originate from the strict path: with FailFast
// off they are combined; with it on a single error is returned.
func TestStrictValidation_FailFast(t *testing.T) {
	server := newStrictTestServer(t)
	defer server.Close()

	makeTrace := func(failFast bool) *TraceContext {
		client, err := New("pk-lf-test-key", "sk-lf-test-key",
			WithBaseURL(server.URL),
			WithBatchSize(1000),
			WithFlushInterval(1*time.Hour),
			WithStructuredLogger(NopLogger{}),
			WithStrictValidation(StrictValidationConfig{Enabled: true, FailFast: failFast}),
		)
		if err != nil {
			t.Fatalf("New failed: %v", err)
		}
		t.Cleanup(func() { client.Shutdown(context.Background()) })
		trace, err := client.NewTrace().Name("ok").Create(context.Background())
		if err != nil {
			t.Fatalf("Create trace failed: %v", err)
		}
		return trace
	}

	t.Run("fail fast returns single error", func(t *testing.T) {
		trace := makeTrace(true)
		_, err := trace.NewGeneration().
			Name(overlongName).
			Usage(&Usage{Input: -1}).
			Create(context.Background())
		if err == nil {
			t.Fatal("expected a validation error, got nil")
		}
		if strings.Contains(err.Error(), "multiple validation errors") {
			t.Errorf("FailFast should return a single error, got combined: %v", err)
		}
	})

	t.Run("accumulate returns combined error", func(t *testing.T) {
		trace := makeTrace(false)
		_, err := trace.NewGeneration().
			Name(overlongName).
			Usage(&Usage{Input: -1}).
			Create(context.Background())
		if err == nil {
			t.Fatal("expected a validation error, got nil")
		}
		if !strings.Contains(err.Error(), "multiple validation errors") {
			t.Errorf("accumulate mode should combine errors, got: %v", err)
		}
	})
}
