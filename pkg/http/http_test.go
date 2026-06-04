package http

import (
	"errors"
	"fmt"
	"net/url"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Mock error types for testing
type mockRetryableError struct {
	retryable bool
}

func (e *mockRetryableError) Error() string {
	return "mock retryable error"
}

func (e *mockRetryableError) IsRetryable() bool {
	return e.retryable
}

type mockRetryAfterError struct {
	retryAfter time.Duration
}

func (e *mockRetryAfterError) Error() string {
	return "mock retry after error"
}

func (e *mockRetryAfterError) SuggestedRetryAfter() time.Duration {
	return e.retryAfter
}

func (e *mockRetryAfterError) IsRetryable() bool {
	return true
}

// mockAPIError models a server APIError (e.g. a 503) that carries both
// retryability and a Retry-After hint, mirroring pkg/errors.APIError. It is used
// to verify retry detection survives error wrapping via fmt.Errorf("...: %w", err).
type mockAPIError struct {
	statusCode int
	retryAfter time.Duration
}

func (e *mockAPIError) Error() string {
	return fmt.Sprintf("API error: status %d", e.statusCode)
}

func (e *mockAPIError) IsRetryable() bool {
	return e.statusCode == 429 || (e.statusCode >= 500 && e.statusCode < 600)
}

func (e *mockAPIError) SuggestedRetryAfter() time.Duration {
	return e.retryAfter
}

// TestExponentialBackoff_ShouldRetry tests the ShouldRetry method.
func TestExponentialBackoff_ShouldRetry(t *testing.T) {
	backoff := NewExponentialBackoff()

	tests := []struct {
		name     string
		attempt  int
		err      error
		expected bool
	}{
		{
			name:     "should retry retryable error on first attempt",
			attempt:  0,
			err:      &mockRetryableError{retryable: true},
			expected: true,
		},
		{
			name:     "should not retry non-retryable error",
			attempt:  0,
			err:      &mockRetryableError{retryable: false},
			expected: false,
		},
		{
			name:     "should not retry after max retries",
			attempt:  3,
			err:      &mockRetryableError{retryable: true},
			expected: false,
		},
		{
			name:     "should retry network timeout",
			attempt:  0,
			err:      errors.New("timeout"),
			expected: true,
		},
		{
			name:     "should not retry connection refused",
			attempt:  0,
			err:      errors.New("connection refused"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := backoff.ShouldRetry(tt.attempt, tt.err)
			if result != tt.expected {
				t.Errorf("ShouldRetry() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestExponentialBackoff_RetryDelay tests the RetryDelay method.
func TestExponentialBackoff_RetryDelay(t *testing.T) {
	backoff := &ExponentialBackoff{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       false, // Disable jitter for predictable tests
		MaxRetries:   3,
	}

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "first retry",
			attempt:  0,
			expected: 100 * time.Millisecond,
		},
		{
			name:     "second retry",
			attempt:  1,
			expected: 200 * time.Millisecond,
		},
		{
			name:     "third retry",
			attempt:  2,
			expected: 400 * time.Millisecond,
		},
		{
			name:     "capped at max delay",
			attempt:  10,
			expected: 1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := backoff.RetryDelay(tt.attempt)
			if result != tt.expected {
				t.Errorf("RetryDelay(%d) = %v, expected %v", tt.attempt, result, tt.expected)
			}
		})
	}
}

// TestExponentialBackoff_JitterNeverExceedsMaxDelay verifies that MaxDelay is a
// true upper bound even with jitter enabled. The cap is applied AFTER jitter, so
// across many samples at/above MaxDelay the returned delay must never exceed
// MaxDelay and must always be positive.
func TestExponentialBackoff_JitterNeverExceedsMaxDelay(t *testing.T) {
	backoff := &ExponentialBackoff{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		MaxRetries:   10,
	}

	maxDelay := 1 * time.Second
	// Use high attempt numbers so the pre-jitter delay is already at/above
	// MaxDelay; with the old code the 1.5x jitter factor would push it to ~1.5s.
	for _, attempt := range []int{5, 6, 7, 8, 9, 10, 20} {
		for i := 0; i < 1000; i++ {
			d := backoff.RetryDelay(attempt)
			if d > maxDelay {
				t.Fatalf("RetryDelay(%d) = %v exceeds MaxDelay %v", attempt, d, maxDelay)
			}
			if d <= 0 {
				t.Fatalf("RetryDelay(%d) = %v, expected a positive delay", attempt, d)
			}
		}
	}
}

// TestExponentialBackoff_JitterTinyDelayStaysPositive verifies the floor guard:
// even with a sub-nanosecond InitialDelay and the low end of the jitter range,
// the returned delay is never zero or negative.
func TestExponentialBackoff_JitterTinyDelayStaysPositive(t *testing.T) {
	backoff := &ExponentialBackoff{
		InitialDelay: 1, // 1 nanosecond
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		MaxRetries:   10,
	}

	for i := 0; i < 1000; i++ {
		d := backoff.RetryDelay(0)
		if d <= 0 {
			t.Fatalf("RetryDelay(0) = %v, expected a positive delay", d)
		}
	}
}

// TestExponentialBackoff_RetryDelayWithError tests RetryDelayWithError.
func TestExponentialBackoff_RetryDelayWithError(t *testing.T) {
	backoff := &ExponentialBackoff{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       false,
		MaxRetries:   3,
	}

	tests := []struct {
		name     string
		attempt  int
		err      error
		expected time.Duration
	}{
		{
			name:     "uses retry-after from error",
			attempt:  0,
			err:      &mockRetryAfterError{retryAfter: 500 * time.Millisecond},
			expected: 500 * time.Millisecond,
		},
		{
			name:     "caps retry-after at max delay",
			attempt:  0,
			err:      &mockRetryAfterError{retryAfter: 2 * time.Second},
			expected: 1 * time.Second,
		},
		{
			name:     "falls back to calculated delay",
			attempt:  1,
			err:      &mockRetryableError{retryable: true},
			expected: 200 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := backoff.RetryDelayWithError(tt.attempt, tt.err)
			if result != tt.expected {
				t.Errorf("RetryDelayWithError(%d, %v) = %v, expected %v", tt.attempt, tt.err, result, tt.expected)
			}
		})
	}
}

// TestExponentialBackoff_WrappedRetryableError verifies that retry detection
// survives error wrapping. A retryable 503 APIError carrying a Retry-After is
// wrapped via fmt.Errorf("...: %w", err); both ShouldRetry and
// RetryDelayWithError must still see through the wrapper via errors.As.
//
// Before the errors.As fix (bare type assertions err.(RetryableError) /
// err.(RetryAfterError)), the wrapped error is no longer the concrete type, so
// ShouldRetry would fall back to IsRetryableNetworkError (false) and
// RetryDelayWithError would ignore the Retry-After and return the calculated
// backoff instead.
func TestExponentialBackoff_WrappedRetryableError(t *testing.T) {
	backoff := &ExponentialBackoff{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       false,
		MaxRetries:   3,
	}

	apiErr := &mockAPIError{statusCode: 503, retryAfter: 500 * time.Millisecond}
	wrapped := fmt.Errorf("request failed: %w", apiErr)

	// Sanity check: the wrapper is not the concrete type, so a bare assertion
	// would miss it.
	if _, ok := wrapped.(RetryableError); ok {
		t.Fatal("expected wrapped error to NOT satisfy a bare RetryableError assertion")
	}

	// ShouldRetry must classify the wrapped retryable error as retryable.
	if !backoff.ShouldRetry(0, wrapped) {
		t.Error("ShouldRetry(0, wrapped 503) = false, expected true")
	}

	// RetryDelayWithError must honor the server's Retry-After through the wrapper.
	if got := backoff.RetryDelayWithError(0, wrapped); got != 500*time.Millisecond {
		t.Errorf("RetryDelayWithError(0, wrapped) = %v, expected 500ms (server Retry-After)", got)
	}
}

// TestRetryStrategies_WrappedRetryableError verifies FixedDelay and
// LinearBackoff also see through wrapped retryable errors via errors.As.
func TestRetryStrategies_WrappedRetryableError(t *testing.T) {
	apiErr := &mockAPIError{statusCode: 503}
	wrapped := fmt.Errorf("request failed: %w", apiErr)

	fd := NewFixedDelay(100*time.Millisecond, 3)
	if !fd.ShouldRetry(0, wrapped) {
		t.Error("FixedDelay.ShouldRetry(0, wrapped 503) = false, expected true")
	}

	lb := NewLinearBackoff(100*time.Millisecond, 50*time.Millisecond, 3)
	if !lb.ShouldRetry(0, wrapped) {
		t.Error("LinearBackoff.ShouldRetry(0, wrapped 503) = false, expected true")
	}
}

// TestCircuitBreaker_StateTransitions tests circuit breaker state transitions.
func TestCircuitBreaker_StateTransitions(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold:    2,
		SuccessThreshold:    2,
		Timeout:             100 * time.Millisecond,
		HalfOpenMaxRequests: 1,
	})

	// Initial state should be closed
	if cb.State() != CircuitClosed {
		t.Errorf("Initial state = %v, expected %v", cb.State(), CircuitClosed)
	}

	// Record failures to open the circuit
	cb.Record(errors.New("error 1"))
	if cb.State() != CircuitClosed {
		t.Errorf("State after 1 failure = %v, expected %v", cb.State(), CircuitClosed)
	}

	cb.Record(errors.New("error 2"))
	if cb.State() != CircuitOpen {
		t.Errorf("State after 2 failures = %v, expected %v", cb.State(), CircuitOpen)
	}

	// Circuit should block requests
	if cb.Allow() {
		t.Error("Allow() returned true for open circuit")
	}

	// Wait for timeout to transition to half-open
	time.Sleep(150 * time.Millisecond)
	if cb.State() != CircuitHalfOpen {
		t.Errorf("State after timeout = %v, expected %v", cb.State(), CircuitHalfOpen)
	}

	// Allow should work in half-open
	if !cb.Allow() {
		t.Error("Allow() returned false for half-open circuit")
	}

	// Record successes to close the circuit
	cb.Record(nil)
	if cb.State() != CircuitHalfOpen {
		t.Errorf("State after 1 success = %v, expected %v", cb.State(), CircuitHalfOpen)
	}

	// Need to allow another request in half-open
	cb.Allow()
	cb.Record(nil)
	if cb.State() != CircuitClosed {
		t.Errorf("State after 2 successes = %v, expected %v", cb.State(), CircuitClosed)
	}
}

// TestCircuitBreaker_Execute tests the Execute method.
func TestCircuitBreaker_Execute(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		Timeout:          100 * time.Millisecond,
	})

	// Execute successful function
	err := cb.Execute(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("Execute() returned error for successful function: %v", err)
	}

	// Execute failing function twice to open circuit
	for range 2 {
		_ = cb.Execute(func() error {
			return errors.New("test error")
		})
	}

	// Circuit should be open now
	err = cb.Execute(func() error {
		return nil
	})
	if err != ErrCircuitOpen {
		t.Errorf("Execute() = %v, expected %v", err, ErrCircuitOpen)
	}
}

