// Package http provides HTTP utilities for the Langfuse SDK.
package http

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

// ============================================================================
// Hook Priority
// ============================================================================

// HookPriority determines how hook failures are handled.
type HookPriority int

const (
	// HookPriorityObservational indicates a hook that should not abort requests on failure.
	// Use for logging, metrics, tracing, and other observational concerns.
	// Errors from these hooks are logged but the request continues.
	HookPriorityObservational HookPriority = iota

	// HookPriorityCritical indicates a hook that should abort requests on failure.
	// Use for authentication, authorization, request signing, and other critical concerns.
	// Errors from these hooks cause the request to fail.
	HookPriorityCritical
)

// String returns a string representation of the hook priority.
func (p HookPriority) String() string {
	switch p {
	case HookPriorityObservational:
		return "observational"
	case HookPriorityCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// ============================================================================
// HTTP Hook Interface
// ============================================================================

// HTTPHook allows customizing HTTP request/response handling.
// Hooks are called in order during request processing.
//
// Use hooks for:
//   - Adding custom headers to all requests
//   - Logging request/response details
//   - Collecting custom metrics
//   - Implementing custom retry logic
type HTTPHook interface {
	// BeforeRequest is called before sending the HTTP request.
	// It can modify the request (e.g., add headers) and return an error to abort.
	BeforeRequest(ctx context.Context, req *http.Request) error

	// AfterResponse is called after receiving the HTTP response.
	// It receives the response, duration, and any error from the request.
	AfterResponse(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error)
}

// ============================================================================
// Classified Hook
// ============================================================================

// ClassifiedHook wraps an HTTPHook with priority information.
// This allows different error handling behavior based on the hook's purpose.
type ClassifiedHook struct {
	// Hook is the underlying HTTP hook.
	Hook HTTPHook

	// Priority determines how failures are handled.
	Priority HookPriority

	// Name is used for error messages and logging.
	Name string
}

// ============================================================================
// Hook Function Adapter
// ============================================================================

// HTTPHookFunc is a function adapter for simple hooks.
// It allows creating hooks from functions without implementing the full interface.
type HTTPHookFunc struct {
	Before func(ctx context.Context, req *http.Request) error
	After  func(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error)
}

// BeforeRequest implements HTTPHook.
func (f HTTPHookFunc) BeforeRequest(ctx context.Context, req *http.Request) error {
	if f.Before != nil {
		return f.Before(ctx, req)
	}
	return nil
}

// AfterResponse implements HTTPHook.
func (f HTTPHookFunc) AfterResponse(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error) {
	if f.After != nil {
		f.After(ctx, req, resp, duration, err)
	}
}

// ============================================================================
// Hook Chain
// ============================================================================

// hookChain combines multiple hooks into a single hook.
type hookChain struct {
	hooks []HTTPHook
}

// BeforeRequest calls all hooks in order.
func (c *hookChain) BeforeRequest(ctx context.Context, req *http.Request) error {
	for _, hook := range c.hooks {
		if err := hook.BeforeRequest(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// AfterResponse calls all hooks in reverse order (like a defer stack).
func (c *hookChain) AfterResponse(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error) {
	// Call in reverse order so hooks "wrap" like middleware
	for i := len(c.hooks) - 1; i >= 0; i-- {
		c.hooks[i].AfterResponse(ctx, req, resp, duration, err)
	}
}

// CombineHooks combines multiple hooks into a single hook.
// If there are no hooks, returns nil. If there is one hook, returns it directly.
func CombineHooks(hooks []HTTPHook) HTTPHook {
	if len(hooks) == 0 {
		return nil
	}
	if len(hooks) == 1 {
		return hooks[0]
	}
	return &hookChain{hooks: hooks}
}

// ============================================================================
// Classified Hook Chain
// ============================================================================

// Logger is a minimal logging interface for hooks.
type Logger interface {
	Printf(format string, v ...any)
}

// Metrics is an interface for SDK telemetry in hooks.
type Metrics interface {
	IncrementCounter(name string, value int64)
}

// ClassifiedHookChain manages multiple hooks with different priorities.
// It provides priority-aware error handling where observational hooks
// don't abort requests on failure.
type ClassifiedHookChain struct {
	hooks   []ClassifiedHook
	logger  Logger
	metrics Metrics
}

// NewClassifiedHookChain creates a new classified hook chain.
func NewClassifiedHookChain(logger Logger, metrics Metrics) *ClassifiedHookChain {
	return &ClassifiedHookChain{
		hooks:   make([]ClassifiedHook, 0),
		logger:  logger,
		metrics: metrics,
	}
}

// Add adds a hook with the specified priority.
func (c *ClassifiedHookChain) Add(name string, hook HTTPHook, priority HookPriority) {
	c.hooks = append(c.hooks, ClassifiedHook{
		Hook:     hook,
		Priority: priority,
		Name:     name,
	})
}

// AddClassified adds a pre-classified hook.
func (c *ClassifiedHookChain) AddClassified(ch ClassifiedHook) {
	c.hooks = append(c.hooks, ch)
}

// Len returns the number of hooks in the chain.
func (c *ClassifiedHookChain) Len() int {
	return len(c.hooks)
}

// BeforeRequest calls all hooks' BeforeRequest methods.
// Critical hook failures abort the request, observational failures are logged.
func (c *ClassifiedHookChain) BeforeRequest(ctx context.Context, req *http.Request) error {
	for _, ch := range c.hooks {
		if err := c.callBeforeRequest(ctx, req, ch); err != nil {
			return err
		}
	}
	return nil
}

// callBeforeRequest calls a single hook's BeforeRequest with priority-aware error handling.
func (c *ClassifiedHookChain) callBeforeRequest(ctx context.Context, req *http.Request, ch ClassifiedHook) error {
	// Recover from panics in hooks
	defer func() {
		if r := recover(); r != nil {
			if c.logger != nil {
				c.logger.Printf("langfuse: hook %q panicked in BeforeRequest: %v", ch.Name, r)
			}
			if c.metrics != nil {
				c.metrics.IncrementCounter("langfuse.hooks.panics", 1)
			}
		}
	}()

	err := ch.Hook.BeforeRequest(ctx, req)
	if err == nil {
		return nil
	}

	// Track hook failures
	if c.metrics != nil {
		c.metrics.IncrementCounter("langfuse.hooks.failures", 1)
		c.metrics.IncrementCounter(fmt.Sprintf("langfuse.hooks.failures.%s", ch.Name), 1)
	}

	switch ch.Priority {
	case HookPriorityCritical:
		return fmt.Errorf("langfuse: critical hook %q failed: %w", ch.Name, err)

	case HookPriorityObservational:
		if c.logger != nil {
			c.logger.Printf("langfuse: observational hook %q failed (continuing): %v", ch.Name, err)
		}
		return nil

	default:
		// Unknown priority, treat as critical for safety
		return fmt.Errorf("langfuse: hook %q failed: %w", ch.Name, err)
	}
}

// AfterResponse calls all hooks' AfterResponse methods.
// Errors and panics are logged but never returned (response already received).
func (c *ClassifiedHookChain) AfterResponse(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error) {
	// Call in reverse order so hooks "wrap" like middleware
	for i := len(c.hooks) - 1; i >= 0; i-- {
		c.callAfterResponse(ctx, req, resp, duration, err, c.hooks[i])
	}
}

// callAfterResponse calls a single hook's AfterResponse with panic recovery.
func (c *ClassifiedHookChain) callAfterResponse(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, requestErr error, ch ClassifiedHook) {
	defer func() {
		if r := recover(); r != nil {
			if c.logger != nil {
				c.logger.Printf("langfuse: hook %q panicked in AfterResponse: %v", ch.Name, r)
			}
			if c.metrics != nil {
				c.metrics.IncrementCounter("langfuse.hooks.panics", 1)
			}
		}
	}()

	ch.Hook.AfterResponse(ctx, req, resp, duration, requestErr)
}

// ============================================================================
// Predefined Hooks
// ============================================================================

// HeaderHook creates a hook that adds custom headers to all requests.
func HeaderHook(headers map[string]string) HTTPHook {
	return HTTPHookFunc{
		Before: func(ctx context.Context, req *http.Request) error {
			for k, v := range headers {
				req.Header.Set(k, v)
			}
			return nil
		},
	}
}

// DynamicHeaderHook creates a hook that adds headers from a function.
// The function is called for each request, allowing dynamic header values.
func DynamicHeaderHook(fn func(ctx context.Context) map[string]string) HTTPHook {
	return HTTPHookFunc{
		Before: func(ctx context.Context, req *http.Request) error {
			headers := fn(ctx)
			for k, v := range headers {
				req.Header.Set(k, v)
			}
			return nil
		},
	}
}

// LoggingHook creates a hook that logs request and response information.
func LoggingHook(logger Logger) HTTPHook {
	return HTTPHookFunc{
		Before: func(ctx context.Context, req *http.Request) error {
			logger.Printf("langfuse: %s %s", req.Method, req.URL.Path)
			return nil
		},
		After: func(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error) {
			if err != nil {
				logger.Printf("langfuse: %s %s failed after %v: %v", req.Method, req.URL.Path, duration, err)
			} else if resp != nil {
				logger.Printf("langfuse: %s %s completed in %v with status %d", req.Method, req.URL.Path, duration, resp.StatusCode)
			}
		},
	}
}

// MetricsHook creates a hook that records request metrics.
// It uses the Metrics interface if provided, or creates a no-op hook.
//
// Metrics recorded:
//   - langfuse.http.requests (counter): Total request count
//   - langfuse.http.duration (timing): Request duration
//   - langfuse.http.errors (counter): Error count
//   - langfuse.http.status.{code} (counter): Per-status-code count
func MetricsHook(m MetricsRecorder) HTTPHook {
	if m == nil {
		return HTTPHookFunc{} // No-op
	}
	return HTTPHookFunc{
		After: func(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error) {
			m.IncrementCounter("langfuse.http.requests", 1)
			m.RecordDuration("langfuse.http.duration", duration)

			if err != nil {
				m.IncrementCounter("langfuse.http.errors", 1)
			}
			if resp != nil {
				statusKey := "langfuse.http.status." + http.StatusText(resp.StatusCode)
				m.IncrementCounter(statusKey, 1)
			}
		},
	}
}

// MetricsRecorder is the interface for recording metrics.
type MetricsRecorder interface {
	IncrementCounter(name string, value int64)
	RecordDuration(name string, duration time.Duration)
}

// DebugHook creates a hook that logs detailed request/response information.
// Only use this in development as it may log sensitive data.
func DebugHook(logger Logger) HTTPHook {
	return HTTPHookFunc{
		Before: func(ctx context.Context, req *http.Request) error {
			logger.Printf("langfuse: DEBUG request: %s %s", req.Method, req.URL.String())
			for k, v := range req.Header {
				if k != "Authorization" { // Don't log auth header
					logger.Printf("langfuse: DEBUG header: %s: %v", k, v)
				}
			}
			return nil
		},
		After: func(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error) {
			if err != nil {
				logger.Printf("langfuse: DEBUG response error: %v (duration: %v)", err, duration)
			} else if resp != nil {
				logger.Printf("langfuse: DEBUG response: status=%d, duration=%v", resp.StatusCode, duration)
				for k, v := range resp.Header {
					logger.Printf("langfuse: DEBUG response header: %s: %v", k, v)
				}
			}
		},
	}
}

// ============================================================================
// Classified Hook Constructors
// ============================================================================

// ObservationalLoggingHook creates a logging hook that won't abort requests on failure.
func ObservationalLoggingHook(logger Logger) ClassifiedHook {
	return ClassifiedHook{
		Hook:     LoggingHook(logger),
		Priority: HookPriorityObservational,
		Name:     "logging",
	}
}

// ObservationalMetricsHook creates a metrics hook that won't abort requests on failure.
func ObservationalMetricsHook(m MetricsRecorder) ClassifiedHook {
	return ClassifiedHook{
		Hook:     MetricsHook(m),
		Priority: HookPriorityObservational,
		Name:     "metrics",
	}
}

// ObservationalDebugHook creates a debug hook that won't abort requests on failure.
func ObservationalDebugHook(logger Logger) ClassifiedHook {
	return ClassifiedHook{
		Hook:     DebugHook(logger),
		Priority: HookPriorityObservational,
		Name:     "debug",
	}
}

// CriticalHeaderHook creates a header hook that aborts requests on failure.
// Use this when headers are required for the request to succeed (e.g., auth headers).
func CriticalHeaderHook(name string, headers map[string]string) ClassifiedHook {
	return ClassifiedHook{
		Hook:     HeaderHook(headers),
		Priority: HookPriorityCritical,
		Name:     name,
	}
}

// CriticalAuthHook creates an authentication hook that aborts requests on failure.
// The authFunc should add authentication headers or return an error.
func CriticalAuthHook(authFunc func(*http.Request) error) ClassifiedHook {
	return ClassifiedHook{
		Hook: HTTPHookFunc{
			Before: func(ctx context.Context, req *http.Request) error {
				return authFunc(req)
			},
		},
		Priority: HookPriorityCritical,
		Name:     "auth",
	}
}

// CriticalValidationHook creates a validation hook that aborts requests on failure.
// The validateFunc should validate the request and return an error if invalid.
func CriticalValidationHook(name string, validateFunc func(*http.Request) error) ClassifiedHook {
	return ClassifiedHook{
		Hook: HTTPHookFunc{
			Before: func(ctx context.Context, req *http.Request) error {
				return validateFunc(req)
			},
		},
		Priority: HookPriorityCritical,
		Name:     name,
	}
}

// NewClassifiedHook creates a ClassifiedHook with the given parameters.
// This is a convenience function for creating custom classified hooks.
func NewClassifiedHook(name string, hook HTTPHook, priority HookPriority) ClassifiedHook {
	return ClassifiedHook{
		Hook:     hook,
		Priority: priority,
		Name:     name,
	}
}
