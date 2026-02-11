package langfuse_test

import (
	"errors"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
	pkghttp "github.com/jdziat/langfuse-go/pkg/http"
)

// isCircuitOpenError checks if the error is a circuit open error.
// This handles both the root package's ErrCircuitOpen and pkg/http's version.
func isCircuitOpenError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, langfuse.ErrCircuitOpen) {
		return true
	}
	if errors.Is(err, pkghttp.ErrCircuitOpen) {
		return true
	}
	// Fallback: check error message
	return strings.Contains(err.Error(), "circuit breaker is open")
}

func TestCircuitState_String(t *testing.T) {
	tests := []struct {
		state    langfuse.CircuitState
		expected string
	}{
		{langfuse.CircuitClosed, "closed"},
		{langfuse.CircuitOpen, "open"},
		{langfuse.CircuitHalfOpen, "half-open"},
		{langfuse.CircuitState(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			if got := tt.state.String(); got != tt.expected {
				t.Errorf("CircuitState.String() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestCircuitBreaker_DefaultConfig(t *testing.T) {
	config := langfuse.DefaultCircuitBreakerConfig()

	if config.FailureThreshold != 5 {
		t.Errorf("FailureThreshold = %d, want 5", config.FailureThreshold)
	}
	if config.SuccessThreshold != 2 {
		t.Errorf("SuccessThreshold = %d, want 2", config.SuccessThreshold)
	}
	if config.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v, want 30s", config.Timeout)
	}
	if config.HalfOpenMaxRequests != 1 {
		t.Errorf("HalfOpenMaxRequests = %d, want 1", config.HalfOpenMaxRequests)
	}
}

func TestCircuitBreaker_StartsInClosedState(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.DefaultCircuitBreakerConfig())

	if cb.State() != langfuse.CircuitClosed {
		t.Errorf("initial state = %v, want Closed", cb.State())
	}
}

func TestCircuitBreaker_OpensAfterFailureThreshold(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold: 3,
		Timeout:          time.Minute,
	})

	testErr := errors.New("test error")

	// Record failures up to threshold
	for i := 0; i < 3; i++ {
		cb.Record(testErr)
	}

	if cb.State() != langfuse.CircuitOpen {
		t.Errorf("state after %d failures = %v, want Open", 3, cb.State())
	}
}

func TestCircuitBreaker_ExecuteBlocksWhenOpen(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          time.Hour, // Long timeout to stay open
	})

	// Force open
	cb.Record(errors.New("error"))

	// Try to execute
	called := false
	err := cb.Execute(func() error {
		called = true
		return nil
	})

	// Check for circuit open error (either root's or pkg/http's version)
	if !isCircuitOpenError(err) {
		t.Errorf("Execute() error = %v, want ErrCircuitOpen", err)
	}
	if called {
		t.Error("function was called when circuit was open")
	}
}

func TestCircuitBreaker_TransitionsToHalfOpenAfterTimeout(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          10 * time.Millisecond,
	})

	// Force open
	cb.Record(errors.New("error"))
	if cb.State() != langfuse.CircuitOpen {
		t.Fatal("circuit should be open")
	}

	// Wait for timeout
	time.Sleep(20 * time.Millisecond)

	// Should transition to half-open
	if cb.State() != langfuse.CircuitHalfOpen {
		t.Errorf("state after timeout = %v, want HalfOpen", cb.State())
	}
}

func TestCircuitBreaker_ClosesAfterSuccessInHalfOpen(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold:    1,
		SuccessThreshold:    2,
		Timeout:             10 * time.Millisecond,
		HalfOpenMaxRequests: 3,
	})

	// Force open
	cb.Record(errors.New("error"))

	// Wait for half-open
	time.Sleep(20 * time.Millisecond)

	// Record successes
	cb.Allow() // Enter half-open
	cb.Record(nil)
	cb.Allow()
	cb.Record(nil)

	if cb.State() != langfuse.CircuitClosed {
		t.Errorf("state after successes = %v, want Closed", cb.State())
	}
}

