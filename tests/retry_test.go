package langfuse_test

import (
	"context"
	"net"
	"testing"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
)

// TestExponentialBackoffShouldRetry tests the ShouldRetry method.
func TestExponentialBackoffShouldRetry(t *testing.T) {
	tests := []struct {
		name        string
		strategy    *langfuse.ExponentialBackoff
		attempt     int
		err         error
		shouldRetry bool
	}{
		{
			name:        "first attempt with retryable API error",
			strategy:    langfuse.NewExponentialBackoff(),
			attempt:     0,
			err:         &langfuse.APIError{StatusCode: 503},
			shouldRetry: true,
		},
		{
			name:        "first attempt with non-retryable API error",
			strategy:    langfuse.NewExponentialBackoff(),
			attempt:     0,
			err:         &langfuse.APIError{StatusCode: 400},
			shouldRetry: false,
		},
		{
			name:        "first attempt with rate limit",
			strategy:    langfuse.NewExponentialBackoff(),
			attempt:     0,
			err:         &langfuse.APIError{StatusCode: 429},
			shouldRetry: true,
		},
		{
			name:        "exceeds max retries",
			strategy:    langfuse.NewExponentialBackoff(),
			attempt:     3, // Default max is 3
			err:         &langfuse.APIError{StatusCode: 503},
			shouldRetry: false,
		},
		{
			name:        "custom max retries not exceeded",
			strategy:    &langfuse.ExponentialBackoff{MaxRetries: 5},
			attempt:     4,
			err:         &langfuse.APIError{StatusCode: 503},
			shouldRetry: true,
		},
		{
			name:        "custom max retries exceeded",
			strategy:    &langfuse.ExponentialBackoff{MaxRetries: 5},
			attempt:     5,
			err:         &langfuse.APIError{StatusCode: 503},
			shouldRetry: false,
		},
		{
			name:        "retryable network error",
			strategy:    langfuse.NewExponentialBackoff(),
			attempt:     0,
			err:         context.DeadlineExceeded,
			shouldRetry: true,
		},
		{
			name:        "non-retryable network error",
			strategy:    langfuse.NewExponentialBackoff(),
			attempt:     0,
			err:         &net.DNSError{Err: "no such host"},
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.strategy.ShouldRetry(tt.attempt, tt.err)
			if got != tt.shouldRetry {
				t.Errorf("ShouldRetry(%d, %v) = %v, want %v", tt.attempt, tt.err, got, tt.shouldRetry)
			}
		})
	}
}

// TestExponentialBackoffRetryDelay tests the delay calculation.
func TestExponentialBackoffRetryDelay(t *testing.T) {
	strategy := &langfuse.ExponentialBackoff{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       false, // Disable jitter for deterministic tests
		MaxRetries:   5,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 400 * time.Millisecond},
		{3, 800 * time.Millisecond},
		{4, 1 * time.Second}, // Capped at MaxDelay
		{5, 1 * time.Second}, // Still capped
	}

	for _, tt := range tests {
		t.Run("attempt_"+string(rune('0'+tt.attempt)), func(t *testing.T) {
			got := strategy.RetryDelay(tt.attempt)
			if got != tt.expected {
				t.Errorf("RetryDelay(%d) = %v, want %v", tt.attempt, got, tt.expected)
			}
		})
	}
}

// TestExponentialBackoffRetryDelayWithJitter tests that jitter is applied.
func TestExponentialBackoffRetryDelayWithJitter(t *testing.T) {
	strategy := &langfuse.ExponentialBackoff{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     1 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		MaxRetries:   5,
	}

	// Run multiple times to verify jitter produces variation
	delays := make([]time.Duration, 10)
	for i := 0; i < 10; i++ {
		delays[i] = strategy.RetryDelay(0)
	}

	// With jitter, delays should be between 50ms and 150ms (0.5x to 1.5x of 100ms)
	for i, d := range delays {
		if d < 50*time.Millisecond || d > 150*time.Millisecond {
			t.Errorf("delays[%d] = %v, want between 50ms and 150ms", i, d)
		}
	}
}