// TestCircuitBreaker_RecoversThroughExecute verifies that a breaker created with
// DefaultCircuitBreakerConfig (SuccessThreshold:2, HalfOpenMaxRequests:1) can fully
// recover to Closed when driven exclusively through Execute() — the production path.
//
// This is a regression test for a bug where, after the first half-open probe, the
// in-flight slot was never released: halfOpenRequests stayed at its max so Allow()
// returned false forever and Record() was never called again, stranding the breaker
// in half-open. The test never calls Record() after a false Allow(); it relies solely
// on Execute() to admit and complete each probe.
func TestCircuitBreaker_RecoversThroughExecute(t *testing.T) {
	// Start from the shipped defaults (the values that trigger the bug) and only
	// shorten the timeout so the test does not wait the full 30s recovery window.
	config := DefaultCircuitBreakerConfig()
	config.Timeout = 20 * time.Millisecond

	cb := NewCircuitBreaker(config)

	// Drive the breaker open with FailureThreshold consecutive failures.
	failing := func() error { return errors.New("service unavailable") }
	for i := 0; i < config.FailureThreshold; i++ {
		_ = cb.Execute(failing)
	}
	if cb.State() != CircuitOpen {
		t.Fatalf("after %d failures: state = %v, want Open", config.FailureThreshold, cb.State())
	}

	// While open, Execute must fail fast without invoking the function.
	called := false
	if err := cb.Execute(func() error { called = true; return nil }); err != ErrCircuitOpen {
		t.Fatalf("Execute() while open = %v, want ErrCircuitOpen", err)
	}
	if called {
		t.Fatal("function was invoked while circuit was open")
	}

	// Wait out the open timeout so the breaker is eligible for half-open probing.
	time.Sleep(2 * config.Timeout)

	// Recovery loop: drive healthy requests through Execute ONLY. We never call
	// Record() ourselves, and we never call Record() after a blocked Allow() — each
	// probe is admitted and completed by Execute(). With the fix, completed probes
	// release their in-flight slot so the next probe is admitted until the breaker
	// accumulates SuccessThreshold successes and closes.
	healthy := func() error { return nil }
	deadline := time.Now().Add(2 * time.Second)
	for cb.State() != CircuitClosed && time.Now().Before(deadline) {
		if err := cb.Execute(healthy); err != nil && err != ErrCircuitOpen {
			t.Fatalf("Execute(healthy) returned unexpected error: %v", err)
		}
	}

	if cb.State() != CircuitClosed {
		t.Fatalf("breaker did not recover: state = %v, want Closed", cb.State())
	}

	// Once closed, requests must flow again.
	if err := cb.Execute(healthy); err != nil {
		t.Fatalf("Execute(healthy) after recovery = %v, want nil", err)
	}
}

