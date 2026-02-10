package client

import (
	pkghttp "github.com/jdziat/langfuse-go/pkg/http"
	pkgid "github.com/jdziat/langfuse-go/pkg/id"
)

// Re-export ID generation types from pkg/id.
// Note: IDGenerationMode is defined in config.go.
type (
	IDStats           = pkgid.IDStats
	IDGenerator       = pkgid.IDGenerator
	IDGeneratorConfig = pkgid.IDGeneratorConfig
)

// Doer is an interface for making HTTP requests.
// Re-exported from pkg/http for convenience.
type Doer = pkghttp.Doer

// Re-export circuit breaker types from pkg/http.
type (
	CircuitState          = pkghttp.CircuitState
	CircuitBreaker        = pkghttp.CircuitBreaker
	CircuitBreakerConfig  = pkghttp.CircuitBreakerConfig
	RetryStrategy         = pkghttp.RetryStrategy
	RetryStrategyWithError = pkghttp.RetryStrategyWithError
	ExponentialBackoff    = pkghttp.ExponentialBackoff
	FixedDelay            = pkghttp.FixedDelay
	LinearBackoff         = pkghttp.LinearBackoff
	NoRetry               = pkghttp.NoRetry
)

// Circuit breaker state constants.
const (
	CircuitClosed   = pkghttp.CircuitClosed
	CircuitOpen     = pkghttp.CircuitOpen
	CircuitHalfOpen = pkghttp.CircuitHalfOpen
)

// ErrCircuitOpen is returned when the circuit breaker is open.
var ErrCircuitOpen = pkghttp.ErrCircuitOpen

// CircuitBreakerState returns the current circuit breaker state.
func (c *Client) CircuitBreakerState() CircuitState {
	if c.http.circuitBreaker == nil {
		return CircuitClosed
	}
	return c.http.circuitBreaker.State()
}

// IsUnderBackpressure returns true if the client is experiencing backpressure.
func (c *Client) IsUnderBackpressure() bool {
	if c.backpressure == nil {
		return false
	}
	return c.backpressure.Monitor().Level() >= BackpressureWarning
}

// IDStats returns current ID generation statistics.
func (c *Client) IDStats() IDStats {
	if c.idGenerator == nil {
		return IDStats{}
	}
	return c.idGenerator.Stats()
}
