package builders

import (
	"context"
)

// EventQueuer is an interface for types that can queue ingestion events.
// This interface allows builders to work with any type that can submit events,
// breaking the circular dependency between builders and the Client type.
//
// The root langfuse.Client implements this interface.
type EventQueuer interface {
	// QueueEvent queues an event for batch submission.
	// The event should be a fully constructed ingestion event.
	QueueEvent(ctx context.Context, event IngestionEvent) error
}

// IngestionEvent represents a single event in a batch.
// This mirrors pkg/client.IngestionEvent for use in builders.
type IngestionEvent struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Timestamp Time   `json:"timestamp"`
	Body      any    `json:"body"`
}

// Time represents a timestamp for events.
// This is a simple wrapper around the RFC3339 format used by the API.
type Time struct {
	// Value is the RFC3339 formatted timestamp string.
	Value string `json:"$date"`
}

// Event type constants for the ingestion API.
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
