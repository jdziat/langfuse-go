package langfuse

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

// defaultStderrLogger is used as a fallback when no logger is configured.
// This ensures async errors are never silently dropped.
var defaultStderrLogger = log.New(os.Stderr, "langfuse: ", log.LstdFlags)

// batchRequest represents a batch of events to be sent.
type batchRequest struct {
	events []ingestionEvent
	ctx    context.Context
}

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

// batchProcessor processes batch requests from the queue.
// It handles graceful shutdown by listening for drainSignal and processing
// all remaining events before signaling completion via drainComplete.
func (c *Client) batchProcessor() {
	defer c.wg.Done()
	defer close(c.drainComplete) // Signal that drain is complete

	for {
		select {
		case <-c.drainSignal:
			// Graceful shutdown: drain all remaining batches and pending events
			c.drainAllEvents()
			return

		case <-c.ctx.Done():
			// Forced shutdown (context cancelled without drain signal)
			// This shouldn't happen in normal operation but handle it safely
			c.log("batch processor context cancelled without drain signal")
			return

		case req := <-c.batchQueue:
			c.processBatchRequest(req)
		}
	}
}

// processBatchRequest handles sending a single batch request.
func (c *Client) processBatchRequest(req batchRequest) {
	start := time.Now()

	// Check if request context is already cancelled before sending
	if req.ctx.Err() != nil {
		c.log("batch request context cancelled, using background context")
		// Use a timeout context instead of the cancelled one
		sendCtx, cancel := context.WithTimeout(context.Background(), DefaultBackgroundSendTimeout)
		if err := c.sendBatch(sendCtx, req.events); err != nil {
			c.handleError(err)
		}
		cancel()
		if c.config.Metrics != nil {
			c.config.Metrics.IncrementCounter("langfuse.batch.context_cancelled", 1)
		}
	} else {
		if err := c.sendBatch(req.ctx, req.events); err != nil {
			c.handleError(err)
		}
	}

	if c.config.Metrics != nil {
		c.config.Metrics.RecordDuration("langfuse.batch.duration", time.Since(start))
		c.config.Metrics.IncrementCounter("langfuse.batch.sent", 1)
		c.config.Metrics.IncrementCounter("langfuse.events.sent", int64(len(req.events)))
	}
}

// drainAllEvents drains all pending events and queued batches during shutdown.
// Uses a fresh context since the client context may be cancelled.
func (c *Client) drainAllEvents() {
	drainCtx, cancel := context.WithTimeout(context.Background(), c.config.ShutdownTimeout)
	defer cancel()

	// First, drain any pending events that haven't been batched yet
	pendingEvents := c.drainPendingEvents()
	if len(pendingEvents) > 0 {
		c.log("draining %d pending events during shutdown", len(pendingEvents))
		if err := c.sendBatch(drainCtx, pendingEvents); err != nil {
			c.handleError(err)
		}
	}

	// Then drain any batches already in the queue
	drained := 0
	for {
		select {
		case req := <-c.batchQueue:
			if err := c.sendBatch(drainCtx, req.events); err != nil {
				c.handleError(err)
			}
			drained++
		case <-drainCtx.Done():
			c.log("drain timeout, %d batches drained, some may be lost", drained)
			return
		default:
			// Queue is empty
			if drained > 0 {
				c.log("drained %d batches during shutdown", drained)
			}
			return
		}
	}
}

// flushLoop periodically flushes pending events.
func (c *Client) flushLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopFlush:
			return
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			if err := c.Flush(c.ctx); err != nil && err != ErrClientClosed {
				c.handleError(err)
			}
		}
	}
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

// Health checks the health of the Langfuse API.
func (c *Client) Health(ctx context.Context) (*HealthStatus, error) {
	var result HealthStatus
	err := c.http.get(ctx, endpoints.Health, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Flush sends all pending events to the Langfuse API.
func (c *Client) Flush(ctx context.Context) error {
	events, err := c.extractPendingEvents()
	if err != nil {
		return err
	}
	if len(events) == 0 {
		return nil
	}
	return c.sendBatch(ctx, events)
}

// extractPendingEvents atomically extracts and clears pending events.
// Uses defer for safe mutex handling.
func (c *Client) extractPendingEvents() ([]ingestionEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, ErrClientClosed
	}
	if len(c.pendingEvents) == 0 {
		return nil, nil
	}

	events := c.pendingEvents
	c.pendingEvents = make([]ingestionEvent, 0, c.config.BatchSize)
	return events, nil
}

// markClosed atomically marks the client as closed.
// Returns ErrClientClosed if already closed.
func (c *Client) markClosed() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrClientClosed
	}
	c.closed = true
	return nil
}

