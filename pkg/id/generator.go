package id

import (
	"crypto/rand"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

// Logger interface for logging within the id package.
// This is a minimal interface that avoids circular dependencies.
type Logger interface {
	Printf(format string, v ...any)
}

// Metrics interface for recording metrics within the id package.
// This is a minimal interface that avoids circular dependencies.
type Metrics interface {
	IncrementCounter(name string, value int64)
}

// IDGenerationMode controls how IDs are generated when crypto/rand fails.
type IDGenerationMode int

const (
	// IDModeFallback uses an atomic counter fallback when crypto/rand fails.
	// This is the default mode for backwards compatibility.
	IDModeFallback IDGenerationMode = iota

	// IDModeStrict returns an error when crypto/rand fails.
	// Recommended for production deployments where ID uniqueness is critical.
	IDModeStrict
)

// String returns a string representation of the ID generation mode.
func (m IDGenerationMode) String() string {
	switch m {
	case IDModeFallback:
		return "fallback"
	case IDModeStrict:
		return "strict"
	default:
		return "unknown"
	}
}

var (
	// fallbackCounter provides uniqueness when combined with timestamp.
	// Incremented atomically to ensure uniqueness across goroutines.
	fallbackCounter uint64

	// processID is cached at startup for fallback ID generation.
	processID = os.Getpid()

	// cryptoFailures tracks crypto/rand failures for monitoring.
	cryptoFailures atomic.Int64
)

// IDGenerator generates unique IDs with configurable failure handling.
type IDGenerator struct {
	mode    IDGenerationMode
	metrics Metrics
	logger  Logger
}

// IDGeneratorConfig configures the ID generator.
type IDGeneratorConfig struct {
	// Mode controls behavior when crypto/rand fails.
	Mode IDGenerationMode

	// Metrics is used to track ID generation statistics.
	Metrics Metrics

	// Logger is used to log warnings.
	Logger Logger
}

// NewIDGenerator creates an ID generator with the specified configuration.
func NewIDGenerator(cfg *IDGeneratorConfig) *IDGenerator {
	if cfg == nil {
		cfg = &IDGeneratorConfig{Mode: IDModeFallback}
	}
	return &IDGenerator{
		mode:    cfg.Mode,
		metrics: cfg.Metrics,
		logger:  cfg.Logger,
	}
}

// Generate creates a new unique ID.
// Returns an error only in IDModeStrict when crypto/rand fails.
func (g *IDGenerator) Generate() (string, error) {
	id, err := g.generateCryptoUUID()
	if err == nil {
		if g.metrics != nil {
			g.metrics.IncrementCounter("langfuse.id.generated", 1)
		}
		return id, nil
	}

	// Track crypto failure
	failures := cryptoFailures.Add(1)
	if g.metrics != nil {
		g.metrics.IncrementCounter("langfuse.id.crypto_failures", 1)
	}

	switch g.mode {
	case IDModeStrict:
		return "", fmt.Errorf("langfuse: crypto/rand failed (strict mode enabled, %d total failures): %w", failures, err)

	case IDModeFallback:
		// Log warning on first failure
		if failures == 1 && g.logger != nil {
			g.logger.Printf("WARNING: crypto/rand failed, using fallback ID generation: %v", err)
		}

		fallbackID := g.generateFallbackID()
		if g.metrics != nil {
			g.metrics.IncrementCounter("langfuse.id.fallback_used", 1)
		}
		return fallbackID, nil

	default:
		return "", fmt.Errorf("langfuse: unknown ID generation mode: %d", g.mode)
	}
}

// MustGenerate generates an ID or panics on failure.
// Use only when ID generation must succeed (e.g., in strict mode tests).
func (g *IDGenerator) MustGenerate() string {
	id, err := g.Generate()
	if err != nil {
		panic(err)
	}
	return id
}

// generateCryptoUUID generates a UUID v4 using crypto/rand.
func (g *IDGenerator) generateCryptoUUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	// Set version (4) and variant bits per RFC 4122
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// generateFallbackID generates a unique ID without crypto/rand.
// Combines timestamp, atomic counter, and process ID for uniqueness.
// Format: fb-{timestamp_hex}-{counter_hex}-{pid}
func (g *IDGenerator) generateFallbackID() string {
	counter := atomic.AddUint64(&fallbackCounter, 1)
	now := time.Now()

	// Use both UnixNano and counter for uniqueness
	// Format provides ~18 quintillion unique IDs per nanosecond per process
	return fmt.Sprintf("fb-%x-%08x-%d",
		now.UnixNano(),
		counter,
		processID,
	)
}

// CryptoFailureCount returns the total number of crypto/rand failures.
func CryptoFailureCount() int64 {
	return cryptoFailures.Load()
}

// ResetCryptoFailureCount resets the failure counter (for testing).
func ResetCryptoFailureCount() {
	cryptoFailures.Store(0)
}

// defaultIDGenerator is the package-level ID generator.
// Initialized lazily with fallback mode for backwards compatibility.
var defaultIDGenerator = &IDGenerator{mode: IDModeFallback}

// SetDefaultIDGenerator sets the package-level ID generator.
// Call this early in application startup to configure ID generation.
func SetDefaultIDGenerator(g *IDGenerator) {
	defaultIDGenerator = g
}

// GenerateID generates a unique ID using the default generator.
// This is the primary entry point for ID generation in the SDK.
func GenerateID() (string, error) {
	return defaultIDGenerator.Generate()
}

// MustGenerateID generates an ID or panics.
// Use only when ID generation must succeed.
func MustGenerateID() string {
	return defaultIDGenerator.MustGenerate()
}

// GenerateIDInternal is used internally and maintains backwards compatibility.
// It never returns an error, using fallback mode implicitly.
func GenerateIDInternal() string {
	id, err := defaultIDGenerator.Generate()
	if err != nil {
		// This should only happen in strict mode, which is not the default
		// Fall back to the fallback generator as a last resort
		return (&IDGenerator{mode: IDModeFallback}).generateFallbackID()
	}
	return id
}

// IsFallbackID returns true if the ID was generated using the fallback method.
// Fallback IDs start with "fb-".
func IsFallbackID(id string) bool {
	return len(id) > 3 && id[0:3] == "fb-"
}

// IDStats contains statistics about ID generation.
type IDStats struct {
	CryptoFailures int64
	Mode           IDGenerationMode
}

// Stats returns current ID generation statistics.
func (g *IDGenerator) Stats() IDStats {
	return IDStats{
		CryptoFailures: cryptoFailures.Load(),
		Mode:           g.mode,
	}
}
