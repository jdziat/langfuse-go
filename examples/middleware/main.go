package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	langfuse "github.com/jdziat/langfuse-go"
)

// This example demonstrates:
// 1. Using HTTP hooks to customize Langfuse API calls
// 2. Creating middleware for HTTP servers that automatically traces requests
// 3. Context propagation patterns
// 4. Using circuit breaker and retry patterns

var client *langfuse.Client

func main() {
	ctx := context.Background()

	// Create a logger for the hooks
	logger := log.New(os.Stdout, "[langfuse] ", log.LstdFlags)

	// Create client with HTTP hooks and resilience features
	var err error
	client, err = langfuse.New(
		os.Getenv("LANGFUSE_PUBLIC_KEY"),
		os.Getenv("LANGFUSE_SECRET_KEY"),
		langfuse.WithRegion(langfuse.RegionUS),
		langfuse.WithDebug(true),

		// HTTP Hooks - customize how requests are made to Langfuse
		langfuse.WithHTTPHooks(
			// Add custom headers to all requests
			langfuse.HeaderHook(map[string]string{
				"X-Application": "langfuse-go-example",
				"X-Environment": os.Getenv("ENVIRONMENT"),
			}),

			// Log all Langfuse API calls
			langfuse.LoggingHook(logger),

			// Propagate trace context to Langfuse API
			langfuse.TracingHook(),
		),

		// Enable circuit breaker for resilience
		langfuse.WithDefaultCircuitBreaker(),

		// Custom retry strategy
		langfuse.WithMaxRetries(5),
		langfuse.WithRetryDelay(500*time.Millisecond),

		// Batch callback for monitoring
		langfuse.WithOnBatchFlushed(func(result langfuse.BatchResult) {
			if result.Success {
				logger.Printf("Batch sent: %d events in %v", result.EventCount, result.Duration)
			} else {
				logger.Printf("Batch failed: %v", result.Error)
			}
		}),
	)
	if err != nil {
		log.Fatalf("Failed to create Langfuse client: %v", err)
	}
	defer client.Shutdown(ctx)

	// Set up HTTP server with tracing middleware
	mux := http.NewServeMux()
	mux.HandleFunc("/api/chat", handleChat)
	mux.HandleFunc("/api/complete", handleComplete)
	mux.HandleFunc("/health", handleHealth)

	// Wrap with tracing middleware
	handler := TracingMiddleware(mux)

	fmt.Println("Server starting on :8080")
	fmt.Println("Try: curl http://localhost:8080/api/chat -d '{\"message\": \"Hello\"}'")

	if err := http.ListenAndServe(":8080", handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// TracingMiddleware creates traces for all incoming HTTP requests
func TracingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		startTime := time.Now()

		// Create a trace for this request
		trace, err := client.NewTrace().
			Name(fmt.Sprintf("%s %s", r.Method, r.URL.Path)).
			Input(map[string]any{
				"method":  r.Method,
				"path":    r.URL.Path,
				"query":   r.URL.RawQuery,
				"headers": sanitizeHeaders(r.Header),
			}).
			Metadata(map[string]any{
				"remote_addr": r.RemoteAddr,
				"user_agent":  r.UserAgent(),
			}).
			Create(ctx)

		if err != nil {
			log.Printf("Failed to create trace: %v", err)
			next.ServeHTTP(w, r)
			return
		}

		// Store trace in context for downstream handlers
		ctx = langfuse.ContextWithTrace(ctx, trace)
		r = r.WithContext(ctx)

		// Wrap ResponseWriter to capture status code
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}

		// Call the next handler
		next.ServeHTTP(rw, r)

		// Calculate duration
		duration := time.Since(startTime)

		// Update trace with response information
		trace.Update().
			Output(map[string]any{
				"status":      rw.status,
				"duration_ms": duration.Milliseconds(),
			}).
			Apply(ctx)

		// Add a score based on response status
		scoreValue := 1.0
		if rw.status >= 400 {
			scoreValue = 0.0
		}
		trace.ScoreNumeric(ctx, "success", scoreValue)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

// handleChat demonstrates a chat completion endpoint with nested observations
func handleChat(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get trace from context
	trace, ok := langfuse.TraceFromContext(ctx)
	if !ok {
		http.Error(w, "No trace context", http.StatusInternalServerError)
		return
	}

	// Parse request
	var req struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	// Create a span for preprocessing
	preprocessSpan, err := trace.NewSpan().
		Name("preprocess-input").
		Input(req.Message).
		Create(ctx)
	if err != nil {
		log.Printf("Failed to create span: %v", err)
	}

	// Simulate preprocessing
	time.Sleep(10 * time.Millisecond)
	processedMessage := req.Message // In reality, you'd process this
	if preprocessSpan != nil {
		preprocessSpan.EndWithOutput(ctx, processedMessage)
	}

	// Create a generation for the LLM call
	generation, err := trace.NewGeneration().
		Name("gpt-4-response").
		Model("gpt-4").
		ModelParameters(map[string]any{
			"temperature": 0.7,
			"max_tokens":  150,
		}).
		Input([]map[string]string{
			{"role": "user", "content": processedMessage},
		}).
		Create(ctx)
	if err != nil {
		log.Printf("Failed to create generation: %v", err)
	}

	// Simulate LLM call
	time.Sleep(50 * time.Millisecond)
	response := "Hello! I'm an AI assistant. How can I help you today?"

	// End generation with usage
	if generation != nil {
		generation.EndWithUsage(ctx, response, 10, 20)
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"response": response,
	})
}

// handleComplete demonstrates a completion endpoint
func handleComplete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	trace, ok := langfuse.TraceFromContext(ctx)
	if !ok {
		http.Error(w, "No trace context", http.StatusInternalServerError)
		return
	}

	// Create a generation
	generation, err := trace.NewGeneration().
		Name("completion").
		Model("gpt-3.5-turbo").
		Input("Complete this: The quick brown fox").
		Create(ctx)
	if err != nil {
		http.Error(w, "Failed to create generation", http.StatusInternalServerError)
		return
	}

	// Simulate completion
	time.Sleep(30 * time.Millisecond)
	completion := "jumps over the lazy dog."

	generation.EndWithUsage(ctx, completion, 6, 6)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"completion": completion,
	})
}

// handleHealth demonstrates a health endpoint (not traced)
func handleHealth(w http.ResponseWriter, r *http.Request) {
	// Check Langfuse health
	ctx := r.Context()
	health, err := client.Health(ctx)

	response := map[string]any{
		"service":  "ok",
		"langfuse": health,
	}

	if err != nil {
		response["langfuse_error"] = err.Error()
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sanitizeHeaders removes sensitive headers before logging
func sanitizeHeaders(h http.Header) map[string]string {
	result := make(map[string]string)
	sensitiveHeaders := map[string]bool{
		"Authorization": true,
		"Cookie":        true,
		"X-Api-Key":     true,
	}

	for k, v := range h {
		if sensitiveHeaders[k] {
			result[k] = "[REDACTED]"
		} else if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}
