package langfuse_test

import (
	"sync"
	"testing"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
)

// testMetrics is a mock metrics implementation for testing
type testMetrics struct {
	mu       sync.Mutex
	counters map[string]int64
	gauges   map[string]float64
}

func newTestMetrics() *testMetrics {
	return &testMetrics{
		counters: make(map[string]int64),
		gauges:   make(map[string]float64),
	}
}

func (m *testMetrics) IncrementCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.counters == nil {
		m.counters = make(map[string]int64)
	}
	m.counters[name] += value
}

func (m *testMetrics) RecordDuration(name string, duration time.Duration) {}

func (m *testMetrics) SetGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.gauges == nil {
		m.gauges = make(map[string]float64)
	}
	m.gauges[name] = value
}

func TestBackpressureLevel_String(t *testing.T) {
	tests := []struct {
		level langfuse.BackpressureLevel
		want  string
	}{
		{langfuse.BackpressureNone, "none"},
		{langfuse.BackpressureWarning, "warning"},
		{langfuse.BackpressureCritical, "critical"},
		{langfuse.BackpressureOverflow, "overflow"},
		{langfuse.BackpressureLevel(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultBackpressureThreshold(t *testing.T) {
	threshold := langfuse.DefaultBackpressureThreshold()

	if threshold.WarningPercent != 50.0 {
		t.Errorf("WarningPercent = %v, want 50.0", threshold.WarningPercent)
	}
	if threshold.CriticalPercent != 80.0 {
		t.Errorf("CriticalPercent = %v, want 80.0", threshold.CriticalPercent)
	}
	if threshold.OverflowPercent != 95.0 {
		t.Errorf("OverflowPercent = %v, want 95.0", threshold.OverflowPercent)
	}
}

func TestQueueMonitor_Creation(t *testing.T) {
	m := langfuse.NewQueueMonitor(nil)

	if m.Level() != langfuse.BackpressureNone {
		t.Errorf("Level() = %v, want %v", m.Level(), langfuse.BackpressureNone)
	}

	if !m.IsHealthy() {
		t.Error("IsHealthy() = false, want true")
	}

	state := m.State()
	if state.Capacity != 1000 {
		t.Errorf("State().Capacity = %d, want 1000", state.Capacity)
	}
}

func TestQueueMonitor_CustomConfig(t *testing.T) {
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 500,
		Threshold: langfuse.BackpressureThreshold{
			WarningPercent:  40.0,
			CriticalPercent: 70.0,
			OverflowPercent: 90.0,
		},
	})

	state := m.State()
	if state.Capacity != 500 {
		t.Errorf("State().Capacity = %d, want 500", state.Capacity)
	}
}

func TestQueueMonitor_Update_None(t *testing.T) {
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
	})

	level := m.Update(10) // 10%

	if level != langfuse.BackpressureNone {
		t.Errorf("Update(10) = %v, want %v", level, langfuse.BackpressureNone)
	}
	if !m.IsHealthy() {
		t.Error("IsHealthy() = false at 10%")
	}
}

func TestQueueMonitor_Update_Warning(t *testing.T) {
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
	})

	level := m.Update(55) // 55%

	if level != langfuse.BackpressureWarning {
		t.Errorf("Update(55) = %v, want %v", level, langfuse.BackpressureWarning)
	}
	if m.IsHealthy() {
		t.Error("IsHealthy() = true at 55%, want false")
	}
	if m.IsCritical() {
		t.Error("IsCritical() = true at 55%, want false")
	}
}

func TestQueueMonitor_Update_Critical(t *testing.T) {
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
	})

	level := m.Update(85) // 85%

	if level != langfuse.BackpressureCritical {
		t.Errorf("Update(85) = %v, want %v", level, langfuse.BackpressureCritical)
	}
	if m.IsHealthy() {
		t.Error("IsHealthy() = true at 85%, want false")
	}
	if !m.IsCritical() {
		t.Error("IsCritical() = false at 85%, want true")
	}
	if m.ShouldBlock() {
		t.Error("ShouldBlock() = true at 85%, want false")
	}
}

