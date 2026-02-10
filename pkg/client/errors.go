package client

import pkgerrors "github.com/jdziat/langfuse-go/pkg/errors"

// Re-export error types from pkg/errors.
// Note: IngestionError is defined locally in ingestion.go with the same structure.
type (
	APIError      = pkgerrors.APIError
	ShutdownError = pkgerrors.ShutdownError
	LangfuseError = pkgerrors.LangfuseError
	ErrorCode     = pkgerrors.ErrorCode
)

// Error code constants.
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

// Sentinel errors - re-exported from pkg/errors
var (
	ErrMissingPublicKey = pkgerrors.ErrMissingPublicKey
	ErrMissingSecretKey = pkgerrors.ErrMissingSecretKey
	ErrMissingBaseURL   = pkgerrors.ErrMissingBaseURL
	ErrInvalidConfig    = pkgerrors.ErrInvalidConfig
	ErrClientClosed     = pkgerrors.ErrClientClosed
	ErrNilRequest       = pkgerrors.ErrNilRequest
	ErrNotFound         = pkgerrors.ErrNotFound
	ErrUnauthorized     = pkgerrors.ErrUnauthorized
	ErrForbidden        = pkgerrors.ErrForbidden
	ErrRateLimited      = pkgerrors.ErrRateLimited
)
