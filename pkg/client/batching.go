package client

import (
	"context"
	"time"

	pkgerrors "github.com/jdziat/langfuse-go/pkg/errors"
	pkgingestion "github.com/jdziat/langfuse-go/pkg/ingestion"
	pkgtypes "github.com/jdziat/langfuse-go/pkg/types"
)

// Re-export backpressure types from pkg/ingestion for convenience.
type (
	BackpressureLevel        = pkgingestion.BackpressureLevel
	BackpressureDecision     = pkgingestion.BackpressureDecision
	BackpressureHandlerStats = pkgingestion.BackpressureHandlerStats
	QueueState               = pkgingestion.QueueState
)

// Backpressure level constants.
const (
	BackpressureNone     = pkgingestion.BackpressureNone
	BackpressureWarning  = pkgingestion.BackpressureWarning
	BackpressureCritical = pkgingestion.BackpressureCritical
	BackpressureOverflow = pkgingestion.BackpressureOverflow
)

// Decision constants.
const (
	DecisionAllow = pkgingestion.DecisionAllow
	DecisionBlock = pkgingestion.DecisionBlock
	DecisionDrop  = pkgingestion.DecisionDrop
)

// API endpoints.
var endpoints = struct {
	Ingestion string
	Health    string
}{
	Ingestion: "/api/public/ingestion",
	Health:    "/api/public/health",
}

// ErrBackpressure is returned when an event is rejected due to backpressure.
var ErrBackpressure = &pkgerrors.APIError{
	StatusCode: 503,
	Message:    "event rejected due to queue backpressure",
}

