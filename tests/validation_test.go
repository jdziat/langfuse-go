package langfuse_test

import (
	"context"
	"strings"
	"testing"

	langfuse "github.com/jdziat/langfuse-go"
)

func TestValidator(t *testing.T) {
	t.Run("initially has no errors", func(t *testing.T) {
		v := &langfuse.Validator{}
		if v.HasErrors() {
			t.Error("new validator should have no errors")
		}
		if len(v.Errors()) != 0 {
			t.Error("new validator should have empty errors slice")
		}
	})

	t.Run("AddError adds error", func(t *testing.T) {
		v := &langfuse.Validator{}
		v.AddError(langfuse.ErrNilRequest)

		if !v.HasErrors() {
			t.Error("should have errors after AddError")
		}
		if len(v.Errors()) != 1 {
			t.Errorf("should have 1 error, got %d", len(v.Errors()))
		}
	})

	t.Run("AddFieldError adds validation error", func(t *testing.T) {
		v := &langfuse.Validator{}
		v.AddFieldError("name", "is required")

		if !v.HasErrors() {
			t.Error("should have errors after AddFieldError")
		}

		err := v.Errors()[0]
		valErr, ok := langfuse.AsValidationError(err)
		if !ok {
			t.Error("error should be ValidationError")
		}
		if valErr.Field != "name" {
			t.Errorf("field = %q, want %q", valErr.Field, "name")
		}
	})

	t.Run("ClearErrors clears all errors", func(t *testing.T) {
		v := &langfuse.Validator{}
		v.AddError(langfuse.ErrNilRequest)
		v.AddFieldError("test", "error")

		v.ClearErrors()

		if v.HasErrors() {
			t.Error("should have no errors after ClearErrors")
		}
	})

	t.Run("CombinedError returns nil for no errors", func(t *testing.T) {
		v := &langfuse.Validator{}
		if v.CombinedError() != nil {
			t.Error("CombinedError should return nil for no errors")
		}
	})

	t.Run("CombinedError returns single error directly", func(t *testing.T) {
		v := &langfuse.Validator{}
		v.AddError(langfuse.ErrNilRequest)

		combined := v.CombinedError()
		if combined != langfuse.ErrNilRequest {
			t.Error("CombinedError should return single error directly")
		}
	})

	t.Run("CombinedError combines multiple errors", func(t *testing.T) {
		v := &langfuse.Validator{}
		v.AddFieldError("name", "is required")
		v.AddFieldError("value", "must be positive")

		combined := v.CombinedError()
		msg := combined.Error()

		if !strings.Contains(msg, "multiple validation errors") {
			t.Error("combined error should mention multiple errors")
		}
		if !strings.Contains(msg, "name") {
			t.Error("combined error should contain first error")
		}
		if !strings.Contains(msg, "value") {
			t.Error("combined error should contain second error")
		}
	})
}

