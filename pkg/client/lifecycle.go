package client

import (
	pkglifecycle "github.com/jdziat/langfuse-go/pkg/lifecycle"
)

// Re-export lifecycle types from pkg/lifecycle.
// This allows pkg/client to use the shared lifecycle implementation.

// ClientState represents the current state of the client lifecycle.
type ClientState = pkglifecycle.ClientState

// Client state constants.
const (
	// ClientStateActive indicates the client is active and accepting events.
	ClientStateActive = pkglifecycle.ClientStateActive
	// ClientStateShuttingDown indicates the client is shutting down.
	ClientStateShuttingDown = pkglifecycle.ClientStateShuttingDown
	// ClientStateClosed indicates the client has been closed.
	ClientStateClosed = pkglifecycle.ClientStateClosed
)

// LifecycleManager handles client lifecycle with leak prevention.
type LifecycleManager = pkglifecycle.Manager

// LifecycleConfig configures the lifecycle manager.
type LifecycleConfig = pkglifecycle.Config

// LifecycleStats contains lifecycle statistics.
type LifecycleStats = pkglifecycle.Stats

// NewLifecycleManager creates a new lifecycle manager.
func NewLifecycleManager(cfg *LifecycleConfig) *LifecycleManager {
	return pkglifecycle.NewManager(cfg)
}

// ErrAlreadyClosed is returned when attempting operations on a closed manager.
// Note: In pkg/client context, we typically return ErrClientClosed instead for
// client-facing errors, but this is available for internal lifecycle operations.
var ErrAlreadyClosed = pkglifecycle.ErrAlreadyClosed
