package langfuse

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventPersistence handles saving and loading events to/from disk.
// This provides resilience against network failures by persisting
// events that couldn't be sent.
//
// Example usage:
//
//	persistence, err := langfuse.NewEventPersistence("/var/lib/langfuse/events")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	client, err := langfuse.New(
//	    publicKey, secretKey,
//	    langfuse.WithErrorHandler(func(err error) {
//	        // Save failed events for retry
//	        if events, ok := langfuse.GetFailedEvents(err); ok {
//	            persistence.Save(events)
//	        }
//	    }),
//	)
//
//	// On startup, recover and resend failed events
//	events, err := persistence.Load()
//	if err == nil && len(events) > 0 {
//	    client.ResendEvents(ctx, events)
//	    persistence.Clear()
//	}
type EventPersistence struct {
	dir string
	mu  sync.Mutex
}

// PersistedEvent represents an event saved to disk.
type PersistedEvent struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	Timestamp time.Time      `json:"timestamp"`
	Body      map[string]any `json:"body"`
}

// PersistedBatch represents a batch of events saved to disk.
type PersistedBatch struct {
	SavedAt time.Time        `json:"saved_at"`
	Events  []PersistedEvent `json:"events"`
}

// NewEventPersistence creates a new event persistence handler.
// The directory will be created if it doesn't exist.
func NewEventPersistence(dir string) (*EventPersistence, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("langfuse: failed to create persistence directory: %w", err)
	}

	return &EventPersistence{
		dir: dir,
	}, nil
}

// Save persists a batch of events to disk.
// Events are saved as JSON files with timestamp-based names.
func (p *EventPersistence) Save(events []PersistedEvent) error {
	if len(events) == 0 {
		return nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	batch := PersistedBatch{
		SavedAt: time.Now(),
		Events:  events,
	}

	data, err := json.MarshalIndent(batch, "", "  ")
	if err != nil {
		return fmt.Errorf("langfuse: failed to marshal events: %w", err)
	}

	filename := fmt.Sprintf("events_%d.json", time.Now().UnixNano())
	path := filepath.Join(p.dir, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("langfuse: failed to write events file: %w", err)
	}

	return nil
}

// Load reads all persisted events from disk.
// Returns events from all saved batch files.
func (p *EventPersistence) Load() ([]PersistedEvent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entries, err := os.ReadDir(p.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("langfuse: failed to read persistence directory: %w", err)
	}

	var allEvents []PersistedEvent

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(p.dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue // Skip files we can't read
		}

		var batch PersistedBatch
		if err := json.Unmarshal(data, &batch); err != nil {
			continue // Skip invalid JSON files
		}

		allEvents = append(allEvents, batch.Events...)
	}

	return allEvents, nil
}

// Clear removes all persisted event files.
// Call this after successfully resending recovered events.
func (p *EventPersistence) Clear() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	entries, err := os.ReadDir(p.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("langfuse: failed to read persistence directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(p.dir, entry.Name())
		if err := os.Remove(path); err != nil {
			// Log but don't fail - best effort cleanup
			continue
		}
	}

	return nil
}

// Count returns the number of persisted event batches.
func (p *EventPersistence) Count() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entries, err := os.ReadDir(p.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}

	count := 0
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			count++
		}
	}

	return count, nil
}

// ClearOlderThan removes event files older than the specified duration.
// Useful for cleaning up stale events that are no longer relevant.
func (p *EventPersistence) ClearOlderThan(age time.Duration) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entries, err := os.ReadDir(p.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("langfuse: failed to read persistence directory: %w", err)
	}

	cutoff := time.Now().Add(-age)
	removed := 0

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		path := filepath.Join(p.dir, entry.Name())

		// Check file modification time
		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(path); err == nil {
				removed++
			}
		}
	}

	return removed, nil
}

// EventsToPersistedEvents converts ingestion events to persisted format.
// This is a helper for saving failed events from the error handler.
func EventsToPersistedEvents(events []ingestionEvent) []PersistedEvent {
	result := make([]PersistedEvent, 0, len(events))

	for _, e := range events {
		// Convert body to map if possible
		var bodyMap map[string]any
		if data, err := json.Marshal(e.Body); err == nil {
			if err := json.Unmarshal(data, &bodyMap); err != nil {
				// If unmarshal fails, store the original body type info for debugging
				bodyMap = map[string]any{
					"_conversionError": err.Error(),
					"_originalType":    fmt.Sprintf("%T", e.Body),
				}
			}
		}

		result = append(result, PersistedEvent{
			ID:        e.ID,
			Type:      e.Type,
			Timestamp: e.Timestamp.Time,
			Body:      bodyMap,
		})
	}

	return result
}