func TestValidateID(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty string", "", true},
		{"valid UUID", "550e8400-e29b-41d4-a716-446655440000", false},
		{"custom ID", "my-custom-id", false},
		{"simple string", "test", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := langfuse.ValidateID("id", tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateIDFormat(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"empty string", "", true},
		{"valid UUID", "550e8400-e29b-41d4-a716-446655440000", false},
		{"valid compact UUID", "550e8400e29b41d4a716446655440000", false},
		{"invalid format", "not-a-uuid", true},
		{"too short", "550e8400", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := langfuse.ValidateIDFormat("id", tt.value)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateIDFormat() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name      string
		value     string
		maxLength int
		wantErr   bool
	}{
		{"empty string", "", 100, false}, // names are optional
		{"valid name", "my-trace", 100, false},
		{"at max length", strings.Repeat("a", 100), 100, false},
		{"exceeds max length", strings.Repeat("a", 101), 100, true},
		{"zero max length means no limit", strings.Repeat("a", 1000), 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := langfuse.ValidateName("name", tt.value, tt.maxLength)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateName() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateMetadata(t *testing.T) {
	tests := []struct {
		name     string
		metadata langfuse.Metadata
		wantErr  bool
	}{
		{"nil metadata", nil, false},
		{"empty metadata", langfuse.Metadata{}, false},
		{"valid metadata", langfuse.Metadata{"key": "value"}, false},
		{"empty key", langfuse.Metadata{"": "value"}, true},
		{"multiple keys with one empty", langfuse.Metadata{"valid": "value", "": "bad"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := langfuse.ValidateMetadata("metadata", tt.metadata)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateMetadata() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTags(t *testing.T) {
	tests := []struct {
		name    string
		tags    []string
		wantErr bool
	}{
		{"nil tags", nil, false},
		{"empty tags", []string{}, false},
		{"valid tags", []string{"production", "v1"}, false},
		{"empty tag", []string{"valid", ""}, true},
		{"first tag empty", []string{"", "valid"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := langfuse.ValidateTags("tags", tt.tags)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTags() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateLevel(t *testing.T) {
	tests := []struct {
		name    string
		level   langfuse.ObservationLevel
		wantErr bool
	}{
		{"empty level", "", false},
		{"debug level", langfuse.ObservationLevelDebug, false},
		{"default level", langfuse.ObservationLevelDefault, false},
		{"warning level", langfuse.ObservationLevelWarning, false},
		{"error level", langfuse.ObservationLevelError, false},
		{"invalid level", langfuse.ObservationLevel("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := langfuse.ValidateLevel("level", tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateLevel() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDataType(t *testing.T) {
	tests := []struct {
		name     string
		dataType langfuse.ScoreDataType
		wantErr  bool
	}{
		{"empty data type", "", false},
		{"numeric", langfuse.ScoreDataTypeNumeric, false},
		{"categorical", langfuse.ScoreDataTypeCategorical, false},
		{"boolean", langfuse.ScoreDataTypeBoolean, false},
		{"invalid", langfuse.ScoreDataType("invalid"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := langfuse.ValidateDataType("dataType", tt.dataType)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDataType() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTraceBuilderValidation(t *testing.T) {
	client, err := langfuse.New("pk-lf-test-key", "sk-lf-test-key")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if client != nil {
			_ = client.Close(context.Background())
		}
	}()

	t.Run("valid trace has no errors", func(t *testing.T) {
		builder := client.NewTrace().
			Name("test-trace").
			Tags([]string{"production"}).
			Metadata(langfuse.Metadata{"key": "value"})

		if builder.HasErrors() {
			t.Error("valid trace should have no errors")
		}
	})

	t.Run("empty tag triggers error", func(t *testing.T) {
		builder := client.NewTrace().
			Name("test-trace").
			Tags([]string{"valid", ""})

		if !builder.HasErrors() {
			t.Error("empty tag should trigger error")
		}
	})

	t.Run("empty metadata key triggers error", func(t *testing.T) {
		builder := client.NewTrace().
			Name("test-trace").
			Metadata(langfuse.Metadata{"": "value"})

		if !builder.HasErrors() {
			t.Error("empty metadata key should trigger error")
		}
	})

	t.Run("too many tags triggers error", func(t *testing.T) {
		tags := make([]string, langfuse.MaxTagCount+1)
		for i := range tags {
			tags[i] = "tag"
		}

		builder := client.NewTrace().
			Name("test-trace").
			Tags(tags)

		if !builder.HasErrors() {
			t.Error("too many tags should trigger error")
		}
	})

	t.Run("Validate returns accumulated errors", func(t *testing.T) {
		builder := client.NewTrace().
			Tags([]string{""}).
			Metadata(langfuse.Metadata{"": "value"})

		err := builder.Validate()
		if err == nil {
			t.Error("Validate should return error")
		}
	})
}

func TestScoreBuilderValidationOnSet(t *testing.T) {
	client, err := langfuse.New("pk-lf-test-key", "sk-lf-test-key")
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer func() {
		if client != nil {
			_ = client.Close(context.Background())
		}
	}()

	trace, err := client.NewTrace().Name("test").Create(context.Background())
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}

	t.Run("valid score has no errors", func(t *testing.T) {
		builder := trace.NewScore().
			Name("quality").
			NumericValue(0.95)

		if builder.HasErrors() {
			t.Error("valid score should have no errors")
		}
	})

	t.Run("empty categorical value triggers error", func(t *testing.T) {
		builder := trace.NewScore().
			Name("category").
			CategoricalValue("")

		if !builder.HasErrors() {
			t.Error("empty categorical value should trigger error")
		}
	})

	t.Run("empty metadata key triggers error", func(t *testing.T) {
		builder := trace.NewScore().
			Name("quality").
			Metadata(langfuse.Metadata{"": "value"})

		if !builder.HasErrors() {
			t.Error("empty metadata key should trigger error")
		}
	})
}
