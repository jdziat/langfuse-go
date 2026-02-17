package client

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	pkgerrors "github.com/jdziat/langfuse-go/pkg/errors"
	pkghttp "github.com/jdziat/langfuse-go/pkg/http"
	pkgingestion "github.com/jdziat/langfuse-go/pkg/ingestion"
)

// Ensure httpClient implements pkghttp.Doer at compile time.
var _ pkghttp.Doer = (*httpClient)(nil)

const (
	// maxResponseSize limits the size of HTTP response bodies to prevent OOM.
	maxResponseSize = 10 * 1024 * 1024 // 10MB

	// maxRequestBodySize limits the size of HTTP request bodies.
	// This prevents accidentally sending extremely large batches.
	maxRequestBodySize = 10 * 1024 * 1024 // 10MB
)

// httpClient handles HTTP requests to the Langfuse API.
type httpClient struct {
	client         *http.Client
	baseURL        string
	authHeader     string
	maxRetries     int
	retryDelay     time.Duration
	retryStrategy  pkghttp.RetryStrategy
	debug          bool
	circuitBreaker *pkghttp.CircuitBreaker
	hook           HTTPHook
}

// newHTTPClient creates a new HTTP client.
func newHTTPClient(cfg *Config) *httpClient {
	auth := base64.StdEncoding.EncodeToString([]byte(cfg.PublicKey + ":" + cfg.SecretKey))

	// Use the provided retry strategy or create a default one
	retryStrategy := cfg.RetryStrategy
	if retryStrategy == nil {
		retryStrategy = &pkghttp.ExponentialBackoff{
			InitialDelay: cfg.RetryDelay,
			MaxDelay:     30 * time.Second,
			Multiplier:   2.0,
			Jitter:       true,
			MaxRetries:   cfg.MaxRetries,
		}
	}

	h := &httpClient{
		client:        cfg.HTTPClient,
		baseURL:       strings.TrimSuffix(cfg.BaseURL, "/"),
		authHeader:    "Basic " + auth,
		maxRetries:    cfg.MaxRetries,
		retryDelay:    cfg.RetryDelay,
		retryStrategy: retryStrategy,
		debug:         cfg.Debug,
		hook:          combineHooks(cfg.HTTPHooks),
	}

	// Initialize circuit breaker if configured
	if cfg.CircuitBreaker != nil {
		h.circuitBreaker = pkghttp.NewCircuitBreaker(*cfg.CircuitBreaker)
	}

	return h
}

// request represents an HTTP request to be made.
type request struct {
	method string
	path   string
	query  url.Values
	body   any
	result any
}

// do executes an HTTP request with retries and optional circuit breaker protection.
func (h *httpClient) do(ctx context.Context, req *request) error {
	// Wrap with circuit breaker if configured
	if h.circuitBreaker != nil {
		return h.circuitBreaker.Execute(func() error {
			return h.doWithRetries(ctx, req)
		})
	}
	return h.doWithRetries(ctx, req)
}

// doWithRetries executes an HTTP request with retries.
func (h *httpClient) doWithRetries(ctx context.Context, req *request) error {
	for attempt := 0; ; attempt++ {
		err := h.doOnce(ctx, req)
		if err == nil {
			return nil
		}

		// Check if we should retry using the retry strategy
		if !h.retryStrategy.ShouldRetry(attempt, err) {
			return err
		}

		// Get the delay from the retry strategy
		// Use RetryDelayWithError if available (supports Retry-After headers)
		var delay time.Duration
		if strategyWithErr, ok := h.retryStrategy.(pkghttp.RetryStrategyWithError); ok {
			delay = strategyWithErr.RetryDelayWithError(attempt, err)
		} else {
			delay = h.retryStrategy.RetryDelay(attempt)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}
}

// doOnce executes a single HTTP request.
func (h *httpClient) doOnce(ctx context.Context, req *request) error {
	// Build URL
	u := h.baseURL + req.path
	if len(req.query) > 0 {
		u += "?" + req.query.Encode()
	}

	// Build body
	var bodyReader io.Reader
	if req.body != nil {
		bodyBytes, err := json.Marshal(req.body)
		if err != nil {
			return fmt.Errorf("langfuse: failed to marshal request body: %w", err)
		}
		if len(bodyBytes) > maxRequestBodySize {
			return fmt.Errorf("langfuse: request body size %d bytes exceeds maximum %d bytes",
				len(bodyBytes), maxRequestBodySize)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, req.method, u, bodyReader)
	if err != nil {
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	// Generate request ID for tracing
	requestID := generateRequestID()

	// Set headers
	httpReq.Header.Set("Authorization", h.authHeader)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "langfuse-go/"+Version)
	httpReq.Header.Set("X-Request-ID", requestID)

	// Check if context has a request ID override
	if ctxRequestID, ok := ctx.Value(requestIDContextKey{}).(string); ok && ctxRequestID != "" {
		requestID = ctxRequestID
		httpReq.Header.Set("X-Request-ID", requestID)
	}

	// Call BeforeRequest hook
	if h.hook != nil {
		if err := h.hook.BeforeRequest(ctx, httpReq); err != nil {
			return fmt.Errorf("langfuse: hook BeforeRequest failed: %w", err)
		}
	}

	// Track request timing for hooks
	startTime := time.Now()

	// Execute request
	resp, err := h.client.Do(httpReq)

	// Calculate duration for hooks
	duration := time.Since(startTime)

	// Call AfterResponse hook (even on error)
	if h.hook != nil {
		h.hook.AfterResponse(ctx, httpReq, resp, duration, err)
	}

	if err != nil {
		return fmt.Errorf("langfuse: request failed (request_id=%s): %w", requestID, err)
	}
	defer resp.Body.Close()

	// Read response body with size limit to prevent OOM
	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize+1))
	if err != nil {
		return fmt.Errorf("langfuse: failed to read response body (request_id=%s): %w", requestID, err)
	}
	if len(respBody) > maxResponseSize {
		return fmt.Errorf("langfuse: response body exceeded maximum size of %d bytes (request_id=%s)", maxResponseSize, requestID)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		apiErr := &pkgerrors.APIError{
			StatusCode: resp.StatusCode,
			RequestID:  requestID,
		}
		if len(respBody) > 0 {
			// Attempt to parse error response body. If parsing fails,
			// we still return the APIError with status code and request ID.
			// Store the raw body in the message if JSON parsing fails.
			if err := json.Unmarshal(respBody, apiErr); err != nil {
				// Parsing failed - include raw body as message for debugging
				apiErr.Message = string(respBody)
			}
		}

		// Parse Retry-After header for rate limit responses
		if resp.StatusCode == 429 {
			apiErr.RetryAfter = parseRetryAfter(resp.Header.Get("Retry-After"))
		}

		return apiErr
	}

	// Parse response
	if req.result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, req.result); err != nil {
			return fmt.Errorf("langfuse: failed to unmarshal response (request_id=%s): %w", requestID, err)
		}
	}

	return nil
}

