package langfuse

// This file re-exports types from pkg/ packages for backward compatibility.
// It provides "Pkg" prefixed type aliases for accessing pkg/ types directly.
//
// Note: Many types and constants are already defined at root level and will be
// cleaned up in Task #28. This facade focuses on providing access to pkg/ types
// without breaking existing code.

import (
	pkgconfig "github.com/jdziat/langfuse-go/pkg/config"
	pkgerrors "github.com/jdziat/langfuse-go/pkg/errors"
	pkghttp "github.com/jdziat/langfuse-go/pkg/http"
	pkgingestion "github.com/jdziat/langfuse-go/pkg/ingestion"
)

// Re-exports from pkg/config
// Note: Region types are already aliased in regions.go
// Note: Env constants and default config constants are already defined in env.go and config.go

// Environment helper functions from pkg/config.
// These are not duplicates - they're explicit re-exports.
var (
	// GetEnvString returns the value of an environment variable or a default.
	GetEnvString = pkgconfig.GetEnvString
	// GetEnvBool returns true if the env var is "true" or "1".
	GetEnvBool = pkgconfig.GetEnvBool
	// GetEnvRegion returns the region from environment or default.
	GetEnvRegion = pkgconfig.GetEnvRegion
)

// Re-exports from pkg/errors
// Note: Root-level error types exist in errors.go, errors_api.go, etc.
// These "Pkg" prefixed aliases provide access to pkg/ versions.

// Type aliases for async error handling (pkg/ versions with Pkg prefix).
type (
	// PkgAsyncError is the pkg/errors version of AsyncError.
	PkgAsyncError = pkgerrors.AsyncError
	// PkgAsyncErrorOperation identifies the type of async operation that failed.
	PkgAsyncErrorOperation = pkgerrors.AsyncErrorOperation
	// PkgAsyncErrorHandler provides structured async error handling.
	PkgAsyncErrorHandler = pkgerrors.AsyncErrorHandler
	// PkgAsyncErrorConfig configures the AsyncErrorHandler.
	PkgAsyncErrorConfig = pkgerrors.AsyncErrorConfig
	// PkgAsyncErrorStats contains statistics about async error handling.
	PkgAsyncErrorStats = pkgerrors.AsyncErrorStats
)

// Type aliases for ingestion result types (pkg/ versions).
type (
	// PkgIngestionResult represents the result of a batch ingestion request.
	PkgIngestionResult = pkgerrors.IngestionResult
	// PkgIngestionSuccess represents a successful ingestion event.
	PkgIngestionSuccess = pkgerrors.IngestionSuccess
	// PkgIngestionError represents an error that occurred during batch ingestion.
	PkgIngestionError = pkgerrors.IngestionError
)

// Type aliases for error types (pkg/ versions).
type (
	// PkgCompilationError represents errors during prompt compilation.
	PkgCompilationError = pkgerrors.CompilationError
	// PkgShutdownError represents an error that occurred during client shutdown.
	PkgShutdownError = pkgerrors.ShutdownError
	// PkgCodedError is an interface for errors that have an error code.
	PkgCodedError = pkgerrors.CodedError
	// PkgLangfuseError is the common interface for all SDK errors.
	PkgLangfuseError = pkgerrors.LangfuseError
	// PkgAPIError represents an error response from the Langfuse API.
	PkgAPIError = pkgerrors.APIError
	// PkgValidationError represents a validation error for a request.
	PkgValidationError = pkgerrors.ValidationError
	// PkgErrorCode represents a category of error for metrics and logging.
	PkgErrorCode = pkgerrors.ErrorCode
)

// Re-exports from pkg/http
// Note: Root-level retry and circuit breaker types exist in retry.go and circuitbreaker.go.
// These "Pkg" prefixed aliases provide access to pkg/ versions.

