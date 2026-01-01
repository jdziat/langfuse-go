package langfuse

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Default configuration values.
const (
	// DefaultTimeout is the default request timeout.
	DefaultTimeout = 30 * time.Second

	// DefaultMaxRetries is the default maximum number of retry attempts.
	DefaultMaxRetries = 3

	// DefaultRetryDelay is the default initial delay between retry attempts.
	DefaultRetryDelay = 1 * time.Second

	// DefaultBatchSize is the default maximum number of events per batch.
	DefaultBatchSize = 100

	// DefaultFlushInterval is the default interval for flushing pending events.
	DefaultFlushInterval = 5 * time.Second

	// DefaultMaxIdleConns is the default maximum number of idle connections.
	DefaultMaxIdleConns = 100

	// DefaultMaxIdleConnsPerHost is the default maximum idle connections per host.
	DefaultMaxIdleConnsPerHost = 10

	// DefaultIdleConnTimeout is the default timeout for idle connections.
	DefaultIdleConnTimeout = 90 * time.Second

	// DefaultShutdownTimeout is the default graceful shutdown timeout.
	// Must be >= DefaultTimeout to allow pending requests to complete.
	DefaultShutdownTimeout = 35 * time.Second

	// DefaultBatchQueueSize is the default size of the background batch queue.
	DefaultBatchQueueSize = 100

	// DefaultBackgroundSendTimeout is the timeout for background batch sends.
	DefaultBackgroundSendTimeout = 30 * time.Second

	// MaxBatchSize is the maximum allowed batch size.
	MaxBatchSize = 10000

	// MaxMaxRetries is the maximum allowed retry count.
	MaxMaxRetries = 100

	// MaxTimeout is the maximum allowed request timeout.
	MaxTimeout = 10 * time.Minute

	// MinFlushInterval is the minimum allowed flush interval.
	MinFlushInterval = 100 * time.Millisecond

	// MinShutdownTimeout is the minimum allowed shutdown timeout.
	MinShutdownTimeout = 1 * time.Second

	// MinKeyLength is the minimum length for API keys.
	MinKeyLength = 8

	// PublicKeyPrefix is the expected prefix for public keys.
	PublicKeyPrefix = "pk-"

	// SecretKeyPrefix is the expected prefix for secret keys.
	SecretKeyPrefix = "sk-"
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
type BatchResult struct {
	// EventCount is the number of events in the batch.
	EventCount int

	// Success indicates whether the batch was sent successfully.
	Success bool

	// Error is the error that occurred, if any.
	Error error

	// Duration is how long the batch send took.
	Duration time.Duration

	// Successes is the number of successfully ingested events.
	Successes int

	// Errors is the number of events that failed to ingest.
	Errors int
}

// applyDefaults sets default values for unset configuration options.
func (c *Config) applyDefaults() {
	if c.BaseURL == "" {
		if c.Region == "" {
			c.Region = RegionEU
		}
		if url, ok := regionBaseURLs[c.Region]; ok {
			c.BaseURL = url
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
