package langfuse

import (
	"sync"
	"time"
)

// InternalMetrics defines all SDK internal metric names.
// These provide comprehensive visibility into SDK internals.
type InternalMetrics struct {
	// Queue metrics
	QueueDepth       string // Gauge: current queue depth
	QueueCapacity    string // Gauge: queue capacity
	QueueUtilization string // Gauge: queue utilization (0-1)

	// Batch metrics
	BatchSize      string // Histogram: events per batch
	BatchDuration  string // Histogram: time to send batch
	BatchRetries   string // Counter: retry attempts
	BatchSuccesses string // Counter: successful batches
	BatchFailures  string // Counter: failed batches

	// Event metrics
	EventsQueued  string // Counter: events added to queue
	EventsSent    string // Counter: events successfully sent
	EventsDropped string // Counter: events dropped

	// HTTP metrics
	HTTPRequestDuration string // Histogram: request latency
	HTTPRequestRetries  string // Counter: HTTP retries
	HTTP2xx             string // Counter: 2xx responses
	HTTP4xx             string // Counter: 4xx responses
	HTTP5xx             string // Counter: 5xx responses

	// Circuit breaker metrics
	CircuitState string // Gauge: 0=closed, 1=half-open, 2=open
	CircuitTrips string // Counter: times circuit opened

	// Hook metrics
	HookDuration string // Histogram: hook execution time
	HookFailures string // Counter: hook failures
	HookPanics   string // Counter: hook panics

	// Lifecycle metrics
	ClientUptime     string // Gauge: seconds since creation
	ShutdownDuration string // Histogram: shutdown time

	// Async error metrics
	AsyncErrorsTotal   string // Counter: total async errors
	AsyncErrorsDropped string // Counter: dropped async errors

	// ID generation metrics
	IDGenerationFailures string // Counter: crypto/rand failures
}

// DefaultInternalMetrics returns metric names with "langfuse." prefix.
var DefaultInternalMetrics = InternalMetrics{
	QueueDepth:           "langfuse.queue.depth",
	QueueCapacity:        "langfuse.queue.capacity",
	QueueUtilization:     "langfuse.queue.utilization",
	BatchSize:            "langfuse.batch.size",
	BatchDuration:        "langfuse.batch.duration_ms",
	BatchRetries:         "langfuse.batch.retries",
	BatchSuccesses:       "langfuse.batch.successes",
	BatchFailures:        "langfuse.batch.failures",
	EventsQueued:         "langfuse.events.queued",
	EventsSent:           "langfuse.events.sent",
	EventsDropped:        "langfuse.events.dropped",
	HTTPRequestDuration:  "langfuse.http.duration_ms",
	HTTPRequestRetries:   "langfuse.http.retries",
	HTTP2xx:              "langfuse.http.2xx",
	HTTP4xx:              "langfuse.http.4xx",
	HTTP5xx:              "langfuse.http.5xx",
	CircuitState:         "langfuse.circuit.state",
	CircuitTrips:         "langfuse.circuit.trips",
	HookDuration:         "langfuse.hook.duration_ms",
	HookFailures:         "langfuse.hook.failures",
	HookPanics:           "langfuse.hook.panics",
	ClientUptime:         "langfuse.client.uptime_seconds",
	ShutdownDuration:     "langfuse.shutdown.duration_ms",
	AsyncErrorsTotal:     "langfuse.async_errors.total",
	AsyncErrorsDropped:   "langfuse.async_errors.dropped",
	IDGenerationFailures: "langfuse.id.crypto_failures",
}

// MetricsRecorder wraps the Metrics interface with convenience methods
// for recording SDK internal metrics with consistent naming.
type MetricsRecorder struct {
	metrics Metrics
	names   InternalMetrics

	// Cached values for gauges that need periodic updates
	startTime time.Time
}

// NewMetricsRecorder creates a new metrics recorder.
// If metrics is nil, all recording methods are no-ops.
func NewMetricsRecorder(m Metrics) *MetricsRecorder {
	return &MetricsRecorder{
		metrics:   m,
		names:     DefaultInternalMetrics,
		startTime: time.Now(),
	}
}

