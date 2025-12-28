package langfuse

import (
	"crypto/rand"
	"fmt"
	"time"
)

// Event types for the ingestion API
const (
	eventTypeTraceCreate      = "trace-create"
	eventTypeTraceUpdate      = "trace-update"
	eventTypeSpanCreate       = "span-create"
	eventTypeSpanUpdate       = "span-update"
	eventTypeGenerationCreate = "generation-create"
	eventTypeGenerationUpdate = "generation-update"
	eventTypeEventCreate      = "event-create"
	eventTypeScoreCreate      = "score-create"
	eventTypeSDKLog           = "sdk-log"
)

// ingestionRequest represents a batch ingestion request.
type ingestionRequest struct {
	Batch    []ingestionEvent `json:"batch"`
	Metadata Metadata         `json:"metadata,omitempty"`
}

// ingestionEvent represents a single event in a batch.
type ingestionEvent struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	Timestamp Time   `json:"timestamp"`
	Body      any    `json:"body"`
}

// traceEvent represents the body of trace-create and trace-update events.
type traceEvent struct {
	ID          string   `json:"id"`
	Timestamp   Time     `json:"timestamp,omitempty"`
	Name        string   `json:"name,omitempty"`
	UserID      string   `json:"userId,omitempty"`
	Input       any      `json:"input,omitempty"`
	Output      any      `json:"output,omitempty"`
	Metadata    Metadata `json:"metadata,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	SessionID   string   `json:"sessionId,omitempty"`
	Release     string   `json:"release,omitempty"`
	Version     string   `json:"version,omitempty"`
	Public      bool     `json:"public,omitempty"`
	Environment string   `json:"environment,omitempty"`
}

// observationEvent represents the body of span/generation/event create/update events.
// This consolidated struct handles all observation types with optional generation-specific fields.
type observationEvent struct {
	// Common observation fields
	ID                  string           `json:"id"`
	TraceID             string           `json:"traceId,omitempty"`
	Name                string           `json:"name,omitempty"`
	StartTime           Time             `json:"startTime,omitempty"`
	EndTime             Time             `json:"endTime,omitempty"`
	Metadata            Metadata         `json:"metadata,omitempty"`
	Level               ObservationLevel `json:"level,omitempty"`
	StatusMessage       string           `json:"statusMessage,omitempty"`
	ParentObservationID string           `json:"parentObservationId,omitempty"`
	Version             string           `json:"version,omitempty"`
	Input               any              `json:"input,omitempty"`
	Output              any              `json:"output,omitempty"`
	Environment         string           `json:"environment,omitempty"`

	// Generation-specific fields (ignored for spans/events)
	Model               string   `json:"model,omitempty"`
	ModelParameters     Metadata `json:"modelParameters,omitempty"`
	Usage               *Usage   `json:"usage,omitempty"`
	PromptName          string   `json:"promptName,omitempty"`
	PromptVersion       int      `json:"promptVersion,omitempty"`
	CompletionStartTime Time     `json:"completionStartTime,omitempty"`
}

// scoreEvent represents the body of a score-create event.
type scoreEvent struct {
	ID            string        `json:"id,omitempty"`
	TraceID       string        `json:"traceId"`
	ObservationID string        `json:"observationId,omitempty"`
	Name          string        `json:"name"`
	Value         any           `json:"value"`
	StringValue   string        `json:"stringValue,omitempty"`
	DataType      ScoreDataType `json:"dataType,omitempty"`
	Comment       string        `json:"comment,omitempty"`
	ConfigID      string        `json:"configId,omitempty"`
	Environment   string        `json:"environment,omitempty"`
	Metadata      Metadata      `json:"metadata,omitempty"`
}

// Type aliases consolidate the 7 legacy event types into 3 unified types.
// This reduces code duplication while maintaining API compatibility.
type (
	createTraceEvent      = traceEvent
	updateTraceEvent      = traceEvent
	createSpanEvent       = observationEvent
	updateSpanEvent       = observationEvent
	createGenerationEvent = observationEvent
	updateGenerationEvent = observationEvent
	createEventEvent      = observationEvent
	createScoreEvent      = scoreEvent
)

// UUID generates a random UUID v4.
func UUID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("langfuse: failed to generate UUID: %w", err)
	}

	// Set version (4) and variant bits
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
}

// generateID generates a random UUID-like ID.
func generateID() string {
	id, err := UUID()
	if err != nil {
		// Fallback to timestamp-based ID if crypto fails
		return fmt.Sprintf("%d-%x", time.Now().UnixNano(), time.Now().Unix())
	}
	return id
}

// IsValidUUID checks if a string is a valid UUID format.
// It accepts both standard UUID format (xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx)
// and compact format without hyphens (32 hex characters).
func IsValidUUID(s string) bool {
	// Standard UUID format: 8-4-4-4-12 = 36 characters
	if len(s) == 36 {
		return isValidStandardUUID(s)
	}
	// Compact format: 32 hex characters without hyphens
	if len(s) == 32 {
		return isHexString(s)
	}
	return false
}

// isValidStandardUUID checks if a string is a valid standard UUID format.
func isValidStandardUUID(s string) bool {
	// Check hyphen positions: 8, 13, 18, 23
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	// Check hex segments
	return isHexString(s[0:8]) &&
		isHexString(s[9:13]) &&
		isHexString(s[14:18]) &&
		isHexString(s[19:23]) &&
		isHexString(s[24:36])
}

// isHexString checks if a string contains only hexadecimal characters.
func isHexString(s string) bool {
	for _, c := range s {
		isDigit := c >= '0' && c <= '9'
		isLowerHex := c >= 'a' && c <= 'f'
		isUpperHex := c >= 'A' && c <= 'F'
		if !isDigit && !isLowerHex && !isUpperHex {
			return false
		}
	}
	return true
}
