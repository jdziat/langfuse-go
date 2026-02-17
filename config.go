package langfuse

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	pkgclient "github.com/jdziat/langfuse-go/pkg/client"
	pkgconfig "github.com/jdziat/langfuse-go/pkg/config"
)

// ============================================================================
// Region Types and Constants - Re-exported from pkg/config
// ============================================================================

// Region represents a Langfuse cloud region.
type Region = pkgconfig.Region

// Region constants.
const (
	// RegionEU is the European cloud region.
	RegionEU = pkgconfig.RegionEU
	// RegionUS is the US cloud region.
	RegionUS = pkgconfig.RegionUS
	// RegionHIPAA is the HIPAA-compliant US region.
	RegionHIPAA = pkgconfig.RegionHIPAA
)

// regionBaseURLs maps regions to their base URLs.
var regionBaseURLs = pkgconfig.RegionBaseURLs

// ============================================================================
// Environment Variable Constants
// ============================================================================

// Environment variable names for configuration.
const (
	// EnvPublicKey is the environment variable for the Langfuse public key.
	EnvPublicKey = "LANGFUSE_PUBLIC_KEY"
	// EnvSecretKey is the environment variable for the Langfuse secret key.
	EnvSecretKey = "LANGFUSE_SECRET_KEY"
	// EnvBaseURL is the environment variable for the Langfuse API base URL.
	EnvBaseURL = "LANGFUSE_BASE_URL"
	// EnvHost is an alias for EnvBaseURL (for compatibility).
	EnvHost = "LANGFUSE_HOST"
	// EnvRegion is the environment variable for the Langfuse cloud region.
	EnvRegion = "LANGFUSE_REGION"
	// EnvDebug is the environment variable to enable debug mode.
	EnvDebug = "LANGFUSE_DEBUG"
)

// ============================================================================
// Default Configuration Values - Re-exported from pkg/config
// ============================================================================

// Default configuration values (re-exported from pkg/config for backward compatibility).
const (
	// DefaultTimeout is the default request timeout.
	DefaultTimeout = pkgconfig.DefaultTimeout

	// DefaultMaxRetries is the default maximum number of retry attempts.
	DefaultMaxRetries = pkgconfig.DefaultMaxRetries

	// DefaultRetryDelay is the default initial delay between retry attempts.
	DefaultRetryDelay = pkgconfig.DefaultRetryDelay

	// DefaultBatchSize is the default maximum number of events per batch.
	DefaultBatchSize = pkgconfig.DefaultBatchSize

	// DefaultFlushInterval is the default interval for flushing pending events.
	DefaultFlushInterval = pkgconfig.DefaultFlushInterval

	// DefaultMaxIdleConns is the default maximum number of idle connections.
	DefaultMaxIdleConns = pkgconfig.DefaultMaxIdleConns

	// DefaultMaxIdleConnsPerHost is the default maximum idle connections per host.
	DefaultMaxIdleConnsPerHost = pkgconfig.DefaultMaxIdleConnsPerHost

	// DefaultIdleConnTimeout is the default timeout for idle connections.
	DefaultIdleConnTimeout = pkgconfig.DefaultIdleConnTimeout

	// DefaultShutdownTimeout is the default graceful shutdown timeout.
	// Must be >= DefaultTimeout to allow pending requests to complete.
	DefaultShutdownTimeout = pkgconfig.DefaultShutdownTimeout

	// DefaultBatchQueueSize is the default size of the background batch queue.
	DefaultBatchQueueSize = pkgconfig.DefaultBatchQueueSize

	// DefaultBackgroundSendTimeout is the timeout for background batch sends.
	DefaultBackgroundSendTimeout = pkgconfig.DefaultBackgroundSendTimeout

	// DefaultMaxBackgroundSenders is the default max concurrent background senders.
	DefaultMaxBackgroundSenders = pkgconfig.DefaultMaxBackgroundSenders

	// MaxBatchSize is the maximum allowed batch size.
	MaxBatchSize = pkgconfig.MaxBatchSize

	// MaxMaxRetries is the maximum allowed retry count.
	MaxMaxRetries = pkgconfig.MaxMaxRetries

	// MaxTimeout is the maximum allowed request timeout.
	MaxTimeout = pkgconfig.MaxTimeout

	// MinFlushInterval is the minimum allowed flush interval.
	MinFlushInterval = pkgconfig.MinFlushInterval

	// MinShutdownTimeout is the minimum allowed shutdown timeout.
	MinShutdownTimeout = pkgconfig.MinShutdownTimeout

	// MinKeyLength is the minimum length for API keys.
	MinKeyLength = pkgconfig.MinKeyLength

	// PublicKeyPrefix is the expected prefix for public keys.
	PublicKeyPrefix = pkgconfig.PublicKeyPrefix

	// SecretKeyPrefix is the expected prefix for secret keys.
	SecretKeyPrefix = pkgconfig.SecretKeyPrefix
)

