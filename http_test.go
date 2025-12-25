package langfuse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestHTTPClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		// Check authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			t.Error("Authorization header missing")
		}

		// Check content type
		if r.Header.Get("Accept") != "application/json" {
			t.Error("Accept header should be application/json")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	cfg := &Config{
		PublicKey:  "pk-test",
		SecretKey:  "sk-test",
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		MaxRetries: 0,
	}

	client := newHTTPClient(cfg)

	var result map[string]string
	err := client.get(context.Background(), "/test", nil, &result)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("Expected status ok, got %s", result["status"])
	}
}

func TestHTTPClientPost(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Error("Content-Type header should be application/json")
		}

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)

		if body["name"] != "test" {
			t.Errorf("Expected name test, got %s", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"id": "123"})
	}))
	defer server.Close()

	cfg := &Config{
		PublicKey:  "pk-test",
		SecretKey:  "sk-test",
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		MaxRetries: 0,
	}

	client := newHTTPClient(cfg)

	var result map[string]string
	err := client.post(context.Background(), "/test", map[string]string{"name": "test"}, &result)
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}

	if result["id"] != "123" {
		t.Errorf("Expected id 123, got %s", result["id"])
	}
}

func TestHTTPClientDelete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}

		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	cfg := &Config{
		PublicKey:  "pk-test",
		SecretKey:  "sk-test",
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		MaxRetries: 0,
	}

	client := newHTTPClient(cfg)

	err := client.delete(context.Background(), "/test/123", nil)
	if err != nil {
		t.Fatalf("delete failed: %v", err)
	}
}

func TestHTTPClientAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"statusCode": 400,
			"message":    "Invalid request",
		})
	}))
	defer server.Close()

	cfg := &Config{
		PublicKey:  "pk-test",
		SecretKey:  "sk-test",
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		MaxRetries: 0,
	}

	client := newHTTPClient(cfg)

	var result map[string]string
	err := client.get(context.Background(), "/test", nil, &result)
	if err == nil {
		t.Fatal("Expected error, got nil")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("Expected APIError, got %T", err)
	}

	if apiErr.StatusCode != 400 {
		t.Errorf("Expected status 400, got %d", apiErr.StatusCode)
	}
}

