package langfuse_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	langfuse "github.com/jdziat/langfuse-go"
)

func TestModelsClientList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models" {
			t.Errorf("Expected /models, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.ModelsListResponse{
			Data: []langfuse.Model{
				{ID: "model-1", ModelName: "gpt-4"},
				{ID: "model-2", ModelName: "gpt-3.5-turbo"},
			},
			Meta: langfuse.MetaResponse{TotalItems: 2},
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Models().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("Expected 2 models, got %d", len(result.Data))
	}
}

func TestModelsClientListWithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("page") != "2" {
			t.Errorf("Expected page=2, got %s", query.Get("page"))
		}
		if query.Get("limit") != "10" {
			t.Errorf("Expected limit=10, got %s", query.Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.ModelsListResponse{
			Data: []langfuse.Model{{ID: "model-1"}},
			Meta: langfuse.MetaResponse{Page: 2, Limit: 10, TotalItems: 15},
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Models().List(context.Background(), &langfuse.ModelsListParams{
		PaginationParams: langfuse.PaginationParams{Page: 2, Limit: 10},
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if result.Meta.Page != 2 {
		t.Errorf("Expected page 2, got %d", result.Meta.Page)
	}
}

func TestModelsClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models/model-123" {
			t.Errorf("Expected /models/model-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.Model{
			ID:          "model-123",
			ModelName:   "gpt-4-custom",
			InputPrice:  0.03,
			OutputPrice: 0.06,
			Unit:        "TOKENS",
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	model, err := client.Models().Get(context.Background(), "model-123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if model.ID != "model-123" {
		t.Errorf("Expected ID model-123, got %s", model.ID)
	}
	if model.ModelName != "gpt-4-custom" {
		t.Errorf("Expected ModelName gpt-4-custom, got %s", model.ModelName)
	}
	if model.InputPrice != 0.03 {
		t.Errorf("Expected InputPrice 0.03, got %f", model.InputPrice)
	}
}

func TestModelsClientCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var req langfuse.CreateModelRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.ModelName != "my-custom-model" {
			t.Errorf("Expected modelName my-custom-model, got %s", req.ModelName)
		}
		if req.InputPrice != 0.001 {
			t.Errorf("Expected inputPrice 0.001, got %f", req.InputPrice)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.Model{
			ID:          "model-new",
			ModelName:   req.ModelName,
			InputPrice:  req.InputPrice,
			OutputPrice: req.OutputPrice,
			Unit:        req.Unit,
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	model, err := client.Models().Create(context.Background(), &langfuse.CreateModelRequest{
		ModelName:    "my-custom-model",
		MatchPattern: "my-custom-.*",
		InputPrice:   0.001,
		OutputPrice:  0.002,
		Unit:         "TOKENS",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if model.ID != "model-new" {
		t.Errorf("Expected ID model-new, got %s", model.ID)
	}
	if model.ModelName != "my-custom-model" {
		t.Errorf("Expected ModelName my-custom-model, got %s", model.ModelName)
	}
}

func TestModelsClientCreateValidation(t *testing.T) {
	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key")
	defer client.Shutdown(context.Background())

	// Nil request
	_, err := client.Models().Create(context.Background(), nil)
	if err != langfuse.ErrNilRequest {
		t.Errorf("Expected ErrNilRequest, got %v", err)
	}

	// Missing modelName
	_, err = client.Models().Create(context.Background(), &langfuse.CreateModelRequest{
		InputPrice: 0.001,
	})
	if err == nil {
		t.Error("Expected validation error for missing modelName")
	}
}

func TestModelsClientDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/models/model-123" {
			t.Errorf("Expected /models/model-123, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	err := client.Models().Delete(context.Background(), "model-123")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
}

func TestModelsClientDeleteNotAllowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"statusCode": 403,
			"message":    "Cannot delete Langfuse-managed models",
		})
	}))
	defer server.Close()

	client, _ := langfuse.New("pk-lf-test-key", "sk-lf-test-key", langfuse.WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	err := client.Models().Delete(context.Background(), "langfuse-managed-model")
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	apiErr, ok := err.(*langfuse.APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}
	if !apiErr.IsForbidden() {
		t.Errorf("Expected 403 error, got %d", apiErr.StatusCode)
	}
}