// Config holds the configuration for the Langfuse client.
type Config struct {
	// PublicKey is the Langfuse public key (required).
	PublicKey string

	// SecretKey is the Langfuse secret key (required).
	SecretKey string

	// BaseURL is the base URL for the Langfuse API.
	// If not set, it will be derived from the Region.
	BaseURL string

	// Region is the Langfuse cloud region.
	// Defaults to RegionEU if not set and BaseURL is empty.
	Region Region

	// HTTPClient is the HTTP client to use for requests.
	// If not set, a default client with sensible timeouts will be used.
	HTTPClient *http.Client

	// Timeout is the request timeout.
	// Defaults to 30 seconds if not set.
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts for failed requests.
	// Defaults to 3 if not set.
	MaxRetries int

	// RetryDelay is the initial delay between retry attempts.
	// Defaults to 1 second if not set.
	RetryDelay time.Duration

	// BatchSize is the maximum number of events to send in a single batch.
	// Defaults to 100 if not set.
	BatchSize int

	// FlushInterval is the interval at which to flush pending events.
	// Defaults to 5 seconds if not set.
	FlushInterval time.Duration

	// Debug enables debug logging.
	Debug bool

	// ErrorHandler is called when async operations fail.
	// If nil, errors are silently dropped unless Debug is true.
	ErrorHandler func(error)

	// Logger is used for SDK logging (printf-style).
	// If nil, logging is disabled unless Debug is true.
	// For structured logging, use StructuredLogger instead.
	Logger Logger

	// StructuredLogger is used for structured SDK logging.
	// If set, this takes precedence over Logger.
	// Compatible with slog.Logger via NewSlogAdapter().
	StructuredLogger StructuredLogger

	// Metrics is used for SDK telemetry.
	// If nil, no metrics are collected.
	Metrics Metrics

	// MaxIdleConns controls the maximum number of idle connections across all hosts.
	// Defaults to 100 if not set.
	MaxIdleConns int

	// MaxIdleConnsPerHost controls the maximum number of idle connections per host.
	// Defaults to 10 if not set.
	MaxIdleConnsPerHost int

	// IdleConnTimeout is how long idle connections are kept.
	// Defaults to 90 seconds if not set.
	IdleConnTimeout time.Duration

	// ShutdownTimeout is the maximum time to wait for graceful shutdown.
	// Defaults to 5 seconds if not set.
	ShutdownTimeout time.Duration

	// BatchQueueSize is the size of the background batch queue.
	// Defaults to 100 if not set.
	BatchQueueSize int

	// RetryStrategy is the strategy for retrying failed requests.
	// If nil, a default exponential backoff strategy is used.
	RetryStrategy RetryStrategy

	// CircuitBreaker configures fault tolerance for API calls.
	// If nil, circuit breaker is disabled.
	CircuitBreaker *CircuitBreakerConfig

	// OnBatchFlushed is called after each batch is sent to the API.
	// It receives the result of the batch send operation.
	// This is useful for monitoring, logging, or custom error handling.
	OnBatchFlushed func(result BatchResult)

	// HTTPHooks are called before and after each HTTP request.
	// Use hooks to add custom headers, log requests, or collect metrics.
	HTTPHooks []HTTPHook

	// IdleWarningDuration triggers a warning if the client is idle for this duration
	// without Shutdown() being called. This helps detect goroutine leaks.
	// Set to 0 to disable (default).
	// Recommended: 5*time.Minute for development, 0 for production.
	IdleWarningDuration time.Duration

	// IDGenerationMode controls how IDs are generated when crypto/rand fails.
	// Default is IDModeFallback for backwards compatibility.
	// Production deployments may want to use IDModeStrict.
	IDGenerationMode IDGenerationMode

	// ClassifiedHooks are priority-aware HTTP hooks with differentiated error handling.
	// Critical hooks block on errors; Observational hooks log errors but continue.
	// If both HTTPHooks and ClassifiedHooks are set, ClassifiedHooks take precedence.
	ClassifiedHooks []ClassifiedHook

	// AsyncErrorConfig configures the async error handler for background operations.
	// If nil, default configuration is used.
	AsyncErrorConfig *AsyncErrorConfig

	// OnAsyncError is called for each async error. This is a convenience
	// alternative to configuring AsyncErrorConfig.OnError directly.
	OnAsyncError func(*AsyncError)

	// BackpressureConfig configures queue monitoring and backpressure handling.
	// If nil, default configuration is used.
	BackpressureConfig *BackpressureHandlerConfig

	// OnBackpressure is called when backpressure level changes.
	// This is a convenience alternative to configuring BackpressureConfig.
	OnBackpressure BackpressureCallback

	// BlockOnQueueFull blocks event submission when the queue is at overflow.
	// If false (default), events may be dropped when the queue is full.
	BlockOnQueueFull bool

	// DropOnQueueFull drops events silently when the queue is at overflow.
	// Only applies if BlockOnQueueFull is false. Default is false (queue events anyway).
	DropOnQueueFull bool

	// MaxBackgroundSenders limits the number of concurrent goroutines used for
	// background batch sending when the queue is full. This prevents unbounded
	// goroutine creation under sustained high load. Default is 10.
	MaxBackgroundSenders int

	// StrictValidation enables strict validation mode with validated builders.
	// When enabled, NewTraceStrict(), NewSpanStrict(), etc. methods become available.
	// These return BuildResult types that force explicit error handling.
	StrictValidation *StrictValidationConfig

	// EnableMetricsRecorder enables the internal metrics recorder.
	// When true, SDK internal metrics are collected and can be exported
	// via the Metrics interface. Requires Metrics to be set.
	EnableMetricsRecorder bool

	// EvaluationConfig configures automatic evaluation mode.
	// When set, traces are automatically structured for LLM-as-a-Judge evaluation.
	// This includes field flattening, automatic metadata, and evaluation tags.
	EvaluationConfig *EvaluationConfig
}

