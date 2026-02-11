package client

import (
	"net/http"
	"time"

	pkghttp "github.com/jdziat/langfuse-go/pkg/http"
)

// ConfigOption configures the client.
type ConfigOption func(*Config)

// WithBaseURL sets the base URL for the Langfuse API.
func WithBaseURL(url string) ConfigOption {
	return func(c *Config) {
		c.BaseURL = url
	}
}

// WithRegion sets the Langfuse cloud region.
func WithRegion(region Region) ConfigOption {
	return func(c *Config) {
		c.Region = region
	}
}

// WithHTTPClient sets a custom HTTP client.
func WithHTTPClient(client *http.Client) ConfigOption {
	return func(c *Config) {
		c.HTTPClient = client
	}
}

// WithTimeout sets the request timeout.
func WithTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.Timeout = timeout
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func WithMaxRetries(maxRetries int) ConfigOption {
	return func(c *Config) {
		c.MaxRetries = maxRetries
	}
}

// WithRetryDelay sets the initial delay between retry attempts.
func WithRetryDelay(delay time.Duration) ConfigOption {
	return func(c *Config) {
		c.RetryDelay = delay
	}
}

// WithBatchSize sets the maximum number of events per batch.
func WithBatchSize(size int) ConfigOption {
	return func(c *Config) {
		c.BatchSize = size
	}
}

// WithFlushInterval sets the interval for flushing pending events.
func WithFlushInterval(interval time.Duration) ConfigOption {
	return func(c *Config) {
		c.FlushInterval = interval
	}
}

// WithDebug enables debug logging.
func WithDebug(debug bool) ConfigOption {
	return func(c *Config) {
		c.Debug = debug
	}
}

// WithErrorHandler sets the error handler for async operations.
func WithErrorHandler(handler func(error)) ConfigOption {
	return func(c *Config) {
		c.ErrorHandler = handler
	}
}

// WithLogger sets the logger for SDK logging.
func WithLogger(logger Logger) ConfigOption {
	return func(c *Config) {
		c.Logger = logger
	}
}

// WithStructuredLogger sets the structured logger.
func WithStructuredLogger(logger StructuredLogger) ConfigOption {
	return func(c *Config) {
		c.StructuredLogger = logger
	}
}

// WithMetrics sets the metrics collector.
func WithMetrics(metrics Metrics) ConfigOption {
	return func(c *Config) {
		c.Metrics = metrics
	}
}

// WithShutdownTimeout sets the graceful shutdown timeout.
func WithShutdownTimeout(timeout time.Duration) ConfigOption {
	return func(c *Config) {
		c.ShutdownTimeout = timeout
	}
}

// WithBatchQueueSize sets the size of the background batch queue.
func WithBatchQueueSize(size int) ConfigOption {
	return func(c *Config) {
		c.BatchQueueSize = size
	}
}

// WithRetryStrategy sets the retry strategy.
func WithRetryStrategy(strategy pkghttp.RetryStrategy) ConfigOption {
	return func(c *Config) {
		c.RetryStrategy = strategy
	}
}

// WithCircuitBreaker configures the circuit breaker.
func WithCircuitBreaker(cfg *pkghttp.CircuitBreakerConfig) ConfigOption {
	return func(c *Config) {
		c.CircuitBreaker = cfg
	}
}

// WithOnBatchFlushed sets the callback for batch flush events.
func WithOnBatchFlushed(fn func(BatchResult)) ConfigOption {
	return func(c *Config) {
		c.OnBatchFlushed = fn
	}
}

// WithIdleWarningDuration sets the idle warning duration.
func WithIdleWarningDuration(duration time.Duration) ConfigOption {
	return func(c *Config) {
		c.IdleWarningDuration = duration
	}
}

// WithIDGenerationMode sets the ID generation mode.
func WithIDGenerationMode(mode IDGenerationMode) ConfigOption {
	return func(c *Config) {
		c.IDGenerationMode = mode
	}
}

// WithBlockOnQueueFull enables blocking when the queue is full.
func WithBlockOnQueueFull(block bool) ConfigOption {
	return func(c *Config) {
		c.BlockOnQueueFull = block
	}
}

// WithDropOnQueueFull enables dropping events when the queue is full.
func WithDropOnQueueFull(drop bool) ConfigOption {
	return func(c *Config) {
		c.DropOnQueueFull = drop
	}
}
