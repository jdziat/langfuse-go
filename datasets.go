package langfuse

import (
	"context"
	"fmt"
	"net/url"
)

// DatasetsClient handles dataset-related API operations.
type DatasetsClient struct {
	client *Client
}

// DatasetsListParams represents parameters for listing datasets.
type DatasetsListParams struct {
	PaginationParams
}

// DatasetsListResponse represents the response from listing datasets.
type DatasetsListResponse struct {
	Data []Dataset    `json:"data"`
	Meta MetaResponse `json:"meta"`
}

// List retrieves a list of datasets.
func (c *DatasetsClient) List(ctx context.Context, params *DatasetsListParams) (*DatasetsListResponse, error) {
	query := url.Values{}
	if params != nil {
		query = params.PaginationParams.ToQuery()
	}

	var result DatasetsListResponse
	err := c.client.http.get(ctx, endpoints.Datasets, query, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a dataset by name.
func (c *DatasetsClient) Get(ctx context.Context, datasetName string) (*Dataset, error) {
	var result Dataset
	err := c.client.http.get(ctx, fmt.Sprintf("%s/%s", endpoints.Datasets, datasetName), nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateDatasetRequest represents a request to create a dataset.
type CreateDatasetRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Metadata    Metadata `json:"metadata,omitempty"`
}

// Create creates a new dataset.
func (c *DatasetsClient) Create(ctx context.Context, req *CreateDatasetRequest) (*Dataset, error) {
	if req == nil {
		return nil, ErrNilRequest
	}
	if req.Name == "" {
		return nil, NewValidationError("name", "dataset name is required")
	}

	var result Dataset
	err := c.client.http.post(ctx, endpoints.Datasets, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DatasetItemsListParams represents parameters for listing dataset items.
type DatasetItemsListParams struct {
	PaginationParams
	DatasetName         string
	SourceTraceID       string
	SourceObservationID string
}

// DatasetItemsListResponse represents the response from listing dataset items.
type DatasetItemsListResponse struct {
	Data []DatasetItem `json:"data"`
	Meta MetaResponse  `json:"meta"`
}

// ListItems retrieves items in a dataset.
func (c *DatasetsClient) ListItems(ctx context.Context, params *DatasetItemsListParams) (*DatasetItemsListResponse, error) {
	query := url.Values{}
	if params != nil {
		query = params.PaginationParams.ToQuery()
		if params.DatasetName != "" {
			query.Set("datasetName", params.DatasetName)
		}
		if params.SourceTraceID != "" {
			query.Set("sourceTraceId", params.SourceTraceID)
		}
		if params.SourceObservationID != "" {
			query.Set("sourceObservationId", params.SourceObservationID)
		}
	}

	var result DatasetItemsListResponse
	err := c.client.http.get(ctx, endpoints.DatasetItems, query, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetItem retrieves a dataset item by ID.
func (c *DatasetsClient) GetItem(ctx context.Context, itemID string) (*DatasetItem, error) {
	var result DatasetItem
	err := c.client.http.get(ctx, fmt.Sprintf("%s/%s", endpoints.DatasetItems, itemID), nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateDatasetItemRequest represents a request to create a dataset item.
type CreateDatasetItemRequest struct {
	DatasetName         string   `json:"datasetName"`
	Input               any      `json:"input,omitempty"`
	ExpectedOutput      any      `json:"expectedOutput,omitempty"`
	Metadata            Metadata `json:"metadata,omitempty"`
	SourceTraceID       string   `json:"sourceTraceId,omitempty"`
	SourceObservationID string   `json:"sourceObservationId,omitempty"`
	Status              string   `json:"status,omitempty"`
	ID                  string   `json:"id,omitempty"`
}

// CreateItem creates a new dataset item.
func (c *DatasetsClient) CreateItem(ctx context.Context, req *CreateDatasetItemRequest) (*DatasetItem, error) {
	if req == nil {
		return nil, ErrNilRequest
	}
	if req.DatasetName == "" {
		return nil, NewValidationError("datasetName", "dataset name is required")
	}

	var result DatasetItem
	err := c.client.http.post(ctx, endpoints.DatasetItems, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteItem deletes a dataset item by ID.
func (c *DatasetsClient) DeleteItem(ctx context.Context, itemID string) error {
	return c.client.http.delete(ctx, fmt.Sprintf("%s/%s", endpoints.DatasetItems, itemID), nil)
}

// DatasetRunsListResponse represents the response from listing dataset runs.
type DatasetRunsListResponse struct {
	Data []DatasetRun `json:"data"`
	Meta MetaResponse `json:"meta"`
}

// ListRuns retrieves runs for a dataset.
func (c *DatasetsClient) ListRuns(ctx context.Context, datasetName string, params *PaginationParams) (*DatasetRunsListResponse, error) {
	query := url.Values{}
	if params != nil {
		query = params.ToQuery()
	}

	var result DatasetRunsListResponse
	err := c.client.http.get(ctx, fmt.Sprintf("/datasets/%s/runs", datasetName), query, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRun retrieves a dataset run by name.
func (c *DatasetsClient) GetRun(ctx context.Context, datasetName string, runName string) (*DatasetRun, error) {
	var result DatasetRun
	err := c.client.http.get(ctx, fmt.Sprintf("/datasets/%s/runs/%s", datasetName, runName), nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteRun deletes a dataset run.
func (c *DatasetsClient) DeleteRun(ctx context.Context, datasetName string, runName string) error {
	return c.client.http.delete(ctx, fmt.Sprintf("/datasets/%s/runs/%s", datasetName, runName), nil)
}

// CreateDatasetRunItemRequest represents a request to create a dataset run item.
type CreateDatasetRunItemRequest struct {
	DatasetItemID  string   `json:"datasetItemId"`
	RunName        string   `json:"runName"`
	RunDescription string   `json:"runDescription,omitempty"`
	TraceID        string   `json:"traceId,omitempty"`
	ObservationID  string   `json:"observationId,omitempty"`
	Metadata       Metadata `json:"metadata,omitempty"`
}

// CreateRunItem creates a dataset run item (links a trace/observation to a dataset item).
func (c *DatasetsClient) CreateRunItem(ctx context.Context, req *CreateDatasetRunItemRequest) (*DatasetRunItem, error) {
	if req == nil {
		return nil, ErrNilRequest
	}
	if req.DatasetItemID == "" {
		return nil, NewValidationError("datasetItemId", "dataset item ID is required")
	}
	if req.RunName == "" {
		return nil, NewValidationError("runName", "run name is required")
	}

	var result DatasetRunItem
	err := c.client.http.post(ctx, endpoints.DatasetRuns, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
