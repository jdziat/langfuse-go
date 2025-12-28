package langfuse

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ErrorCode represents a category of error for metrics and logging.
type ErrorCode string

// Error codes for categorization.
const (
	ErrCodeConfig       ErrorCode = "CONFIG"       // Configuration errors
	ErrCodeValidation   ErrorCode = "VALIDATION"   // Request validation errors
	ErrCodeNetwork      ErrorCode = "NETWORK"      // Network/connection errors
	ErrCodeAPI          ErrorCode = "API"          // API response errors
	ErrCodeAuth         ErrorCode = "AUTH"         // Authentication/authorization errors
	ErrCodeRateLimit    ErrorCode = "RATE_LIMIT"   // Rate limiting errors
	ErrCodeTimeout      ErrorCode = "TIMEOUT"      // Timeout errors
	ErrCodeInternal     ErrorCode = "INTERNAL"     // Internal SDK errors
	ErrCodeShutdown     ErrorCode = "SHUTDOWN"     // Shutdown-related errors
	ErrCodeBackpressure ErrorCode = "BACKPRESSURE" // Backpressure/queue full errors
)

// LangfuseError is the common interface for all SDK errors.
// Use this interface to handle errors generically while still accessing
// error-specific information.
//
// Example:
//
//	var langfuseErr LangfuseError
//	if errors.As(err, &langfuseErr) {
//	    if langfuseErr.IsRetryable() {
//	        // Retry the operation
//	    }
//	    log.Printf("Error code: %s, request ID: %s", langfuseErr.Code(), langfuseErr.GetRequestID())
//	}
type LangfuseError interface {
	error

	// Code returns a machine-readable error code for categorization.
	Code() ErrorCode

	// IsRetryable returns true if the operation can be retried.
	IsRetryable() bool

	// GetRequestID returns the server request ID, if available.
	// Returns empty string if not applicable.
	GetRequestID() string
}

// IsRetryable returns true if the error represents a retryable condition.
// This works with any error type in the SDK.
//
// Retryable conditions include:
//   - Rate limiting (429)
//   - Server errors (5xx)
//   - Network timeouts
//   - Circuit breaker open (may close soon)
//
// Example:
//
//	if langfuse.IsRetryable(err) {
//	    time.Sleep(langfuse.RetryAfter(err))
//	    // Retry the operation
//	}
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Check if error implements LangfuseError
	var langfuseErr LangfuseError
	if errors.As(err, &langfuseErr) {
		return langfuseErr.IsRetryable()
	}

	// Check specific error types
	if apiErr, ok := AsAPIError(err); ok {
		return apiErr.IsRetryable()
	}

	// Network errors are generally retryable
	if errors.Is(err, ErrCircuitOpen) {
		return true
	}

	return false
}

// Sentinel errors for configuration validation.
var (
	ErrMissingPublicKey = errors.New("langfuse: public key is required")
	ErrMissingSecretKey = errors.New("langfuse: secret key is required")
	ErrMissingBaseURL   = errors.New("langfuse: base URL is required")
	ErrInvalidConfig    = errors.New("langfuse: invalid configuration")
	ErrClientClosed     = errors.New("langfuse: client is closed")
	ErrNilRequest       = errors.New("langfuse: request cannot be nil")
)

// Additional sentinel errors for common scenarios.
// Note: ErrCircuitOpen is defined in circuitbreaker.go.
var (
	ErrPromptNotFound   = errors.New("langfuse: prompt not found")
	ErrDatasetNotFound  = errors.New("langfuse: dataset not found")
	ErrTraceNotFound    = errors.New("langfuse: trace not found")
	ErrEmptyBatch       = errors.New("langfuse: batch is empty")
	ErrBatchTooLarge    = errors.New("langfuse: batch exceeds maximum size")
	ErrContextCancelled = errors.New("langfuse: context was cancelled")
	ErrShutdownTimeout  = errors.New("langfuse: shutdown timed out")
)

// Sentinel APIError values for use with errors.Is().
// These match on status code only.
var (
	ErrNotFound     = &APIError{StatusCode: 404}
	ErrUnauthorized = &APIError{StatusCode: 401}
	ErrForbidden    = &APIError{StatusCode: 403}
	ErrRateLimited  = &APIError{StatusCode: 429}
)

