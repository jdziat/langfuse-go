package langfuse

import (
	"context"
	"net/http"
	"strconv"
	"sync"
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

// WithMaxBackgroundSenders sets the maximum number of concurrent goroutines
// used for background batch sending when the queue is full.
// This prevents unbounded goroutine creation under sustained high load.
// Default is 10.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithMaxBackgroundSenders(20),
//	)
func WithMaxBackgroundSenders(n int) ConfigOption {
	return func(c *Config) {
		c.MaxBackgroundSenders = n
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

// WithEvaluationMode enables automatic evaluation structuring for LLM-as-a-Judge.
// When enabled, traces are automatically prepared for evaluation without
// requiring manual JSONPath configuration in Langfuse.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
//	)
//
// Or with RAGAS-specific formatting:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithEvaluationMode(langfuse.EvaluationModeRAGAS),
//	)
func WithEvaluationMode(mode EvaluationMode) ConfigOption {
	return func(c *Config) {
		if c.EvaluationConfig == nil {
			c.EvaluationConfig = DefaultEvaluationConfig()
		}
		c.EvaluationConfig.Mode = mode
	}
}

// WithEvaluationConfig sets a complete evaluation configuration.
// Use this for fine-grained control over evaluation behavior.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithEvaluationConfig(&langfuse.EvaluationConfig{
//	        Mode:            langfuse.EvaluationModeAuto,
//	        DefaultWorkflow: langfuse.WorkflowRAG,
//	        AutoValidate:    true,
//	        IncludeMetadata: true,
//	        IncludeTags:     true,
//	    }),
//	)
func WithEvaluationConfig(config *EvaluationConfig) ConfigOption {
	return func(c *Config) {
		c.EvaluationConfig = config
	}
}

// WithDefaultWorkflow sets the default workflow type for evaluation.
// This helps the SDK understand what fields to expect and structure.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
//	    langfuse.WithDefaultWorkflow(langfuse.WorkflowRAG),
//	)
func WithDefaultWorkflow(workflow WorkflowType) ConfigOption {
	return func(c *Config) {
		if c.EvaluationConfig == nil {
			c.EvaluationConfig = DefaultEvaluationConfig()
		}
		c.EvaluationConfig.DefaultWorkflow = workflow
	}
}

// WithTargetEvaluators sets the evaluators to optimize traces for.
// The SDK will validate that required fields are present before completion.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithEvaluationMode(langfuse.EvaluationModeAuto),
//	    langfuse.WithTargetEvaluators(
//	        langfuse.EvaluatorFaithfulness,
//	        langfuse.EvaluatorHallucination,
//	    ),
//	)
func WithTargetEvaluators(evaluators ...EvaluatorType) ConfigOption {
	return func(c *Config) {
		if c.EvaluationConfig == nil {
			c.EvaluationConfig = DefaultEvaluationConfig()
		}
		c.EvaluationConfig.TargetEvaluators = evaluators
	}
}

// WithRAGASEvaluation enables evaluation mode optimized for RAGAS metrics.
// This is a convenience function that sets up the optimal configuration
// for Faithfulness, Answer Relevance, Context Precision, and Context Recall.
//
// Example:
//
//	client, _ := langfuse.New(pk, sk,
//	    langfuse.WithRAGASEvaluation(),
//	)
func WithRAGASEvaluation() ConfigOption {
	return func(c *Config) {
		c.EvaluationConfig = RAGASEvaluationConfig()
	}
}

// ============================================================================
// Sub-Client Options
// ============================================================================

// SubClientOption is a function type for configuring sub-clients.
// This is the base type that specific sub-client option types can extend.
type SubClientOption func(interface{})

// PromptsOption is a functional option for configuring the PromptsClient.
type PromptsOption func(*promptsConfig)

type promptsConfig struct {
	defaultLabel   string
	defaultVersion int
	cacheTTL       time.Duration
	cacheEnabled   bool
}

// WithDefaultLabel sets a default label for all prompt lookups.
// This is applied when no label is explicitly provided to Get methods.
//
// Example:
//
//	prompts := client.PromptsWithOptions(
//	    langfuse.WithDefaultLabel("production"),
//	)
//	// All Get calls will use "production" label unless overridden
//	prompt, _ := prompts.Get(ctx, "my-prompt", nil)
func WithDefaultLabel(label string) PromptsOption {
	return func(c *promptsConfig) {
		c.defaultLabel = label
	}
}

// WithDefaultVersion sets a default version for prompt lookups.
// This is applied when no version is explicitly provided.
//
// Example:
//
//	prompts := client.PromptsWithOptions(
//	    langfuse.WithDefaultVersion(2),
//	)
func WithDefaultVersion(version int) PromptsOption {
	return func(c *promptsConfig) {
		c.defaultVersion = version
	}
}

// WithPromptCaching enables prompt caching with the specified TTL.
// Cached prompts are stored in memory and reused for subsequent requests.
// Use TTL of 0 to cache indefinitely until client shutdown.
//
// Example:
//
//	prompts := client.PromptsWithOptions(
//	    langfuse.WithPromptCaching(5 * time.Minute),
//	)
func WithPromptCaching(ttl time.Duration) PromptsOption {
	return func(c *promptsConfig) {
		c.cacheEnabled = true
		c.cacheTTL = ttl
	}
}

// TracesOption is a functional option for configuring the TracesClient.
type TracesOption func(*tracesConfig)

type tracesConfig struct {
	defaultMetadata Metadata
	defaultTags     []string
}

// WithDefaultMetadata sets default metadata for all traces.
// This metadata is merged with any metadata provided to individual traces.
//
// Example:
//
//	traces := client.TracesWithOptions(
//	    langfuse.WithDefaultMetadata(langfuse.Metadata{
//	        "service": "my-service",
//	        "version": "1.0",
//	    }),
//	)
func WithDefaultMetadata(metadata Metadata) TracesOption {
	return func(c *tracesConfig) {
		c.defaultMetadata = metadata
	}
}

// WithDefaultTags sets default tags for all traces.
// These tags are appended to any tags provided to individual traces.
//
// Example:
//
//	traces := client.TracesWithOptions(
//	    langfuse.WithDefaultTags([]string{"production", "v1"}),
//	)
func WithDefaultTags(tags []string) TracesOption {
	return func(c *tracesConfig) {
		c.defaultTags = tags
	}
}

// DatasetsOption is a functional option for configuring the DatasetsClient.
type DatasetsOption func(*datasetsConfig)

type datasetsConfig struct {
	defaultPageSize int
}

// WithDefaultPageSize sets the default page size for list operations.
//
// Example:
//
//	datasets := client.DatasetsWithOptions(
//	    langfuse.WithDefaultPageSize(100),
//	)
func WithDefaultPageSize(size int) DatasetsOption {
	return func(c *datasetsConfig) {
		c.defaultPageSize = size
	}
}

// ScoresOption is a functional option for configuring the ScoresClient.
type ScoresOption func(*scoresConfig)

type scoresConfig struct {
	defaultSource string
}

// WithDefaultSource sets a default source for all scores.
//
// Example:
//
//	scores := client.ScoresWithOptions(
//	    langfuse.WithDefaultSource("evaluation-pipeline"),
//	)
func WithDefaultSource(source string) ScoresOption {
	return func(c *scoresConfig) {
		c.defaultSource = source
	}
}

// ModelsOption is a functional option for configuring the ModelsClient.
type ModelsOption func(*modelsConfig)

type modelsConfig struct {
	defaultTimeout time.Duration
}

// WithModelsTimeout sets a default timeout for all model operations.
// This timeout is applied to context when not already set.
//
// Example:
//
//	models := client.ModelsWithOptions(
//	    langfuse.WithModelsTimeout(10 * time.Second),
//	)
func WithModelsTimeout(timeout time.Duration) ModelsOption {
	return func(c *modelsConfig) {
		c.defaultTimeout = timeout
	}
}

// SessionsOption is a functional option for configuring the SessionsClient.
type SessionsOption func(*sessionsConfig)

type sessionsConfig struct {
	defaultTimeout time.Duration
}

// WithSessionsTimeout sets a default timeout for all session operations.
// This timeout is applied to context when not already set.
//
// Example:
//
//	sessions := client.SessionsWithOptions(
//	    langfuse.WithSessionsTimeout(10 * time.Second),
//	)
func WithSessionsTimeout(timeout time.Duration) SessionsOption {
	return func(c *sessionsConfig) {
		c.defaultTimeout = timeout
	}
}

// ============================================================================
// Configured Sub-Client Wrappers
// ============================================================================

// ConfiguredPromptsClient wraps PromptsClient with configured defaults.
type ConfiguredPromptsClient struct {
	*PromptsClient
	config *promptsConfig

	// Cache for prompts
	cacheMu sync.RWMutex
	cache   map[string]cachedPrompt
}

type cachedPrompt struct {
	prompt    *Prompt
	expiresAt time.Time
}

// Get retrieves a prompt by name, applying configured defaults.
// If caching is enabled, cached prompts are returned when available.
func (c *ConfiguredPromptsClient) Get(ctx context.Context, name string, params *GetPromptParams) (*Prompt, error) {
	// Apply defaults
	effectiveParams := c.applyDefaults(params)

	// Check cache if enabled
	if c.config.cacheEnabled {
		if prompt := c.getFromCache(name, effectiveParams); prompt != nil {
			return prompt, nil
		}
	}

	// Fetch from API
	prompt, err := c.PromptsClient.Get(ctx, name, effectiveParams)
	if err != nil {
		return nil, err
	}

	// Cache if enabled
	if c.config.cacheEnabled {
		c.addToCache(name, effectiveParams, prompt)
	}

	return prompt, nil
}

func (c *ConfiguredPromptsClient) applyDefaults(params *GetPromptParams) *GetPromptParams {
	if params == nil {
		params = &GetPromptParams{}
	}

	// Only apply defaults if not explicitly set
	if params.Label == "" && c.config.defaultLabel != "" {
		params.Label = c.config.defaultLabel
	}
	if params.Version == 0 && c.config.defaultVersion > 0 {
		params.Version = c.config.defaultVersion
	}

	return params
}

func (c *ConfiguredPromptsClient) getCacheKey(name string, params *GetPromptParams) string {
	key := name
	if params != nil {
		if params.Label != "" {
			key += ":label=" + params.Label
		}
		if params.Version > 0 {
			key += ":version=" + strconv.Itoa(params.Version)
		}
	}
	return key
}

func (c *ConfiguredPromptsClient) getFromCache(name string, params *GetPromptParams) *Prompt {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	if c.cache == nil {
		return nil
	}

	key := c.getCacheKey(name, params)
	cached, ok := c.cache[key]
	if !ok {
		return nil
	}

	// Check expiration (TTL of 0 means no expiration)
	if c.config.cacheTTL > 0 && time.Now().After(cached.expiresAt) {
		return nil
	}

	return cached.prompt
}

func (c *ConfiguredPromptsClient) addToCache(name string, params *GetPromptParams, prompt *Prompt) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	if c.cache == nil {
		c.cache = make(map[string]cachedPrompt)
	}

	key := c.getCacheKey(name, params)
	expiresAt := time.Time{} // Zero time means no expiration
	if c.config.cacheTTL > 0 {
		expiresAt = time.Now().Add(c.config.cacheTTL)
	}

	c.cache[key] = cachedPrompt{
		prompt:    prompt,
		expiresAt: expiresAt,
	}
}

// ClearCache clears the prompt cache.
func (c *ConfiguredPromptsClient) ClearCache() {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.cache = nil
}

// CacheSize returns the number of cached prompts.
func (c *ConfiguredPromptsClient) CacheSize() int {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	return len(c.cache)
}

// ConfiguredTracesClient wraps TracesClient with configured defaults.
type ConfiguredTracesClient struct {
	*TracesClient
	config *tracesConfig
}

// Get retrieves a trace by ID.
func (c *ConfiguredTracesClient) Get(ctx context.Context, id string) (*Trace, error) {
	return c.TracesClient.Get(ctx, id)
}

// DefaultMetadata returns the configured default metadata.
func (c *ConfiguredTracesClient) DefaultMetadata() Metadata {
	return c.config.defaultMetadata
}

// DefaultTags returns the configured default tags.
func (c *ConfiguredTracesClient) DefaultTags() []string {
	return c.config.defaultTags
}

// ConfiguredDatasetsClient wraps DatasetsClient with configured defaults.
type ConfiguredDatasetsClient struct {
	*DatasetsClient
	config *datasetsConfig
}

// DefaultPageSize returns the configured default page size.
func (c *ConfiguredDatasetsClient) DefaultPageSize() int {
	return c.config.defaultPageSize
}

// ConfiguredScoresClient wraps ScoresClient with configured defaults.
type ConfiguredScoresClient struct {
	*ScoresClient
	config *scoresConfig
}

// DefaultSource returns the configured default source.
func (c *ConfiguredScoresClient) DefaultSource() string {
	return c.config.defaultSource
}

// ConfiguredModelsClient wraps ModelsClient with configured defaults.
type ConfiguredModelsClient struct {
	*ModelsClient
	config *modelsConfig
}

// List retrieves a list of models, applying configured defaults.
func (c *ConfiguredModelsClient) List(ctx context.Context, params *ModelsListParams) (*ModelsListResponse, error) {
	ctx = c.applyTimeout(ctx)
	return c.ModelsClient.List(ctx, params)
}

// Get retrieves a model by ID, applying configured defaults.
func (c *ConfiguredModelsClient) Get(ctx context.Context, modelID string) (*Model, error) {
	ctx = c.applyTimeout(ctx)
	return c.ModelsClient.Get(ctx, modelID)
}

// Create creates a new model definition, applying configured defaults.
func (c *ConfiguredModelsClient) Create(ctx context.Context, req *CreateModelRequest) (*Model, error) {
	ctx = c.applyTimeout(ctx)
	return c.ModelsClient.Create(ctx, req)
}

// Delete deletes a model by ID, applying configured defaults.
func (c *ConfiguredModelsClient) Delete(ctx context.Context, modelID string) error {
	ctx = c.applyTimeout(ctx)
	return c.ModelsClient.Delete(ctx, modelID)
}

func (c *ConfiguredModelsClient) applyTimeout(ctx context.Context) context.Context {
	if c.config.defaultTimeout > 0 {
		// Only apply timeout if context doesn't already have a deadline
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, c.config.defaultTimeout)
			_ = cancel // We're intentionally not canceling here as the caller owns the context lifecycle
		}
	}
	return ctx
}

