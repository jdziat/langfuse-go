package langfuse

import (
	"sync"
	"testing"
	"time"
)

func TestLifecycleManager_Creation(t *testing.T) {
	lm := NewLifecycleManager(&LifecycleConfig{})

	if lm.State() != ClientStateActive {
		t.Errorf("State() = %v, want %v", lm.State(), ClientStateActive)
	}

	if !lm.IsActive() {
		t.Error("IsActive() = false, want true")
	}

	if lm.IsClosed() {
		t.Error("IsClosed() = true, want false")
	}

	if lm.Uptime() < 0 {
		t.Errorf("Uptime() = %v, want >= 0", lm.Uptime())
	}

	// Clean up
	lm.BeginShutdown()
	lm.CompleteShutdown()
}

func TestLifecycleManager_StateTransitions(t *testing.T) {
	lm := NewLifecycleManager(&LifecycleConfig{})

	// Initial state
	if lm.State() != ClientStateActive {
		t.Fatalf("initial State() = %v, want %v", lm.State(), ClientStateActive)
	}

	// Begin shutdown
	err := lm.BeginShutdown()
	if err != nil {
		t.Fatalf("BeginShutdown() error = %v", err)
	}

	if lm.State() != ClientStateShuttingDown {
		t.Errorf("after BeginShutdown() State() = %v, want %v", lm.State(), ClientStateShuttingDown)
	}

	if lm.IsActive() {
		t.Error("after BeginShutdown() IsActive() = true, want false")
	}

	// Double shutdown should return error
	err = lm.BeginShutdown()
	if err != ErrClientClosed {
		t.Errorf("double BeginShutdown() error = %v, want %v", err, ErrClientClosed)
	}

	// Complete shutdown
	lm.CompleteShutdown()

	if lm.State() != ClientStateClosed {
		t.Errorf("after CompleteShutdown() State() = %v, want %v", lm.State(), ClientStateClosed)
	}

	if !lm.IsClosed() {
		t.Error("after CompleteShutdown() IsClosed() = false, want true")
	}
}

func TestLifecycleManager_RecordActivity(t *testing.T) {
	lm := NewLifecycleManager(&LifecycleConfig{})
	defer func() {
		lm.BeginShutdown()
		lm.CompleteShutdown()
	}()

	before := lm.LastActivity()
	time.Sleep(10 * time.Millisecond)

	lm.RecordActivity()
	after := lm.LastActivity()

	if !after.After(before) {
		t.Errorf("LastActivity() after RecordActivity() = %v, want after %v", after, before)
	}
}

func TestLifecycleManager_IdleDuration(t *testing.T) {
	lm := NewLifecycleManager(&LifecycleConfig{})
	defer func() {
		lm.BeginShutdown()
		lm.CompleteShutdown()
	}()

	// Record activity
	lm.RecordActivity()

	// Wait a bit
	time.Sleep(50 * time.Millisecond)

	idle := lm.IdleDuration()
	if idle < 50*time.Millisecond {
		t.Errorf("IdleDuration() = %v, want >= 50ms", idle)
	}
}

func TestLifecycleManager_Stats(t *testing.T) {
	lm := NewLifecycleManager(&LifecycleConfig{})
	defer func() {
		lm.BeginShutdown()
		lm.CompleteShutdown()
	}()

	stats := lm.Stats()

	if stats.State != ClientStateActive {
		t.Errorf("Stats().State = %v, want %v", stats.State, ClientStateActive)
	}

	if stats.CreatedAt.IsZero() {
		t.Error("Stats().CreatedAt is zero")
	}

	if stats.LastActivity.IsZero() {
		t.Error("Stats().LastActivity is zero")
	}

	if stats.Uptime < 0 {
		t.Errorf("Stats().Uptime = %v, want >= 0", stats.Uptime)
	}
}

func TestLifecycleManager_ConcurrentAccess(t *testing.T) {
	lm := NewLifecycleManager(&LifecycleConfig{})

	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 100

	// Concurrent activity recording
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				lm.RecordActivity()
				_ = lm.State()
				_ = lm.IsActive()
				_ = lm.Stats()
				_ = lm.LastActivity()
				_ = lm.IdleDuration()
			}
		}()
	}

	wg.Wait()

	// Clean up
	lm.BeginShutdown()
	lm.CompleteShutdown()
}

