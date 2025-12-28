package langfuse

import (
	"sync"
	"testing"
	"time"
)

// metricsTestRecorder is a test implementation of Metrics
type metricsTestRecorder struct {
	mu        sync.Mutex
	counters  map[string]int64
	durations map[string]time.Duration
	gauges    map[string]float64
}

func newMetricsTestRecorder() *metricsTestRecorder {
	return &metricsTestRecorder{
		counters:  make(map[string]int64),
		durations: make(map[string]time.Duration),
		gauges:    make(map[string]float64),
	}
}

func (m *metricsTestRecorder) IncrementCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[name] += value
}

func (m *metricsTestRecorder) RecordDuration(name string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.durations[name] = duration
}

func (m *metricsTestRecorder) SetGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.gauges[name] = value
}

func (m *metricsTestRecorder) getCounter(name string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.counters[name]
}

func (m *metricsTestRecorder) getDuration(name string) time.Duration {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.durations[name]
}

func (m *metricsTestRecorder) getGauge(name string) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.gauges[name]
}

func TestMetricsRecorder_Creation(t *testing.T) {
	t.Run("with nil metrics", func(t *testing.T) {
		r := NewMetricsRecorder(nil)
		if r.IsEnabled() {
			t.Error("IsEnabled() should return false for nil metrics")
		}
	})

	t.Run("with metrics", func(t *testing.T) {
		r := NewMetricsRecorder(newMetricsTestRecorder())
		if !r.IsEnabled() {
			t.Error("IsEnabled() should return true for non-nil metrics")
		}
	})
}

func TestMetricsRecorder_RecordQueueState(t *testing.T) {
	m := newMetricsTestRecorder()
	r := NewMetricsRecorder(m)

	r.RecordQueueState(50, 100)

	if m.getGauge(DefaultInternalMetrics.QueueDepth) != 50 {
		t.Errorf("QueueDepth = %v, want 50", m.getGauge(DefaultInternalMetrics.QueueDepth))
	}
	if m.getGauge(DefaultInternalMetrics.QueueCapacity) != 100 {
		t.Errorf("QueueCapacity = %v, want 100", m.getGauge(DefaultInternalMetrics.QueueCapacity))
	}
	if m.getGauge(DefaultInternalMetrics.QueueUtilization) != 0.5 {
		t.Errorf("QueueUtilization = %v, want 0.5", m.getGauge(DefaultInternalMetrics.QueueUtilization))
	}
}

func TestMetricsRecorder_RecordEventQueued(t *testing.T) {
	m := newMetricsTestRecorder()
	r := NewMetricsRecorder(m)

	r.RecordEventQueued(5)
	r.RecordEventQueued(3)

	if m.getCounter(DefaultInternalMetrics.EventsQueued) != 8 {
		t.Errorf("EventsQueued = %v, want 8", m.getCounter(DefaultInternalMetrics.EventsQueued))
	}
}

func TestMetricsRecorder_RecordEventDropped(t *testing.T) {
	m := newMetricsTestRecorder()
	r := NewMetricsRecorder(m)

	r.RecordEventDropped(2)

	if m.getCounter(DefaultInternalMetrics.EventsDropped) != 2 {
		t.Errorf("EventsDropped = %v, want 2", m.getCounter(DefaultInternalMetrics.EventsDropped))
	}
}

func TestMetricsRecorder_RecordBatchSend(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := newMetricsTestRecorder()
		r := NewMetricsRecorder(m)

		r.RecordBatchSend(10, 100*time.Millisecond, true, 0)

		if m.getCounter(DefaultInternalMetrics.BatchSuccesses) != 1 {
			t.Errorf("BatchSuccesses = %v, want 1", m.getCounter(DefaultInternalMetrics.BatchSuccesses))
		}
		if m.getCounter(DefaultInternalMetrics.EventsSent) != 10 {
			t.Errorf("EventsSent = %v, want 10", m.getCounter(DefaultInternalMetrics.EventsSent))
		}
		if m.getDuration(DefaultInternalMetrics.BatchDuration) != 100*time.Millisecond {
			t.Errorf("BatchDuration = %v, want 100ms", m.getDuration(DefaultInternalMetrics.BatchDuration))
		}
	})

	t.Run("failure with retries", func(t *testing.T) {
		m := newMetricsTestRecorder()
		r := NewMetricsRecorder(m)

		r.RecordBatchSend(5, 50*time.Millisecond, false, 3)

		if m.getCounter(DefaultInternalMetrics.BatchFailures) != 1 {
			t.Errorf("BatchFailures = %v, want 1", m.getCounter(DefaultInternalMetrics.BatchFailures))
		}
		if m.getCounter(DefaultInternalMetrics.BatchRetries) != 3 {
			t.Errorf("BatchRetries = %v, want 3", m.getCounter(DefaultInternalMetrics.BatchRetries))
		}
	})
}

