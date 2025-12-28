package langfuse

import (
	"context"
	"fmt"
	"net/url"
)

// TracesClient handles trace-related API operations.
type TracesClient struct {
	client *Client
}

// TracesListParams represents parameters for listing traces.
type TracesListParams struct {
	PaginationParams
	FilterParams
}

// TracesListResponse represents the response from listing traces.
type TracesListResponse struct {
	Data []Trace      `json:"data"`
	Meta MetaResponse `json:"meta"`
}

// List retrieves a list of traces.
func (c *TracesClient) List(ctx context.Context, params *TracesListParams) (*TracesListResponse, error) {
	query := url.Values{}
	if params != nil {
		query = mergeQuery(params.PaginationParams.ToQuery(), params.FilterParams.ToQuery())
	}

	var result TracesListResponse
	err := c.client.http.get(ctx, endpoints.Traces, query, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single trace by ID.
func (c *TracesClient) Get(ctx context.Context, traceID string) (*Trace, error) {
	var result Trace
	err := c.client.http.get(ctx, fmt.Sprintf("%s/%s", endpoints.Traces, traceID), nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a trace by ID.
func (c *TracesClient) Delete(ctx context.Context, traceID string) error {
	return c.client.http.delete(ctx, fmt.Sprintf("%s/%s", endpoints.Traces, traceID), nil)
}
