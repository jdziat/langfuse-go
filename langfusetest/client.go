package langfusetest

import (
	"context"

	langfuse "github.com/jdziat/langfuse-go"
)

// TestingT is an interface that matches *testing.T and *testing.B.
type TestingT interface {
	Fatalf(format string, args ...any)
	Cleanup(func())
	Helper()
}

// TestPublicKey is the default test public key.
const TestPublicKey = "pk-lf-test-key"

// TestSecretKey is the default test secret key.
const TestSecretKey = "sk-lf-test-key"

// NewTestClient creates a client configured for testing.
// It uses a mock server that doesn't make real API calls.
// The client and server are automatically cleaned up when the test ends.
func NewTestClient(t TestingT) (*langfuse.Client, *MockServer) {
	t.Helper()

	server := NewMockServer()

	client, err := langfuse.New(TestPublicKey, TestSecretKey,
		langfuse.WithBaseURL(server.URL),
		langfuse.WithBatchSize(1000),         // Don't auto-flush in tests
		langfuse.WithFlushInterval(60*1e9),   // Very long interval
		langfuse.WithShutdownTimeout(60*1e9), // Match timeout for validation
	)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	t.Cleanup(func() {
		client.Shutdown(context.Background())
		server.Close()
	})

	return client, server
}

// NewTestClientWithConfig creates a client with custom configuration for testing.
// Base options (mock server URL, large batch size, long flush interval) are applied first,
// then the provided options are applied on top.
func NewTestClientWithConfig(t TestingT, opts ...langfuse.ConfigOption) (*langfuse.Client, *MockServer) {
	t.Helper()

	server := NewMockServer()

	// Prepend base options, then apply custom options
	baseOpts := []langfuse.ConfigOption{
		langfuse.WithBaseURL(server.URL),
		langfuse.WithBatchSize(1000),
		langfuse.WithFlushInterval(60 * 1e9),
		langfuse.WithShutdownTimeout(60 * 1e9),
	}

	allOpts := append(baseOpts, opts...)

	client, err := langfuse.New(TestPublicKey, TestSecretKey, allOpts...)
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	t.Cleanup(func() {
		client.Shutdown(context.Background())
		server.Close()
	})

	return client, server
}
