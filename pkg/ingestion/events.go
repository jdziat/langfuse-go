package ingestion

// Event types for the ingestion API.
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