func TestCircuitBreaker_ReturnsToOpenOnFailureInHalfOpen(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold:    1,
		Timeout:             10 * time.Millisecond,
		HalfOpenMaxRequests: 2,
	})

	// Force open
	cb.Record(errors.New("error"))

	// Wait for half-open
	time.Sleep(20 * time.Millisecond)

	// Enter half-open and fail
	cb.Allow()
	cb.Record(errors.New("another error"))

	if cb.State() != langfuse.CircuitOpen {
		t.Errorf("state after failure in half-open = %v, want Open", cb.State())
	}
}

func TestCircuitBreaker_LimitsRequestsInHalfOpen(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold:    1,
		Timeout:             10 * time.Millisecond,
		HalfOpenMaxRequests: 2,
	})

	// Force open
	cb.Record(errors.New("error"))

	// Wait for half-open
	time.Sleep(20 * time.Millisecond)

	// First two requests should be allowed
	if !cb.Allow() {
		t.Error("first request in half-open should be allowed")
	}
	if !cb.Allow() {
		t.Error("second request in half-open should be allowed")
	}

	// Third should be blocked
	if cb.Allow() {
		t.Error("third request in half-open should be blocked")
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          time.Hour,
	})

	// Force open
	cb.Record(errors.New("error"))
	if cb.State() != langfuse.CircuitOpen {
		t.Fatal("circuit should be open")
	}

	// Reset
	cb.Reset()

	if cb.State() != langfuse.CircuitClosed {
		t.Errorf("state after reset = %v, want Closed", cb.State())
	}
	if cb.Failures() != 0 {
		t.Errorf("failures after reset = %d, want 0", cb.Failures())
	}
	if cb.ConsecutiveErrors() != 0 {
		t.Errorf("consecutive errors after reset = %d, want 0", cb.ConsecutiveErrors())
	}
}

func TestCircuitBreaker_SuccessResetsConsecutiveErrors(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold: 3,
	})

	// Record some failures
	cb.Record(errors.New("error 1"))
	cb.Record(errors.New("error 2"))

	if cb.ConsecutiveErrors() != 2 {
		t.Errorf("consecutive errors = %d, want 2", cb.ConsecutiveErrors())
	}

	// Record success
	cb.Record(nil)

	if cb.ConsecutiveErrors() != 0 {
		t.Errorf("consecutive errors after success = %d, want 0", cb.ConsecutiveErrors())
	}

	// Circuit should still be closed
	if cb.State() != langfuse.CircuitClosed {
		t.Errorf("state = %v, want Closed", cb.State())
	}
}

func TestCircuitBreaker_CustomFailureChecker(t *testing.T) {
	// Only count specific errors as failures
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold: 2,
		IsFailure: func(err error) bool {
			return err != nil && err.Error() == "critical"
		},
	})

	// Non-critical errors shouldn't count
	cb.Record(errors.New("minor"))
	cb.Record(errors.New("warning"))

	if cb.State() != langfuse.CircuitClosed {
		t.Errorf("state after non-critical errors = %v, want Closed", cb.State())
	}

	// Critical errors should count
	cb.Record(errors.New("critical"))
	cb.Record(errors.New("critical"))

	if cb.State() != langfuse.CircuitOpen {
		t.Errorf("state after critical errors = %v, want Open", cb.State())
	}
}

func TestCircuitBreaker_StateChangeCallback(t *testing.T) {
	var transitions []struct {
		from, to langfuse.CircuitState
	}
	var mu sync.Mutex

	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          10 * time.Millisecond,
		OnStateChange: func(from, to langfuse.CircuitState) {
			mu.Lock()
			transitions = append(transitions, struct{ from, to langfuse.CircuitState }{from, to})
			mu.Unlock()
		},
	})

	// Trigger open
	cb.Record(errors.New("error"))

	// Wait for callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if len(transitions) < 1 {
		t.Error("expected at least one state transition")
	} else {
		if transitions[0].from != langfuse.CircuitClosed || transitions[0].to != langfuse.CircuitOpen {
			t.Errorf("first transition = %v->%v, want Closed->Open",
				transitions[0].from, transitions[0].to)
		}
	}
	mu.Unlock()
}

