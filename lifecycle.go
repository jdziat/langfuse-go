package langfuse

import (
	"context"
	"encoding/json"
	"net/http"

	pkgingestion "github.com/jdziat/langfuse-go/pkg/ingestion"
	pkglifecycle "github.com/jdziat/langfuse-go/pkg/lifecycle"
)

// ============================================================================
// Lifecycle Types - Re-exported from pkg/lifecycle
// ============================================================================

// ClientState represents the current state of the client lifecycle.
type ClientState = pkglifecycle.ClientState

const (
	// ClientStateActive indicates the client is active and accepting events.
	ClientStateActive = pkglifecycle.ClientStateActive

	// ClientStateShuttingDown indicates the client is shutting down.
	ClientStateShuttingDown = pkglifecycle.ClientStateShuttingDown

	// ClientStateClosed indicates the client has been closed.
	ClientStateClosed = pkglifecycle.ClientStateClosed
)

// LifecycleConfig configures the lifecycle manager.
type LifecycleConfig = pkglifecycle.Config

// LifecycleStats contains lifecycle statistics.
type LifecycleStats = pkglifecycle.Stats

// LifecycleManager handles client lifecycle with leak prevention.
type LifecycleManager = pkglifecycle.Manager

// NewLifecycleManager creates a new lifecycle manager.
var NewLifecycleManager = pkglifecycle.NewManager

// ErrAlreadyClosed is returned when attempting to shutdown an already closed manager.
// For Client.Shutdown(), ErrClientClosed is returned instead for backward compatibility.
var ErrAlreadyClosed = pkglifecycle.ErrAlreadyClosed

// ============================================================================
// Client Lifecycle Methods
// ============================================================================

// Note: Most core lifecycle methods (Close, Flush, State, IsActive, Uptime,
// LifecycleStats, Health) are provided by the embedded *pkgclient.Client.
// Shutdown is wrapped to convert ErrAlreadyClosed to ErrClientClosed for
// backward compatibility.

// Shutdown gracefully shuts down the client, flushing any pending events.
// Returns ErrClientClosed if already closed (for backward compatibility).
func (c *Client) Shutdown(ctx context.Context) error {
	err := c.Client.Shutdown(ctx)
	// Convert ErrAlreadyClosed to ErrClientClosed for backward compatibility
	if err == ErrAlreadyClosed {
		return ErrClientClosed
	}
	return err
}