func TestQueueMonitor_Update_Overflow(t *testing.T) {
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
	})

	level := m.Update(98) // 98%

	if level != langfuse.BackpressureOverflow {
		t.Errorf("Update(98) = %v, want %v", level, langfuse.BackpressureOverflow)
	}
	if !m.ShouldBlock() {
		t.Error("ShouldBlock() = false at 98%, want true")
	}
}

func TestQueueMonitor_Callback(t *testing.T) {
	var receivedState langfuse.QueueState
	var mu sync.Mutex

	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
		OnBackpressure: func(state langfuse.QueueState) {
			mu.Lock()
			receivedState = state
			mu.Unlock()
		},
	})

	m.Update(55) // Should trigger warning

	mu.Lock()
	got := receivedState
	mu.Unlock()

	if got.Level != langfuse.BackpressureWarning {
		t.Errorf("callback received level %v, want %v", got.Level, langfuse.BackpressureWarning)
	}
}

func TestQueueMonitor_CallbackOnlyOnChange(t *testing.T) {
	var callCount int
	var mu sync.Mutex

	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
		OnBackpressure: func(state langfuse.QueueState) {
			mu.Lock()
			callCount++
			mu.Unlock()
		},
	})

	// Same level, should not trigger callback multiple times
	m.Update(55) // Warning
	m.Update(60) // Still warning
	m.Update(65) // Still warning

	mu.Lock()
	got := callCount
	mu.Unlock()

	if got != 1 {
		t.Errorf("callback called %d times, want 1", got)
	}
}

func TestQueueMonitor_SetCallback(t *testing.T) {
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
	})

	var called bool
	m.SetCallback(func(state langfuse.QueueState) {
		called = true
	})

	m.Update(55) // Trigger warning

	if !called {
		t.Error("SetCallback callback was not called")
	}
}

func TestQueueMonitor_Stats(t *testing.T) {
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
	})

	m.Update(55) // Warning
	m.Update(85) // Critical
	m.Update(98) // Overflow
	m.Update(10) // Back to none

	stats := m.Stats()

	if stats.WarningCount < 1 {
		t.Errorf("Stats().WarningCount = %d, want >= 1", stats.WarningCount)
	}
	if stats.CriticalCount < 1 {
		t.Errorf("Stats().CriticalCount = %d, want >= 1", stats.CriticalCount)
	}
	if stats.OverflowCount < 1 {
		t.Errorf("Stats().OverflowCount = %d, want >= 1", stats.OverflowCount)
	}
	if stats.StateChanges < 4 {
		t.Errorf("Stats().StateChanges = %d, want >= 4", stats.StateChanges)
	}
}

func TestQueueMonitor_State(t *testing.T) {
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
	})

	m.Update(55)
	state := m.State()

	if state.Size != 55 {
		t.Errorf("State().Size = %d, want 55", state.Size)
	}
	if state.Capacity != 100 {
		t.Errorf("State().Capacity = %d, want 100", state.Capacity)
	}
	if state.Level != langfuse.BackpressureWarning {
		t.Errorf("State().Level = %v, want %v", state.Level, langfuse.BackpressureWarning)
	}
	if state.PercentFull < 54.9 || state.PercentFull > 55.1 {
		t.Errorf("State().PercentFull = %v, want ~55.0", state.PercentFull)
	}
	if state.Timestamp.IsZero() {
		t.Error("State().Timestamp is zero")
	}
}

func TestQueueMonitor_ConcurrentAccess(t *testing.T) {
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
	})

	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 100

	for i := range goroutines {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := range iterations {
				size := (id*iterations + j) % 100
				m.Update(size)
				_ = m.Level()
				_ = m.State()
				_ = m.IsHealthy()
				_ = m.IsCritical()
				_ = m.Stats()
			}
		}(i)
	}

	wg.Wait()
}

func TestQueueMonitor_WithMetrics(t *testing.T) {
	metrics := newTestMetrics()
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
		Metrics:  metrics,
	})

	m.Update(55)

	if metrics.gauges["langfuse.queue.size"] != 55.0 {
		t.Errorf("gauge size = %v, want 55.0", metrics.gauges["langfuse.queue.size"])
	}
	percentFull := metrics.gauges["langfuse.queue.percent_full"]
	if percentFull < 54.9 || percentFull > 55.1 {
		t.Errorf("gauge percent_full = %v, want ~55.0", percentFull)
	}
}

