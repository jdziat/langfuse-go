// Package sessions provides the Langfuse Sessions API client.
package sessions

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jdziat/langfuse-go/pkg/http"
)

// Endpoint for the sessions API.
const Endpoint = "/sessions"

// Client handles session-related API operations.
// It uses generic result types to avoid circular dependencies with the root package.
type Client struct {
	http http.Doer
}

// New creates a new sessions client with the given HTTP doer.
func New(doer http.Doer) *Client {
	return &Client{http: doer}
}

// List retrieves a list of sessions.
// The result parameter should be a pointer to the response type (e.g., *SessionsListResponse).
func (c *Client) List(ctx context.Context, query url.Values, result any) error {
	return c.http.Get(ctx, Endpoint, query, result)
}

// Get retrieves a single session by ID.
// The result parameter should be a pointer to the session type (e.g., *Session).
func (c *Client) Get(ctx context.Context, sessionID string, result any) error {
	return c.http.Get(ctx, fmt.Sprintf("%s/%s", Endpoint, sessionID), nil, result)
}