// Retry strategy types (pkg/ versions).
type (
	// PkgRetryStrategy defines how failed requests are retried.
	PkgRetryStrategy = pkghttp.RetryStrategy
	// PkgRetryStrategyWithError allows retry delay to be influenced by the error.
	PkgRetryStrategyWithError = pkghttp.RetryStrategyWithError
	// PkgExponentialBackoff implements exponential backoff with optional jitter.
	PkgExponentialBackoff = pkghttp.ExponentialBackoff
	// PkgNoRetry is a retry strategy that never retries.
	PkgNoRetry = pkghttp.NoRetry
	// PkgFixedDelay is a retry strategy with a fixed delay between retries.
	PkgFixedDelay = pkghttp.FixedDelay
	// PkgLinearBackoff is a retry strategy with linearly increasing delays.
	PkgLinearBackoff = pkghttp.LinearBackoff
	// PkgRetryableError is an interface for errors that know if they're retryable.
	PkgRetryableError = pkghttp.RetryableError
	// PkgRetryAfterError is an interface for errors with retry-after hints.
	PkgRetryAfterError = pkghttp.RetryAfterError
)

// Circuit breaker types (pkg/ versions).
type (
	// PkgCircuitBreaker implements the circuit breaker pattern for fault tolerance.
	PkgCircuitBreaker = pkghttp.CircuitBreaker
	// PkgCircuitBreakerConfig configures the circuit breaker behavior.
	PkgCircuitBreakerConfig = pkghttp.CircuitBreakerConfig
	// PkgCircuitBreakerOption configures a circuit breaker.
	PkgCircuitBreakerOption = pkghttp.CircuitBreakerOption
	// PkgCircuitState represents the state of a circuit breaker.
	PkgCircuitState = pkghttp.CircuitState
)

// Pagination types (pkg/ versions).
type (
	// PkgPaginationParams represents pagination parameters for list requests.
	PkgPaginationParams = pkghttp.PaginationParams
	// PkgPaginatedResponse represents a paginated response.
	PkgPaginatedResponse = pkghttp.PaginatedResponse
	// PkgMetaResponse represents pagination metadata.
	PkgMetaResponse = pkghttp.MetaResponse
	// PkgFilterParams represents common filter parameters.
	PkgFilterParams = pkghttp.FilterParams
)

// Re-exports from pkg/ingestion
// Note: Root-level backpressure and ID types exist in backpressure.go and id.go.
// These "Pkg" prefixed aliases provide access to pkg/ versions.

// Backpressure types (pkg/ versions).
type (
	// PkgBackpressureLevel indicates the severity of queue backpressure.
	PkgBackpressureLevel = pkgingestion.BackpressureLevel
	// PkgBackpressureThreshold defines when backpressure levels are triggered.
	PkgBackpressureThreshold = pkgingestion.BackpressureThreshold
	// PkgQueueState represents the current state of the event queue.
	PkgQueueState = pkgingestion.QueueState
	// PkgBackpressureCallback is called when backpressure level changes.
	PkgBackpressureCallback = pkgingestion.BackpressureCallback
	// PkgQueueMonitor monitors queue state and signals backpressure.
	PkgQueueMonitor = pkgingestion.QueueMonitor
	// PkgQueueMonitorConfig configures the QueueMonitor.
	PkgQueueMonitorConfig = pkgingestion.QueueMonitorConfig
	// PkgQueueMonitorStats contains statistics about queue monitoring.
	PkgQueueMonitorStats = pkgingestion.QueueMonitorStats
	// PkgBackpressureHandler provides a higher-level API for handling backpressure.
	PkgBackpressureHandler = pkgingestion.BackpressureHandler
	// PkgBackpressureHandlerConfig configures the BackpressureHandler.
	PkgBackpressureHandlerConfig = pkgingestion.BackpressureHandlerConfig
	// PkgBackpressureHandlerStats contains statistics about backpressure handling.
	PkgBackpressureHandlerStats = pkgingestion.BackpressureHandlerStats
	// PkgBackpressureDecision represents the decision made by the handler.
	PkgBackpressureDecision = pkgingestion.BackpressureDecision
)

