package langfuse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTracesClientList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/traces" {
			t.Errorf("Expected /traces, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		// Check query parameters
		query := r.URL.Query()
		if query.Get("page") != "1" {
			t.Errorf("Expected page=1, got %s", query.Get("page"))
		}
		if query.Get("limit") != "10" {
			t.Errorf("Expected limit=10, got %s", query.Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TracesListResponse{
			Data: []Trace{
				{ID: "trace-1", Name: "Trace 1"},
				{ID: "trace-2", Name: "Trace 2"},
			},
			Meta: MetaResponse{
				Page:       1,
				Limit:      10,
				TotalItems: 2,
				TotalPages: 1,
			},
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Traces().List(context.Background(), &TracesListParams{
		PaginationParams: PaginationParams{Page: 1, Limit: 10},
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("Expected 2 traces, got %d", len(result.Data))
	}
	if result.Data[0].ID != "trace-1" {
		t.Errorf("Expected trace-1, got %s", result.Data[0].ID)
	}
}

func TestTracesClientListWithFilters(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("userId") != "user-123" {
			t.Errorf("Expected userId=user-123, got %s", query.Get("userId"))
		}
		if query.Get("sessionId") != "session-456" {
			t.Errorf("Expected sessionId=session-456, got %s", query.Get("sessionId"))
		}
		if query.Get("name") != "test-trace" {
			t.Errorf("Expected name=test-trace, got %s", query.Get("name"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TracesListResponse{
			Data: []Trace{{ID: "trace-1"}},
			Meta: MetaResponse{TotalItems: 1},
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Traces().List(context.Background(), &TracesListParams{
		FilterParams: FilterParams{
			UserID:    "user-123",
			SessionID: "session-456",
			Name:      "test-trace",
		},
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Data) != 1 {
		t.Errorf("Expected 1 trace, got %d", len(result.Data))
	}
}

func TestTracesClientListNilParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TracesListResponse{
			Data: []Trace{},
			Meta: MetaResponse{},
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Traces().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List with nil params failed: %v", err)
	}

	if result == nil {
		t.Error("Result should not be nil")
	}
}

func TestTracesClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/traces/trace-123" {
			t.Errorf("Expected /traces/trace-123, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Trace{
			ID:        "trace-123",
			Name:      "Test Trace",
			UserID:    "user-456",
			SessionID: "session-789",
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	trace, err := client.Traces().Get(context.Background(), "trace-123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if trace.ID != "trace-123" {
		t.Errorf("Expected ID trace-123, got %s", trace.ID)
	}
	if trace.Name != "Test Trace" {
		t.Errorf("Expected Name Test Trace, got %s", trace.Name)
	}
}

func TestTracesClientGetNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"statusCode": 404,
			"message":    "Trace not found",
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	_, err := client.Traces().Get(context.Background(), "nonexistent")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}
	if !apiErr.IsNotFound() {
		t.Errorf("Expected 404 error, got %d", apiErr.StatusCode)
	}
}

func TestTracesClientDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/traces/trace-123" {
			t.Errorf("Expected /traces/trace-123, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	err := client.Traces().Delete(context.Background(), "trace-123")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}
