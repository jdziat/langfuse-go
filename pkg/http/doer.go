// Package http provides HTTP utilities for the Langfuse SDK.
package http

import (
	"context"
	"net/url"
)

// Doer is an interface for making HTTP requests.
// This interface allows sub-clients to be decoupled from the main Client
// and enables dependency injection for testing.
type Doer interface {
	// Get performs an HTTP GET request.
	Get(ctx context.Context, path string, query url.Values, result any) error

	// Post performs an HTTP POST request.
	Post(ctx context.Context, path string, body, result any) error

	// Delete performs an HTTP DELETE request.
	Delete(ctx context.Context, path string, result any) error
}
