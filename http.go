package langfuse

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
)

// httpClient handles HTTP requests to the Langfuse API.
type httpClient struct {
	client     *http.Client
	baseURL    string
	authHeader string
	maxRetries int
	retryDelay time.Duration
	debug      bool
}

// newHTTPClient creates a new HTTP client.
func newHTTPClient(cfg *Config) *httpClient {
	auth := base64.StdEncoding.EncodeToString([]byte(cfg.PublicKey + ":" + cfg.SecretKey))
	return &httpClient{
		client:     cfg.HTTPClient,
		baseURL:    strings.TrimSuffix(cfg.BaseURL, "/"),
		authHeader: "Basic " + auth,
		maxRetries: cfg.MaxRetries,
		retryDelay: cfg.RetryDelay,
		debug:      cfg.Debug,
	}
}

// request represents an HTTP request to be made.
type request struct {
	method string
	path   string
	query  url.Values
	body   interface{}
	result interface{}
}

// do executes an HTTP request with retries.
func (h *httpClient) do(ctx context.Context, req *request) error {
	var lastErr error

	for attempt := 0; attempt <= h.maxRetries; attempt++ {
		if attempt > 0 {
			delay := h.retryDelay * time.Duration(1<<uint(attempt-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := h.doOnce(ctx, req)
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if error is retryable
		if apiErr, ok := err.(*APIError); ok {
			if !apiErr.IsRetryable() {
				return err
			}
		} else {
			// Non-API errors (network errors) are retryable
			continue
		}
	}

	return lastErr
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
		bodyReader = bytes.NewReader(bodyBytes)
	}

	// Create request
	httpReq, err := http.NewRequestWithContext(ctx, req.method, u, bodyReader)
	if err != nil {
		return fmt.Errorf("langfuse: failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Authorization", h.authHeader)
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "langfuse-go/1.0.0")

	// Execute request
	resp, err := h.client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("langfuse: request failed: %w", err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("langfuse: failed to read response body: %w", err)
	}

	// Check for errors
	if resp.StatusCode >= 400 {
		apiErr := &APIError{StatusCode: resp.StatusCode}
		if len(respBody) > 0 {
			json.Unmarshal(respBody, apiErr)
		}
		return apiErr
	}

	// Parse response
	if req.result != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, req.result); err != nil {
			return fmt.Errorf("langfuse: failed to unmarshal response: %w", err)
		}
	}

	return nil
}

// get performs a GET request.
func (h *httpClient) get(ctx context.Context, path string, query url.Values, result interface{}) error {
	return h.do(ctx, &request{
		method: http.MethodGet,
		path:   path,
		query:  query,
		result: result,
	})
}

// post performs a POST request.
func (h *httpClient) post(ctx context.Context, path string, body interface{}, result interface{}) error {
	return h.do(ctx, &request{
		method: http.MethodPost,
		path:   path,
		body:   body,
		result: result,
	})
}

// put performs a PUT request.
func (h *httpClient) put(ctx context.Context, path string, body interface{}, result interface{}) error {
	return h.do(ctx, &request{
		method: http.MethodPut,
		path:   path,
		body:   body,
		result: result,
	})
}

// patch performs a PATCH request.
func (h *httpClient) patch(ctx context.Context, path string, body interface{}, result interface{}) error {
	return h.do(ctx, &request{
		method: http.MethodPatch,
		path:   path,
		body:   body,
		result: result,
	})
}

// delete performs a DELETE request.
func (h *httpClient) delete(ctx context.Context, path string, result interface{}) error {
	return h.do(ctx, &request{
		method: http.MethodDelete,
		path:   path,
		result: result,
	})
}

// Pagination helpers

// PaginationParams represents pagination parameters for list requests.
type PaginationParams struct {
	Page   int
	Limit  int
	Cursor string
}

// ToQuery converts pagination parameters to URL query values.
func (p *PaginationParams) ToQuery() url.Values {
	q := url.Values{}
	if p.Page > 0 {
		q.Set("page", strconv.Itoa(p.Page))
	}
	if p.Limit > 0 {
		q.Set("limit", strconv.Itoa(p.Limit))
	}
	if p.Cursor != "" {
		q.Set("cursor", p.Cursor)
	}
	return q
}

// PaginatedResponse represents a paginated response.
type PaginatedResponse struct {
	Meta MetaResponse `json:"meta"`
}

// MetaResponse represents pagination metadata.
type MetaResponse struct {
	Page       int    `json:"page"`
	Limit      int    `json:"limit"`
	TotalItems int    `json:"totalItems"`
	TotalPages int    `json:"totalPages"`
	NextCursor string `json:"nextCursor,omitempty"`
}

// HasMore returns true if there are more pages.
func (m *MetaResponse) HasMore() bool {
	return m.NextCursor != "" || m.Page < m.TotalPages
}

// FilterParams represents common filter parameters.
type FilterParams struct {
	Name          string
	UserID        string
	Type          string
	TraceID       string
	SessionID     string
	Level         string
	Version       string
	Environment   string
	FromStartTime time.Time
	ToStartTime   time.Time
	Tags          []string
}

// ToQuery converts filter parameters to URL query values.
func (f *FilterParams) ToQuery() url.Values {
	q := url.Values{}
	if f.Name != "" {
		q.Set("name", f.Name)
	}
	if f.UserID != "" {
		q.Set("userId", f.UserID)
	}
	if f.Type != "" {
		q.Set("type", f.Type)
	}
	if f.TraceID != "" {
		q.Set("traceId", f.TraceID)
	}
	if f.SessionID != "" {
		q.Set("sessionId", f.SessionID)
	}
	if f.Level != "" {
		q.Set("level", f.Level)
	}
	if f.Version != "" {
		q.Set("version", f.Version)
	}
	if f.Environment != "" {
		q.Set("environment", f.Environment)
	}
	if !f.FromStartTime.IsZero() {
		q.Set("fromStartTime", f.FromStartTime.Format(time.RFC3339))
	}
	if !f.ToStartTime.IsZero() {
		q.Set("toStartTime", f.ToStartTime.Format(time.RFC3339))
	}
	for _, tag := range f.Tags {
		q.Add("tags", tag)
	}
	return q
}

// mergeQuery merges multiple url.Values into one.
func mergeQuery(queries ...url.Values) url.Values {
	result := url.Values{}
	for _, q := range queries {
		for k, v := range q {
			for _, val := range v {
				result.Add(k, val)
			}
		}
	}
	return result
}
