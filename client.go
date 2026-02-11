package langfuse

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	pkgclient "github.com/jdziat/langfuse-go/pkg/client"
	pkghttp "github.com/jdziat/langfuse-go/pkg/http"
)

// ============================================================================
// API Endpoints
// ============================================================================

// ============================================================================
// Client
// ============================================================================

// defaultStderrLogger is used as a fallback when no logger is configured.
// This ensures async errors are never silently dropped.
var defaultStderrLogger = log.New(os.Stderr, "langfuse: ", log.LstdFlags)

// Client is the main Langfuse client.
// It embeds *pkgclient.Client for core functionality (HTTP, lifecycle, batching)
// and adds sub-clients for the Langfuse API.
type Client struct {
	// Embed core client for HTTP, lifecycle, batching, and backpressure
	*pkgclient.Client

	// Root-specific config (extends pkg/client.Config with evaluation, etc.)
	rootConfig *Config

	// Sub-clients for Langfuse API
	traces       *TracesClient
	observations *ObservationsClient
	scores       *ScoresClient
	prompts      *PromptsClient
	datasets     *DatasetsClient
	sessions     *SessionsClient
	models       *ModelsClient
}

// New creates a new Langfuse client.
func New(publicKey, secretKey string, opts ...ConfigOption) (*Client, error) {
	cfg := &Config{
		PublicKey: publicKey,
		SecretKey: secretKey,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	return NewWithConfig(cfg)
}

// NewWithConfig creates a new Langfuse client from a Config struct.
// This is useful when you want to configure the client using a struct
// rather than functional options.
//
// Example:
//
//	client, err := langfuse.NewWithConfig(&langfuse.Config{
//	    PublicKey: os.Getenv("LANGFUSE_PUBLIC_KEY"),
//	    SecretKey: os.Getenv("LANGFUSE_SECRET_KEY"),
//	    Region:    langfuse.RegionUS,
//	    BatchSize: 50,
//	})
func NewWithConfig(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, ErrNilRequest
	}

	// Make a copy to avoid modifying the original
	cfgCopy := *cfg

	cfgCopy.applyDefaults()

	if err := cfgCopy.validate(); err != nil {
		return nil, err
	}

	// Convert root Config to pkg/client.Config
	pkgCfg := convertToPkgClientConfig(&cfgCopy)

	// Create the core client from pkg/client
	coreClient, err := pkgclient.NewWithConfig(pkgCfg)
	if err != nil {
		return nil, err
	}

	c := &Client{
		Client:     coreClient,
		rootConfig: &cfgCopy,
	}

	// Initialize sub-clients using the embedded client's HTTP() method
	c.traces = newTracesClient(c)
	c.observations = newObservationsClient(c)
	c.scores = newScoresClient(c)
	c.prompts = newPromptsClient(c)
	c.datasets = newDatasetsClient(c)
	c.sessions = newSessionsClient(c)
	c.models = newModelsClient(c)

	return c, nil
}

