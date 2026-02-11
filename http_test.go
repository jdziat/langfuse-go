package langfuse

import (
	"net/url"
	"testing"
	"time"
)

// Note: HTTP client unit tests have been moved to pkg/client/http_test.go
// These tests cover the public pagination and filter parameter types.

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
