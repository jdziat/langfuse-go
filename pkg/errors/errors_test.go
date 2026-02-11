package errors

import (
	"errors"
	"testing"
	"time"
)

// TestAPIError_Creation tests creating and using APIError.
func TestAPIError_Creation(t *testing.T) {
	tests := []struct {
		name      string
		apiErr    *APIError
		wantMsg   string
		wantCode  ErrorCode
		wantRetry bool
		wantReqID string
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
		name          string
		statusCode    int
		wantNotFound  bool
		wantUnauth    bool
		wantForbidden bool
		wantRateLimit bool
		wantServerErr bool
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

// TestNewAsyncErrorHandler tests creating AsyncErrorHandler with various configs.
func TestNewAsyncErrorHandler(t *testing.T) {
	t.Run("nil config uses defaults", func(t *testing.T) {
		h := NewAsyncErrorHandler(nil)
		if h == nil {
			t.Fatal("NewAsyncErrorHandler(nil) returned nil")
		}
		if h.Errors == nil {
			t.Error("Errors channel is nil")
		}
		// Default buffer size is 100
		if cap(h.Errors) != 100 {
			t.Errorf("Buffer size = %d, want 100", cap(h.Errors))
		}
	})

	t.Run("custom buffer size", func(t *testing.T) {
		h := NewAsyncErrorHandler(&AsyncErrorConfig{BufferSize: 50})
		if cap(h.Errors) != 50 {
			t.Errorf("Buffer size = %d, want 50", cap(h.Errors))
		}
	})

	t.Run("zero buffer size uses default", func(t *testing.T) {
		h := NewAsyncErrorHandler(&AsyncErrorConfig{BufferSize: 0})
		if cap(h.Errors) != 100 {
			t.Errorf("Buffer size = %d, want 100", cap(h.Errors))
		}
	})

	t.Run("negative buffer size uses default", func(t *testing.T) {
		h := NewAsyncErrorHandler(&AsyncErrorConfig{BufferSize: -1})
		if cap(h.Errors) != 100 {
			t.Errorf("Buffer size = %d, want 100", cap(h.Errors))
		}
	})
}

// TestAsyncErrorHandler_Handle tests error handling.
func TestAsyncErrorHandler_Handle(t *testing.T) {
	t.Run("handles error successfully", func(t *testing.T) {
		h := NewAsyncErrorHandler(&AsyncErrorConfig{BufferSize: 10})
		defer h.Close()

		asyncErr := NewAsyncError(AsyncOpBatchSend, errors.New("test error"))
		h.Handle(asyncErr)

		// Check error was sent to channel
		select {
		case got := <-h.Errors:
			if got != asyncErr {
				t.Errorf("Got different error from channel")
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Timeout waiting for error in channel")
		}

		// Check statistics
		if got := h.TotalErrors(); got != 1 {
			t.Errorf("TotalErrors() = %d, want 1", got)
		}
		if got := h.ErrorsByOperation(AsyncOpBatchSend); got != 1 {
			t.Errorf("ErrorsByOperation() = %d, want 1", got)
		}
	})

	t.Run("handles nil error gracefully", func(t *testing.T) {
		h := NewAsyncErrorHandler(&AsyncErrorConfig{BufferSize: 10})
		defer h.Close()

		h.Handle(nil)

		if got := h.TotalErrors(); got != 0 {
			t.Errorf("TotalErrors() = %d, want 0", got)
		}
	})

	t.Run("buffer overflow drops errors", func(t *testing.T) {
		h := NewAsyncErrorHandler(&AsyncErrorConfig{BufferSize: 2})
		defer h.Close()

		// Fill the buffer
		for range 2 {
			h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("test")))
		}

		// This should be dropped
		h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("dropped")))

		if got := h.DroppedCount(); got != 1 {
			t.Errorf("DroppedCount() = %d, want 1", got)
		}
		if got := h.TotalErrors(); got != 3 {
			t.Errorf("TotalErrors() = %d, want 3", got)
		}
	})

	t.Run("invokes error callback", func(t *testing.T) {
		var called bool
		var receivedErr *AsyncError

		h := NewAsyncErrorHandler(&AsyncErrorConfig{
			BufferSize: 10,
			OnError: func(err *AsyncError) {
				called = true
				receivedErr = err
			},
		})
		defer h.Close()

		asyncErr := NewAsyncError(AsyncOpFlush, errors.New("test"))
		h.Handle(asyncErr)

		// Give callback time to execute
		time.Sleep(10 * time.Millisecond)

		if !called {
			t.Error("OnError callback was not called")
		}
		if receivedErr != asyncErr {
			t.Error("OnError received different error")
		}
	})

	t.Run("invokes overflow callback", func(t *testing.T) {
		var called bool
		var droppedCount int

		h := NewAsyncErrorHandler(&AsyncErrorConfig{
			BufferSize: 1,
			OnOverflow: func(dropped int) {
				called = true
				droppedCount = dropped
			},
		})
		defer h.Close()

		// Fill buffer
		h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("test1")))
		// Trigger overflow
		h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("test2")))

		// Give callback time to execute
		time.Sleep(10 * time.Millisecond)

		if !called {
			t.Error("OnOverflow callback was not called")
		}
		if droppedCount != 1 {
			t.Errorf("Dropped count = %d, want 1", droppedCount)
		}
	})
}

