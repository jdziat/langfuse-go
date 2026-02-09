package ingestion

import (
	"sync"
	"sync/atomic"
	"time"
)

// Logger is a minimal logging interface.
type Logger interface {
	Printf(format string, v ...any)
}

// Metrics is a minimal metrics interface.
type Metrics interface {
	IncrementCounter(name string, value int64)
	SetGauge(name string, value float64)
}

// BackpressureLevel indicates the severity of queue backpressure.
type BackpressureLevel int

const (
	// BackpressureNone indicates the queue is operating normally.
	BackpressureNone BackpressureLevel = iota
	// BackpressureWarning indicates the queue is filling up but not critical.
	BackpressureWarning
	// BackpressureCritical indicates the queue is nearly full.
	BackpressureCritical
	// BackpressureOverflow indicates events are being dropped.
	BackpressureOverflow
)

// String returns a human-readable representation of the backpressure level.
func (l BackpressureLevel) String() string {
	switch l {
	case BackpressureNone:
		return "none"
	case BackpressureWarning:
		return "warning"
	case BackpressureCritical:
		return "critical"
	case BackpressureOverflow:
		return "overflow"
	default:
		return "unknown"
	}
}

// BackpressureThreshold defines when backpressure levels are triggered.
type BackpressureThreshold struct {
	// WarningPercent triggers warning level (default: 50%).
	WarningPercent float64
	// CriticalPercent triggers critical level (default: 80%).
	CriticalPercent float64
	// OverflowPercent triggers overflow level (default: 95%).
	OverflowPercent float64
}

// DefaultBackpressureThreshold returns sensible default thresholds.
func DefaultBackpressureThreshold() BackpressureThreshold {
	return BackpressureThreshold{
		WarningPercent:  50.0,
		CriticalPercent: 80.0,
		OverflowPercent: 95.0,
	}
}

// QueueState represents the current state of the event queue.
type QueueState struct {
	// Size is the current number of items in the queue.
	Size int
	// Capacity is the maximum queue capacity.
	Capacity int
	// Level is the current backpressure level.
	Level BackpressureLevel
	// PercentFull is the percentage of queue capacity in use.
	PercentFull float64
	// Timestamp is when this state was captured.
	Timestamp time.Time
}

// BackpressureCallback is called when backpressure level changes.
type BackpressureCallback func(state QueueState)

// QueueMonitor monitors queue state and signals backpressure.
// It provides proactive notification when the queue is filling up,
// allowing callers to take action before events are dropped.
type QueueMonitor struct {
	threshold BackpressureThreshold
	capacity  int

	// Callbacks
	mu       sync.RWMutex
	callback BackpressureCallback

	// Metrics and logging
	metrics Metrics
	logger  Logger

	// State
	currentLevel atomic.Int32
	lastState    atomic.Value // stores QueueState

	// Statistics
	warningCount  atomic.Int64
	criticalCount atomic.Int64
	overflowCount atomic.Int64
	stateChanges  atomic.Int64
}

// QueueMonitorConfig configures the QueueMonitor.
type QueueMonitorConfig struct {
	// Threshold defines when backpressure levels are triggered.
	Threshold BackpressureThreshold

	// Capacity is the maximum queue capacity.
	Capacity int

	// OnBackpressure is called when backpressure level changes.
	OnBackpressure BackpressureCallback

	// Metrics is used for backpressure metrics.
	Metrics Metrics

	// Logger is used for backpressure logging.
	Logger Logger
}

// NewQueueMonitor creates a new queue monitor.
func NewQueueMonitor(cfg *QueueMonitorConfig) *QueueMonitor {
	if cfg == nil {
		cfg = &QueueMonitorConfig{}
	}

	threshold := cfg.Threshold
	if threshold.WarningPercent <= 0 {
		threshold.WarningPercent = 50.0
	}
	if threshold.CriticalPercent <= 0 {
		threshold.CriticalPercent = 80.0
	}
	if threshold.OverflowPercent <= 0 {
		threshold.OverflowPercent = 95.0
	}

	capacity := cfg.Capacity
	if capacity <= 0 {
		capacity = 1000 // Default capacity
	}

	m := &QueueMonitor{
		threshold: threshold,
		capacity:  capacity,
		callback:  cfg.OnBackpressure,
		metrics:   cfg.Metrics,
		logger:    cfg.Logger,
	}

	// Initialize state
	m.currentLevel.Store(int32(BackpressureNone))
	m.lastState.Store(QueueState{
		Capacity:  capacity,
		Level:     BackpressureNone,
		Timestamp: time.Now(),
	})

	return m
}