// ConfiguredSessionsClient wraps SessionsClient with configured defaults.
type ConfiguredSessionsClient struct {
	*SessionsClient
	config *sessionsConfig
}

// List retrieves a list of sessions, applying configured defaults.
func (c *ConfiguredSessionsClient) List(ctx context.Context, params *SessionsListParams) (*SessionsListResponse, error) {
	ctx = c.applyTimeout(ctx)
	return c.SessionsClient.List(ctx, params)
}

// Get retrieves a session by ID, applying configured defaults.
func (c *ConfiguredSessionsClient) Get(ctx context.Context, sessionID string) (*Session, error) {
	ctx = c.applyTimeout(ctx)
	return c.SessionsClient.Get(ctx, sessionID)
}

// GetWithTraces retrieves a session with all its traces, applying configured defaults.
func (c *ConfiguredSessionsClient) GetWithTraces(ctx context.Context, sessionID string) (*SessionWithTraces, error) {
	ctx = c.applyTimeout(ctx)
	return c.SessionsClient.GetWithTraces(ctx, sessionID)
}

func (c *ConfiguredSessionsClient) applyTimeout(ctx context.Context) context.Context {
	if c.config.defaultTimeout > 0 {
		// Only apply timeout if context doesn't already have a deadline
		if _, hasDeadline := ctx.Deadline(); !hasDeadline {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, c.config.defaultTimeout)
			_ = cancel // We're intentionally not canceling here as the caller owns the context lifecycle
		}
	}
	return ctx
}

