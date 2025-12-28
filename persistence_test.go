package langfuse

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewEventPersistence(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "events")

	p, err := NewEventPersistence(subdir)
	if err != nil {
		t.Fatalf("NewEventPersistence() error = %v", err)
	}
	if p == nil {
		t.Fatal("NewEventPersistence() returned nil")
	}

	// Verify directory was created
	if _, err := os.Stat(subdir); os.IsNotExist(err) {
		t.Error("persistence directory was not created")
	}
}

func TestEventPersistence_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	p, err := NewEventPersistence(dir)
	if err != nil {
		t.Fatalf("NewEventPersistence() error = %v", err)
	}

	// Create test events
	events := []PersistedEvent{
		{
			ID:        "event-1",
			Type:      "trace-create",
			Timestamp: time.Now(),
			Body:      map[string]any{"name": "test-trace"},
		},
		{
			ID:        "event-2",
			Type:      "span-create",
			Timestamp: time.Now(),
			Body:      map[string]any{"name": "test-span"},
		},
	}

	// Save events
	if err := p.Save(events); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load events
	loaded, err := p.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded) != len(events) {
		t.Errorf("Load() returned %d events, want %d", len(loaded), len(events))
	}

	// Verify event data
	if loaded[0].ID != "event-1" {
		t.Errorf("loaded[0].ID = %q, want %q", loaded[0].ID, "event-1")
	}
	if loaded[0].Type != "trace-create" {
		t.Errorf("loaded[0].Type = %q, want %q", loaded[0].Type, "trace-create")
	}
}

func TestEventPersistence_SaveEmpty(t *testing.T) {
	dir := t.TempDir()
	p, err := NewEventPersistence(dir)
	if err != nil {
		t.Fatalf("NewEventPersistence() error = %v", err)
	}

	// Save empty events should not create file
	if err := p.Save([]PersistedEvent{}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	count, _ := p.Count()
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}
}

func TestEventPersistence_LoadEmpty(t *testing.T) {
	dir := t.TempDir()
	p, err := NewEventPersistence(dir)
	if err != nil {
		t.Fatalf("NewEventPersistence() error = %v", err)
	}

	// Load from empty directory
	events, err := p.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(events) != 0 {
		t.Errorf("Load() returned %d events, want 0", len(events))
	}
}

func TestEventPersistence_LoadNonExistent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nonexistent")
	p := &EventPersistence{dir: dir}

	// Should not error on non-existent directory
	events, err := p.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if events != nil && len(events) != 0 {
		t.Error("Load() should return nil or empty for non-existent directory")
	}
}

func TestEventPersistence_Clear(t *testing.T) {
	dir := t.TempDir()
	p, err := NewEventPersistence(dir)
	if err != nil {
		t.Fatalf("NewEventPersistence() error = %v", err)
	}

	// Save some events
	events := []PersistedEvent{
		{ID: "event-1", Type: "test", Timestamp: time.Now()},
	}
	p.Save(events)
	p.Save(events) // Save twice

	count, _ := p.Count()
	if count != 2 {
		t.Fatalf("Count() = %d, want 2", count)
	}

	// Clear
	if err := p.Clear(); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	count, _ = p.Count()
	if count != 0 {
		t.Errorf("Count() after clear = %d, want 0", count)
	}
}

func TestEventPersistence_Count(t *testing.T) {
	dir := t.TempDir()
	p, err := NewEventPersistence(dir)
	if err != nil {
		t.Fatalf("NewEventPersistence() error = %v", err)
	}

	events := []PersistedEvent{
		{ID: "event-1", Type: "test", Timestamp: time.Now()},
	}

	// Save multiple batches
	for i := 0; i < 5; i++ {
		if err := p.Save(events); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		time.Sleep(time.Millisecond) // Ensure different filenames
	}

	count, err := p.Count()
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 5 {
		t.Errorf("Count() = %d, want 5", count)
	}
}

