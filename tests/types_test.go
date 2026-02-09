package langfuse_test

import (
	"encoding/json"
	"testing"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
)

func TestTimeMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		time     langfuse.Time
		expected string
	}{
		{
			name:     "zero time",
			time:     langfuse.Time{},
			expected: "null",
		},
		{
			name:     "valid time",
			time:     langfuse.Time{Time: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)},
			expected: `"2024-01-15T10:30:00Z"`,
		},
		{
			name:     "time with nanoseconds",
			time:     langfuse.Time{Time: time.Date(2024, 1, 15, 10, 30, 0, 123456789, time.UTC)},
			expected: `"2024-01-15T10:30:00.123456789Z"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.time)
			if err != nil {
				t.Fatalf("Marshal failed: %v", err)
			}
			if string(data) != tt.expected {
				t.Errorf("Marshal = %s, want %s", string(data), tt.expected)
			}
		})
	}
}

func TestTimeUnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected time.Time
		wantErr  bool
	}{
		{
			name:     "RFC3339Nano format",
			input:    `"2024-01-15T10:30:00.123456789Z"`,
			expected: time.Date(2024, 1, 15, 10, 30, 0, 123456789, time.UTC),
			wantErr:  false,
		},
		{
			name:     "RFC3339 format",
			input:    `"2024-01-15T10:30:00Z"`,
			expected: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
			wantErr:  false,
		},
		{
			name:     "milliseconds format",
			input:    `"2024-01-15T10:30:00.123Z"`,
			expected: time.Date(2024, 1, 15, 10, 30, 0, 123000000, time.UTC),
			wantErr:  false,
		},
		{
			name:     "empty string",
			input:    `""`,
			expected: time.Time{},
			wantErr:  false,
		},
		{
			name:     "unix timestamp",
			input:    `1705315800`,
			expected: time.Unix(1705315800, 0),
			wantErr:  false,
		},
		{
			name:     "unix timestamp with decimals",
			input:    `1705315800.5`,
			expected: time.Unix(1705315800, 500000000),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result langfuse.Time
			err := json.Unmarshal([]byte(tt.input), &result)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Unmarshal error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !result.Time.Equal(tt.expected) {
				t.Errorf("Unmarshal = %v, want %v", result.Time, tt.expected)
			}
		})
	}
}

func TestTimeRoundTrip(t *testing.T) {
	original := langfuse.Time{Time: time.Date(2024, 1, 15, 10, 30, 0, 123456789, time.UTC)}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result langfuse.Time
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if !result.Time.Equal(original.Time) {
		t.Errorf("Round trip failed: got %v, want %v", result.Time, original.Time)
	}
}

func TestNow(t *testing.T) {
	before := time.Now()
	result := langfuse.Now()
	after := time.Now()

	if result.Time.Before(before) || result.Time.After(after) {
		t.Errorf("Now() returned time outside expected range")
	}
}