// convertToPkgClientConfig converts a root Config to a pkg/client.Config.
// This maps all common fields and converts types where needed.
func convertToPkgClientConfig(cfg *Config) *pkgclient.Config {
	pkgCfg := &pkgclient.Config{
		PublicKey:           cfg.PublicKey,
		SecretKey:           cfg.SecretKey,
		BaseURL:             cfg.BaseURL,
		Region:              cfg.Region,
		HTTPClient:          cfg.HTTPClient,
		Timeout:             cfg.Timeout,
		MaxRetries:          cfg.MaxRetries,
		RetryDelay:          cfg.RetryDelay,
		BatchSize:           cfg.BatchSize,
		FlushInterval:       cfg.FlushInterval,
		Debug:               cfg.Debug,
		ErrorHandler:        cfg.ErrorHandler,
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConnsPerHost,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		ShutdownTimeout:     cfg.ShutdownTimeout,
		BatchQueueSize:      cfg.BatchQueueSize,
		IdleWarningDuration: cfg.IdleWarningDuration,
		IDGenerationMode:    pkgclient.IDGenerationMode(cfg.IDGenerationMode),
		BlockOnQueueFull:    cfg.BlockOnQueueFull,
		DropOnQueueFull:     cfg.DropOnQueueFull,
	}

	// Convert Logger interface
	if cfg.Logger != nil {
		pkgCfg.Logger = cfg.Logger
	}

	// Convert StructuredLogger interface
	if cfg.StructuredLogger != nil {
		pkgCfg.StructuredLogger = &structuredLoggerWrapper{cfg.StructuredLogger}
	}

	// Convert Metrics interface
	if cfg.Metrics != nil {
		pkgCfg.Metrics = &metricsWrapper{cfg.Metrics}
	}

	// Convert RetryStrategy
	if cfg.RetryStrategy != nil {
		pkgCfg.RetryStrategy = cfg.RetryStrategy
	}

	// Convert CircuitBreaker config
	if cfg.CircuitBreaker != nil {
		pkgCfg.CircuitBreaker = cfg.CircuitBreaker
	}

	// Convert OnBatchFlushed callback
	if cfg.OnBatchFlushed != nil {
		pkgCfg.OnBatchFlushed = func(result pkgclient.BatchResult) {
			cfg.OnBatchFlushed(BatchResult{
				EventCount: result.EventCount,
				Success:    result.Success,
				Error:      result.Error,
				Duration:   result.Duration,
				Successes:  result.Successes,
				Errors:     result.Errors,
			})
		}
	}

	// Convert HTTP hooks
	if len(cfg.HTTPHooks) > 0 {
		pkgHooks := make([]pkgclient.HTTPHook, len(cfg.HTTPHooks))
		for i, hook := range cfg.HTTPHooks {
			pkgHooks[i] = &httpHookWrapper{hook}
		}
		pkgCfg.HTTPHooks = pkgHooks
	}

	// Convert BackpressureConfig
	if cfg.BackpressureConfig != nil {
		pkgCfg.BackpressureConfig = cfg.BackpressureConfig
	}

	// Convert OnBackpressure callback
	if cfg.OnBackpressure != nil {
		pkgCfg.OnBackpressure = cfg.OnBackpressure
	}

	return pkgCfg
}

// structuredLoggerWrapper wraps root StructuredLogger for pkg/client.
type structuredLoggerWrapper struct {
	logger StructuredLogger
}

func (w *structuredLoggerWrapper) Debug(msg string, args ...any) {
	w.logger.Debug(msg, args...)
}

func (w *structuredLoggerWrapper) Info(msg string, args ...any) {
	w.logger.Info(msg, args...)
}

func (w *structuredLoggerWrapper) Warn(msg string, args ...any) {
	w.logger.Warn(msg, args...)
}

func (w *structuredLoggerWrapper) Error(msg string, args ...any) {
	w.logger.Error(msg, args...)
}

// metricsWrapper wraps root Metrics for pkg/client.
type metricsWrapper struct {
	metrics Metrics
}

func (w *metricsWrapper) IncrementCounter(name string, value int64) {
	w.metrics.IncrementCounter(name, value)
}

func (w *metricsWrapper) RecordDuration(name string, duration time.Duration) {
	w.metrics.RecordDuration(name, duration)
}

func (w *metricsWrapper) SetGauge(name string, value float64) {
	w.metrics.SetGauge(name, value)
}

// httpHookWrapper wraps root HTTPHook for pkg/client.
type httpHookWrapper struct {
	hook HTTPHook
}

func (w *httpHookWrapper) BeforeRequest(ctx context.Context, req *http.Request) error {
	return w.hook.BeforeRequest(ctx, req)
}

func (w *httpHookWrapper) AfterResponse(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error) {
	w.hook.AfterResponse(ctx, req, resp, duration, err)
}

// handleError handles async errors for root-specific functionality.
// For core client errors, the embedded client's handleError is used.
// This method handles errors specific to root-level features.
func (c *Client) handleRootError(err error) {
	handled := false

	if c.rootConfig.ErrorHandler != nil {
		c.rootConfig.ErrorHandler(err)
		handled = true
	}

	if c.rootConfig.StructuredLogger != nil {
		c.rootConfig.StructuredLogger.Error("async error", "error", err)
		handled = true
	} else if c.rootConfig.Logger != nil {
		c.rootConfig.Logger.Printf("error: %v", err)
		handled = true
	}

	if c.rootConfig.Metrics != nil {
		c.rootConfig.Metrics.IncrementCounter("langfuse.errors", 1)
	}

	// Never silently drop errors - log to stderr as fallback
	if !handled {
		defaultStderrLogger.Printf("unhandled async error: %v", err)
	}
}

// Note: log, logInfo, logError methods are provided by the embedded *pkgclient.Client

