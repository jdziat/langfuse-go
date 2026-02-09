package errors

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Logger interface for logging within the errors package.
// This is a minimal interface that avoids circular dependencies.
type Logger interface {
	Printf(format string, v ...any)
}

// Metrics interface for recording metrics within the errors package.
// This is a minimal interface that avoids circular dependencies.
type Metrics interface {
	IncrementCounter(name string, value int64)
}

// AsyncErrorOperation identifies the type of async operation that failed.
type AsyncErrorOperation string

// Async error operations.
const (
	AsyncOpBatchSend AsyncErrorOperation = "batch_send"
	AsyncOpFlush     AsyncErrorOperation = "flush"
	AsyncOpHook      AsyncErrorOperation = "hook"
	AsyncOpShutdown  AsyncErrorOperation = "shutdown"
	AsyncOpQueue     AsyncErrorOperation = "queue"
	AsyncOpInternal  AsyncErrorOperation = "internal"
)

// AsyncError represents an error that occurred in background processing.
// It provides structured information about what went wrong and when.
type AsyncError struct {
	// Time is when the error occurred.
	Time time.Time

	// Operation identifies the async operation that failed.
	Operation AsyncErrorOperation

	// EventIDs contains IDs of affected events, if known.
	EventIDs []string

	// Err is the underlying error.
	Err error

	// Retryable indicates if the operation can be retried.
	Retryable bool

	// Context contains additional context about the error.
	Context map[string]any
}

// Error implements the error interface.
func (e *AsyncError) Error() string {
	if len(e.EventIDs) > 0 {
		return fmt.Sprintf("langfuse async error [%s] at %s (%d events affected): %v",
			e.Operation, e.Time.Format(time.RFC3339), len(e.EventIDs), e.Err)
	}
	return fmt.Sprintf("langfuse async error [%s] at %s: %v",
		e.Operation, e.Time.Format(time.RFC3339), e.Err)
}

// Unwrap returns the underlying error for error chain support.
func (e *AsyncError) Unwrap() error {
	return e.Err
}

// NewAsyncError creates a new async error.
func NewAsyncError(op AsyncErrorOperation, err error) *AsyncError {
	return &AsyncError{
		Time:      time.Now(),
		Operation: op,
		Err:       err,
	}
}

// WithEventIDs adds affected event IDs to the error.
func (e *AsyncError) WithEventIDs(ids ...string) *AsyncError {
	e.EventIDs = ids
	return e
}

// WithRetryable marks the error as retryable or not.
func (e *AsyncError) WithRetryable(retryable bool) *AsyncError {
	e.Retryable = retryable
	return e
}

// WithContext adds context to the error.
func (e *AsyncError) WithContext(key string, value any) *AsyncError {
	if e.Context == nil {
		e.Context = make(map[string]any)
	}
	e.Context[key] = value
	return e
}

// AsyncErrorHandler provides structured async error handling.
// It buffers errors in a channel and provides callbacks for notification.
type AsyncErrorHandler struct {
	// Errors is a buffered channel for receiving async errors.
	// Consumers can read from this channel to process errors programmatically.
	Errors chan *AsyncError

	// Configuration
	bufferSize int
	metrics    Metrics
	logger     Logger

	// Callbacks
	mu         sync.RWMutex
	onError    func(*AsyncError)
	onOverflow func(dropped int)

	// Statistics
	totalErrors  atomic.Int64
	droppedCount atomic.Int64
	errorsByOp   sync.Map // map[AsyncErrorOperation]*atomic.Int64
}

// AsyncErrorConfig configures the AsyncErrorHandler.
type AsyncErrorConfig struct {
	// BufferSize is the size of the error channel buffer.
	// Default: 100
	BufferSize int

	// Metrics is used for error metrics.
	Metrics Metrics

	// Logger is used for logging dropped errors.
	Logger Logger

	// OnError is called for each error (before buffering).
	OnError func(*AsyncError)

	// OnOverflow is called when errors are dropped due to full buffer.
	OnOverflow func(dropped int)
}

// NewAsyncErrorHandler creates a new async error handler.
func NewAsyncErrorHandler(cfg *AsyncErrorConfig) *AsyncErrorHandler {
	if cfg == nil {
		cfg = &AsyncErrorConfig{}
	}

	bufferSize := cfg.BufferSize
	if bufferSize <= 0 {
		bufferSize = 100
	}

	h := &AsyncErrorHandler{
		Errors:     make(chan *AsyncError, bufferSize),
		bufferSize: bufferSize,
		metrics:    cfg.Metrics,
		logger:     cfg.Logger,
		onError:    cfg.OnError,
		onOverflow: cfg.OnOverflow,
	}

	return h
}