// String returns a string representation of the config with masked credentials.
// This is safe to use in logs and debug output.
func (c *Config) String() string {
	return fmt.Sprintf("Config{PublicKey: %q, SecretKey: %q, BaseURL: %q, Region: %q, BatchSize: %d, FlushInterval: %v}",
		MaskCredential(c.PublicKey),
		MaskCredential(c.SecretKey),
		c.BaseURL,
		c.Region,
		c.BatchSize,
		c.FlushInterval,
	)
}

// BatchResult contains information about a flushed batch.
// It is an alias to pkgclient.BatchResult for type compatibility.
type BatchResult = pkgclient.BatchResult

// applyDefaults sets default values for unset configuration options.
func (c *Config) applyDefaults() {
	if c.BaseURL == "" {
		if c.Region == "" {
			c.Region = RegionEU
		}
		if baseURL, ok := regionBaseURLs[c.Region]; ok {
			c.BaseURL = baseURL
		}
	}

	if c.Timeout == 0 {
		c.Timeout = DefaultTimeout
	}

	if c.MaxRetries == 0 {
		c.MaxRetries = DefaultMaxRetries
	}

	if c.RetryDelay == 0 {
		c.RetryDelay = DefaultRetryDelay
	}

	if c.BatchSize == 0 {
		c.BatchSize = DefaultBatchSize
	}

	if c.FlushInterval == 0 {
		c.FlushInterval = DefaultFlushInterval
	}

	if c.MaxIdleConns == 0 {
		c.MaxIdleConns = DefaultMaxIdleConns
	}

	if c.MaxIdleConnsPerHost == 0 {
		c.MaxIdleConnsPerHost = DefaultMaxIdleConnsPerHost
	}

	if c.IdleConnTimeout == 0 {
		c.IdleConnTimeout = DefaultIdleConnTimeout
	}

	if c.ShutdownTimeout == 0 {
		c.ShutdownTimeout = DefaultShutdownTimeout
	}

	if c.BatchQueueSize == 0 {
		c.BatchQueueSize = DefaultBatchQueueSize
	}

	if c.MaxBackgroundSenders == 0 {
		c.MaxBackgroundSenders = DefaultMaxBackgroundSenders
	}

	// Set default logger if debug is enabled and no logger is set
	if c.Debug && c.Logger == nil {
		c.Logger = &defaultLogger{
			logger: log.New(os.Stderr, "langfuse: ", log.LstdFlags),
		}
	}

	if c.HTTPClient == nil {
		c.HTTPClient = &http.Client{
			Timeout: c.Timeout,
			Transport: &http.Transport{
				MaxIdleConns:        c.MaxIdleConns,
				MaxIdleConnsPerHost: c.MaxIdleConnsPerHost,
				IdleConnTimeout:     c.IdleConnTimeout,
				DisableKeepAlives:   false,
			},
		}
	}
}