// drainPendingEvents atomically drains all pending events during shutdown.
func (c *Client) drainPendingEvents() []ingestionEvent {
	c.mu.Lock()
	defer c.mu.Unlock()

	events := c.pendingEvents
	c.pendingEvents = nil
	return events
}

// Shutdown flushes pending events and closes the client gracefully.
//
// The shutdown process:
//  1. Stop accepting new events (mark closed)
//  2. Stop the flush loop
//  3. Signal batch processor to drain all pending and queued events
//  4. Wait for drain to complete (or timeout)
//  5. Cancel context to stop any remaining goroutines
//  6. Wait for all goroutines to finish
//
// Returns a ShutdownError if the shutdown times out, which includes
// information about how many events may have been lost.
func (c *Client) Shutdown(ctx context.Context) error {
	// Use lifecycle manager to begin shutdown
	if c.lifecycle != nil {
		if err := c.lifecycle.BeginShutdown(); err != nil {
			return err
		}
	}

	// Step 1: Stop accepting new events
	if err := c.markClosed(); err != nil {
		if c.lifecycle != nil {
			c.lifecycle.CompleteShutdown()
		}
		return err
	}

	// Step 2: Stop the flush loop
	close(c.stopFlush)

	// Step 3: Signal batch processor to drain all events
	// IMPORTANT: Do this BEFORE canceling context so drain can complete
	close(c.drainSignal)

	// Step 4: Wait for drain to complete with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(ctx, c.config.ShutdownTimeout)
	defer shutdownCancel()

	drainedSuccessfully := false
	select {
	case <-c.drainComplete:
		// Batch processor finished draining all events
		drainedSuccessfully = true
		c.log("batch processor drain complete")
	case <-shutdownCtx.Done():
		// Timeout waiting for drain
		c.log("drain timeout, forcing shutdown")
	}

	// Step 5: Cancel context to stop any remaining goroutines
	c.cancel()

	// Step 6: Wait for all goroutines with remaining timeout
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		// All goroutines stopped
		if c.lifecycle != nil {
			c.lifecycle.CompleteShutdown()
		}

		if drainedSuccessfully {
			c.logInfo("shutdown complete", "drained", true)
			if c.config.Metrics != nil {
				c.config.Metrics.IncrementCounter("langfuse.shutdown.success", 1)
			}
			return nil
		}

		// Drain timed out but goroutines stopped
		c.logInfo("shutdown complete with drain timeout")
		if c.config.Metrics != nil {
			c.config.Metrics.IncrementCounter("langfuse.shutdown.drain_timeout", 1)
		}
		return nil

	case <-shutdownCtx.Done():
		// Complete lifecycle even on timeout
		if c.lifecycle != nil {
			c.lifecycle.CompleteShutdown()
		}

		// Estimate potentially lost events
		queuedBatches := len(c.batchQueue)
		potentiallyLost := queuedBatches * c.config.BatchSize

		c.logError("shutdown timeout", "potentially_lost_events", potentiallyLost, "queued_batches", queuedBatches)

		if c.config.Metrics != nil {
			c.config.Metrics.IncrementCounter("langfuse.shutdown.timeout", 1)
			c.config.Metrics.SetGauge("langfuse.shutdown.lost_events", float64(potentiallyLost))
		}

		return &ShutdownError{
			Cause:         shutdownCtx.Err(),
			PendingEvents: potentiallyLost,
			Message:       "timeout waiting for background goroutines",
		}
	}
}

// Close is an alias for Shutdown.
func (c *Client) Close(ctx context.Context) error {
	return c.Shutdown(ctx)
}

// ErrBackpressure is returned when an event is rejected due to backpressure.
var ErrBackpressure = &APIError{
	StatusCode: 503,
	Message:    "event rejected due to queue backpressure",
}

