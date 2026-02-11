package langfuse

import (
	"errors"
	"fmt"
	"time"

	pkgconfig "github.com/jdziat/langfuse-go/pkg/config"
	pkgerrors "github.com/jdziat/langfuse-go/pkg/errors"
	pkgid "github.com/jdziat/langfuse-go/pkg/id"
	"github.com/jdziat/langfuse-go/pkg/types"
)

// ============================================================================
// Type Aliases - Re-exported from pkg/types for backward compatibility
// ============================================================================

// Base types
type (
	// JSON is an alias for any, representing any JSON value.
	JSON = types.JSON

	// JSONObject is an alias for map[string]any, representing a JSON object.
	JSONObject = types.JSONObject

	// Time is a custom time type that handles JSON marshaling/unmarshaling.
	Time = types.Time

	// Metadata provides type-safe metadata storage with JSON serialization.
	Metadata = types.Metadata
)

// Enum types
type (
	// ObservationType represents the type of observation.
	ObservationType = types.ObservationType

	// ObservationLevel represents the severity level of an observation.
	ObservationLevel = types.ObservationLevel

	// ScoreDataType represents the data type of a score.
	ScoreDataType = types.ScoreDataType

	// ScoreSource represents the source of a score.
	ScoreSource = types.ScoreSource

	// PromptType represents the type of a prompt.
	PromptType = types.PromptType
)

// Core types
type (
	// Trace represents a trace in Langfuse.
	Trace = types.Trace

	// Observation represents a span, generation, or event in a trace.
	Observation = types.Observation

	// Usage represents token usage for a generation.
	Usage = types.Usage

	// Score represents a score attached to a trace or observation.
	Score = types.Score

	// Prompt represents a prompt in Langfuse.
	Prompt = types.Prompt

	// TextPrompt represents a text-based prompt.
	TextPrompt = types.TextPrompt

	// ChatPrompt represents a chat-based prompt with messages.
	ChatPrompt = types.ChatPrompt

	// ChatMessage represents a message in a chat prompt.
	ChatMessage = types.ChatMessage

	// Session represents a session in Langfuse.
	Session = types.Session

	// Dataset represents a dataset in Langfuse.
	Dataset = types.Dataset

	// DatasetItem represents an item in a dataset.
	DatasetItem = types.DatasetItem

	// DatasetRun represents a run against a dataset.
	DatasetRun = types.DatasetRun

	// DatasetRunItem represents an item in a dataset run.
	DatasetRunItem = types.DatasetRunItem

	// Model represents a model definition in Langfuse.
	Model = types.Model

	// Comment represents a comment on a trace, observation, session, or prompt.
	Comment = types.Comment

	// HealthStatus represents the health status of the Langfuse API.
	HealthStatus = types.HealthStatus
)

// ============================================================================
// Observation Type Constants
// ============================================================================

const (
	ObservationTypeSpan       = types.ObservationTypeSpan
	ObservationTypeGeneration = types.ObservationTypeGeneration
	ObservationTypeEvent      = types.ObservationTypeEvent
)

// ============================================================================
// Observation Level Constants
// ============================================================================

const (
	ObservationLevelDebug   = types.ObservationLevelDebug
	ObservationLevelDefault = types.ObservationLevelDefault
	ObservationLevelWarning = types.ObservationLevelWarning
	ObservationLevelError   = types.ObservationLevelError
)

// ============================================================================
// Score Data Type Constants
// ============================================================================

const (
	ScoreDataTypeNumeric     = types.ScoreDataTypeNumeric
	ScoreDataTypeCategorical = types.ScoreDataTypeCategorical
	ScoreDataTypeBoolean     = types.ScoreDataTypeBoolean
)

// ============================================================================
// Score Source Constants
// ============================================================================

const (
	ScoreSourceAPI        = types.ScoreSourceAPI
	ScoreSourceAnnotation = types.ScoreSourceAnnotation
	ScoreSourceEval       = types.ScoreSourceEval
)

// ============================================================================
// Prompt Type Constants
// ============================================================================

const (
	PromptTypeText = types.PromptTypeText
	PromptTypeChat = types.PromptTypeChat
)

// ============================================================================
// Environment Constants
// ============================================================================