// Traces returns the traces sub-client.
func (c *Client) Traces() *TracesClient {
	return c.traces
}

// Observations returns the observations sub-client.
func (c *Client) Observations() *ObservationsClient {
	return c.observations
}

// Scores returns the scores sub-client.
func (c *Client) Scores() *ScoresClient {
	return c.scores
}

// Prompts returns the prompts sub-client.
func (c *Client) Prompts() *PromptsClient {
	return c.prompts
}

// Datasets returns the datasets sub-client.
func (c *Client) Datasets() *DatasetsClient {
	return c.datasets
}

// Sessions returns the sessions sub-client.
func (c *Client) Sessions() *SessionsClient {
	return c.sessions
}

// Models returns the models sub-client.
func (c *Client) Models() *ModelsClient {
	return c.models
}

// PromptsWithOptions returns a configured prompts sub-client.
// Options allow setting defaults like labels, versions, and caching.
//
// Example:
//
//	prompts := client.PromptsWithOptions(
//	    langfuse.WithDefaultLabel("production"),
//	    langfuse.WithPromptCaching(5 * time.Minute),
//	)
func (c *Client) PromptsWithOptions(opts ...PromptsOption) *ConfiguredPromptsClient {
	cfg := &promptsConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &ConfiguredPromptsClient{
		PromptsClient: c.prompts,
		config:        cfg,
	}
}

// TracesWithOptions returns a configured traces sub-client.
// Options allow setting default metadata and tags for all traces.
//
// Example:
//
//	traces := client.TracesWithOptions(
//	    langfuse.WithDefaultMetadata(langfuse.Metadata{"env": "prod"}),
//	)
func (c *Client) TracesWithOptions(opts ...TracesOption) *ConfiguredTracesClient {
	cfg := &tracesConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &ConfiguredTracesClient{
		TracesClient: c.traces,
		config:       cfg,
	}
}

// DatasetsWithOptions returns a configured datasets sub-client.
// Options allow setting defaults like page size.
func (c *Client) DatasetsWithOptions(opts ...DatasetsOption) *ConfiguredDatasetsClient {
	cfg := &datasetsConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &ConfiguredDatasetsClient{
		DatasetsClient: c.datasets,
		config:         cfg,
	}
}

// ScoresWithOptions returns a configured scores sub-client.
// Options allow setting defaults like source.
func (c *Client) ScoresWithOptions(opts ...ScoresOption) *ConfiguredScoresClient {
	cfg := &scoresConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &ConfiguredScoresClient{
		ScoresClient: c.scores,
		config:       cfg,
	}
}

// SessionsWithOptions returns a configured sessions sub-client.
// Options allow setting defaults like timeout.
//
// Example:
//
//	sessions := client.SessionsWithOptions(
//	    langfuse.WithSessionsTimeout(10 * time.Second),
//	)
func (c *Client) SessionsWithOptions(opts ...SessionsOption) *ConfiguredSessionsClient {
	cfg := &sessionsConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &ConfiguredSessionsClient{
		SessionsClient: c.sessions,
		config:         cfg,
	}
}

// ModelsWithOptions returns a configured models sub-client.
// Options allow setting defaults like timeout.
//
// Example:
//
//	models := client.ModelsWithOptions(
//	    langfuse.WithModelsTimeout(10 * time.Second),
//	)
func (c *Client) ModelsWithOptions(opts ...ModelsOption) *ConfiguredModelsClient {
	cfg := &modelsConfig{}
	for _, opt := range opts {
		opt(cfg)
	}
	return &ConfiguredModelsClient{
		ModelsClient: c.models,
		config:       cfg,
	}
}

// Note: CircuitBreakerState, IsUnderBackpressure, GenerateID, IDStats methods
// are provided by the embedded *pkgclient.Client

// BackpressureStatus returns the current backpressure state.
// Use this to monitor queue health and make decisions about event submission.
// Note: This wraps the embedded client's method to return the root BackpressureHandlerStats type.
func (c *Client) BackpressureStatus() BackpressureHandlerStats {
	return c.Client.BackpressureStatus()
}

// BackpressureLevel returns the current backpressure level.
// Returns BackpressureNone if no backpressure handler is configured.
// Note: This wraps the embedded client's method.
func (c *Client) BackpressureLevel() BackpressureLevel {
	return c.Client.BackpressureLevel()
}

// ============================================================================
// Client Interfaces
// ============================================================================

