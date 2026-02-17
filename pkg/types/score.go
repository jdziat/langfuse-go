package types

// Score represents a score attached to a trace or observation.
type Score struct {
	ID            string        `json:"id,omitempty"`
	TraceID       string        `json:"traceId"`
	ObservationID string        `json:"observationId,omitempty"`
	Name          string        `json:"name"`
	Value         any           `json:"value"`
	StringValue   string        `json:"stringValue,omitempty"`
	DataType      ScoreDataType `json:"dataType,omitempty"`
	Source        ScoreSource   `json:"source,omitempty"`
	Comment       string        `json:"comment,omitempty"`
	ConfigID      string        `json:"configId,omitempty"`
	Environment   string        `json:"environment,omitempty"`
	Metadata      Metadata      `json:"metadata,omitempty"`

	// Read-only fields
	ProjectID    string `json:"projectId,omitempty"`
	Timestamp    Time   `json:"timestamp,omitempty"`
	CreatedAt    Time   `json:"createdAt,omitempty"`
	UpdatedAt    Time   `json:"updatedAt,omitempty"`
	AuthorUserID string `json:"authorUserId,omitempty"`
}
