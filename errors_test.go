package langfuse

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestAPIErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      APIError
		expected string
	}{
		{
			name: "with message",
			err: APIError{
				StatusCode: 400,
				Message:    "Bad Request",
			},
			expected: "langfuse: API error (status 400): Bad Request",
		},
		{
			name: "with error message",
			err: APIError{
				StatusCode:   500,
				ErrorMessage: "Internal Server Error",
			},
			expected: "langfuse: API error (status 500): Internal Server Error",
		},
		{
			name: "message takes precedence",
			err: APIError{
				StatusCode:   400,
				Message:      "Bad Request",
				ErrorMessage: "Error Detail",
			},
			expected: "langfuse: API error (status 400): Bad Request",
		},
		{
			name: "status code only",
			err: APIError{
				StatusCode: 404,
			},
			expected: "langfuse: API error (status 404)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAPIErrorStatusChecks(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		isNotFound     bool
		isUnauthorized bool
		isForbidden    bool
		isRateLimited  bool
		isServerError  bool
		isRetryable    bool
	}{
		{
			name:       "200 OK",
			statusCode: 200,
		},
		{
			name:           "401 Unauthorized",
			statusCode:     401,
			isUnauthorized: true,
		},
		{
			name:        "403 Forbidden",
			statusCode:  403,
			isForbidden: true,
		},
		{
			name:       "404 Not Found",
			statusCode: 404,
			isNotFound: true,
		},
		{
			name:          "429 Too Many Requests",
			statusCode:    429,
			isRateLimited: true,
			isRetryable:   true,
		},
		{
			name:          "500 Internal Server Error",
			statusCode:    500,
			isServerError: true,
			isRetryable:   true,
		},
		{
			name:          "502 Bad Gateway",
			statusCode:    502,
			isServerError: true,
			isRetryable:   true,
		},
		{
			name:          "503 Service Unavailable",
			statusCode:    503,
			isServerError: true,
			isRetryable:   true,
		},
		{
			name:          "599 edge case",
			statusCode:    599,
			isServerError: true,
			isRetryable:   true,
		},
		{
			name:       "600 not server error",
			statusCode: 600,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &APIError{StatusCode: tt.statusCode}

			if got := err.IsNotFound(); got != tt.isNotFound {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.isNotFound)
			}
			if got := err.IsUnauthorized(); got != tt.isUnauthorized {
				t.Errorf("IsUnauthorized() = %v, want %v", got, tt.isUnauthorized)
			}
			if got := err.IsForbidden(); got != tt.isForbidden {
				t.Errorf("IsForbidden() = %v, want %v", got, tt.isForbidden)
			}
			if got := err.IsRateLimited(); got != tt.isRateLimited {
				t.Errorf("IsRateLimited() = %v, want %v", got, tt.isRateLimited)
			}
			if got := err.IsServerError(); got != tt.isServerError {
				t.Errorf("IsServerError() = %v, want %v", got, tt.isServerError)
			}
			if got := err.IsRetryable(); got != tt.isRetryable {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.isRetryable)
			}
		})
	}
}

func TestIngestionResultHasErrors(t *testing.T) {
	tests := []struct {
		name     string
		result   IngestionResult
		hasError bool
	}{
		{
			name:     "empty result",
			result:   IngestionResult{},
			hasError: false,
		},
		{
			name: "successes only",
			result: IngestionResult{
				Successes: []IngestionSuccess{
					{ID: "1", Status: 200},
					{ID: "2", Status: 200},
				},
			},
			hasError: false,
		},
		{
			name: "with errors",
			result: IngestionResult{
				Successes: []IngestionSuccess{
					{ID: "1", Status: 200},
				},
				Errors: []IngestionError{
					{ID: "2", Status: 400, Message: "Invalid"},
				},
			},
			hasError: true,
		},
		{
			name: "errors only",
			result: IngestionResult{
				Errors: []IngestionError{
					{ID: "1", Status: 400, Message: "Invalid"},
				},
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.result.HasErrors(); got != tt.hasError {
				t.Errorf("HasErrors() = %v, want %v", got, tt.hasError)
			}
		})
	}
}

func TestValidationError(t *testing.T) {
	err := NewValidationError("name", "field is required")

	if err.Field != "name" {
		t.Errorf("Field = %v, want %v", err.Field, "name")
	}
	if err.Message != "field is required" {
		t.Errorf("Message = %v, want %v", err.Message, "field is required")
	}

	expected := `langfuse: validation error for field "name": field is required`
	if got := err.Error(); got != expected {
		t.Errorf("Error() = %v, want %v", got, expected)
	}
}