// Tracer defines the core tracing interface.
// This interface is implemented by *Client and can be used for
// dependency injection and testing.
//
// Example:
//
//	func ProcessRequest(ctx context.Context, tracer langfuse.Tracer) error {
//	    trace, err := tracer.NewTrace().Name("process-request").Create(ctx)
//	    if err != nil {
//	        return err
//	    }
//	    // ... do work ...
//	    return nil
//	}
type Tracer interface {
	// NewTrace creates a new trace builder.
	NewTrace() *TraceBuilder

	// Flush sends all pending events to Langfuse.
	// It blocks until all events are sent or the context is cancelled.
	Flush(ctx context.Context) error

	// Shutdown gracefully shuts down the client, flushing any pending events.
	// After Shutdown returns, the client should not be used.
	Shutdown(ctx context.Context) error
}

// Ensure Client implements Tracer at compile time.
var _ Tracer = (*Client)(nil)

// Observer defines the interface for creating observations within a trace.
// This interface is implemented by TraceContext, SpanContext, and GenerationContext.
//
// The Advanced API uses builder methods (NewSpan, NewGeneration, etc.) for full control.
// The Simple API uses convenience methods (Span, Generation, etc.) for common use cases.
type Observer interface {
	// ID returns the ID of this trace or observation.
	ID() string

	// TraceID returns the trace ID that this observation belongs to.
	TraceID() string

	// NewSpan creates a new span builder as a child of this context (Advanced API).
	NewSpan() *SpanBuilder

	// NewGeneration creates a new generation builder as a child of this context (Advanced API).
	NewGeneration() *GenerationBuilder

	// NewEvent creates a new event builder as a child of this context (Advanced API).
	NewEvent() *EventBuilder

	// NewScore creates a new score builder for this context (Advanced API).
	NewScore() *ScoreBuilder
}

// Ensure contexts implement Observer at compile time.
var (
	_ Observer = (*TraceContext)(nil)
	_ Observer = (*SpanContext)(nil)
	_ Observer = (*GenerationContext)(nil)
)

// Flusher defines the interface for types that can flush pending data.
type Flusher interface {
	Flush(ctx context.Context) error
}

// Ensure Client implements Flusher at compile time.
var _ Flusher = (*Client)(nil)

// HealthChecker defines the interface for health checking.
type HealthChecker interface {
	Health(ctx context.Context) (*HealthStatus, error)
}

// Ensure Client implements HealthChecker at compile time.
var _ HealthChecker = (*Client)(nil)

// ============================================================================
// HTTP Hooks - Re-exported from pkg/http
// ============================================================================

// HookPriority determines how hook failures are handled.
type HookPriority = pkghttp.HookPriority

const (
	// HookPriorityObservational indicates a hook that should not abort requests on failure.
	HookPriorityObservational = pkghttp.HookPriorityObservational

	// HookPriorityCritical indicates a hook that should abort requests on failure.
	HookPriorityCritical = pkghttp.HookPriorityCritical
)

// HTTPHook allows customizing HTTP request/response handling.
type HTTPHook = pkghttp.HTTPHook

// ClassifiedHook wraps an HTTPHook with priority information.
type ClassifiedHook = pkghttp.ClassifiedHook

// HTTPHookFunc is a function adapter for simple hooks.
type HTTPHookFunc = pkghttp.HTTPHookFunc

// combineHooks combines multiple hooks into a single hook.
var combineHooks = pkghttp.CombineHooks

// ClassifiedHookChain manages multiple hooks with different priorities.
type ClassifiedHookChain = pkghttp.ClassifiedHookChain

// NewClassifiedHookChain creates a new classified hook chain.
func NewClassifiedHookChain(logger Logger, metrics Metrics) *ClassifiedHookChain {
	return pkghttp.NewClassifiedHookChain(logger, metrics)
}

// ============================================================================
// Predefined Hooks - Re-exported from pkg/http
// ============================================================================

// HeaderHook creates a hook that adds custom headers to all requests.
var HeaderHook = pkghttp.HeaderHook

// DynamicHeaderHook creates a hook that adds headers from a function.
var DynamicHeaderHook = pkghttp.DynamicHeaderHook

// LoggingHook creates a hook that logs request and response information.
func LoggingHook(logger Logger) HTTPHook {
	return pkghttp.LoggingHook(logger)
}

// MetricsHook creates a hook that records request metrics.
func MetricsHook(m Metrics) HTTPHook {
	return pkghttp.MetricsHook(m)
}