// ============================================================================
// Unified Observation Options
// ============================================================================
//
// These options work across all observation types (Span, Generation, Event).
// They provide a convenient way to set common properties without needing
// type-specific option functions.
//
// Example:
//
//	span, _ := trace.Span(ctx, "preprocessing",
//	    langfuse.Input(data),
//	    langfuse.Metadata(langfuse.M{"step": 1}),
//	    langfuse.Level(langfuse.ObservationLevelDebug))
//
//	gen, _ := trace.Generation(ctx, "llm-call",
//	    langfuse.Input(prompt),
//	    langfuse.Output(response),
//	    langfuse.Metadata(langfuse.M{"model": "gpt-4"}))

// ObservationOption is an option that can be applied to any observation type.
// It implements SpanOption, GenerationOption, and EventOption.
type ObservationOption interface {
	SpanOption
	GenerationOption
	EventOption
}

// Input sets the input for any observation type.
// This is a unified option that works with Span, Generation, and Event.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process", langfuse.Input(data))
//	gen, _ := trace.Generation(ctx, "llm", langfuse.Input(prompt))
func Input(input any) ObservationOption {
	return &unifiedInputOption{input: input}
}

type unifiedInputOption struct {
	input any
}

func (o *unifiedInputOption) apply(c *spanConfig)        { c.input = o.input }
func (o *unifiedInputOption) apply2(c *generationConfig) { c.input = o.input }
func (o *unifiedInputOption) apply3(c *eventConfig)      { c.input = o.input }