// requestIDContextKey is the context key for request IDs.
type requestIDContextKey struct{}

// WithRequestID returns a context with the given request ID.
// This ID will be sent to the Langfuse API and can be used for debugging.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey{}, requestID)
}

// generateRequestID generates a unique request ID.
func generateRequestID() string {
	id, err := pkgingestion.UUID()
	if err != nil {
		// Fallback to timestamp-based ID
		return fmt.Sprintf("req-%d", time.Now().UnixNano())
	}
	return id
}

// parseRetryAfter parses the Retry-After header value.
// It supports both seconds (integer) and HTTP-date formats.
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}

	// Try parsing as seconds (integer)
	if seconds, err := strconv.Atoi(value); err == nil {
		return time.Duration(seconds) * time.Second
	}

	// Try parsing as HTTP-date (RFC 7231)
	if t, err := http.ParseTime(value); err == nil {
		return time.Until(t)
	}

	return 0
}

// get performs a GET request.
func (h *httpClient) get(ctx context.Context, path string, query url.Values, result any) error {
	return h.do(ctx, &request{
		method: http.MethodGet,
		path:   path,
		query:  query,
		result: result,
	})
}

// post performs a POST request.
func (h *httpClient) post(ctx context.Context, path string, body any, result any) error {
	return h.do(ctx, &request{
		method: http.MethodPost,
		path:   path,
		body:   body,
		result: result,
	})
}

// delete performs a DELETE request.
func (h *httpClient) delete(ctx context.Context, path string, result any) error {
	return h.do(ctx, &request{
		method: http.MethodDelete,
		path:   path,
		result: result,
	})
}

// Get performs an HTTP GET request (implements http.Doer).
func (h *httpClient) Get(ctx context.Context, path string, query url.Values, result any) error {
	return h.get(ctx, path, query, result)
}

// Post performs an HTTP POST request (implements http.Doer).
func (h *httpClient) Post(ctx context.Context, path string, body, result any) error {
	return h.post(ctx, path, body, result)
}

// Delete performs an HTTP DELETE request (implements http.Doer).
func (h *httpClient) Delete(ctx context.Context, path string, result any) error {
	return h.delete(ctx, path, result)
}

// combineHooks combines multiple hooks into one.
func combineHooks(hooks []HTTPHook) HTTPHook {
	if len(hooks) == 0 {
		return nil
	}
	if len(hooks) == 1 {
		return hooks[0]
	}
	return &combinedHook{hooks: hooks}
}

// combinedHook combines multiple hooks into one.
type combinedHook struct {
	hooks []HTTPHook
}

// BeforeRequest calls BeforeRequest on all hooks.
func (c *combinedHook) BeforeRequest(ctx context.Context, req *http.Request) error {
	for _, h := range c.hooks {
		if err := h.BeforeRequest(ctx, req); err != nil {
			return err
		}
	}
	return nil
}

// AfterResponse calls AfterResponse on all hooks.
func (c *combinedHook) AfterResponse(ctx context.Context, req *http.Request, resp *http.Response, duration time.Duration, err error) {
	for _, h := range c.hooks {
		h.AfterResponse(ctx, req, resp, duration, err)
	}
}

// Re-export pagination types from pkg/http for convenience.
type (
	PaginationParams  = pkghttp.PaginationParams
	PaginatedResponse = pkghttp.PaginatedResponse
	MetaResponse      = pkghttp.MetaResponse
	FilterParams      = pkghttp.FilterParams
)

// MergeQuery merges multiple url.Values into one.
var MergeQuery = pkghttp.MergeQuery
