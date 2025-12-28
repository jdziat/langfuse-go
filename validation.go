package langfuse

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// ValidationMode controls how validation errors are handled.
type ValidationMode int

const (
	// ValidationModeDeferred collects errors and reports them at Create() time.
	// This is the default mode that maintains the fluent API.
	ValidationModeDeferred ValidationMode = iota

	// ValidationModeImmediate causes setters to store errors immediately.
	// Errors must be checked with HasErrors() before calling Create().
	ValidationModeImmediate
)

// Validator provides validation methods for builder types.
// Builders can embed this to gain validation capabilities.
type Validator struct {
	errors []error
}

// AddError adds a validation error.
func (v *Validator) AddError(err error) {
	v.errors = append(v.errors, err)
}

// AddFieldError adds a validation error for a specific field.
func (v *Validator) AddFieldError(field, message string) {
	v.errors = append(v.errors, NewValidationError(field, message))
}

// HasErrors returns true if there are any validation errors.
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Errors returns all accumulated validation errors.
func (v *Validator) Errors() []error {
	return v.errors
}

// ClearErrors clears all validation errors.
func (v *Validator) ClearErrors() {
	v.errors = nil
}

// CombinedError returns a single error combining all validation errors,
// or nil if there are no errors.
func (v *Validator) CombinedError() error {
	if len(v.errors) == 0 {
		return nil
	}
	if len(v.errors) == 1 {
		return v.errors[0]
	}

	msgs := make([]string, len(v.errors))
	for i, err := range v.errors {
		msgs[i] = err.Error()
	}
	return fmt.Errorf("langfuse: multiple validation errors: %s", strings.Join(msgs, "; "))
}

// Validation rules

// ValidateID validates an ID field.
// IDs must be non-empty and either valid UUIDs or custom identifiers.
func ValidateID(field, value string) error {
	if value == "" {
		return NewValidationError(field, "cannot be empty")
	}
	// Allow any non-empty string as ID (UUIDs and custom IDs are both valid)
	return nil
}

// ValidateIDFormat validates that an ID is a valid UUID format.
// Use this when strict UUID format is required.
func ValidateIDFormat(field, value string) error {
	if value == "" {
		return NewValidationError(field, "cannot be empty")
	}
	if !IsValidUUID(value) {
		return NewValidationError(field, "must be a valid UUID format")
	}
	return nil
}

// ValidateName validates a name field.
// Names must be non-empty and within reasonable length.
func ValidateName(field, value string, maxLength int) error {
	if value == "" {
		return nil // Names are optional
	}
	if maxLength > 0 && utf8.RuneCountInString(value) > maxLength {
		return NewValidationError(field, fmt.Sprintf("exceeds maximum length of %d characters", maxLength))
	}
	return nil
}

// ValidateRequired validates that a required field is not empty.
func ValidateRequired(field, value string) error {
	if value == "" {
		return NewValidationError(field, "is required")
	}
	return nil
}

// ValidatePositive validates that a numeric field is positive.
func ValidatePositive(field string, value int) error {
	if value < 0 {
		return NewValidationError(field, "must be non-negative")
	}
	return nil
}

// ValidateRange validates that a numeric field is within a range.
func ValidateRange(field string, value, min, max int) error {
	if value < min || value > max {
		return NewValidationError(field, fmt.Sprintf("must be between %d and %d", min, max))
	}
	return nil
}

// ValidateMetadata validates metadata fields.
// Checks for nil keys or values that can't be serialized.
func ValidateMetadata(field string, metadata Metadata) error {
	if metadata == nil {
		return nil // nil metadata is valid
	}
	for key := range metadata {
		if key == "" {
			return NewValidationError(field, "metadata keys cannot be empty")
		}
	}
	return nil
}

// ValidateTags validates a tags slice.
func ValidateTags(field string, tags []string) error {
	if tags == nil {
		return nil // nil tags is valid
	}
	for i, tag := range tags {
		if tag == "" {
			return NewValidationError(field, fmt.Sprintf("tag at index %d cannot be empty", i))
		}
	}
	return nil
}

// ValidateLevel validates an observation level.
func ValidateLevel(field string, level ObservationLevel) error {
	switch level {
	case "", ObservationLevelDebug, ObservationLevelDefault, ObservationLevelWarning, ObservationLevelError:
		return nil
	default:
		return NewValidationError(field, fmt.Sprintf("invalid level: %s", level))
	}
}

// ValidateScoreValue validates a score value is within expected range.
func ValidateScoreValue(field string, value float64, min, max float64) error {
	if value < min || value > max {
		return NewValidationError(field, fmt.Sprintf("must be between %.2f and %.2f", min, max))
	}
	return nil
}

// ValidateDataType validates a score data type.
func ValidateDataType(field string, dataType ScoreDataType) error {
	switch dataType {
	case "", ScoreDataTypeNumeric, ScoreDataTypeCategorical, ScoreDataTypeBoolean:
		return nil
	default:
		return NewValidationError(field, fmt.Sprintf("invalid data type: %s", dataType))
	}
}

// MaxNameLength is the maximum allowed length for name fields.
const MaxNameLength = 500

// MaxTagLength is the maximum allowed length for individual tags.
const MaxTagLength = 100

// MaxTagCount is the maximum number of tags allowed.
const MaxTagCount = 50