func TestEventPersistence_ClearOlderThan(t *testing.T) {
	dir := t.TempDir()
	p, err := NewEventPersistence(dir)
	if err != nil {
		t.Fatalf("NewEventPersistence() error = %v", err)
	}

	events := []PersistedEvent{
		{ID: "event-1", Type: "test", Timestamp: time.Now()},
	}

	// Save an event
	if err := p.Save(events); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Clear events older than 1 hour (should keep the event)
	removed, err := p.ClearOlderThan(time.Hour)
	if err != nil {
		t.Fatalf("ClearOlderThan() error = %v", err)
	}
	if removed != 0 {
		t.Errorf("ClearOlderThan(1h) removed = %d, want 0", removed)
	}

	count, _ := p.Count()
	if count != 1 {
		t.Errorf("Count() = %d, want 1", count)
	}
}

func TestEventPersistence_MultipleBatches(t *testing.T) {
	dir := t.TempDir()
	p, err := NewEventPersistence(dir)
	if err != nil {
		t.Fatalf("NewEventPersistence() error = %v", err)
	}

	// Save first batch
	batch1 := []PersistedEvent{
		{ID: "batch1-event1", Type: "test", Timestamp: time.Now()},
		{ID: "batch1-event2", Type: "test", Timestamp: time.Now()},
	}
	if err := p.Save(batch1); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	time.Sleep(time.Millisecond)

	// Save second batch
	batch2 := []PersistedEvent{
		{ID: "batch2-event1", Type: "test", Timestamp: time.Now()},
	}
	if err := p.Save(batch2); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load all events
	loaded, err := p.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded) != 3 {
		t.Errorf("Load() returned %d events, want 3", len(loaded))
	}
}