func TestCircuitBreaker_Execute(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.DefaultCircuitBreakerConfig())

	// Successful execution
	result := 0
	err := cb.Execute(func() error {
		result = 42
		return nil
	})

	if err != nil {
		t.Errorf("Execute() error = %v, want nil", err)
	}
	if result != 42 {
		t.Errorf("result = %d, want 42", result)
	}

	// Failed execution
	testErr := errors.New("test error")
	err = cb.Execute(func() error {
		return testErr
	})

	if !errors.Is(err, testErr) {
		t.Errorf("Execute() error = %v, want %v", err, testErr)
	}
}

func TestCircuitBreaker_ConcurrentAccess(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold:    100,
		SuccessThreshold:    10,
		Timeout:             time.Millisecond,
		HalfOpenMaxRequests: 50,
	})

	var wg sync.WaitGroup
	var successCount int64
	var errorCount int64

	// Concurrent executions
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				err := cb.Execute(func() error {
					if idx%2 == 0 {
						return errors.New("error")
					}
					return nil
				})

				if err == nil {
					atomic.AddInt64(&successCount, 1)
				} else {
					atomic.AddInt64(&errorCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	// Should have processed all requests without panics
	total := atomic.LoadInt64(&successCount) + atomic.LoadInt64(&errorCount)
	if total != 10000 {
		t.Errorf("total requests = %d, want 10000", total)
	}
}

func TestCircuitBreaker_WithOptions(t *testing.T) {
	cb := langfuse.NewCircuitBreakerWithOptions(
		langfuse.WithFailureThreshold(10),
		langfuse.WithSuccessThreshold(5),
		langfuse.WithCircuitTimeout(time.Minute),
		langfuse.WithHalfOpenMaxRequests(3),
		langfuse.WithStateChangeCallback(func(from, to langfuse.CircuitState) {
			// Callback registered (verified by not panicking)
		}),
		langfuse.WithFailureChecker(func(err error) bool {
			return err != nil && err.Error() == "fatal"
		}),
	)

	// Verify FailureThreshold by observing behavior: should take 10 failures to open
	for i := 0; i < 9; i++ {
		cb.Record(errors.New("fatal"))
	}
	if cb.State() != langfuse.CircuitClosed {
		t.Errorf("after 9 failures: state = %v, want Closed", cb.State())
	}
	cb.Record(errors.New("fatal"))
	if cb.State() != langfuse.CircuitOpen {
		t.Errorf("after 10 failures: state = %v, want Open", cb.State())
	}

	// Test custom failure checker - non-fatal errors shouldn't count
	cb.Reset()
	cb.Record(errors.New("warning")) // Should not count
	if cb.ConsecutiveErrors() != 0 {
		t.Errorf("consecutive errors after non-fatal = %d, want 0", cb.ConsecutiveErrors())
	}
}

func TestCircuitBreaker_ZeroConfigValues(t *testing.T) {
	// Zero values should get defaults
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{})

	// Verify default FailureThreshold (5) by observing behavior
	for i := 0; i < 4; i++ {
		cb.Record(errors.New("error"))
	}
	if cb.State() != langfuse.CircuitClosed {
		t.Errorf("after 4 failures: state = %v, want Closed (default threshold is 5)", cb.State())
	}
	cb.Record(errors.New("error"))
	if cb.State() != langfuse.CircuitOpen {
		t.Errorf("after 5 failures: state = %v, want Open (default threshold is 5)", cb.State())
	}
}

func TestCircuitBreaker_FailuresCounter(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold: 10,
	})

	for i := 0; i < 5; i++ {
		cb.Record(errors.New("error"))
	}

	if cb.Failures() != 5 {
		t.Errorf("Failures() = %d, want 5", cb.Failures())
	}
}