// validate checks that the configuration is valid.
func (c *Config) validate() error {
	if c.PublicKey == "" {
		return ErrMissingPublicKey
	}
	if c.SecretKey == "" {
		return ErrMissingSecretKey
	}
	if c.BaseURL == "" {
		return ErrMissingBaseURL
	}

	// Validate credential formats
	if len(c.PublicKey) < MinKeyLength {
		return fmt.Errorf("langfuse: public key is too short (minimum %d characters)", MinKeyLength)
	}
	if len(c.SecretKey) < MinKeyLength {
		return fmt.Errorf("langfuse: secret key is too short (minimum %d characters)", MinKeyLength)
	}
	if !strings.HasPrefix(c.PublicKey, PublicKeyPrefix) {
		return fmt.Errorf("langfuse: public key should start with %q", PublicKeyPrefix)
	}
	if !strings.HasPrefix(c.SecretKey, SecretKeyPrefix) {
		return fmt.Errorf("langfuse: secret key should start with %q", SecretKeyPrefix)
	}

	// Validate URL format
	if _, err := url.Parse(c.BaseURL); err != nil {
		return fmt.Errorf("langfuse: invalid base URL: %w", err)
	}

	// Validate numeric ranges
	if c.BatchSize < 1 {
		return fmt.Errorf("langfuse: batch size must be at least 1, got %d", c.BatchSize)
	}
	if c.BatchSize > MaxBatchSize {
		return fmt.Errorf("langfuse: batch size cannot exceed %d, got %d", MaxBatchSize, c.BatchSize)
	}
	if c.MaxRetries < 0 {
		return fmt.Errorf("langfuse: max retries cannot be negative, got %d", c.MaxRetries)
	}
	if c.MaxRetries > MaxMaxRetries {
		return fmt.Errorf("langfuse: max retries cannot exceed %d, got %d", MaxMaxRetries, c.MaxRetries)
	}
	if c.BatchQueueSize < 1 {
		return fmt.Errorf("langfuse: batch queue size must be at least 1, got %d", c.BatchQueueSize)
	}

	// Validate durations
	if c.Timeout < 0 {
		return fmt.Errorf("langfuse: timeout cannot be negative")
	}
	if c.Timeout > MaxTimeout {
		return fmt.Errorf("langfuse: timeout cannot exceed %v", MaxTimeout)
	}
	if c.FlushInterval < MinFlushInterval {
		return fmt.Errorf("langfuse: flush interval must be at least %v", MinFlushInterval)
	}
	if c.ShutdownTimeout < MinShutdownTimeout {
		return fmt.Errorf("langfuse: shutdown timeout must be at least %v", MinShutdownTimeout)
	}

	// Validate config relationships
	if c.MaxIdleConnsPerHost > c.MaxIdleConns {
		return fmt.Errorf("langfuse: max idle connections per host (%d) cannot exceed total max idle connections (%d)",
			c.MaxIdleConnsPerHost, c.MaxIdleConns)
	}
	if c.ShutdownTimeout < c.Timeout {
		return fmt.Errorf("langfuse: shutdown timeout (%v) should be >= request timeout (%v) to allow pending requests to complete",
			c.ShutdownTimeout, c.Timeout)
	}

	// Validate backpressure options are mutually exclusive
	if c.BlockOnQueueFull && c.DropOnQueueFull {
		return fmt.Errorf("langfuse: BlockOnQueueFull and DropOnQueueFull are mutually exclusive; set only one")
	}

	// Validate MaxBackgroundSenders is non-negative
	if c.MaxBackgroundSenders < 0 {
		return fmt.Errorf("langfuse: MaxBackgroundSenders cannot be negative, got %d", c.MaxBackgroundSenders)
	}

	return nil
}

