package langfuse_test

import (
	"testing"

	langfuse "github.com/jdziat/langfuse-go"
	pkgconfig "github.com/jdziat/langfuse-go/pkg/config"
	pkgerrors "github.com/jdziat/langfuse-go/pkg/errors"
	pkghttp "github.com/jdziat/langfuse-go/pkg/http"
)

// TestFacadeTypeAliases verifies that facade type aliases work correctly.
func TestFacadeTypeAliases(t *testing.T) {
	t.Run("pkg/config exports", func(t *testing.T) {
		// Test that environment helper functions are accessible
		if langfuse.GetEnvString == nil {
			t.Error("GetEnvString should be exported")
		}
		if langfuse.GetEnvBool == nil {
			t.Error("GetEnvBool should be exported")
		}
		if langfuse.GetEnvRegion == nil {
			t.Error("GetEnvRegion should be exported")
		}
	})

	t.Run("pkg/errors type aliases", func(t *testing.T) {
		// Test type compatibility - regular (non-Pkg-prefixed) types
		var asyncErr *langfuse.AsyncError
		var pkgAsyncErr *pkgerrors.AsyncError
		asyncErr = pkgAsyncErr
		if asyncErr == nil {
			// This is just to use the variable
		}

		var apiErr *langfuse.APIError
		var pkgAPIErr *pkgerrors.APIError
		apiErr = pkgAPIErr
		if apiErr == nil {
			// This is just to use the variable
		}

		var valErr *langfuse.ValidationError
		var pkgValErr *pkgerrors.ValidationError
		valErr = pkgValErr
		if valErr == nil {
			// This is just to use the variable
		}
	})

	t.Run("pkg/http type aliases", func(t *testing.T) {
		// Test type compatibility
		var backoff *langfuse.ExponentialBackoff
		var pkgBackoff *pkghttp.ExponentialBackoff
		backoff = pkgBackoff
		if backoff == nil {
			// This is just to use the variable
		}

		var cb *langfuse.CircuitBreaker
		var pkgCB *pkghttp.CircuitBreaker
		cb = pkgCB
		if cb == nil {
			// This is just to use the variable
		}
	})

	t.Run("constructor functions", func(t *testing.T) {
		// Test that constructor functions are accessible
		if langfuse.NewExponentialBackoff == nil {
			t.Error("NewExponentialBackoff should be exported")
		}
		if langfuse.NewCircuitBreaker == nil {
			t.Error("NewCircuitBreaker should be exported")
		}
		if langfuse.NewValidationError == nil {
			t.Error("NewValidationError should be exported")
		}
		if langfuse.NewAsyncError == nil {
			t.Error("NewAsyncError should be exported")
		}
	})

	t.Run("error helper functions", func(t *testing.T) {
		// Test that error helper functions work by calling them
		// IsRetryable should return false for nil error
		if langfuse.IsRetryable(nil) {
			t.Error("IsRetryable(nil) should return false")
		}

		// AsAPIError should return nil, false for nil error
		apiErr, ok := langfuse.AsAPIError(nil)
		if ok || apiErr != nil {
			t.Error("AsAPIError(nil) should return nil, false")
		}

		// AsValidationError should return nil, false for nil error
		valErr, ok := langfuse.AsValidationError(nil)
		if ok || valErr != nil {
			t.Error("AsValidationError(nil) should return nil, false")
		}
	})
}

// TestFacadeUsageExample demonstrates using the facade types.
func TestFacadeUsageExample(t *testing.T) {
	t.Run("use pkg types via facade", func(t *testing.T) {
		// Create a circuit breaker using the facade
		cb := langfuse.NewCircuitBreaker(pkghttp.CircuitBreakerConfig{
			FailureThreshold: 3,
		})

		if cb.State() != pkghttp.CircuitClosed {
			t.Errorf("expected circuit to be closed, got %v", cb.State())
		}

		// Create a retry strategy using the facade
		backoff := langfuse.NewExponentialBackoff()
		if backoff == nil {
			t.Error("expected non-nil backoff")
		}

		// Generate an ID using the facade
		id, err := langfuse.GenerateID()
		if err != nil {
			t.Errorf("GenerateID failed: %v", err)
		}
		if id == "" {
			t.Error("expected non-empty ID")
		}

		// Use environment helpers
		val := langfuse.GetEnvString("NONEXISTENT_VAR", "default")
		if val != "default" {
			t.Errorf("expected 'default', got %q", val)
		}
	})

	t.Run("use pkg error helpers via facade", func(t *testing.T) {
		// Create a validation error using the facade
		err := langfuse.NewValidationError("field", "invalid value")
		if err == nil {
			t.Fatal("expected non-nil error")
		}

		// Extract it using the facade helper
		valErr, ok := langfuse.AsValidationError(err)
		if !ok {
			t.Error("expected to extract validation error")
		}
		if valErr.Field != "field" {
			t.Errorf("expected field 'field', got %q", valErr.Field)
		}

		// Test retryability
		if langfuse.IsRetryable(err) {
			t.Error("validation errors should not be retryable")
		}

		// Test error code
		code := langfuse.ErrorCodeOf(err)
		if code != pkgerrors.ErrCodeValidation {
			t.Errorf("expected ErrCodeValidation, got %v", code)
		}
	})

	t.Run("use pkg config helpers", func(t *testing.T) {
		// Test GetEnvBool
		t.Setenv("TEST_BOOL", "true")
		if !langfuse.GetEnvBool("TEST_BOOL") {
			t.Error("expected GetEnvBool to return true")
		}

		// Test GetEnvString
		t.Setenv("TEST_STRING", "value")
		if langfuse.GetEnvString("TEST_STRING", "default") != "value" {
			t.Error("expected GetEnvString to return 'value'")
		}

		// Test GetEnvRegion - note that it reads from LANGFUSE_REGION env var
		t.Setenv("LANGFUSE_REGION", "us")
		region := langfuse.GetEnvRegion(pkgconfig.RegionEU)
		if region != pkgconfig.RegionUS {
			t.Errorf("expected RegionUS, got %v", region)
		}
	})
}
