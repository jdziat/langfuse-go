package langfuse

import (
	"context"
	"fmt"
	"net/url"
)

// ObservationsClient handles observation-related API operations.
type ObservationsClient struct {
	client *Client
}

// ObservationsListParams represents parameters for listing observations.
type ObservationsListParams struct {
	PaginationParams
	FilterParams
	ParentObservationID string
}

// ObservationsListResponse represents the response from listing observations.
type ObservationsListResponse struct {
	Data []Observation `json:"data"`
	Meta MetaResponse  `json:"meta"`
}

// List retrieves a list of observations.
func (c *ObservationsClient) List(ctx context.Context, params *ObservationsListParams) (*ObservationsListResponse, error) {
	query := url.Values{}
	if params != nil {
		query = mergeQuery(params.PaginationParams.ToQuery(), params.FilterParams.ToQuery())
		if params.ParentObservationID != "" {
			query.Set("parentObservationId", params.ParentObservationID)
		}
	}

	var result ObservationsListResponse
	err := c.client.http.get(ctx, endpoints.Observations, query, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single observation by ID.
func (c *ObservationsClient) Get(ctx context.Context, observationID string) (*Observation, error) {
	var result Observation
	err := c.client.http.get(ctx, fmt.Sprintf("%s/%s", endpoints.Observations, observationID), nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListByTrace retrieves all observations for a specific trace.
func (c *ObservationsClient) ListByTrace(ctx context.Context, traceID string, params *PaginationParams) (*ObservationsListResponse, error) {
	query := url.Values{}
	query.Set("traceId", traceID)
	if params != nil {
		query = mergeQuery(query, params.ToQuery())
	}

	var result ObservationsListResponse
	err := c.client.http.get(ctx, endpoints.Observations, query, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// ListSpans retrieves all spans.
func (c *ObservationsClient) ListSpans(ctx context.Context, params *ObservationsListParams) (*ObservationsListResponse, error) {
	if params == nil {
		params = &ObservationsListParams{}
	}
	params.Type = string(ObservationTypeSpan)
	return c.List(ctx, params)
}

// ListGenerations retrieves all generations.
func (c *ObservationsClient) ListGenerations(ctx context.Context, params *ObservationsListParams) (*ObservationsListResponse, error) {
	if params == nil {
		params = &ObservationsListParams{}
	}
	params.Type = string(ObservationTypeGeneration)
	return c.List(ctx, params)
}

// ListEvents retrieves all events.
func (c *ObservationsClient) ListEvents(ctx context.Context, params *ObservationsListParams) (*ObservationsListResponse, error) {
	if params == nil {
		params = &ObservationsListParams{}
	}
	params.Type = string(ObservationTypeEvent)
	return c.List(ctx, params)
}