// DefaultConfig returns a production-ready configuration with sensible defaults.
// Use this as a starting point for most production deployments.
//
// Example:
//
//	cfg := langfuse.DefaultConfig("pk-xxx", "sk-xxx")
//	client, err := langfuse.NewWithConfig(cfg)
func DefaultConfig(publicKey, secretKey string) *Config {
	return &Config{
		PublicKey: publicKey,
		SecretKey: secretKey,
		Region:    RegionEU,
	}
}

// DevelopmentConfig returns a configuration suitable for development.
// Features:
//   - Debug logging enabled
//   - BatchSize of 1 for immediate flushing (see events in real-time)
//   - Shorter flush interval for faster feedback
//
// Example:
//
//	cfg := langfuse.DevelopmentConfig("pk-xxx", "sk-xxx")
//	client, err := langfuse.NewWithConfig(cfg)
func DevelopmentConfig(publicKey, secretKey string) *Config {
	return &Config{
		PublicKey:     publicKey,
		SecretKey:     secretKey,
		Region:        RegionEU,
		Debug:         true,
		BatchSize:     1,
		FlushInterval: 1 * time.Second,
	}
}

// HighThroughputConfig returns a configuration optimized for high-volume ingestion.
// Features:
//   - Larger batch size (500 events)
//   - Larger batch queue (500 batches)
//   - Longer flush interval (10s) to maximize batching
//   - More idle connections for concurrent requests
//
// Example:
//
//	cfg := langfuse.HighThroughputConfig("pk-xxx", "sk-xxx")
//	client, err := langfuse.NewWithConfig(cfg)
func HighThroughputConfig(publicKey, secretKey string) *Config {
	return &Config{
		PublicKey:           publicKey,
		SecretKey:           secretKey,
		Region:              RegionEU,
		BatchSize:           500,
		BatchQueueSize:      500,
		FlushInterval:       10 * time.Second,
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 50,
	}
}

// ============================================================================
// Environment Variable Configuration
// ============================================================================

// NewFromEnv creates a new client using environment variables for configuration.
// It reads LANGFUSE_PUBLIC_KEY, LANGFUSE_SECRET_KEY, and optionally
// LANGFUSE_BASE_URL (or LANGFUSE_HOST), LANGFUSE_REGION, and LANGFUSE_DEBUG.
//
// Example:
//
//	client, err := langfuse.NewFromEnv()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer client.Shutdown(context.Background())
func NewFromEnv(opts ...ConfigOption) (*Client, error) {
	publicKey := os.Getenv(EnvPublicKey)
	secretKey := os.Getenv(EnvSecretKey)

	if publicKey == "" {
		return nil, fmt.Errorf("langfuse: %s environment variable is required", EnvPublicKey)
	}
	if secretKey == "" {
		return nil, fmt.Errorf("langfuse: %s environment variable is required", EnvSecretKey)
	}

	// Prepend env var options so explicit options can override them
	envOpts := make([]ConfigOption, 0, 4)

	// Check for base URL (LANGFUSE_BASE_URL or LANGFUSE_HOST)
	if baseURL := os.Getenv(EnvBaseURL); baseURL != "" {
		envOpts = append(envOpts, WithBaseURL(baseURL))
	} else if host := os.Getenv(EnvHost); host != "" {
		envOpts = append(envOpts, WithBaseURL(host))
	}

	// Check for region
	if region := os.Getenv(EnvRegion); region != "" {
		envOpts = append(envOpts, WithRegion(Region(region)))
	}

	// Check for debug mode
	if debug := os.Getenv(EnvDebug); debug == "true" || debug == "1" {
		envOpts = append(envOpts, WithDebug(true))
	}

	// Combine env options with explicit options (explicit options take precedence)
	allOpts := append(envOpts, opts...)

	return New(publicKey, secretKey, allOpts...)
}