const (
	EnvProduction  = types.EnvProduction
	EnvDevelopment = types.EnvDevelopment
	EnvStaging     = types.EnvStaging
	EnvTest        = types.EnvTest
)

// ============================================================================
// Label Constants
// ============================================================================

const (
	LabelProduction  = types.LabelProduction
	LabelDevelopment = types.LabelDevelopment
	LabelStaging     = types.LabelStaging
	LabelLatest      = types.LabelLatest
)

// ============================================================================
// Model Name Constants
// ============================================================================

const (
	// OpenAI models
	ModelGPT4          = types.ModelGPT4
	ModelGPT4Turbo     = types.ModelGPT4Turbo
	ModelGPT4o         = types.ModelGPT4o
	ModelGPT4oMini     = types.ModelGPT4oMini
	ModelGPT35Turbo    = types.ModelGPT35Turbo
	ModelO1            = types.ModelO1
	ModelO1Mini        = types.ModelO1Mini
	ModelO1Preview     = types.ModelO1Preview
	ModelO3Mini        = types.ModelO3Mini
	ModelTextEmbedding = types.ModelTextEmbedding

	// Anthropic models
	ModelClaude3Opus    = types.ModelClaude3Opus
	ModelClaude3Sonnet  = types.ModelClaude3Sonnet
	ModelClaude3Haiku   = types.ModelClaude3Haiku
	ModelClaude35Sonnet = types.ModelClaude35Sonnet
	ModelClaude35Haiku  = types.ModelClaude35Haiku
	ModelClaude4Opus    = types.ModelClaude4Opus
	ModelClaude4Sonnet  = types.ModelClaude4Sonnet

	// Google models
	ModelGeminiPro     = types.ModelGeminiPro
	ModelGemini15Pro   = types.ModelGemini15Pro
	ModelGemini15Flash = types.ModelGemini15Flash
	ModelGemini20Flash = types.ModelGemini20Flash
)

// ============================================================================
// Helper Functions - Re-exported from pkg/types
// ============================================================================

// Now returns the current time as a Time.
var Now = types.Now

// TimePtr returns a pointer to a Time value.
var TimePtr = types.TimePtr

// TimeNow returns a pointer to the current time.
var TimeNow = types.TimeNow

// NewMetadata creates a new empty Metadata instance.
var NewMetadata = types.NewMetadata

// ============================================================================
// Re-exports from pkg/config
// ============================================================================

// Environment helper functions from pkg/config.
var (
	// GetEnvString returns the value of an environment variable or a default.
	GetEnvString = pkgconfig.GetEnvString
	// GetEnvBool returns true if the env var is "true" or "1".
	GetEnvBool = pkgconfig.GetEnvBool
	// GetEnvRegion returns the region from environment or default.
	GetEnvRegion = pkgconfig.GetEnvRegion
)

// ============================================================================
// ID Generation - Re-exported from pkg/id for backward compatibility
// ============================================================================

// IDGenerationMode controls how IDs are generated when crypto/rand fails.
type IDGenerationMode = pkgid.IDGenerationMode

// IDGenerator generates unique IDs with configurable failure handling.
type IDGenerator = pkgid.IDGenerator

// IDGeneratorConfig configures the ID generator.
type IDGeneratorConfig = pkgid.IDGeneratorConfig

// IDStats contains statistics about ID generation.
type IDStats = pkgid.IDStats

// ID Generation Mode Constants
const (
	// IDModeFallback uses an atomic counter fallback when crypto/rand fails.
	// This is the default mode for backwards compatibility.
	IDModeFallback = pkgid.IDModeFallback

	// IDModeStrict returns an error when crypto/rand fails.
	// Recommended for production deployments where ID uniqueness is critical.
	IDModeStrict = pkgid.IDModeStrict
)

// ID Generation Functions
var (
	// NewIDGenerator creates an ID generator with the specified configuration.
	NewIDGenerator = pkgid.NewIDGenerator

	// GenerateID generates a unique ID using the default generator.
	// This is the primary entry point for ID generation in the SDK.
	GenerateID = pkgid.GenerateID

	// MustGenerateID generates an ID or panics.
	// Use only when ID generation must succeed.
	MustGenerateID = pkgid.MustGenerateID

	// SetDefaultIDGenerator sets the package-level ID generator.
	// Call this early in application startup to configure ID generation.
	SetDefaultIDGenerator = pkgid.SetDefaultIDGenerator

	// CryptoFailureCount returns the total number of crypto/rand failures.
	CryptoFailureCount = pkgid.CryptoFailureCount

	// ResetCryptoFailureCount resets the failure counter (for testing).
	ResetCryptoFailureCount = pkgid.ResetCryptoFailureCount

	// IsFallbackID returns true if the ID was generated using the fallback method.
	// Fallback IDs start with "fb-".
	IsFallbackID = pkgid.IsFallbackID
)

