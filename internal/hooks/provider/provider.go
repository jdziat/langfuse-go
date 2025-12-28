// Package provider defines the LLM provider interface and implementations.
package provider

import (
	"context"
)

// Message represents a chat message.
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Request represents an LLM completion request.
type Request struct {
	Messages  []Message
	MaxTokens int
}

// Response represents an LLM completion response.
type Response struct {
	Content string
}

// Provider defines the interface for LLM providers.
type Provider interface {
	// Complete sends a completion request and returns the response.
	Complete(ctx context.Context, req *Request) (*Response, error)

	// Name returns the provider name.
	Name() string
}