// ============================================================================
// Logging Interfaces and Implementations
// ============================================================================

// Logger is a minimal logging interface for the SDK.
// It's compatible with standard library log.Logger and popular logging frameworks.
//
// Deprecated: Use StructuredLogger instead, which provides leveled logging.
// You can wrap a printf-style logger using WrapPrintfLogger():
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.WrapPrintfLogger(log.Default())),
//	)
type Logger = pkgclient.Logger

// StructuredLogger provides structured logging support for the SDK.
// This is the preferred logging interface and is compatible with Go 1.21's
// slog package and similar structured logging libraries.
//
// Use WithStructuredLogger() to configure:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.NewSlogAdapter(slog.Default())),
//	)
//
// Or wrap a standard logger:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.WrapPrintfLogger(log.Default())),
//	)
type StructuredLogger = pkgclient.StructuredLogger

// printfLoggerWrapper wraps a printf-style logger to implement StructuredLogger.
type printfLoggerWrapper struct {
	logger Logger
}

// WrapPrintfLogger wraps a printf-style Logger (like *log.Logger) to implement
// StructuredLogger. All messages are logged at the same level with formatted
// key-value pairs appended.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.WrapPrintfLogger(log.Default())),
//	)
func WrapPrintfLogger(l Logger) StructuredLogger {
	return &printfLoggerWrapper{logger: l}
}

// WrapStdLogger wraps a standard library *log.Logger to implement StructuredLogger.
// This is a convenience function equivalent to WrapPrintfLogger(l).
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.WrapStdLogger(log.Default())),
//	)
func WrapStdLogger(l *log.Logger) StructuredLogger {
	return &printfLoggerWrapper{logger: &defaultLogger{logger: l}}
}

func (w *printfLoggerWrapper) Debug(msg string, args ...any) {
	w.logger.Printf("[DEBUG] " + msg + formatArgs(args))
}

func (w *printfLoggerWrapper) Info(msg string, args ...any) {
	w.logger.Printf("[INFO] " + msg + formatArgs(args))
}

func (w *printfLoggerWrapper) Warn(msg string, args ...any) {
	w.logger.Printf("[WARN] " + msg + formatArgs(args))
}

func (w *printfLoggerWrapper) Error(msg string, args ...any) {
	w.logger.Printf("[ERROR] " + msg + formatArgs(args))
}

// Ensure printfLoggerWrapper implements StructuredLogger.
var _ StructuredLogger = (*printfLoggerWrapper)(nil)

// Metrics is an optional interface for SDK telemetry.
type Metrics = pkgclient.Metrics

// defaultLogger wraps the standard library logger.
type defaultLogger struct {
	logger *log.Logger
}

func (l *defaultLogger) Printf(format string, v ...any) {
	l.logger.Printf(format, v...)
}

// formatArgs formats structured logging arguments as a string.
func formatArgs(args []any) string {
	if len(args) == 0 {
		return ""
	}
	result := " |"
	for i := 0; i < len(args)-1; i += 2 {
		key := args[i]
		var value any
		if i+1 < len(args) {
			value = args[i+1]
		}
		result += fmt.Sprintf(" %v=%v", key, value)
	}
	return result
}

// NopLogger is a logger that discards all log messages.
// Use this to disable logging entirely.
type NopLogger struct{}

// Printf implements Logger.Printf.
func (NopLogger) Printf(format string, v ...any) {}

// Debug implements StructuredLogger.Debug.
func (NopLogger) Debug(msg string, args ...any) {}

// Info implements StructuredLogger.Info.
func (NopLogger) Info(msg string, args ...any) {}

// Warn implements StructuredLogger.Warn.
func (NopLogger) Warn(msg string, args ...any) {}

// Error implements StructuredLogger.Error.
func (NopLogger) Error(msg string, args ...any) {}

// Ensure NopLogger implements both interfaces.
var (
	_ Logger           = NopLogger{}
	_ StructuredLogger = NopLogger{}
)

