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

	"github.com/jdziat/langfuse-go/internal/version"
	pkgclient "github.com/jdziat/langfuse-go/pkg/client"
)

// TestVersionSurfacesAgree ensures every version surface derives from the single
// source of truth in internal/version, so they can never drift:
//   - the embedded internal/version.Version (the source)
//   - the root langfuse.Version
//   - the pkg/client.Version
//   - the root VERSION file (a generated mirror kept in sync by the release tooling)
func TestVersionSurfacesAgree(t *testing.T) {
	source := version.Version
	if source == "" {
		t.Fatal("internal/version.Version is empty; the embedded source is missing")
	}

	if Version != source {
		t.Errorf("root langfuse.Version = %q, want %q (internal/version source)", Version, source)
	}

	if pkgclient.Version != source {
		t.Errorf("pkg/client.Version = %q, want %q (internal/version source)", pkgclient.Version, source)
	}

	data, err := os.ReadFile("VERSION")
	if err != nil {
		t.Fatalf("failed to read VERSION file: %v", err)
	}
	if got := strings.TrimSpace(string(data)); got != source {
		t.Errorf("root VERSION file = %q, want %q (internal/version source); the mirror has drifted", got, source)
	}
}

// TestUserAgentUsesVersion ensures outgoing requests from the root client carry
// a User-Agent built from the embedded version source rather than a stale
// hard-coded literal.
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

	want := "langfuse-go/" + version.Version
	if got != want {
		t.Errorf("User-Agent = %q, want %q", got, want)
	}
}