func TestMetricsRecorder_RecordHTTPRequest(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantMetric string
	}{
		{"2xx", 200, DefaultInternalMetrics.HTTP2xx},
		{"4xx", 400, DefaultInternalMetrics.HTTP4xx},
		{"5xx", 500, DefaultInternalMetrics.HTTP5xx},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := newMetricsTestRecorder()
			r := NewMetricsRecorder(m)

			r.RecordHTTPRequest(tt.statusCode, 100*time.Millisecond)

			if m.getCounter(tt.wantMetric) != 1 {
				t.Errorf("%s = %v, want 1", tt.wantMetric, m.getCounter(tt.wantMetric))
			}
		})
	}
}

func TestMetricsRecorder_RecordHTTPRetry(t *testing.T) {
	m := newMetricsTestRecorder()
	r := NewMetricsRecorder(m)

	r.RecordHTTPRetry()
	r.RecordHTTPRetry()

	if m.getCounter(DefaultInternalMetrics.HTTPRequestRetries) != 2 {
		t.Errorf("HTTPRequestRetries = %v, want 2", m.getCounter(DefaultInternalMetrics.HTTPRequestRetries))
	}
}

func TestMetricsRecorder_RecordCircuitState(t *testing.T) {
	m := newMetricsTestRecorder()
	r := NewMetricsRecorder(m)

	r.RecordCircuitState(CircuitOpen)

	if m.getGauge(DefaultInternalMetrics.CircuitState) != float64(CircuitOpen) {
		t.Errorf("CircuitState = %v, want %v", m.getGauge(DefaultInternalMetrics.CircuitState), float64(CircuitOpen))
	}
}

func TestMetricsRecorder_RecordCircuitTrip(t *testing.T) {
	m := newMetricsTestRecorder()
	r := NewMetricsRecorder(m)

	r.RecordCircuitTrip()

	if m.getCounter(DefaultInternalMetrics.CircuitTrips) != 1 {
		t.Errorf("CircuitTrips = %v, want 1", m.getCounter(DefaultInternalMetrics.CircuitTrips))
	}
}

func TestMetricsRecorder_RecordHookExecution(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		m := newMetricsTestRecorder()
		r := NewMetricsRecorder(m)

		r.RecordHookExecution(10*time.Millisecond, true, false)

		if m.getCounter(DefaultInternalMetrics.HookFailures) != 0 {
			t.Errorf("HookFailures = %v, want 0", m.getCounter(DefaultInternalMetrics.HookFailures))
		}
	})

	t.Run("failure", func(t *testing.T) {
		m := newMetricsTestRecorder()
		r := NewMetricsRecorder(m)

		r.RecordHookExecution(10*time.Millisecond, false, false)

		if m.getCounter(DefaultInternalMetrics.HookFailures) != 1 {
			t.Errorf("HookFailures = %v, want 1", m.getCounter(DefaultInternalMetrics.HookFailures))
		}
	})

	t.Run("panic", func(t *testing.T) {
		m := newMetricsTestRecorder()
		r := NewMetricsRecorder(m)

		r.RecordHookExecution(10*time.Millisecond, false, true)

		if m.getCounter(DefaultInternalMetrics.HookPanics) != 1 {
			t.Errorf("HookPanics = %v, want 1", m.getCounter(DefaultInternalMetrics.HookPanics))
		}
	})
}

func TestMetricsRecorder_RecordUptime(t *testing.T) {
	m := newMetricsTestRecorder()
	r := NewMetricsRecorder(m)

	time.Sleep(10 * time.Millisecond)
	r.RecordUptime()

	uptime := m.getGauge(DefaultInternalMetrics.ClientUptime)
	if uptime < 0.01 {
		t.Errorf("ClientUptime = %v, want >= 0.01", uptime)
	}
}

func TestMetricsRecorder_RecordShutdown(t *testing.T) {
	m := newMetricsTestRecorder()
	r := NewMetricsRecorder(m)

	r.RecordShutdown(500 * time.Millisecond)

	if m.getDuration(DefaultInternalMetrics.ShutdownDuration) != 500*time.Millisecond {
		t.Errorf("ShutdownDuration = %v, want 500ms", m.getDuration(DefaultInternalMetrics.ShutdownDuration))
	}
}

func TestMetricsRecorder_RecordAsyncError(t *testing.T) {
	m := newMetricsTestRecorder()
	r := NewMetricsRecorder(m)

	r.RecordAsyncError()
	r.RecordAsyncError()

	if m.getCounter(DefaultInternalMetrics.AsyncErrorsTotal) != 2 {
		t.Errorf("AsyncErrorsTotal = %v, want 2", m.getCounter(DefaultInternalMetrics.AsyncErrorsTotal))
	}
}