// TestCircuitBreaker_Reset tests the Reset method.
func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 1,
	})

	// Open the circuit
	cb.Record(errors.New("error"))
	if cb.State() != CircuitOpen {
		t.Errorf("State = %v, expected %v", cb.State(), CircuitOpen)
	}

	// Reset should close the circuit
	cb.Reset()
	if cb.State() != CircuitClosed {
		t.Errorf("State after Reset() = %v, expected %v", cb.State(), CircuitClosed)
	}

	// Counters should be reset
	if cb.Failures() != 0 {
		t.Errorf("Failures() = %d, expected 0", cb.Failures())
	}
	if cb.ConsecutiveErrors() != 0 {
		t.Errorf("ConsecutiveErrors() = %d, expected 0", cb.ConsecutiveErrors())
	}
}

// captureLogger is a Logger that records formatted messages for assertions.
type captureLogger struct {
	mu   sync.Mutex
	msgs []string
}

func (l *captureLogger) Printf(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.msgs = append(l.msgs, fmt.Sprintf(format, v...))
}

func (l *captureLogger) count() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.msgs)
}

// TestCircuitBreaker_PanickingCallbackDoesNotCrash verifies that an
// OnStateChange callback that panics does not crash the breaker, that state
// transitions still occur, and that the panic is routed to the configured
// Logger. The callback must not run under the breaker lock, so even though it
// blocks/panics, breaker methods remain usable (no deadlock).
func TestCircuitBreaker_PanickingCallbackDoesNotCrash(t *testing.T) {
	logger := &captureLogger{}
	var calls atomic.Int32

	cb := NewCircuitBreaker(CircuitBreakerConfig{
		FailureThreshold: 2,
		Timeout:          50 * time.Millisecond,
		Logger:           logger,
		OnStateChange: func(from, to CircuitState) {
			calls.Add(1)
			panic("boom in user callback")
		},
	})

	// Drive the breaker open; this triggers the panicking callback.
	cb.Record(errors.New("error 1"))
	cb.Record(errors.New("error 2"))

	// The transition must have occurred despite the panicking callback.
	if cb.State() != CircuitOpen {
		t.Fatalf("State after 2 failures = %v, expected %v", cb.State(), CircuitOpen)
	}

	// The breaker is not deadlocked: lock-taking methods still respond.
	if cb.Allow() {
		t.Error("Allow() returned true for open circuit")
	}

	// Wait for the detached callback goroutine to run and recover.
	deadline := time.Now().Add(time.Second)
	for calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if calls.Load() == 0 {
		t.Fatal("OnStateChange callback was never invoked")
	}

	// The recovered panic must have been routed to the logger.
	for logger.count() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if logger.count() == 0 {
		t.Error("expected recovered panic to be reported via Logger")
	}
}