func TestSentinelErrors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{
			name:     "ErrMissingPublicKey",
			err:      ErrMissingPublicKey,
			contains: "public key",
		},
		{
			name:     "ErrMissingSecretKey",
			err:      ErrMissingSecretKey,
			contains: "secret key",
		},
		{
			name:     "ErrMissingBaseURL",
			err:      ErrMissingBaseURL,
			contains: "base URL",
		},
		{
			name:     "ErrClientClosed",
			err:      ErrClientClosed,
			contains: "closed",
		},
		{
			name:     "ErrNilRequest",
			err:      ErrNilRequest,
			contains: "nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Error("Error should not be nil")
			}
			errStr := tt.err.Error()
			if !containsSubstring(errStr, tt.contains) {
				t.Errorf("Error() = %v, should contain %v", errStr, tt.contains)
			}
		})
	}
}

// containsSubstring checks if s contains substr
func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCompilationError(t *testing.T) {
	t.Run("single error", func(t *testing.T) {
		err := &CompilationError{
			Errors: []error{
				NewValidationError("role", "missing role"),
			},
		}
		errStr := err.Error()
		if !containsSubstring(errStr, "prompt compilation failed") {
			t.Errorf("Error() = %v, should contain 'prompt compilation failed'", errStr)
		}
		if !containsSubstring(errStr, "missing role") {
			t.Errorf("Error() = %v, should contain 'missing role'", errStr)
		}

		// Test Unwrap for single error
		unwrapped := err.Unwrap()
		if unwrapped == nil {
			t.Error("Unwrap() should return the single error")
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		err := &CompilationError{
			Errors: []error{
				NewValidationError("role", "missing role"),
				NewValidationError("content", "missing content"),
			},
		}
		errStr := err.Error()
		if !containsSubstring(errStr, "2 errors") {
			t.Errorf("Error() = %v, should contain '2 errors'", errStr)
		}
		if !containsSubstring(errStr, "missing role") {
			t.Errorf("Error() = %v, should contain 'missing role'", errStr)
		}
		if !containsSubstring(errStr, "missing content") {
			t.Errorf("Error() = %v, should contain 'missing content'", errStr)
		}

		// Test Unwrap for multiple errors returns nil
		unwrapped := err.Unwrap()
		if unwrapped != nil {
			t.Error("Unwrap() should return nil for multiple errors")
		}
	})

	t.Run("empty errors", func(t *testing.T) {
		err := &CompilationError{}
		errStr := err.Error()
		if !containsSubstring(errStr, "prompt compilation failed") {
			t.Errorf("Error() = %v, should contain 'prompt compilation failed'", errStr)
		}
	})

	t.Run("IsCompilationError helper", func(t *testing.T) {
		err := &CompilationError{
			Errors: []error{NewValidationError("test", "test error")},
		}

		compErr, ok := IsCompilationError(err)
		if !ok {
			t.Error("IsCompilationError() should return true for CompilationError")
		}
		if compErr == nil {
			t.Error("IsCompilationError() should return the error")
		}

		// Test with non-CompilationError
		_, ok = IsCompilationError(ErrClientClosed)
		if ok {
			t.Error("IsCompilationError() should return false for non-CompilationError")
		}
	})
}

func TestShutdownError(t *testing.T) {
	t.Run("with pending events", func(t *testing.T) {
		err := &ShutdownError{
			Cause:         ErrClientClosed,
			PendingEvents: 10,
			Message:       "timeout",
		}
		errStr := err.Error()
		if !containsSubstring(errStr, "10 pending events") {
			t.Errorf("Error() = %v, should contain '10 pending events'", errStr)
		}
	})

	t.Run("without pending events", func(t *testing.T) {
		err := &ShutdownError{
			Cause:   ErrClientClosed,
			Message: "timeout",
		}
		errStr := err.Error()
		if containsSubstring(errStr, "pending events") {
			t.Errorf("Error() = %v, should not contain 'pending events'", errStr)
		}
	})

	t.Run("Unwrap", func(t *testing.T) {
		cause := ErrClientClosed
		err := &ShutdownError{Cause: cause}
		if err.Unwrap() != cause {
			t.Error("Unwrap() should return the cause")
		}
	})

	t.Run("IsShutdownError helper", func(t *testing.T) {
		err := &ShutdownError{Message: "test"}

		shutdownErr, ok := IsShutdownError(err)
		if !ok {
			t.Error("IsShutdownError() should return true for ShutdownError")
		}
		if shutdownErr == nil {
			t.Error("IsShutdownError() should return the error")
		}
	})
}

func TestAPIErrorWithRequestID(t *testing.T) {
	err := &APIError{
		StatusCode: 500,
		Message:    "Internal error",
		RequestID:  "req-12345",
	}
	errStr := err.Error()
	if !containsSubstring(errStr, "req-12345") {
		t.Errorf("Error() = %v, should contain request ID", errStr)
	}
}

func TestAPIErrorIs(t *testing.T) {
	err := &APIError{StatusCode: 404}

	if err.Is(ErrNotFound) != true {
		t.Error("404 error should match ErrNotFound")
	}
	if err.Is(ErrUnauthorized) != false {
		t.Error("404 error should not match ErrUnauthorized")
	}
	if err.Is(ErrClientClosed) != false {
		t.Error("APIError should not match non-APIError")
	}
}

func TestAPIErrorWithError(t *testing.T) {
	cause := ErrClientClosed
	err := &APIError{StatusCode: 500}
	wrapped := err.WithError(cause)

	if wrapped.Unwrap() != cause {
		t.Error("WithError should set the underlying error")
	}
}

func TestAsAPIError(t *testing.T) {
	t.Run("with APIError", func(t *testing.T) {
		err := &APIError{StatusCode: 404, Message: "Not found"}
		apiErr, ok := AsAPIError(err)
		if !ok {
			t.Error("AsAPIError() should return true for APIError")
		}
		if apiErr == nil {
			t.Error("AsAPIError() should return the error")
		}
		if apiErr.StatusCode != 404 {
			t.Errorf("StatusCode = %d, want 404", apiErr.StatusCode)
		}
	})

	t.Run("with non-APIError", func(t *testing.T) {
		apiErr, ok := AsAPIError(ErrClientClosed)
		if ok {
			t.Error("AsAPIError() should return false for non-APIError")
		}
		if apiErr != nil {
			t.Error("AsAPIError() should return nil for non-APIError")
		}
	})

	t.Run("with nil error", func(t *testing.T) {
		apiErr, ok := AsAPIError(nil)
		if ok {
			t.Error("AsAPIError() should return false for nil")
		}
		if apiErr != nil {
			t.Error("AsAPIError() should return nil for nil")
		}
	})
}

func TestAsValidationError(t *testing.T) {
	t.Run("with ValidationError", func(t *testing.T) {
		err := NewValidationError("field", "is required")
		valErr, ok := AsValidationError(err)
		if !ok {
			t.Error("AsValidationError() should return true for ValidationError")
		}
		if valErr == nil {
			t.Error("AsValidationError() should return the error")
		}
		if valErr.Field != "field" {
			t.Errorf("Field = %s, want 'field'", valErr.Field)
		}
	})

	t.Run("with non-ValidationError", func(t *testing.T) {
		valErr, ok := AsValidationError(ErrClientClosed)
		if ok {
			t.Error("AsValidationError() should return false for non-ValidationError")
		}
		if valErr != nil {
			t.Error("AsValidationError() should return nil for non-ValidationError")
		}
	})
}

func TestAsIngestionError(t *testing.T) {
	t.Run("with IngestionError", func(t *testing.T) {
		err := &IngestionError{ID: "event-1", Status: 400, Message: "Bad request"}
		ingErr, ok := AsIngestionError(err)
		if !ok {
			t.Error("AsIngestionError() should return true for IngestionError")
		}
		if ingErr == nil {
			t.Error("AsIngestionError() should return the error")
		}
		if ingErr.ID != "event-1" {
			t.Errorf("ID = %s, want 'event-1'", ingErr.ID)
		}
	})

	t.Run("with non-IngestionError", func(t *testing.T) {
		ingErr, ok := AsIngestionError(ErrClientClosed)
		if ok {
			t.Error("AsIngestionError() should return false for non-IngestionError")
		}
		if ingErr != nil {
			t.Error("AsIngestionError() should return nil for non-IngestionError")
		}
	})
}

func TestAsShutdownError(t *testing.T) {
	t.Run("with ShutdownError", func(t *testing.T) {
		err := &ShutdownError{PendingEvents: 10, Message: "timeout"}
		shutdownErr, ok := AsShutdownError(err)
		if !ok {
			t.Error("AsShutdownError() should return true for ShutdownError")
		}
		if shutdownErr == nil {
			t.Error("AsShutdownError() should return the error")
		}
		if shutdownErr.PendingEvents != 10 {
			t.Errorf("PendingEvents = %d, want 10", shutdownErr.PendingEvents)
		}
	})

	t.Run("with non-ShutdownError", func(t *testing.T) {
		shutdownErr, ok := AsShutdownError(ErrClientClosed)
		if ok {
			t.Error("AsShutdownError() should return false for non-ShutdownError")
		}
		if shutdownErr != nil {
			t.Error("AsShutdownError() should return nil for non-ShutdownError")
		}
	})
}

func TestAsCompilationError(t *testing.T) {
	t.Run("with CompilationError", func(t *testing.T) {
		err := &CompilationError{Errors: []error{NewValidationError("test", "error")}}
		compErr, ok := AsCompilationError(err)
		if !ok {
			t.Error("AsCompilationError() should return true for CompilationError")
		}
		if compErr == nil {
			t.Error("AsCompilationError() should return the error")
		}
		if len(compErr.Errors) != 1 {
			t.Errorf("Errors length = %d, want 1", len(compErr.Errors))
		}
	})

	t.Run("with non-CompilationError", func(t *testing.T) {
		compErr, ok := AsCompilationError(ErrClientClosed)
		if ok {
			t.Error("AsCompilationError() should return false for non-CompilationError")
		}
		if compErr != nil {
			t.Error("AsCompilationError() should return nil for non-CompilationError")
		}
	})
}

func TestErrorExtractionHelpers(t *testing.T) {
	t.Run("AsAPIError extracts API error", func(t *testing.T) {
		err := &APIError{StatusCode: 500}
		apiErr, ok := AsAPIError(err)
		if !ok || apiErr == nil {
			t.Error("AsAPIError should extract API error")
		}
	})

	t.Run("AsValidationError extracts validation error", func(t *testing.T) {
		err := NewValidationError("test", "error")
		valErr, ok := AsValidationError(err)
		if !ok || valErr == nil {
			t.Error("AsValidationError should extract validation error")
		}
	})

	t.Run("AsIngestionError extracts ingestion error", func(t *testing.T) {
		err := &IngestionError{ID: "1", Status: 400}
		ingErr, ok := AsIngestionError(err)
		if !ok || ingErr == nil {
			t.Error("AsIngestionError should extract ingestion error")
		}
	})
}

func TestErrorCodeOf(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorCode
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "config error - missing public key",
			err:      ErrMissingPublicKey,
			expected: ErrCodeConfig,
		},
		{
			name:     "config error - missing secret key",
			err:      ErrMissingSecretKey,
			expected: ErrCodeConfig,
		},
		{
			name:     "config error - missing base URL",
			err:      ErrMissingBaseURL,
			expected: ErrCodeConfig,
		},
		{
			name:     "config error - invalid config",
			err:      ErrInvalidConfig,
			expected: ErrCodeConfig,
		},
		{
			name:     "shutdown error - client closed",
			err:      ErrClientClosed,
			expected: ErrCodeShutdown,
		},
		{
			name:     "timeout error - context cancelled",
			err:      ErrContextCancelled,
			expected: ErrCodeTimeout,
		},
		{
			name:     "network error - circuit open",
			err:      ErrCircuitOpen,
			expected: ErrCodeNetwork,
		},
		{
			name:     "API error - unauthorized",
			err:      &APIError{StatusCode: 401},
			expected: ErrCodeAuth,
		},
		{
			name:     "API error - forbidden",
			err:      &APIError{StatusCode: 403},
			expected: ErrCodeAuth,
		},
		{
			name:     "API error - rate limited",
			err:      &APIError{StatusCode: 429},
			expected: ErrCodeRateLimit,
		},
		{
			name:     "API error - server error",
			err:      &APIError{StatusCode: 500},
			expected: ErrCodeAPI,
		},
		{
			name:     "validation error",
			err:      NewValidationError("field", "message"),
			expected: ErrCodeValidation,
		},
		{
			name:     "shutdown error struct",
			err:      &ShutdownError{Message: "test"},
			expected: ErrCodeShutdown,
		},
		{
			name:     "unknown error",
			err:      errors.New("unknown error"),
			expected: ErrCodeInternal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ErrorCodeOf(tt.err)
			if got != tt.expected {
				t.Errorf("ErrorCodeOf() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		result := WrapError(nil, "context")
		if result != nil {
			t.Errorf("WrapError(nil) should return nil, got %v", result)
		}
	})

	t.Run("wraps error with context", func(t *testing.T) {
		original := errors.New("original error")
		wrapped := WrapError(original, "failed to process")

		if wrapped == nil {
			t.Fatal("WrapError should return wrapped error")
		}

		// Check that message contains both context and original error
		msg := wrapped.Error()
		if !strings.Contains(msg, "langfuse:") {
			t.Error("Wrapped error should have langfuse prefix")
		}
		if !strings.Contains(msg, "failed to process") {
			t.Error("Wrapped error should contain context message")
		}
		if !strings.Contains(msg, "original error") {
			t.Error("Wrapped error should contain original error message")
		}

		// Check that original error can be unwrapped
		if !errors.Is(wrapped, original) {
			t.Error("Wrapped error should unwrap to original")
		}
	})
}

func TestWrapErrorf(t *testing.T) {
	t.Run("nil error returns nil", func(t *testing.T) {
		result := WrapErrorf(nil, "context %s", "arg")
		if result != nil {
			t.Errorf("WrapErrorf(nil) should return nil, got %v", result)
		}
	})

	t.Run("wraps error with formatted context", func(t *testing.T) {
		original := errors.New("original error")
		wrapped := WrapErrorf(original, "failed to process trace %s", "trace-123")

		if wrapped == nil {
			t.Fatal("WrapErrorf should return wrapped error")
		}

		msg := wrapped.Error()
		if !strings.Contains(msg, "trace-123") {
			t.Error("Wrapped error should contain formatted argument")
		}

		if !errors.Is(wrapped, original) {
			t.Error("Wrapped error should unwrap to original")
		}
	})
}

func TestAllSentinelErrorsPrefix(t *testing.T) {
	// Test that all sentinel errors have consistent format
	sentinels := []error{
		ErrMissingPublicKey,
		ErrMissingSecretKey,
		ErrMissingBaseURL,
		ErrInvalidConfig,
		ErrClientClosed,
		ErrNilRequest,
		ErrPromptNotFound,
		ErrDatasetNotFound,
		ErrTraceNotFound,
		ErrEmptyBatch,
		ErrBatchTooLarge,
		ErrCircuitOpen,
		ErrContextCancelled,
		ErrShutdownTimeout,
	}

	for _, err := range sentinels {
		msg := err.Error()
		if !strings.HasPrefix(msg, "langfuse:") {
			t.Errorf("Sentinel error %q should have 'langfuse:' prefix", msg)
		}
	}
}

func TestIngestionErrorError(t *testing.T) {
	tests := []struct {
		name     string
		err      IngestionError
		expected string
	}{
		{
			name: "with message",
			err: IngestionError{
				ID:      "event-123",
				Status:  400,
				Message: "validation failed",
			},
			expected: "langfuse: ingestion error for event-123 (status 400): validation failed",
		},
		{
			name: "with error message",
			err: IngestionError{
				ID:           "event-456",
				Status:       500,
				ErrorMessage: "internal error",
			},
			expected: "langfuse: ingestion error for event-456 (status 500): internal error",
		},
		{
			name: "message takes precedence",
			err: IngestionError{
				ID:           "event-789",
				Status:       400,
				Message:      "primary message",
				ErrorMessage: "secondary message",
			},
			expected: "langfuse: ingestion error for event-789 (status 400): primary message",
		},
		{
			name: "no message",
			err: IngestionError{
				ID:     "event-000",
				Status: 500,
			},
			expected: "langfuse: ingestion error for event-000 (status 500)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("Error() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIngestionResultFirstError(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		result := &IngestionResult{
			Successes: []IngestionSuccess{{ID: "1", Status: 200}},
		}
		if err := result.FirstError(); err != nil {
			t.Errorf("FirstError() = %v, want nil", err)
		}
	})

	t.Run("with errors", func(t *testing.T) {
		result := &IngestionResult{
			Errors: []IngestionError{
				{ID: "1", Status: 400, Message: "first error"},
				{ID: "2", Status: 400, Message: "second error"},
			},
		}
		err := result.FirstError()
		if err == nil {
			t.Fatal("FirstError() = nil, want error")
		}
		if !strings.Contains(err.Error(), "first error") {
			t.Errorf("FirstError() = %q, should contain 'first error'", err.Error())
		}
	})
}

func TestNewValidationErrorWithCause(t *testing.T) {
	cause := errors.New("underlying cause")
	err := NewValidationErrorWithCause("field_name", "validation message", cause)

	if err.Field != "field_name" {
		t.Errorf("Field = %q, want %q", err.Field, "field_name")
	}
	if err.Message != "validation message" {
		t.Errorf("Message = %q, want %q", err.Message, "validation message")
	}
	if err.Err != cause {
		t.Error("Err should be the cause")
	}

	// Test unwrap
	unwrapped := errors.Unwrap(err)
	if unwrapped != cause {
		t.Error("Unwrap should return the cause")
	}
}

func TestAPIErrorString(t *testing.T) {
	err := &APIError{
		StatusCode: 400,
		Message:    "bad request",
	}
	str := err.String()
	if !strings.Contains(str, "400") {
		t.Errorf("String() = %q, should contain status code", str)
	}
	if !strings.Contains(str, "bad request") {
		t.Errorf("String() = %q, should contain message", str)
	}

	// Test with ErrorMessage instead of Message
	err2 := &APIError{
		StatusCode:   500,
		ErrorMessage: "server error",
	}
	str2 := err2.String()
	if !strings.Contains(str2, "server error") {
		t.Errorf("String() = %q, should contain error message", str2)
	}
}

func TestRetryAfterFunction(t *testing.T) {
	t.Run("non-API error", func(t *testing.T) {
		err := errors.New("generic error")
		if duration := RetryAfter(err); duration != 0 {
			t.Errorf("RetryAfter() = %v, want 0", duration)
		}
	})

	t.Run("API error without RetryAfter", func(t *testing.T) {
		err := &APIError{StatusCode: 429}
		if duration := RetryAfter(err); duration != 0 {
			t.Errorf("RetryAfter() = %v, want 0", duration)
		}
	})

	t.Run("API error with RetryAfter", func(t *testing.T) {
		err := &APIError{
			StatusCode: 429,
			RetryAfter: 60000000000, // 60 seconds in nanoseconds
		}
		expected := 60000000000
		if duration := RetryAfter(err); int64(duration) != int64(expected) {
			t.Errorf("RetryAfter() = %v, want %v", duration, expected)
		}
	})
}

func TestAPIErrorWithRequestIDExtended(t *testing.T) {
	t.Run("error with request ID and message", func(t *testing.T) {
		err := &APIError{
			StatusCode: 400,
			Message:    "bad request",
			RequestID:  "req-123",
		}
		errStr := err.Error()
		if !strings.Contains(errStr, "req-123") {
			t.Errorf("Error() = %q, should contain request ID", errStr)
		}
	})

	t.Run("error with request ID without message", func(t *testing.T) {
		err := &APIError{
			StatusCode: 500,
			RequestID:  "req-456",
		}
		errStr := err.Error()
		if !strings.Contains(errStr, "req-456") {
			t.Errorf("Error() = %q, should contain request ID", errStr)
		}
	})
}

// ============================================================================
// Async Errors Tests
// ============================================================================

func TestAsyncError_Error(t *testing.T) {
	err := NewAsyncError(AsyncOpBatchSend, errors.New("connection refused"))

	got := err.Error()
	if got == "" {
		t.Error("Error() returned empty string")
	}
	if !strings.Contains(got, "batch_send") {
		t.Errorf("Error() = %q, want to contain 'batch_send'", got)
	}
	if !strings.Contains(got, "connection refused") {
		t.Errorf("Error() = %q, want to contain 'connection refused'", got)
	}
}

func TestAsyncError_ErrorWithEventIDs(t *testing.T) {
	err := NewAsyncError(AsyncOpBatchSend, errors.New("failed")).
		WithEventIDs("id1", "id2", "id3")

	got := err.Error()
	if !strings.Contains(got, "3 events affected") {
		t.Errorf("Error() = %q, want to contain '3 events affected'", got)
	}
}

func TestAsyncError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := NewAsyncError(AsyncOpFlush, underlying)

	if errors.Unwrap(err) != underlying {
		t.Error("Unwrap() did not return underlying error")
	}
}

func TestAsyncError_WithMethods(t *testing.T) {
	err := NewAsyncError(AsyncOpQueue, errors.New("queue full")).
		WithRetryable(true).
		WithContext("size", 1000).
		WithContext("capacity", 1000).
		WithEventIDs("event-1")

	if !err.Retryable {
		t.Error("WithRetryable(true) did not set Retryable")
	}
	if err.Context["size"] != 1000 {
		t.Error("WithContext did not set context value")
	}
	if err.Context["capacity"] != 1000 {
		t.Error("WithContext did not set second context value")
	}
	if len(err.EventIDs) != 1 || err.EventIDs[0] != "event-1" {
		t.Error("WithEventIDs did not set event IDs")
	}
}

func TestAsyncErrorHandler_Creation(t *testing.T) {
	h := NewAsyncErrorHandler(nil)

	if h.Errors == nil {
		t.Error("Errors channel is nil")
	}
	if cap(h.Errors) != 100 {
		t.Errorf("Errors channel capacity = %d, want 100", cap(h.Errors))
	}
}

func TestAsyncErrorHandler_CustomBufferSize(t *testing.T) {
	h := NewAsyncErrorHandler(&AsyncErrorConfig{
		BufferSize: 50,
	})

	if cap(h.Errors) != 50 {
		t.Errorf("Errors channel capacity = %d, want 50", cap(h.Errors))
	}
}

func TestAsyncErrorHandler_Handle(t *testing.T) {
	h := NewAsyncErrorHandler(&AsyncErrorConfig{
		BufferSize: 10,
	})

	err := NewAsyncError(AsyncOpBatchSend, errors.New("test error"))
	h.Handle(err)

	if h.TotalErrors() != 1 {
		t.Errorf("TotalErrors() = %d, want 1", h.TotalErrors())
	}

	if h.Pending() != 1 {
		t.Errorf("Pending() = %d, want 1", h.Pending())
	}

	// Receive the error
	select {
	case received := <-h.Errors:
		if received != err {
			t.Error("received different error than sent")
		}
	default:
		t.Error("no error in channel")
	}
}

func TestAsyncErrorHandler_HandleNil(t *testing.T) {
	h := NewAsyncErrorHandler(nil)
	h.Handle(nil)

	if h.TotalErrors() != 0 {
		t.Errorf("TotalErrors() = %d after handling nil, want 0", h.TotalErrors())
	}
}

func TestAsyncErrorHandler_Callback(t *testing.T) {
	var received *AsyncError
	var mu sync.Mutex

	h := NewAsyncErrorHandler(&AsyncErrorConfig{
		OnError: func(err *AsyncError) {
			mu.Lock()
			received = err
			mu.Unlock()
		},
	})

	err := NewAsyncError(AsyncOpFlush, errors.New("callback test"))
	h.Handle(err)

	mu.Lock()
	got := received
	mu.Unlock()

	if got != err {
		t.Error("callback did not receive error")
	}
}

func TestAsyncErrorHandler_SetCallback(t *testing.T) {
	h := NewAsyncErrorHandler(nil)

	var called bool
	h.SetCallback(func(err *AsyncError) {
		called = true
	})

	h.Handle(NewAsyncError(AsyncOpInternal, errors.New("test")))

	if !called {
		t.Error("SetCallback callback was not called")
	}
}

func TestAsyncErrorHandler_Overflow(t *testing.T) {
	var overflowCount int
	var mu sync.Mutex

	h := NewAsyncErrorHandler(&AsyncErrorConfig{
		BufferSize: 2,
		OnOverflow: func(dropped int) {
			mu.Lock()
			overflowCount = dropped
			mu.Unlock()
		},
	})

	// Fill the buffer
	h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("1")))
	h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("2")))

	// This should overflow
	h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("3")))

	if h.DroppedCount() != 1 {
		t.Errorf("DroppedCount() = %d, want 1", h.DroppedCount())
	}

	mu.Lock()
	got := overflowCount
	mu.Unlock()

	if got != 1 {
		t.Errorf("overflow callback received %d, want 1", got)
	}
}

