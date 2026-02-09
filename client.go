package langfuse

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
)

// defaultStderrLogger is used as a fallback when no logger is configured.
// This ensures async errors are never silently dropped.
var defaultStderrLogger = log.New(os.Stderr, "langfuse: ", log.LstdFlags)

// Client is the main Langfuse client.
type Client struct {
	config *Config
	http   *httpClient

	// Lifecycle management (goroutine leak prevention)
	lifecycle   *LifecycleManager
	idGenerator *IDGenerator

	// Batching with proper lifecycle management
	mu            sync.Mutex
	pendingEvents []ingestionEvent
	closed        bool

	// Background goroutine management
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	batchQueue chan batchRequest
	stopFlush  chan struct{}

	// Graceful shutdown signaling
	drainSignal   chan struct{} // Signal batch processor to drain
	drainComplete chan struct{} // Batch processor signals drain complete

	// Backpressure management
	backpressure *BackpressureHandler

	// Sub-clients
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

	httpClient := newHTTPClient(&cfgCopy)

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize lifecycle manager for goroutine leak detection
	lifecycle := NewLifecycleManager(&LifecycleConfig{
		IdleWarningDuration: cfgCopy.IdleWarningDuration,
		Logger:              cfgCopy.Logger,
		Metrics:             cfgCopy.Metrics,
	})

	// Initialize ID generator with configured mode
	idGenerator := NewIDGenerator(&IDGeneratorConfig{
		Mode:    cfgCopy.IDGenerationMode,
		Metrics: cfgCopy.Metrics,
		Logger:  cfgCopy.Logger,
	})

	// Initialize backpressure handler
	var backpressureHandler *BackpressureHandler
	if cfgCopy.BackpressureConfig != nil {
		backpressureHandler = NewBackpressureHandler(cfgCopy.BackpressureConfig)
	} else {
		// Create default backpressure handler using queue capacity
		queueCapacity := cfgCopy.BatchSize * cfgCopy.BatchQueueSize
		backpressureHandler = NewBackpressureHandler(&BackpressureHandlerConfig{
			Monitor: NewQueueMonitor(&QueueMonitorConfig{
				Threshold:      DefaultBackpressureThreshold(),
				Capacity:       queueCapacity,
				OnBackpressure: cfgCopy.OnBackpressure,
				Metrics:        cfgCopy.Metrics,
				Logger:         cfgCopy.Logger,
			}),
			BlockOnFull: cfgCopy.BlockOnQueueFull,
			DropOnFull:  cfgCopy.DropOnQueueFull,
			Logger:      cfgCopy.Logger,
			Metrics:     cfgCopy.Metrics,
		})
	}

	c := &Client{
		config:        &cfgCopy,
		http:          httpClient,
		lifecycle:     lifecycle,
		idGenerator:   idGenerator,
		pendingEvents: make([]ingestionEvent, 0, cfgCopy.BatchSize),
		ctx:           ctx,
		cancel:        cancel,
		batchQueue:    make(chan batchRequest, cfgCopy.BatchQueueSize),
		stopFlush:     make(chan struct{}),
		drainSignal:   make(chan struct{}),
		drainComplete: make(chan struct{}),
		backpressure:  backpressureHandler,
	}

	// Initialize sub-clients
	c.traces = &TracesClient{client: c}
	c.observations = &ObservationsClient{client: c}
	c.scores = &ScoresClient{client: c}
	c.prompts = &PromptsClient{client: c}
	c.datasets = &DatasetsClient{client: c}
	c.sessions = &SessionsClient{client: c}
	c.models = &ModelsClient{client: c}

	// Start background batch processor
	c.wg.Add(1)
	go c.batchProcessor()

	// Start flush timer
	c.wg.Add(1)
	go c.flushLoop()

	return c, nil
}

// handleError handles async errors.
// Errors are NEVER silently dropped - if no handler is configured,
// they are logged to stderr as a fallback.
func (c *Client) handleError(err error) {
	handled := false

	if c.config.ErrorHandler != nil {
		c.config.ErrorHandler(err)
		handled = true
	}

	if c.config.StructuredLogger != nil {
		c.config.StructuredLogger.Error("async error", "error", err)
		handled = true
	} else if c.config.Logger != nil {
		c.config.Logger.Printf("error: %v", err)
		handled = true
	}

	if c.config.Metrics != nil {
		c.config.Metrics.IncrementCounter("langfuse.errors", 1)
	}

	// Never silently drop errors - log to stderr as fallback
	if !handled {
		defaultStderrLogger.Printf("unhandled async error: %v", err)
	}
}

// log logs a message if logging is enabled.
func (c *Client) log(format string, v ...any) {
	if c.config.StructuredLogger != nil {
		c.config.StructuredLogger.Debug(fmt.Sprintf(format, v...))
	} else if c.config.Logger != nil {
		c.config.Logger.Printf(format, v...)
	}
}

// logInfo logs an info-level message.
func (c *Client) logInfo(msg string, args ...any) {
	if c.config.StructuredLogger != nil {
		c.config.StructuredLogger.Info(msg, args...)
	} else if c.config.Logger != nil {
		c.config.Logger.Printf(msg + formatArgs(args))
	}
}

// logError logs an error-level message.
func (c *Client) logError(msg string, args ...any) {
	if c.config.StructuredLogger != nil {
		c.config.StructuredLogger.Error(msg, args...)
	} else if c.config.Logger != nil {
		c.config.Logger.Printf("[ERROR] " + msg + formatArgs(args))
	}
}

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

// CircuitBreakerState returns the current state of the circuit breaker.
// Returns CircuitClosed if no circuit breaker is configured.
func (c *Client) CircuitBreakerState() CircuitState {
	if c.http.circuitBreaker == nil {
		return CircuitClosed
	}
	return c.http.circuitBreaker.State()
}

// BackpressureStatus returns the current backpressure state.
// Use this to monitor queue health and make decisions about event submission.
func (c *Client) BackpressureStatus() BackpressureHandlerStats {
	if c.backpressure == nil {
		return BackpressureHandlerStats{}
	}
	return c.backpressure.Stats()
}

// BackpressureLevel returns the current backpressure level.
// Returns BackpressureNone if no backpressure handler is configured.
func (c *Client) BackpressureLevel() BackpressureLevel {
	if c.backpressure == nil {
		return BackpressureNone
	}
	return c.backpressure.Monitor().Level()
}

// IsUnderBackpressure returns true if the event queue is experiencing backpressure.
// Use this for adaptive behavior in high-throughput scenarios.
func (c *Client) IsUnderBackpressure() bool {
	if c.backpressure == nil {
		return false
	}
	return c.backpressure.Monitor().Level() >= BackpressureWarning
}

// GenerateID generates a unique ID using the client's configured ID generator.
// Returns an error if the client is configured with IDModeStrict and crypto/rand fails.
func (c *Client) GenerateID() (string, error) {
	if c.idGenerator == nil {
		return GenerateID()
	}
	return c.idGenerator.Generate()
}

// IDStats returns statistics about ID generation.
func (c *Client) IDStats() IDStats {
	if c.idGenerator == nil {
		return IDStats{}
	}
	return c.idGenerator.Stats()
}
