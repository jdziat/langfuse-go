package http

import (
	"net/url"
	"strconv"
	"time"
)

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

// MergeQuery merges multiple url.Values into one.
func MergeQuery(queries ...url.Values) url.Values {
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