// TestAsyncErrorHandler_SetCallback tests callback setters.
func TestAsyncErrorHandler_SetCallback(t *testing.T) {
	h := NewAsyncErrorHandler(&AsyncErrorConfig{BufferSize: 10})
	defer h.Close()

	var called bool
	h.SetCallback(func(err *AsyncError) {
		called = true
	})

	h.Handle(NewAsyncError(AsyncOpFlush, errors.New("test")))
	time.Sleep(10 * time.Millisecond)

	if !called {
		t.Error("SetCallback did not set callback")
	}
}

// TestAsyncErrorHandler_SetOverflowCallback tests overflow callback setter.
func TestAsyncErrorHandler_SetOverflowCallback(t *testing.T) {
	h := NewAsyncErrorHandler(&AsyncErrorConfig{BufferSize: 1})
	defer h.Close()

	var called bool
	h.SetOverflowCallback(func(dropped int) {
		called = true
	})

	// Fill buffer and trigger overflow
	h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("test1")))
	h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("test2")))
	time.Sleep(10 * time.Millisecond)

	if !called {
		t.Error("SetOverflowCallback did not set callback")
	}
}

// TestAsyncErrorHandler_Drain tests draining pending errors.
func TestAsyncErrorHandler_Drain(t *testing.T) {
	h := NewAsyncErrorHandler(&AsyncErrorConfig{BufferSize: 10})
	defer h.Close()

	// Add some errors
	err1 := NewAsyncError(AsyncOpBatchSend, errors.New("error1"))
	err2 := NewAsyncError(AsyncOpFlush, errors.New("error2"))
	err3 := NewAsyncError(AsyncOpHook, errors.New("error3"))

	h.Handle(err1)
	h.Handle(err2)
	h.Handle(err3)

	// Drain all errors
	drained := h.Drain()

	if len(drained) != 3 {
		t.Errorf("Drained %d errors, want 3", len(drained))
	}

	// Verify channel is empty
	if h.Pending() != 0 {
		t.Errorf("Pending() = %d, want 0", h.Pending())
	}

	// Drain again should return empty
	drained2 := h.Drain()
	if len(drained2) != 0 {
		t.Errorf("Second drain returned %d errors, want 0", len(drained2))
	}
}