// TracingHook creates a hook that propagates tracing context.
// It extracts trace IDs from context and adds them as headers.
//
// Headers added:
//   - X-Trace-ID: The trace ID if present in context
//   - X-Parent-Span-ID: The parent span ID if present in context
//
// Example:
//
//	langfuse.WithHTTPHooks(
//	    langfuse.TracingHook(),
//	)
func TracingHook() HTTPHook {
	return HTTPHookFunc{
		Before: func(ctx context.Context, req *http.Request) error {
			// Check for trace context
			if tc, ok := TraceFromContext(ctx); ok {
				req.Header.Set("X-Langfuse-Trace-ID", tc.ID())
			}
			return nil
		},
	}
}

// DebugHook creates a hook that logs detailed request/response information.
func DebugHook(logger Logger) HTTPHook {
	return pkghttp.DebugHook(logger)
}

// ============================================================================
// Classified Hook Constructors - Re-exported from pkg/http
// ============================================================================

// ObservationalLoggingHook creates a logging hook that won't abort requests on failure.
func ObservationalLoggingHook(logger Logger) ClassifiedHook {
	return pkghttp.ObservationalLoggingHook(logger)
}

// ObservationalMetricsHook creates a metrics hook that won't abort requests on failure.
func ObservationalMetricsHook(m Metrics) ClassifiedHook {
	return pkghttp.ObservationalMetricsHook(m)
}

// ObservationalTracingHook creates a tracing hook that won't abort requests on failure.
// This hook uses TraceFromContext to propagate Langfuse trace IDs.
func ObservationalTracingHook() ClassifiedHook {
	return ClassifiedHook{
		Hook:     TracingHook(),
		Priority: HookPriorityObservational,
		Name:     "tracing",
	}
}

// ObservationalDebugHook creates a debug hook that won't abort requests on failure.
func ObservationalDebugHook(logger Logger) ClassifiedHook {
	return pkghttp.ObservationalDebugHook(logger)
}

// CriticalHeaderHook creates a header hook that aborts requests on failure.
var CriticalHeaderHook = pkghttp.CriticalHeaderHook

// CriticalAuthHook creates an authentication hook that aborts requests on failure.
var CriticalAuthHook = pkghttp.CriticalAuthHook

// CriticalValidationHook creates a validation hook that aborts requests on failure.
var CriticalValidationHook = pkghttp.CriticalValidationHook

// NewClassifiedHook creates a ClassifiedHook with the given parameters.
var NewClassifiedHook = pkghttp.NewClassifiedHook

// ============================================================================
// Retry Strategy - Re-exports from pkg/http
// ============================================================================

// RetryStrategy defines how failed requests are retried.
type RetryStrategy = pkghttp.RetryStrategy

// RetryStrategyWithError is an optional extension of RetryStrategy that
// allows the retry delay to be influenced by the error.
// If a strategy implements this interface, RetryDelayWithError is called
// instead of RetryDelay, allowing it to use Retry-After headers.
type RetryStrategyWithError = pkghttp.RetryStrategyWithError

// ExponentialBackoff implements exponential backoff with optional jitter.
type ExponentialBackoff = pkghttp.ExponentialBackoff

// NewExponentialBackoff creates a new exponential backoff strategy with defaults.
var NewExponentialBackoff = pkghttp.NewExponentialBackoff

// NoRetry is a retry strategy that never retries.
type NoRetry = pkghttp.NoRetry

// FixedDelay is a retry strategy with a fixed delay between retries.
type FixedDelay = pkghttp.FixedDelay

// NewFixedDelay creates a new fixed delay retry strategy.
func NewFixedDelay(delay time.Duration, maxRetries int) *FixedDelay {
	return pkghttp.NewFixedDelay(delay, maxRetries)
}

// LinearBackoff is a retry strategy with linearly increasing delays.
type LinearBackoff = pkghttp.LinearBackoff

// NewLinearBackoff creates a new linear backoff retry strategy.
func NewLinearBackoff(initialDelay, increment time.Duration, maxRetries int) *LinearBackoff {
	return pkghttp.NewLinearBackoff(initialDelay, increment, maxRetries)
}

// ============================================================================
// Circuit Breaker - Re-exports from pkg/http
// ============================================================================

// CircuitState represents the state of a circuit breaker.
type CircuitState = pkghttp.CircuitState

