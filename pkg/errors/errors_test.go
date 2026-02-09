package errors

import (
	"errors"
	"testing"
	"time"
)

// TestAPIError_Creation tests creating and using APIError.
func TestAPIError_Creation(t *testing.T) {
	tests := []struct {
		name       string
		apiErr     *APIError
		wantMsg    string
		wantCode   ErrorCode
		wantRetry  bool
		wantReqID  string
	}{
		{
			name: "not found",
			apiErr: &APIError{
				StatusCode: 404,
				Message:    "Resource not found",
			},
			wantMsg:   "langfuse: API error (status 404): Resource not found",
			wantCode:  ErrCodeAPI,
			wantRetry: false,
		},
		{
			name: "unauthorized",
			apiErr: &APIError{
				StatusCode: 401,
				Message:    "Invalid credentials",
				RequestID:  "req-123",
			},
			wantMsg:   "langfuse: API error (status 401, request req-123): Invalid credentials",
			wantCode:  ErrCodeAuth,
			wantRetry: false,
			wantReqID: "req-123",
		},
		{
			name: "rate limited",
			apiErr: &APIError{
				StatusCode: 429,
				Message:    "Too many requests",
			},
			wantMsg:   "langfuse: API error (status 429): Too many requests",
			wantCode:  ErrCodeRateLimit,
			wantRetry: true,
		},
		{
			name: "server error",
			apiErr: &APIError{
				StatusCode: 500,
				Message:    "Internal server error",
			},
			wantMsg:   "langfuse: API error (status 500): Internal server error",
			wantCode:  ErrCodeAPI,
			wantRetry: true,
		},
		{
			name: "error message fallback",
			apiErr: &APIError{
				StatusCode:   503,
				ErrorMessage: "Service unavailable",
			},
			wantMsg:   "langfuse: API error (status 503): Service unavailable",
			wantCode:  ErrCodeAPI,
			wantRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.apiErr.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
			if got := tt.apiErr.Code(); got != tt.wantCode {
				t.Errorf("Code() = %q, want %q", got, tt.wantCode)
			}
			if got := tt.apiErr.IsRetryable(); got != tt.wantRetry {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.wantRetry)
			}
			if got := tt.apiErr.GetRequestID(); got != tt.wantReqID {
				t.Errorf("GetRequestID() = %q, want %q", got, tt.wantReqID)
			}
		})
	}
}

// TestAPIError_Is tests sentinel error comparison.
func TestAPIError_Is(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{
			name:   "matches rate limited",
			err:    &APIError{StatusCode: 429, Message: "Too many requests"},
			target: ErrRateLimited,
			want:   true,
		},
		{
			name:   "matches not found",
			err:    &APIError{StatusCode: 404, Message: "Not found"},
			target: ErrNotFound,
			want:   true,
		},
		{
			name:   "does not match different status",
			err:    &APIError{StatusCode: 500, Message: "Server error"},
			target: ErrRateLimited,
			want:   false,
		},
		{
			name:   "does not match non-APIError",
			err:    &APIError{StatusCode: 404, Message: "Not found"},
			target: ErrClientClosed,
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, tt.target); got != tt.want {
				t.Errorf("errors.Is() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestAPIError_Methods tests APIError convenience methods.
func TestAPIError_Methods(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		wantNotFound   bool
		wantUnauth     bool
		wantForbidden  bool
		wantRateLimit  bool
		wantServerErr  bool
	}{
		{"not found", 404, true, false, false, false, false},
		{"unauthorized", 401, false, true, false, false, false},
		{"forbidden", 403, false, false, true, false, false},
		{"rate limited", 429, false, false, false, true, false},
		{"server error 500", 500, false, false, false, false, true},
		{"server error 503", 503, false, false, false, false, true},
		{"bad request", 400, false, false, false, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			apiErr := &APIError{StatusCode: tt.statusCode}

			if got := apiErr.IsNotFound(); got != tt.wantNotFound {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.wantNotFound)
			}
			if got := apiErr.IsUnauthorized(); got != tt.wantUnauth {
				t.Errorf("IsUnauthorized() = %v, want %v", got, tt.wantUnauth)
			}
			if got := apiErr.IsForbidden(); got != tt.wantForbidden {
				t.Errorf("IsForbidden() = %v, want %v", got, tt.wantForbidden)
			}
			if got := apiErr.IsRateLimited(); got != tt.wantRateLimit {
				t.Errorf("IsRateLimited() = %v, want %v", got, tt.wantRateLimit)
			}
			if got := apiErr.IsServerError(); got != tt.wantServerErr {
				t.Errorf("IsServerError() = %v, want %v", got, tt.wantServerErr)
			}
		})
	}
}

