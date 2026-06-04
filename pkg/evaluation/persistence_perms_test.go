package evaluation

import (
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"
)

// TestPersistence_FilePermissions verifies that persisted batches (which may
// contain PII or secrets in cleartext) are written with owner-only file
// permissions (0600) inside an owner-only directory (0700).
func TestPersistence_FilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX file mode bits are not meaningful on Windows")
	}

	// Clear the umask so the asserted permission bits are not masked away,
	// making the assertions deterministic regardless of the caller's umask.
	oldMask := syscall.Umask(0)
	defer syscall.Umask(oldMask)

	dir := filepath.Join(t.TempDir(), "events")

	p, err := NewEventPersistence(dir)
	if err != nil {
		t.Fatalf("NewEventPersistence() error = %v", err)
	}

	// The persistence directory must be created owner-only (0700).
	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat persistence dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0700 {
		t.Errorf("directory mode = %o, want %o", got, 0700)
	}

	// Save a batch so a file is written to disk.
	events := []PersistedEvent{
		{
			ID:        "event-1",
			Type:      "trace-create",
			Timestamp: time.Now(),
			Body:      map[string]any{"input": "secret-value"},
		},
	}
	if err := p.Save(events); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	// Every persisted JSON file must be owner read/write only (0600).
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read persistence dir: %v", err)
	}

	found := false
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		found = true

		info, err := entry.Info()
		if err != nil {
			t.Fatalf("stat persisted file %q: %v", entry.Name(), err)
		}
		if got := info.Mode().Perm(); got != 0600 {
			t.Errorf("file %q mode = %o, want %o", entry.Name(), got, 0600)
		}
	}

	if !found {
		t.Fatal("no persisted .json file was written")
	}
}
