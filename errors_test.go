package langfuse

import (
	"testing"
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