// Handle processes an async error.
// It sends the error to the channel, invokes callbacks, and updates metrics.
func (h *AsyncErrorHandler) Handle(err *AsyncError) {
	if err == nil {
		return
	}

	// Update statistics
	h.totalErrors.Add(1)
	h.incrementOpCounter(err.Operation)

	// Try to send to channel
	select {
	case h.Errors <- err:
		// Sent successfully
	default:
		// Channel full - track dropped error
		dropped := h.droppedCount.Add(1)

		if h.metrics != nil {
			h.metrics.IncrementCounter("langfuse.async_errors.dropped", 1)
		}

		if h.logger != nil {
			h.logger.Printf("langfuse: async error dropped (buffer full, %d total dropped): %v", dropped, err)
		}

		// Call overflow callback
		h.mu.RLock()
		onOverflow := h.onOverflow
		h.mu.RUnlock()

		if onOverflow != nil {
			onOverflow(int(dropped))
		}
	}

	// Call error callback
	h.mu.RLock()
	onError := h.onError
	h.mu.RUnlock()

	if onError != nil {
		onError(err)
	}

	// Update metrics
	if h.metrics != nil {
		h.metrics.IncrementCounter("langfuse.async_errors.total", 1)
		h.metrics.IncrementCounter(
			fmt.Sprintf("langfuse.async_errors.%s", err.Operation), 1)

		if err.Retryable {
			h.metrics.IncrementCounter("langfuse.async_errors.retryable", 1)
		}
	}
}

// incrementOpCounter increments the counter for a specific operation.
func (h *AsyncErrorHandler) incrementOpCounter(op AsyncErrorOperation) {
	counter, _ := h.errorsByOp.LoadOrStore(op, &atomic.Int64{})
	counter.(*atomic.Int64).Add(1)
}

// SetCallback sets the error callback.
// This replaces any previously set callback.
func (h *AsyncErrorHandler) SetCallback(fn func(*AsyncError)) {
	h.mu.Lock()
	h.onError = fn
	h.mu.Unlock()
}

// SetOverflowCallback sets the overflow callback.
// This is called when errors are dropped due to a full buffer.
func (h *AsyncErrorHandler) SetOverflowCallback(fn func(dropped int)) {
	h.mu.Lock()
	h.onOverflow = fn
	h.mu.Unlock()
}

// DroppedCount returns the total number of dropped errors.
func (h *AsyncErrorHandler) DroppedCount() int64 {
	return h.droppedCount.Load()
}

// TotalErrors returns the total number of errors handled.
func (h *AsyncErrorHandler) TotalErrors() int64 {
	return h.totalErrors.Load()
}

// ErrorsByOperation returns the error count for a specific operation.
func (h *AsyncErrorHandler) ErrorsByOperation(op AsyncErrorOperation) int64 {
	counter, ok := h.errorsByOp.Load(op)
	if !ok {
		return 0
	}
	return counter.(*atomic.Int64).Load()
}

// Pending returns the number of errors waiting in the buffer.
func (h *AsyncErrorHandler) Pending() int {
	return len(h.Errors)
}

// Drain returns all pending errors from the channel without blocking.
func (h *AsyncErrorHandler) Drain() []*AsyncError {
	var errors []*AsyncError
	for {
		select {
		case err := <-h.Errors:
			errors = append(errors, err)
		default:
			return errors
		}
	}
}

// Close closes the error channel.
// Call this during shutdown after all error producers have stopped.
func (h *AsyncErrorHandler) Close() {
	close(h.Errors)
}

// AsyncErrorStats contains statistics about async error handling.
type AsyncErrorStats struct {
	TotalErrors  int64
	DroppedCount int64
	Pending      int
	BufferSize   int
}

// Stats returns current error handling statistics.
func (h *AsyncErrorHandler) Stats() AsyncErrorStats {
	return AsyncErrorStats{
		TotalErrors:  h.totalErrors.Load(),
		DroppedCount: h.droppedCount.Load(),
		Pending:      len(h.Errors),
		BufferSize:   h.bufferSize,
	}
}

// WrapAsyncError wraps an error in an AsyncError if it isn't already.
func WrapAsyncError(op AsyncErrorOperation, err error) *AsyncError {
	if err == nil {
		return nil
	}

	// If already an AsyncError, return as-is
	if asyncErr, ok := err.(*AsyncError); ok {
		return asyncErr
	}

	return NewAsyncError(op, err)
}
