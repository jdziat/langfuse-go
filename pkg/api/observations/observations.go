// Package observations provides the Langfuse Observations API client.
package observations

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jdziat/langfuse-go/pkg/http"
)

// Endpoint for the observations API.
const Endpoint = "/observations"

// Client handles observation-related API operations.
// It uses generic result types to avoid circular dependencies with the root package.
type Client struct {
	http http.Doer
}

// New creates a new observations client with the given HTTP doer.
func New(doer http.Doer) *Client {
	return &Client{http: doer}
}

// List retrieves a list of observations.
// The result parameter should be a pointer to the response type (e.g., *ObservationsListResponse).
func (c *Client) List(ctx context.Context, query url.Values, result any) error {
	return c.http.Get(ctx, Endpoint, query, result)
}

// Get retrieves a single observation by ID.
// The result parameter should be a pointer to the observation type (e.g., *Observation).
func (c *Client) Get(ctx context.Context, observationID string, result any) error {
	return c.http.Get(ctx, fmt.Sprintf("%s/%s", Endpoint, observationID), nil, result)
}
