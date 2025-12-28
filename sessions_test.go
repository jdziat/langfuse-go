package langfuse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSessionsClientList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sessions" {
			t.Errorf("Expected /sessions, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SessionsListResponse{
			Data: []Session{
				{ID: "session-1"},
				{ID: "session-2"},
			},
			Meta: MetaResponse{TotalItems: 2},
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Sessions().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("Expected 2 sessions, got %d", len(result.Data))
	}
}

func TestSessionsClientListWithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("fromTimestamp") != "2024-01-01T00:00:00Z" {
			t.Errorf("Expected fromTimestamp, got %s", query.Get("fromTimestamp"))
		}
		if query.Get("toTimestamp") != "2024-12-31T23:59:59Z" {
			t.Errorf("Expected toTimestamp, got %s", query.Get("toTimestamp"))
		}
		if query.Get("page") != "1" {
			t.Errorf("Expected page=1, got %s", query.Get("page"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SessionsListResponse{
			Data: []Session{{ID: "session-1"}},
			Meta: MetaResponse{TotalItems: 1},
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Sessions().List(context.Background(), &SessionsListParams{
		PaginationParams: PaginationParams{Page: 1},
		FromTimestamp:    "2024-01-01T00:00:00Z",
		ToTimestamp:      "2024-12-31T23:59:59Z",
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Data) != 1 {
		t.Errorf("Expected 1 session, got %d", len(result.Data))
	}
}

func TestSessionsClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sessions/session-123" {
			t.Errorf("Expected /sessions/session-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Session{
			ID:        "session-123",
			ProjectID: "project-456",
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	session, err := client.Sessions().Get(context.Background(), "session-123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if session.ID != "session-123" {
		t.Errorf("Expected ID session-123, got %s", session.ID)
	}
}

func TestSessionsClientGetWithTraces(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		if r.URL.Path == "/sessions/session-123" {
			json.NewEncoder(w).Encode(Session{
				ID: "session-123",
			})
			return
		}

		if r.URL.Path == "/traces" {
			query := r.URL.Query()
			if query.Get("sessionId") != "session-123" {
				t.Errorf("Expected sessionId=session-123, got %s", query.Get("sessionId"))
			}
			json.NewEncoder(w).Encode(TracesListResponse{
				Data: []Trace{
					{ID: "trace-1", SessionID: "session-123"},
					{ID: "trace-2", SessionID: "session-123"},
				},
				Meta: MetaResponse{TotalItems: 2},
			})
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Sessions().GetWithTraces(context.Background(), "session-123")
	if err != nil {
		t.Fatalf("GetWithTraces failed: %v", err)
	}

	if result.ID != "session-123" {
		t.Errorf("Expected session ID session-123, got %s", result.ID)
	}

	if len(result.Traces) != 2 {
		t.Errorf("Expected 2 traces, got %d", len(result.Traces))
	}

	if callCount != 2 {
		t.Errorf("Expected 2 API calls (session + traces), got %d", callCount)
	}
}

func TestSessionsClientGetNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"statusCode": 404,
			"message":    "Session not found",
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	_, err := client.Sessions().Get(context.Background(), "nonexistent")
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