// generateIDInternal is used internally and maintains backwards compatibility.
// It never returns an error, using fallback mode implicitly.
func generateIDInternal() string {
	return pkgid.GenerateIDInternal()
}

// ============================================================================
// Error Types - Re-exported from pkg/errors for backward compatibility
// ============================================================================

// Error code types
type (
	// ErrorCode represents a category of error for metrics and logging.
	ErrorCode = pkgerrors.ErrorCode
)

// Error code constants
const (
	ErrCodeConfig       = pkgerrors.ErrCodeConfig
	ErrCodeValidation   = pkgerrors.ErrCodeValidation
	ErrCodeNetwork      = pkgerrors.ErrCodeNetwork
	ErrCodeAPI          = pkgerrors.ErrCodeAPI
	ErrCodeAuth         = pkgerrors.ErrCodeAuth
	ErrCodeRateLimit    = pkgerrors.ErrCodeRateLimit
	ErrCodeTimeout      = pkgerrors.ErrCodeTimeout
	ErrCodeInternal     = pkgerrors.ErrCodeInternal
	ErrCodeShutdown     = pkgerrors.ErrCodeShutdown
	ErrCodeBackpressure = pkgerrors.ErrCodeBackpressure
)

// Error types
type (
	// LangfuseError is the common interface for all SDK errors.
	LangfuseError = pkgerrors.LangfuseError

	// CodedError is an interface for errors that have an error code.
	CodedError = pkgerrors.CodedError

	// APIError represents an error response from the Langfuse API.
	APIError = pkgerrors.APIError

	// ValidationError represents a validation error for a request.
	ValidationError = pkgerrors.ValidationError

	// ShutdownError represents an error that occurred during client shutdown.
	ShutdownError = pkgerrors.ShutdownError

	// CompilationError represents errors during prompt compilation.
	CompilationError = pkgerrors.CompilationError

	// IngestionError represents an error that occurred during batch ingestion.
	IngestionError = pkgerrors.IngestionError

	// IngestionResult represents the result of a batch ingestion request.
	IngestionResult = pkgerrors.IngestionResult

	// IngestionSuccess represents a successful ingestion event.
	IngestionSuccess = pkgerrors.IngestionSuccess
)

// Async error types
type (
	// AsyncErrorOperation identifies the type of async operation that failed.
	AsyncErrorOperation = pkgerrors.AsyncErrorOperation

	// AsyncError represents an error that occurred in background processing.
	AsyncError = pkgerrors.AsyncError

	// AsyncErrorHandler provides structured async error handling.
	AsyncErrorHandler = pkgerrors.AsyncErrorHandler

	// AsyncErrorConfig configures the AsyncErrorHandler.
	AsyncErrorConfig = pkgerrors.AsyncErrorConfig

	// AsyncErrorStats contains statistics about async error handling.
	AsyncErrorStats = pkgerrors.AsyncErrorStats
)

// Async operation constants
const (
	AsyncOpBatchSend = pkgerrors.AsyncOpBatchSend
	AsyncOpFlush     = pkgerrors.AsyncOpFlush
	AsyncOpHook      = pkgerrors.AsyncOpHook
	AsyncOpShutdown  = pkgerrors.AsyncOpShutdown
	AsyncOpQueue     = pkgerrors.AsyncOpQueue
	AsyncOpInternal  = pkgerrors.AsyncOpInternal
)

// ============================================================================
// Sentinel Errors - Re-exported from pkg/errors
// ============================================================================

// Configuration validation errors
var (
	ErrMissingPublicKey = pkgerrors.ErrMissingPublicKey
	ErrMissingSecretKey = pkgerrors.ErrMissingSecretKey
	ErrMissingBaseURL   = pkgerrors.ErrMissingBaseURL
	ErrInvalidConfig    = pkgerrors.ErrInvalidConfig
	ErrClientClosed     = pkgerrors.ErrClientClosed
	ErrNilRequest       = pkgerrors.ErrNilRequest
)

