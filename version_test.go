package langfuse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
)

// TestVersionMatchesVERSIONFile ensures Version is sourced from the VERSION
// file at the module root, providing a single source of truth.
func TestVersionMatchesVERSIONFile(t *testing.T) {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		t.Fatalf("failed to read VERSION file: %v", err)
	}

	want := strings.TrimSpace(string(data))
	if Version != want {
		t.Errorf("Version = %q, want %q (from VERSION file)", Version, want)
	}
}

// TestUserAgentUsesVersion ensures outgoing requests carry a User-Agent built
// from the embedded VERSION value rather than a stale hard-coded constant.
func TestUserAgentUsesVersion(t *testing.T) {
	var (
		mu        sync.Mutex
		userAgent string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		userAgent = r.Header.Get("User-Agent")
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(HealthStatus{Status: "ok", Version: Version})
	}))
	defer server.Close()

	client, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer client.Shutdown(context.Background())

	if _, err := client.Health(context.Background()); err != nil {
		t.Fatalf("Health failed: %v", err)
	}

	mu.Lock()
	got := userAgent
	mu.Unlock()

	want := "langfuse-go/" + Version
	if got != want {
		t.Errorf("User-Agent = %q, want %q", got, want)
	}
}