// queueEvent adds an event to the pending queue.
// The provided context is used for immediate batch sends when the batch is full.
//
// Backpressure handling:
//   - If configured with BlockOnQueueFull, this will block until space is available
//   - If configured with DropOnQueueFull, events are silently dropped when full
//   - Otherwise, events are queued normally (may overflow)
func (c *Client) queueEvent(ctx context.Context, event ingestionEvent) error {
	// Record activity for idle detection
	if c.lifecycle != nil {
		c.lifecycle.RecordActivity()
	}

	// Check backpressure before queuing
	if c.backpressure != nil {
		currentQueueSize := c.estimateQueueSize()
		decision := c.backpressure.Decide(currentQueueSize)

		switch decision {
		case DecisionDrop:
			// Drop the event silently (already logged/metriced by handler)
			return nil
		case DecisionBlock:
			// Block until space is available or context is cancelled
			if err := c.waitForQueueSpace(ctx); err != nil {
				return err
			}
		}
		// DecisionAllow: continue with queueing
	}

	events, err := c.addEventToQueue(event)
	if err != nil {
		return err
	}

	// Non-critical section: send to channel (no lock held)
	if len(events) > 0 {
		select {
		case c.batchQueue <- batchRequest{events: events, ctx: ctx}:
			// Successfully queued
		default:
			// Queue is full, spawn tracked goroutine
			c.handleQueueFull(ctx, events)
		}
	}

	return nil
}

// waitForQueueSpace blocks until queue space is available or context is cancelled.
func (c *Client) waitForQueueSpace(ctx context.Context) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ErrBackpressure
		case <-c.ctx.Done():
			return ErrClientClosed
		case <-ticker.C:
			// Check if queue has space now
			if c.backpressure != nil {
				currentQueueSize := c.estimateQueueSize()
				if c.backpressure.Decide(currentQueueSize) != DecisionBlock {
					return nil
				}
			} else {
				return nil
			}
		}
	}
}

// estimateQueueSize returns an estimate of the current queue size.
// This is thread-safe and provides a point-in-time estimate.
func (c *Client) estimateQueueSize() int {
	c.mu.Lock()
	pendingCount := len(c.pendingEvents)
	c.mu.Unlock()

	// Estimate batch queue contribution
	// len() on channels is safe for concurrent access in Go
	batchQueueCount := len(c.batchQueue) * c.config.BatchSize

	return pendingCount + batchQueueCount
}

// addEventToQueue atomically adds an event and returns events to flush if batch is full.
// Uses defer for safe mutex handling.
func (c *Client) addEventToQueue(event ingestionEvent) ([]ingestionEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, ErrClientClosed
	}

	c.pendingEvents = append(c.pendingEvents, event)

	if c.config.Metrics != nil {
		c.config.Metrics.SetGauge("langfuse.pending_events", float64(len(c.pendingEvents)))
	}

	// Check if we need to flush
	if len(c.pendingEvents) >= c.config.BatchSize {
		events := c.pendingEvents
		c.pendingEvents = make([]ingestionEvent, 0, c.config.BatchSize)
		return events, nil
	}

	return nil, nil
}

// handleQueueFull handles the case when the batch queue is full.
// It spawns a tracked goroutine with its own timeout context to send the batch.
func (c *Client) handleQueueFull(ctx context.Context, events []ingestionEvent) {
	c.log("batch queue full, sending in background goroutine")

	// Track the goroutine to ensure graceful shutdown
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		// Use a timeout context that inherits from the provided context
		// but has a maximum timeout to ensure completion
		sendCtx, cancel := context.WithTimeout(ctx, DefaultBackgroundSendTimeout)
		defer cancel()

		if err := c.sendBatch(sendCtx, events); err != nil {
			c.handleError(err)
		}
	}()
}

// sendBatch sends a batch of events to the API.
func (c *Client) sendBatch(ctx context.Context, events []ingestionEvent) error {
	if len(events) == 0 {
		return nil
	}

	start := time.Now()
	req := &ingestionRequest{
		Batch: events,
	}

	var result IngestionResult

	// Circuit breaker is now handled by httpClient.do() automatically
	err := c.http.post(ctx, endpoints.Ingestion, req, &result)
	duration := time.Since(start)

	// Prepare batch result for callback
	batchResult := BatchResult{
		EventCount: len(events),
		Success:    err == nil,
		Error:      err,
		Duration:   duration,
	}

	if err == nil {
		batchResult.Successes = len(result.Successes)
		batchResult.Errors = len(result.Errors)
	}

	// Call the batch callback if configured
	if c.config.OnBatchFlushed != nil {
		c.config.OnBatchFlushed(batchResult)
	}

	if err != nil {
		return err
	}

	// Log errors if any
	if result.HasErrors() {
		for _, e := range result.Errors {
			c.log("ingestion error for event %s: %s", e.ID, e.Message)
		}
		if c.config.Metrics != nil {
			c.config.Metrics.IncrementCounter("langfuse.ingestion.errors", int64(len(result.Errors)))
		}
	}

	return nil
}

