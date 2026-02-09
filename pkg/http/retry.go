package http

import (
	"context"
	"errors"
	"math"
	"math/rand/v2"
	"net"
	"net/url"
	"strings"
	"syscall"
	"time"
)

// RetryableError is an interface for errors that know if they're retryable.
type RetryableError interface {
	error
	IsRetryable() bool
}

// RetryAfterError is an interface for errors with retry-after hints.
type RetryAfterError interface {
	error
	RetryAfter() time.Duration
}

// IsRetryableNetworkError determines if a network error is transient and should be retried.
// Returns false for permanent errors like DNS failures, connection refused, TLS errors.
func IsRetryableNetworkError(err error) bool {
	if err == nil {
		return false
	}

	// Context cancellation/timeout - depends on the context
	if errors.Is(err, context.DeadlineExceeded) {
		return true // Timeout - worth retrying
	}
	if errors.Is(err, context.Canceled) {
		return false // Explicit cancellation - don't retry
	}

	// Check for net.Error timeout
	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return true // Timeout - worth retrying
		}
	}

	// Check for specific syscall errors
	var syscallErr syscall.Errno
	if errors.As(err, &syscallErr) {
		switch syscallErr {
		case syscall.ECONNRESET: // Connection reset by peer
			return true
		case syscall.ETIMEDOUT: // Connection timed out
			return true
		case syscall.ECONNREFUSED: // Connection refused - server not listening
			return false
		case syscall.ENETUNREACH: // Network unreachable
			return false
		case syscall.EHOSTUNREACH: // Host unreachable
			return false
		}
	}

	// Check for DNS errors - usually permanent
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		// DNS lookup failures are usually permanent
		// (typos in hostname, non-existent domain, etc.)
		return false
	}

	// Check for URL errors
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		// Recurse into the underlying error
		return IsRetryableNetworkError(urlErr.Err)
	}

	// Check for common error message patterns (fallback)
	errStr := err.Error()
	nonRetryablePatterns := []string{
		"certificate",  // TLS certificate errors
		"x509:",        // x509 certificate errors
		"tls:",         // TLS handshake errors
		"no such host", // DNS resolution failure
		"connection refused",
	}
	for _, pattern := range nonRetryablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return false
		}
	}

	retryablePatterns := []string{
		"timeout",
		"reset by peer",
		"broken pipe",
		"temporary failure",
		"eof", // Unexpected EOF during read (lowercase for comparison)
	}
	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	// Default: don't retry unknown errors to be safe
	return false
}

// RetryStrategy defines how failed requests are retried.
type RetryStrategy interface {
	// ShouldRetry returns true if the request should be retried.
	ShouldRetry(attempt int, err error) bool

	// RetryDelay returns how long to wait before the next attempt.
	RetryDelay(attempt int) time.Duration
}

// RetryStrategyWithError is an optional extension of RetryStrategy that
// allows the retry delay to be influenced by the error.
// If a strategy implements this interface, RetryDelayWithError is called
// instead of RetryDelay, allowing it to use Retry-After headers.
type RetryStrategyWithError interface {
	RetryStrategy
	// RetryDelayWithError returns how long to wait, considering the error.
	// For rate limit errors with Retry-After headers, this can return the
	// server-specified delay.
	RetryDelayWithError(attempt int, err error) time.Duration
}

// ExponentialBackoff implements exponential backoff with optional jitter.
type ExponentialBackoff struct {
	// InitialDelay is the delay before the first retry.
	// Defaults to 1 second if not set.
	InitialDelay time.Duration

	// MaxDelay is the maximum delay between retries.
	// Defaults to 30 seconds if not set.
	MaxDelay time.Duration

	// Multiplier is the factor by which the delay increases.
	// Defaults to 2.0 if not set.
	Multiplier float64

	// Jitter adds randomness to the delay to prevent thundering herd.
	// If true, the delay is multiplied by a random factor between 0.5 and 1.5.
	Jitter bool

	// MaxRetries is the maximum number of retry attempts.
	// Defaults to 3 if not set.
	MaxRetries int
}

// NewExponentialBackoff creates a new exponential backoff strategy with defaults.
func NewExponentialBackoff() *ExponentialBackoff {
	return &ExponentialBackoff{
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
		MaxRetries:   3,
	}
}

// ShouldRetry implements RetryStrategy.ShouldRetry.
func (e *ExponentialBackoff) ShouldRetry(attempt int, err error) bool {
	// Check max retries
	maxRetries := e.MaxRetries
	if maxRetries == 0 {
		maxRetries = 3
	}
	if attempt >= maxRetries {
		return false
	}

	// Check if the error implements RetryableError interface
	if retryableErr, ok := err.(RetryableError); ok {
		return retryableErr.IsRetryable()
	}

	// Only retry transient network errors
	return IsRetryableNetworkError(err)
}

