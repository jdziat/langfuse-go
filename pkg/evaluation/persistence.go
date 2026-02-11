package evaluation

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

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

// EventPersistence handles saving and loading events to/from disk.
type EventPersistence struct {
	dir string
	mu  sync.Mutex
}

// NewEventPersistence creates a new event persistence handler.
func NewEventPersistence(dir string) (*EventPersistence, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create persistence directory: %w", err)
	}

	return &EventPersistence{
		dir: dir,
	}, nil
}

// Save persists a batch of events to disk.
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
		return fmt.Errorf("failed to marshal events: %w", err)
	}

	filename := fmt.Sprintf("events_%d.json", time.Now().UnixNano())
	path := filepath.Join(p.dir, filename)

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write events file: %w", err)
	}

	return nil
}

// Load reads all persisted events from disk.
func (p *EventPersistence) Load() ([]PersistedEvent, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entries, err := os.ReadDir(p.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read persistence directory: %w", err)
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
func (p *EventPersistence) Clear() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	entries, err := os.ReadDir(p.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read persistence directory: %w", err)
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
			continue // Best effort cleanup
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
func (p *EventPersistence) ClearOlderThan(age time.Duration) (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	entries, err := os.ReadDir(p.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to read persistence directory: %w", err)
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

// PersistenceConfig configures the event persistence behavior.
type PersistenceConfig struct {
	// Directory is the path where events are persisted.
	Directory string

	// MaxAge is the maximum age of persisted events.
	MaxAge time.Duration

	// MaxFiles is the maximum number of event batch files to keep.
	MaxFiles int

	// CleanupInterval is how often to run automatic cleanup.
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

// NewManagedPersistence creates a new managed persistence handler.
func NewManagedPersistence(config PersistenceConfig) (*ManagedPersistence, error) {
	if config.Directory == "" {
		return nil, fmt.Errorf("persistence directory is required")
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

// Config returns the current persistence configuration.
func (mp *ManagedPersistence) Config() PersistenceConfig {
	return mp.config
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
			mp.RunCleanup()
		}
	}
}

// RunCleanup performs a single cleanup pass (exported for testing).
func (mp *ManagedPersistence) RunCleanup() {
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

	// Sort by modification time (oldest first) and remove excess files
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