// Common scenario errors
var (
	ErrPromptNotFound   = pkgerrors.ErrPromptNotFound
	ErrDatasetNotFound  = pkgerrors.ErrDatasetNotFound
	ErrTraceNotFound    = pkgerrors.ErrTraceNotFound
	ErrEmptyBatch       = pkgerrors.ErrEmptyBatch
	ErrBatchTooLarge    = pkgerrors.ErrBatchTooLarge
	ErrContextCancelled = pkgerrors.ErrContextCancelled
	ErrShutdownTimeout  = pkgerrors.ErrShutdownTimeout
)

// Sentinel APIError values for use with errors.Is().
var (
	ErrNotFound     = pkgerrors.ErrNotFound
	ErrUnauthorized = pkgerrors.ErrUnauthorized
	ErrForbidden    = pkgerrors.ErrForbidden
	ErrRateLimited  = pkgerrors.ErrRateLimited
)

// ============================================================================
// Error Constructor Functions - Re-exported from pkg/errors
// ============================================================================

// NewValidationError creates a new validation error.
var NewValidationError = pkgerrors.NewValidationError

// NewValidationErrorWithCause creates a new validation error with an underlying cause.
var NewValidationErrorWithCause = pkgerrors.NewValidationErrorWithCause

// NewAsyncError creates a new async error.
var NewAsyncError = pkgerrors.NewAsyncError

// NewAsyncErrorHandler creates a new async error handler.
var NewAsyncErrorHandler = pkgerrors.NewAsyncErrorHandler

// WrapAsyncError wraps an error in an AsyncError if it isn't already.
var WrapAsyncError = pkgerrors.WrapAsyncError

// ============================================================================
// Error Helper Functions
// ============================================================================

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
	// Note: ErrCircuitOpen is defined in http.go
	if errors.Is(err, ErrCircuitOpen) {
		return true
	}

	return false
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
	return pkgerrors.AsAPIError(err)
}

// AsValidationError extracts a ValidationError from the error chain.
// Returns the ValidationError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
func AsValidationError(err error) (*ValidationError, bool) {
	return pkgerrors.AsValidationError(err)
}

// AsIngestionError extracts an IngestionError from the error chain.
// Returns the IngestionError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
func AsIngestionError(err error) (*IngestionError, bool) {
	return pkgerrors.AsIngestionError(err)
}

// AsShutdownError extracts a ShutdownError from the error chain.
// Returns the ShutdownError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
func AsShutdownError(err error) (*ShutdownError, bool) {
	return pkgerrors.AsShutdownError(err)
}

// AsCompilationError extracts a CompilationError from the error chain.
// Returns the CompilationError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
// Also handles types.CompilationError from pkg/types.
func AsCompilationError(err error) (*CompilationError, bool) {
	var compErr *CompilationError
	if errors.As(err, &compErr) {
		return compErr, true
	}
	// Also check for types.CompilationError (returned by Prompt.Compile methods)
	var typesCompErr *types.CompilationError
	if errors.As(err, &typesCompErr) {
		// Convert to root CompilationError for backward compatibility
		return &CompilationError{Errors: typesCompErr.Errors}, true
	}
	return nil, false
}

// AsAsyncError extracts an AsyncError from the error chain.
// Returns the AsyncError and true if found, nil and false otherwise.
// This follows Go's errors.As() convention.
func AsAsyncError(err error) (*AsyncError, bool) {
	return pkgerrors.AsAsyncError(err)
}

// RetryAfter returns the suggested retry delay from a rate limit error.
// Returns 0 if the error is not a rate limit error or has no Retry-After hint.
func RetryAfter(err error) time.Duration {
	if apiErr, ok := AsAPIError(err); ok {
		return apiErr.RetryAfter
	}
	return 0
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

// Deprecated: IsShutdownError is deprecated, use AsShutdownError instead.
func IsShutdownError(err error) (*ShutdownError, bool) {
	return AsShutdownError(err)
}

// Deprecated: IsCompilationError is deprecated, use AsCompilationError instead.
func IsCompilationError(err error) (*CompilationError, bool) {
	return AsCompilationError(err)
}
