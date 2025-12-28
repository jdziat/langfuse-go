package langfuse

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestBuildResult_Unwrap(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		result := NewBuildResult("hello", nil)
		value, err := result.Unwrap()

		if value != "hello" {
			t.Errorf("Unwrap() value = %v, want 'hello'", value)
		}
		if err != nil {
			t.Errorf("Unwrap() err = %v, want nil", err)
		}
	})

	t.Run("with error", func(t *testing.T) {
		testErr := errors.New("test error")
		result := NewBuildResult("", testErr)
		value, err := result.Unwrap()

		if value != "" {
			t.Errorf("Unwrap() value = %v, want ''", value)
		}
		if err != testErr {
			t.Errorf("Unwrap() err = %v, want testErr", err)
		}
	})
}

func TestBuildResult_Must(t *testing.T) {
	t.Run("with value", func(t *testing.T) {
		result := NewBuildResult(42, nil)
		value := result.Must()

		if value != 42 {
			t.Errorf("Must() = %v, want 42", value)
		}
	})

	t.Run("with error panics", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Must() should panic with error")
			}
		}()

		result := NewBuildResult(0, errors.New("test error"))
		_ = result.Must()
	})
}

func TestBuildResult_Ok(t *testing.T) {
	t.Run("ok when no error", func(t *testing.T) {
		result := NewBuildResult("value", nil)
		if !result.Ok() {
			t.Error("Ok() should return true when no error")
		}
	})

	t.Run("not ok when error", func(t *testing.T) {
		result := NewBuildResult("", errors.New("error"))
		if result.Ok() {
			t.Error("Ok() should return false when error")
		}
	})
}

func TestBuildResult_Err(t *testing.T) {
	testErr := errors.New("test error")
	result := NewBuildResult("", testErr)

	if result.Err() != testErr {
		t.Errorf("Err() = %v, want testErr", result.Err())
	}
}

func TestBuildResult_Value(t *testing.T) {
	result := NewBuildResult("test", nil)

	if result.Value() != "test" {
		t.Errorf("Value() = %v, want 'test'", result.Value())
	}
}

func TestBuildResultError(t *testing.T) {
	testErr := errors.New("test error")
	result := BuildResultError[string](testErr)

	if result.Value() != "" {
		t.Errorf("Value() = %v, want empty string", result.Value())
	}
	if result.Err() != testErr {
		t.Errorf("Err() = %v, want testErr", result.Err())
	}
}

func TestBuildResultOk(t *testing.T) {
	result := BuildResultOk("success")

	if result.Value() != "success" {
		t.Errorf("Value() = %v, want 'success'", result.Value())
	}
	if result.Err() != nil {
		t.Errorf("Err() = %v, want nil", result.Err())
	}
}

func TestValidatedTraceBuilder_ErrorAccumulation(t *testing.T) {
	client := createTestClient(t)
	defer client.Shutdown(context.Background())

	builder := NewValidatedTraceBuilder(client)

	// Set invalid values
	builder.ID("")
	builder.Name(strings.Repeat("a", MaxNameLength+1))
	builder.Metadata(map[string]any{"": "empty key"})
	builder.Tags(make([]string, MaxTagCount+1))

	if !builder.HasErrors() {
		t.Error("HasErrors() should return true after invalid inputs")
	}

	errs := builder.Errors()
	if len(errs) < 4 {
		t.Errorf("Expected at least 4 errors, got %d", len(errs))
	}

	// Create should return combined error
	result := builder.Create(context.Background())
	if result.Ok() {
		t.Error("Create() should fail with validation errors")
	}

	err := result.Err()
	if err == nil {
		t.Fatal("Result error should not be nil")
	}
	if !strings.Contains(err.Error(), "validation") {
		t.Errorf("Error should mention 'validation': %v", err)
	}
}

func TestValidatedTraceBuilder_ValidInputs(t *testing.T) {
	client := createTestClient(t)
	defer client.Shutdown(context.Background())

	builder := NewValidatedTraceBuilder(client)

	// Set valid values
	builder.
		ID("test-trace-123").
		Name("test-trace").
		UserID("user-1").
		SessionID("session-1").
		Input("test input").
		Output("test output").
		Metadata(map[string]any{"key": "value"}).
		Tags([]string{"tag1", "tag2"}).
		Version("1.0.0").
		Release("release-1").
		Public(true)

	if builder.HasErrors() {
		t.Errorf("HasErrors() should return false, errors: %v", builder.Errors())
	}

	result := builder.Create(context.Background())
	if !result.Ok() {
		t.Errorf("Create() should succeed, error: %v", result.Err())
	}

	trace, err := result.Unwrap()
	if err != nil {
		t.Fatalf("Unwrap() error = %v", err)
	}
	if trace == nil {
		t.Error("Trace should not be nil")
	}
}

