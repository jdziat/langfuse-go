package langfuse

import (
	"encoding/json"
	"testing"
	"time"
)

func TestTimeMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		time     Time
		expected string
	}{
		{
			name:     "zero time",
			time:     Time{},
			expected: "null",
		},
		{
			name:     "valid time",
			time:     Time{Time: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)},
			expected: `"2024-01-15T10:30:00Z"`,
		},
		{
			name:     "time with nanoseconds",
			time:     Time{Time: time.Date(2024, 1, 15, 10, 30, 0, 123456789, time.UTC)},
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
			var result Time
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
	original := Time{Time: time.Date(2024, 1, 15, 10, 30, 0, 123456789, time.UTC)}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result Time
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
	result := Now()
	after := time.Now()

	if result.Time.Before(before) || result.Time.After(after) {
		t.Errorf("Now() returned time outside expected range")
	}
}

func TestObservationTypeConstants(t *testing.T) {
	tests := []struct {
		obsType  ObservationType
		expected string
	}{
		{ObservationTypeSpan, "SPAN"},
		{ObservationTypeGeneration, "GENERATION"},
		{ObservationTypeEvent, "EVENT"},
	}

	for _, tt := range tests {
		t.Run(string(tt.obsType), func(t *testing.T) {
			if string(tt.obsType) != tt.expected {
				t.Errorf("ObservationType = %v, want %v", tt.obsType, tt.expected)
			}
		})
	}
}

func TestObservationLevelConstants(t *testing.T) {
	tests := []struct {
		level    ObservationLevel
		expected string
	}{
		{ObservationLevelDebug, "DEBUG"},
		{ObservationLevelDefault, "DEFAULT"},
		{ObservationLevelWarning, "WARNING"},
		{ObservationLevelError, "ERROR"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			if string(tt.level) != tt.expected {
				t.Errorf("ObservationLevel = %v, want %v", tt.level, tt.expected)
			}
		})
	}
}

func TestScoreDataTypeConstants(t *testing.T) {
	tests := []struct {
		dataType ScoreDataType
		expected string
	}{
		{ScoreDataTypeNumeric, "NUMERIC"},
		{ScoreDataTypeCategorical, "CATEGORICAL"},
		{ScoreDataTypeBoolean, "BOOLEAN"},
	}

	for _, tt := range tests {
		t.Run(string(tt.dataType), func(t *testing.T) {
			if string(tt.dataType) != tt.expected {
				t.Errorf("ScoreDataType = %v, want %v", tt.dataType, tt.expected)
			}
		})
	}
}

func TestScoreSourceConstants(t *testing.T) {
	tests := []struct {
		source   ScoreSource
		expected string
	}{
		{ScoreSourceAPI, "API"},
		{ScoreSourceAnnotation, "ANNOTATION"},
		{ScoreSourceEval, "EVAL"},
	}

	for _, tt := range tests {
		t.Run(string(tt.source), func(t *testing.T) {
			if string(tt.source) != tt.expected {
				t.Errorf("ScoreSource = %v, want %v", tt.source, tt.expected)
			}
		})
	}
}

func TestTraceJSONSerialization(t *testing.T) {
	trace := Trace{
		ID:        "trace-123",
		Name:      "test-trace",
		UserID:    "user-456",
		SessionID: "session-789",
		Input:     map[string]interface{}{"query": "hello"},
		Output:    map[string]interface{}{"response": "world"},
		Tags:      []string{"test", "unit"},
		Metadata:  map[string]interface{}{"key": "value"},
		Public:    true,
	}

	data, err := json.Marshal(trace)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result Trace
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.ID != trace.ID {
		t.Errorf("ID = %v, want %v", result.ID, trace.ID)
	}
	if result.Name != trace.Name {
		t.Errorf("Name = %v, want %v", result.Name, trace.Name)
	}
	if result.UserID != trace.UserID {
		t.Errorf("UserID = %v, want %v", result.UserID, trace.UserID)
	}
	if result.SessionID != trace.SessionID {
		t.Errorf("SessionID = %v, want %v", result.SessionID, trace.SessionID)
	}
	if result.Public != trace.Public {
		t.Errorf("Public = %v, want %v", result.Public, trace.Public)
	}
	if len(result.Tags) != len(trace.Tags) {
		t.Errorf("Tags length = %v, want %v", len(result.Tags), len(trace.Tags))
	}
}

func TestUsageJSONSerialization(t *testing.T) {
	usage := Usage{
		Input:      100,
		Output:     50,
		Total:      150,
		Unit:       "TOKENS",
		InputCost:  0.001,
		OutputCost: 0.002,
		TotalCost:  0.003,
	}

	data, err := json.Marshal(usage)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	var result Usage
	err = json.Unmarshal(data, &result)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if result.Input != usage.Input {
		t.Errorf("Input = %v, want %v", result.Input, usage.Input)
	}
	if result.Output != usage.Output {
		t.Errorf("Output = %v, want %v", result.Output, usage.Output)
	}
	if result.Total != usage.Total {
		t.Errorf("Total = %v, want %v", result.Total, usage.Total)
	}
	if result.TotalCost != usage.TotalCost {
		t.Errorf("TotalCost = %v, want %v", result.TotalCost, usage.TotalCost)
	}
}
