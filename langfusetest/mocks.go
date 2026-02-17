package langfusetest

import (
	"sync"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
)

// Compile-time interface assertions to catch drift between mock implementations
// and the actual interfaces they're supposed to implement.
var (
	_ langfuse.Metrics = (*MockMetrics)(nil)
	_ langfuse.Logger  = (*MockLogger)(nil)
)

// MockMetrics is a mock implementation of the Metrics interface for testing.
// It records all metrics operations for later verification.
type MockMetrics struct {
	mu       sync.Mutex
	Counters map[string]int64
	Gauges   map[string]float64
	Timings  map[string][]int64 // Duration in nanoseconds
}

// NewMockMetrics creates a new mock metrics collector.
func NewMockMetrics() *MockMetrics {
	return &MockMetrics{
		Counters: make(map[string]int64),
		Gauges:   make(map[string]float64),
		Timings:  make(map[string][]int64),
	}
}

// IncrementCounter implements Metrics.IncrementCounter.
func (m *MockMetrics) IncrementCounter(name string, value int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Counters[name] += value
}

// RecordDuration implements Metrics.RecordDuration.
func (m *MockMetrics) RecordDuration(name string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Timings[name] = append(m.Timings[name], duration.Nanoseconds())
}

// SetGauge implements Metrics.SetGauge.
func (m *MockMetrics) SetGauge(name string, value float64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Gauges[name] = value
}

// GetCounter returns the value of a counter.
func (m *MockMetrics) GetCounter(name string) int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Counters[name]
}

// GetGauge returns the value of a gauge.
func (m *MockMetrics) GetGauge(name string) float64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.Gauges[name]
}

// GetTimings returns all recorded timings for a metric.
func (m *MockMetrics) GetTimings(name string) []int64 {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]int64{}, m.Timings[name]...)
}

// Reset clears all recorded metrics.
func (m *MockMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Counters = make(map[string]int64)
	m.Gauges = make(map[string]float64)
	m.Timings = make(map[string][]int64)
}

// MockLogger is a mock implementation of the Logger interface for testing.
// It captures all log messages for later verification.
type MockLogger struct {
	mu       sync.Mutex
	Messages []string
}

// NewMockLogger creates a new mock logger.
func NewMockLogger() *MockLogger {
	return &MockLogger{
		Messages: make([]string, 0),
	}
}

// Printf implements Logger.Printf.
func (l *MockLogger) Printf(format string, v ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Messages = append(l.Messages, format)
}

// GetMessages returns all logged messages.
func (l *MockLogger) GetMessages() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string{}, l.Messages...)
}

// MessageCount returns the number of logged messages.
func (l *MockLogger) MessageCount() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.Messages)
}

// Reset clears all logged messages.
func (l *MockLogger) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.Messages = make([]string, 0)
}