func TestCircuitBreaker_AllowInClosedState(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.DefaultCircuitBreakerConfig())

	// Should always allow in closed state
	for i := 0; i < 10; i++ {
		if !cb.Allow() {
			t.Error("Allow() should always return true in closed state")
		}
	}
}

func TestCircuitBreaker_ExecuteInHalfOpenWithFailure(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold:    1,
		Timeout:             10 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	})

	// Force open
	cb.Record(errors.New("error"))

	// Wait for half-open
	time.Sleep(20 * time.Millisecond)

	// Execute and fail in half-open
	testErr := errors.New("half-open failure")
	err := cb.Execute(func() error {
		return testErr
	})

	if !errors.Is(err, testErr) {
		t.Errorf("Execute() error = %v, want %v", err, testErr)
	}

	// Circuit should be back to open
	if cb.State() != langfuse.CircuitOpen {
		t.Errorf("state after failure in half-open = %v, want Open", cb.State())
	}
}

func TestCircuitBreaker_AllowBlockedWhenHalfOpenLimitReached(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold:    1,
		Timeout:             10 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	})

	// Force open
	cb.Record(errors.New("error"))

	// Wait for half-open
	time.Sleep(20 * time.Millisecond)

	// First request allowed
	if !cb.Allow() {
		t.Error("first Allow() in half-open should return true")
	}

	// Second request blocked (limit is 1)
	if cb.Allow() {
		t.Error("second Allow() in half-open should return false when limit reached")
	}
}

func TestCircuitBreaker_MultipleTransitions(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    1,
		Timeout:             10 * time.Millisecond,
		HalfOpenMaxRequests: 5,
	})

	// Closed -> Open
	cb.Record(errors.New("error"))
	cb.Record(errors.New("error"))
	if cb.State() != langfuse.CircuitOpen {
		t.Errorf("expected Open, got %v", cb.State())
	}

	// Wait -> Half-Open
	time.Sleep(20 * time.Millisecond)
	if cb.State() != langfuse.CircuitHalfOpen {
		t.Errorf("expected HalfOpen, got %v", cb.State())
	}

	// Half-Open -> Closed (on success)
	cb.Allow()
	cb.Record(nil)
	if cb.State() != langfuse.CircuitClosed {
		t.Errorf("expected Closed, got %v", cb.State())
	}

	// Closed -> Open again
	cb.Record(errors.New("error"))
	cb.Record(errors.New("error"))
	if cb.State() != langfuse.CircuitOpen {
		t.Errorf("expected Open again, got %v", cb.State())
	}
}

func TestCircuitBreaker_RecordNilInClosed(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold: 3,
	})

	// Record some failures
	cb.Record(errors.New("error"))
	cb.Record(errors.New("error"))

	// Record success
	cb.Record(nil)

	// Should stay closed and reset consecutive errors
	if cb.State() != langfuse.CircuitClosed {
		t.Errorf("state = %v, want Closed", cb.State())
	}
	if cb.ConsecutiveErrors() != 0 {
		t.Errorf("consecutive errors = %d, want 0", cb.ConsecutiveErrors())
	}
}

func TestCircuitBreaker_ErrCircuitOpen(t *testing.T) {
	// Test that ErrCircuitOpen has the expected message
	expected := "langfuse: circuit breaker is open"
	if langfuse.ErrCircuitOpen.Error() != expected {
		t.Errorf("ErrCircuitOpen.Error() = %q, want %q", langfuse.ErrCircuitOpen.Error(), expected)
	}
}

func TestCircuitBreaker_RecordInOpenState(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold: 1,
		Timeout:          time.Hour, // Long timeout to stay open
	})

	// Force open
	cb.Record(errors.New("error"))

	// Recording more errors in open state should have no effect
	cb.Record(errors.New("another error"))
	cb.Record(nil) // Even success shouldn't change anything

	// Should still be open
	if cb.State() != langfuse.CircuitOpen {
		t.Errorf("state = %v, want Open", cb.State())
	}
}

