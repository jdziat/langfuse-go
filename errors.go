package langfuse

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorCode represents a category of error for metrics and logging.
type ErrorCode string

// Error codes for categorization.
const (
	ErrCodeConfig       ErrorCode = "CONFIG"       // Configuration errors
	ErrCodeValidation   ErrorCode = "VALIDATION"   // Request validation errors
	ErrCodeNetwork      ErrorCode = "NETWORK"      // Network/connection errors
	ErrCodeAPI          ErrorCode = "API"          // API response errors
	ErrCodeAuth         ErrorCode = "AUTH"         // Authentication/authorization errors
	ErrCodeRateLimit    ErrorCode = "RATE_LIMIT"   // Rate limiting errors
	ErrCodeTimeout      ErrorCode = "TIMEOUT"      // Timeout errors
	ErrCodeInternal     ErrorCode = "INTERNAL"     // Internal SDK errors
	ErrCodeShutdown     ErrorCode = "SHUTDOWN"     // Shutdown-related errors
	ErrCodeBackpressure ErrorCode = "BACKPRESSURE" // Backpressure/queue full errors
)

// LangfuseError is the common interface for all SDK errors.
// Use this interface to handle errors generically while still accessing
// error-specific information.
//
// Example:
//
//	var langfuseErr LangfuseError
//	if errors.As(err, &langfuseErr) {
//	    if langfuseErr.IsRetryable() {
//	        // Retry the operation
//	    }
//	    log.Printf("Error code: %s, request ID: %s", langfuseErr.Code(), langfuseErr.GetRequestID())
//	}
type LangfuseError interface {
	error

	// Code returns a machine-readable error code for categorization.
	Code() ErrorCode

	// IsRetryable returns true if the operation can be retried.
	IsRetryable() bool

	// GetRequestID returns the server request ID, if available.
	// Returns empty string if not applicable.
	GetRequestID() string
}

// Sentinel errors for configuration validation.
var (
	ErrMissingPublicKey = errors.New("langfuse: public key is required")
	ErrMissingSecretKey = errors.New("langfuse: secret key is required")
	ErrMissingBaseURL   = errors.New("langfuse: base URL is required")
	ErrInvalidConfig    = errors.New("langfuse: invalid configuration")
	ErrClientClosed     = errors.New("langfuse: client is closed")
	ErrNilRequest       = errors.New("langfuse: request cannot be nil")
)

// Additional sentinel errors for common scenarios.
// Note: ErrCircuitOpen is defined in circuitbreaker.go.
var (
	ErrPromptNotFound   = errors.New("langfuse: prompt not found")
	ErrDatasetNotFound  = errors.New("langfuse: dataset not found")
	ErrTraceNotFound    = errors.New("langfuse: trace not found")
	ErrEmptyBatch       = errors.New("langfuse: batch is empty")
	ErrBatchTooLarge    = errors.New("langfuse: batch exceeds maximum size")
	ErrContextCancelled = errors.New("langfuse: context was cancelled")
	ErrShutdownTimeout  = errors.New("langfuse: shutdown timed out")
)

// ShutdownError represents an error that occurred during client shutdown.
// It provides context about what was lost during the shutdown failure.
type ShutdownError struct {
	Cause         error // The underlying error (e.g., context deadline exceeded)
	PendingEvents int   // Number of events that may not have been sent
	Message       string
}

// Error implements the error interface.
func (e *ShutdownError) Error() string {
	if e.PendingEvents > 0 {
		return fmt.Sprintf("langfuse: shutdown failed (%s): %d pending events may be lost", e.Message, e.PendingEvents)
	}
	return fmt.Sprintf("langfuse: shutdown failed: %s", e.Message)
}

// Unwrap returns the underlying error for error chain support.
func (e *ShutdownError) Unwrap() error {
	return e.Cause
}

// Code returns the error code for the shutdown error.
// Implements the LangfuseError interface.
func (e *ShutdownError) Code() ErrorCode {
	return ErrCodeShutdown
}

// IsRetryable returns false for shutdown errors (shutdown cannot be retried).
// Implements the LangfuseError interface.
func (e *ShutdownError) IsRetryable() bool {
	return false
}

// GetRequestID returns an empty string (shutdown errors don't have request IDs).
// Implements the LangfuseError interface.
func (e *ShutdownError) GetRequestID() string {
	return ""
}

// Ensure ShutdownError implements LangfuseError.
var _ LangfuseError = (*ShutdownError)(nil)

// CompilationError represents errors during prompt compilation.
// It collects all errors encountered during compilation, allowing
// partial results to be returned alongside the errors.
type CompilationError struct {
	Errors []error // All errors encountered during compilation
}

// Error implements the error interface.
func (e *CompilationError) Error() string {
	if len(e.Errors) == 0 {
		return "langfuse: prompt compilation failed"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("langfuse: prompt compilation failed: %s", e.Errors[0].Error())
	}
	msgs := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		msgs[i] = err.Error()
	}
	return fmt.Sprintf("langfuse: prompt compilation failed with %d errors: %s",
		len(e.Errors), strings.Join(msgs, "; "))
}

// Unwrap returns the first error for single-error cases.
// For multiple errors, use Errors field directly.
func (e *CompilationError) Unwrap() error {
	if len(e.Errors) == 1 {
		return e.Errors[0]
	}
	return nil
}

// CodedError is an interface for errors that have an error code.
// Implement this interface to allow ErrorCodeOf to extract the code.
type CodedError interface {
	error
	Code() ErrorCode
}
