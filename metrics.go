package langfuse

import (
	"encoding/json"
	"net/http"
)

// ClientStats contains a snapshot of all client metrics.
// Use this for monitoring and observability.
type ClientStats struct {
	// Client state
	State       ClientState `json:"state"`
	Uptime      string      `json:"uptime"`
	UptimeNanos int64       `json:"uptime_nanos"`

	// Queue metrics
	QueueSize        int              `json:"queue_size"`
	QueueCapacity    int              `json:"queue_capacity"`
	QueueUtilization float64          `json:"queue_utilization"`
	BackpressureInfo BackpressureInfo `json:"backpressure"`

	// Lifecycle metrics
	Lifecycle LifecycleStats `json:"lifecycle"`

	// ID generation metrics
	IDGeneration IDStats `json:"id_generation"`

	// Circuit breaker state
	CircuitBreaker CircuitBreakerInfo `json:"circuit_breaker"`

	// Batch processing metrics
	Batch BatchStats `json:"batch"`
}

// BackpressureInfo contains backpressure-related metrics.
type BackpressureInfo struct {
	Level           BackpressureLevel `json:"level"`
	LevelString     string            `json:"level_string"`
	DroppedCount    int64             `json:"dropped_count"`
	BlockedCount    int64             `json:"blocked_count"`
	PercentFull     float64           `json:"percent_full"`
	IsUnderPressure bool              `json:"is_under_pressure"`
}

// CircuitBreakerInfo contains circuit breaker metrics.
type CircuitBreakerInfo struct {
	Enabled           bool         `json:"enabled"`
	State             CircuitState `json:"state"`
	StateString       string       `json:"state_string"`
	Failures          int          `json:"failures"`
	ConsecutiveErrors int          `json:"consecutive_errors"`
}

// BatchStats contains batch processing metrics.
type BatchStats struct {
	PendingEvents int `json:"pending_events"`
	QueuedBatches int `json:"queued_batches"`
}

// Stats returns a snapshot of all client metrics.
// This is safe to call concurrently and provides a consistent view
// of the client's current state.
//
// Example:
//
//	stats := client.Stats()
//	log.Printf("Queue: %d/%d (%.1f%%)",
//	    stats.QueueSize, stats.QueueCapacity, stats.QueueUtilization*100)
func (c *Client) Stats() ClientStats {
	stats := ClientStats{
		State:        c.State(),
		Uptime:       c.Uptime().String(),
		UptimeNanos:  c.Uptime().Nanoseconds(),
		Lifecycle:    c.LifecycleStats(),
		IDGeneration: c.IDStats(),
	}

	// Queue metrics
	c.mu.Lock()
	pendingEvents := len(c.pendingEvents)
	c.mu.Unlock()

	queuedBatches := len(c.batchQueue)
	queueCapacity := c.config.BatchSize * c.config.BatchQueueSize
	queueSize := pendingEvents + (queuedBatches * c.config.BatchSize)

	stats.QueueSize = queueSize
	stats.QueueCapacity = queueCapacity
	if queueCapacity > 0 {
		stats.QueueUtilization = float64(queueSize) / float64(queueCapacity)
	}

	stats.Batch = BatchStats{
		PendingEvents: pendingEvents,
		QueuedBatches: queuedBatches,
	}

	// Backpressure metrics
	if c.backpressure != nil {
		bpStats := c.backpressure.Stats()
		level := c.backpressure.Monitor().Level()
		stats.BackpressureInfo = BackpressureInfo{
			Level:           level,
			LevelString:     level.String(),
			DroppedCount:    bpStats.DroppedCount,
			BlockedCount:    bpStats.BlockedCount,
			PercentFull:     bpStats.MonitorStats.LastState.PercentFull,
			IsUnderPressure: level >= BackpressureWarning,
		}
	}

	// Circuit breaker metrics
	if c.http.circuitBreaker != nil {
		cb := c.http.circuitBreaker
		state := cb.State()
		stats.CircuitBreaker = CircuitBreakerInfo{
			Enabled:           true,
			State:             state,
			StateString:       state.String(),
			Failures:          cb.Failures(),
			ConsecutiveErrors: cb.ConsecutiveErrors(),
		}
	}

	return stats
}

// StatsHandler returns an http.Handler that serves client statistics as JSON.
// This is useful for monitoring and debugging.
//
// Example:
//
//	http.Handle("/langfuse/stats", client.StatsHandler())
//	http.ListenAndServe(":8080", nil)
func (c *Client) StatsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		stats := c.Stats()

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

		encoder := json.NewEncoder(w)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(stats); err != nil {
			http.Error(w, "Failed to encode stats", http.StatusInternalServerError)
		}
	})
}

// HealthHandler returns an http.Handler for health checks.
// Returns 200 if the client is active, 503 if shutting down or closed.
//
// Example:
//
//	http.Handle("/health", client.HealthHandler())
func (c *Client) HealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := c.State()

		response := struct {
			Status string `json:"status"`
			State  string `json:"state"`
		}{
			State: string(state),
		}

		if state == ClientStateActive {
			response.Status = "healthy"
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		response.Status = "unhealthy"
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
	})
}

// ReadyHandler returns an http.Handler for readiness checks.
// Returns 200 if the client is ready to accept events, 503 otherwise.
// This considers both client state and circuit breaker state.
//
// Example:
//
//	http.Handle("/ready", client.ReadyHandler())
func (c *Client) ReadyHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state := c.State()
		cbState := c.CircuitBreakerState()
		underPressure := c.IsUnderBackpressure()

		response := struct {
			Ready             bool   `json:"ready"`
			State             string `json:"state"`
			CircuitBreaker    string `json:"circuit_breaker"`
			UnderBackpressure bool   `json:"under_backpressure"`
		}{
			State:             string(state),
			CircuitBreaker:    cbState.String(),
			UnderBackpressure: underPressure,
		}

		ready := state == ClientStateActive && cbState != CircuitOpen

		if ready {
			response.Ready = true
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
			return
		}

		response.Ready = false
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(response)
	})
}