func TestQueueMonitor_WithLogger(t *testing.T) {
	var logged bool
	var mu sync.Mutex

	logger := &bpTestLogger{
		onPrintf: func(format string, args ...any) {
			mu.Lock()
			logged = true
			mu.Unlock()
		},
	}

	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
		Logger:   logger,
	})

	m.Update(55) // Trigger warning

	mu.Lock()
	got := logged
	mu.Unlock()

	if !got {
		t.Error("Logger was not called on level change")
	}
}

// BackpressureHandler tests

func TestBackpressureHandler_Creation(t *testing.T) {
	h := langfuse.NewBackpressureHandler(nil)

	if h.Monitor() == nil {
		t.Error("Monitor() returned nil")
	}
}

func TestBackpressureHandler_Decide_Allow(t *testing.T) {
	h := langfuse.NewBackpressureHandler(&langfuse.BackpressureHandlerConfig{
		Monitor: langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
			Capacity: 100,
		}),
	})

	decision := h.Decide(10)

	if decision != langfuse.DecisionAllow {
		t.Errorf("Decide(10) = %v, want %v", decision, langfuse.DecisionAllow)
	}
}

func TestBackpressureHandler_Decide_Block(t *testing.T) {
	h := langfuse.NewBackpressureHandler(&langfuse.BackpressureHandlerConfig{
		Monitor: langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
			Capacity: 100,
		}),
		BlockOnFull: true,
	})

	decision := h.Decide(98) // Overflow

	if decision != langfuse.DecisionBlock {
		t.Errorf("Decide(98) with BlockOnFull = %v, want %v", decision, langfuse.DecisionBlock)
	}

	stats := h.Stats()
	if stats.BlockedCount != 1 {
		t.Errorf("Stats().BlockedCount = %d, want 1", stats.BlockedCount)
	}
}

func TestBackpressureHandler_Decide_Drop(t *testing.T) {
	h := langfuse.NewBackpressureHandler(&langfuse.BackpressureHandlerConfig{
		Monitor: langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
			Capacity: 100,
		}),
		DropOnFull: true,
	})

	decision := h.Decide(98) // Overflow

	if decision != langfuse.DecisionDrop {
		t.Errorf("Decide(98) with DropOnFull = %v, want %v", decision, langfuse.DecisionDrop)
	}

	stats := h.Stats()
	if stats.DroppedCount != 1 {
		t.Errorf("Stats().DroppedCount = %d, want 1", stats.DroppedCount)
	}
}

func TestBackpressureHandler_Stats(t *testing.T) {
	h := langfuse.NewBackpressureHandler(&langfuse.BackpressureHandlerConfig{
		Monitor: langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
			Capacity: 100,
		}),
		BlockOnFull: true,
		DropOnFull:  false,
	})

	stats := h.Stats()

	if !stats.BlockOnFull {
		t.Error("Stats().BlockOnFull = false, want true")
	}
	if stats.DropOnFull {
		t.Error("Stats().DropOnFull = true, want false")
	}
}

func TestBackpressureDecision_String(t *testing.T) {
	tests := []struct {
		decision langfuse.BackpressureDecision
		want     string
	}{
		{langfuse.DecisionAllow, "allow"},
		{langfuse.DecisionBlock, "block"},
		{langfuse.DecisionDrop, "drop"},
		{langfuse.BackpressureDecision(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.decision.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBackpressureHandler_WithMetrics(t *testing.T) {
	metrics := newTestMetrics()
	h := langfuse.NewBackpressureHandler(&langfuse.BackpressureHandlerConfig{
		Monitor: langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
			Capacity: 100,
		}),
		BlockOnFull: true,
		Metrics:     metrics,
	})

	h.Decide(98) // Overflow -> block

	if metrics.counters["langfuse.backpressure.blocked"] != 1 {
		t.Error("blocked counter not incremented")
	}
}

func TestBackpressureHandler_WithLogger(t *testing.T) {
	var logged bool
	var mu sync.Mutex

	logger := &bpTestLogger{
		onPrintf: func(format string, args ...any) {
			mu.Lock()
			logged = true
			mu.Unlock()
		},
	}

	h := langfuse.NewBackpressureHandler(&langfuse.BackpressureHandlerConfig{
		Monitor: langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
			Capacity: 100,
		}),
		DropOnFull: true,
		Logger:     logger,
	})

	h.Decide(98) // Overflow -> drop

	mu.Lock()
	got := logged
	mu.Unlock()

	if !got {
		t.Error("Logger was not called on drop")
	}
}