func TestAsyncErrorHandler_Drain(t *testing.T) {
	h := NewAsyncErrorHandler(&AsyncErrorConfig{
		BufferSize: 10,
	})

	// Add errors
	for i := 0; i < 5; i++ {
		h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("test")))
	}

	drainedErrors := h.Drain()
	if len(drainedErrors) != 5 {
		t.Errorf("Drain() returned %d errors, want 5", len(drainedErrors))
	}

	if h.Pending() != 0 {
		t.Errorf("Pending() after Drain() = %d, want 0", h.Pending())
	}
}

func TestAsyncErrorHandler_Stats(t *testing.T) {
	h := NewAsyncErrorHandler(&AsyncErrorConfig{
		BufferSize: 10,
	})

	h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("1")))
	h.Handle(NewAsyncError(AsyncOpFlush, errors.New("2")))

	stats := h.Stats()
	if stats.TotalErrors != 2 {
		t.Errorf("Stats().TotalErrors = %d, want 2", stats.TotalErrors)
	}
	if stats.BufferSize != 10 {
		t.Errorf("Stats().BufferSize = %d, want 10", stats.BufferSize)
	}
	if stats.Pending != 2 {
		t.Errorf("Stats().Pending = %d, want 2", stats.Pending)
	}
}

func TestAsyncErrorHandler_ErrorsByOperation(t *testing.T) {
	h := NewAsyncErrorHandler(nil)

	h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("1")))
	h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("2")))
	h.Handle(NewAsyncError(AsyncOpFlush, errors.New("3")))

	if h.ErrorsByOperation(AsyncOpBatchSend) != 2 {
		t.Errorf("ErrorsByOperation(batch_send) = %d, want 2", h.ErrorsByOperation(AsyncOpBatchSend))
	}
	if h.ErrorsByOperation(AsyncOpFlush) != 1 {
		t.Errorf("ErrorsByOperation(flush) = %d, want 1", h.ErrorsByOperation(AsyncOpFlush))
	}
	if h.ErrorsByOperation(AsyncOpHook) != 0 {
		t.Errorf("ErrorsByOperation(hook) = %d, want 0", h.ErrorsByOperation(AsyncOpHook))
	}
}