// APIError represents an error response from the Langfuse API.
// It supports error wrapping via Unwrap() and comparison via Is().
type APIError struct {
	StatusCode   int           `json:"statusCode"`
	Message      string        `json:"message"`
	ErrorMessage string        `json:"error"`
	RequestID    string        `json:"-"` // Request ID for debugging
	RetryAfter   time.Duration `json:"-"` // From Retry-After header
	Err          error         `json:"-"` // Underlying error for wrapping
}

// Error implements the error interface.
func (e *APIError) Error() string {
	var msg string
	if e.Message != "" {
		msg = e.Message
	} else if e.ErrorMessage != "" {
		msg = e.ErrorMessage
	}

	if msg != "" {
		if e.RequestID != "" {
			return fmt.Sprintf("langfuse: API error (status %d, request %s): %s", e.StatusCode, e.RequestID, msg)
		}
		return fmt.Sprintf("langfuse: API error (status %d): %s", e.StatusCode, msg)
	}

	if e.RequestID != "" {
		return fmt.Sprintf("langfuse: API error (status %d, request %s)", e.StatusCode, e.RequestID)
	}
	return fmt.Sprintf("langfuse: API error (status %d)", e.StatusCode)
}

// String returns a compact string representation for debugging.
func (e *APIError) String() string {
	msg := e.Message
	if msg == "" {
		msg = e.ErrorMessage
	}
	return fmt.Sprintf("APIError{Status: %d, Message: %q}", e.StatusCode, msg)
}

// Unwrap returns the underlying error for error chain support.
// This allows using errors.Unwrap() and errors.Is() with wrapped errors.
func (e *APIError) Unwrap() error {
	return e.Err
}

// Is implements error comparison for errors.Is().
// It matches on status code, allowing comparisons like:
//
//	if errors.Is(err, langfuse.ErrRateLimited) { ... }
func (e *APIError) Is(target error) bool {
	t, ok := target.(*APIError)
	if !ok {
		return false
	}
	// Match on status code for sentinel error comparison
	return e.StatusCode == t.StatusCode
}

// WithError wraps an underlying error in the APIError.
func (e *APIError) WithError(err error) *APIError {
	e.Err = err
	return e
}

// IsNotFound returns true if the error is a 404 Not Found error.
func (e *APIError) IsNotFound() bool {
	return e.StatusCode == 404
}

// IsUnauthorized returns true if the error is a 401 Unauthorized error.
func (e *APIError) IsUnauthorized() bool {
	return e.StatusCode == 401
}

// IsForbidden returns true if the error is a 403 Forbidden error.
func (e *APIError) IsForbidden() bool {
	return e.StatusCode == 403
}

// IsRateLimited returns true if the error is a 429 Too Many Requests error.
func (e *APIError) IsRateLimited() bool {
	return e.StatusCode == 429
}

// IsServerError returns true if the error is a 5xx server error.
func (e *APIError) IsServerError() bool {
	return e.StatusCode >= 500 && e.StatusCode < 600
}

// IsRetryable returns true if the request should be retried.
func (e *APIError) IsRetryable() bool {
	return e.IsRateLimited() || e.IsServerError()
}

// Code returns the error code for the API error.
// Implements the LangfuseError interface.
func (e *APIError) Code() ErrorCode {
	switch {
	case e.IsUnauthorized(), e.IsForbidden():
		return ErrCodeAuth
	case e.IsRateLimited():
		return ErrCodeRateLimit
	default:
		return ErrCodeAPI
	}
}

// GetRequestID returns the request ID for the API error.
// Implements the LangfuseError interface.
func (e *APIError) GetRequestID() string {
	return e.RequestID
}

// Ensure APIError implements LangfuseError.
var _ LangfuseError = (*APIError)(nil)

// IngestionError represents an error that occurred during batch ingestion.
type IngestionError struct {
	ID           string `json:"id"`
	Status       int    `json:"status"`
	Message      string `json:"message"`
	ErrorMessage string `json:"error"`
}

// Error implements the error interface.
func (e *IngestionError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("langfuse: ingestion error for %s (status %d): %s", e.ID, e.Status, e.Message)
	}
	if e.ErrorMessage != "" {
		return fmt.Sprintf("langfuse: ingestion error for %s (status %d): %s", e.ID, e.Status, e.ErrorMessage)
	}
	return fmt.Sprintf("langfuse: ingestion error for %s (status %d)", e.ID, e.Status)
}

// IngestionResult represents the result of a batch ingestion request.
type IngestionResult struct {
	Successes []IngestionSuccess `json:"successes"`
	Errors    []IngestionError   `json:"errors"`
}