// Ensure unifiedInputOption implements all option interfaces
var _ SpanOption = (*unifiedInputOption)(nil)
var _ GenerationOption = (*unifiedInputOption)(nil)
var _ EventOption = (*unifiedInputOption)(nil)

// Output sets the output for any observation type.
// This is a unified option that works with Span, Generation, and Event.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process", langfuse.Output(result))
//	gen, _ := trace.Generation(ctx, "llm", langfuse.Output(response))
func Output(output any) ObservationOption {
	return &unifiedOutputOption{output: output}
}

type unifiedOutputOption struct {
	output any
}

func (o *unifiedOutputOption) apply(c *spanConfig)        { c.output = o.output }
func (o *unifiedOutputOption) apply2(c *generationConfig) { c.output = o.output }
func (o *unifiedOutputOption) apply3(c *eventConfig)      { c.output = o.output }

var _ SpanOption = (*unifiedOutputOption)(nil)
var _ GenerationOption = (*unifiedOutputOption)(nil)
var _ EventOption = (*unifiedOutputOption)(nil)

// ObsMetadata sets the metadata for any observation type.
// This is a unified option that works with Span, Generation, and Event.
// Named ObsMetadata to avoid conflict with the Metadata type alias.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process",
//	    langfuse.ObsMetadata(langfuse.M{"step": 1}))
func ObsMetadata(metadata Metadata) ObservationOption {
	return &unifiedMetadataOption{metadata: metadata}
}

