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
// Use NewTestClient for a pre-configured client with a mock server. Pass the
// test's *testing.T; the client is cleaned up automatically when the test ends:
//
//	client, server := langfusetest.NewTestClient(t)
//
//	trace, _ := client.NewTrace().Name("test").Create(ctx)
//	_ = trace
//
//	if server.RequestCount() != 1 {
//	    t.Errorf("expected 1 request, got %d", server.RequestCount())
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
//	client, _ := langfuse.New("pk", "sk", langfuse.WithStructuredLogger(logger))
//	// ... use client ...
//
//	messages := logger.GetMessages()
package langfusetest
