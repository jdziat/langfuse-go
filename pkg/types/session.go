package types

// Session represents a session in Langfuse.
type Session struct {
	ID        string `json:"id"`
	CreatedAt Time   `json:"createdAt,omitempty"`
	ProjectID string `json:"projectId,omitempty"`
}