// NewMetricsRecorderWithNames creates a metrics recorder with custom metric names.
func NewMetricsRecorderWithNames(m Metrics, names InternalMetrics) *MetricsRecorder {
	return &MetricsRecorder{
		metrics:   m,
		names:     names,
		startTime: time.Now(),
	}
}

// IsEnabled returns true if metrics recording is enabled.
func (r *MetricsRecorder) IsEnabled() bool {
	return r != nil && r.metrics != nil
}

// --- Queue Metrics ---

// RecordQueueState records the current queue state.
func (r *MetricsRecorder) RecordQueueState(depth, capacity int) {
	if !r.IsEnabled() {
		return
	}
	r.metrics.SetGauge(r.names.QueueDepth, float64(depth))
	r.metrics.SetGauge(r.names.QueueCapacity, float64(capacity))
	if capacity > 0 {
		r.metrics.SetGauge(r.names.QueueUtilization, float64(depth)/float64(capacity))
	}
}

// RecordEventQueued increments the events queued counter.
func (r *MetricsRecorder) RecordEventQueued(count int) {
	if !r.IsEnabled() {
		return
	}
	r.metrics.IncrementCounter(r.names.EventsQueued, int64(count))
}

// RecordEventDropped increments the events dropped counter.
func (r *MetricsRecorder) RecordEventDropped(count int) {
	if !r.IsEnabled() {
		return
	}
	r.metrics.IncrementCounter(r.names.EventsDropped, int64(count))
}

// --- Batch Metrics ---

// RecordBatchSend records metrics for a batch send operation.
func (r *MetricsRecorder) RecordBatchSend(size int, duration time.Duration, success bool, retries int) {
	if !r.IsEnabled() {
		return
	}
	r.metrics.RecordDuration(r.names.BatchDuration, duration)
	r.metrics.SetGauge(r.names.BatchSize, float64(size))

	if success {
		r.metrics.IncrementCounter(r.names.BatchSuccesses, 1)
		r.metrics.IncrementCounter(r.names.EventsSent, int64(size))
	} else {
		r.metrics.IncrementCounter(r.names.BatchFailures, 1)
	}

	if retries > 0 {
		r.metrics.IncrementCounter(r.names.BatchRetries, int64(retries))
	}
}

// RecordBatchSuccess records a successful batch send.
func (r *MetricsRecorder) RecordBatchSuccess(size int, duration time.Duration) {
	r.RecordBatchSend(size, duration, true, 0)
}

// RecordBatchFailure records a failed batch send.
func (r *MetricsRecorder) RecordBatchFailure(size int, duration time.Duration, retries int) {
	r.RecordBatchSend(size, duration, false, retries)
}

// --- HTTP Metrics ---

// RecordHTTPRequest records metrics for an HTTP request.
func (r *MetricsRecorder) RecordHTTPRequest(statusCode int, duration time.Duration) {
	if !r.IsEnabled() {
		return
	}
	r.metrics.RecordDuration(r.names.HTTPRequestDuration, duration)

	switch {
	case statusCode >= 200 && statusCode < 300:
		r.metrics.IncrementCounter(r.names.HTTP2xx, 1)
	case statusCode >= 400 && statusCode < 500:
		r.metrics.IncrementCounter(r.names.HTTP4xx, 1)
	case statusCode >= 500:
		r.metrics.IncrementCounter(r.names.HTTP5xx, 1)
	}
}

// RecordHTTPRetry increments the HTTP retry counter.
func (r *MetricsRecorder) RecordHTTPRetry() {
	if !r.IsEnabled() {
		return
	}
	r.metrics.IncrementCounter(r.names.HTTPRequestRetries, 1)
}

// --- Circuit Breaker Metrics ---

// RecordCircuitState records the current circuit breaker state.
// Uses CircuitState from circuitbreaker.go (CircuitClosed, CircuitOpen, CircuitHalfOpen).
func (r *MetricsRecorder) RecordCircuitState(state CircuitState) {
	if !r.IsEnabled() {
		return
	}
	r.metrics.SetGauge(r.names.CircuitState, float64(state))
}

