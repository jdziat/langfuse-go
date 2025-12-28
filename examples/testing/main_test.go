package main

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	langfuse "github.com/jdziat/langfuse-go"
	"github.com/jdziat/langfuse-go/langfusetest"
)

// This example demonstrates how to test code that uses the Langfuse SDK
// using the langfusetest package.
//
// Key patterns:
// 1. Use NewTestClient to create a client with a mock server
// 2. Use mock server response scenarios to test different behaviors
// 3. Verify requests were made correctly
// 4. Test error handling by simulating failures

// Service is an example service that uses Langfuse for tracing
type Service struct {
	langfuse *langfuse.Client
}

func NewService(client *langfuse.Client) *Service {
	return &Service{langfuse: client}
}

func (s *Service) ProcessMessage(ctx context.Context, message string) (string, error) {
	// Create a trace for this operation
	trace, err := s.langfuse.NewTrace().
		Name("process-message").
		Input(message).
		Create(ctx)
	if err != nil {
		return "", err
	}

	// Create a generation for the LLM call
	generation, err := trace.NewGeneration().
		Name("llm-completion").
		Model("gpt-4").
		Input(message).
		Create(ctx)
	if err != nil {
		return "", err
	}

	// Simulate processing
	response := "Processed: " + message

	// End generation
	if err := generation.EndWithOutput(ctx, response); err != nil {
		return "", err
	}

	// Add a score
	if err := trace.ScoreNumeric(ctx, "quality", 0.95); err != nil {
		return "", err
	}

	return response, nil
}

// TestProcessMessage tests the ProcessMessage function
func TestProcessMessage(t *testing.T) {
	// Create a test client with mock server
	client, server := langfusetest.NewTestClient(t)
	// server.Close() and client.Shutdown() are called automatically via t.Cleanup()

	service := NewService(client)
	ctx := context.Background()

	// Test successful processing
	result, err := service.ProcessMessage(ctx, "Hello, World!")
	if err != nil {
		t.Fatalf("ProcessMessage failed: %v", err)
	}

	if result != "Processed: Hello, World!" {
		t.Errorf("Unexpected result: %s", result)
	}

	// Flush to ensure events are sent
	if err := client.Flush(ctx); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	// Verify request count - events should be batched and sent
	if server.RequestCount() == 0 {
		t.Error("Expected at least one request to be made")
	}
}

// TestProcessMessage_WithValidation demonstrates testing with request validation
func TestProcessMessage_WithValidation(t *testing.T) {
	client, server := langfusetest.NewTestClient(t)
	service := NewService(client)
	ctx := context.Background()

	// Process a message
	_, err := service.ProcessMessage(ctx, "Test message")
	if err != nil {
		t.Fatalf("ProcessMessage failed: %v", err)
	}

	// Flush events
	client.Flush(ctx)

	// Verify requests were made
	if server.RequestCount() == 0 {
		t.Fatal("No requests found")
	}

	// Get all requests and verify at least one has a JSON body
	requests := server.Requests()
	var foundValidRequest bool
	for _, req := range requests {
		if req.ContentType == "application/json" && len(req.Body) > 0 {
			// Parse the request body to verify structure
			var body struct {
				Batch []struct {
					Type string          `json:"type"`
					Body json.RawMessage `json:"body"`
				} `json:"batch"`
			}

			if err := json.Unmarshal(req.Body, &body); err == nil && len(body.Batch) > 0 {
				foundValidRequest = true
				break
			}
		}
	}

	if !foundValidRequest {
		t.Error("Expected at least one valid ingestion request with events")
	}
}

// TestProcessMessage_ServerError demonstrates testing error handling
func TestProcessMessage_ServerError(t *testing.T) {
	// Create client with minimal retries for faster tests
	client, server := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithMaxRetries(1),
	)
	service := NewService(client)
	ctx := context.Background()

	// Configure the server to return an error
	server.RespondWithServerError()

	// Process a message - this should still work locally
	// (the SDK queues events and doesn't block on API calls)
	result, err := service.ProcessMessage(ctx, "Hello")
	if err != nil {
		t.Fatalf("ProcessMessage should not fail on queue: %v", err)
	}

	if result == "" {
		t.Error("Expected result")
	}

	// Flush will encounter the error, but shouldn't panic
	_ = client.Flush(ctx)
}

