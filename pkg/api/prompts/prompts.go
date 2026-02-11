// Package prompts provides the Langfuse Prompts API client.
package prompts

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jdziat/langfuse-go/pkg/http"
)

// Endpoint for the prompts API (v2).
const Endpoint = "/v2/prompts"

// Client handles prompt-related API operations.
// It uses generic result types to avoid circular dependencies with the root package.
type Client struct {
	http http.Doer
}

// New creates a new prompts client with the given HTTP doer.
func New(doer http.Doer) *Client {
	return &Client{http: doer}
}

// List retrieves a list of prompts.
// The result parameter should be a pointer to the response type (e.g., *PromptsListResponse).
func (c *Client) List(ctx context.Context, query url.Values, result any) error {
	return c.http.Get(ctx, Endpoint, query, result)
}

// Get retrieves a prompt by name with optional query parameters.
// The result parameter should be a pointer to the prompt type (e.g., *Prompt).
func (c *Client) Get(ctx context.Context, name string, query url.Values, result any) error {
	return c.http.Get(ctx, fmt.Sprintf("%s/%s", Endpoint, name), query, result)
}

// Create creates a new prompt.
// The body should be the request struct, result should be a pointer to the prompt type.
func (c *Client) Create(ctx context.Context, body any, result any) error {
	return c.http.Post(ctx, Endpoint, body, result)
}