// RecordCircuitTrip increments the circuit trip counter.
func (r *MetricsRecorder) RecordCircuitTrip() {
	if !r.IsEnabled() {
		return
	}
	r.metrics.IncrementCounter(r.names.CircuitTrips, 1)
}

// --- Hook Metrics ---

// RecordHookExecution records metrics for a hook execution.
func (r *MetricsRecorder) RecordHookExecution(duration time.Duration, success bool, panicked bool) {
	if !r.IsEnabled() {
		return
	}
	r.metrics.RecordDuration(r.names.HookDuration, duration)

	if !success {
		r.metrics.IncrementCounter(r.names.HookFailures, 1)
	}
	if panicked {
		r.metrics.IncrementCounter(r.names.HookPanics, 1)
	}
}

// RecordHookFailure increments the hook failure counter.
func (r *MetricsRecorder) RecordHookFailure() {
	if !r.IsEnabled() {
		return
	}
	r.metrics.IncrementCounter(r.names.HookFailures, 1)
}

// RecordHookPanic increments the hook panic counter.
func (r *MetricsRecorder) RecordHookPanic() {
	if !r.IsEnabled() {
		return
	}
	r.metrics.IncrementCounter(r.names.HookPanics, 1)
}

// --- Lifecycle Metrics ---

// RecordUptime records the client uptime.
func (r *MetricsRecorder) RecordUptime() {
	if !r.IsEnabled() {
		return
	}
	uptime := time.Since(r.startTime).Seconds()
	r.metrics.SetGauge(r.names.ClientUptime, uptime)
}

// RecordShutdown records the shutdown duration.
func (r *MetricsRecorder) RecordShutdown(duration time.Duration) {
	if !r.IsEnabled() {
		return
	}
	r.metrics.RecordDuration(r.names.ShutdownDuration, duration)
}

// --- Async Error Metrics ---

// RecordAsyncError increments the async error counter.
func (r *MetricsRecorder) RecordAsyncError() {
	if !r.IsEnabled() {
		return
	}
	r.metrics.IncrementCounter(r.names.AsyncErrorsTotal, 1)
}

// RecordAsyncErrorDropped increments the dropped async error counter.
func (r *MetricsRecorder) RecordAsyncErrorDropped() {
	if !r.IsEnabled() {
		return
	}
	r.metrics.IncrementCounter(r.names.AsyncErrorsDropped, 1)
}

// --- ID Generation Metrics ---

// RecordIDGenerationFailure increments the ID generation failure counter.
func (r *MetricsRecorder) RecordIDGenerationFailure() {
	if !r.IsEnabled() {
		return
	}
	r.metrics.IncrementCounter(r.names.IDGenerationFailures, 1)
}

// --- Aggregate Recording ---

// MetricsSnapshot represents a point-in-time snapshot of SDK metrics.
type MetricsSnapshot struct {
	Timestamp        time.Time
	QueueDepth       int
	QueueCapacity    int
	QueueUtilization float64
	EventsQueued     int64
	EventsSent       int64
	EventsDropped    int64
	BatchSuccesses   int64
	BatchFailures    int64
	HTTP2xx          int64
	HTTP4xx          int64
	HTTP5xx          int64
	HookFailures     int64
	AsyncErrors      int64
	UptimeSeconds    float64
}

// MetricsAggregator collects metrics for periodic reporting.
type MetricsAggregator struct {
	mu sync.RWMutex

	eventsQueued   int64
	eventsSent     int64
	eventsDropped  int64
	batchSuccesses int64
	batchFailures  int64
	http2xx        int64
	http4xx        int64
	http5xx        int64
	hookFailures   int64
	asyncErrors    int64
	queueDepth     int
	queueCapacity  int

	startTime time.Time
}

// NewMetricsAggregator creates a new metrics aggregator.
func NewMetricsAggregator() *MetricsAggregator {
	return &MetricsAggregator{
		startTime: time.Now(),
	}
}

