package http

import (
	"errors"
	"sync"
	"time"
)

// Circuit breaker states.
const (
	// CircuitClosed allows requests to pass through normally.
	CircuitClosed CircuitState = iota
	// CircuitOpen blocks all requests immediately.
	CircuitOpen
	// CircuitHalfOpen allows a limited number of requests to test if the service recovered.
	CircuitHalfOpen
)

// CircuitState represents the state of a circuit breaker.
type CircuitState int

// String returns the string representation of the circuit state.
func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when the circuit breaker is open and blocking requests.
var ErrCircuitOpen = errors.New("langfuse: circuit breaker is open")

// CircuitBreakerConfig configures the circuit breaker behavior.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit.
	// Default: 5
	FailureThreshold int

	// SuccessThreshold is the number of consecutive successes in half-open state
	// before closing the circuit.
	// Default: 2
	SuccessThreshold int

	// Timeout is the duration the circuit stays open before transitioning to half-open.
	// Default: 30 seconds
	Timeout time.Duration

	// HalfOpenMaxRequests is the maximum number of requests allowed in half-open state.
	// Default: 1
	HalfOpenMaxRequests int

	// OnStateChange is called when the circuit state changes.
	OnStateChange func(from, to CircuitState)

	// IsFailure determines if an error should count as a failure.
	// If nil, all non-nil errors are considered failures.
	IsFailure func(err error) bool
}

// DefaultCircuitBreakerConfig returns a CircuitBreakerConfig with sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold:    5,
		SuccessThreshold:    2,
		Timeout:             30 * time.Second,
		HalfOpenMaxRequests: 1,
	}
}

// CircuitBreaker implements the circuit breaker pattern for fault tolerance.
// It prevents cascading failures by failing fast when a service is unhealthy.
//
// States:
//   - Closed: Normal operation, requests pass through
//   - Open: Service is unhealthy, requests fail immediately with ErrCircuitOpen
//   - Half-Open: Testing if service recovered, limited requests allowed
//
// Example:
//
//	cb := http.NewCircuitBreaker(http.CircuitBreakerConfig{
//	    FailureThreshold: 5,
//	    Timeout:          30 * time.Second,
//	    OnStateChange: func(from, to http.CircuitState) {
//	        log.Printf("Circuit breaker: %s -> %s", from, to)
//	    },
//	})
//
//	err := cb.Execute(func() error {
//	    return client.Flush(ctx)
//	})
type CircuitBreaker struct {
	config CircuitBreakerConfig

	mu                sync.RWMutex
	state             CircuitState
	failures          int
	successes         int
	lastFailure       time.Time
	halfOpenRequests  int
	consecutiveErrors int
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2
	}
	if config.Timeout <= 0 {
		config.Timeout = 30 * time.Second
	}
	if config.HalfOpenMaxRequests <= 0 {
		config.HalfOpenMaxRequests = 1
	}
	// Recovery sanity: HalfOpenMaxRequests bounds the number of probes IN FLIGHT,
	// not the total number of probes admitted while half-open. Record() releases a
	// probe's slot when it completes (see Record), so even HalfOpenMaxRequests == 1
	// admits successive probes until SuccessThreshold successes close the circuit.
	// Recovery is therefore always possible for any positive configuration; no
	// further clamping is required.

	return &CircuitBreaker{
		config: config,
		state:  CircuitClosed,
	}
}

// State returns the current state of the circuit breaker.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.currentState()
}

// currentState returns the current state, potentially transitioning from open to half-open.
// Must be called with at least a read lock held.
func (cb *CircuitBreaker) currentState() CircuitState {
	if cb.state == CircuitOpen {
		if time.Since(cb.lastFailure) >= cb.config.Timeout {
			return CircuitHalfOpen
		}
	}
	return cb.state
}

// Execute runs the given function if the circuit allows it.
// Returns ErrCircuitOpen if the circuit is open.
func (cb *CircuitBreaker) Execute(fn func() error) error {
	if !cb.Allow() {
		return ErrCircuitOpen
	}

	err := fn()
	cb.Record(err)
	return err
}

