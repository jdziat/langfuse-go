package types

// Comment represents a comment on a trace, observation, session, or prompt.
type Comment struct {
	ID           string `json:"id,omitempty"`
	ObjectType   string `json:"objectType"`
	ObjectID     string `json:"objectId"`
	Content      string `json:"content"`
	AuthorUserID string `json:"authorUserId,omitempty"`

	// Read-only fields
	ProjectID string `json:"projectId,omitempty"`
	CreatedAt Time   `json:"createdAt,omitempty"`
	UpdatedAt Time   `json:"updatedAt,omitempty"`
}

// HealthStatus represents the health status of the Langfuse API.
type HealthStatus struct {
	Status  string `json:"status"`
	Version string `json:"version,omitempty"`
	Message string `json:"message,omitempty"`
}