// IngestionSuccess represents a successful ingestion event.
type IngestionSuccess struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
}

// HasErrors returns true if the ingestion result contains any errors.
func (r *IngestionResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// FirstError returns the first ingestion error, or nil if there are none.
func (r *IngestionResult) FirstError() error {
	if len(r.Errors) == 0 {
		return nil
	}
	return &r.Errors[0]
}

// ValidationError represents a validation error for a request.
type ValidationError struct {
	Field   string
	Message string
	Err     error // Underlying error for wrapping
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	return fmt.Sprintf("langfuse: validation error for field %q: %s", e.Field, e.Message)
}

// Unwrap returns the underlying error for error chain support.
func (e *ValidationError) Unwrap() error {
	return e.Err
}

// Code returns the error code for the validation error.
// Implements the LangfuseError interface.
func (e *ValidationError) Code() ErrorCode {
	return ErrCodeValidation
}

// IsRetryable returns false for validation errors (they should be fixed, not retried).
// Implements the LangfuseError interface.
func (e *ValidationError) IsRetryable() bool {
	return false
}

// GetRequestID returns an empty string (validation errors don't have request IDs).
// Implements the LangfuseError interface.
func (e *ValidationError) GetRequestID() string {
	return ""
}

// Ensure ValidationError implements LangfuseError.
var _ LangfuseError = (*ValidationError)(nil)

// NewValidationError creates a new validation error.
func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

// NewValidationErrorWithCause creates a new validation error with an underlying cause.
func NewValidationErrorWithCause(field, message string, cause error) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
		Err:     cause,
	}
}

// AsAPIError extracts an APIError from the error chain.
// Returns the APIError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
//
// Example:
//
//	if apiErr, ok := langfuse.AsAPIError(err); ok {
//	    log.Printf("API error %d: %s", apiErr.StatusCode, apiErr.Message)
//	}
func AsAPIError(err error) (*APIError, bool) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		return apiErr, true
	}
	return nil, false
}

// AsValidationError extracts a ValidationError from the error chain.
// Returns the ValidationError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
func AsValidationError(err error) (*ValidationError, bool) {
	var valErr *ValidationError
	if errors.As(err, &valErr) {
		return valErr, true
	}
	return nil, false
}

// AsIngestionError extracts an IngestionError from the error chain.
// Returns the IngestionError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
func AsIngestionError(err error) (*IngestionError, bool) {
	var ingErr *IngestionError
	if errors.As(err, &ingErr) {
		return ingErr, true
	}
	return nil, false
}

// AsShutdownError extracts a ShutdownError from the error chain.
// Returns the ShutdownError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
func AsShutdownError(err error) (*ShutdownError, bool) {
	var shutdownErr *ShutdownError
	if errors.As(err, &shutdownErr) {
		return shutdownErr, true
	}
	return nil, false
}

// AsCompilationError extracts a CompilationError from the error chain.
// Returns the CompilationError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
func AsCompilationError(err error) (*CompilationError, bool) {
	var compErr *CompilationError
	if errors.As(err, &compErr) {
		return compErr, true
	}
	return nil, false
}

// RetryAfter returns the suggested retry delay from a rate limit error.
// Returns 0 if the error is not a rate limit error or has no Retry-After hint.
func RetryAfter(err error) time.Duration {
	if apiErr, ok := AsAPIError(err); ok {
		return apiErr.RetryAfter
	}
	return 0
}

// ShutdownError represents an error that occurred during client shutdown.
// It provides context about what was lost during the shutdown failure.
type ShutdownError struct {
	Cause         error // The underlying error (e.g., context deadline exceeded)
	PendingEvents int   // Number of events that may not have been sent
	Message       string
}

// Error implements the error interface.
func (e *ShutdownError) Error() string {
	if e.PendingEvents > 0 {
		return fmt.Sprintf("langfuse: shutdown failed (%s): %d pending events may be lost", e.Message, e.PendingEvents)
	}
	return fmt.Sprintf("langfuse: shutdown failed: %s", e.Message)
}

// Unwrap returns the underlying error for error chain support.
func (e *ShutdownError) Unwrap() error {
	return e.Cause
}

// Code returns the error code for the shutdown error.
// Implements the LangfuseError interface.
func (e *ShutdownError) Code() ErrorCode {
	return ErrCodeShutdown
}