// ============================================================================
// Client Statistics
// ============================================================================

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
//	log.Printf("State: %s, Uptime: %s", stats.State, stats.Uptime)
func (c *Client) Stats() ClientStats {
	stats := ClientStats{
		State:        c.State(),
		Uptime:       c.Uptime().String(),
		UptimeNanos:  c.Uptime().Nanoseconds(),
		Lifecycle:    c.LifecycleStats(),
		IDGeneration: c.IDStats(),
	}

	// Queue/batch metrics from rootConfig
	queueCapacity := c.rootConfig.BatchSize * c.rootConfig.BatchQueueSize
	stats.QueueCapacity = queueCapacity

	// Backpressure metrics using exported methods
	bpLevel := c.BackpressureLevel()
	bpStats := c.BackpressureStatus()
	stats.BackpressureInfo = BackpressureInfo{
		Level:           bpLevel,
		LevelString:     bpLevel.String(),
		DroppedCount:    bpStats.DroppedCount,
		BlockedCount:    bpStats.BlockedCount,
		PercentFull:     bpStats.MonitorStats.LastState.PercentFull,
		IsUnderPressure: c.IsUnderBackpressure(),
	}

	// Circuit breaker metrics using exported method
	cbState := c.CircuitBreakerState()
	stats.CircuitBreaker = CircuitBreakerInfo{
		Enabled:     cbState != CircuitClosed || c.rootConfig.CircuitBreaker != nil,
		State:       cbState,
		StateString: cbState.String(),
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

// ============================================================================
// Internal Metrics - Re-exported from pkg/lifecycle
// ============================================================================

// InternalMetrics defines all SDK internal metric names.
// Re-exported from pkg/lifecycle.
type InternalMetrics = pkglifecycle.InternalMetrics

// DefaultInternalMetrics returns metric names with "langfuse." prefix.
var DefaultInternalMetrics = pkglifecycle.DefaultInternalMetrics

// MetricsRecorder wraps the Metrics interface with convenience methods
// for recording SDK internal metrics.
// Re-exported from pkg/lifecycle.
type MetricsRecorder = pkglifecycle.MetricsRecorder

// MetricsRecorder constructor functions.
var (
	NewMetricsRecorder          = pkglifecycle.NewMetricsRecorder
	NewMetricsRecorderWithNames = pkglifecycle.NewMetricsRecorderWithNames
)

// MetricsSnapshot represents a point-in-time snapshot of SDK metrics.
// Re-exported from pkg/lifecycle.
type MetricsSnapshot = pkglifecycle.MetricsSnapshot

// MetricsAggregator collects metrics for periodic reporting.
// Re-exported from pkg/lifecycle.
type MetricsAggregator = pkglifecycle.MetricsAggregator

// NewMetricsAggregator creates a new metrics aggregator.
var NewMetricsAggregator = pkglifecycle.NewMetricsAggregator

// ============================================================================
// Ingestion Event Type Constants - Re-exported from pkg/ingestion
// ============================================================================

const (
	eventTypeTraceCreate      = pkgingestion.EventTypeTraceCreate
	eventTypeTraceUpdate      = pkgingestion.EventTypeTraceUpdate
	eventTypeSpanCreate       = pkgingestion.EventTypeSpanCreate
	eventTypeSpanUpdate       = pkgingestion.EventTypeSpanUpdate
	eventTypeGenerationCreate = pkgingestion.EventTypeGenerationCreate
	eventTypeGenerationUpdate = pkgingestion.EventTypeGenerationUpdate
	eventTypeEventCreate      = pkgingestion.EventTypeEventCreate
	eventTypeScoreCreate      = pkgingestion.EventTypeScoreCreate
	eventTypeSDKLog           = pkgingestion.EventTypeSDKLog
)

// ============================================================================
// UUID Functions - Re-exported from pkg/ingestion
// ============================================================================

// UUID generates a random UUID v4.
var UUID = pkgingestion.UUID

// IsValidUUID checks if a string is a valid UUID format.
var IsValidUUID = pkgingestion.IsValidUUID

// generateID generates a random UUID-like ID.
var generateID = pkgingestion.GenerateID

// ============================================================================
// Backpressure Types - Re-exported from pkg/ingestion
// ============================================================================

// BackpressureLevel indicates the severity of queue backpressure.
type BackpressureLevel = pkgingestion.BackpressureLevel

const (
	// BackpressureNone indicates the queue is operating normally.
	BackpressureNone = pkgingestion.BackpressureNone
	// BackpressureWarning indicates the queue is filling up but not critical.
	BackpressureWarning = pkgingestion.BackpressureWarning
	// BackpressureCritical indicates the queue is nearly full.
	BackpressureCritical = pkgingestion.BackpressureCritical
	// BackpressureOverflow indicates events are being dropped.
	BackpressureOverflow = pkgingestion.BackpressureOverflow
)

// BackpressureThreshold defines when backpressure levels are triggered.
type BackpressureThreshold = pkgingestion.BackpressureThreshold

// DefaultBackpressureThreshold returns sensible default thresholds.
var DefaultBackpressureThreshold = pkgingestion.DefaultBackpressureThreshold

// QueueState represents the current state of the event queue.
type QueueState = pkgingestion.QueueState

// BackpressureCallback is called when backpressure level changes.
type BackpressureCallback = pkgingestion.BackpressureCallback

// QueueMonitor monitors queue state and signals backpressure.
type QueueMonitor = pkgingestion.QueueMonitor

// QueueMonitorConfig configures the QueueMonitor.
type QueueMonitorConfig = pkgingestion.QueueMonitorConfig

// NewQueueMonitor creates a new queue monitor.
var NewQueueMonitor = pkgingestion.NewQueueMonitor

// QueueMonitorStats contains statistics about queue monitoring.
type QueueMonitorStats = pkgingestion.QueueMonitorStats

// BackpressureHandler provides a higher-level API for handling backpressure.
type BackpressureHandler = pkgingestion.BackpressureHandler

// BackpressureHandlerConfig configures the BackpressureHandler.
type BackpressureHandlerConfig = pkgingestion.BackpressureHandlerConfig

// NewBackpressureHandler creates a new backpressure handler.
var NewBackpressureHandler = pkgingestion.NewBackpressureHandler

// BackpressureDecision represents the decision made by the handler.
type BackpressureDecision = pkgingestion.BackpressureDecision

const (
	// DecisionAllow indicates the event should be queued.
	DecisionAllow = pkgingestion.DecisionAllow
	// DecisionBlock indicates the caller should wait.
	DecisionBlock = pkgingestion.DecisionBlock
	// DecisionDrop indicates the event should be dropped.
	DecisionDrop = pkgingestion.DecisionDrop
)

// BackpressureHandlerStats contains statistics about backpressure handling.
type BackpressureHandlerStats = pkgingestion.BackpressureHandlerStats

// ============================================================================
// Internal Helper Functions
// ============================================================================

// isHexString checks if a string contains only hexadecimal characters.
func isHexString(s string) bool {
	for _, c := range s {
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		isUpperHex := c >= 'A' && c <= 'F'
		if !isDigit && !isLowerHex && !isUpperHex {
			return false
		}
	}
	return true
}

// ============================================================================
// Event Structures (Root-specific)
// ============================================================================

// ingestionRequest represents a batch ingestion request.
type ingestionRequest struct {
	Batch    []ingestionEvent `json:"batch"`
	Metadata Metadata         `json:"metadata,omitempty"`
}

// ingestionEvent represents a single event in a batch.
type ingestionEvent struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Timestamp Time   `json:"timestamp"`
	Body      any    `json:"body"`
}

// traceEvent represents the body of trace-create and trace-update events.
type traceEvent struct {
	ID          string   `json:"id"`
	Timestamp   Time     `json:"timestamp,omitempty"`
	Name        string   `json:"name,omitempty"`
	UserID      string   `json:"userId,omitempty"`
	Input       any      `json:"input,omitempty"`
	Output      any      `json:"output,omitempty"`
	Metadata    Metadata `json:"metadata,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	SessionID   string   `json:"sessionId,omitempty"`
	Release     string   `json:"release,omitempty"`
	Version     string   `json:"version,omitempty"`
	Public      bool     `json:"public,omitempty"`
	Environment string   `json:"environment,omitempty"`
}

// observationEvent represents the body of span/generation/event create/update events.
type observationEvent struct {
	// Common observation fields
	ID                  string           `json:"id"`
	TraceID             string           `json:"traceId,omitempty"`
	Name                string           `json:"name,omitempty"`
	StartTime           Time             `json:"startTime,omitempty"`
	EndTime             Time             `json:"endTime,omitempty"`
	Metadata            Metadata         `json:"metadata,omitempty"`
	Level               ObservationLevel `json:"level,omitempty"`
	StatusMessage       string           `json:"statusMessage,omitempty"`
	ParentObservationID string           `json:"parentObservationId,omitempty"`
	Version             string           `json:"version,omitempty"`
	Input               any              `json:"input,omitempty"`
	Output              any              `json:"output,omitempty"`
	Environment         string           `json:"environment,omitempty"`

	// Generation-specific fields (ignored for spans/events)
	Model               string   `json:"model,omitempty"`
	ModelParameters     Metadata `json:"modelParameters,omitempty"`
	Usage               *Usage   `json:"usage,omitempty"`
	PromptName          string   `json:"promptName,omitempty"`
	PromptVersion       int      `json:"promptVersion,omitempty"`
	CompletionStartTime Time     `json:"completionStartTime,omitempty"`
}

// scoreEvent represents the body of a score-create event.
type scoreEvent struct {
	ID            string        `json:"id,omitempty"`
	TraceID       string        `json:"traceId"`
	ObservationID string        `json:"observationId,omitempty"`
	Name          string        `json:"name"`
	Value         any           `json:"value"`
	StringValue   string        `json:"stringValue,omitempty"`
	DataType      ScoreDataType `json:"dataType,omitempty"`
	Comment       string        `json:"comment,omitempty"`
	ConfigID      string        `json:"configId,omitempty"`
	Environment   string        `json:"environment,omitempty"`
	Metadata      Metadata      `json:"metadata,omitempty"`
}

// Type aliases consolidate the 7 legacy event types into 3 unified types.
type (
	createTraceEvent      = traceEvent
	updateTraceEvent      = traceEvent
	createSpanEvent       = observationEvent
	updateSpanEvent       = observationEvent
	createGenerationEvent = observationEvent
	updateGenerationEvent = observationEvent
	createEventEvent      = observationEvent
	createScoreEvent      = scoreEvent
)

// ============================================================================
// Backpressure Sentinel Error
// ============================================================================

// ErrBackpressure is returned when an event is rejected due to backpressure.
var ErrBackpressure = &APIError{
	StatusCode: 503,
	Message:    "event rejected due to queue backpressure",
}

// Note: Most batch processing methods (batchProcessor, processBatchRequest, sendBatch,
// waitForQueueSpace, estimateQueueSize, addEventToQueue, handleQueueFull)
// are provided by the embedded *pkgclient.Client.
//
// queueEvent is a wrapper that converts root's ingestionEvent to pkgclient.IngestionEvent.
func (c *Client) queueEvent(ctx context.Context, event ingestionEvent) error {
	// Convert root ingestionEvent to pkgclient.IngestionEvent
	pkgEvent := PkgClientIngestionEvent{
		ID:        event.ID,
		Type:      event.Type,
		Timestamp: PkgTime{Time: event.Timestamp.Time},
		Body:      event.Body,
	}
	return c.Client.QueueEvent(ctx, pkgEvent)
}
