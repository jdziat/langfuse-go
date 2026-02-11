package types

// Metadata provides type-safe metadata storage with JSON serialization.
// This replaces raw map[string]any for better type clarity.
type Metadata map[string]any

// NewMetadata creates a new empty Metadata instance.
func NewMetadata() Metadata {
	return make(Metadata)
}

// Set sets a key-value pair in the metadata.
// Returns the Metadata for method chaining.
func (m Metadata) Set(key string, value any) Metadata {
	m[key] = value
	return m
}

// Get retrieves a value from the metadata.
// Returns the value and true if found, nil and false otherwise.
func (m Metadata) Get(key string) (any, bool) {
	v, ok := m[key]
	return v, ok
}

// GetString retrieves a string value from the metadata.
// Returns the string and true if found and is a string, empty string and false otherwise.
func (m Metadata) GetString(key string) (string, bool) {
	v, ok := m[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// GetInt retrieves an int value from the metadata.
// Returns the int and true if found and is an int, 0 and false otherwise.
// Also handles float64 values (common from JSON unmarshaling).
func (m Metadata) GetInt(key string) (int, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case int:
		return n, true
	case float64:
		return int(n), true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

// GetFloat retrieves a float64 value from the metadata.
// Returns the float64 and true if found and is numeric, 0 and false otherwise.
func (m Metadata) GetFloat(key string) (float64, bool) {
	v, ok := m[key]
	if !ok {
		return 0, false
	}
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// GetBool retrieves a bool value from the metadata.
// Returns the bool and true if found and is a bool, false and false otherwise.
func (m Metadata) GetBool(key string) (bool, bool) {
	v, ok := m[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// Has returns true if the key exists in the metadata.
func (m Metadata) Has(key string) bool {
	_, ok := m[key]
	return ok
}

// Delete removes a key from the metadata.
// Returns the Metadata for method chaining.
func (m Metadata) Delete(key string) Metadata {
	delete(m, key)
	return m
}

// Merge merges another Metadata into this one.
// Values from other will overwrite values in m for duplicate keys.
// Returns the Metadata for method chaining.
func (m Metadata) Merge(other Metadata) Metadata {
	for k, v := range other {
		m[k] = v
	}
	return m
}

// Clone creates a shallow copy of the metadata.
func (m Metadata) Clone() Metadata {
	clone := make(Metadata, len(m))
	for k, v := range m {
		clone[k] = v
	}
	return clone
}

// Keys returns all keys in the metadata.
func (m Metadata) Keys() []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

// Len returns the number of entries in the metadata.
func (m Metadata) Len() int {
	return len(m)
}

// IsEmpty returns true if the metadata has no entries.
func (m Metadata) IsEmpty() bool {
	return len(m) == 0
}

// Filter returns a new Metadata containing only the specified keys.
// Keys that don't exist in the source metadata are ignored.
func (m Metadata) Filter(keys ...string) Metadata {
	result := make(Metadata, len(keys))
	for _, k := range keys {
		if v, ok := m[k]; ok {
			result[k] = v
		}
	}
	return result
}
