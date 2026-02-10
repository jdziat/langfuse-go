package client

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	pkgid "github.com/jdziat/langfuse-go/pkg/id"
	pkgingestion "github.com/jdziat/langfuse-go/pkg/ingestion"
)

// Version is the SDK version.
const Version = "0.1.0"

// defaultStderrLogger is used as a fallback when no logger is configured.
var defaultStderrLogger = log.New(os.Stderr, "langfuse: ", log.LstdFlags)

// Client is the main Langfuse client.
type Client struct {
	config *Config
	http   *httpClient

	// Lifecycle management
	lifecycle   *LifecycleManager
	idGenerator *pkgid.IDGenerator

	// Batching
	mu            sync.Mutex
	pendingEvents []IngestionEvent
	closed        bool

	// Background goroutine management
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	batchQueue chan batchRequest
	stopFlush  chan struct{}

	// Graceful shutdown signaling
	drainSignal   chan struct{}
	drainComplete chan struct{}

	// Backpressure management
	backpressure *pkgingestion.BackpressureHandler
}

// batchRequest represents a batch of events to be sent.
type batchRequest struct {
	events []IngestionEvent
	ctx    context.Context
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
func NewWithConfig(cfg *Config) (*Client, error) {
	if cfg == nil {
		return nil, ErrNilRequest
	}

	// Make a copy to avoid modifying the original
	cfgCopy := *cfg
	cfgCopy.ApplyDefaults()

	if err := cfgCopy.Validate(); err != nil {
		return nil, err
	}

	httpClient := newHTTPClient(&cfgCopy)

	ctx, cancel := context.WithCancel(context.Background())

	// Initialize lifecycle manager
	lifecycle := NewLifecycleManager(&LifecycleConfig{
		IdleWarningDuration: cfgCopy.IdleWarningDuration,
		Logger:              cfgCopy.Logger,
		Metrics:             cfgCopy.Metrics,
	})

	// Initialize ID generator
	idGenerator := pkgid.NewIDGenerator(&pkgid.IDGeneratorConfig{
		Mode: pkgid.IDGenerationMode(cfgCopy.IDGenerationMode),
	})

	// Initialize backpressure handler
	var backpressureHandler *pkgingestion.BackpressureHandler
	if cfgCopy.BackpressureConfig != nil {
		backpressureHandler = pkgingestion.NewBackpressureHandler(cfgCopy.BackpressureConfig)
	} else {
		queueCapacity := cfgCopy.BatchSize * cfgCopy.BatchQueueSize
		backpressureHandler = pkgingestion.NewBackpressureHandler(&pkgingestion.BackpressureHandlerConfig{
			Monitor: pkgingestion.NewQueueMonitor(&pkgingestion.QueueMonitorConfig{
				Threshold:      pkgingestion.DefaultBackpressureThreshold(),
				Capacity:       queueCapacity,
				OnBackpressure: cfgCopy.OnBackpressure,
				Metrics:        wrapMetrics(cfgCopy.Metrics),
				Logger:         wrapLogger(cfgCopy.Logger),
			}),
			BlockOnFull: cfgCopy.BlockOnQueueFull,
			DropOnFull:  cfgCopy.DropOnQueueFull,
			Logger:      wrapLogger(cfgCopy.Logger),
		})
	}

	c := &Client{
		config:        &cfgCopy,
		http:          httpClient,
		lifecycle:     lifecycle,
		idGenerator:   idGenerator,
		pendingEvents: make([]IngestionEvent, 0, cfgCopy.BatchSize),
		ctx:           ctx,
		cancel:        cancel,
		batchQueue:    make(chan batchRequest, cfgCopy.BatchQueueSize),
		stopFlush:     make(chan struct{}),
		drainSignal:   make(chan struct{}),
		drainComplete: make(chan struct{}),
		backpressure:  backpressureHandler,
	}

	// Start background batch processor
	c.wg.Add(1)
	go c.batchProcessor()

	// Start flush timer
	c.wg.Add(1)
	go c.flushLoop()

	return c, nil
}

// handleError handles async errors.
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

// formatArgs formats structured logging arguments.
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

// GenerateID generates a unique ID using the client's configured ID generator.
func (c *Client) GenerateID() (string, error) {
	if c.idGenerator == nil {
		return pkgid.GenerateID()
	}
	return c.idGenerator.Generate()
}

// Config returns a copy of the client configuration.
func (c *Client) Config() Config {
	return *c.config
}

// HTTP returns the HTTP client as a Doer interface.
// This is used by sub-clients (traces, observations, etc.) to make API calls.
func (c *Client) HTTP() Doer {
	return c.http
}

// wrapMetrics converts our Metrics interface to pkg/ingestion's interface.
func wrapMetrics(m Metrics) pkgingestion.Metrics {
	if m == nil {
		return nil
	}
	return &metricsWrapper{m: m}
}

type metricsWrapper struct {
	m Metrics
}

func (w *metricsWrapper) IncrementCounter(name string, value int64) {
	w.m.IncrementCounter(name, value)
}

func (w *metricsWrapper) RecordDuration(name string, duration time.Duration) {
	w.m.RecordDuration(name, duration)
}

func (w *metricsWrapper) SetGauge(name string, value float64) {
	w.m.SetGauge(name, value)
}

// wrapLogger converts our Logger interface to pkg/ingestion's interface.
func wrapLogger(l Logger) pkgingestion.Logger {
	if l == nil {
		return nil
	}
	return &loggerWrapper{l: l}
}

type loggerWrapper struct {
	l Logger
}

func (w *loggerWrapper) Printf(format string, v ...any) {
	w.l.Printf(format, v...)
}
