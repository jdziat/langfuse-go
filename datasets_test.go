package langfuse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDatasetsClientList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/datasets" {
			t.Errorf("Expected /v2/datasets, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DatasetsListResponse{
			Data: []Dataset{
				{ID: "ds-1", Name: "Dataset 1"},
				{ID: "ds-2", Name: "Dataset 2"},
			},
			Meta: MetaResponse{TotalItems: 2},
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Datasets().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("Expected 2 datasets, got %d", len(result.Data))
	}
}

func TestDatasetsClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/datasets/my-dataset" {
			t.Errorf("Expected /v2/datasets/my-dataset, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Dataset{
			ID:          "ds-123",
			Name:        "my-dataset",
			Description: "Test dataset",
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	dataset, err := client.Datasets().Get(context.Background(), "my-dataset")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if dataset.Name != "my-dataset" {
		t.Errorf("Expected name my-dataset, got %s", dataset.Name)
	}
}

func TestDatasetsClientCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var req CreateDatasetRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Name != "new-dataset" {
			t.Errorf("Expected name new-dataset, got %s", req.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Dataset{
			ID:          "ds-new",
			Name:        req.Name,
			Description: req.Description,
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	dataset, err := client.Datasets().Create(context.Background(), &CreateDatasetRequest{
		Name:        "new-dataset",
		Description: "A new dataset",
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if dataset.Name != "new-dataset" {
		t.Errorf("Expected name new-dataset, got %s", dataset.Name)
	}
}

func TestDatasetsClientCreateValidation(t *testing.T) {
	client, _ := New("pk-lf-test-key", "sk-lf-test-key")
	defer client.Shutdown(context.Background())

	// Nil request
	_, err := client.Datasets().Create(context.Background(), nil)
	if err != ErrNilRequest {
		t.Errorf("Expected ErrNilRequest, got %v", err)
	}

	// Missing name
	_, err = client.Datasets().Create(context.Background(), &CreateDatasetRequest{
		Description: "Description only",
	})
	if err == nil {
		t.Error("Expected validation error for missing name")
	}
}

func TestDatasetsClientListItems(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dataset-items" {
			t.Errorf("Expected /dataset-items, got %s", r.URL.Path)
		}

		query := r.URL.Query()
		if query.Get("datasetName") != "my-dataset" {
			t.Errorf("Expected datasetName=my-dataset, got %s", query.Get("datasetName"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DatasetItemsListResponse{
			Data: []DatasetItem{
				{ID: "item-1"},
				{ID: "item-2"},
			},
			Meta: MetaResponse{TotalItems: 2},
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Datasets().ListItems(context.Background(), &DatasetItemsListParams{
		DatasetName: "my-dataset",
	})
	if err != nil {
		t.Fatalf("ListItems failed: %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("Expected 2 items, got %d", len(result.Data))
	}
}

func TestDatasetsClientGetItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dataset-items/item-123" {
			t.Errorf("Expected /dataset-items/item-123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DatasetItem{
			ID:             "item-123",
			DatasetName:    "my-dataset",
			Input:          map[string]string{"question": "What is 2+2?"},
			ExpectedOutput: map[string]string{"answer": "4"},
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	item, err := client.Datasets().GetItem(context.Background(), "item-123")
	if err != nil {
		t.Fatalf("GetItem failed: %v", err)
	}

	if item.ID != "item-123" {
		t.Errorf("Expected ID item-123, got %s", item.ID)
	}
}

func TestDatasetsClientCreateItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var req CreateDatasetItemRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.DatasetName != "my-dataset" {
			t.Errorf("Expected datasetName my-dataset, got %s", req.DatasetName)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DatasetItem{
			ID:             "item-new",
			DatasetName:    req.DatasetName,
			Input:          req.Input,
			ExpectedOutput: req.ExpectedOutput,
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	item, err := client.Datasets().CreateItem(context.Background(), &CreateDatasetItemRequest{
		DatasetName:    "my-dataset",
		Input:          map[string]string{"question": "What is 3+3?"},
		ExpectedOutput: map[string]string{"answer": "6"},
	})
	if err != nil {
		t.Fatalf("CreateItem failed: %v", err)
	}

	if item.ID != "item-new" {
		t.Errorf("Expected ID item-new, got %s", item.ID)
	}
}

func TestDatasetsClientCreateItemValidation(t *testing.T) {
	client, _ := New("pk-lf-test-key", "sk-lf-test-key")
	defer client.Shutdown(context.Background())

	// Nil request
	_, err := client.Datasets().CreateItem(context.Background(), nil)
	if err != ErrNilRequest {
		t.Errorf("Expected ErrNilRequest, got %v", err)
	}

	// Missing datasetName
	_, err = client.Datasets().CreateItem(context.Background(), &CreateDatasetItemRequest{
		Input: map[string]string{"q": "test"},
	})
	if err == nil {
		t.Error("Expected validation error for missing datasetName")
	}
}

func TestDatasetsClientDeleteItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/dataset-items/item-123" {
			t.Errorf("Expected /dataset-items/item-123, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	err := client.Datasets().DeleteItem(context.Background(), "item-123")
	if err != nil {
		t.Fatalf("DeleteItem failed: %v", err)
	}
}

func TestDatasetsClientListRuns(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/datasets/my-dataset/runs" {
			t.Errorf("Expected /datasets/my-dataset/runs, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DatasetRunsListResponse{
			Data: []DatasetRun{
				{ID: "run-1", Name: "Run 1"},
				{ID: "run-2", Name: "Run 2"},
			},
			Meta: MetaResponse{TotalItems: 2},
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Datasets().ListRuns(context.Background(), "my-dataset", nil)
	if err != nil {
		t.Fatalf("ListRuns failed: %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("Expected 2 runs, got %d", len(result.Data))
	}
}

func TestDatasetsClientGetRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/datasets/my-dataset/runs/my-run" {
			t.Errorf("Expected /datasets/my-dataset/runs/my-run, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DatasetRun{
			ID:          "run-123",
			Name:        "my-run",
			DatasetName: "my-dataset",
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	run, err := client.Datasets().GetRun(context.Background(), "my-dataset", "my-run")
	if err != nil {
		t.Fatalf("GetRun failed: %v", err)
	}

	if run.Name != "my-run" {
		t.Errorf("Expected name my-run, got %s", run.Name)
	}
}

func TestDatasetsClientDeleteRun(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/datasets/my-dataset/runs/my-run" {
			t.Errorf("Expected /datasets/my-dataset/runs/my-run, got %s", r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	err := client.Datasets().DeleteRun(context.Background(), "my-dataset", "my-run")
	if err != nil {
		t.Fatalf("DeleteRun failed: %v", err)
	}
}

func TestDatasetsClientCreateRunItem(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var req CreateDatasetRunItemRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.DatasetItemID != "item-123" {
			t.Errorf("Expected datasetItemId item-123, got %s", req.DatasetItemID)
		}
		if req.RunName != "my-run" {
			t.Errorf("Expected runName my-run, got %s", req.RunName)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DatasetRunItem{
			ID:             "run-item-new",
			DatasetItemID:  req.DatasetItemID,
			DatasetRunName: req.RunName,
			TraceID:        req.TraceID,
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	runItem, err := client.Datasets().CreateRunItem(context.Background(), &CreateDatasetRunItemRequest{
		DatasetItemID: "item-123",
		RunName:       "my-run",
		TraceID:       "trace-456",
	})
	if err != nil {
		t.Fatalf("CreateRunItem failed: %v", err)
	}

	if runItem.ID != "run-item-new" {
		t.Errorf("Expected ID run-item-new, got %s", runItem.ID)
	}
}

func TestDatasetsClientCreateRunItemValidation(t *testing.T) {
	client, _ := New("pk-lf-test-key", "sk-lf-test-key")
	defer client.Shutdown(context.Background())

	// Nil request
	_, err := client.Datasets().CreateRunItem(context.Background(), nil)
	if err != ErrNilRequest {
		t.Errorf("Expected ErrNilRequest, got %v", err)
	}

	// Missing datasetItemId
	_, err = client.Datasets().CreateRunItem(context.Background(), &CreateDatasetRunItemRequest{
		RunName: "my-run",
	})
	if err == nil {
		t.Error("Expected validation error for missing datasetItemId")
	}

	// Missing runName
	_, err = client.Datasets().CreateRunItem(context.Background(), &CreateDatasetRunItemRequest{
		DatasetItemID: "item-123",
	})
	if err == nil {
		t.Error("Expected validation error for missing runName")
	}
}