// Update updates the monitor with the current queue size.
// It returns the new backpressure level.
func (m *QueueMonitor) Update(size int) BackpressureLevel {
	percentFull := float64(size) / float64(m.capacity) * 100.0

	var newLevel BackpressureLevel
	switch {
	case percentFull >= m.threshold.OverflowPercent:
		newLevel = BackpressureOverflow
	case percentFull >= m.threshold.CriticalPercent:
		newLevel = BackpressureCritical
	case percentFull >= m.threshold.WarningPercent:
		newLevel = BackpressureWarning
	default:
		newLevel = BackpressureNone
	}

	state := QueueState{
		Size:        size,
		Capacity:    m.capacity,
		Level:       newLevel,
		PercentFull: percentFull,
		Timestamp:   time.Now(),
	}

	oldLevel := BackpressureLevel(m.currentLevel.Swap(int32(newLevel)))
	m.lastState.Store(state)

	// Track statistics
	switch newLevel {
	case BackpressureWarning:
		m.warningCount.Add(1)
	case BackpressureCritical:
		m.criticalCount.Add(1)
	case BackpressureOverflow:
		m.overflowCount.Add(1)
	}

	// Handle level change
	if oldLevel != newLevel {
		m.stateChanges.Add(1)
		m.onLevelChange(oldLevel, newLevel, state)
	}

	// Update metrics
	if m.metrics != nil {
		m.metrics.SetGauge("langfuse.queue.size", float64(size))
		m.metrics.SetGauge("langfuse.queue.percent_full", percentFull)
		m.metrics.IncrementCounter("langfuse.queue.updates", 1)
	}

	return newLevel
}

// onLevelChange handles backpressure level transitions.
func (m *QueueMonitor) onLevelChange(from, to BackpressureLevel, state QueueState) {
	// Log the transition
	if m.logger != nil {
		if to > BackpressureNone {
			m.logger.Printf("langfuse: backpressure level changed from %s to %s (queue: %d/%d, %.1f%%)",
				from, to, state.Size, state.Capacity, state.PercentFull)
		} else {
			m.logger.Printf("langfuse: backpressure cleared (queue: %d/%d, %.1f%%)",
				state.Size, state.Capacity, state.PercentFull)
		}
	}

	// Update metrics
	if m.metrics != nil {
		m.metrics.IncrementCounter("langfuse.queue.level_changes", 1)
		m.metrics.SetGauge("langfuse.queue.level", float64(to))
	}

	// Invoke callback
	m.mu.RLock()
	callback := m.callback
	m.mu.RUnlock()

	if callback != nil {
		callback(state)
	}
}

// Level returns the current backpressure level.
func (m *QueueMonitor) Level() BackpressureLevel {
	return BackpressureLevel(m.currentLevel.Load())
}

// State returns the current queue state.
func (m *QueueMonitor) State() QueueState {
	return m.lastState.Load().(QueueState)
}

// SetCallback sets the backpressure callback.
// This replaces any previously set callback.
func (m *QueueMonitor) SetCallback(fn BackpressureCallback) {
	m.mu.Lock()
	m.callback = fn
	m.mu.Unlock()
}

// IsHealthy returns true if the queue is not experiencing backpressure.
func (m *QueueMonitor) IsHealthy() bool {
	return m.Level() == BackpressureNone
}

// IsCritical returns true if backpressure is critical or worse.
func (m *QueueMonitor) IsCritical() bool {
	level := m.Level()
	return level >= BackpressureCritical
}

// ShouldBlock returns true if operations should block due to backpressure.
// This is true when the queue is at overflow level.
func (m *QueueMonitor) ShouldBlock() bool {
	return m.Level() == BackpressureOverflow
}

// QueueMonitorStats contains statistics about queue monitoring.
type QueueMonitorStats struct {
	CurrentLevel  BackpressureLevel
	WarningCount  int64
	CriticalCount int64
	OverflowCount int64
	StateChanges  int64
	LastState     QueueState
}

