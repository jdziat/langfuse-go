package types

import (
	"encoding/json"
	"time"
)

// JSON is an alias for any, representing any JSON value.
// Use this for input/output fields that accept arbitrary JSON data.
//
// Example:
//
//	trace.Generation().
//	    Input(types.JSON("What is Go?")).
//	    Output(types.JSON(map[string]any{"answer": "Go is..."})).
//	    Create()
type JSON = any

// JSONObject is an alias for map[string]any, representing a JSON object.
// Use this for metadata and structured data fields.
//
// Example:
//
//	trace.Generation().
//	    Metadata(types.JSONObject{"model": "gpt-4", "temperature": 0.7}).
//	    Create()
type JSONObject = map[string]any

// Time is a custom time type that handles JSON marshaling/unmarshaling.
// When the time is zero, it marshals to JSON null.
// Note: The omitempty tag does NOT prevent zero times from being marshaled.
// If you need true omitempty behavior, use *Time (pointer) instead.
type Time struct {
	time.Time
}

// IsZero returns true if the time is the zero value.
// This method is used by encoding/json for omitempty checks in Go 1.18+.
func (t Time) IsZero() bool {
	return t.Time.IsZero()
}

// MarshalJSON implements json.Marshaler.
// Zero times are marshaled as JSON null.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time.Format(time.RFC3339Nano))
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *Time) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		// Try parsing as a number (Unix timestamp)
		var ts float64
		if err := json.Unmarshal(data, &ts); err != nil {
			return err
		}
		t.Time = time.Unix(int64(ts), int64((ts-float64(int64(ts)))*1e9))
		return nil
	}
	if s == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		// Try other formats
		parsed, err = time.Parse(time.RFC3339, s)
		if err != nil {
			parsed, err = time.Parse("2006-01-02T15:04:05.000Z", s)
			if err != nil {
				return err
			}
		}
	}
	t.Time = parsed
	return nil
}

// Now returns the current time as a Time.
func Now() Time {
	return Time{Time: time.Now()}
}

// TimePtr returns a pointer to a Time value.
// Use this when you need true omitempty behavior with JSON marshaling.
func TimePtr(t time.Time) *Time {
	return &Time{Time: t}
}

// TimeNow returns a pointer to the current time.
// Convenience function for TimePtr(time.Now()).
func TimeNow() *Time {
	return &Time{Time: time.Now()}
}