// Circuit breaker states.
const (
	// CircuitClosed allows requests to pass through normally.
	CircuitClosed = pkghttp.CircuitClosed
	// CircuitOpen blocks all requests immediately.
	CircuitOpen = pkghttp.CircuitOpen
	// CircuitHalfOpen allows a limited number of requests to test if the service recovered.
	CircuitHalfOpen = pkghttp.CircuitHalfOpen
)

// ErrCircuitOpen is returned when the circuit breaker is open and blocking requests.
var ErrCircuitOpen = errors.New("langfuse: circuit breaker is open")

// CircuitBreakerConfig configures the circuit breaker behavior.
type CircuitBreakerConfig = pkghttp.CircuitBreakerConfig

// DefaultCircuitBreakerConfig returns a CircuitBreakerConfig with sensible defaults.
var DefaultCircuitBreakerConfig = pkghttp.DefaultCircuitBreakerConfig

// CircuitBreaker implements the circuit breaker pattern for fault tolerance.
type CircuitBreaker = pkghttp.CircuitBreaker

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
var NewCircuitBreaker = pkghttp.NewCircuitBreaker

// CircuitBreakerOption configures a circuit breaker.
type CircuitBreakerOption = pkghttp.CircuitBreakerOption

// WithFailureThreshold sets the failure threshold.
func WithFailureThreshold(n int) CircuitBreakerOption {
	return pkghttp.WithFailureThreshold(n)
}

// WithSuccessThreshold sets the success threshold for half-open state.
func WithSuccessThreshold(n int) CircuitBreakerOption {
	return pkghttp.WithSuccessThreshold(n)
}

// WithCircuitTimeout sets the timeout before transitioning from open to half-open.
func WithCircuitTimeout(d time.Duration) CircuitBreakerOption {
	return pkghttp.WithCircuitTimeout(d)
}

// WithHalfOpenMaxRequests sets the max requests allowed in half-open state.
func WithHalfOpenMaxRequests(n int) CircuitBreakerOption {
	return pkghttp.WithHalfOpenMaxRequests(n)
}

// WithStateChangeCallback sets the callback for state changes.
func WithStateChangeCallback(fn func(from, to CircuitState)) CircuitBreakerOption {
	return pkghttp.WithStateChangeCallback(fn)
}

// WithFailureChecker sets a custom function to determine if an error is a failure.
func WithFailureChecker(fn func(err error) bool) CircuitBreakerOption {
	return pkghttp.WithFailureChecker(fn)
}

// NewCircuitBreakerWithOptions creates a circuit breaker with functional options.
var NewCircuitBreakerWithOptions = pkghttp.NewCircuitBreakerWithOptions

// ============================================================================
// HTTP Client Implementation
// ============================================================================

// Ensure httpClient implements pkghttp.Doer at compile time.
var _ pkghttp.Doer = (*httpClient)(nil)

const (
	// maxResponseSize limits the size of HTTP response bodies to prevent OOM.
	maxResponseSize = 10 * 1024 * 1024 // 10MB
)

// httpClient handles HTTP requests to the Langfuse API.
type httpClient struct {
	client         *http.Client
	baseURL        string
	authHeader     string
	maxRetries     int
	retryDelay     time.Duration
	retryStrategy  RetryStrategy
	debug          bool
	circuitBreaker *CircuitBreaker
	hook           HTTPHook
}

// newHTTPClient creates a new HTTP client.
func newHTTPClient(cfg *Config) *httpClient {
	auth := base64.StdEncoding.EncodeToString([]byte(cfg.PublicKey + ":" + cfg.SecretKey))

	// Use the provided retry strategy or create a default one
	retryStrategy := cfg.RetryStrategy
	if retryStrategy == nil {
		retryStrategy = &ExponentialBackoff{
			InitialDelay: cfg.RetryDelay,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       true,
			MaxRetries:   cfg.MaxRetries,
		}
	}

	h := &httpClient{
		client:        cfg.HTTPClient,
		baseURL:       strings.TrimSuffix(cfg.BaseURL, "/"),
		authHeader:    "Basic " + auth,
		maxRetries:    cfg.MaxRetries,
		retryDelay:    cfg.RetryDelay,
		retryStrategy: retryStrategy,
		debug:         cfg.Debug,
		hook:          combineHooks(cfg.HTTPHooks),
	}

	// Initialize circuit breaker if configured
	if cfg.CircuitBreaker != nil {
		h.circuitBreaker = NewCircuitBreaker(*cfg.CircuitBreaker)
	}

	return h
}

