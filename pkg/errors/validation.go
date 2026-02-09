package errors

import "fmt"

// ValidationError represents a validation error for a request.
type ValidationError struct {
	Field   string
	Message string
	Err     error // Underlying error for wrapping
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("langfuse: validation error for field %q: %s", e.Field, e.Message)
}

// Unwrap returns the underlying error for error chain support.
func (e *ValidationError) Unwrap() error {
	return e.Err
}

// Code returns the error code for the validation error.
// Implements the LangfuseError interface.
func (e *ValidationError) Code() ErrorCode {
	return ErrCodeValidation
}

// IsRetryable returns false for validation errors (they should be fixed, not retried).
// Implements the LangfuseError interface.
func (e *ValidationError) IsRetryable() bool {
	return false
}

// GetRequestID returns an empty string (validation errors don't have request IDs).
// Implements the LangfuseError interface.
func (e *ValidationError) GetRequestID() string {
	return ""
}

// Ensure ValidationError implements LangfuseError.
var _ LangfuseError = (*ValidationError)(nil)

// NewValidationError creates a new validation error.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// NewValidationErrorWithCause creates a new validation error with an underlying cause.
func NewValidationErrorWithCause(field, message string, cause error) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
		Err:     cause,
	}
}