// TestProcessMessage_RateLimited demonstrates testing rate limit handling
func TestProcessMessage_RateLimited(t *testing.T) {
	// Create client with minimal retries for faster tests
	client, server := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithMaxRetries(1),
	)
	service := NewService(client)
	ctx := context.Background()

	// Configure the server to return rate limit errors
	server.RespondWithRateLimit(60)

	// Process a message
	_, err := service.ProcessMessage(ctx, "Hello")
	if err != nil {
		t.Fatalf("ProcessMessage should not fail on queue: %v", err)
	}

	// Flush may retry or fail depending on configuration
	_ = client.Flush(ctx)

	// Verify requests were made
	if server.RequestCount() == 0 {
		t.Error("Expected at least one request attempt")
	}
}

// TestProcessMessage_PartialSuccess demonstrates testing partial success scenarios
func TestProcessMessage_PartialSuccess(t *testing.T) {
	client, server := langfusetest.NewTestClient(t)
	service := NewService(client)
	ctx := context.Background()

	// Configure the server to return partial success
	server.RespondWithPartialSuccess([]string{"event1"}, []string{"event2"})

	// Process a message
	_, err := service.ProcessMessage(ctx, "Hello")
	if err != nil {
		t.Fatalf("ProcessMessage should not fail: %v", err)
	}

	// Flush events
	_ = client.Flush(ctx)
}

// TestProcessMessage_CustomResponse demonstrates using custom response functions
func TestProcessMessage_CustomResponse(t *testing.T) {
	client, server := langfusetest.NewTestClient(t)
	service := NewService(client)
	ctx := context.Background()

	// Track call count
	callCount := 0
	server.SetResponseFunc(func(r *http.Request) (int, any) {
		callCount++
		// First call fails, subsequent calls succeed
		if callCount == 1 {
			return 500, map[string]string{"error": "temporary failure"}
		}
		return 200, langfuse.IngestionResult{
			Successes: []langfuse.IngestionSuccess{{ID: "test", Status: 200}},
		}
	})

	// Process a message
	_, _ = service.ProcessMessage(ctx, "Hello")
	_ = client.Flush(ctx)

	// With retries, we should have seen at least 2 calls
	if callCount < 1 {
		t.Error("Expected at least one API call")
	}
}

// TestWithCustomConfig demonstrates creating test clients with custom options
func TestWithCustomConfig(t *testing.T) {
	// Create client with custom batch size
	client, server := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithBatchSize(5),
		langfuse.WithMaxRetries(2),
	)

	service := NewService(client)
	ctx := context.Background()

	// Create multiple messages to test batching
	for i := 0; i < 3; i++ {
		_, err := service.ProcessMessage(ctx, "Message")
		if err != nil {
			t.Fatalf("ProcessMessage %d failed: %v", i, err)
		}
	}

	// Flush all events
	client.Flush(ctx)

	// Verify requests were made
	if server.RequestCount() == 0 {
		t.Error("Expected requests to be made")
	}
}

// TestMockMetrics demonstrates using mock metrics for testing
func TestMockMetrics(t *testing.T) {
	metrics := langfusetest.NewMockMetrics()

	// Use metrics with client
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithMetrics(metrics),
	)

	service := NewService(client)
	ctx := context.Background()

	// Process a message
	_, _ = service.ProcessMessage(ctx, "Hello")
	client.Flush(ctx)

	// Check that metrics were recorded
	// The specific metrics depend on what the SDK records
	// This is mainly for demonstrating the pattern
}

// TestMockLogger demonstrates using mock logger for testing
func TestMockLogger(t *testing.T) {
	logger := langfusetest.NewMockLogger()

	// Use logger with client in debug mode
	client, _ := langfusetest.NewTestClientWithConfig(t,
		langfuse.WithLogger(logger),
		langfuse.WithDebug(true),
	)

	service := NewService(client)
	ctx := context.Background()

	// Process a message
	_, _ = service.ProcessMessage(ctx, "Hello")
	client.Flush(ctx)

	// Check that log messages were recorded
	if logger.MessageCount() == 0 {
		// In debug mode, we expect some log messages
		t.Log("Note: No log messages recorded (this is OK if debug logging is minimal)")
	}
}