// Stats returns current queue monitoring statistics.
func (m *QueueMonitor) Stats() QueueMonitorStats {
	return QueueMonitorStats{
		CurrentLevel:  m.Level(),
		WarningCount:  m.warningCount.Load(),
		CriticalCount: m.criticalCount.Load(),
		OverflowCount: m.overflowCount.Load(),
		StateChanges:  m.stateChanges.Load(),
		LastState:     m.State(),
	}
}

// BackpressureHandler provides a higher-level API for handling backpressure.
// It wraps a QueueMonitor and provides blocking/non-blocking behavior.
type BackpressureHandler struct {
	monitor     *QueueMonitor
	blockOnFull bool
	dropOnFull  bool
	logger      Logger
	metrics     Metrics

	// Statistics
	blockedCount atomic.Int64
	droppedCount atomic.Int64
}

// BackpressureHandlerConfig configures the BackpressureHandler.
type BackpressureHandlerConfig struct {
	// Monitor is the underlying queue monitor.
	Monitor *QueueMonitor

	// BlockOnFull blocks Send calls when queue is at overflow.
	BlockOnFull bool

	// DropOnFull drops events when queue is at overflow (if not blocking).
	DropOnFull bool

	// Logger is used for logging.
	Logger Logger

	// Metrics is used for metrics.
	Metrics Metrics
}

// NewBackpressureHandler creates a new backpressure handler.
func NewBackpressureHandler(cfg *BackpressureHandlerConfig) *BackpressureHandler {
	if cfg == nil {
		cfg = &BackpressureHandlerConfig{}
	}

	monitor := cfg.Monitor
	if monitor == nil {
		monitor = NewQueueMonitor(nil)
	}

	return &BackpressureHandler{
		monitor:     monitor,
		blockOnFull: cfg.BlockOnFull,
		dropOnFull:  cfg.DropOnFull,
		logger:      cfg.Logger,
		metrics:     cfg.Metrics,
	}
}

// BackpressureDecision represents the decision made by the handler.
type BackpressureDecision int

const (
	// DecisionAllow indicates the event should be queued.
	DecisionAllow BackpressureDecision = iota
	// DecisionBlock indicates the caller should wait.
	DecisionBlock
	// DecisionDrop indicates the event should be dropped.
	DecisionDrop
)

// String returns a human-readable representation of the decision.
func (d BackpressureDecision) String() string {
	switch d {
	case DecisionAllow:
		return "allow"
	case DecisionBlock:
		return "block"
	case DecisionDrop:
		return "drop"
	default:
		return "unknown"
	}
}

// Decide returns a decision for whether an event should be queued.
func (h *BackpressureHandler) Decide(queueSize int) BackpressureDecision {
	level := h.monitor.Update(queueSize)

	switch level {
	case BackpressureOverflow:
		if h.blockOnFull {
			h.blockedCount.Add(1)
			if h.metrics != nil {
				h.metrics.IncrementCounter("langfuse.backpressure.blocked", 1)
			}
			return DecisionBlock
		}
		if h.dropOnFull {
			h.droppedCount.Add(1)
			if h.metrics != nil {
				h.metrics.IncrementCounter("langfuse.backpressure.dropped", 1)
			}
			if h.logger != nil {
				h.logger.Printf("langfuse: dropping event due to backpressure (queue overflow)")
			}
			return DecisionDrop
		}
		return DecisionAllow

	case BackpressureCritical:
		// At critical level, we allow but may warn
		if h.metrics != nil {
			h.metrics.IncrementCounter("langfuse.backpressure.critical_events", 1)
		}
		return DecisionAllow

	default:
		return DecisionAllow
	}
}

// Monitor returns the underlying queue monitor.
func (h *BackpressureHandler) Monitor() *QueueMonitor {
	return h.monitor
}

// BackpressureHandlerStats contains statistics about backpressure handling.
type BackpressureHandlerStats struct {
	BlockedCount int64
	DroppedCount int64
	MonitorStats QueueMonitorStats
	BlockOnFull  bool
	DropOnFull   bool
}

// Stats returns current backpressure handler statistics.
func (h *BackpressureHandler) Stats() BackpressureHandlerStats {
	return BackpressureHandlerStats{
		BlockedCount: h.blockedCount.Load(),
		DroppedCount: h.droppedCount.Load(),
		MonitorStats: h.monitor.Stats(),
		BlockOnFull:  h.blockOnFull,
		DropOnFull:   h.dropOnFull,
	}
}
