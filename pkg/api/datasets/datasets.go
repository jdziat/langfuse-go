// Package datasets provides the Langfuse Datasets API client.
package datasets

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jdziat/langfuse-go/pkg/http"
)

// Endpoints for the datasets API.
const (
	DatasetsEndpoint     = "/v2/datasets"
	DatasetItemsEndpoint = "/dataset-items"
	DatasetRunsEndpoint  = "/dataset-run-items"
)

// Client handles dataset-related API operations.
// It uses generic result types to avoid circular dependencies with the root package.
type Client struct {
	http http.Doer
}

// New creates a new datasets client with the given HTTP doer.
func New(doer http.Doer) *Client {
	return &Client{http: doer}
}

// List retrieves a list of datasets.
// The result parameter should be a pointer to the response type (e.g., *DatasetsListResponse).
func (c *Client) List(ctx context.Context, query url.Values, result any) error {
	return c.http.Get(ctx, DatasetsEndpoint, query, result)
}

// Get retrieves a dataset by name.
// The result parameter should be a pointer to the dataset type (e.g., *Dataset).
func (c *Client) Get(ctx context.Context, datasetName string, result any) error {
	return c.http.Get(ctx, fmt.Sprintf("%s/%s", DatasetsEndpoint, datasetName), nil, result)
}

// Create creates a new dataset.
// The body should be the request struct, result should be a pointer to the dataset type.
func (c *Client) Create(ctx context.Context, body any, result any) error {
	return c.http.Post(ctx, DatasetsEndpoint, body, result)
}

// ListItems retrieves dataset items.
// The result parameter should be a pointer to the response type (e.g., *DatasetItemsListResponse).
func (c *Client) ListItems(ctx context.Context, query url.Values, result any) error {
	return c.http.Get(ctx, DatasetItemsEndpoint, query, result)
}

// GetItem retrieves a dataset item by ID.
// The result parameter should be a pointer to the item type (e.g., *DatasetItem).
func (c *Client) GetItem(ctx context.Context, itemID string, result any) error {
	return c.http.Get(ctx, fmt.Sprintf("%s/%s", DatasetItemsEndpoint, itemID), nil, result)
}

// CreateItem creates a new dataset item.
// The body should be the request struct, result should be a pointer to the item type.
func (c *Client) CreateItem(ctx context.Context, body any, result any) error {
	return c.http.Post(ctx, DatasetItemsEndpoint, body, result)
}

// DeleteItem deletes a dataset item by ID.
func (c *Client) DeleteItem(ctx context.Context, itemID string) error {
	return c.http.Delete(ctx, fmt.Sprintf("%s/%s", DatasetItemsEndpoint, itemID), nil)
}

// ListRuns retrieves runs for a dataset.
// The result parameter should be a pointer to the response type (e.g., *DatasetRunsListResponse).
func (c *Client) ListRuns(ctx context.Context, datasetName string, query url.Values, result any) error {
	return c.http.Get(ctx, fmt.Sprintf("/datasets/%s/runs", datasetName), query, result)
}

// GetRun retrieves a dataset run by name.
// The result parameter should be a pointer to the run type (e.g., *DatasetRun).
func (c *Client) GetRun(ctx context.Context, datasetName, runName string, result any) error {
	return c.http.Get(ctx, fmt.Sprintf("/datasets/%s/runs/%s", datasetName, runName), nil, result)
}

// DeleteRun deletes a dataset run.
func (c *Client) DeleteRun(ctx context.Context, datasetName, runName string) error {
	return c.http.Delete(ctx, fmt.Sprintf("/datasets/%s/runs/%s", datasetName, runName), nil)
}

// CreateRunItem creates a dataset run item.
// The body should be the request struct, result should be a pointer to the run item type.
func (c *Client) CreateRunItem(ctx context.Context, body any, result any) error {
	return c.http.Post(ctx, DatasetRunsEndpoint, body, result)
}