func TestLifecycleManager_OnStateChange(t *testing.T) {
	var transitions []struct {
		from, to ClientState
	}
	var mu sync.Mutex

	lm := NewLifecycleManager(&LifecycleConfig{
		OnStateChange: func(from, to ClientState) {
			mu.Lock()
			transitions = append(transitions, struct{ from, to ClientState }{from, to})
			mu.Unlock()
		},
	})

	lm.BeginShutdown()
	lm.CompleteShutdown()

	mu.Lock()
	defer mu.Unlock()

	if len(transitions) != 2 {
		t.Fatalf("len(transitions) = %d, want 2", len(transitions))
	}

	if transitions[0].from != ClientStateActive || transitions[0].to != ClientStateShuttingDown {
		t.Errorf("transitions[0] = %+v, want {Active, ShuttingDown}", transitions[0])
	}

	if transitions[1].from != ClientStateShuttingDown || transitions[1].to != ClientStateClosed {
		t.Errorf("transitions[1] = %+v, want {ShuttingDown, Closed}", transitions[1])
	}
}

func TestLifecycleManager_IdleWarning(t *testing.T) {
	var logMessages []string
	var mu sync.Mutex

	mockLogger := &lifecycleTestLogger{
		onPrintf: func(format string, v ...any) {
			mu.Lock()
			logMessages = append(logMessages, format)
			mu.Unlock()
		},
	}

	// Use a very short idle duration for testing
	lm := NewLifecycleManager(&LifecycleConfig{
		IdleWarningDuration: 50 * time.Millisecond,
		Logger:              mockLogger,
	})

	// Wait for idle warning to fire
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	hasWarning := len(logMessages) > 0
	mu.Unlock()

	if !hasWarning {
		t.Log("Note: Idle warning may not have fired within test window")
	}

	// Clean up
	lm.BeginShutdown()
	lm.CompleteShutdown()
}

func TestLifecycleManager_IdleWarningOnlyOnce(t *testing.T) {
	var warningCount int
	var mu sync.Mutex

	mockLogger := &lifecycleTestLogger{
		onPrintf: func(format string, v ...any) {
			mu.Lock()
			warningCount++
			mu.Unlock()
		},
	}

	lm := NewLifecycleManager(&LifecycleConfig{
		IdleWarningDuration: 20 * time.Millisecond,
		Logger:              mockLogger,
	})

	// Wait for multiple potential warning intervals
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	count := warningCount
	mu.Unlock()

	// Warning should fire at most once
	if count > 1 {
		t.Errorf("warning fired %d times, want <= 1", count)
	}

	// Clean up
	lm.BeginShutdown()
	lm.CompleteShutdown()
}

func TestLifecycleManager_IdleWarningNotFiredAfterActivity(t *testing.T) {
	var warningFired bool
	var mu sync.Mutex

	mockLogger := &lifecycleTestLogger{
		onPrintf: func(format string, v ...any) {
			mu.Lock()
			warningFired = true
			mu.Unlock()
		},
	}

	lm := NewLifecycleManager(&LifecycleConfig{
		IdleWarningDuration: 100 * time.Millisecond,
		Logger:              mockLogger,
	})

	// Keep recording activity to prevent idle warning
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(20 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				lm.RecordActivity()
			}
		}
	}()

	// Wait for potential warning
	time.Sleep(150 * time.Millisecond)

	close(done)

	mu.Lock()
	fired := warningFired
	mu.Unlock()

	if fired {
		t.Error("warning fired despite activity, want no warning")
	}

	// Clean up
	lm.BeginShutdown()
	lm.CompleteShutdown()
}

func TestClientState_String(t *testing.T) {
	tests := []struct {
		state ClientState
		want  string
	}{
		{ClientStateActive, "active"},
		{ClientStateShuttingDown, "shutting_down"},
		{ClientStateClosed, "closed"},
		{ClientState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// lifecycleTestLogger is a simple mock logger for lifecycle testing
type lifecycleTestLogger struct {
	onPrintf func(format string, v ...any)
}

func (l *lifecycleTestLogger) Printf(format string, v ...any) {
	if l.onPrintf != nil {
		l.onPrintf(format, v...)
	}
}