func TestAsyncErrorHandler_ConcurrentAccess(t *testing.T) {
	h := NewAsyncErrorHandler(&AsyncErrorConfig{
		BufferSize: 1000,
	})

	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("concurrent")))
				_ = h.TotalErrors()
				_ = h.DroppedCount()
				_ = h.Pending()
				_ = h.Stats()
			}
		}()
	}

	wg.Wait()

	if h.TotalErrors() != goroutines*iterations {
		t.Errorf("TotalErrors() = %d, want %d", h.TotalErrors(), goroutines*iterations)
	}
}

func TestAsyncErrorHandler_Close(t *testing.T) {
	h := NewAsyncErrorHandler(nil)

	h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("test")))
	h.Close()

	// Channel should be closed
	_, ok := <-h.Errors
	if ok {
		// First receive should get the pending error
		_, ok = <-h.Errors
		if ok {
			t.Error("channel should be closed after second receive")
		}
	}
}

func TestAsyncErrorHandler_WithMetrics(t *testing.T) {
	metrics := &testMetrics{}
	h := NewAsyncErrorHandler(&AsyncErrorConfig{
		Metrics: metrics,
	})

	h.Handle(NewAsyncError(AsyncOpBatchSend, errors.New("test")).WithRetryable(true))

	if metrics.counters["langfuse.async_errors.total"] != 1 {
		t.Error("metrics.IncrementCounter not called for total")
	}
	if metrics.counters["langfuse.async_errors.batch_send"] != 1 {
		t.Error("metrics.IncrementCounter not called for operation")
	}
	if metrics.counters["langfuse.async_errors.retryable"] != 1 {
		t.Error("metrics.IncrementCounter not called for retryable")
	}
}

