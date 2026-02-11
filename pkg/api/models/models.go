// Package models provides the Langfuse Models API client.
package models

import (
	"context"
	"fmt"
	"net/url"

	"github.com/jdziat/langfuse-go/pkg/http"
)

// Endpoint for the models API.
const Endpoint = "/models"

// Client handles model-related API operations.
// It uses generic result types to avoid circular dependencies with the root package.
type Client struct {
	http http.Doer
}

// New creates a new models client with the given HTTP doer.
func New(doer http.Doer) *Client {
	return &Client{http: doer}
}

// List retrieves a list of models.
// The result parameter should be a pointer to the response type (e.g., *ModelsListResponse).
func (c *Client) List(ctx context.Context, query url.Values, result any) error {
	return c.http.Get(ctx, Endpoint, query, result)
}

// Get retrieves a single model by ID.
// The result parameter should be a pointer to the model type (e.g., *Model).
func (c *Client) Get(ctx context.Context, modelID string, result any) error {
	return c.http.Get(ctx, fmt.Sprintf("%s/%s", Endpoint, modelID), nil, result)
}

// Create creates a new model.
// The body should be the request struct, result should be a pointer to the model type.
func (c *Client) Create(ctx context.Context, body any, result any) error {
	return c.http.Post(ctx, Endpoint, body, result)
}

// Delete deletes a model by ID.
func (c *Client) Delete(ctx context.Context, modelID string) error {
	return c.http.Delete(ctx, fmt.Sprintf("%s/%s", Endpoint, modelID), nil)
}
