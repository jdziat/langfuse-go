// Package lifecycle provides client lifecycle management with leak prevention.
//
// The LifecycleManager tracks client state, monitors for idle clients,
// and provides warnings when clients are not properly shut down.
package lifecycle

import (
	"context"
	"errors"
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
	RecordDuration(name string, d time.Duration)
}

// ErrAlreadyClosed is returned when attempting to shutdown an already closed manager.
var ErrAlreadyClosed = errors.New("lifecycle: already closed or shutting down")

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

// Config configures the lifecycle manager.
type Config struct {
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

// Stats contains lifecycle statistics.
type Stats struct {
	State        ClientState
	CreatedAt    time.Time
	LastActivity time.Time
	Uptime       time.Duration
	IdleDuration time.Duration
}

// Manager handles client lifecycle with leak prevention.
// It tracks client state, monitors for idle clients, and provides
// warnings when clients are not properly shut down.
type Manager struct {
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

// NewManager creates a new lifecycle manager.
func NewManager(cfg *Config) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	now := time.Now()

	m := &Manager{
		createdAt:           now,
		ctx:                 ctx,
		cancel:              cancel,
		idleWarningDuration: cfg.IdleWarningDuration,
		logger:              cfg.Logger,
		metrics:             cfg.Metrics,
		onStateChange:       cfg.OnStateChange,
	}

	m.state.Store(int32(ClientStateActive))
	m.lastActivity.Store(now.UnixNano())

	// Start idle detector if configured
	if cfg.IdleWarningDuration > 0 && cfg.Logger != nil {
		m.wg.Add(1)
		go m.idleDetector()
	}

	// Record creation metric
	if cfg.Metrics != nil {
		cfg.Metrics.IncrementCounter("langfuse.client.created", 1)
	}

	return m
}

// idleDetector monitors for idle clients and logs warnings.
func (m *Manager) idleDetector() {
	defer m.wg.Done()

	// Check at half the idle duration for responsiveness
	checkInterval := m.idleWarningDuration / 2
	if checkInterval < time.Second {
		checkInterval = time.Second
	}

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			if m.State() != ClientStateActive {
				return
			}

			lastActivity := time.Unix(0, m.lastActivity.Load())
			idle := time.Since(lastActivity)

			if idle > m.idleWarningDuration && m.warningFired.CompareAndSwap(false, true) {
				m.logger.Printf(
					"WARNING: Langfuse client has been idle for %v without Shutdown() being called. "+
						"This may indicate a goroutine leak. Always call client.Shutdown(ctx) when done. "+
						"Created at: %s",
					idle.Round(time.Second),
					m.createdAt.Format(time.RFC3339),
				)

				if m.metrics != nil {
					m.metrics.IncrementCounter("langfuse.client.idle_warning", 1)
				}
			}
		}
	}
}

// State returns the current client state.
func (m *Manager) State() ClientState {
	return ClientState(m.state.Load())
}

// IsActive returns true if the client is active.
func (m *Manager) IsActive() bool {
	return m.State() == ClientStateActive
}

// IsClosed returns true if the client is closed.
func (m *Manager) IsClosed() bool {
	return m.State() == ClientStateClosed
}

// RecordActivity updates the last activity timestamp.
// Call this when the client performs any operation.
func (m *Manager) RecordActivity() {
	m.lastActivity.Store(time.Now().UnixNano())
}

// LastActivity returns the time of the last recorded activity.
func (m *Manager) LastActivity() time.Time {
	return time.Unix(0, m.lastActivity.Load())
}

// Uptime returns the duration since the client was created.
func (m *Manager) Uptime() time.Duration {
	return time.Since(m.createdAt)
}

// IdleDuration returns the duration since the last activity.
func (m *Manager) IdleDuration() time.Duration {
	return time.Since(m.LastActivity())
}

// CreatedAt returns the time the client was created.
func (m *Manager) CreatedAt() time.Time {
	return m.createdAt
}

// Context returns the lifecycle context.
// This context is cancelled when shutdown begins.
func (m *Manager) Context() context.Context {
	return m.ctx
}

// transition attempts to transition to a new state.
// Returns true if the transition was successful.
func (m *Manager) transition(from, to ClientState) bool {
	if m.state.CompareAndSwap(int32(from), int32(to)) {
		if m.onStateChange != nil {
			m.onStateChange(from, to)
		}
		if m.metrics != nil {
			m.metrics.SetGauge("langfuse.client.state", float64(to))
		}
		return true
	}
	return false
}

// BeginShutdown initiates the shutdown process.
// Returns ErrAlreadyClosed if already shutting down or closed.
func (m *Manager) BeginShutdown() error {
	if !m.transition(ClientStateActive, ClientStateShuttingDown) {
		state := m.State()
		if state == ClientStateShuttingDown || state == ClientStateClosed {
			return ErrAlreadyClosed
		}
		return ErrAlreadyClosed
	}

	// Cancel the context to signal all goroutines
	m.cancel()

	if m.metrics != nil {
		m.metrics.IncrementCounter("langfuse.client.shutdown_initiated", 1)
		m.metrics.RecordDuration("langfuse.client.uptime", m.Uptime())
	}

	return nil
}

// CompleteShutdown marks the shutdown as complete.
func (m *Manager) CompleteShutdown() {
	m.transition(ClientStateShuttingDown, ClientStateClosed)

	// Wait for idle detector to stop
	m.wg.Wait()

	if m.metrics != nil {
		m.metrics.IncrementCounter("langfuse.client.shutdown_complete", 1)
	}
}

// WaitGroup returns the lifecycle WaitGroup for tracking goroutines.
func (m *Manager) WaitGroup() *sync.WaitGroup {
	return &m.wg
}

// Stats returns current lifecycle statistics.
func (m *Manager) Stats() Stats {
	return Stats{
		State:        m.State(),
		CreatedAt:    m.createdAt,
		LastActivity: m.LastActivity(),
		Uptime:       m.Uptime(),
		IdleDuration: m.IdleDuration(),
	}
}