// TestAsAPIError tests the AsAPIError helper.
func TestAsAPIError(t *testing.T) {
	apiErr := &APIError{StatusCode: 404, Message: "Not found"}
	wrapped := WrapError(apiErr, "failed to fetch")

	tests := []struct {
		name    string
		err     error
		wantOK  bool
		wantErr *APIError
	}{
		{
			name:    "direct APIError",
			err:     apiErr,
			wantOK:  true,
			wantErr: apiErr,
		},
		{
			name:    "wrapped APIError",
			err:     wrapped,
			wantOK:  true,
			wantErr: apiErr,
		},
		{
			name:    "non-APIError",
			err:     ErrClientClosed,
			wantOK:  false,
			wantErr: nil,
		},
		{
			name:    "nil error",
			err:     nil,
			wantOK:  false,
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := AsAPIError(tt.err)
			if ok != tt.wantOK {
				t.Errorf("AsAPIError() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.wantErr {
				t.Errorf("AsAPIError() = %v, want %v", got, tt.wantErr)
			}
		})
	}
}

// TestValidationError tests ValidationError.
func TestValidationError(t *testing.T) {
	valErr := NewValidationError("name", "must not be empty")

	if got := valErr.Error(); got != `langfuse: validation error for field "name": must not be empty` {
		t.Errorf("Error() = %q", got)
	}
	if got := valErr.Code(); got != ErrCodeValidation {
		t.Errorf("Code() = %q, want %q", got, ErrCodeValidation)
	}
	if valErr.IsRetryable() {
		t.Error("IsRetryable() = true, want false")
	}
	if got := valErr.GetRequestID(); got != "" {
		t.Errorf("GetRequestID() = %q, want empty", got)
	}
}

// TestValidationError_WithCause tests ValidationError with underlying cause.
func TestValidationError_WithCause(t *testing.T) {
	cause := errors.New("invalid format")
	valErr := NewValidationErrorWithCause("email", "invalid email address", cause)

	if valErr.Unwrap() != cause {
		t.Errorf("Unwrap() = %v, want %v", valErr.Unwrap(), cause)
	}
	if !errors.Is(valErr, cause) {
		t.Error("errors.Is() = false, want true for wrapped error")
	}
}

// TestAsValidationError tests the AsValidationError helper.
func TestAsValidationError(t *testing.T) {
	valErr := NewValidationError("field", "invalid")
	wrapped := WrapError(valErr, "validation failed")

	got, ok := AsValidationError(wrapped)
	if !ok {
		t.Fatal("AsValidationError() ok = false, want true")
	}
	if got != valErr {
		t.Errorf("AsValidationError() = %v, want %v", got, valErr)
	}

	_, ok = AsValidationError(ErrClientClosed)
	if ok {
		t.Error("AsValidationError() ok = true for non-ValidationError, want false")
	}
}

