package langfuse

import (
	"context"
	"time"
)

// ModelsOption is a functional option for configuring the ModelsClient.
type ModelsOption func(*modelsConfig)

type modelsConfig struct {
	defaultTimeout time.Duration
}

// WithModelsTimeout sets a default timeout for all model operations.
// This timeout is applied to context when not already set.
//
// Example:
//
//	models := client.ModelsWithOptions(
//	    langfuse.WithModelsTimeout(10 * time.Second),
//	)
func WithModelsTimeout(timeout time.Duration) ModelsOption {
	return func(c *modelsConfig) {
		c.defaultTimeout = timeout
	}
}

// ConfiguredModelsClient wraps ModelsClient with configured defaults.
type ConfiguredModelsClient struct {
	*ModelsClient
	config *modelsConfig
}

// List retrieves a list of models, applying configured defaults.
func (c *ConfiguredModelsClient) List(ctx context.Context, params *ModelsListParams) (*ModelsListResponse, error) {
	ctx = c.applyTimeout(ctx)
	return c.ModelsClient.List(ctx, params)
}

// Get retrieves a model by ID, applying configured defaults.
func (c *ConfiguredModelsClient) Get(ctx context.Context, modelID string) (*Model, error) {
	ctx = c.applyTimeout(ctx)
	return c.ModelsClient.Get(ctx, modelID)
}

// Create creates a new model definition, applying configured defaults.
func (c *ConfiguredModelsClient) Create(ctx context.Context, req *CreateModelRequest) (*Model, error) {
	ctx = c.applyTimeout(ctx)
	return c.ModelsClient.Create(ctx, req)
}

// Delete deletes a model by ID, applying configured defaults.
func (c *ConfiguredModelsClient) Delete(ctx context.Context, modelID string) error {
	ctx = c.applyTimeout(ctx)
	return c.ModelsClient.Delete(ctx, modelID)
}

func (c *ConfiguredModelsClient) applyTimeout(ctx context.Context) context.Context {
	if c.config.defaultTimeout > 0 {
		// Only apply timeout if context doesn't already have a deadline
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, c.config.defaultTimeout)
			_ = cancel // We're intentionally not canceling here as the caller owns the context lifecycle
		}
	}
	return ctx
}
