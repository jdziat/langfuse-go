package ingestion

import (
	"strings"
	"sync"
	"testing"
	"time"
)

// TestUUID tests UUID generation.
func TestUUID(t *testing.T) {
	// Generate multiple UUIDs
	uuids := make(map[string]bool)
	for range 100 {
		uuid, err := UUID()
		if err != nil {
			t.Fatalf("UUID() error: %v", err)
		}

		// Check format (36 characters with hyphens at positions 8, 13, 18, 23)
		if len(uuid) != 36 {
			t.Errorf("UUID length = %d, want 36", len(uuid))
		}

		if uuid[8] != '-' || uuid[13] != '-' || uuid[18] != '-' || uuid[23] != '-' {
			t.Errorf("UUID format invalid: %s", uuid)
		}

		// Check uniqueness
		if uuids[uuid] {
			t.Errorf("Duplicate UUID generated: %s", uuid)
		}
		uuids[uuid] = true

		// Validate using IsValidUUID
		if !IsValidUUID(uuid) {
			t.Errorf("Generated UUID failed validation: %s", uuid)
		}
	}
}

// TestGenerateID tests ID generation with fallback.
func TestGenerateID(t *testing.T) {
	// Generate multiple IDs
	ids := make(map[string]bool)
	for range 100 {
		id := GenerateID()
		if id == "" {
			t.Error("GenerateID() returned empty string")
		}

		// Check uniqueness
		if ids[id] {
			t.Errorf("Duplicate ID generated: %s", id)
		}
		ids[id] = true

		// Should be valid UUID format (most of the time)
		// or timestamp format (fallback)
		if len(id) != 36 && !strings.Contains(id, "-") {
			t.Errorf("ID has unexpected format: %s", id)
		}
	}
}

// TestIsValidUUID tests UUID validation.
func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{
			name:  "valid standard UUID",
			input: "550e8400-e29b-41d4-a716-446655440000",
			want:  true,
		},
		{
			name:  "valid compact UUID",
			input: "550e8400e29b41d4a716446655440000",
			want:  true,
		},
		{
			name:  "valid uppercase UUID",
			input: "550E8400-E29B-41D4-A716-446655440000",
			want:  true,
		},
		{
			name:  "invalid - too short",
			input: "550e8400-e29b-41d4-a716",
			want:  false,
		},
		{
			name:  "invalid - too long",
			input: "550e8400-e29b-41d4-a716-446655440000-extra",
			want:  false,
		},
		{
			name:  "invalid - wrong hyphen positions",
			input: "550e8400e-29b-41d4-a716-446655440000",
			want:  false,
		},
		{
			name:  "invalid - non-hex characters",
			input: "550e8400-e29b-41d4-a716-44665544000g",
			want:  false,
		},
		{
			name:  "invalid - empty string",
			input: "",
			want:  false,
		},
		{
			name:  "invalid - spaces",
			input: "550e8400 e29b 41d4 a716 446655440000",
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsValidUUID(tt.input)
			if got != tt.want {
				t.Errorf("IsValidUUID(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestEventTypeConstants tests that event type constants are exported and correct.
func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  string
	}{
		{"TraceCreate", EventTypeTraceCreate, "trace-create"},
		{"TraceUpdate", EventTypeTraceUpdate, "trace-update"},
		{"SpanCreate", EventTypeSpanCreate, "span-create"},
		{"SpanUpdate", EventTypeSpanUpdate, "span-update"},
		{"GenerationCreate", EventTypeGenerationCreate, "generation-create"},
		{"GenerationUpdate", EventTypeGenerationUpdate, "generation-update"},
		{"EventCreate", EventTypeEventCreate, "event-create"},
		{"ScoreCreate", EventTypeScoreCreate, "score-create"},
		{"SDKLog", EventTypeSDKLog, "sdk-log"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.value != tt.want {
				t.Errorf("%s = %q, want %q", tt.name, tt.value, tt.want)
			}
		})
	}
}