// TestPaginationParams_ToQuery tests PaginationParams.ToQuery.
func TestPaginationParams_ToQuery(t *testing.T) {
	tests := []struct {
		name   string
		params PaginationParams
		expect map[string]string
	}{
		{
			name: "all parameters",
			params: PaginationParams{
				Page:   2,
				Limit:  50,
				Cursor: "abc123",
			},
			expect: map[string]string{
				"page":   "2",
				"limit":  "50",
				"cursor": "abc123",
			},
		},
		{
			name: "only page",
			params: PaginationParams{
				Page: 1,
			},
			expect: map[string]string{
				"page": "1",
			},
		},
		{
			name:   "empty parameters",
			params: PaginationParams{},
			expect: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q := tt.params.ToQuery()
			for key, expectedValue := range tt.expect {
				if q.Get(key) != expectedValue {
					t.Errorf("Query parameter %q = %q, expected %q", key, q.Get(key), expectedValue)
				}
			}
			// Check no extra parameters
			if len(q) != len(tt.expect) {
				t.Errorf("Query has %d parameters, expected %d", len(q), len(tt.expect))
			}
		})
	}
}

// TestMetaResponse_HasMore tests MetaResponse.HasMore.
func TestMetaResponse_HasMore(t *testing.T) {
	tests := []struct {
		name     string
		meta     MetaResponse
		expected bool
	}{
		{
			name: "has next cursor",
			meta: MetaResponse{
				Page:       1,
				TotalPages: 3,
				NextCursor: "abc",
			},
			expected: true,
		},
		{
			name: "has more pages",
			meta: MetaResponse{
				Page:       1,
				TotalPages: 3,
			},
			expected: true,
		},
		{
			name: "last page",
			meta: MetaResponse{
				Page:       3,
				TotalPages: 3,
			},
			expected: false,
		},
		{
			name: "no more data",
			meta: MetaResponse{
				Page:       5,
				TotalPages: 3,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.meta.HasMore()
			if result != tt.expected {
				t.Errorf("HasMore() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestMergeQuery tests MergeQuery function.
func TestMergeQuery(t *testing.T) {
	q1 := url.Values{}
	q1.Set("page", "1")
	q1.Set("limit", "10")

	q2 := url.Values{}
	q2.Set("name", "test")
	q2.Add("tags", "tag1")
	q2.Add("tags", "tag2")

	result := MergeQuery(q1, q2)

	// Check all values are present
	if result.Get("page") != "1" {
		t.Errorf("Merged query page = %q, expected %q", result.Get("page"), "1")
	}
	if result.Get("limit") != "10" {
		t.Errorf("Merged query limit = %q, expected %q", result.Get("limit"), "10")
	}
	if result.Get("name") != "test" {
		t.Errorf("Merged query name = %q, expected %q", result.Get("name"), "test")
	}

	tags := result["tags"]
	if len(tags) != 2 {
		t.Errorf("Merged query has %d tags, expected 2", len(tags))
	}
	if tags[0] != "tag1" || tags[1] != "tag2" {
		t.Errorf("Merged query tags = %v, expected [tag1 tag2]", tags)
	}
}

// TestNoRetry tests the NoRetry strategy.
func TestNoRetry(t *testing.T) {
	nr := &NoRetry{}

	if nr.ShouldRetry(0, errors.New("test")) {
		t.Error("NoRetry.ShouldRetry() returned true, expected false")
	}

	if nr.RetryDelay(0) != 0 {
		t.Errorf("NoRetry.RetryDelay() = %v, expected 0", nr.RetryDelay(0))
	}
}

// TestFixedDelay tests the FixedDelay strategy.
func TestFixedDelay(t *testing.T) {
	fd := NewFixedDelay(100*time.Millisecond, 3)

	// Should retry retryable errors within max retries
	if !fd.ShouldRetry(0, &mockRetryableError{retryable: true}) {
		t.Error("FixedDelay.ShouldRetry() returned false for retryable error")
	}

	// Should not retry after max retries
	if fd.ShouldRetry(3, &mockRetryableError{retryable: true}) {
		t.Error("FixedDelay.ShouldRetry() returned true after max retries")
	}

	// Delay should be constant
	if fd.RetryDelay(0) != 100*time.Millisecond {
		t.Errorf("FixedDelay.RetryDelay(0) = %v, expected 100ms", fd.RetryDelay(0))
	}
	if fd.RetryDelay(2) != 100*time.Millisecond {
		t.Errorf("FixedDelay.RetryDelay(2) = %v, expected 100ms", fd.RetryDelay(2))
	}
}

// TestLinearBackoff tests the LinearBackoff strategy.
func TestLinearBackoff(t *testing.T) {
	lb := NewLinearBackoff(100*time.Millisecond, 50*time.Millisecond, 3)

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{
			name:     "first retry",
			attempt:  0,
			expected: 100 * time.Millisecond,
		},
		{
			name:     "second retry",
			attempt:  1,
			expected: 150 * time.Millisecond,
		},
		{
			name:     "third retry",
			attempt:  2,
			expected: 200 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := lb.RetryDelay(tt.attempt)
			if result != tt.expected {
				t.Errorf("LinearBackoff.RetryDelay(%d) = %v, expected %v", tt.attempt, result, tt.expected)
			}
		})
	}

	// Should cap at max delay
	lb.MaxDelay = 180 * time.Millisecond
	if lb.RetryDelay(2) != 180*time.Millisecond {
		t.Errorf("LinearBackoff.RetryDelay(2) = %v, expected 180ms (capped)", lb.RetryDelay(2))
	}
}

// TestIsRetryableNetworkError tests the IsRetryableNetworkError function.
func TestIsRetryableNetworkError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "timeout error",
			err:      errors.New("timeout"),
			expected: true,
		},
		{
			name:     "connection refused",
			err:      errors.New("connection refused"),
			expected: false,
		},
		{
			name:     "no such host",
			err:      errors.New("no such host"),
			expected: false,
		},
		{
			name:     "certificate error",
			err:      errors.New("certificate verify failed"),
			expected: false,
		},
		{
			name:     "reset by peer",
			err:      errors.New("connection reset by peer"),
			expected: true,
		},
		{
			name:     "broken pipe",
			err:      errors.New("broken pipe"),
			expected: true,
		},
		{
			name:     "EOF",
			err:      errors.New("unexpected EOF"),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableNetworkError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableNetworkError(%v) = %v, expected %v", tt.err, result, tt.expected)
			}
		})
	}
}

// TestCircuitBreakerWithOptions tests NewCircuitBreakerWithOptions.
func TestCircuitBreakerWithOptions(t *testing.T) {
	var stateChanges atomic.Int32
	cb := NewCircuitBreakerWithOptions(
		WithFailureThreshold(3),
		WithSuccessThreshold(1),
		WithCircuitTimeout(50*time.Millisecond),
		WithHalfOpenMaxRequests(2),
		WithStateChangeCallback(func(from, to CircuitState) {
			stateChanges.Add(1)
		}),
	)

	// Verify config was applied
	if cb.config.FailureThreshold != 3 {
		t.Errorf("FailureThreshold = %d, expected 3", cb.config.FailureThreshold)
	}
	if cb.config.SuccessThreshold != 1 {
		t.Errorf("SuccessThreshold = %d, expected 1", cb.config.SuccessThreshold)
	}

	// Trigger state change
	for range 3 {
		cb.Record(errors.New("error"))
	}

	// Give callback goroutine time to execute
	time.Sleep(10 * time.Millisecond)

	if stateChanges.Load() == 0 {
		t.Error("OnStateChange callback was not called")
	}
}
