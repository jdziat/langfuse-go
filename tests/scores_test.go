package langfuse_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	langfuse "github.com/jdziat/langfuse-go"
)

func TestScoresClientList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scores" {
			t.Errorf("Expected /scores, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.ScoresListResponse{
			Data: []langfuse.Score{
				{ID: "score-1", Name: "quality", Value: 0.95},
				{ID: "score-2", Name: "relevance", Value: 0.88},
			},
			Meta: langfuse.MetaResponse{TotalItems: 2},
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Scores().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("Expected 2 scores, got %d", len(result.Data))
	}
}

func TestScoresClientListWithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("traceId") != "trace-123" {
			t.Errorf("Expected traceId=trace-123, got %s", query.Get("traceId"))
		}
		if query.Get("name") != "quality" {
			t.Errorf("Expected name=quality, got %s", query.Get("name"))
		}
		if query.Get("dataType") != "NUMERIC" {
			t.Errorf("Expected dataType=NUMERIC, got %s", query.Get("dataType"))
		}
		if query.Get("source") != "API" {
			t.Errorf("Expected source=API, got %s", query.Get("source"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.ScoresListResponse{
			Data: []langfuse.Score{{ID: "score-1"}},
			Meta: langfuse.MetaResponse{TotalItems: 1},
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Scores().List(context.Background(), &langfuse.ScoresListParams{
		TraceID:  "trace-123",
		Name:     "quality",
		DataType: langfuse.ScoreDataTypeNumeric,
		Source:   langfuse.ScoreSourceAPI,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Data) != 1 {
		t.Errorf("Expected 1 score, got %d", len(result.Data))
	}
}

func TestScoresClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scores/score-123" {
			t.Errorf("Expected /scores/score-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.Score{
			ID:      "score-123",
			Name:    "quality",
			Value:   0.95,
			TraceID: "trace-456",
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	score, err := client.Scores().Get(context.Background(), "score-123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if score.ID != "score-123" {
		t.Errorf("Expected ID score-123, got %s", score.ID)
	}
	if score.Name != "quality" {
		t.Errorf("Expected name quality, got %s", score.Name)
	}
}

func TestScoresClientCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var req langfuse.CreateScoreRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.TraceID != "trace-123" {
			t.Errorf("Expected traceId trace-123, got %s", req.TraceID)
		}
		if req.Name != "quality" {
			t.Errorf("Expected name quality, got %s", req.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.Score{
			ID:      "score-new",
			TraceID: req.TraceID,
			Name:    req.Name,
			Value:   req.Value,
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	score, err := client.Scores().Create(context.Background(), &langfuse.CreateScoreRequest{
		TraceID:  "trace-123",
		Name:     "quality",
		Value:    0.95,
		DataType: langfuse.ScoreDataTypeNumeric,
		Comment:  "High quality response",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if score.ID != "score-new" {
		t.Errorf("Expected ID score-new, got %s", score.ID)
	}
}

func TestScoresClientCreateValidation(t *testing.T) {
	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key")
	defer client.Shutdown(context.Background())

	// Nil request
	_, err := client.Scores().Create(context.Background(), nil)
	if err != langfuse.ErrNilRequest {
		t.Errorf("Expected ErrNilRequest, got %v", err)
	}

	// Missing traceId
	_, err = client.Scores().Create(context.Background(), &langfuse.CreateScoreRequest{
		Name:  "quality",
		Value: 0.9,
	})
	if err == nil {
		t.Error("Expected validation error for missing traceId")
	}

	// Missing name
	_, err = client.Scores().Create(context.Background(), &langfuse.CreateScoreRequest{
		TraceID: "trace-123",
		Value:   0.9,
	})
	if err == nil {
		t.Error("Expected validation error for missing name")
	}
}

func TestScoresClientDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/scores/score-123" {
			t.Errorf("Expected /scores/score-123, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	err := client.Scores().Delete(context.Background(), "score-123")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestScoresClientListByTrace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("traceId") != "trace-123" {
			t.Errorf("Expected traceId=trace-123, got %s", query.Get("traceId"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.ScoresListResponse{
			Data: []langfuse.Score{{ID: "score-1", TraceID: "trace-123"}},
			Meta: langfuse.MetaResponse{TotalItems: 1},
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Scores().ListByTrace(context.Background(), "trace-123", nil)
	if err != nil {
		t.Fatalf("ListByTrace failed: %v", err)
	}

	if len(result.Data) != 1 {
		t.Errorf("Expected 1 score, got %d", len(result.Data))
	}
}

func TestScoresClientListByObservation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("observationId") != "obs-123" {
			t.Errorf("Expected observationId=obs-123, got %s", query.Get("observationId"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.ScoresListResponse{
			Data: []langfuse.Score{{ID: "score-1", ObservationID: "obs-123"}},
			Meta: langfuse.MetaResponse{TotalItems: 1},
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Scores().ListByObservation(context.Background(), "obs-123", nil)
	if err != nil {
		t.Fatalf("ListByObservation failed: %v", err)
	}

	if len(result.Data) != 1 {
		t.Errorf("Expected 1 score, got %d", len(result.Data))
	}
}