// TestBackpressureLevel tests backpressure level string conversion.
func TestBackpressureLevel(t *testing.T) {
	tests := []struct {
		level BackpressureLevel
		want  string
	}{
		{BackpressureNone, "none"},
		{BackpressureWarning, "warning"},
		{BackpressureCritical, "critical"},
		{BackpressureOverflow, "overflow"},
		{BackpressureLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.level.String()
			if got != tt.want {
				t.Errorf("BackpressureLevel.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestBackpressureThreshold tests default threshold values.
func TestBackpressureThreshold(t *testing.T) {
	threshold := DefaultBackpressureThreshold()

	if threshold.WarningPercent != 50.0 {
		t.Errorf("WarningPercent = %.1f, want 50.0", threshold.WarningPercent)
	}
	if threshold.CriticalPercent != 80.0 {
		t.Errorf("CriticalPercent = %.1f, want 80.0", threshold.CriticalPercent)
	}
	if threshold.OverflowPercent != 95.0 {
		t.Errorf("OverflowPercent = %.1f, want 95.0", threshold.OverflowPercent)
	}
}

// mockLogger implements the Logger interface for testing.
type mockLogger struct {
	mu       sync.Mutex
	messages []string
}

func (m *mockLogger) Printf(format string, v ...any) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// Don't need to format for testing, just track that it was called
	m.messages = append(m.messages, format)
}

func (m *mockLogger) MessageCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

// mockMetrics implements the Metrics interface for testing.
type mockMetrics struct {
	mu       sync.Mutex
	counters map[string]int64
	gauges   map[string]float64
}

func newMockMetrics() *mockMetrics {
	return &mockMetrics{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

func (m *mockMetrics) IncrementCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

func (m *mockMetrics) SetGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *mockMetrics) GetCounter(name string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[name]
}

func (m *mockMetrics) GetGauge(name string) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.gauges[name]
}

// TestQueueMonitor tests queue monitoring and level transitions.
func TestQueueMonitor(t *testing.T) {
	logger := &mockLogger{}
	metrics := newMockMetrics()

	cfg := &QueueMonitorConfig{
		Capacity: 100,
		Threshold: BackpressureThreshold{
			WarningPercent:  50.0,
			CriticalPercent: 80.0,
			OverflowPercent: 95.0,
		},
		Logger:  logger,
		Metrics: metrics,
	}

	monitor := NewQueueMonitor(cfg)

	// Test initial state
	if !monitor.IsHealthy() {
		t.Error("Monitor should start healthy")
	}
	if monitor.IsCritical() {
		t.Error("Monitor should not start critical")
	}

	// Test level transitions
	tests := []struct {
		size      int
		wantLevel BackpressureLevel
	}{
		{0, BackpressureNone},
		{25, BackpressureNone},
		{50, BackpressureWarning},
		{75, BackpressureWarning},
		{80, BackpressureCritical},
		{90, BackpressureCritical},
		{95, BackpressureOverflow},
		{100, BackpressureOverflow},
		{50, BackpressureWarning},
		{10, BackpressureNone},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			level := monitor.Update(tt.size)
			if level != tt.wantLevel {
				t.Errorf("Update(%d) = %v, want %v", tt.size, level, tt.wantLevel)
			}

			state := monitor.State()
			if state.Size != tt.size {
				t.Errorf("State.Size = %d, want %d", state.Size, tt.size)
			}
			if state.Level != tt.wantLevel {
				t.Errorf("State.Level = %v, want %v", state.Level, tt.wantLevel)
			}
		})
	}

	// Verify metrics were updated
	if metrics.GetGauge("langfuse.queue.size") != 10.0 {
		t.Errorf("Final queue size metric = %.1f, want 10.0", metrics.GetGauge("langfuse.queue.size"))
	}

	// Verify statistics
	stats := monitor.Stats()
	if stats.WarningCount == 0 {
		t.Error("Expected some warning count")
	}
	if stats.CriticalCount == 0 {
		t.Error("Expected some critical count")
	}
	if stats.OverflowCount == 0 {
		t.Error("Expected some overflow count")
	}
	if stats.StateChanges == 0 {
		t.Error("Expected some state changes")
	}
}

// TestQueueMonitorCallback tests backpressure callbacks.
func TestQueueMonitorCallback(t *testing.T) {
	var callbackCount int
	var lastState QueueState
	var mu sync.Mutex

	callback := func(state QueueState) {
		mu.Lock()
		defer mu.Unlock()
		callbackCount++
		lastState = state
	}

	cfg := &QueueMonitorConfig{
		Capacity:       100,
		OnBackpressure: callback,
	}

	monitor := NewQueueMonitor(cfg)

	// Trigger level changes
	monitor.Update(50)  // none -> warning
	monitor.Update(80)  // warning -> critical
	monitor.Update(95)  // critical -> overflow
	monitor.Update(100) // overflow -> overflow (no change)
	monitor.Update(10)  // overflow -> none

	// Allow time for callbacks
	time.Sleep(10 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	// Should have 4 state changes (last update doesn't change level)
	if callbackCount != 4 {
		t.Errorf("Callback count = %d, want 4", callbackCount)
	}

	// Last callback should be for level None
	if lastState.Level != BackpressureNone {
		t.Errorf("Last callback level = %v, want %v", lastState.Level, BackpressureNone)
	}
}

// TestBackpressureDecision tests decision string conversion.
func TestBackpressureDecision(t *testing.T) {
	tests := []struct {
		decision BackpressureDecision
		want     string
	}{
		{DecisionAllow, "allow"},
		{DecisionBlock, "block"},
		{DecisionDrop, "drop"},
		{BackpressureDecision(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := tt.decision.String()
			if got != tt.want {
				t.Errorf("BackpressureDecision.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestBackpressureHandler tests backpressure decision making.
func TestBackpressureHandler(t *testing.T) {
	t.Run("block on full", func(t *testing.T) {
		monitor := NewQueueMonitor(&QueueMonitorConfig{
			Capacity: 100,
		})

		handler := NewBackpressureHandler(&BackpressureHandlerConfig{
			Monitor:     monitor,
			BlockOnFull: true,
		})

		// Normal level - allow
		decision := handler.Decide(10)
		if decision != DecisionAllow {
			t.Errorf("Decide(10) = %v, want %v", decision, DecisionAllow)
		}

		// Overflow level - block
		decision = handler.Decide(95)
		if decision != DecisionBlock {
			t.Errorf("Decide(95) = %v, want %v", decision, DecisionBlock)
		}

		stats := handler.Stats()
		if stats.BlockedCount != 1 {
			t.Errorf("BlockedCount = %d, want 1", stats.BlockedCount)
		}
	})

	t.Run("drop on full", func(t *testing.T) {
		logger := &mockLogger{}
		metrics := newMockMetrics()

		monitor := NewQueueMonitor(&QueueMonitorConfig{
			Capacity: 100,
		})

		handler := NewBackpressureHandler(&BackpressureHandlerConfig{
			Monitor:    monitor,
			DropOnFull: true,
			Logger:     logger,
			Metrics:    metrics,
		})

		// Overflow level - drop
		decision := handler.Decide(95)
		if decision != DecisionDrop {
			t.Errorf("Decide(95) = %v, want %v", decision, DecisionDrop)
		}

		stats := handler.Stats()
		if stats.DroppedCount != 1 {
			t.Errorf("DroppedCount = %d, want 1", stats.DroppedCount)
		}

		if logger.MessageCount() == 0 {
			t.Error("Expected logger to be called")
		}

		if metrics.GetCounter("langfuse.backpressure.dropped") != 1 {
			t.Error("Expected drop counter to be incremented")
		}
	})

	t.Run("allow on full (no block/drop)", func(t *testing.T) {
		monitor := NewQueueMonitor(&QueueMonitorConfig{
			Capacity: 100,
		})

		handler := NewBackpressureHandler(&BackpressureHandlerConfig{
			Monitor:     monitor,
			BlockOnFull: false,
			DropOnFull:  false,
		})

		// Overflow level - still allow
		decision := handler.Decide(95)
		if decision != DecisionAllow {
			t.Errorf("Decide(95) = %v, want %v", decision, DecisionAllow)
		}
	})
}

// TestQueueMonitorConcurrency tests concurrent updates to queue monitor.
func TestQueueMonitorConcurrency(t *testing.T) {
	monitor := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 100,
	})

	var wg sync.WaitGroup
	iterations := 1000

	// Concurrent updates
	for i := range 10 {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			for j := range iterations {
				size := (j + offset) % 101
				monitor.Update(size)
			}
		}(i * 10)
	}

	wg.Wait()

	// Should not panic and should have valid stats
	stats := monitor.Stats()
	if stats.StateChanges < 0 {
		t.Error("StateChanges should not be negative")
	}
}
