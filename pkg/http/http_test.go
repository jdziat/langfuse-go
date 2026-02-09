package http

import (
	"errors"
	"net/url"
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

func (e *mockRetryAfterError) RetryAfter() time.Duration {
	return e.retryAfter
}

func (e *mockRetryAfterError) IsRetryable() bool {
	return true
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
	stateChanges := 0
	cb := NewCircuitBreakerWithOptions(
		WithFailureThreshold(3),
		WithSuccessThreshold(1),
		WithCircuitTimeout(50*time.Millisecond),
		WithHalfOpenMaxRequests(2),
		WithStateChangeCallback(func(from, to CircuitState) {
			stateChanges++
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

	if stateChanges == 0 {
		t.Error("OnStateChange callback was not called")
	}
}
