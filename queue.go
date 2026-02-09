package langfuse

import (
	"context"
	"time"
)

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
