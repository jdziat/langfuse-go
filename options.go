package langfuse

import (
	"net/http"
	"time"
)

// ConfigOption is a function that modifies a Config.
type ConfigOption func(*Config)

// WithRegion sets the Langfuse cloud region.
func WithRegion(region Region) ConfigOption {
	return func(c *Config) {
		c.Region = region
	}
}

// WithBaseURL sets a custom base URL for the Langfuse API.
func WithBaseURL(baseURL string) ConfigOption {
	return func(c *Config) {
		c.BaseURL = baseURL
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

// WithBatchSize sets the maximum batch size for ingestion.
func WithBatchSize(size int) ConfigOption {
	return func(c *Config) {
		c.BatchSize = size
	}
}

// WithFlushInterval sets the flush interval for batched events.
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

// WithErrorHandler sets an error callback for async failures.
func WithErrorHandler(handler func(error)) ConfigOption {
	return func(c *Config) {
		c.ErrorHandler = handler
	}
}

// WithLogger sets a custom logger (printf-style).
//
// Deprecated: Use WithStructuredLogger with WrapPrintfLogger instead:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.WrapPrintfLogger(myLogger)),
//	)
//
// Or for standard library loggers:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.WrapStdLogger(log.Default())),
//	)
func WithLogger(logger Logger) ConfigOption {
	return func(c *Config) {
		c.Logger = logger
	}
}

// WithStructuredLogger sets a structured logger.
// This takes precedence over Logger set via WithLogger.
//
// Example with slog:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStructuredLogger(langfuse.NewSlogAdapter(slog.Default())),
//	)
func WithStructuredLogger(logger StructuredLogger) ConfigOption {
	return func(c *Config) {
		c.StructuredLogger = logger
	}
}

// WithMetrics sets a metrics collector.
func WithMetrics(metrics Metrics) ConfigOption {
	return func(c *Config) {
		c.Metrics = metrics
	}
}

// WithMaxIdleConns sets the maximum number of idle connections.
func WithMaxIdleConns(n int) ConfigOption {
	return func(c *Config) {
		c.MaxIdleConns = n
	}
}

// WithMaxIdleConnsPerHost sets the maximum number of idle connections per host.
func WithMaxIdleConnsPerHost(n int) ConfigOption {
	return func(c *Config) {
		c.MaxIdleConnsPerHost = n
	}
}

// WithIdleConnTimeout sets the idle connection timeout.
func WithIdleConnTimeout(d time.Duration) ConfigOption {
	return func(c *Config) {
		c.IdleConnTimeout = d
	}
}

// WithShutdownTimeout sets the graceful shutdown timeout.
func WithShutdownTimeout(d time.Duration) ConfigOption {
	return func(c *Config) {
		c.ShutdownTimeout = d
	}
}

// WithBatchQueueSize sets the size of the background batch queue.
func WithBatchQueueSize(size int) ConfigOption {
	return func(c *Config) {
		c.BatchQueueSize = size
	}
}

// WithRetryStrategy sets the retry strategy for failed requests.
func WithRetryStrategy(strategy RetryStrategy) ConfigOption {
	return func(c *Config) {
		c.RetryStrategy = strategy
	}
}

// WithCircuitBreaker enables circuit breaker protection for API calls.
// The circuit breaker prevents cascading failures by failing fast when
// the Langfuse API is unhealthy.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithCircuitBreaker(langfuse.CircuitBreakerConfig{
//	        FailureThreshold: 5,
//	        Timeout:          30 * time.Second,
//	    }),
//	)
func WithCircuitBreaker(config CircuitBreakerConfig) ConfigOption {
	return func(c *Config) {
		c.CircuitBreaker = &config
	}
}

// WithDefaultCircuitBreaker enables circuit breaker with default settings.
// Use this for quick setup with sensible defaults.
func WithDefaultCircuitBreaker() ConfigOption {
	return func(c *Config) {
		config := DefaultCircuitBreakerConfig()
		c.CircuitBreaker = &config
	}
}

// WithOnBatchFlushed sets a callback that is called after each batch is sent.
// This is useful for monitoring, logging, or custom error handling.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithOnBatchFlushed(func(result langfuse.BatchResult) {
//	        if !result.Success {
//	            log.Printf("Batch failed: %v", result.Error)
//	        } else {
//	            log.Printf("Sent %d events in %v", result.EventCount, result.Duration)
//	        }
//	    }),
//	)
func WithOnBatchFlushed(callback func(BatchResult)) ConfigOption {
	return func(c *Config) {
		c.OnBatchFlushed = callback
	}
}

// WithHTTPHooks sets HTTP hooks for request/response customization.
// Hooks are called in order before requests and in reverse order after responses.
//
// Use hooks for:
//   - Adding custom headers to all requests
//   - Logging request/response details
//   - Collecting custom metrics
//   - Implementing custom retry logic
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithHTTPHooks(
//	        langfuse.LoggingHook(log.Default()),
//	        langfuse.HeaderHook(map[string]string{"X-Custom": "value"}),
//	    ),
//	)
func WithHTTPHooks(hooks ...HTTPHook) ConfigOption {
	return func(c *Config) {
		c.HTTPHooks = append(c.HTTPHooks, hooks...)
	}
}

// WithIdleWarning enables warnings when the client is idle without shutdown.
// This helps detect goroutine leaks in development and testing.
//
// When enabled, if no activity occurs for the specified duration and Shutdown()
// hasn't been called, a warning is logged. This catches the common mistake of
// creating clients without proper cleanup.
//
// Recommended values:
//   - Development: 5 * time.Minute
//   - Testing: 30 * time.Second
//   - Production: 0 (disabled) - rely on proper shutdown handling
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithIdleWarning(5*time.Minute),
//	)
func WithIdleWarning(duration time.Duration) ConfigOption {
	return func(c *Config) {
		c.IdleWarningDuration = duration
	}
}

// WithIDGenerationMode sets the ID generation failure mode.
//
// IDModeFallback (default): Uses an atomic counter fallback when crypto/rand fails.
// This ensures IDs are always generated but may produce less random IDs under
// extreme resource exhaustion.
//
// IDModeStrict: Returns an error when crypto/rand fails. Recommended for
// production deployments where ID uniqueness is critical and crypto failures
// should surface as errors rather than degrade silently.
//
// Example:
//
//	// For production with strict ID requirements
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithIDGenerationMode(langfuse.IDModeStrict),
//	)
func WithIDGenerationMode(mode IDGenerationMode) ConfigOption {
	return func(c *Config) {
		c.IDGenerationMode = mode
	}
}

// WithClassifiedHooks sets priority-aware HTTP hooks with differentiated error handling.
// Critical hooks block on errors; Observational hooks log errors but continue.
//
// Use classified hooks when you need fine-grained control over hook error handling.
// If both HTTPHooks and ClassifiedHooks are set, ClassifiedHooks take precedence.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithClassifiedHooks(
//	        langfuse.ObservationalLoggingHook(log.Default()),
//	        langfuse.CriticalAuthHook(authFunc),
//	    ),
//	)
func WithClassifiedHooks(hooks ...ClassifiedHook) ConfigOption {
	return func(c *Config) {
		c.ClassifiedHooks = append(c.ClassifiedHooks, hooks...)
	}
}

// WithAsyncErrorConfig sets the async error handler configuration.
// This provides structured error handling for all background operations.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithAsyncErrorConfig(&langfuse.AsyncErrorConfig{
//	        BufferSize: 200,
//	        OnError: func(err *langfuse.AsyncError) {
//	            log.Printf("Async error: %v", err)
//	        },
//	    }),
//	)
func WithAsyncErrorConfig(config *AsyncErrorConfig) ConfigOption {
	return func(c *Config) {
		c.AsyncErrorConfig = config
	}
}

// WithOnAsyncError sets a callback for async errors.
// This is a convenience method for simple error handling needs.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithOnAsyncError(func(err *langfuse.AsyncError) {
//	        log.Printf("Async error [%s]: %v", err.Operation, err.Err)
//	    }),
//	)
func WithOnAsyncError(handler func(*AsyncError)) ConfigOption {
	return func(c *Config) {
		c.OnAsyncError = handler
	}
}

// WithBackpressureConfig sets the backpressure handler configuration.
// This provides proactive notification when the event queue is filling up.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithBackpressureConfig(&langfuse.BackpressureHandlerConfig{
//	        Monitor: langfuse.NewQueueMonitor(&langfuse.QueueMonitorConfig{
//	            Threshold: langfuse.DefaultBackpressureThreshold(),
//	            Capacity:  1000,
//	        }),
//	        BlockOnFull: true,
//	    }),
//	)
func WithBackpressureConfig(config *BackpressureHandlerConfig) ConfigOption {
	return func(c *Config) {
		c.BackpressureConfig = config
	}
}

// WithOnBackpressure sets a callback for backpressure level changes.
// This is a convenience method for monitoring queue health.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithOnBackpressure(func(state langfuse.QueueState) {
//	        if state.Level >= langfuse.BackpressureCritical {
//	            log.Printf("Queue critical: %d/%d (%.1f%%)",
//	                state.Size, state.Capacity, state.PercentFull)
//	        }
//	    }),
//	)
func WithOnBackpressure(callback BackpressureCallback) ConfigOption {
	return func(c *Config) {
		c.OnBackpressure = callback
	}
}

// WithBackpressureThreshold sets custom thresholds for backpressure levels.
// This is a convenience method that creates a backpressure handler with the
// specified thresholds.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithBackpressureThreshold(langfuse.BackpressureThreshold{
//	        WarningPercent:  40.0,  // Warn at 40% full
//	        CriticalPercent: 70.0,  // Critical at 70% full
//	        OverflowPercent: 90.0,  // Overflow at 90% full
//	    }),
//	)
func WithBackpressureThreshold(threshold BackpressureThreshold) ConfigOption {
	return func(c *Config) {
		if c.BackpressureConfig == nil {
			c.BackpressureConfig = &BackpressureHandlerConfig{}
		}
		if c.BackpressureConfig.Monitor == nil {
			c.BackpressureConfig.Monitor = NewQueueMonitor(&QueueMonitorConfig{
				Threshold: threshold,
			})
		}
	}
}

// WithBlockOnQueueFull enables blocking when the event queue is at overflow.
// When enabled, event submission will block until space is available.
// This prevents event loss but may slow down the caller.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithBlockOnQueueFull(true),
//	)
func WithBlockOnQueueFull(block bool) ConfigOption {
	return func(c *Config) {
		c.BlockOnQueueFull = block
	}
}

// WithDropOnQueueFull enables dropping events when the queue is at overflow.
// This prevents blocking but events will be silently dropped.
// Only applies when BlockOnQueueFull is false.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithDropOnQueueFull(true),
//	)
func WithDropOnQueueFull(drop bool) ConfigOption {
	return func(c *Config) {
		c.DropOnQueueFull = drop
	}
}

// WithStrictValidation enables strict validation mode.
// When enabled, validated builders accumulate errors and force explicit
// error handling via BuildResult types.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStrictValidation(langfuse.StrictValidationConfig{
//	        Enabled:  true,
//	        FailFast: false, // Accumulate all errors
//	    }),
//	)
func WithStrictValidation(config StrictValidationConfig) ConfigOption {
	return func(c *Config) {
		c.StrictValidation = &config
	}
}

// WithStrictValidationEnabled is a convenience function to enable strict
// validation with default settings (all errors accumulated).
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithStrictValidationEnabled(),
//	)
//
//	// Now use validated builders:
//	trace, err := client.NewTraceStrict().
//	    Name("my-trace").
//	    Create(ctx).
//	    Unwrap()
//	if err != nil {
//	    log.Printf("Validation failed: %v", err)
//	}
func WithStrictValidationEnabled() ConfigOption {
	return func(c *Config) {
		config := DefaultStrictValidationConfig()
		c.StrictValidation = &config
	}
}

// WithMetricsRecorder enables the internal metrics recorder.
// This collects detailed SDK internal metrics and exports them via the
// configured Metrics interface. Requires WithMetrics to be set.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithMetrics(myMetricsImpl),
//	    langfuse.WithMetricsRecorder(),
//	)
func WithMetricsRecorder() ConfigOption {
	return func(c *Config) {
		c.EnableMetricsRecorder = true
	}
}