// PersistenceConfig configures the event persistence behavior.
type PersistenceConfig struct {
	// Directory is the path where events are persisted.
	Directory string

	// MaxAge is the maximum age of persisted events.
	// Events older than this are automatically cleaned up.
	// Default: 7 days
	MaxAge time.Duration

	// MaxFiles is the maximum number of event batch files to keep.
	// Oldest files are removed when this limit is exceeded.
	// Default: 1000
	MaxFiles int

	// CleanupInterval is how often to run automatic cleanup.
	// Default: 1 hour
	CleanupInterval time.Duration
}

// DefaultPersistenceConfig returns a PersistenceConfig with sensible defaults.
func DefaultPersistenceConfig() PersistenceConfig {
	return PersistenceConfig{
		MaxAge:          7 * 24 * time.Hour,
		MaxFiles:        1000,
		CleanupInterval: time.Hour,
	}
}

// ManagedPersistence wraps EventPersistence with automatic cleanup.
type ManagedPersistence struct {
	*EventPersistence
	config   PersistenceConfig
	stopChan chan struct{}
	wg       sync.WaitGroup
}

// NewManagedPersistence creates a new managed persistence handler
// with automatic cleanup of old events.
func NewManagedPersistence(config PersistenceConfig) (*ManagedPersistence, error) {
	if config.Directory == "" {
		return nil, fmt.Errorf("langfuse: persistence directory is required")
	}

	// Apply defaults
	if config.MaxAge == 0 {
		config.MaxAge = 7 * 24 * time.Hour
	}
	if config.MaxFiles == 0 {
		config.MaxFiles = 1000
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = time.Hour
	}

	ep, err := NewEventPersistence(config.Directory)
	if err != nil {
		return nil, err
	}

	mp := &ManagedPersistence{
		EventPersistence: ep,
		config:           config,
		stopChan:         make(chan struct{}),
	}

	// Start cleanup goroutine
	mp.wg.Add(1)
	go mp.cleanupLoop()

	return mp, nil
}

// cleanupLoop runs periodic cleanup of old events.
func (mp *ManagedPersistence) cleanupLoop() {
	defer mp.wg.Done()

	ticker := time.NewTicker(mp.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-mp.stopChan:
			return
		case <-ticker.C:
			mp.runCleanup()
		}
	}
}

// runCleanup performs a single cleanup pass.
func (mp *ManagedPersistence) runCleanup() {
	// Remove old events
	_, _ = mp.ClearOlderThan(mp.config.MaxAge)

	// Limit total file count
	mp.limitFileCount()
}

// limitFileCount removes oldest files if count exceeds MaxFiles.
func (mp *ManagedPersistence) limitFileCount() {
	mp.mu.Lock()
	defer mp.mu.Unlock()

	entries, err := os.ReadDir(mp.dir)
	if err != nil {
		return
	}

	// Filter to only JSON files
	var files []os.DirEntry
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			files = append(files, entry)
		}
	}

	// If under limit, nothing to do
	if len(files) <= mp.config.MaxFiles {
		return
	}

	// Sort by modification time (oldest first)
	// and remove excess files
	type fileInfo struct {
		name    string
		modTime time.Time
	}

	infos := make([]fileInfo, 0, len(files))
	for _, f := range files {
		info, err := f.Info()
		if err != nil {
			continue
		}
		infos = append(infos, fileInfo{
			name:    f.Name(),
			modTime: info.ModTime(),
		})
	}

	// Simple bubble sort (files list is typically small)
	for i := 0; i < len(infos)-1; i++ {
		for j := i + 1; j < len(infos); j++ {
			if infos[j].modTime.Before(infos[i].modTime) {
				infos[i], infos[j] = infos[j], infos[i]
			}
		}
	}

	// Remove oldest files
	toRemove := len(infos) - mp.config.MaxFiles
	for i := 0; i < toRemove; i++ {
		path := filepath.Join(mp.dir, infos[i].name)
		_ = os.Remove(path)
	}
}

// Stop stops the cleanup goroutine and waits for it to finish.
func (mp *ManagedPersistence) Stop() {
	close(mp.stopChan)
	mp.wg.Wait()
}
