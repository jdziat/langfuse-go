package langfuse

import (
	"sync"
	"testing"
)

func TestBackpressureLevel_String(t *testing.T) {
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
			if got := tt.level.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDefaultBackpressureThreshold(t *testing.T) {
	threshold := DefaultBackpressureThreshold()

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
	m := NewQueueMonitor(nil)

	if m.Level() != BackpressureNone {
		t.Errorf("Level() = %v, want %v", m.Level(), BackpressureNone)
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
	m := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 500,
		Threshold: BackpressureThreshold{
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
	m := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 100,
	})

	level := m.Update(10) // 10%

	if level != BackpressureNone {
		t.Errorf("Update(10) = %v, want %v", level, BackpressureNone)
	}
	if !m.IsHealthy() {
		t.Error("IsHealthy() = false at 10%")
	}
}

func TestQueueMonitor_Update_Warning(t *testing.T) {
	m := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 100,
	})

	level := m.Update(55) // 55%

	if level != BackpressureWarning {
		t.Errorf("Update(55) = %v, want %v", level, BackpressureWarning)
	}
	if m.IsHealthy() {
		t.Error("IsHealthy() = true at 55%, want false")
	}
	if m.IsCritical() {
		t.Error("IsCritical() = true at 55%, want false")
	}
}

func TestQueueMonitor_Update_Critical(t *testing.T) {
	m := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 100,
	})

	level := m.Update(85) // 85%

	if level != BackpressureCritical {
		t.Errorf("Update(85) = %v, want %v", level, BackpressureCritical)
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
	m := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 100,
	})

	level := m.Update(98) // 98%

	if level != BackpressureOverflow {
		t.Errorf("Update(98) = %v, want %v", level, BackpressureOverflow)
	}
	if !m.ShouldBlock() {
		t.Error("ShouldBlock() = false at 98%, want true")
	}
}

func TestQueueMonitor_Callback(t *testing.T) {
	var receivedState QueueState
	var mu sync.Mutex

	m := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 100,
		OnBackpressure: func(state QueueState) {
			mu.Lock()
			receivedState = state
			mu.Unlock()
		},
	})

	m.Update(55) // Should trigger warning

	mu.Lock()
	got := receivedState
	mu.Unlock()

	if got.Level != BackpressureWarning {
		t.Errorf("callback received level %v, want %v", got.Level, BackpressureWarning)
	}
}

func TestQueueMonitor_CallbackOnlyOnChange(t *testing.T) {
	var callCount int
	var mu sync.Mutex

	m := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 100,
		OnBackpressure: func(state QueueState) {
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
	m := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 100,
	})

	var called bool
	m.SetCallback(func(state QueueState) {
		called = true
	})

	m.Update(55) // Trigger warning

	if !called {
		t.Error("SetCallback callback was not called")
	}
}