type unifiedMetadataOption struct {
	metadata Metadata
}

func (o *unifiedMetadataOption) apply(c *spanConfig)        { c.metadata = o.metadata }
func (o *unifiedMetadataOption) apply2(c *generationConfig) { c.metadata = o.metadata }
func (o *unifiedMetadataOption) apply3(c *eventConfig)      { c.metadata = o.metadata }

var _ SpanOption = (*unifiedMetadataOption)(nil)
var _ GenerationOption = (*unifiedMetadataOption)(nil)
var _ EventOption = (*unifiedMetadataOption)(nil)

// ObsLevel sets the observation level for any observation type.
// This is a unified option that works with Span, Generation, and Event.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process",
//	    langfuse.ObsLevel(langfuse.ObservationLevelDebug))
func ObsLevel(level ObservationLevel) ObservationOption {
	return &unifiedLevelOption{level: level}
}

type unifiedLevelOption struct {
	level ObservationLevel
}

func (o *unifiedLevelOption) apply(c *spanConfig) {
	c.level = o.level
	c.hasLevel = true
}
func (o *unifiedLevelOption) apply2(c *generationConfig) {
	c.level = o.level
	c.hasLevel = true
}
func (o *unifiedLevelOption) apply3(c *eventConfig) {
	c.level = o.level
	c.hasLevel = true
}