// IncrementEventsQueued increments the events queued counter.
func (a *MetricsAggregator) IncrementEventsQueued(count int64) {
	a.mu.Lock()
	a.eventsQueued += count
	a.mu.Unlock()
}

// IncrementEventsSent increments the events sent counter.
func (a *MetricsAggregator) IncrementEventsSent(count int64) {
	a.mu.Lock()
	a.eventsSent += count
	a.mu.Unlock()
}

// IncrementEventsDropped increments the events dropped counter.
func (a *MetricsAggregator) IncrementEventsDropped(count int64) {
	a.mu.Lock()
	a.eventsDropped += count
	a.mu.Unlock()
}

// IncrementBatchSuccess increments the batch success counter.
func (a *MetricsAggregator) IncrementBatchSuccess() {
	a.mu.Lock()
	a.batchSuccesses++
	a.mu.Unlock()
}

// IncrementBatchFailure increments the batch failure counter.
func (a *MetricsAggregator) IncrementBatchFailure() {
	a.mu.Lock()
	a.batchFailures++
	a.mu.Unlock()
}

// IncrementHTTP records an HTTP response status code category.
func (a *MetricsAggregator) IncrementHTTP(statusCode int) {
	a.mu.Lock()
	switch {
	case statusCode >= 200 && statusCode < 300:
		a.http2xx++
	case statusCode >= 400 && statusCode < 500:
		a.http4xx++
	case statusCode >= 500:
		a.http5xx++
	}
	a.mu.Unlock()
}

// IncrementHookFailure increments the hook failure counter.
func (a *MetricsAggregator) IncrementHookFailure() {
	a.mu.Lock()
	a.hookFailures++
	a.mu.Unlock()
}

// IncrementAsyncError increments the async error counter.
func (a *MetricsAggregator) IncrementAsyncError() {
	a.mu.Lock()
	a.asyncErrors++
	a.mu.Unlock()
}

// SetQueueState sets the current queue state.
func (a *MetricsAggregator) SetQueueState(depth, capacity int) {
	a.mu.Lock()
	a.queueDepth = depth
	a.queueCapacity = capacity
	a.mu.Unlock()
}

// Snapshot returns a point-in-time snapshot of all metrics.
func (a *MetricsAggregator) Snapshot() MetricsSnapshot {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var utilization float64
	if a.queueCapacity > 0 {
		utilization = float64(a.queueDepth) / float64(a.queueCapacity)
	}

	return MetricsSnapshot{
		Timestamp:        time.Now(),
		QueueDepth:       a.queueDepth,
		QueueCapacity:    a.queueCapacity,
		QueueUtilization: utilization,
		EventsQueued:     a.eventsQueued,
		EventsSent:       a.eventsSent,
		EventsDropped:    a.eventsDropped,
		BatchSuccesses:   a.batchSuccesses,
		BatchFailures:    a.batchFailures,
		HTTP2xx:          a.http2xx,
		HTTP4xx:          a.http4xx,
		HTTP5xx:          a.http5xx,
		HookFailures:     a.hookFailures,
		AsyncErrors:      a.asyncErrors,
		UptimeSeconds:    time.Since(a.startTime).Seconds(),
	}
}

// Reset resets all counters to zero. Useful for periodic reporting.
func (a *MetricsAggregator) Reset() {
	a.mu.Lock()
	a.eventsQueued = 0
	a.eventsSent = 0
	a.eventsDropped = 0
	a.batchSuccesses = 0
	a.batchFailures = 0
	a.http2xx = 0
	a.http4xx = 0
	a.http5xx = 0
	a.hookFailures = 0
	a.asyncErrors = 0
	a.mu.Unlock()
}

// FlushTo flushes all metrics to a MetricsRecorder and resets.
func (a *MetricsAggregator) FlushTo(r *MetricsRecorder) MetricsSnapshot {
	snapshot := a.Snapshot()

	if r.IsEnabled() {
		r.RecordQueueState(snapshot.QueueDepth, snapshot.QueueCapacity)
		r.RecordUptime()

		// Note: Counter increments should be done at the time of the event,
		// not in batch. This is mainly for gauge updates.
	}

	return snapshot
}
