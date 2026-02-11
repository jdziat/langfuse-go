// Package traces provides the Langfuse Traces API client.
package traces

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jdziat/langfuse-go/pkg/http"
)

// Endpoint for the traces API.
const Endpoint = "/traces"

// Client handles trace-related API operations.
// It uses generic result types to avoid circular dependencies with the root package.
type Client struct {
	http http.Doer
}

// New creates a new traces client with the given HTTP doer.
func New(doer http.Doer) *Client {
	return &Client{http: doer}
}

// List retrieves a list of traces.
// The result parameter should be a pointer to the response type (e.g., *TracesListResponse).
func (c *Client) List(ctx context.Context, query url.Values, result any) error {
	return c.http.Get(ctx, Endpoint, query, result)
}

// Get retrieves a single trace by ID.
// The result parameter should be a pointer to the trace type (e.g., *Trace).
func (c *Client) Get(ctx context.Context, traceID string, result any) error {
	return c.http.Get(ctx, fmt.Sprintf("%s/%s", Endpoint, traceID), nil, result)
}

// Delete deletes a trace by ID.
func (c *Client) Delete(ctx context.Context, traceID string) error {
	return c.http.Delete(ctx, fmt.Sprintf("%s/%s", Endpoint, traceID), nil)
}
