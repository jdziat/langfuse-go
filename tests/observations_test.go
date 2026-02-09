package langfuse_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	langfuse "github.com/jdziat/langfuse-go"
)

func TestObservationsClientList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/observations" {
			t.Errorf("Expected /observations, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.ObservationsListResponse{
			Data: []langfuse.Observation{
				{ID: "obs-1", Name: "Observation 1", Type: langfuse.ObservationTypeSpan},
				{ID: "obs-2", Name: "Observation 2", Type: langfuse.ObservationTypeGeneration},
			},
			Meta: langfuse.MetaResponse{TotalItems: 2},
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Observations().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("Expected 2 observations, got %d", len(result.Data))
	}
}

func TestObservationsClientListWithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("traceId") != "trace-123" {
			t.Errorf("Expected traceId=trace-123, got %s", query.Get("traceId"))
		}
		if query.Get("parentObservationId") != "parent-456" {
			t.Errorf("Expected parentObservationId=parent-456, got %s", query.Get("parentObservationId"))
		}
		if query.Get("type") != "GENERATION" {
			t.Errorf("Expected type=GENERATION, got %s", query.Get("type"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.ObservationsListResponse{
			Data: []langfuse.Observation{{ID: "obs-1"}},
			Meta: langfuse.MetaResponse{TotalItems: 1},
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Observations().List(context.Background(), &langfuse.ObservationsListParams{
		FilterParams:        langfuse.FilterParams{TraceID: "trace-123", Type: "GENERATION"},
		ParentObservationID: "parent-456",
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Data) != 1 {
		t.Errorf("Expected 1 observation, got %d", len(result.Data))
	}
}

func TestObservationsClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/observations/obs-123" {
			t.Errorf("Expected /observations/obs-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.Observation{
			ID:      "obs-123",
			Name:    "Test Observation",
			Type:    langfuse.ObservationTypeGeneration,
			TraceID: "trace-456",
			Model:   "gpt-4",
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	obs, err := client.Observations().Get(context.Background(), "obs-123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if obs.ID != "obs-123" {
		t.Errorf("Expected ID obs-123, got %s", obs.ID)
	}
	if obs.Type != langfuse.ObservationTypeGeneration {
		t.Errorf("Expected type GENERATION, got %s", obs.Type)
	}
	if obs.Model != "gpt-4" {
		t.Errorf("Expected model gpt-4, got %s", obs.Model)
	}
}

func TestObservationsClientListByTrace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("traceId") != "trace-123" {
			t.Errorf("Expected traceId=trace-123, got %s", query.Get("traceId"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.ObservationsListResponse{
			Data: []langfuse.Observation{
				{ID: "obs-1", TraceID: "trace-123"},
				{ID: "obs-2", TraceID: "trace-123"},
			},
			Meta: langfuse.MetaResponse{TotalItems: 2},
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Observations().ListByTrace(context.Background(), "trace-123", nil)
	if err != nil {
		t.Fatalf("ListByTrace failed: %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("Expected 2 observations, got %d", len(result.Data))
	}
}

func TestObservationsClientListSpans(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("type") != "SPAN" {
			t.Errorf("Expected type=SPAN, got %s", query.Get("type"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.ObservationsListResponse{
			Data: []langfuse.Observation{
				{ID: "span-1", Type: langfuse.ObservationTypeSpan},
			},
			Meta: langfuse.MetaResponse{TotalItems: 1},
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Observations().ListSpans(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListSpans failed: %v", err)
	}

	if len(result.Data) != 1 {
		t.Errorf("Expected 1 span, got %d", len(result.Data))
	}
}

func TestObservationsClientListGenerations(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("type") != "GENERATION" {
			t.Errorf("Expected type=GENERATION, got %s", query.Get("type"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.ObservationsListResponse{
			Data: []langfuse.Observation{
				{ID: "gen-1", Type: langfuse.ObservationTypeGeneration},
			},
			Meta: langfuse.MetaResponse{TotalItems: 1},
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Observations().ListGenerations(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListGenerations failed: %v", err)
	}

	if len(result.Data) != 1 {
		t.Errorf("Expected 1 generation, got %d", len(result.Data))
	}
}

func TestObservationsClientListEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("type") != "EVENT" {
			t.Errorf("Expected type=EVENT, got %s", query.Get("type"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.ObservationsListResponse{
			Data: []langfuse.Observation{
				{ID: "event-1", Type: langfuse.ObservationTypeEvent},
			},
			Meta: langfuse.MetaResponse{TotalItems: 1},
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Observations().ListEvents(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListEvents failed: %v", err)
	}

	if len(result.Data) != 1 {
		t.Errorf("Expected 1 event, got %d", len(result.Data))
	}
}