// ErrBatchDropped is returned when a batch is dropped because all background
// sender slots are occupied and the batch queue is full.
var ErrBatchDropped = &pkgerrors.APIError{
	StatusCode: 503,
	Message:    "batch dropped: background sender limit reached",
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

// sendBatch sends a batch of events to the API.
// Always signals space availability on return (via defer) since the batch was
// removed from the queue regardless of send success or failure.
func (c *Client) sendBatch(ctx context.Context, events []IngestionEvent) error {
	if len(events) == 0 {
		return nil
	}

	// Always signal space availability when done - the batch was already removed
	// from the queue, so space IS available regardless of send success/failure.
	// This prevents waiters from blocking unnecessarily on persistent errors.
	defer c.signalSpaceAvailable()

	start := time.Now()
	req := &IngestionRequest{
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

// signalSpaceAvailable broadcasts to ALL waiters that queue space may be available.
// Uses close-and-recreate pattern: closing the channel wakes all waiters simultaneously.
// This prevents signal starvation where only one waiter would wake per signal.
func (c *Client) signalSpaceAvailable() {
	c.spaceAvailableMu.Lock()
	close(c.spaceAvailableCh)                // Wake ALL waiters
	c.spaceAvailableCh = make(chan struct{}) // Create new channel for future waiters
	c.spaceAvailableMu.Unlock()
}

// getSpaceAvailableCh returns the current space-available channel under lock.
// Callers should wait on the returned channel, which will be closed when space is available.
func (c *Client) getSpaceAvailableCh() chan struct{} {
	c.spaceAvailableMu.Lock()
	ch := c.spaceAvailableCh
	c.spaceAvailableMu.Unlock()
	return ch
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

// QueueEvent adds an event to the pending queue.
// The provided context is used for immediate batch sends when the batch is full.
//
// Backpressure handling:
//   - If configured with BlockOnQueueFull, this will block until space is available
//   - If configured with DropOnQueueFull, events are silently dropped when full
//   - Otherwise, events are queued normally (may overflow)
func (c *Client) QueueEvent(ctx context.Context, event IngestionEvent) error {
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
			// Return error if batch was dropped so caller knows about data loss
			if err := c.handleQueueFull(events); err != nil {
				return err
			}
		}
	}

	return nil
}

// waitForQueueSpace blocks until queue space is available or context is cancelled.
// Uses broadcast signaling (close-and-recreate pattern) to wake ALL waiters when space
// becomes available, preventing signal starvation.
func (c *Client) waitForQueueSpace(ctx context.Context) error {
	for {
		// Get current channel - will be closed when space is available
		spaceCh := c.getSpaceAvailableCh()

		select {
		case <-ctx.Done():
			return ErrBackpressure
		case <-c.ctx.Done():
			return ErrClientClosed
		case <-spaceCh:
			// Space may be available - check backpressure decision
			if c.backpressure != nil {
				currentQueueSize := c.estimateQueueSize()
				if c.backpressure.Decide(currentQueueSize) != DecisionBlock {
					return nil
				}
				// Still at overflow, wait for next signal
			} else {
				return nil
			}
		}
	}
}

// estimateQueueSize returns an approximate estimate of the current queue size.
//
// DESIGN NOTE: This intentionally provides an approximation rather than an exact count.
// The mutex is released before reading the batch queue length, creating a brief window
// where events could move between pendingEvents and batchQueue. This is acceptable because:
//
//  1. Backpressure decisions don't require exact counts - approximate values are sufficient
//     for determining if we're near capacity thresholds (50%, 80%, 95%).
//  2. Holding the mutex while reading the channel length could cause contention with
//     the batch processor goroutine, degrading throughput under load.
//  3. The estimate errs on the side of overestimating (counting events in both places
//     during the transition), which is the safer direction for backpressure.
//
// For exact queue metrics, use the Stats() method which provides a consistent snapshot.
func (c *Client) estimateQueueSize() int {
	c.mu.Lock()
	pendingCount := len(c.pendingEvents)
	c.mu.Unlock()

	// len() on channels is safe for concurrent access in Go
	batchQueueCount := len(c.batchQueue) * c.config.BatchSize

	return pendingCount + batchQueueCount
}

// addEventToQueue atomically adds an event and returns events to flush if batch is full.
// Uses defer for safe mutex handling.
func (c *Client) addEventToQueue(event IngestionEvent) ([]IngestionEvent, error) {
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
		c.pendingEvents = make([]IngestionEvent, 0, c.config.BatchSize)
		return events, nil
	}

	return nil, nil
}

// handleQueueFull handles the case when the batch queue is full.
// It spawns a tracked goroutine with its own timeout context to send the batch.
// The number of concurrent background senders is limited by MaxBackgroundSenders
// to prevent unbounded goroutine creation under sustained high load.
//
// Returns ErrBatchDropped if all background sender slots are occupied.
// The error is returned so callers can be informed of data loss.
func (c *Client) handleQueueFull(events []IngestionEvent) error {
	// Try to acquire the semaphore without blocking
	select {
	case c.backgroundSendSem <- struct{}{}:
		// Acquired semaphore slot, proceed with background send
		c.log("batch queue full, sending in background goroutine")

		c.wg.Add(1)
		go func() {
			defer func() { <-c.backgroundSendSem }() // Release semaphore
			defer c.wg.Done()

			// Use context.Background() - NOT the user's context.
			// Once the batch is accepted for background sending, user cancellation
			// should not affect it. The batch was already removed from pendingEvents.
			sendCtx, cancel := context.WithTimeout(context.Background(), DefaultBackgroundSendTimeout)
			defer cancel()

			if err := c.sendBatch(sendCtx, events); err != nil {
				c.handleError(err)
			}
		}()
		return nil
	default:
		// Semaphore full - all background sender slots are in use
		c.logError("background sender limit reached (%d), dropping batch of %d events",
			c.config.MaxBackgroundSenders, len(events))

		if c.config.Metrics != nil {
			c.config.Metrics.IncrementCounter("langfuse.batches_dropped", 1)
		}
		return ErrBatchDropped
	}
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
func (c *Client) extractPendingEvents() ([]IngestionEvent, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, ErrClientClosed
	}
	if len(c.pendingEvents) == 0 {
		return nil, nil
	}

	events := c.pendingEvents
	c.pendingEvents = make([]IngestionEvent, 0, c.config.BatchSize)
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
func (c *Client) drainPendingEvents() []IngestionEvent {
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

		return &pkgerrors.ShutdownError{
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

// LifecycleStats returns current lifecycle statistics.
// This includes uptime, last activity time, and client state.
func (c *Client) LifecycleStats() LifecycleStats {
	if c.lifecycle == nil {
		return LifecycleStats{}
	}
	return c.lifecycle.Stats()
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

// HealthStatus is re-exported from pkg/types for consistency.
type HealthStatus = pkgtypes.HealthStatus