func TestQueueMonitor_Stats(t *testing.T) {
	m := NewQueueMonitor(&QueueMonitorConfig{
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
	m := NewQueueMonitor(&QueueMonitorConfig{
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
	if state.Level != BackpressureWarning {
		t.Errorf("State().Level = %v, want %v", state.Level, BackpressureWarning)
	}
	if state.PercentFull < 54.9 || state.PercentFull > 55.1 {
		t.Errorf("State().PercentFull = %v, want ~55.0", state.PercentFull)
	}
	if state.Timestamp.IsZero() {
		t.Error("State().Timestamp is zero")
	}
}

func TestQueueMonitor_ConcurrentAccess(t *testing.T) {
	m := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 100,
	})

	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
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
	metrics := &testMetrics{}
	m := NewQueueMonitor(&QueueMonitorConfig{
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

	m := NewQueueMonitor(&QueueMonitorConfig{
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
	h := NewBackpressureHandler(nil)

	if h.Monitor() == nil {
		t.Error("Monitor() returned nil")
	}
}

func TestBackpressureHandler_Decide_Allow(t *testing.T) {
	h := NewBackpressureHandler(&BackpressureHandlerConfig{
		Monitor: NewQueueMonitor(&QueueMonitorConfig{
			Capacity: 100,
		}),
	})

	decision := h.Decide(10)

	if decision != DecisionAllow {
		t.Errorf("Decide(10) = %v, want %v", decision, DecisionAllow)
	}
}

func TestBackpressureHandler_Decide_Block(t *testing.T) {
	h := NewBackpressureHandler(&BackpressureHandlerConfig{
		Monitor: NewQueueMonitor(&QueueMonitorConfig{
			Capacity: 100,
		}),
		BlockOnFull: true,
	})

	decision := h.Decide(98) // Overflow

	if decision != DecisionBlock {
		t.Errorf("Decide(98) with BlockOnFull = %v, want %v", decision, DecisionBlock)
	}

	stats := h.Stats()
	if stats.BlockedCount != 1 {
		t.Errorf("Stats().BlockedCount = %d, want 1", stats.BlockedCount)
	}
}

func TestBackpressureHandler_Decide_Drop(t *testing.T) {
	h := NewBackpressureHandler(&BackpressureHandlerConfig{
		Monitor: NewQueueMonitor(&QueueMonitorConfig{
			Capacity: 100,
		}),
		DropOnFull: true,
	})

	decision := h.Decide(98) // Overflow

	if decision != DecisionDrop {
		t.Errorf("Decide(98) with DropOnFull = %v, want %v", decision, DecisionDrop)
	}

	stats := h.Stats()
	if stats.DroppedCount != 1 {
		t.Errorf("Stats().DroppedCount = %d, want 1", stats.DroppedCount)
	}
}

func TestBackpressureHandler_Stats(t *testing.T) {
	h := NewBackpressureHandler(&BackpressureHandlerConfig{
		Monitor: NewQueueMonitor(&QueueMonitorConfig{
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
			if got := tt.decision.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBackpressureHandler_WithMetrics(t *testing.T) {
	metrics := &testMetrics{}
	h := NewBackpressureHandler(&BackpressureHandlerConfig{
		Monitor: NewQueueMonitor(&QueueMonitorConfig{
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

	h := NewBackpressureHandler(&BackpressureHandlerConfig{
		Monitor: NewQueueMonitor(&QueueMonitorConfig{
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
	m := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 1000,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Update(i % 1000)
	}
}

func BenchmarkQueueMonitor_UpdateConcurrent(b *testing.B) {
	m := NewQueueMonitor(&QueueMonitorConfig{
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
	h := NewBackpressureHandler(&BackpressureHandlerConfig{
		Monitor: NewQueueMonitor(&QueueMonitorConfig{
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
	m := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 0, // Should use default
	})

	state := m.State()
	if state.Capacity != 1000 {
		t.Errorf("State().Capacity = %d with 0 config, want 1000", state.Capacity)
	}
}

func TestQueueMonitor_NegativeThresholds(t *testing.T) {
	m := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 100,
		Threshold: BackpressureThreshold{
			WarningPercent:  -10, // Should use default
			CriticalPercent: -20,
			OverflowPercent: -30,
		},
	})

	// Should use defaults
	m.Update(55) // 55% should be warning with default 50%
	if m.Level() != BackpressureWarning {
		t.Errorf("Level() = %v at 55%% with default threshold, want %v", m.Level(), BackpressureWarning)
	}
}

func TestQueueMonitor_ExactThresholdBoundaries(t *testing.T) {
	m := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 100,
		Threshold: BackpressureThreshold{
			WarningPercent:  50.0,
			CriticalPercent: 80.0,
			OverflowPercent: 95.0,
		},
	})

	// Exactly at warning threshold
	level := m.Update(50)
	if level != BackpressureWarning {
		t.Errorf("Update(50) at exact warning threshold = %v, want %v", level, BackpressureWarning)
	}

	// Just below warning threshold
	level = m.Update(49)
	if level != BackpressureNone {
		t.Errorf("Update(49) below warning threshold = %v, want %v", level, BackpressureNone)
	}
}

func TestQueueMonitor_RecoveringFromOverflow(t *testing.T) {
	var levels []BackpressureLevel
	var mu sync.Mutex

	m := NewQueueMonitor(&QueueMonitorConfig{
		Capacity: 100,
		OnBackpressure: func(state QueueState) {
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

	expected := []BackpressureLevel{
		BackpressureOverflow,
		BackpressureCritical,
		BackpressureWarning,
		BackpressureNone,
	}

	for i, want := range expected {
		if i < len(levels) && levels[i] != want {
			t.Errorf("levels[%d] = %v, want %v", i, levels[i], want)
		}
	}
}