// Constructors and helpers from pkg/ packages
// These provide access to pkg/ implementations without conflicts.

// From pkg/http
var (
	// NewPkgExponentialBackoff creates a new exponential backoff strategy.
	NewPkgExponentialBackoff = pkghttp.NewExponentialBackoff
	// NewPkgFixedDelay creates a new fixed delay retry strategy.
	NewPkgFixedDelay = pkghttp.NewFixedDelay
	// NewPkgLinearBackoff creates a new linear backoff retry strategy.
	NewPkgLinearBackoff = pkghttp.NewLinearBackoff
	// NewPkgCircuitBreaker creates a new circuit breaker.
	NewPkgCircuitBreaker = pkghttp.NewCircuitBreaker
	// NewPkgCircuitBreakerWithOptions creates a circuit breaker with functional options.
	NewPkgCircuitBreakerWithOptions = pkghttp.NewCircuitBreakerWithOptions
	// PkgMergeQuery merges multiple url.Values into one.
	PkgMergeQuery = pkghttp.MergeQuery
)

// From pkg/ingestion
var (
	// NewPkgQueueMonitor creates a new queue monitor.
	NewPkgQueueMonitor = pkgingestion.NewQueueMonitor
	// NewPkgBackpressureHandler creates a new backpressure handler.
	NewPkgBackpressureHandler = pkgingestion.NewBackpressureHandler
	// PkgUUID generates a random UUID v4.
	PkgUUID = pkgingestion.UUID
	// PkgGenerateID generates a random UUID-like ID.
	PkgGenerateID = pkgingestion.GenerateID
	// PkgIsValidUUID checks if a string is a valid UUID format.
	PkgIsValidUUID = pkgingestion.IsValidUUID
)

// From pkg/errors
var (
	// NewPkgValidationError creates a new validation error.
	NewPkgValidationError = pkgerrors.NewValidationError
	// NewPkgAsyncError creates a new async error.
	NewPkgAsyncError = pkgerrors.NewAsyncError
	// NewPkgAsyncErrorHandler creates a new async error handler.
	NewPkgAsyncErrorHandler = pkgerrors.NewAsyncErrorHandler
	// PkgWrapAsyncError wraps an error in an AsyncError if it isn't already.
	PkgWrapAsyncError = pkgerrors.WrapAsyncError
	// PkgIsRetryable returns true if the error represents a retryable condition.
	PkgIsRetryable = pkgerrors.IsRetryable
	// PkgAsAPIError extracts an APIError from the error chain.
	PkgAsAPIError = pkgerrors.AsAPIError
	// PkgAsValidationError extracts a ValidationError from the error chain.
	PkgAsValidationError = pkgerrors.AsValidationError
	// PkgAsIngestionError extracts an IngestionError from the error chain.
	PkgAsIngestionError = pkgerrors.AsIngestionError
	// PkgAsShutdownError extracts a ShutdownError from the error chain.
	PkgAsShutdownError = pkgerrors.AsShutdownError
	// PkgAsCompilationError extracts a CompilationError from the error chain.
	PkgAsCompilationError = pkgerrors.AsCompilationError
	// PkgAsAsyncError extracts an AsyncError from the error chain.
	PkgAsAsyncError = pkgerrors.AsAsyncError
	// PkgRetryAfter returns the suggested retry delay from a rate limit error.
	PkgRetryAfter = pkgerrors.RetryAfter
	// PkgErrorCodeOf returns the error code for an error.
	PkgErrorCodeOf = pkgerrors.ErrorCodeOf
	// PkgWrapError wraps an error with additional context.
	PkgWrapError = pkgerrors.WrapError
	// PkgWrapErrorf wraps an error with a formatted message.
	PkgWrapErrorf = pkgerrors.WrapErrorf
)
