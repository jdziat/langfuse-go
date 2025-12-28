// Package langfusetest provides testing utilities for applications using the langfuse-go SDK.
//
// This package provides mock implementations and test helpers that make it easy to
// test code that uses the Langfuse client without making real API calls.
//
// # Mock Server
//
// Use MockServer to record and inspect HTTP requests:
//
//	server := langfusetest.NewMockServer()
//	defer server.Close()
//
//	client, _ := langfuse.New("pk", "sk", langfuse.WithBaseURL(server.URL))
//	// ... use client ...
//
//	requests := server.Requests()
//	// assert on requests
//
// # Test Client
//
// Use NewTestClient for a pre-configured client with a mock server:
//
//	func TestMyFeature(t *testing.T) {
//	    client, server := langfusetest.NewTestClient(t)
//	    // client is automatically cleaned up when test ends
//
//	    trace, _ := client.NewTrace().Name("test").Create()
//	    // ...
//
//	    if server.RequestCount() != 1 {
//	        t.Error("expected 1 request")
//	    }
//	}
//
// # Mock Metrics
//
// Use MockMetrics to verify metrics are recorded correctly:
//
//	metrics := langfusetest.NewMockMetrics()
//	client, _ := langfuse.New("pk", "sk", langfuse.WithMetrics(metrics))
//	// ... use client ...
//
//	if metrics.GetCounter("events_queued") != 5 {
//	    t.Error("expected 5 events queued")
//	}
//
// # Mock Logger
//
// Use MockLogger to capture log output:
//
//	logger := langfusetest.NewMockLogger()
//	client, _ := langfuse.New("pk", "sk", langfuse.WithLogger(logger))
//	// ... use client ...
//
//	messages := logger.GetMessages()
package langfusetest