func TestEventPersistence_IgnoresNonJSONFiles(t *testing.T) {
	dir := t.TempDir()
	p, err := NewEventPersistence(dir)
	if err != nil {
		t.Fatalf("NewEventPersistence() error = %v", err)
	}

	// Create a non-JSON file
	nonJSON := filepath.Join(dir, "readme.txt")
	if err := os.WriteFile(nonJSON, []byte("not json"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Create a subdirectory
	subdir := filepath.Join(dir, "subdir")
	if err := os.Mkdir(subdir, 0755); err != nil {
		t.Fatalf("failed to create test directory: %v", err)
	}

	// Count should be 0
	count, err := p.Count()
	if err != nil {
		t.Fatalf("Count() error = %v", err)
	}
	if count != 0 {
		t.Errorf("Count() = %d, want 0", count)
	}
}

func TestEventPersistence_SkipsInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	p, err := NewEventPersistence(dir)
	if err != nil {
		t.Fatalf("NewEventPersistence() error = %v", err)
	}

	// Create an invalid JSON file
	invalidJSON := filepath.Join(dir, "invalid.json")
	if err := os.WriteFile(invalidJSON, []byte("not valid json"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	// Save a valid batch
	events := []PersistedEvent{
		{ID: "valid-event", Type: "test", Timestamp: time.Now()},
	}
	if err := p.Save(events); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Load should only return valid events
	loaded, err := p.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(loaded) != 1 {
		t.Errorf("Load() returned %d events, want 1", len(loaded))
	}
	if len(loaded) > 0 && loaded[0].ID != "valid-event" {
		t.Errorf("loaded[0].ID = %q, want %q", loaded[0].ID, "valid-event")
	}
}

func TestDefaultPersistenceConfig(t *testing.T) {
	config := DefaultPersistenceConfig()

	if config.MaxAge != 7*24*time.Hour {
		t.Errorf("MaxAge = %v, want 7d", config.MaxAge)
	}
	if config.MaxFiles != 1000 {
		t.Errorf("MaxFiles = %d, want 1000", config.MaxFiles)
	}
	if config.CleanupInterval != time.Hour {
		t.Errorf("CleanupInterval = %v, want 1h", config.CleanupInterval)
	}
}

func TestManagedPersistence_New(t *testing.T) {
	dir := t.TempDir()

	mp, err := NewManagedPersistence(PersistenceConfig{
		Directory:       dir,
		CleanupInterval: time.Hour,
	})
	if err != nil {
		t.Fatalf("NewManagedPersistence() error = %v", err)
	}
	defer mp.Stop()

	if mp == nil {
		t.Fatal("NewManagedPersistence() returned nil")
	}
}

func TestManagedPersistence_RequiresDirectory(t *testing.T) {
	_, err := NewManagedPersistence(PersistenceConfig{})
	if err == nil {
		t.Error("NewManagedPersistence() should require directory")
	}
}

func TestManagedPersistence_AppliesDefaults(t *testing.T) {
	dir := t.TempDir()

	mp, err := NewManagedPersistence(PersistenceConfig{
		Directory: dir,
	})
	if err != nil {
		t.Fatalf("NewManagedPersistence() error = %v", err)
	}
	defer mp.Stop()

	if mp.config.MaxAge != 7*24*time.Hour {
		t.Errorf("MaxAge = %v, want 7d", mp.config.MaxAge)
	}
	if mp.config.MaxFiles != 1000 {
		t.Errorf("MaxFiles = %d, want 1000", mp.config.MaxFiles)
	}
}

func TestManagedPersistence_Stop(t *testing.T) {
	dir := t.TempDir()

	mp, err := NewManagedPersistence(PersistenceConfig{
		Directory:       dir,
		CleanupInterval: time.Millisecond,
	})
	if err != nil {
		t.Fatalf("NewManagedPersistence() error = %v", err)
	}

	// Wait a bit for cleanup loop to run
	time.Sleep(10 * time.Millisecond)

	// Stop should not hang
	done := make(chan struct{})
	go func() {
		mp.Stop()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(time.Second):
		t.Error("Stop() hung")
	}
}

func TestManagedPersistence_LimitsFileCount(t *testing.T) {
	dir := t.TempDir()

	mp, err := NewManagedPersistence(PersistenceConfig{
		Directory:       dir,
		MaxFiles:        3,
		CleanupInterval: time.Hour, // Don't auto-run
	})
	if err != nil {
		t.Fatalf("NewManagedPersistence() error = %v", err)
	}
	defer mp.Stop()

	events := []PersistedEvent{
		{ID: "event", Type: "test", Timestamp: time.Now()},
	}

	// Save 5 batches
	for i := 0; i < 5; i++ {
		if err := mp.Save(events); err != nil {
			t.Fatalf("Save() error = %v", err)
		}
		time.Sleep(2 * time.Millisecond) // Ensure different timestamps
	}

	count, _ := mp.Count()
	if count != 5 {
		t.Fatalf("Count() = %d, want 5 before cleanup", count)
	}

	// Run cleanup
	mp.runCleanup()

	count, _ = mp.Count()
	if count != 3 {
		t.Errorf("Count() = %d, want 3 after cleanup", count)
	}
}

func TestEventsToPersistedEvents(t *testing.T) {
	now := time.Now()
	events := []ingestionEvent{
		{
			ID:        "event-1",
			Type:      eventTypeTraceCreate,
			Timestamp: Time{Time: now},
			Body: &createTraceEvent{
				ID:   "trace-1",
				Name: "test-trace",
			},
		},
	}

	persisted := EventsToPersistedEvents(events)

	if len(persisted) != 1 {
		t.Fatalf("len(persisted) = %d, want 1", len(persisted))
	}

	if persisted[0].ID != "event-1" {
		t.Errorf("ID = %q, want %q", persisted[0].ID, "event-1")
	}
	if persisted[0].Type != string(eventTypeTraceCreate) {
		t.Errorf("Type = %q, want %q", persisted[0].Type, string(eventTypeTraceCreate))
	}
	if !persisted[0].Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v, want %v", persisted[0].Timestamp, now)
	}
}