func TestCircuitBreaker_StateStringValues(t *testing.T) {
	// Ensure String() returns correct values for all states
	tests := []struct {
		state    langfuse.CircuitState
		expected string
	}{
		{langfuse.CircuitClosed, "closed"},
		{langfuse.CircuitOpen, "open"},
		{langfuse.CircuitHalfOpen, "half-open"},
		{langfuse.CircuitState(100), "unknown"},
		{langfuse.CircuitState(-1), "unknown"},
	}

	for _, tt := range tests {
		got := tt.state.String()
		if got != tt.expected {
			t.Errorf("CircuitState(%d).String() = %q, want %q", tt.state, got, tt.expected)
		}
	}
}

// TestCircuitBreaker_FullStateTransitionCycle tests the complete lifecycle of
// state transitions: Closed -> Open -> HalfOpen -> Closed -> Open -> HalfOpen -> Open.
func TestCircuitBreaker_FullStateTransitionCycle(t *testing.T) {
	var transitions []struct {
		from, to langfuse.CircuitState
	}
	var mu sync.Mutex

	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold:    3,
		SuccessThreshold:    2,
		Timeout:             20 * time.Millisecond,
		HalfOpenMaxRequests: 5,
		OnStateChange: func(from, to langfuse.CircuitState) {
			mu.Lock()
			transitions = append(transitions, struct{ from, to langfuse.CircuitState }{from, to})
			mu.Unlock()
		},
	})

	// Phase 1: Closed -> Open (fail 3 times)
	if cb.State() != langfuse.CircuitClosed {
		t.Fatalf("initial state = %v, want Closed", cb.State())
	}

	for i := 0; i < 3; i++ {
		cb.Record(errors.New("failure"))
	}

	if cb.State() != langfuse.CircuitOpen {
		t.Fatalf("after 3 failures: state = %v, want Open", cb.State())
	}

	// Phase 2: Open -> HalfOpen (wait for timeout)
	time.Sleep(30 * time.Millisecond)

	if cb.State() != langfuse.CircuitHalfOpen {
		t.Fatalf("after timeout: state = %v, want HalfOpen", cb.State())
	}

	// Phase 3: HalfOpen -> Closed (succeed 2 times)
	cb.Allow()
	cb.Record(nil)
	if cb.State() != langfuse.CircuitHalfOpen {
		t.Fatalf("after 1 success: state = %v, want HalfOpen", cb.State())
	}

	cb.Allow()
	cb.Record(nil)
	if cb.State() != langfuse.CircuitClosed {
		t.Fatalf("after 2 successes: state = %v, want Closed", cb.State())
	}

	// Phase 4: Closed -> Open again
	for i := 0; i < 3; i++ {
		cb.Record(errors.New("failure"))
	}

	if cb.State() != langfuse.CircuitOpen {
		t.Fatalf("after 3 more failures: state = %v, want Open", cb.State())
	}

	// Phase 5: Open -> HalfOpen
	time.Sleep(30 * time.Millisecond)

	if cb.State() != langfuse.CircuitHalfOpen {
		t.Fatalf("after second timeout: state = %v, want HalfOpen", cb.State())
	}

	// Phase 6: HalfOpen -> Open (fail in half-open)
	cb.Allow()
	cb.Record(errors.New("failure in half-open"))

	if cb.State() != langfuse.CircuitOpen {
		t.Fatalf("after failure in half-open: state = %v, want Open", cb.State())
	}

	// Verify all transitions were recorded
	time.Sleep(10 * time.Millisecond) // Allow callback to execute

	mu.Lock()
	expectedTransitions := 6 // Closed->Open, Open->HalfOpen, HalfOpen->Closed, Closed->Open, Open->HalfOpen, HalfOpen->Open
	if len(transitions) != expectedTransitions {
		t.Errorf("expected %d transitions, got %d: %v", expectedTransitions, len(transitions), transitions)
	}
	mu.Unlock()
}