func TestMetricsRecorder_RecordIDGenerationFailure(t *testing.T) {
	m := newMetricsTestRecorder()
	r := NewMetricsRecorder(m)

	r.RecordIDGenerationFailure()

	if m.getCounter(DefaultInternalMetrics.IDGenerationFailures) != 1 {
		t.Errorf("IDGenerationFailures = %v, want 1", m.getCounter(DefaultInternalMetrics.IDGenerationFailures))
	}
}

func TestMetricsRecorder_NilRecorder(t *testing.T) {
	var r *MetricsRecorder

	// These should not panic
	r.RecordQueueState(50, 100)
	r.RecordEventQueued(5)
	r.RecordEventDropped(2)
	r.RecordBatchSend(10, 100*time.Millisecond, true, 0)
	r.RecordHTTPRequest(200, 100*time.Millisecond)
	r.RecordHTTPRetry()
	r.RecordCircuitState(CircuitOpen)
	r.RecordCircuitTrip()
	r.RecordHookExecution(10*time.Millisecond, true, false)
	r.RecordUptime()
	r.RecordShutdown(500 * time.Millisecond)
	r.RecordAsyncError()
	r.RecordIDGenerationFailure()
}

func TestMetricsRecorderWithNames(t *testing.T) {
	customNames := InternalMetrics{
		QueueDepth:    "custom.queue.depth",
		QueueCapacity: "custom.queue.capacity",
	}

	m := newMetricsTestRecorder()
	r := NewMetricsRecorderWithNames(m, customNames)

	r.RecordQueueState(25, 50)

	if m.getGauge("custom.queue.depth") != 25 {
		t.Errorf("custom.queue.depth = %v, want 25", m.getGauge("custom.queue.depth"))
	}
}

// MetricsAggregator tests

func TestMetricsAggregator_Creation(t *testing.T) {
	a := NewMetricsAggregator()
	if a == nil {
		t.Fatal("NewMetricsAggregator returned nil")
	}
}

func TestMetricsAggregator_IncrementCounters(t *testing.T) {
	a := NewMetricsAggregator()

	a.IncrementEventsQueued(10)
	a.IncrementEventsSent(5)
	a.IncrementEventsDropped(2)
	a.IncrementBatchSuccess()
	a.IncrementBatchFailure()
	a.IncrementHookFailure()
	a.IncrementAsyncError()

	snapshot := a.Snapshot()

	if snapshot.EventsQueued != 10 {
		t.Errorf("EventsQueued = %v, want 10", snapshot.EventsQueued)
	}
	if snapshot.EventsSent != 5 {
		t.Errorf("EventsSent = %v, want 5", snapshot.EventsSent)
	}
	if snapshot.EventsDropped != 2 {
		t.Errorf("EventsDropped = %v, want 2", snapshot.EventsDropped)
	}
	if snapshot.BatchSuccesses != 1 {
		t.Errorf("BatchSuccesses = %v, want 1", snapshot.BatchSuccesses)
	}
	if snapshot.BatchFailures != 1 {
		t.Errorf("BatchFailures = %v, want 1", snapshot.BatchFailures)
	}
	if snapshot.HookFailures != 1 {
		t.Errorf("HookFailures = %v, want 1", snapshot.HookFailures)
	}
	if snapshot.AsyncErrors != 1 {
		t.Errorf("AsyncErrors = %v, want 1", snapshot.AsyncErrors)
	}
}

func TestMetricsAggregator_IncrementHTTP(t *testing.T) {
	a := NewMetricsAggregator()

	a.IncrementHTTP(200)
	a.IncrementHTTP(201)
	a.IncrementHTTP(400)
	a.IncrementHTTP(500)
	a.IncrementHTTP(503)

	snapshot := a.Snapshot()

	if snapshot.HTTP2xx != 2 {
		t.Errorf("HTTP2xx = %v, want 2", snapshot.HTTP2xx)
	}
	if snapshot.HTTP4xx != 1 {
		t.Errorf("HTTP4xx = %v, want 1", snapshot.HTTP4xx)
	}
	if snapshot.HTTP5xx != 2 {
		t.Errorf("HTTP5xx = %v, want 2", snapshot.HTTP5xx)
	}
}

func TestMetricsAggregator_SetQueueState(t *testing.T) {
	a := NewMetricsAggregator()

	a.SetQueueState(75, 100)

	snapshot := a.Snapshot()

	if snapshot.QueueDepth != 75 {
		t.Errorf("QueueDepth = %v, want 75", snapshot.QueueDepth)
	}
	if snapshot.QueueCapacity != 100 {
		t.Errorf("QueueCapacity = %v, want 100", snapshot.QueueCapacity)
	}
	if snapshot.QueueUtilization != 0.75 {
		t.Errorf("QueueUtilization = %v, want 0.75", snapshot.QueueUtilization)
	}
}

