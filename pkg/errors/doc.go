// Package errors provides error types and handling for the Langfuse Go SDK.
//
// This package defines all error types used throughout the SDK, including:
//   - Base error interfaces and types
//   - API errors with status codes and retry logic
//   - Async errors for background operations
//   - Validation errors
//   - Helper functions for error handling
//
// # Error Types
//
// The package provides several specialized error types:
//
//   - APIError: Represents errors from the Langfuse API with HTTP status codes
//   - ValidationError: Represents validation failures for SDK inputs
//   - AsyncError: Represents errors in background processing
//   - ShutdownError: Represents errors during client shutdown
//   - CompilationError: Represents errors during prompt compilation
//
// # Error Handling
//
// All SDK errors implement the LangfuseError interface, which provides
// consistent error information:
//
//	var langfuseErr errors.LangfuseError
//	if stdErrors.As(err, &langfuseErr) {
//	    if langfuseErr.IsRetryable() {
//	        // Retry the operation
//	    }
//	    log.Printf("Error code: %s", langfuseErr.Code())
//	}
//
// Helper functions are provided for common error operations:
//
//	if apiErr, ok := errors.AsAPIError(err); ok {
//	    fmt.Printf("API error %d: %s", apiErr.StatusCode, apiErr.Message)
//	}
//
//	if errors.IsRetryable(err) {
//	    time.Sleep(errors.RetryAfter(err))
//	    // Retry the operation
//	}
//
// # Sentinel Errors
//
// The package defines sentinel errors for common scenarios:
//
//   - ErrMissingPublicKey, ErrMissingSecretKey: Configuration errors
//   - ErrClientClosed: Operations on closed client
//   - ErrNotFound, ErrUnauthorized, ErrRateLimited: API errors
//
// Use errors.Is() for sentinel error comparison:
//
//	if stdErrors.Is(err, errors.ErrRateLimited) {
//	    // Handle rate limiting
//	}
package errors
