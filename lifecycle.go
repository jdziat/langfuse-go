package langfuse

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

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

// ============================================================================
// Client Lifecycle Methods
// ============================================================================

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

// Health checks the health of the Langfuse API.
func (c *Client) Health(ctx context.Context) (*HealthStatus, error) {
	var result HealthStatus
	err := c.http.get(ctx, endpoints.Health, nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
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
