package langfuse

import (
	"context"
	"time"
)

// batchRequest represents a batch of events to be sent.
type batchRequest struct {
	events []ingestionEvent
	ctx    context.Context
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
