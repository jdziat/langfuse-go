package client

import "time"

// Event types for the ingestion API
const (
	EventTypeTraceCreate      = "trace-create"
	EventTypeTraceUpdate      = "trace-update"
	EventTypeSpanCreate       = "span-create"
	EventTypeSpanUpdate       = "span-update"
	EventTypeGenerationCreate = "generation-create"
	EventTypeGenerationUpdate = "generation-update"
	EventTypeEventCreate      = "event-create"
	EventTypeScoreCreate      = "score-create"
	EventTypeSDKLog           = "sdk-log"
)

// IngestionEvent represents a single event in a batch.
type IngestionEvent struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Timestamp Time   `json:"timestamp"`
	Body      any    `json:"body"`
}

// Time is a time.Time that marshals to RFC3339 format.
type Time struct {
	time.Time
}

// Now returns the current time as a Time.
func Now() Time {
	return Time{time.Now()}
}

// IngestionRequest represents a batch ingestion request.
type IngestionRequest struct {
	Batch    []IngestionEvent `json:"batch"`
	Metadata Metadata         `json:"metadata,omitempty"`
}

// Metadata is a map of string to any for arbitrary metadata.
type Metadata = map[string]any

// IngestionResult represents the result of a batch ingestion.
type IngestionResult struct {
	Successes []IngestionSuccess `json:"successes"`
	Errors    []IngestionError   `json:"errors"`
}

// IngestionSuccess represents a successful ingestion event.
type IngestionSuccess struct {
	ID     string `json:"id"`
	Status int    `json:"status"`
}

// IngestionError represents an error during ingestion.
type IngestionError struct {
	ID      string `json:"id"`
	Status  int    `json:"status"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

// HasErrors returns true if there were any ingestion errors.
func (r *IngestionResult) HasErrors() bool {
	return len(r.Errors) > 0
}