// TestExponentialBackoffRetryDelayWithError tests Retry-After header handling.
func TestExponentialBackoffRetryDelayWithError(t *testing.T) {
	strategy := &langfuse.ExponentialBackoff{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		Multiplier:   2.0,
		Jitter:       false,
		MaxRetries:   5,
	}

	tests := []struct {
		name     string
		attempt  int
		err      error
		expected time.Duration
	}{
		{
			name:     "no retry-after header",
			attempt:  0,
			err:      &langfuse.APIError{StatusCode: 503},
			expected: 100 * time.Millisecond,
		},
		{
			name:     "retry-after within max",
			attempt:  0,
			err:      &langfuse.APIError{StatusCode: 429, RetryAfter: 2 * time.Second},
			expected: 2 * time.Second,
		},
		{
			name:     "retry-after exceeds max",
			attempt:  0,
			err:      &langfuse.APIError{StatusCode: 429, RetryAfter: 10 * time.Second},
			expected: 5 * time.Second, // Capped at MaxDelay
		},
		{
			name:     "non-API error falls back to calculated",
			attempt:  1,
			err:      context.DeadlineExceeded,
			expected: 200 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.RetryDelayWithError(tt.attempt, tt.err)
			if got != tt.expected {
				t.Errorf("RetryDelayWithError(%d, %v) = %v, want %v", tt.attempt, tt.err, got, tt.expected)
			}
		})
	}
}

// TestExponentialBackoffDefaults tests that defaults are applied.
func TestExponentialBackoffDefaults(t *testing.T) {
	strategy := &langfuse.ExponentialBackoff{} // All zero values

	// Should use defaults
	if !strategy.ShouldRetry(0, &langfuse.APIError{StatusCode: 503}) {
		t.Error("ShouldRetry should return true with default max retries")
	}
	if strategy.ShouldRetry(3, &langfuse.APIError{StatusCode: 503}) {
		t.Error("ShouldRetry should return false when attempt >= default max retries (3)")
	}

	delay := strategy.RetryDelay(0)
	if delay < 500*time.Millisecond || delay > 1500*time.Millisecond {
		t.Errorf("RetryDelay(0) = %v, want ~1s (default with jitter)", delay)
	}
}

// TestNoRetry tests the NoRetry strategy.
func TestNoRetry(t *testing.T) {
	strategy := &langfuse.NoRetry{}

	if strategy.ShouldRetry(0, &langfuse.APIError{StatusCode: 503}) {
		t.Error("NoRetry.ShouldRetry should always return false")
	}
	if strategy.ShouldRetry(0, context.DeadlineExceeded) {
		t.Error("NoRetry.ShouldRetry should always return false")
	}
	if strategy.RetryDelay(0) != 0 {
		t.Error("NoRetry.RetryDelay should always return 0")
	}
}

// TestFixedDelayShouldRetry tests FixedDelay.ShouldRetry.
func TestFixedDelayShouldRetry(t *testing.T) {
	strategy := langfuse.NewFixedDelay(500*time.Millisecond, 3)

	tests := []struct {
		name        string
		attempt     int
		err         error
		shouldRetry bool
	}{
		{
			name:        "first attempt retryable",
			attempt:     0,
			err:         &langfuse.APIError{StatusCode: 503},
			shouldRetry: true,
		},
		{
			name:        "last valid attempt",
			attempt:     2,
			err:         &langfuse.APIError{StatusCode: 503},
			shouldRetry: true,
		},
		{
			name:        "exceeds max retries",
			attempt:     3,
			err:         &langfuse.APIError{StatusCode: 503},
			shouldRetry: false,
		},
		{
			name:        "non-retryable API error",
			attempt:     0,
			err:         &langfuse.APIError{StatusCode: 400},
			shouldRetry: false,
		},
		{
			name:        "retryable network error",
			attempt:     0,
			err:         context.DeadlineExceeded,
			shouldRetry: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.ShouldRetry(tt.attempt, tt.err)
			if got != tt.shouldRetry {
				t.Errorf("ShouldRetry(%d, %v) = %v, want %v", tt.attempt, tt.err, got, tt.shouldRetry)
			}
		})
	}
}

