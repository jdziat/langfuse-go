package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/jdziat/langfuse-go/internal/version"
)

// TestVersionIsSource ensures the package-level Version derives from the single
// source of truth in internal/version and is not a stale hard-coded literal.
func TestVersionIsSource(t *testing.T) {
	if Version == "" {
		t.Fatal("pkg/client.Version is empty")
	}
	if Version != version.Version {
		t.Errorf("pkg/client.Version = %q, want %q (internal/version source)", Version, version.Version)
	}
}

// TestDirectClientUserAgent ensures a client constructed directly via the
// pkg/client package - with no Config.Version injection from the root langfuse
// package - reports the real embedded version on the wire (User-Agent), proving
// the version surface does not depend on injection.
func TestDirectClientUserAgent(t *testing.T) {
	var (
		mu        sync.Mutex
		userAgent string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		userAgent = r.Header.Get("User-Agent")
		mu.Unlock()

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{}`))
	}))
	defer server.Close()

	// Note: no WithVersion / Config.Version is set, so the User-Agent must come
	// from the package-level Version (the embedded source), not injection.
	c, err := New(
		"pk-lf-test-key",
		"sk-lf-test-key",
		WithBaseURL(server.URL),
	)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer c.Shutdown(context.Background())

	if c.Config().Version != "" {
		t.Fatalf("expected no injected Config.Version, got %q", c.Config().Version)
	}

	// Drive a request through the client's HTTP layer so the User-Agent header is set.
	if err := c.HTTP().Get(context.Background(), "/health", url.Values{}, nil); err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	mu.Lock()
	got := userAgent
	mu.Unlock()

	want := "langfuse-go/" + version.Version
	if got != want {
		t.Errorf("User-Agent = %q, want %q", got, want)
	}
}
