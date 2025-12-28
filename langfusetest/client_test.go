package langfusetest

import (
	"context"
	"testing"

	langfuse "github.com/jdziat/langfuse-go"
)

func TestNewTestClient(t *testing.T) {
	client, server := NewTestClient(t)

	if client == nil {
		t.Fatal("NewTestClient returned nil client")
	}
	if server == nil {
		t.Fatal("NewTestClient returned nil server")
	}

	// Create a trace to verify client works
	trace, err := client.NewTrace().Name("test-trace").Create(context.Background())
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}
	if trace == nil {
		t.Fatal("Create returned nil trace")
	}

	// Flush to send to mock server
	err = client.Flush(context.Background())
	if err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}

	// Verify request was sent to mock server
	if server.RequestCount() == 0 {
		t.Error("Expected request to be sent to mock server")
	}
}

func TestNewTestClientWithConfig(t *testing.T) {
	client, server := NewTestClientWithConfig(t,
		langfuse.WithBatchSize(5), // Override batch size
	)

	if client == nil {
		t.Fatal("NewTestClientWithConfig returned nil client")
	}
	if server == nil {
		t.Fatal("NewTestClientWithConfig returned nil server")
	}

	// Create a trace
	trace, err := client.NewTrace().Name("test-trace").Create(context.Background())
	if err != nil {
		t.Fatalf("Failed to create trace: %v", err)
	}
	if trace == nil {
		t.Fatal("Create returned nil trace")
	}

	// Flush and verify
	err = client.Flush(context.Background())
	if err != nil {
		t.Fatalf("Failed to flush: %v", err)
	}

	if server.RequestCount() == 0 {
		t.Error("Expected request to be sent to mock server")
	}
}

func TestNewTestClient_MockServerResponses(t *testing.T) {
	client, server := NewTestClient(t)

	// Test that mock server can return different responses
	t.Run("success response", func(t *testing.T) {
		server.Reset()
		server.RespondWithSuccess()

		_, err := client.NewTrace().Name("test").Create(context.Background())
		if err != nil {
			t.Fatalf("Failed to create trace: %v", err)
		}

		err = client.Flush(context.Background())
		if err != nil {
			t.Fatalf("Flush should succeed with success response: %v", err)
		}
	})

	t.Run("error response", func(t *testing.T) {
		server.Reset()
		server.RespondWithError(400, "Bad request")

		_, err := client.NewTrace().Name("test").Create(context.Background())
		if err != nil {
			t.Fatalf("Failed to create trace: %v", err)
		}

		// Flush may or may not return error depending on retry logic
		_ = client.Flush(context.Background())
		// Just verify request was made
		if server.RequestCount() == 0 {
			t.Error("Expected request to be sent")
		}
	})
}

func TestTestConstants(t *testing.T) {
	if TestPublicKey == "" {
		t.Error("TestPublicKey should not be empty")
	}
	if TestSecretKey == "" {
		t.Error("TestSecretKey should not be empty")
	}
	if len(TestPublicKey) < 8 {
		t.Error("TestPublicKey should be at least 8 characters")
	}
	if len(TestSecretKey) < 8 {
		t.Error("TestSecretKey should be at least 8 characters")
	}
}
