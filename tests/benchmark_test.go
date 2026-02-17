package langfuse_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jdziat/langfuse-go"
)

// setupBenchmarkServer creates a test server for benchmarks.
func setupBenchmarkServer(b *testing.B) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(langfuse.IngestionResult{
			Successes: []langfuse.IngestionSuccess{{ID: "test", Status: 200}},
		})
	}))
}

// setupBenchmarkClient creates a test client for benchmarks.
func setupBenchmarkClient(b *testing.B, serverURL string) *langfuse.Client {
	client, err := langfuse.New("pk-lf-test-key", "sk-lf-test-key",
		langfuse.WithBaseURL(serverURL),
		langfuse.WithBatchSize(1000),                  // Don't auto-flush during benchmarks
		langfuse.WithFlushInterval(60*1000*1000*1000), // Very long interval
	)
	if err != nil {
		b.Fatalf("Failed to create test client: %v", err)
	}
	return client
}

func BenchmarkTraceCreation(b *testing.B) {
	server := setupBenchmarkServer(b)
	defer server.Close()

	client := setupBenchmarkClient(b, server.URL)
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := client.NewTrace().
				Name("benchmark").
				UserID("user-123").
				Create(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkTraceCreationSequential(b *testing.B) {
	server := setupBenchmarkServer(b)
	defer server.Close()

	client := setupBenchmarkClient(b, server.URL)
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := client.NewTrace().
			Name("benchmark").
			UserID("user-123").
			Create(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSpanCreation(b *testing.B) {
	server := setupBenchmarkServer(b)
	defer server.Close()

	client := setupBenchmarkClient(b, server.URL)
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, err := client.NewTrace().Name("benchmark-trace").Create(ctx)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := trace.NewSpan().
				Name("benchmark-span").
				Create(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkGenerationCreation(b *testing.B) {
	server := setupBenchmarkServer(b)
	defer server.Close()

	client := setupBenchmarkClient(b, server.URL)
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	trace, err := client.NewTrace().Name("benchmark-trace").Create(ctx)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := trace.NewGeneration().
				Name("benchmark-generation").
				Model("gpt-4").
				UsageTokens(100, 50).
				Create(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func BenchmarkBatchIngestion(b *testing.B) {
	server := setupBenchmarkServer(b)
	defer server.Close()

	client := setupBenchmarkClient(b, server.URL)
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.NewTrace().Name("test").Create(ctx)
	}
	client.Flush(ctx)
}

func BenchmarkFlush(b *testing.B) {
	server := setupBenchmarkServer(b)
	defer server.Close()

	client := setupBenchmarkClient(b, server.URL)
	defer client.Shutdown(context.Background())

	ctx := context.Background()
	// Pre-populate with events
	for i := 0; i < 100; i++ {
		_, _ = client.NewTrace().Name("test").Create(ctx)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client.Flush(ctx)
		// Re-populate
		for j := 0; j < 100; j++ {
			_, _ = client.NewTrace().Name("test").Create(ctx)
		}
	}
}

// Note: BenchmarkJSONMarshaling was removed because it uses unexported type createTraceEvent

// Note: BenchmarkIDGeneration was removed because it uses unexported function generateID

func BenchmarkUUIDGeneration(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = langfuse.UUID()
	}
}

func BenchmarkGenerateID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = langfuse.GenerateID()
	}
}

func BenchmarkPromptCompile(b *testing.B) {
	prompt := &langfuse.Prompt{
		Prompt: "Hello {{name}}, welcome to {{place}}!",
	}
	variables := map[string]string{
		"name":  "World",
		"place": "Go",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := prompt.Compile(variables)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPromptCompileLarge(b *testing.B) {
	// Create a prompt with many variables
	prompt := &langfuse.Prompt{
		Prompt: "Hello {{name}}, welcome to {{place}}! Your ID is {{id}} and your email is {{email}}. " +
			"You have {{count}} items in your cart. Your address is {{address}} in {{city}}, {{country}}. " +
			"Today's date is {{date}} and the time is {{time}}. Your order number is {{order_number}}.",
	}
	variables := map[string]string{
		"name":         "John Doe",
		"place":        "Go SDK",
		"id":           "12345",
		"email":        "john@example.com",
		"count":        "5",
		"address":      "123 Main St",
		"city":         "San Francisco",
		"country":      "USA",
		"date":         "2024-01-15",
		"time":         "14:30",
		"order_number": "ORD-2024-001",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := prompt.Compile(variables)
		if err != nil {
			b.Fatal(err)
		}
	}
}
