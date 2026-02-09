package langfuse

import (
	"testing"

	pkgconfig "github.com/jdziat/langfuse-go/pkg/config"
	pkgerrors "github.com/jdziat/langfuse-go/pkg/errors"
	pkghttp "github.com/jdziat/langfuse-go/pkg/http"
	pkgingestion "github.com/jdziat/langfuse-go/pkg/ingestion"
)

// TestFacadeTypeAliases verifies that facade type aliases work correctly.
func TestFacadeTypeAliases(t *testing.T) {
	t.Run("pkg/config exports", func(t *testing.T) {
		// Test that environment helper functions are accessible
		if GetEnvString == nil {
			t.Error("GetEnvString should be exported")
		}
		if GetEnvBool == nil {
			t.Error("GetEnvBool should be exported")
		}
		if GetEnvRegion == nil {
			t.Error("GetEnvRegion should be exported")
		}
	})

	t.Run("pkg/errors type aliases", func(t *testing.T) {
		// Test type compatibility
		var asyncErr *PkgAsyncError
		var pkgAsyncErr *pkgerrors.AsyncError
		asyncErr = pkgAsyncErr
		if asyncErr == nil {
			// This is just to use the variable
		}

		var apiErr *PkgAPIError
		var pkgAPIErr *pkgerrors.APIError
		apiErr = pkgAPIErr
		if apiErr == nil {
			// This is just to use the variable
		}

		var valErr *PkgValidationError
		var pkgValErr *pkgerrors.ValidationError
		valErr = pkgValErr
		if valErr == nil {
			// This is just to use the variable
		}
	})

	t.Run("pkg/http type aliases", func(t *testing.T) {
		// Test type compatibility
		var backoff *PkgExponentialBackoff
		var pkgBackoff *pkghttp.ExponentialBackoff
		backoff = pkgBackoff
		if backoff == nil {
			// This is just to use the variable
		}

		var cb *PkgCircuitBreaker
		var pkgCB *pkghttp.CircuitBreaker
		cb = pkgCB
		if cb == nil {
			// This is just to use the variable
		}
	})

	t.Run("pkg/ingestion type aliases", func(t *testing.T) {
		// Test type compatibility
		var monitor *PkgQueueMonitor
		var pkgMonitor *pkgingestion.QueueMonitor
		monitor = pkgMonitor
		if monitor == nil {
			// This is just to use the variable
		}

		var handler *PkgBackpressureHandler
		var pkgHandler *pkgingestion.BackpressureHandler
		handler = pkgHandler
		if handler == nil {
			// This is just to use the variable
		}
	})

	t.Run("constructor functions", func(t *testing.T) {
		// Test that constructor functions are accessible
		if NewPkgExponentialBackoff == nil {
			t.Error("NewPkgExponentialBackoff should be exported")
		}
		if NewPkgCircuitBreaker == nil {
			t.Error("NewPkgCircuitBreaker should be exported")
		}
		if NewPkgQueueMonitor == nil {
			t.Error("NewPkgQueueMonitor should be exported")
		}
		if NewPkgBackpressureHandler == nil {
			t.Error("NewPkgBackpressureHandler should be exported")
		}
		if PkgUUID == nil {
			t.Error("PkgUUID should be exported")
		}
		if PkgGenerateID == nil {
			t.Error("PkgGenerateID should be exported")
		}
	})

	t.Run("helper functions", func(t *testing.T) {
		// Test that helper functions are accessible
		if PkgIsRetryable == nil {
			t.Error("PkgIsRetryable should be exported")
		}
		if PkgAsAPIError == nil {
			t.Error("PkgAsAPIError should be exported")
		}
		if PkgWrapError == nil {
			t.Error("PkgWrapError should be exported")
		}
		if PkgWrapErrorf == nil {
			t.Error("PkgWrapErrorf should be exported")
		}
	})
}

// TestFacadeUsageExample demonstrates using the facade types.
func TestFacadeUsageExample(t *testing.T) {
	t.Run("use pkg types via facade", func(t *testing.T) {
		// Create a circuit breaker using the facade
		cb := NewPkgCircuitBreaker(pkghttp.CircuitBreakerConfig{
			FailureThreshold: 3,
		})

		if cb.State() != pkghttp.CircuitClosed {
			t.Errorf("expected circuit to be closed, got %v", cb.State())
		}

		// Create a retry strategy using the facade
		backoff := NewPkgExponentialBackoff()
		if backoff == nil {
			t.Error("expected non-nil backoff")
		}

		// Generate an ID using the facade
		id := PkgGenerateID()
		if id == "" {
			t.Error("expected non-empty ID")
		}

		// Use environment helpers
		val := GetEnvString("NONEXISTENT_VAR", "default")
		if val != "default" {
			t.Errorf("expected 'default', got %q", val)
		}
	})

	t.Run("use pkg error helpers via facade", func(t *testing.T) {
		// Create a validation error using pkg
		err := NewPkgValidationError("field", "invalid value")
		if err == nil {
			t.Fatal("expected non-nil error")
		}

		// Extract it using the facade helper
		valErr, ok := PkgAsValidationError(err)
		if !ok {
			t.Error("expected to extract validation error")
		}
		if valErr.Field != "field" {
			t.Errorf("expected field 'field', got %q", valErr.Field)
		}

		// Test retryability
		if PkgIsRetryable(err) {
			t.Error("validation errors should not be retryable")
		}

		// Test error code
		code := PkgErrorCodeOf(err)
		if code != pkgerrors.ErrCodeValidation {
			t.Errorf("expected ErrCodeValidation, got %v", code)
		}
	})

	t.Run("use pkg config helpers", func(t *testing.T) {
		// Test GetEnvBool
		t.Setenv("TEST_BOOL", "true")
		if !GetEnvBool("TEST_BOOL") {
			t.Error("expected GetEnvBool to return true")
		}

		// Test GetEnvString
		t.Setenv("TEST_STRING", "value")
		if GetEnvString("TEST_STRING", "default") != "value" {
			t.Error("expected GetEnvString to return 'value'")
		}

		// Test GetEnvRegion - note that it reads from LANGFUSE_REGION env var
		t.Setenv("LANGFUSE_REGION", "us")
		region := GetEnvRegion(pkgconfig.RegionEU)
		if region != pkgconfig.RegionUS {
			t.Errorf("expected RegionUS, got %v", region)
		}
	})
}