// TestFixedDelayRetryDelay tests that delay is constant.
func TestFixedDelayRetryDelay(t *testing.T) {
	delay := 500 * time.Millisecond
	strategy := langfuse.NewFixedDelay(delay, 3)

	for attempt := 0; attempt < 5; attempt++ {
		got := strategy.RetryDelay(attempt)
		if got != delay {
			t.Errorf("RetryDelay(%d) = %v, want %v", attempt, got, delay)
		}
	}
}

// TestLinearBackoffShouldRetry tests LinearBackoff.ShouldRetry.
func TestLinearBackoffShouldRetry(t *testing.T) {
	strategy := langfuse.NewLinearBackoff(100*time.Millisecond, 100*time.Millisecond, 3)

	tests := []struct {
		name        string
		attempt     int
		err         error
		shouldRetry bool
	}{
		{
			name:        "first attempt retryable",
			attempt:     0,
			err:         &langfuse.APIError{StatusCode: 503},
			shouldRetry: true,
		},
		{
			name:        "exceeds max retries",
			attempt:     3,
			err:         &langfuse.APIError{StatusCode: 503},
			shouldRetry: false,
		},
		{
			name:        "non-retryable API error",
			attempt:     0,
			err:         &langfuse.APIError{StatusCode: 401},
			shouldRetry: false,
		},
		{
			name:        "retryable network error",
			attempt:     0,
			err:         context.DeadlineExceeded,
			shouldRetry: true,
		},
		{
			name:        "non-retryable network error",
			attempt:     0,
			err:         &net.DNSError{Err: "no such host"},
			shouldRetry: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := strategy.ShouldRetry(tt.attempt, tt.err)
			if got != tt.shouldRetry {
				t.Errorf("ShouldRetry(%d, %v) = %v, want %v", tt.attempt, tt.err, got, tt.shouldRetry)
			}
		})
	}
}

// TestLinearBackoffRetryDelay tests linear delay increase.
func TestLinearBackoffRetryDelay(t *testing.T) {
	strategy := &langfuse.LinearBackoff{
		InitialDelay: 100 * time.Millisecond,
		Increment:    100 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		MaxRetries:   10,
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond},
		{1, 200 * time.Millisecond},
		{2, 300 * time.Millisecond},
		{3, 400 * time.Millisecond},
		{4, 500 * time.Millisecond}, // Capped at MaxDelay
		{5, 500 * time.Millisecond}, // Still capped
	}

	for _, tt := range tests {
		t.Run("attempt_"+string(rune('0'+tt.attempt)), func(t *testing.T) {
			got := strategy.RetryDelay(tt.attempt)
			if got != tt.expected {
				t.Errorf("RetryDelay(%d) = %v, want %v", tt.attempt, got, tt.expected)
			}
		})
	}
}

// TestNewExponentialBackoff tests the constructor.
func TestNewExponentialBackoff(t *testing.T) {
	strategy := langfuse.NewExponentialBackoff()

	if strategy.InitialDelay != 1*time.Second {
		t.Errorf("InitialDelay = %v, want 1s", strategy.InitialDelay)
	}
	if strategy.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", strategy.MaxDelay)
	}
	if strategy.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v, want 2.0", strategy.Multiplier)
	}
	if !strategy.Jitter {
		t.Error("Jitter should be true by default")
	}
	if strategy.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", strategy.MaxRetries)
	}
}

// TestRetryStrategyInterface ensures all strategies implement the interface.
func TestRetryStrategyInterface(t *testing.T) {
	var _ langfuse.RetryStrategy = (*langfuse.ExponentialBackoff)(nil)
	var _ langfuse.RetryStrategy = (*langfuse.NoRetry)(nil)
	var _ langfuse.RetryStrategy = (*langfuse.FixedDelay)(nil)
	var _ langfuse.RetryStrategy = (*langfuse.LinearBackoff)(nil)

	// ExponentialBackoff also implements RetryStrategyWithError
	var _ langfuse.RetryStrategyWithError = (*langfuse.ExponentialBackoff)(nil)
}
