package langfusetest

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"

	"github.com/jdziat/langfuse-go"
)

// MockServer is a test HTTP server that records requests for verification.
type MockServer struct {
	*httptest.Server

	mu       sync.Mutex
	requests []*RecordedRequest

	// ResponseFunc allows customizing responses. If nil, returns default success.
	ResponseFunc func(r *http.Request) (int, any)
}

// RecordedRequest represents a recorded HTTP request.
type RecordedRequest struct {
	Method      string
	Path        string
	Body        []byte
	ContentType string
}

// NewMockServer creates a new mock server for testing.
func NewMockServer() *MockServer {
	ms := &MockServer{
		requests: make([]*RecordedRequest, 0),
	}

	ms.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Record the request
		var body []byte
		if r.Body != nil {
			body = make([]byte, r.ContentLength)
			r.Body.Read(body)
		}

		ms.mu.Lock()
		ms.requests = append(ms.requests, &RecordedRequest{
			Method:      r.Method,
			Path:        r.URL.Path,
			Body:        body,
			ContentType: r.Header.Get("Content-Type"),
		})
		ms.mu.Unlock()

		// Generate response
		status := http.StatusOK
		var response any

		if ms.ResponseFunc != nil {
			status, response = ms.ResponseFunc(r)
		} else {
			response = langfuse.IngestionResult{
				Successes: []langfuse.IngestionSuccess{{ID: "test", Status: 200}},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		json.NewEncoder(w).Encode(response)
	}))

	return ms
}

// Requests returns all recorded requests.
func (ms *MockServer) Requests() []*RecordedRequest {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return append([]*RecordedRequest{}, ms.requests...)
}

// RequestCount returns the number of recorded requests.
func (ms *MockServer) RequestCount() int {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	return len(ms.requests)
}

// Reset clears all recorded requests.
func (ms *MockServer) Reset() {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	ms.requests = make([]*RecordedRequest, 0)
}

// LastRequest returns the most recent request, or nil if none.
func (ms *MockServer) LastRequest() *RecordedRequest {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if len(ms.requests) == 0 {
		return nil
	}
	return ms.requests[len(ms.requests)-1]
}

// RequestAt returns the request at the given index, or nil if out of bounds.
func (ms *MockServer) RequestAt(index int) *RecordedRequest {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	if index < 0 || index >= len(ms.requests) {
		return nil
	}
	return ms.requests[index]
}

// SetResponseFunc sets the response function for customizing responses.
func (ms *MockServer) SetResponseFunc(fn func(r *http.Request) (int, any)) {
	ms.ResponseFunc = fn
}

// Response scenarios

// RespondWithSuccess configures the server to respond with a successful ingestion.
func (ms *MockServer) RespondWithSuccess() {
	ms.ResponseFunc = func(r *http.Request) (int, any) {
		return http.StatusOK, langfuse.IngestionResult{
			Successes: []langfuse.IngestionSuccess{{ID: "test", Status: 200}},
		}
	}
}

// RespondWithError configures the server to respond with an error.
func (ms *MockServer) RespondWithError(statusCode int, message string) {
	ms.ResponseFunc = func(r *http.Request) (int, any) {
		return statusCode, map[string]string{
			"error":   message,
			"message": message,
		}
	}
}

// RespondWithRateLimit configures the server to respond with a rate limit error.
func (ms *MockServer) RespondWithRateLimit(retryAfter int) {
	ms.ResponseFunc = func(r *http.Request) (int, any) {
		return http.StatusTooManyRequests, map[string]any{
			"error":       "Rate limit exceeded",
			"message":     "Rate limit exceeded",
			"retry_after": retryAfter,
		}
	}
}

// RespondWithUnauthorized configures the server to respond with an unauthorized error.
func (ms *MockServer) RespondWithUnauthorized() {
	ms.ResponseFunc = func(r *http.Request) (int, any) {
		return http.StatusUnauthorized, map[string]string{
			"error":   "Invalid credentials",
			"message": "Invalid credentials. Confirm that you've configured the correct host.",
		}
	}
}

// RespondWithServerError configures the server to respond with a 500 error.
func (ms *MockServer) RespondWithServerError() {
	ms.ResponseFunc = func(r *http.Request) (int, any) {
		return http.StatusInternalServerError, map[string]string{
			"error":   "Internal server error",
			"message": "Internal server error",
		}
	}
}

// RespondWithPartialSuccess configures the server to respond with partial success.
func (ms *MockServer) RespondWithPartialSuccess(successIDs, errorIDs []string) {
	ms.ResponseFunc = func(r *http.Request) (int, any) {
		successes := make([]langfuse.IngestionSuccess, len(successIDs))
		for i, id := range successIDs {
			successes[i] = langfuse.IngestionSuccess{ID: id, Status: 200}
		}
		errors := make([]langfuse.IngestionError, len(errorIDs))
		for i, id := range errorIDs {
			errors[i] = langfuse.IngestionError{ID: id, Status: 400, Message: "Validation failed"}
		}
		return http.StatusOK, langfuse.IngestionResult{
			Successes: successes,
			Errors:    errors,
		}
	}
}

// RespondWith configures the server to respond with a custom status and body.
func (ms *MockServer) RespondWith(statusCode int, body any) {
	ms.ResponseFunc = func(r *http.Request) (int, any) {
		return statusCode, body
	}
}

// HasRequestWithPath returns true if any request matched the given path.
func (ms *MockServer) HasRequestWithPath(path string) bool {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	for _, req := range ms.requests {
		if req.Path == path {
			return true
		}
	}
	return false
}

// RequestsWithPath returns all requests that matched the given path.
func (ms *MockServer) RequestsWithPath(path string) []*RecordedRequest {
	ms.mu.Lock()
	defer ms.mu.Unlock()
	var matched []*RecordedRequest
	for _, req := range ms.requests {
		if req.Path == path {
			matched = append(matched, req)
		}
	}
	return matched
}