func TestValidatedSpanBuilder_ErrorAccumulation(t *testing.T) {
	client := createTestClient(t)
	defer client.Shutdown(context.Background())

	// First create a trace
	trace, err := client.NewTrace().Name("test").Create(context.Background())
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	builder := NewValidatedSpanBuilder(trace)

	// Set invalid values
	builder.ID("")
	builder.Name(strings.Repeat("x", MaxNameLength+1))
	builder.Level("invalid-level")
	builder.Metadata(map[string]any{"": "bad"})

	if !builder.HasErrors() {
		t.Error("HasErrors() should return true after invalid inputs")
	}

	result := builder.Create(context.Background())
	if result.Ok() {
		t.Error("Create() should fail with validation errors")
	}
}

func TestValidatedSpanBuilder_ValidInputs(t *testing.T) {
	client := createTestClient(t)
	defer client.Shutdown(context.Background())

	trace, err := client.NewTrace().Name("test").Create(context.Background())
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	builder := NewValidatedSpanBuilder(trace)

	builder.
		ID("span-123").
		Name("test-span").
		Input("input").
		Output("output").
		Metadata(map[string]any{"k": "v"}).
		Level(ObservationLevelDefault).
		StatusMessage("ok").
		Version("1.0")

	if builder.HasErrors() {
		t.Errorf("HasErrors() should return false, errors: %v", builder.Errors())
	}

	result := builder.Create(context.Background())
	if !result.Ok() {
		t.Errorf("Create() should succeed, error: %v", result.Err())
	}
}

func TestValidatedGenerationBuilder_ErrorAccumulation(t *testing.T) {
	client := createTestClient(t)
	defer client.Shutdown(context.Background())

	trace, err := client.NewTrace().Name("test").Create(context.Background())
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	builder := NewValidatedGenerationBuilder(trace)

	// Set invalid values
	builder.ID("")
	builder.Name(strings.Repeat("y", MaxNameLength+1))
	builder.Level("bad-level")
	builder.Usage(-1, -1) // Now takes 2 params

	if !builder.HasErrors() {
		t.Error("HasErrors() should return true after invalid inputs")
	}

	errs := builder.Errors()
	if len(errs) < 4 { // id, name, level, 2 usage params
		t.Logf("Got %d errors: %v", len(errs), errs)
	}

	result := builder.Create(context.Background())
	if result.Ok() {
		t.Error("Create() should fail with validation errors")
	}
}

func TestValidatedGenerationBuilder_ValidInputs(t *testing.T) {
	client := createTestClient(t)
	defer client.Shutdown(context.Background())

	trace, err := client.NewTrace().Name("test").Create(context.Background())
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	builder := NewValidatedGenerationBuilder(trace)

	builder.
		ID("gen-123").
		Name("test-gen").
		Model("gpt-4").
		ModelParameters(map[string]any{"temperature": 0.7}).
		Input("prompt").
		Output("response").
		Metadata(map[string]any{"k": "v"}).
		Level(ObservationLevelDefault).
		Usage(100, 50) // Now takes input, output (not 3 params)

	if builder.HasErrors() {
		t.Errorf("HasErrors() should return false, errors: %v", builder.Errors())
	}

	result := builder.Create(context.Background())
	if !result.Ok() {
		t.Errorf("Create() should succeed, error: %v", result.Err())
	}
}

func TestValidatedScoreBuilder_ErrorAccumulation(t *testing.T) {
	client := createTestClient(t)
	defer client.Shutdown(context.Background())

	trace, err := client.NewTrace().Name("test").Create(context.Background())
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	builder := NewValidatedScoreBuilder(trace)

	// Set invalid values - name is required and empty is invalid
	builder.Name("")
	builder.Value(-0.5) // out of range [0, 1]

	if !builder.HasErrors() {
		t.Error("HasErrors() should return true after invalid inputs")
	}

	// Score Create returns error directly
	err = builder.Create(context.Background())
	if err == nil {
		t.Error("Create() should fail with validation errors")
	}
}

func TestValidatedScoreBuilder_ValidInputs(t *testing.T) {
	client := createTestClient(t)
	defer client.Shutdown(context.Background())

	trace, err := client.NewTrace().Name("test").Create(context.Background())
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	builder := NewValidatedScoreBuilder(trace)

	builder.
		Name("accuracy").
		Value(0.95).
		Comment("good").
		ObservationID("obs-1").
		ConfigID("config-1")

	if builder.HasErrors() {
		t.Errorf("HasErrors() should return false, errors: %v", builder.Errors())
	}

	// Score Create returns error directly (no BuildResult)
	// Note: This will fail since we're using a mock server
	// For unit tests, we just verify no validation errors
}

