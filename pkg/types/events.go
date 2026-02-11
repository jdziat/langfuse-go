package types

// TraceEventBody represents the body of a trace-create or trace-update event.
// This is sent to the Langfuse ingestion API.
type TraceEventBody struct {
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

// ObservationEventBody represents the body of span/generation/event create/update events.
// This is sent to the Langfuse ingestion API.
type ObservationEventBody struct {
	// Common observation fields
	ID                  string           `json:"id"`
	TraceID             string           `json:"traceId,omitempty"`
	Type                ObservationType  `json:"type,omitempty"`
	Name                string           `json:"name,omitempty"`
	StartTime           Time             `json:"startTime,omitempty"`
	EndTime             Time             `json:"endTime,omitempty"`
	CompletionStartTime Time             `json:"completionStartTime,omitempty"`
	Metadata            Metadata         `json:"metadata,omitempty"`
	Level               ObservationLevel `json:"level,omitempty"`
	StatusMessage       string           `json:"statusMessage,omitempty"`
	ParentObservationID string           `json:"parentObservationId,omitempty"`
	Version             string           `json:"version,omitempty"`
	Input               any              `json:"input,omitempty"`
	Output              any              `json:"output,omitempty"`
	Environment         string           `json:"environment,omitempty"`

	// Generation-specific fields
	Model           string         `json:"model,omitempty"`
	ModelParameters map[string]any `json:"modelParameters,omitempty"`
	Usage           *Usage         `json:"usage,omitempty"`
	PromptName      string         `json:"promptName,omitempty"`
	PromptVersion   int            `json:"promptVersion,omitempty"`
}

// ScoreEventBody represents the body of a score-create event.
// This is sent to the Langfuse ingestion API.
type ScoreEventBody struct {
	ID            string        `json:"id"`
	TraceID       string        `json:"traceId"`
	Name          string        `json:"name"`
	Value         any           `json:"value,omitempty"`
	ObservationID string        `json:"observationId,omitempty"`
	Comment       string        `json:"comment,omitempty"`
	DataType      ScoreDataType `json:"dataType,omitempty"`
	ConfigID      string        `json:"configId,omitempty"`
	Source        ScoreSource   `json:"source,omitempty"`
	Metadata      Metadata      `json:"metadata,omitempty"`
	QueueID       string        `json:"queueId,omitempty"`
	Environment   string        `json:"environment,omitempty"`
}

// SDKLogEventBody represents the body of an sdk-log event.
// This is used for SDK diagnostics and monitoring.
type SDKLogEventBody struct {
	Log any `json:"log"`
}