// RetryDelay implements RetryStrategy.RetryDelay.
func (e *ExponentialBackoff) RetryDelay(attempt int) time.Duration {
	return e.calculateDelay(attempt)
}

// RetryDelayWithError implements RetryStrategyWithError.RetryDelayWithError.
// If the error implements RetryAfterError with a RetryAfter value from the server,
// that value is used (capped at MaxDelay).
func (e *ExponentialBackoff) RetryDelayWithError(attempt int, err error) time.Duration {
	// Check if error has a Retry-After hint from the server
	if retryAfterErr, ok := err.(RetryAfterError); ok {
		retryAfter := retryAfterErr.RetryAfter()
		if retryAfter > 0 {
			maxDelay := e.MaxDelay
			if maxDelay == 0 {
				maxDelay = 30 * time.Second
			}
			// Use server's Retry-After, but cap at max delay
			if retryAfter > maxDelay {
				return maxDelay
			}
			return retryAfter
		}
	}

	// Fall back to calculated delay
	return e.calculateDelay(attempt)
}

// calculateDelay computes the exponential backoff delay for an attempt.
func (e *ExponentialBackoff) calculateDelay(attempt int) time.Duration {
	initialDelay := e.InitialDelay
	if initialDelay == 0 {
		initialDelay = 1 * time.Second
	}

	maxDelay := e.MaxDelay
	if maxDelay == 0 {
		maxDelay = 30 * time.Second
	}

	multiplier := e.Multiplier
	if multiplier == 0 {
		multiplier = 2.0
	}

	delay := float64(initialDelay) * math.Pow(multiplier, float64(attempt))
	if delay > float64(maxDelay) {
		delay = float64(maxDelay)
	}

	if e.Jitter {
		// Add jitter: delay * random(0.5, 1.5)
		jitterFactor := 0.5 + rand.Float64()
		delay *= jitterFactor
	}

	return time.Duration(delay)
}

// NoRetry is a retry strategy that never retries.
type NoRetry struct{}

// ShouldRetry implements RetryStrategy.ShouldRetry.
func (n *NoRetry) ShouldRetry(attempt int, err error) bool {
	return false
}

// RetryDelay implements RetryStrategy.RetryDelay.
func (n *NoRetry) RetryDelay(attempt int) time.Duration {
	return 0
}

// FixedDelay is a retry strategy with a fixed delay between retries.
type FixedDelay struct {
	// Delay is the fixed delay between retries.
	Delay time.Duration

	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int
}

// NewFixedDelay creates a new fixed delay retry strategy.
func NewFixedDelay(delay time.Duration, maxRetries int) *FixedDelay {
	return &FixedDelay{
		Delay:      delay,
		MaxRetries: maxRetries,
	}
}

// ShouldRetry implements RetryStrategy.ShouldRetry.
func (f *FixedDelay) ShouldRetry(attempt int, err error) bool {
	if attempt >= f.MaxRetries {
		return false
	}

	// Check if the error implements RetryableError interface
	if retryableErr, ok := err.(RetryableError); ok {
		return retryableErr.IsRetryable()
	}

	// Only retry transient network errors
	return IsRetryableNetworkError(err)
}

// RetryDelay implements RetryStrategy.RetryDelay.
func (f *FixedDelay) RetryDelay(attempt int) time.Duration {
	return f.Delay
}

// LinearBackoff is a retry strategy with linearly increasing delays.
type LinearBackoff struct {
	// InitialDelay is the delay before the first retry.
	InitialDelay time.Duration

	// Increment is added to the delay after each attempt.
	Increment time.Duration

	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration

	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int
}

// NewLinearBackoff creates a new linear backoff retry strategy.
func NewLinearBackoff(initialDelay, increment time.Duration, maxRetries int) *LinearBackoff {
	return &LinearBackoff{
		InitialDelay: initialDelay,
		Increment:    increment,
		MaxDelay:     60 * time.Second,
		MaxRetries:   maxRetries,
	}
}

// ShouldRetry implements RetryStrategy.ShouldRetry.
func (l *LinearBackoff) ShouldRetry(attempt int, err error) bool {
	if attempt >= l.MaxRetries {
		return false
	}

	// Check if the error implements RetryableError interface
	if retryableErr, ok := err.(RetryableError); ok {
		return retryableErr.IsRetryable()
	}

	// Only retry transient network errors
	return IsRetryableNetworkError(err)
}

// RetryDelay implements RetryStrategy.RetryDelay.
func (l *LinearBackoff) RetryDelay(attempt int) time.Duration {
	delay := l.InitialDelay + time.Duration(attempt)*l.Increment
	return min(delay, l.MaxDelay)
}
