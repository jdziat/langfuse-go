package client

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	pkgconfig "github.com/jdziat/langfuse-go/pkg/config"
	pkghttp "github.com/jdziat/langfuse-go/pkg/http"
	pkgingestion "github.com/jdziat/langfuse-go/pkg/ingestion"
)

// Re-export Region from pkg/config
type Region = pkgconfig.Region

// Region constants
const (
	RegionEU    = pkgconfig.RegionEU
	RegionUS    = pkgconfig.RegionUS
	RegionHIPAA = pkgconfig.RegionHIPAA
)

// Default configuration values
const (
	DefaultTimeout               = pkgconfig.DefaultTimeout
	DefaultMaxRetries            = pkgconfig.DefaultMaxRetries
	DefaultRetryDelay            = pkgconfig.DefaultRetryDelay
	DefaultBatchSize             = pkgconfig.DefaultBatchSize
	DefaultFlushInterval         = pkgconfig.DefaultFlushInterval
	DefaultMaxIdleConns          = pkgconfig.DefaultMaxIdleConns
	DefaultMaxIdleConnsPerHost   = pkgconfig.DefaultMaxIdleConnsPerHost
	DefaultIdleConnTimeout       = pkgconfig.DefaultIdleConnTimeout
	DefaultShutdownTimeout       = pkgconfig.DefaultShutdownTimeout
	DefaultBatchQueueSize        = pkgconfig.DefaultBatchQueueSize
	DefaultBackgroundSendTimeout = pkgconfig.DefaultBackgroundSendTimeout
	MaxBatchSize                 = pkgconfig.MaxBatchSize
	MaxMaxRetries                = pkgconfig.MaxMaxRetries
	MaxTimeout                   = pkgconfig.MaxTimeout
	MinFlushInterval             = pkgconfig.MinFlushInterval
	MinShutdownTimeout           = pkgconfig.MinShutdownTimeout
	MinKeyLength                 = pkgconfig.MinKeyLength
	PublicKeyPrefix              = pkgconfig.PublicKeyPrefix
	SecretKeyPrefix              = pkgconfig.SecretKeyPrefix
)

// Logger is a minimal logging interface for the SDK.
type Logger interface {
	Printf(format string, v ...any)
}

// StructuredLogger provides structured logging support.
type StructuredLogger interface {
	Debug(msg string, args ...any)
	Info(msg string, args ...any)
	Warn(msg string, args ...any)
	Error(msg string, args ...any)
}

// Metrics is an interface for SDK telemetry.
type Metrics interface {
	IncrementCounter(name string, value int64)
	RecordDuration(name string, duration time.Duration)
	SetGauge(name string, value float64)
}

// HTTPHook allows customizing HTTP request/response handling.
type HTTPHook interface {
	BeforeRequest(ctx context.Context, req *http.Request) error
	AfterResponse(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error)
}

// ClassifiedHook wraps an HTTPHook with priority information.
type ClassifiedHook struct {
	Hook     HTTPHook
	Priority int
	Name     string
}