// Allow checks if a request should be allowed through.
// Returns true if the request can proceed, false if blocked.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	state := cb.currentState()

	switch state {
	case CircuitClosed:
		return true

	case CircuitOpen:
		return false

	case CircuitHalfOpen:
		// Transition from open to half-open if the timeout has elapsed.
		// currentState() reports HalfOpen while cb.state is still Open, so this
		// branch performs the real state change exactly once.
		if cb.state == CircuitOpen {
			cb.setState(CircuitHalfOpen)
		}

		// halfOpenRequests tracks probes currently IN FLIGHT. A probe is admitted
		// while fewer than HalfOpenMaxRequests are outstanding; Record() decrements
		// the counter when a probe completes, freeing the slot for the next probe.
		if cb.halfOpenRequests < cb.config.HalfOpenMaxRequests {
			cb.halfOpenRequests++
			return true
		}
		return false
	}

	return false
}

// Record records the result of a request.
// Pass nil for success, or an error for failure.
func (cb *CircuitBreaker) Record(err error) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	isFailure := err != nil
	if cb.config.IsFailure != nil && err != nil {
		isFailure = cb.config.IsFailure(err)
	}

	state := cb.currentState()

	switch state {
	case CircuitClosed:
		if isFailure {
			cb.failures++
			cb.consecutiveErrors++
			cb.lastFailure = time.Now()

			if cb.consecutiveErrors >= cb.config.FailureThreshold {
				cb.setState(CircuitOpen)
			}
		} else {
			cb.consecutiveErrors = 0
		}

	case CircuitHalfOpen:
		// A half-open probe has completed; release its in-flight slot so the
		// next Allow() can admit another probe. setState resets the counter on
		// transition, so only decrement when staying in half-open.
		if isFailure {
			// Failed in half-open, go back to open.
			cb.lastFailure = time.Now()
			cb.setState(CircuitOpen)
		} else {
			cb.successes++
			if cb.successes >= cb.config.SuccessThreshold {
				// Enough successes, close the circuit.
				cb.setState(CircuitClosed)
			} else if cb.halfOpenRequests > 0 {
				cb.halfOpenRequests--
			}
		}
	}
}

// Reset resets the circuit breaker to closed state.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.setState(CircuitClosed)
	cb.failures = 0
	cb.successes = 0
	cb.consecutiveErrors = 0
	cb.halfOpenRequests = 0
}

// Failures returns the total number of recorded failures.
func (cb *CircuitBreaker) Failures() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failures
}

// ConsecutiveErrors returns the current count of consecutive errors.
func (cb *CircuitBreaker) ConsecutiveErrors() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.consecutiveErrors
}

// setState changes the state and calls the OnStateChange callback if configured.
// Must be called with the lock held.
func (cb *CircuitBreaker) setState(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState

	// Reset counters on state change
	switch newState {
	case CircuitClosed:
		cb.failures = 0
		cb.successes = 0
		cb.consecutiveErrors = 0
		cb.halfOpenRequests = 0
	case CircuitHalfOpen:
		cb.halfOpenRequests = 0
		cb.successes = 0
	}

	if cb.config.OnStateChange != nil {
		// Call callback without lock to prevent deadlocks
		go cb.config.OnStateChange(oldState, newState)
	}
}

// CircuitBreakerOption configures a circuit breaker.
type CircuitBreakerOption func(*CircuitBreakerConfig)

// WithFailureThreshold sets the failure threshold.
func WithFailureThreshold(n int) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.FailureThreshold = n
	}
}

// WithSuccessThreshold sets the success threshold for half-open state.
func WithSuccessThreshold(n int) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.SuccessThreshold = n
	}
}

// WithCircuitTimeout sets the timeout before transitioning from open to half-open.
func WithCircuitTimeout(d time.Duration) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.Timeout = d
	}
}

// WithHalfOpenMaxRequests sets the max requests allowed in half-open state.
func WithHalfOpenMaxRequests(n int) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.HalfOpenMaxRequests = n
	}
}

// WithStateChangeCallback sets the callback for state changes.
func WithStateChangeCallback(fn func(from, to CircuitState)) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.OnStateChange = fn
	}
}

// WithFailureChecker sets a custom function to determine if an error is a failure.
func WithFailureChecker(fn func(err error) bool) CircuitBreakerOption {
	return func(c *CircuitBreakerConfig) {
		c.IsFailure = fn
	}
}

// NewCircuitBreakerWithOptions creates a circuit breaker with functional options.
func NewCircuitBreakerWithOptions(opts ...CircuitBreakerOption) *CircuitBreaker {
	config := DefaultCircuitBreakerConfig()
	for _, opt := range opts {
		opt(&config)
	}
	return NewCircuitBreaker(config)
}