func TestWrapAsyncError_Nil(t *testing.T) {
	if WrapAsyncError(AsyncOpBatchSend, nil) != nil {
		t.Error("WrapAsyncError(nil) should return nil")
	}
}

func TestWrapAsyncError_AlreadyAsyncError(t *testing.T) {
	original := NewAsyncError(AsyncOpFlush, errors.New("original"))
	wrapped := WrapAsyncError(AsyncOpBatchSend, original)

	if wrapped != original {
		t.Error("WrapAsyncError should return existing AsyncError as-is")
	}
}

func TestWrapAsyncError_RegularError(t *testing.T) {
	regular := errors.New("regular error")
	wrapped := WrapAsyncError(AsyncOpBatchSend, regular)

	if wrapped == nil {
		t.Fatal("WrapAsyncError returned nil")
	}
	if wrapped.Operation != AsyncOpBatchSend {
		t.Errorf("Operation = %v, want batch_send", wrapped.Operation)
	}
	if wrapped.Err != regular {
		t.Error("Err should be the original error")
	}
}

func TestAsyncError_Time(t *testing.T) {
	before := time.Now()
	err := NewAsyncError(AsyncOpBatchSend, errors.New("test"))
	after := time.Now()

	if err.Time.Before(before) || err.Time.After(after) {
		t.Error("AsyncError.Time is not within expected range")
	}
}
