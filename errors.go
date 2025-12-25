package langfuse

import (
	"errors"
	"fmt"
)

// Sentinel errors for configuration validation.
var (
	ErrMissingPublicKey = errors.New("langfuse: public key is required")
	ErrMissingSecretKey = errors.New("langfuse: secret key is required")
	ErrMissingBaseURL   = errors.New("langfuse: base URL is required")
	ErrClientClosed     = errors.New("langfuse: client is closed")
	ErrNilRequest       = errors.New("langfuse: request cannot be nil")
)

// APIError represents an error response from the Langfuse API.
type APIError struct {
	StatusCode   int    `json:"statusCode"`
	Message      string `json:"message"`
	ErrorMessage string `json:"error"`
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("langfuse: API error (status %d): %s", e.StatusCode, e.Message)
	}
	if e.ErrorMessage != "" {
		return fmt.Sprintf("langfuse: API error (status %d): %s", e.StatusCode, e.ErrorMessage)
	}
	return fmt.Sprintf("langfuse: API error (status %d)", e.StatusCode)
}

// IsNotFound returns true if the error is a 404 Not Found error.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsUnauthorized returns true if the error is a 401 Unauthorized error.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

// IsForbidden returns true if the error is a 403 Forbidden error.
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == 403
}

// IsRateLimited returns true if the error is a 429 Too Many Requests error.
func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == 429
}

// IsServerError returns true if the error is a 5xx server error.
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// IsRetryable returns true if the request should be retried.
func (e *APIError) IsRetryable() bool {
	return e.IsRateLimited() || e.IsServerError()
}

// IngestionError represents an error that occurred during batch ingestion.
type IngestionError struct {
	ID           string `json:"id"`
	Status       int    `json:"status"`
	Message      string `json:"message"`
	ErrorMessage string `json:"error"`
}

// IngestionResult represents the result of a batch ingestion request.
type IngestionResult struct {
	Successes []IngestionSuccess `json:"successes"`
	Errors    []IngestionError   `json:"errors"`
}

// IngestionSuccess represents a successful ingestion event.
type IngestionSuccess struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
}

// HasErrors returns true if the ingestion result contains any errors.
func (r *IngestionResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// ValidationError represents a validation error for a request.
type ValidationError struct {
	Field   string
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("langfuse: validation error for field %q: %s", e.Field, e.Message)
}

// NewValidationError creates a new validation error.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}