// request represents an HTTP request to be made.
type request struct {
	method string
	path   string
	query  url.Values
	body   any
	result any
}

// do executes an HTTP request with retries and optional circuit breaker protection.
func (h *httpClient) do(ctx context.Context, req *request) error {
	// Wrap with circuit breaker if configured
	if h.circuitBreaker != nil {
		return h.circuitBreaker.Execute(func() error {
			return h.doWithRetries(ctx, req)
		})
	}
	return h.doWithRetries(ctx, req)
}

// doWithRetries executes an HTTP request with retries.
func (h *httpClient) doWithRetries(ctx context.Context, req *request) error {
	for attempt := 0; ; attempt++ {
		err := h.doOnce(ctx, req)
		if err == nil {
			return nil
		}

		// Check if we should retry using the retry strategy
		if !h.retryStrategy.ShouldRetry(attempt, err) {
			return err
		}

		// Get the delay from the retry strategy
		// Use RetryDelayWithError if available (supports Retry-After headers)
		var delay time.Duration
		if strategyWithErr, ok := h.retryStrategy.(RetryStrategyWithError); ok {
			delay = strategyWithErr.RetryDelayWithError(attempt, err)
		} else {
			delay = h.retryStrategy.RetryDelay(attempt)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

// doOnce executes a single HTTP request.
func (h *httpClient) doOnce(ctx context.Context, req *request) error {
	// Build URL
	u := h.baseURL + req.path
	if len(req.query) > 0 {
		u += "?" + req.query.Encode()
	}

	// Build body
	var bodyReader io.Reader
	if req.body != nil {
		bodyBytes, err := json.Marshal(req.body)
		if err != nil {
			return fmt.Errorf("langfuse: failed to marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, req.method, u, bodyReader)
	if err != nil {
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	// Generate request ID for tracing
	requestID := generateRequestID()

	// Set headers
	httpReq.Header.Set("Authorization", h.authHeader)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "langfuse-go/"+Version)
	httpReq.Header.Set("X-Request-ID", requestID)

	// Check if context has a request ID override
	if ctxRequestID, ok := ctx.Value(requestIDContextKey{}).(string); ok && ctxRequestID != "" {
		requestID = ctxRequestID
		httpReq.Header.Set("X-Request-ID", requestID)
	}

	// Call BeforeRequest hook
	if h.hook != nil {
		if err := h.hook.BeforeRequest(ctx, httpReq); err != nil {
			return fmt.Errorf("langfuse: hook BeforeRequest failed: %w", err)
		}
	}

	// Track request timing for hooks
	startTime := time.Now()

	// Execute request
	resp, err := h.client.Do(httpReq)

	// Calculate duration for hooks
	duration := time.Since(startTime)

	// Call AfterResponse hook (even on error)
	if h.hook != nil {
		h.hook.AfterResponse(ctx, httpReq, resp, duration, err)
	}

	if err != nil {
		return fmt.Errorf("langfuse: request failed (request_id=%s): %w", requestID, err)
	}
	defer resp.Body.Close()

	// Read response body with size limit to prevent OOM
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return fmt.Errorf("langfuse: failed to read response body (request_id=%s): %w", requestID, err)
	}
	if len(respBody) > maxResponseSize {
		return fmt.Errorf("langfuse: response body exceeded maximum size of %d bytes (request_id=%s)", maxResponseSize, requestID)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		apiErr := &APIError{
			StatusCode: resp.StatusCode,
			RequestID:  requestID,
		}
		if len(respBody) > 0 {
			// Attempt to parse error response body. If parsing fails,
			// we still return the APIError with status code and request ID.
			// Store the raw body in the message if JSON parsing fails.
			if err := json.Unmarshal(respBody, apiErr); err != nil {
				// Parsing failed - include raw body as message for debugging
				apiErr.Message = string(respBody)
			}
		}

		// Parse Retry-After header for rate limit responses
		if resp.StatusCode == 429 {
			apiErr.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
		}

		return apiErr
	}

	// Parse response
	if req.result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, req.result); err != nil {
			return fmt.Errorf("langfuse: failed to unmarshal response (request_id=%s): %w", requestID, err)
		}
	}

	return nil
}

// requestIDContextKey is the context key for request IDs.
type requestIDContextKey struct{}

// WithRequestID returns a context with the given request ID.
// This ID will be sent to the Langfuse API and can be used for debugging.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// generateRequestID generates a unique request ID.
func generateRequestID() string {
	id, err := UUID()
	if err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return id
}

// parseRetryAfter parses the Retry-After header value.
// It supports both seconds (integer) and HTTP-date formats.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}

	// Try parsing as seconds (integer)
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP-date (RFC 7231)
	if t, err := http.ParseTime(value); err == nil {
		return time.Until(t)
	}

	return 0
}

// get performs a GET request.
func (h *httpClient) get(ctx context.Context, path string, query url.Values, result any) error {
	return h.do(ctx, &request{
		method: http.MethodGet,
		path:   path,
		query:  query,
		result: result,
	})
}

// post performs a POST request.
func (h *httpClient) post(ctx context.Context, path string, body any, result any) error {
	return h.do(ctx, &request{
		method: http.MethodPost,
		path:   path,
		body:   body,
		result: result,
	})
}

// delete performs a DELETE request.
func (h *httpClient) delete(ctx context.Context, path string, result any) error {
	return h.do(ctx, &request{
		method: http.MethodDelete,
		path:   path,
		result: result,
	})
}

// Get performs an HTTP GET request (implements http.Doer).
func (h *httpClient) Get(ctx context.Context, path string, query url.Values, result any) error {
	return h.get(ctx, path, query, result)
}

// Post performs an HTTP POST request (implements http.Doer).
func (h *httpClient) Post(ctx context.Context, path string, body, result any) error {
	return h.post(ctx, path, body, result)
}

// Delete performs an HTTP DELETE request (implements http.Doer).
func (h *httpClient) Delete(ctx context.Context, path string, result any) error {
	return h.delete(ctx, path, result)
}

// ============================================================================
// Pagination Helpers
// ============================================================================

// PaginationParams represents pagination parameters for list requests.
type PaginationParams struct {
	Page   int
	Limit  int
	Cursor string
}

// ToQuery converts pagination parameters to URL query values.
func (p *PaginationParams) ToQuery() url.Values {
	q := url.Values{}
	if p.Page > 0 {
		q.Set("page", strconv.Itoa(p.Page))
	}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Cursor != "" {
		q.Set("cursor", p.Cursor)
	}
	return q
}

// PaginatedResponse represents a paginated response.
type PaginatedResponse struct {
	Meta MetaResponse `json:"meta"`
}

// MetaResponse represents pagination metadata.
type MetaResponse struct {
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	TotalItems int    `json:"totalItems"`
	TotalPages int    `json:"totalPages"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// HasMore returns true if there are more pages.
func (m *MetaResponse) HasMore() bool {
	return m.NextCursor != "" || m.Page < m.TotalPages
}

// FilterParams represents common filter parameters.
type FilterParams struct {
	Name          string
	UserID        string
	Type          string
	TraceID       string
	SessionID     string
	Level         string
	Version       string
	Environment   string
	FromStartTime time.Time
	ToStartTime   time.Time
	Tags          []string
}

// ToQuery converts filter parameters to URL query values.
func (f *FilterParams) ToQuery() url.Values {
	q := url.Values{}
	if f.Name != "" {
		q.Set("name", f.Name)
	}
	if f.UserID != "" {
		q.Set("userId", f.UserID)
	}
	if f.Type != "" {
		q.Set("type", f.Type)
	}
	if f.TraceID != "" {
		q.Set("traceId", f.TraceID)
	}
	if f.SessionID != "" {
		q.Set("sessionId", f.SessionID)
	}
	if f.Level != "" {
		q.Set("level", f.Level)
	}
	if f.Version != "" {
		q.Set("version", f.Version)
	}
	if f.Environment != "" {
		q.Set("environment", f.Environment)
	}
	if !f.FromStartTime.IsZero() {
		q.Set("fromStartTime", f.FromStartTime.Format(time.RFC3339))
	}
	if !f.ToStartTime.IsZero() {
		q.Set("toStartTime", f.ToStartTime.Format(time.RFC3339))
	}
	for _, tag := range f.Tags {
		q.Add("tags", tag)
	}
	return q
}

// mergeQuery merges multiple url.Values into one.
func mergeQuery(queries ...url.Values) url.Values {
	result := url.Values{}
	for _, q := range queries {
		for k, v := range q {
			for _, val := range v {
				result.Add(k, val)
			}
		}
	}
	return result
}