var _ SpanOption = (*unifiedLevelOption)(nil)
var _ GenerationOption = (*unifiedLevelOption)(nil)
var _ EventOption = (*unifiedLevelOption)(nil)

// StatusMessage sets the status message for any observation type.
// This is a unified option that works with Span, Generation, and Event.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process",
//	    langfuse.StatusMessage("processing complete"))
func StatusMessage(msg string) ObservationOption {
	return &unifiedStatusMessageOption{msg: msg}
}

type unifiedStatusMessageOption struct {
	msg string
}

func (o *unifiedStatusMessageOption) apply(c *spanConfig)        { c.statusMessage = o.msg }
func (o *unifiedStatusMessageOption) apply2(c *generationConfig) { c.statusMessage = o.msg }
func (o *unifiedStatusMessageOption) apply3(c *eventConfig)      { c.statusMessage = o.msg }

var _ SpanOption = (*unifiedStatusMessageOption)(nil)
var _ GenerationOption = (*unifiedStatusMessageOption)(nil)
var _ EventOption = (*unifiedStatusMessageOption)(nil)

// ObsVersion sets the version for any observation type.
// This is a unified option that works with Span, Generation, and Event.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process", langfuse.ObsVersion("1.0.0"))
func ObsVersion(version string) ObservationOption {
	return &unifiedVersionOption{version: version}
}

type unifiedVersionOption struct {
	version string
}

func (o *unifiedVersionOption) apply(c *spanConfig)        { c.version = o.version }
func (o *unifiedVersionOption) apply2(c *generationConfig) { c.version = o.version }
func (o *unifiedVersionOption) apply3(c *eventConfig)      { c.version = o.version }

var _ SpanOption = (*unifiedVersionOption)(nil)
var _ GenerationOption = (*unifiedVersionOption)(nil)
var _ EventOption = (*unifiedVersionOption)(nil)

// ObsEnvironment sets the environment for any observation type.
// This is a unified option that works with Span, Generation, and Event.
//
// Example:
//
//	span, _ := trace.Span(ctx, "process", langfuse.ObsEnvironment("production"))
func ObsEnvironment(env string) ObservationOption {
	return &unifiedEnvironmentOption{env: env}
}

type unifiedEnvironmentOption struct {
	env string
}

func (o *unifiedEnvironmentOption) apply(c *spanConfig)        { c.environment = o.env }
func (o *unifiedEnvironmentOption) apply2(c *generationConfig) { c.environment = o.env }
func (o *unifiedEnvironmentOption) apply3(c *eventConfig)      { c.environment = o.env }

var _ SpanOption = (*unifiedEnvironmentOption)(nil)
var _ GenerationOption = (*unifiedEnvironmentOption)(nil)
var _ EventOption = (*unifiedEnvironmentOption)(nil)

// ObsStartTime sets the start time for any observation type.
// This is a unified option that works with Span, Generation, and Event.
//
// Example:
//
//	startedAt := time.Now()
//	// ... do work ...
//	span, _ := trace.Span(ctx, "process", langfuse.ObsStartTime(startedAt))
func ObsStartTime(t time.Time) ObservationOption {
	return &unifiedStartTimeOption{t: t}
}

type unifiedStartTimeOption struct {
	t time.Time
}

func (o *unifiedStartTimeOption) apply(c *spanConfig) {
	c.startTime = o.t
	c.hasStartTime = true
}
func (o *unifiedStartTimeOption) apply2(c *generationConfig) {
	c.startTime = o.t
	c.hasStartTime = true
}
func (o *unifiedStartTimeOption) apply3(c *eventConfig) {
	c.startTime = o.t
	c.hasStartTime = true
}

var _ SpanOption = (*unifiedStartTimeOption)(nil)
var _ GenerationOption = (*unifiedStartTimeOption)(nil)
var _ EventOption = (*unifiedStartTimeOption)(nil)
