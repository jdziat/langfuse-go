package langfuse

import (
	"context"
	"time"
)

// SessionsOption is a functional option for configuring the SessionsClient.
type SessionsOption func(*sessionsConfig)

type sessionsConfig struct {
	defaultTimeout time.Duration
}

// WithSessionsTimeout sets a default timeout for all session operations.
// This timeout is applied to context when not already set.
//
// Example:
//
//	sessions := client.SessionsWithOptions(
//	    langfuse.WithSessionsTimeout(10 * time.Second),
//	)
func WithSessionsTimeout(timeout time.Duration) SessionsOption {
	return func(c *sessionsConfig) {
		c.defaultTimeout = timeout
	}
}

// ConfiguredSessionsClient wraps SessionsClient with configured defaults.
type ConfiguredSessionsClient struct {
	*SessionsClient
	config *sessionsConfig
}

// List retrieves a list of sessions, applying configured defaults.
func (c *ConfiguredSessionsClient) List(ctx context.Context, params *SessionsListParams) (*SessionsListResponse, error) {
	ctx = c.applyTimeout(ctx)
	return c.SessionsClient.List(ctx, params)
}

// Get retrieves a session by ID, applying configured defaults.
func (c *ConfiguredSessionsClient) Get(ctx context.Context, sessionID string) (*Session, error) {
	ctx = c.applyTimeout(ctx)
	return c.SessionsClient.Get(ctx, sessionID)
}

// GetWithTraces retrieves a session with all its traces, applying configured defaults.
func (c *ConfiguredSessionsClient) GetWithTraces(ctx context.Context, sessionID string) (*SessionWithTraces, error) {
	ctx = c.applyTimeout(ctx)
	return c.SessionsClient.GetWithTraces(ctx, sessionID)
}

func (c *ConfiguredSessionsClient) applyTimeout(ctx context.Context) context.Context {
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