// bpTestLogger is a simple mock logger for backpressure testing
type bpTestLogger struct {
	onPrintf func(format string, args ...any)
}

func (l *bpTestLogger) Printf(format string, args ...any) {
	if l.onPrintf != nil {
		l.onPrintf(format, args...)
	}
}

func BenchmarkQueueMonitor_Update(b *testing.B) {
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 1000,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Update(i % 1000)
	}
}

func BenchmarkQueueMonitor_UpdateConcurrent(b *testing.B) {
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 1000,
	})

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			m.Update(i % 1000)
			i++
		}
	})
}

func BenchmarkBackpressureHandler_Decide(b *testing.B) {
	h := langfuse.NewBackpressureHandler(&langfuse.BackpressureHandlerConfig{
		Monitor: langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
			Capacity: 1000,
		}),
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		h.Decide(i % 1000)
	}
}

// Edge cases

func TestQueueMonitor_ZeroCapacity(t *testing.T) {
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 0, // Should use default
	})

	state := m.State()
	if state.Capacity != 1000 {
		t.Errorf("State().Capacity = %d with 0 config, want 1000", state.Capacity)
	}
}

func TestQueueMonitor_NegativeThresholds(t *testing.T) {
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
		Threshold: langfuse.BackpressureThreshold{
			WarningPercent:  -10, // Should use default
			CriticalPercent: -20,
			OverflowPercent: -30,
		},
	})

	// Should use defaults
	m.Update(55) // 55% should be warning with default 50%
	if m.Level() != langfuse.BackpressureWarning {
		t.Errorf("Level() = %v at 55%% with default threshold, want %v", m.Level(), langfuse.BackpressureWarning)
	}
}

func TestQueueMonitor_ExactThresholdBoundaries(t *testing.T) {
	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
		Threshold: langfuse.BackpressureThreshold{
			WarningPercent:  50.0,
			CriticalPercent: 80.0,
			OverflowPercent: 95.0,
		},
	})

	// Exactly at warning threshold
	level := m.Update(50)
	if level != langfuse.BackpressureWarning {
		t.Errorf("Update(50) at exact warning threshold = %v, want %v", level, langfuse.BackpressureWarning)
	}

	// Just below warning threshold
	level = m.Update(49)
	if level != langfuse.BackpressureNone {
		t.Errorf("Update(49) below warning threshold = %v, want %v", level, langfuse.BackpressureNone)
	}
}

func TestQueueMonitor_RecoveringFromOverflow(t *testing.T) {
	var levels []langfuse.BackpressureLevel
	var mu sync.Mutex

	m := langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
		Capacity: 100,
		OnBackpressure: func(state langfuse.QueueState) {
			mu.Lock()
			levels = append(levels, state.Level)
			mu.Unlock()
		},
	})

	// Go to overflow
	m.Update(98)
	// Recover to critical
	m.Update(82)
	// Recover to warning
	m.Update(55)
	// Recover to none
	m.Update(10)

	mu.Lock()
	defer mu.Unlock()

	if len(levels) != 4 {
		t.Errorf("len(levels) = %d, want 4", len(levels))
	}

	expected := []langfuse.BackpressureLevel{
		langfuse.BackpressureOverflow,
		langfuse.BackpressureCritical,
		langfuse.BackpressureWarning,
		langfuse.BackpressureNone,
	}

	for i, want := range expected {
		if i < len(levels) && levels[i] != want {
			t.Errorf("levels[%d] = %v, want %v", i, levels[i], want)
		}
	}
}