// Config holds the configuration for the Langfuse client.
type Config struct {
	// PublicKey is the Langfuse public key (required).
	PublicKey string

	// SecretKey is the Langfuse secret key (required).
	SecretKey string

	// BaseURL is the base URL for the Langfuse API.
	BaseURL string

	// Region is the Langfuse cloud region.
	Region Region

	// HTTPClient is the HTTP client to use for requests.
	HTTPClient *http.Client

	// Timeout is the request timeout.
	Timeout time.Duration

	// MaxRetries is the maximum number of retry attempts.
	MaxRetries int

	// RetryDelay is the initial delay between retry attempts.
	RetryDelay time.Duration

	// BatchSize is the maximum number of events per batch.
	BatchSize int

	// FlushInterval is the interval for flushing pending events.
	FlushInterval time.Duration

	// Debug enables debug logging.
	Debug bool

	// ErrorHandler is called when async operations fail.
	ErrorHandler func(error)

	// Logger is used for SDK logging.
	Logger Logger

	// StructuredLogger is used for structured SDK logging.
	StructuredLogger StructuredLogger

	// Metrics is used for SDK telemetry.
	Metrics Metrics

	// MaxIdleConns controls the maximum number of idle connections.
	MaxIdleConns int

	// MaxIdleConnsPerHost controls the maximum idle connections per host.
	MaxIdleConnsPerHost int

	// IdleConnTimeout is how long idle connections are kept.
	IdleConnTimeout time.Duration

	// ShutdownTimeout is the maximum time for graceful shutdown.
	ShutdownTimeout time.Duration

	// BatchQueueSize is the size of the background batch queue.
	BatchQueueSize int

	// RetryStrategy is the strategy for retrying failed requests.
	RetryStrategy pkghttp.RetryStrategy

	// CircuitBreaker configures fault tolerance for API calls.
	CircuitBreaker *pkghttp.CircuitBreakerConfig

	// OnBatchFlushed is called after each batch is sent.
	OnBatchFlushed func(result BatchResult)

	// HTTPHooks are called before and after each HTTP request.
	HTTPHooks []HTTPHook

	// ClassifiedHooks are priority-aware HTTP hooks.
	ClassifiedHooks []ClassifiedHook

	// IdleWarningDuration triggers idle warnings.
	IdleWarningDuration time.Duration

	// IDGenerationMode controls ID generation behavior.
	IDGenerationMode IDGenerationMode

	// BackpressureConfig configures backpressure handling.
	BackpressureConfig *pkgingestion.BackpressureHandlerConfig

	// OnBackpressure is called when backpressure level changes.
	OnBackpressure pkgingestion.BackpressureCallback

	// BlockOnQueueFull blocks when queue is at overflow.
	BlockOnQueueFull bool

	// DropOnQueueFull drops events when queue is at overflow.
	DropOnQueueFull bool
}

// IDGenerationMode controls how IDs are generated.
type IDGenerationMode int

const (
	// IDModeFallback falls back to timestamp-based IDs on crypto failure.
	IDModeFallback IDGenerationMode = iota
	// IDModeStrict returns errors on crypto failure.
	IDModeStrict
)

// BatchResult contains information about a flushed batch.
type BatchResult struct {
	EventCount int
	Success    bool
	Error      error
	Duration   time.Duration
	Successes  int
	Errors     int
}

// ApplyDefaults sets default values for unset configuration options.
func (c *Config) ApplyDefaults() {
	if c.BaseURL == "" {
		if c.Region == "" {
			c.Region = RegionEU
		}
		if url, ok := pkgconfig.RegionBaseURLs[c.Region]; ok {
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

// Validate checks that the configuration is valid.
func (c *Config) Validate() error {
	if c.PublicKey == "" {
		return ErrMissingPublicKey
	}
	if c.SecretKey == "" {
		return ErrMissingSecretKey
	}
	if c.BaseURL == "" {
		return ErrMissingBaseURL
	}

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

	if _, err := url.Parse(c.BaseURL); err != nil {
		return fmt.Errorf("langfuse: invalid base URL: %w", err)
	}

	if c.BatchSize < 1 || c.BatchSize > MaxBatchSize {
		return fmt.Errorf("langfuse: batch size must be between 1 and %d", MaxBatchSize)
	}
	if c.MaxRetries < 0 || c.MaxRetries > MaxMaxRetries {
		return fmt.Errorf("langfuse: max retries must be between 0 and %d", MaxMaxRetries)
	}
	if c.BatchQueueSize < 1 {
		return fmt.Errorf("langfuse: batch queue size must be at least 1")
	}

	return nil
}

// String returns a string representation with masked credentials.
func (c *Config) String() string {
	return fmt.Sprintf("Config{PublicKey: %q, BaseURL: %q, Region: %q, BatchSize: %d}",
		maskCredential(c.PublicKey),
		c.BaseURL,
		c.Region,
		c.BatchSize,
	)
}

// maskCredential masks a credential for safe logging.
func maskCredential(s string) string {
	if len(s) <= 8 {
		return "****"
	}
	return s[:6] + "****" + s[len(s)-4:]
}

// defaultLogger wraps the standard library logger.
type defaultLogger struct {
	logger *log.Logger
}

func (l *defaultLogger) Printf(format string, v ...any) {
	l.logger.Printf(format, v...)
}

