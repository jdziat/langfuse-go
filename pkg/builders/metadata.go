package builders

import (
	"time"

	"github.com/jdziat/langfuse-go/pkg/types"
)

// Metadata is a type alias for pkg/types.Metadata.
type Metadata = types.Metadata

// MetadataBuilder provides a type-safe way to build metadata with
// typed setter methods. This complements the Metadata type's Set() method
// by providing methods like String(), Int(), Float(), etc.
//
// Example:
//
//	metadata := BuildMetadata().
//	    String("user_id", "123").
//	    Int("request_count", 5).
//	    Bool("is_premium", true).
//	    Float("score", 0.95).
//	    Build()
//
//	trace.Metadata(metadata).Create(ctx)
type MetadataBuilder struct {
	data Metadata
}

// BuildMetadata creates a new MetadataBuilder with typed setter methods.
// For simple metadata construction, you can also use NewMetadata().Set() directly.
func BuildMetadata() *MetadataBuilder {
	return &MetadataBuilder{data: make(Metadata)}
}

// String adds a string value to the metadata.
func (m *MetadataBuilder) String(key, value string) *MetadataBuilder {
	m.data[key] = value
	return m
}

// Int adds an integer value to the metadata.
func (m *MetadataBuilder) Int(key string, value int) *MetadataBuilder {
	m.data[key] = value
	return m
}

// Int64 adds an int64 value to the metadata.
func (m *MetadataBuilder) Int64(key string, value int64) *MetadataBuilder {
	m.data[key] = value
	return m
}

// Float adds a float64 value to the metadata.
func (m *MetadataBuilder) Float(key string, value float64) *MetadataBuilder {
	m.data[key] = value
	return m
}

// Bool adds a boolean value to the metadata.
func (m *MetadataBuilder) Bool(key string, value bool) *MetadataBuilder {
	m.data[key] = value
	return m
}

// Time adds a time value to the metadata (formatted as RFC3339).
func (m *MetadataBuilder) Time(key string, value time.Time) *MetadataBuilder {
	m.data[key] = value.Format(time.RFC3339Nano)
	return m
}

// Duration adds a duration value to the metadata (as string).
func (m *MetadataBuilder) Duration(key string, value time.Duration) *MetadataBuilder {
	m.data[key] = value.String()
	return m
}

// DurationMs adds a duration value as milliseconds.
func (m *MetadataBuilder) DurationMs(key string, value time.Duration) *MetadataBuilder {
	m.data[key] = value.Milliseconds()
	return m
}

// JSON adds an arbitrary JSON-serializable value to the metadata.
func (m *MetadataBuilder) JSON(key string, value any) *MetadataBuilder {
	m.data[key] = value
	return m
}

// Strings adds a string slice to the metadata.
func (m *MetadataBuilder) Strings(key string, values []string) *MetadataBuilder {
	m.data[key] = values
	return m
}

// Map adds a nested map to the metadata.
func (m *MetadataBuilder) Map(key string, value map[string]any) *MetadataBuilder {
	m.data[key] = value
	return m
}

// Merge merges another metadata map into this builder.
// Existing keys are overwritten.
func (m *MetadataBuilder) Merge(other Metadata) *MetadataBuilder {
	for k, v := range other {
		m.data[k] = v
	}
	return m
}

// Build returns the constructed metadata map.
func (m *MetadataBuilder) Build() Metadata {
	return m.data
}