// MaskCredential masks a credential string for safe logging.
// It preserves the prefix and shows only the last 4 characters.
// For short strings, it returns a fully masked version.
//
// Examples:
//
//	MaskCredential("pk-lf-1234567890abcdef") => "pk-lf-************cdef"
//	MaskCredential("sk-lf-abcd1234efgh5678") => "sk-lf-************5678"
//	MaskCredential("short") => "****t"
func MaskCredential(s string) string {
	if s == "" {
		return ""
	}

	const (
		visibleSuffix = 4
		minMaskLength = 8
	)

	// For very short strings, mask most of it
	if len(s) <= visibleSuffix {
		return "****"
	}

	// Find prefix (up to first hyphen after the type prefix)
	// e.g., "pk-lf-" or "sk-lf-"
	prefixEnd := 0
	hyphenCount := 0
	for i, c := range s {
		if c == '-' {
			hyphenCount++
			if hyphenCount == 2 {
				prefixEnd = i + 1
				break
			}
		}
	}

	// If no valid prefix found, just mask the middle
	if prefixEnd == 0 {
		if len(s) <= minMaskLength {
			return "****" + s[len(s)-visibleSuffix:]
		}
		maskLen := len(s) - visibleSuffix
		return repeatRune('*', maskLen) + s[len(s)-visibleSuffix:]
	}

	// Mask everything between prefix and last 4 chars
	prefix := s[:prefixEnd]
	suffix := s[len(s)-visibleSuffix:]
	maskLen := len(s) - prefixEnd - visibleSuffix
	if maskLen < 0 {
		maskLen = 0
	}

	return prefix + repeatRune('*', maskLen) + suffix
}

// repeatRune creates a string of the given rune repeated n times.
func repeatRune(r rune, n int) string {
	if n <= 0 {
		return ""
	}
	result := make([]rune, n)
	for i := range result {
		result[i] = r
	}
	return string(result)
}

// MaskAuthHeader masks a Basic auth header for safe logging.
// It replaces the base64 encoded credentials with asterisks.
func MaskAuthHeader(header string) string {
	if len(header) > 6 && header[:6] == "Basic " {
		return "Basic ********"
	}
	if len(header) > 7 && header[:7] == "Bearer " {
		return "Bearer ********"
	}
	return "********"
}

// ============================================================================
// Slog Adapter
// ============================================================================

// SlogAdapter adapts a slog.Logger to the StructuredLogger interface.
// This allows seamless integration with Go 1.21+'s structured logging.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.NewSlogAdapter(slog.Default())),
//	)
//
//	// Or with a custom logger:
//	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.NewSlogAdapter(logger)),
//	)
type SlogAdapter struct {
	logger *slog.Logger
}

// NewSlogAdapter creates a new SlogAdapter wrapping the given slog.Logger.
// If logger is nil, slog.Default() is used.
func NewSlogAdapter(logger *slog.Logger) *SlogAdapter {
	if logger == nil {
		logger = slog.Default()
	}
	return &SlogAdapter{logger: logger}
}

// Debug implements StructuredLogger.Debug.
func (a *SlogAdapter) Debug(msg string, args ...any) {
	a.logger.Debug(msg, args...)
}

// Info implements StructuredLogger.Info.
func (a *SlogAdapter) Info(msg string, args ...any) {
	a.logger.Info(msg, args...)
}

// Warn implements StructuredLogger.Warn.
func (a *SlogAdapter) Warn(msg string, args ...any) {
	a.logger.Warn(msg, args...)
}

// Error implements StructuredLogger.Error.
func (a *SlogAdapter) Error(msg string, args ...any) {
	a.logger.Error(msg, args...)
}

// Printf implements Logger.Printf for backward compatibility.
// Logs at Info level with the formatted message.
func (a *SlogAdapter) Printf(format string, v ...any) {
	a.logger.Info(fmt.Sprintf(format, v...))
}

// WithContext returns a new SlogAdapter that uses a logger with the given context.
// This is useful for propagating trace context through logs.
func (a *SlogAdapter) WithContext(ctx context.Context) *SlogAdapter {
	return &SlogAdapter{
		logger: a.logger,
	}
}

// WithGroup returns a new SlogAdapter with a log group prefix.
func (a *SlogAdapter) WithGroup(name string) *SlogAdapter {
	return &SlogAdapter{
		logger: a.logger.WithGroup(name),
	}
}

// With returns a new SlogAdapter with the given attributes added.
func (a *SlogAdapter) With(args ...any) *SlogAdapter {
	return &SlogAdapter{
		logger: a.logger.With(args...),
	}
}
