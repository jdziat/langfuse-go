package errors

import (
	"fmt"
	"time"
)

// Sentinel APIError values for use with errors.Is().
// These match on status code only.
var (
	ErrNotFound     = &APIError{StatusCode: 404}
	ErrUnauthorized = &APIError{StatusCode: 401}
	ErrForbidden    = &APIError{StatusCode: 403}
	ErrRateLimited  = &APIError{StatusCode: 429}
)

// APIError represents an error response from the Langfuse API.
// It supports error wrapping via Unwrap() and comparison via Is().
type APIError struct {
	StatusCode   int           `json:"statusCode"`
	Message      string        `json:"message"`
	ErrorMessage string        `json:"error"`
	RequestID    string        `json:"-"` // Request ID for debugging
	RetryAfter   time.Duration `json:"-"` // From Retry-After header
	Err          error         `json:"-"` // Underlying error for wrapping
}

// Error implements the error interface.
func (e *APIError) Error() string {
	var msg string
	if e.Message != "" {
		msg = e.Message
	} else if e.ErrorMessage != "" {
		msg = e.ErrorMessage
	}

	if msg != "" {
		if e.RequestID != "" {
			return fmt.Sprintf("langfuse: API error (status %d, request %s): %s", e.StatusCode, e.RequestID, msg)
		}
		return fmt.Sprintf("langfuse: API error (status %d): %s", e.StatusCode, msg)
	}

	if e.RequestID != "" {
		return fmt.Sprintf("langfuse: API error (status %d, request %s)", e.StatusCode, e.RequestID)
	}
	return fmt.Sprintf("langfuse: API error (status %d)", e.StatusCode)
}

// String returns a compact string representation for debugging.
func (e *APIError) String() string {
	msg := e.Message
	if msg == "" {
		msg = e.ErrorMessage
	}
	return fmt.Sprintf("APIError{Status: %d, Message: %q}", e.StatusCode, msg)
}

// Unwrap returns the underlying error for error chain support.
// This allows using errors.Unwrap() and errors.Is() with wrapped errors.
func (e *APIError) Unwrap() error {
	return e.Err
}

// Is implements error comparison for errors.Is().
// It matches on status code, allowing comparisons like:
//
//	if errors.Is(err, langfuse.ErrRateLimited) { ... }
func (e *APIError) Is(target error) bool {
	t, ok := target.(*APIError)
	if !ok {
		return false
	}
	// Match on status code for sentinel error comparison
	return e.StatusCode == t.StatusCode
}

// WithError wraps an underlying error in the APIError.
func (e *APIError) WithError(err error) *APIError {
	e.Err = err
	return e
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

// SuggestedRetryAfter returns the suggested retry delay from the Retry-After header.
// This method implements the pkghttp.RetryAfterError interface.
// Note: The method is named SuggestedRetryAfter to avoid conflict with the RetryAfter field.
func (e *APIError) SuggestedRetryAfter() time.Duration {
	return e.RetryAfter
}

// Code returns the error code for the API error.
// Implements the LangfuseError interface.
func (e *APIError) Code() ErrorCode {
	switch {
	case e.IsUnauthorized(), e.IsForbidden():
		return ErrCodeAuth
	case e.IsRateLimited():
		return ErrCodeRateLimit
	default:
		return ErrCodeAPI
	}
}

// GetRequestID returns the request ID for the API error.
// Implements the LangfuseError interface.
func (e *APIError) GetRequestID() string {
	return e.RequestID
}

// Ensure APIError implements LangfuseError.
var _ LangfuseError = (*APIError)(nil)

// IngestionError represents an error that occurred during batch ingestion.
type IngestionError struct {
	ID           string `json:"id"`
	Status       int    `json:"status"`
	Message      string `json:"message"`
	ErrorMessage string `json:"error"`
}

// Error implements the error interface.
func (e *IngestionError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("langfuse: ingestion error for %s (status %d): %s", e.ID, e.Status, e.Message)
	}
	if e.ErrorMessage != "" {
		return fmt.Sprintf("langfuse: ingestion error for %s (status %d): %s", e.ID, e.Status, e.ErrorMessage)
	}
	return fmt.Sprintf("langfuse: ingestion error for %s (status %d)", e.ID, e.Status)
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

// FirstError returns the first ingestion error, or nil if there are none.
func (r *IngestionResult) FirstError() error {
	if len(r.Errors) == 0 {
		return nil
	}
	return &r.Errors[0]
}
