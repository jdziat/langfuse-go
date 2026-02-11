// Package scores provides the Langfuse Scores API client.
package scores

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jdziat/langfuse-go/pkg/http"
)

// Endpoint for the scores API.
const Endpoint = "/scores"

// Client handles score-related API operations.
// It uses generic result types to avoid circular dependencies with the root package.
type Client struct {
	http http.Doer
}

// New creates a new scores client with the given HTTP doer.
func New(doer http.Doer) *Client {
	return &Client{http: doer}
}

// List retrieves a list of scores.
// The result parameter should be a pointer to the response type (e.g., *ScoresListResponse).
func (c *Client) List(ctx context.Context, query url.Values, result any) error {
	return c.http.Get(ctx, Endpoint, query, result)
}

// Get retrieves a single score by ID.
// The result parameter should be a pointer to the score type (e.g., *Score).
func (c *Client) Get(ctx context.Context, scoreID string, result any) error {
	return c.http.Get(ctx, fmt.Sprintf("%s/%s", Endpoint, scoreID), nil, result)
}

// Create creates a new score.
// The body should be the request struct, result should be a pointer to the score type.
func (c *Client) Create(ctx context.Context, body any, result any) error {
	return c.http.Post(ctx, Endpoint, body, result)
}

// Delete deletes a score by ID.
func (c *Client) Delete(ctx context.Context, scoreID string) error {
	return c.http.Delete(ctx, fmt.Sprintf("%s/%s", Endpoint, scoreID), nil)
}