// TestIsRetryable tests the IsRetryable helper.
func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "rate limited",
			err:  &APIError{StatusCode: 429},
			want: true,
		},
		{
			name: "server error",
			err:  &APIError{StatusCode: 500},
			want: true,
		},
		{
			name: "not found",
			err:  &APIError{StatusCode: 404},
			want: false,
		},
		{
			name: "validation error",
			err:  NewValidationError("field", "invalid"),
			want: false,
		},
		{
			name: "wrapped rate limited",
			err:  WrapError(&APIError{StatusCode: 429}, "request failed"),
			want: true,
		},
		{
			name: "sentinel error",
			err:  ErrClientClosed,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.want {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestRetryAfter tests the RetryAfter helper.
func TestRetryAfter(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want time.Duration
	}{
		{
			name: "nil error",
			err:  nil,
			want: 0,
		},
		{
			name: "APIError with retry after",
			err:  &APIError{StatusCode: 429, RetryAfter: 5 * time.Second},
			want: 5 * time.Second,
		},
		{
			name: "APIError without retry after",
			err:  &APIError{StatusCode: 429},
			want: 0,
		},
		{
			name: "non-APIError",
			err:  ErrClientClosed,
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := RetryAfter(tt.err); got != tt.want {
				t.Errorf("RetryAfter() = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestErrorCodeOf tests the ErrorCodeOf helper.
func TestErrorCodeOf(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want ErrorCode
	}{
		{
			name: "nil error",
			err:  nil,
			want: "",
		},
		{
			name: "config error",
			err:  ErrMissingPublicKey,
			want: ErrCodeConfig,
		},
		{
			name: "shutdown error",
			err:  ErrClientClosed,
			want: ErrCodeShutdown,
		},
		{
			name: "validation error",
			err:  NewValidationError("field", "invalid"),
			want: ErrCodeValidation,
		},
		{
			name: "API error - auth",
			err:  &APIError{StatusCode: 401},
			want: ErrCodeAuth,
		},
		{
			name: "API error - rate limit",
			err:  &APIError{StatusCode: 429},
			want: ErrCodeRateLimit,
		},
		{
			name: "API error - generic",
			err:  &APIError{StatusCode: 404},
			want: ErrCodeAPI,
		},
		{
			name: "shutdown error struct",
			err:  &ShutdownError{Message: "failed"},
			want: ErrCodeShutdown,
		},
		{
			name: "unknown error",
			err:  errors.New("unknown error"),
			want: ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ErrorCodeOf(tt.err); got != tt.want {
				t.Errorf("ErrorCodeOf() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestWrapError tests error wrapping helpers.
func TestWrapError(t *testing.T) {
	baseErr := errors.New("base error")

	wrapped := WrapError(baseErr, "operation failed")
	if wrapped == nil {
		t.Fatal("WrapError() returned nil")
	}
	if !errors.Is(wrapped, baseErr) {
		t.Error("wrapped error does not contain base error")
	}

	// Test nil wrapping
	if got := WrapError(nil, "message"); got != nil {
		t.Errorf("WrapError(nil) = %v, want nil", got)
	}

	// Test formatted wrapping
	wrappedF := WrapErrorf(baseErr, "operation %s failed", "test")
	if wrappedF == nil {
		t.Fatal("WrapErrorf() returned nil")
	}
	if !errors.Is(wrappedF, baseErr) {
		t.Error("wrapped error does not contain base error")
	}

	// Test nil formatted wrapping
	if got := WrapErrorf(nil, "format %s", "test"); got != nil {
		t.Errorf("WrapErrorf(nil) = %v, want nil", got)
	}
}

// TestShutdownError tests ShutdownError.
func TestShutdownError(t *testing.T) {
	baseErr := errors.New("context deadline exceeded")
	shutdownErr := &ShutdownError{
		Cause:         baseErr,
		PendingEvents: 10,
		Message:       "flush timeout",
	}

	if got := shutdownErr.Error(); got != "langfuse: shutdown failed (flush timeout): 10 pending events may be lost" {
		t.Errorf("Error() = %q", got)
	}
	if shutdownErr.Unwrap() != baseErr {
		t.Errorf("Unwrap() = %v, want %v", shutdownErr.Unwrap(), baseErr)
	}
	if got := shutdownErr.Code(); got != ErrCodeShutdown {
		t.Errorf("Code() = %q, want %q", got, ErrCodeShutdown)
	}
	if shutdownErr.IsRetryable() {
		t.Error("IsRetryable() = true, want false")
	}

	// Test without pending events
	shutdownErr2 := &ShutdownError{Message: "failed"}
	if got := shutdownErr2.Error(); got != "langfuse: shutdown failed: failed" {
		t.Errorf("Error() = %q", got)
	}
}

// TestCompilationError tests CompilationError.
func TestCompilationError(t *testing.T) {
	err1 := errors.New("missing variable")
	err2 := errors.New("invalid syntax")

	// Single error
	compErr1 := &CompilationError{Errors: []error{err1}}
	if got := compErr1.Error(); got != "langfuse: prompt compilation failed: missing variable" {
		t.Errorf("Error() = %q", got)
	}
	if compErr1.Unwrap() != err1 {
		t.Errorf("Unwrap() = %v, want %v", compErr1.Unwrap(), err1)
	}

	// Multiple errors
	compErr2 := &CompilationError{Errors: []error{err1, err2}}
	if got := compErr2.Error(); got != "langfuse: prompt compilation failed with 2 errors: missing variable; invalid syntax" {
		t.Errorf("Error() = %q", got)
	}
	if compErr2.Unwrap() != nil {
		t.Errorf("Unwrap() = %v, want nil for multiple errors", compErr2.Unwrap())
	}

	// No errors
	compErr3 := &CompilationError{}
	if got := compErr3.Error(); got != "langfuse: prompt compilation failed" {
		t.Errorf("Error() = %q", got)
	}
}

// TestIngestionError tests IngestionError.
func TestIngestionError(t *testing.T) {
	tests := []struct {
		name    string
		ingErr  *IngestionError
		wantMsg string
	}{
		{
			name: "with message",
			ingErr: &IngestionError{
				ID:      "evt-123",
				Status:  400,
				Message: "Invalid event",
			},
			wantMsg: "langfuse: ingestion error for evt-123 (status 400): Invalid event",
		},
		{
			name: "with error message",
			ingErr: &IngestionError{
				ID:           "evt-456",
				Status:       500,
				ErrorMessage: "Server error",
			},
			wantMsg: "langfuse: ingestion error for evt-456 (status 500): Server error",
		},
		{
			name: "no message",
			ingErr: &IngestionError{
				ID:     "evt-789",
				Status: 400,
			},
			wantMsg: "langfuse: ingestion error for evt-789 (status 400)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ingErr.Error(); got != tt.wantMsg {
				t.Errorf("Error() = %q, want %q", got, tt.wantMsg)
			}
		})
	}
}

// TestIngestionResult tests IngestionResult.
func TestIngestionResult(t *testing.T) {
	// Result with no errors
	result1 := &IngestionResult{
		Successes: []IngestionSuccess{{ID: "evt-1", Status: 200}},
	}
	if result1.HasErrors() {
		t.Error("HasErrors() = true, want false")
	}
	if got := result1.FirstError(); got != nil {
		t.Errorf("FirstError() = %v, want nil", got)
	}

	// Result with errors
	result2 := &IngestionResult{
		Successes: []IngestionSuccess{{ID: "evt-1", Status: 200}},
		Errors: []IngestionError{
			{ID: "evt-2", Status: 400, Message: "Invalid"},
			{ID: "evt-3", Status: 500, Message: "Server error"},
		},
	}
	if !result2.HasErrors() {
		t.Error("HasErrors() = false, want true")
	}
	firstErr := result2.FirstError()
	if firstErr == nil {
		t.Fatal("FirstError() = nil, want error")
	}
	if got := firstErr.Error(); got != "langfuse: ingestion error for evt-2 (status 400): Invalid" {
		t.Errorf("FirstError().Error() = %q", got)
	}
}