// TestAsyncErrorHandler_Statistics tests statistics methods.
func TestAsyncErrorHandler_Statistics(t *testing.T) {
	h := NewAsyncErrorHandler(&AsyncErrorConfig{BufferSize: 10})
	defer h.Close()

	// Handle various errors
	h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("error1")))
	h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("error2")))
	h.Handle(NewAsyncError(AsyncOpFlush, errors.New("error3")))

	// Test TotalErrors
	if got := h.TotalErrors(); got != 3 {
		t.Errorf("TotalErrors() = %d, want 3", got)
	}

	// Test ErrorsByOperation
	if got := h.ErrorsByOperation(AsyncOpBatchSend); got != 2 {
		t.Errorf("ErrorsByOperation(batch_send) = %d, want 2", got)
	}
	if got := h.ErrorsByOperation(AsyncOpFlush); got != 1 {
		t.Errorf("ErrorsByOperation(flush) = %d, want 1", got)
	}
	if got := h.ErrorsByOperation(AsyncOpHook); got != 0 {
		t.Errorf("ErrorsByOperation(hook) = %d, want 0", got)
	}

	// Test Pending
	if got := h.Pending(); got != 3 {
		t.Errorf("Pending() = %d, want 3", got)
	}

	// Test Stats
	stats := h.Stats()
	if stats.TotalErrors != 3 {
		t.Errorf("Stats.TotalErrors = %d, want 3", stats.TotalErrors)
	}
	if stats.DroppedCount != 0 {
		t.Errorf("Stats.DroppedCount = %d, want 0", stats.DroppedCount)
	}
	if stats.Pending != 3 {
		t.Errorf("Stats.Pending = %d, want 3", stats.Pending)
	}
	if stats.BufferSize != 10 {
		t.Errorf("Stats.BufferSize = %d, want 10", stats.BufferSize)
	}
}

// TestWrapAsyncError tests the WrapAsyncError helper.
func TestWrapAsyncError(t *testing.T) {
	t.Run("wraps regular error", func(t *testing.T) {
		err := errors.New("test error")
		asyncErr := WrapAsyncError(AsyncOpBatchSend, err)

		if asyncErr == nil {
			t.Fatal("WrapAsyncError returned nil")
		}
		if asyncErr.Operation != AsyncOpBatchSend {
			t.Errorf("Operation = %q, want %q", asyncErr.Operation, AsyncOpBatchSend)
		}
		if asyncErr.Err != err {
			t.Error("Underlying error not preserved")
		}
	})

	t.Run("returns nil for nil error", func(t *testing.T) {
		asyncErr := WrapAsyncError(AsyncOpFlush, nil)
		if asyncErr != nil {
			t.Errorf("WrapAsyncError(nil) = %v, want nil", asyncErr)
		}
	})

	t.Run("returns AsyncError as-is", func(t *testing.T) {
		original := NewAsyncError(AsyncOpFlush, errors.New("test"))
		wrapped := WrapAsyncError(AsyncOpBatchSend, original)

		if wrapped != original {
			t.Error("WrapAsyncError changed existing AsyncError")
		}
	})
}

// TestAsAsyncError tests the AsAsyncError helper.
func TestAsAsyncError(t *testing.T) {
	asyncErr := NewAsyncError(AsyncOpBatchSend, errors.New("test"))
	wrapped := WrapError(asyncErr, "failed")

	tests := []struct {
		name    string
		err     error
		wantOK  bool
		wantErr *AsyncError
	}{
		{
			name:    "direct AsyncError",
			err:     asyncErr,
			wantOK:  true,
			wantErr: asyncErr,
		},
		{
			name:    "wrapped AsyncError",
			err:     wrapped,
			wantOK:  true,
			wantErr: asyncErr,
		},
		{
			name:    "non-AsyncError",
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
			got, ok := AsAsyncError(tt.err)
			if ok != tt.wantOK {
				t.Errorf("AsAsyncError() ok = %v, want %v", ok, tt.wantOK)
			}
			if got != tt.wantErr {
				t.Errorf("AsAsyncError() = %v, want %v", got, tt.wantErr)
			}
		})
	}
}