func TestHTTPClientRetry(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	cfg := &Config{
		PublicKey:  "pk-test",
		SecretKey:  "sk-test",
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		MaxRetries: 3,
		RetryDelay: 10 * time.Millisecond,
	}

	client := newHTTPClient(cfg)

	var result map[string]string
	err := client.get(context.Background(), "/test", nil, &result)
	if err != nil {
		t.Fatalf("get failed after retries: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestHTTPClientContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &Config{
		PublicKey:  "pk-test",
		SecretKey:  "sk-test",
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		MaxRetries: 0,
	}

	client := newHTTPClient(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := client.get(ctx, "/test", nil, nil)
	if err == nil {
		t.Error("Expected error due to cancelled context")
	}
}

func TestHTTPClientQueryParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("page") != "1" {
			t.Errorf("Expected page=1, got %s", query.Get("page"))
		}
		if query.Get("limit") != "10" {
			t.Errorf("Expected limit=10, got %s", query.Get("limit"))
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := &Config{
		PublicKey:  "pk-test",
		SecretKey:  "sk-test",
		BaseURL:    server.URL,
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
		MaxRetries: 0,
	}

	client := newHTTPClient(cfg)

	query := url.Values{}
	query.Set("page", "1")
	query.Set("limit", "10")

	err := client.get(context.Background(), "/test", query, nil)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
}

func TestPaginationParamsToQuery(t *testing.T) {
	tests := []struct {
		name     string
		params   PaginationParams
		expected map[string]string
	}{
		{
			name:     "empty params",
			params:   PaginationParams{},
			expected: map[string]string{},
		},
		{
			name: "page and limit",
			params: PaginationParams{
				Page:  1,
				Limit: 10,
			},
			expected: map[string]string{
				"page":  "1",
				"limit": "10",
			},
		},
		{
			name: "cursor only",
			params: PaginationParams{
				Cursor: "abc123",
			},
			expected: map[string]string{
				"cursor": "abc123",
			},
		},
		{
			name: "all params",
			params: PaginationParams{
				Page:   2,
				Limit:  20,
				Cursor: "xyz",
			},
			expected: map[string]string{
				"page":   "2",
				"limit":  "20",
				"cursor": "xyz",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := tt.params.ToQuery()
			for key, expected := range tt.expected {
				if got := query.Get(key); got != expected {
					t.Errorf("Query[%s] = %v, want %v", key, got, expected)
				}
			}
		})
	}
}

func TestFilterParamsToQuery(t *testing.T) {
	now := time.Now()
	params := FilterParams{
		Name:          "test",
		UserID:        "user-123",
		Type:          "GENERATION",
		TraceID:       "trace-456",
		SessionID:     "session-789",
		Level:         "ERROR",
		Version:       "1.0",
		Environment:   "production",
		FromStartTime: now,
		ToStartTime:   now.Add(time.Hour),
		Tags:          []string{"tag1", "tag2"},
	}

	query := params.ToQuery()

	if query.Get("name") != "test" {
		t.Errorf("name = %v, want test", query.Get("name"))
	}
	if query.Get("userId") != "user-123" {
		t.Errorf("userId = %v, want user-123", query.Get("userId"))
	}
	if query.Get("type") != "GENERATION" {
		t.Errorf("type = %v, want GENERATION", query.Get("type"))
	}
	if query.Get("traceId") != "trace-456" {
		t.Errorf("traceId = %v, want trace-456", query.Get("traceId"))
	}
	if query.Get("sessionId") != "session-789" {
		t.Errorf("sessionId = %v, want session-789", query.Get("sessionId"))
	}
	if query.Get("level") != "ERROR" {
		t.Errorf("level = %v, want ERROR", query.Get("level"))
	}
	if query.Get("version") != "1.0" {
		t.Errorf("version = %v, want 1.0", query.Get("version"))
	}
	if query.Get("environment") != "production" {
		t.Errorf("environment = %v, want production", query.Get("environment"))
	}

	tags := query["tags"]
	if len(tags) != 2 {
		t.Errorf("tags length = %v, want 2", len(tags))
	}
}

func TestMergeQuery(t *testing.T) {
	q1 := url.Values{}
	q1.Set("a", "1")
	q1.Set("b", "2")

	q2 := url.Values{}
	q2.Set("c", "3")
	q2.Add("d", "4")
	q2.Add("d", "5")

	result := mergeQuery(q1, q2)

	if result.Get("a") != "1" {
		t.Errorf("a = %v, want 1", result.Get("a"))
	}
	if result.Get("b") != "2" {
		t.Errorf("b = %v, want 2", result.Get("b"))
	}
	if result.Get("c") != "3" {
		t.Errorf("c = %v, want 3", result.Get("c"))
	}

	dValues := result["d"]
	if len(dValues) != 2 {
		t.Errorf("d values length = %v, want 2", len(dValues))
	}
}

func TestMetaResponseHasMore(t *testing.T) {
	tests := []struct {
		name     string
		meta     MetaResponse
		expected bool
	}{
		{
			name: "has next cursor",
			meta: MetaResponse{
				NextCursor: "abc",
			},
			expected: true,
		},
		{
			name: "has more pages",
			meta: MetaResponse{
				Page:       1,
				TotalPages: 3,
			},
			expected: true,
		},
		{
			name: "last page",
			meta: MetaResponse{
				Page:       3,
				TotalPages: 3,
			},
			expected: false,
		},
		{
			name:     "empty meta",
			meta:     MetaResponse{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.meta.HasMore(); got != tt.expected {
				t.Errorf("HasMore() = %v, want %v", got, tt.expected)
			}
		})
	}
}