// TestCircuitBreaker_RapidStateChanges tests the circuit breaker under rapid
// state change conditions to ensure no race conditions.
func TestCircuitBreaker_RapidStateChanges(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    1,
		Timeout:             5 * time.Millisecond,
		HalfOpenMaxRequests: 10,
	})

	var wg sync.WaitGroup

	// Goroutine 1: Record failures
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			cb.Record(errors.New("error"))
			time.Sleep(time.Millisecond)
		}
	}()

	// Goroutine 2: Record successes
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			cb.Record(nil)
			time.Sleep(time.Millisecond)
		}
	}()

	// Goroutine 3: Check Allow() and State()
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			_ = cb.Allow()
			_ = cb.State()
			time.Sleep(500 * time.Microsecond)
		}
	}()

	// Goroutine 4: Execute operations
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			_ = cb.Execute(func() error {
				if i%2 == 0 {
					return errors.New("error")
				}
				return nil
			})
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()

	// Test should complete without deadlocks or panics
	// Final state doesn't matter, we're testing for race conditions
	t.Logf("Final state: %v, Failures: %d, Consecutive: %d",
		cb.State(), cb.Failures(), cb.ConsecutiveErrors())
}

// TestCircuitBreaker_HalfOpenRequestTracking tests that half-open request
// tracking accurately limits the total number of requests allowed.
func TestCircuitBreaker_HalfOpenRequestTracking(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold:    1,
		SuccessThreshold:    3,
		Timeout:             10 * time.Millisecond,
		HalfOpenMaxRequests: 3,
	})

	// Force open
	cb.Record(errors.New("error"))

	// Wait for half-open
	time.Sleep(20 * time.Millisecond)

	if cb.State() != langfuse.CircuitHalfOpen {
		t.Fatalf("expected half-open state, got %s", cb.State())
	}

	// All 3 requests should be allowed (exhausts the half-open budget)
	for i := 0; i < 3; i++ {
		if !cb.Allow() {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// 4th request should be blocked (budget exhausted)
	if cb.Allow() {
		t.Error("4th request should be blocked - half-open budget exhausted")
	}

	// Record success - this doesn't free up slots, it counts toward SuccessThreshold
	cb.Record(nil)

	// Still blocked - recording success doesn't replenish the half-open budget
	if cb.Allow() {
		t.Error("request should still be blocked after single success")
	}

	// Record 2 more successes to reach SuccessThreshold (3)
	cb.Record(nil)
	cb.Record(nil)

	// Now circuit should be closed
	if cb.State() != langfuse.CircuitClosed {
		t.Errorf("expected closed state after 3 successes, got %s", cb.State())
	}

	// Requests should now be allowed (circuit is closed)
	if !cb.Allow() {
		t.Error("request should be allowed after circuit closes")
	}
}

// TestCircuitBreaker_ConsecutiveSuccessTracking tests that consecutive
// successes are tracked correctly in half-open state.
func TestCircuitBreaker_ConsecutiveSuccessTracking(t *testing.T) {
	cb := langfuse.NewCircuitBreaker(langfuse.CircuitBreakerConfig{
		FailureThreshold:    1,
		SuccessThreshold:    3,
		Timeout:             10 * time.Millisecond,
		HalfOpenMaxRequests: 10,
	})

	// Force open
	cb.Record(errors.New("error"))

	// Wait for half-open
	time.Sleep(20 * time.Millisecond)

	// 2 successes - should still be half-open
	cb.Allow()
	cb.Record(nil)
	cb.Allow()
	cb.Record(nil)

	if cb.State() != langfuse.CircuitHalfOpen {
		t.Errorf("after 2 successes: state = %v, want HalfOpen", cb.State())
	}

	// 3rd success - should transition to closed
	cb.Allow()
	cb.Record(nil)

	if cb.State() != langfuse.CircuitClosed {
		t.Errorf("after 3 successes: state = %v, want Closed", cb.State())
	}
}