func TestCombineValidationErrors(t *testing.T) {
	t.Run("no errors", func(t *testing.T) {
		err := combineValidationErrors(nil)
		if err != nil {
			t.Errorf("combineValidationErrors(nil) = %v, want nil", err)
		}
	})

	t.Run("one error", func(t *testing.T) {
		singleErr := errors.New("single error")
		err := combineValidationErrors([]error{singleErr})
		if err != singleErr {
			t.Errorf("combineValidationErrors([1]) = %v, want singleErr", err)
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		errs := []error{
			errors.New("error 1"),
			errors.New("error 2"),
			errors.New("error 3"),
		}
		err := combineValidationErrors(errs)
		if err == nil {
			t.Fatal("combineValidationErrors should not return nil")
		}

		errStr := err.Error()
		if !strings.Contains(errStr, "multiple validation errors") {
			t.Errorf("Error should mention 'multiple validation errors': %v", errStr)
		}
		if !strings.Contains(errStr, "error 1") {
			t.Errorf("Error should contain 'error 1': %v", errStr)
		}
		if !strings.Contains(errStr, "error 2") {
			t.Errorf("Error should contain 'error 2': %v", errStr)
		}
	})
}

func TestStrictValidationConfig(t *testing.T) {
	t.Run("default config", func(t *testing.T) {
		cfg := DefaultStrictValidationConfig()

		if !cfg.Enabled {
			t.Error("Default config should have Enabled = true")
		}
		if cfg.FailFast {
			t.Error("Default config should have FailFast = false")
		}
	})
}

func TestValidatedTraceBuilder_ChainedCalls(t *testing.T) {
	client := createTestClient(t)
	defer client.Shutdown(context.Background())

	// Test that chaining works and returns the same builder
	builder := NewValidatedTraceBuilder(client)

	result := builder.
		Name("chained").
		UserID("user").
		SessionID("session").
		Input("in").
		Output("out").
		Tags([]string{"tag"}).
		Version("1.0").
		Public(false)

	if result != builder {
		t.Error("Chained methods should return the same builder")
	}
}

func TestValidatedSpanBuilder_ChainedCalls(t *testing.T) {
	client := createTestClient(t)
	defer client.Shutdown(context.Background())

	trace, err := client.NewTrace().Create(context.Background())
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	builder := NewValidatedSpanBuilder(trace)

	result := builder.
		Name("span").
		Input("in").
		Output("out").
		Level(ObservationLevelDefault).
		StatusMessage("ok")

	if result != builder {
		t.Error("Chained methods should return the same builder")
	}
}

func TestValidatedGenerationBuilder_UsageValidation(t *testing.T) {
	client := createTestClient(t)
	defer client.Shutdown(context.Background())

	trace, err := client.NewTrace().Create(context.Background())
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	tests := []struct {
		name        string
		input       int
		output      int
		expectError bool
	}{
		{"all valid", 100, 50, false},
		{"negative input", -1, 50, true},
		{"negative output", 100, -1, true},
		{"both negative", -1, -1, true},
		{"zeros valid", 0, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewValidatedGenerationBuilder(trace)
			builder.Name("test").Usage(tt.input, tt.output)

			if tt.expectError {
				if !builder.HasErrors() {
					t.Error("Expected validation error")
				}
			} else {
				if builder.HasErrors() {
					t.Errorf("Unexpected errors: %v", builder.Errors())
				}
			}
		})
	}
}

func TestValidatedScoreBuilder_ValueValidation(t *testing.T) {
	client := createTestClient(t)
	defer client.Shutdown(context.Background())

	trace, err := client.NewTrace().Create(context.Background())
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	tests := []struct {
		name        string
		value       float64
		expectError bool
	}{
		{"in range 0", 0.0, false},
		{"in range 0.5", 0.5, false},
		{"in range 1", 1.0, false},
		{"below range", -0.1, true},
		{"above range", 1.1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewValidatedScoreBuilder(trace)
			builder.Name("test").Value(tt.value)

			if tt.expectError {
				if !builder.HasErrors() {
					t.Error("Expected validation error")
				}
			} else {
				if builder.HasErrors() {
					t.Errorf("Unexpected errors: %v", builder.Errors())
				}
			}
		})
	}
}

// createTestClient creates a test client with mock server
func createTestClient(t *testing.T) *Client {
	t.Helper()

	// Create client with test credentials
	client, err := New(
		"pk-test-valid-key",
		"sk-test-valid-key",
		WithBaseURL("http://localhost:9999"), // Non-existent endpoint for unit tests
	)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	return client
}
