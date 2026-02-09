package langfuse

import (
	"errors"
	"fmt"
	"time"
)

// IsRetryable returns true if the error represents a retryable condition.
// This works with any error type in the SDK.
//
// Retryable conditions include:
//   - Rate limiting (429)
//   - Server errors (5xx)
//   - Network timeouts
//   - Circuit breaker open (may close soon)
//
// Example:
//
//	if langfuse.IsRetryable(err) {
//	    time.Sleep(langfuse.RetryAfter(err))
//	    // Retry the operation
//	}
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check if error implements LangfuseError
	var langfuseErr LangfuseError
	if errors.As(err, &langfuseErr) {
		return langfuseErr.IsRetryable()
	}

	// Check specific error types
	if apiErr, ok := AsAPIError(err); ok {
		return apiErr.IsRetryable()
	}

	// Network errors are generally retryable
	if errors.Is(err, ErrCircuitOpen) {
		return true
	}

	return false
}

// AsAPIError extracts an APIError from the error chain.
// Returns the APIError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
//
// Example:
//
//	if apiErr, ok := langfuse.AsAPIError(err); ok {
//	    log.Printf("API error %d: %s", apiErr.StatusCode, apiErr.Message)
//	}
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// AsValidationError extracts a ValidationError from the error chain.
// Returns the ValidationError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
func AsValidationError(err error) (*ValidationError, bool) {
	var valErr *ValidationError
	if errors.As(err, &valErr) {
		return valErr, true
	}
	return nil, false
}

// AsIngestionError extracts an IngestionError from the error chain.
// Returns the IngestionError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
func AsIngestionError(err error) (*IngestionError, bool) {
	var ingErr *IngestionError
	if errors.As(err, &ingErr) {
		return ingErr, true
	}
	return nil, false
}

// AsShutdownError extracts a ShutdownError from the error chain.
// Returns the ShutdownError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
func AsShutdownError(err error) (*ShutdownError, bool) {
	var shutdownErr *ShutdownError
	if errors.As(err, &shutdownErr) {
		return shutdownErr, true
	}
	return nil, false
}

// AsCompilationError extracts a CompilationError from the error chain.
// Returns the CompilationError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
func AsCompilationError(err error) (*CompilationError, bool) {
	var compErr *CompilationError
	if errors.As(err, &compErr) {
		return compErr, true
	}
	return nil, false
}

// RetryAfter returns the suggested retry delay from a rate limit error.
// Returns 0 if the error is not a rate limit error or has no Retry-After hint.
func RetryAfter(err error) time.Duration {
	if apiErr, ok := AsAPIError(err); ok {
		return apiErr.RetryAfter
	}
	return 0
}

// ErrorCodeOf returns the error code for an error.
// It checks if the error implements CodedError, then falls back to
// inferring the code from the error type.
func ErrorCodeOf(err error) ErrorCode {
	if err == nil {
		return ""
	}

	// Check if error implements CodedError
	var coded CodedError
	if errors.As(err, &coded) {
		return coded.Code()
	}

	// Infer code from error type
	switch {
	case errors.Is(err, ErrMissingPublicKey),
		errors.Is(err, ErrMissingSecretKey),
		errors.Is(err, ErrMissingBaseURL),
		errors.Is(err, ErrInvalidConfig):
		return ErrCodeConfig

	case errors.Is(err, ErrClientClosed),
		errors.Is(err, ErrShutdownTimeout):
		return ErrCodeShutdown

	case errors.Is(err, ErrContextCancelled):
		return ErrCodeTimeout

	case errors.Is(err, ErrCircuitOpen):
		return ErrCodeNetwork
	}

	// Check for API errors
	if apiErr, ok := AsAPIError(err); ok {
		switch {
		case apiErr.IsUnauthorized(), apiErr.IsForbidden():
			return ErrCodeAuth
		case apiErr.IsRateLimited():
			return ErrCodeRateLimit
		default:
			return ErrCodeAPI
		}
	}

	// Check for validation errors
	if _, ok := AsValidationError(err); ok {
		return ErrCodeValidation
	}

	// Check for shutdown errors
	if _, ok := AsShutdownError(err); ok {
		return ErrCodeShutdown
	}

	return ErrCodeInternal
}

// WrapError wraps an error with additional context.
// It returns nil if err is nil.
//
// Example:
//
//	if err := doSomething(); err != nil {
//	    return langfuse.WrapError(err, "failed to process trace")
//	}
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("langfuse: %s: %w", message, err)
}

// WrapErrorf wraps an error with a formatted message.
// It returns nil if err is nil.
//
// Example:
//
//	if err := doSomething(id); err != nil {
//	    return langfuse.WrapErrorf(err, "failed to process trace %s", id)
//	}
func WrapErrorf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	message := fmt.Sprintf(format, args...)
	return fmt.Errorf("langfuse: %s: %w", message, err)
}

// Deprecated: IsShutdownError is deprecated, use AsShutdownError instead.
func IsShutdownError(err error) (*ShutdownError, bool) {
	return AsShutdownError(err)
}

// Deprecated: IsCompilationError is deprecated, use AsCompilationError instead.
func IsCompilationError(err error) (*CompilationError, bool) {
	return AsCompilationError(err)
}