// LifecycleStats returns current lifecycle statistics.
// This includes uptime, last activity time, and client state.
func (c *Client) LifecycleStats() LifecycleStats {
	if c.lifecycle == nil {
		return LifecycleStats{}
	}
	return c.lifecycle.Stats()
}

// State returns the current client state.
func (c *Client) State() ClientState {
	if c.lifecycle == nil {
		c.mu.Lock()
		closed := c.closed
		c.mu.Unlock()
		if closed {
			return ClientStateClosed
		}
		return ClientStateActive
	}
	return c.lifecycle.State()
}

// IsActive returns true if the client is active and accepting events.
func (c *Client) IsActive() bool {
	return c.State() == ClientStateActive
}

// Uptime returns the duration since the client was created.
func (c *Client) Uptime() time.Duration {
	if c.lifecycle == nil {
		return 0
	}
	return c.lifecycle.Uptime()
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

// ============================================================================
// Lifecycle Management
// ============================================================================

// ClientState represents the current state of the client lifecycle.
type ClientState int32

const (
	// ClientStateActive indicates the client is active and accepting events.
	ClientStateActive ClientState = iota

	// ClientStateShuttingDown indicates the client is shutting down.
	ClientStateShuttingDown

	// ClientStateClosed indicates the client has been closed.
	ClientStateClosed
)

// String returns a string representation of the client state.
func (s ClientState) String() string {
	switch s {
	case ClientStateActive:
		return "active"
	case ClientStateShuttingDown:
		return "shutting_down"
	case ClientStateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

// LifecycleManager handles client lifecycle with leak prevention.
// It tracks client state, monitors for idle clients, and provides
// warnings when clients are not properly shut down.
type LifecycleManager struct {
	state        atomic.Int32
	createdAt    time.Time
	lastActivity atomic.Int64 // Unix nano timestamp

	// Shutdown coordination
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// Idle detection
	idleWarningDuration time.Duration
	warningFired        atomic.Bool
	logger              Logger
	metrics             Metrics

	// Callbacks
	onStateChange func(old, new ClientState)
}

// LifecycleConfig configures the lifecycle manager.
type LifecycleConfig struct {
	// IdleWarningDuration triggers a warning if no activity occurs within this duration.
	// Set to 0 to disable idle warnings.
	IdleWarningDuration time.Duration

	// Logger is used for warning messages.
	Logger Logger

	// Metrics is used for lifecycle metrics.
	Metrics Metrics

	// OnStateChange is called when the client state changes.
	OnStateChange func(old, new ClientState)
}

// NewLifecycleManager creates a new lifecycle manager.
func NewLifecycleManager(cfg *LifecycleConfig) *LifecycleManager {
	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now()

	lm := &LifecycleManager{
		createdAt:           now,
		ctx:                 ctx,
		cancel:              cancel,
		idleWarningDuration: cfg.IdleWarningDuration,
		logger:              cfg.Logger,
		metrics:             cfg.Metrics,
		onStateChange:       cfg.OnStateChange,
	}

	lm.state.Store(int32(ClientStateActive))
	lm.lastActivity.Store(now.UnixNano())

	// Start idle detector if configured
	if cfg.IdleWarningDuration > 0 && cfg.Logger != nil {
		lm.wg.Add(1)
		go lm.idleDetector()
	}

	// Record creation metric
	if cfg.Metrics != nil {
		cfg.Metrics.IncrementCounter("langfuse.client.created", 1)
	}

	return lm
}

// idleDetector monitors for idle clients and logs warnings.
func (lm *LifecycleManager) idleDetector() {
	defer lm.wg.Done()

	// Check at half the idle duration for responsiveness
	checkInterval := lm.idleWarningDuration / 2
	if checkInterval < time.Second {
		checkInterval = time.Second
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-lm.ctx.Done():
			return
		case <-ticker.C:
			if lm.State() != ClientStateActive {
				return
			}

			lastActivity := time.Unix(0, lm.lastActivity.Load())
			idle := time.Since(lastActivity)

			if idle > lm.idleWarningDuration && lm.warningFired.CompareAndSwap(false, true) {
				lm.logger.Printf(
					"WARNING: Langfuse client has been idle for %v without Shutdown() being called. "+
						"This may indicate a goroutine leak. Always call client.Shutdown(ctx) when done. "+
						"Created at: %s",
					idle.Round(time.Second),
					lm.createdAt.Format(time.RFC3339),
				)

				if lm.metrics != nil {
					lm.metrics.IncrementCounter("langfuse.client.idle_warning", 1)
				}
			}
		}
	}
}

// State returns the current client state.
func (lm *LifecycleManager) State() ClientState {
	return ClientState(lm.state.Load())
}

// IsActive returns true if the client is active.
func (lm *LifecycleManager) IsActive() bool {
	return lm.State() == ClientStateActive
}

// IsClosed returns true if the client is closed.
func (lm *LifecycleManager) IsClosed() bool {
	return lm.State() == ClientStateClosed
}

// RecordActivity updates the last activity timestamp.
// Call this when the client performs any operation.
func (lm *LifecycleManager) RecordActivity() {
	lm.lastActivity.Store(time.Now().UnixNano())
}

// LastActivity returns the time of the last recorded activity.
func (lm *LifecycleManager) LastActivity() time.Time {
	return time.Unix(0, lm.lastActivity.Load())
}

// Uptime returns the duration since the client was created.
func (lm *LifecycleManager) Uptime() time.Duration {
	return time.Since(lm.createdAt)
}

// IdleDuration returns the duration since the last activity.
func (lm *LifecycleManager) IdleDuration() time.Duration {
	return time.Since(lm.LastActivity())
}

// CreatedAt returns the time the client was created.
func (lm *LifecycleManager) CreatedAt() time.Time {
	return lm.createdAt
}

// Context returns the lifecycle context.
// This context is cancelled when shutdown begins.
func (lm *LifecycleManager) Context() context.Context {
	return lm.ctx
}

// transition attempts to transition to a new state.
// Returns true if the transition was successful.
func (lm *LifecycleManager) transition(from, to ClientState) bool {
	if lm.state.CompareAndSwap(int32(from), int32(to)) {
		if lm.onStateChange != nil {
			lm.onStateChange(from, to)
		}
		if lm.metrics != nil {
			lm.metrics.SetGauge("langfuse.client.state", float64(to))
		}
		return true
	}
	return false
}

// BeginShutdown initiates the shutdown process.
// Returns ErrClientClosed if already shutting down or closed.
func (lm *LifecycleManager) BeginShutdown() error {
	if !lm.transition(ClientStateActive, ClientStateShuttingDown) {
		state := lm.State()
		if state == ClientStateShuttingDown || state == ClientStateClosed {
			return ErrClientClosed
		}
		return ErrClientClosed
	}

	// Cancel the context to signal all goroutines
	lm.cancel()

	if lm.metrics != nil {
		lm.metrics.IncrementCounter("langfuse.client.shutdown_initiated", 1)
		lm.metrics.RecordDuration("langfuse.client.uptime", lm.Uptime())
	}

	return nil
}

// CompleteShutdown marks the shutdown as complete.
func (lm *LifecycleManager) CompleteShutdown() {
	lm.transition(ClientStateShuttingDown, ClientStateClosed)

	// Wait for idle detector to stop
	lm.wg.Wait()

	if lm.metrics != nil {
		lm.metrics.IncrementCounter("langfuse.client.shutdown_complete", 1)
	}
}

// WaitGroup returns the lifecycle WaitGroup for tracking goroutines.
func (lm *LifecycleManager) WaitGroup() *sync.WaitGroup {
	return &lm.wg
}

// LifecycleStats contains lifecycle statistics.
type LifecycleStats struct {
	State        ClientState
	CreatedAt    time.Time
	LastActivity time.Time
	Uptime       time.Duration
	IdleDuration time.Duration
}

// Stats returns current lifecycle statistics.
func (lm *LifecycleManager) Stats() LifecycleStats {
	return LifecycleStats{
		State:        lm.State(),
		CreatedAt:    lm.createdAt,
		LastActivity: lm.LastActivity(),
		Uptime:       lm.Uptime(),
		IdleDuration: lm.IdleDuration(),
	}
}