// IsRetryable returns false for shutdown errors (shutdown cannot be retried).
// Implements the LangfuseError interface.
func (e *ShutdownError) IsRetryable() bool {
	return false
}

// GetRequestID returns an empty string (shutdown errors don't have request IDs).
// Implements the LangfuseError interface.
func (e *ShutdownError) GetRequestID() string {
	return ""
}

// Ensure ShutdownError implements LangfuseError.
var _ LangfuseError = (*ShutdownError)(nil)

// Deprecated: IsShutdownError is deprecated, use AsShutdownError instead.
func IsShutdownError(err error) (*ShutdownError, bool) {
	return AsShutdownError(err)
}

// CompilationError represents errors during prompt compilation.
// It collects all errors encountered during compilation, allowing
// partial results to be returned alongside the errors.
type CompilationError struct {
	Errors []error // All errors encountered during compilation
}

// Error implements the error interface.
func (e *CompilationError) Error() string {
	if len(e.Errors) == 0 {
		return "langfuse: prompt compilation failed"
	}
	if len(e.Errors) == 1 {
		return fmt.Sprintf("langfuse: prompt compilation failed: %s", e.Errors[0].Error())
	}
	msgs := make([]string, len(e.Errors))
	for i, err := range e.Errors {
		msgs[i] = err.Error()
	}
	return fmt.Sprintf("langfuse: prompt compilation failed with %d errors: %s",
		len(e.Errors), strings.Join(msgs, "; "))
}

// Unwrap returns the first error for single-error cases.
// For multiple errors, use Errors field directly.
func (e *CompilationError) Unwrap() error {
	if len(e.Errors) == 1 {
		return e.Errors[0]
	}
	return nil
}

// Deprecated: IsCompilationError is deprecated, use AsCompilationError instead.
func IsCompilationError(err error) (*CompilationError, bool) {
	return AsCompilationError(err)
}

// CodedError is an interface for errors that have an error code.
// Implement this interface to allow ErrorCodeOf to extract the code.
type CodedError interface {
	error
	Code() ErrorCode
}

// ErrorCodeOf returns the error code for an error.
// It checks if the error implements CodedError, then falls back to
// inferring the code from the error type.
func ErrorCodeOf(err error) ErrorCode {
	if err == nil {
		return ""
	}

	// Check if error implements CodedError
	var coded CodedError
	if errors.As(err, &coded) {
		return coded.Code()
	}

	// Infer code from error type
	switch {
	case errors.Is(err, ErrMissingPublicKey),
		errors.Is(err, ErrMissingSecretKey),
		errors.Is(err, ErrMissingBaseURL),
		errors.Is(err, ErrInvalidConfig):
		return ErrCodeConfig

	case errors.Is(err, ErrClientClosed),
		errors.Is(err, ErrShutdownTimeout):
		return ErrCodeShutdown

	case errors.Is(err, ErrContextCancelled):
		return ErrCodeTimeout

	case errors.Is(err, ErrCircuitOpen):
		return ErrCodeNetwork
	}

	// Check for API errors
	if apiErr, ok := AsAPIError(err); ok {
		switch {
		case apiErr.IsUnauthorized(), apiErr.IsForbidden():
			return ErrCodeAuth
		case apiErr.IsRateLimited():
			return ErrCodeRateLimit
		default:
			return ErrCodeAPI
		}
	}

	// Check for validation errors
	if _, ok := AsValidationError(err); ok {
		return ErrCodeValidation
	}

	// Check for shutdown errors
	if _, ok := AsShutdownError(err); ok {
		return ErrCodeShutdown
	}

	return ErrCodeInternal
}

// WrapError wraps an error with additional context.
// It returns nil if err is nil.
//
// Example:
//
//	if err := doSomething(); err != nil {
//	    return langfuse.WrapError(err, "failed to process trace")
//	}
func WrapError(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("langfuse: %s: %w", message, err)
}

// WrapErrorf wraps an error with a formatted message.
// It returns nil if err is nil.
//
// Example:
//
//	if err := doSomething(id); err != nil {
//	    return langfuse.WrapErrorf(err, "failed to process trace %s", id)
//	}
func WrapErrorf(err error, format string, args ...any) error {
	if err == nil {
		return nil
	}
	message := fmt.Sprintf(format, args...)
	return fmt.Errorf("langfuse: %s: %w", message, err)
}

// ============================================================================
// Async Errors
// ============================================================================

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