func TestMetricsAggregator_Snapshot(t *testing.T) {
	a := NewMetricsAggregator()

	a.IncrementEventsQueued(100)
	a.SetQueueState(50, 100)

	snapshot := a.Snapshot()

	if snapshot.Timestamp.IsZero() {
		t.Error("Snapshot timestamp should not be zero")
	}
	if snapshot.UptimeSeconds < 0 {
		t.Errorf("UptimeSeconds = %v, should be >= 0", snapshot.UptimeSeconds)
	}
}

func TestMetricsAggregator_Reset(t *testing.T) {
	a := NewMetricsAggregator()

	a.IncrementEventsQueued(100)
	a.IncrementEventsSent(50)
	a.IncrementHTTP(200)
	a.Reset()

	snapshot := a.Snapshot()

	if snapshot.EventsQueued != 0 {
		t.Errorf("EventsQueued after reset = %v, want 0", snapshot.EventsQueued)
	}
	if snapshot.EventsSent != 0 {
		t.Errorf("EventsSent after reset = %v, want 0", snapshot.EventsSent)
	}
	if snapshot.HTTP2xx != 0 {
		t.Errorf("HTTP2xx after reset = %v, want 0", snapshot.HTTP2xx)
	}
}

func TestMetricsAggregator_FlushTo(t *testing.T) {
	m := newMetricsTestRecorder()
	r := NewMetricsRecorder(m)
	a := NewMetricsAggregator()

	a.IncrementEventsQueued(100)
	a.SetQueueState(30, 100)

	snapshot := a.FlushTo(r)

	if snapshot.EventsQueued != 100 {
		t.Errorf("Snapshot EventsQueued = %v, want 100", snapshot.EventsQueued)
	}

	if m.getGauge(DefaultInternalMetrics.QueueDepth) != 30 {
		t.Errorf("Recorder QueueDepth = %v, want 30", m.getGauge(DefaultInternalMetrics.QueueDepth))
	}
}

func TestMetricsAggregator_ConcurrentAccess(t *testing.T) {
	a := NewMetricsAggregator()

	var wg sync.WaitGroup
	const goroutines = 10
	const iterations = 100

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				a.IncrementEventsQueued(1)
				a.IncrementEventsSent(1)
				a.IncrementHTTP(200)
				a.SetQueueState(j, 100)
				_ = a.Snapshot()
			}
		}()
	}

	wg.Wait()

	snapshot := a.Snapshot()
	expected := int64(goroutines * iterations)

	if snapshot.EventsQueued != expected {
		t.Errorf("EventsQueued = %v, want %v", snapshot.EventsQueued, expected)
	}
	if snapshot.EventsSent != expected {
		t.Errorf("EventsSent = %v, want %v", snapshot.EventsSent, expected)
	}
}

func TestCircuitState_Values(t *testing.T) {
	// CircuitState constants are defined in circuitbreaker.go
	if CircuitClosed != 0 {
		t.Errorf("CircuitClosed = %v, want 0", CircuitClosed)
	}
	// Note: CircuitOpen is 1, CircuitHalfOpen is 2 in circuitbreaker.go
	if CircuitOpen != 1 {
		t.Errorf("CircuitOpen = %v, want 1", CircuitOpen)
	}
	if CircuitHalfOpen != 2 {
		t.Errorf("CircuitHalfOpen = %v, want 2", CircuitHalfOpen)
	}
}

func TestDefaultInternalMetrics(t *testing.T) {
	// Verify all metric names are populated
	m := DefaultInternalMetrics

	if m.QueueDepth == "" {
		t.Error("QueueDepth metric name is empty")
	}
	if m.BatchSize == "" {
		t.Error("BatchSize metric name is empty")
	}
	if m.EventsQueued == "" {
		t.Error("EventsQueued metric name is empty")
	}
	if m.HTTPRequestDuration == "" {
		t.Error("HTTPRequestDuration metric name is empty")
	}
	if m.CircuitState == "" {
		t.Error("CircuitState metric name is empty")
	}
	if m.HookDuration == "" {
		t.Error("HookDuration metric name is empty")
	}
	if m.ClientUptime == "" {
		t.Error("ClientUptime metric name is empty")
	}
	if m.AsyncErrorsTotal == "" {
		t.Error("AsyncErrorsTotal metric name is empty")
	}
	if m.IDGenerationFailures == "" {
		t.Error("IDGenerationFailures metric name is empty")
	}
}
