package types

// Trace represents a trace in Langfuse.
type Trace struct {
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

	// Read-only fields returned by the API
	ProjectID    string  `json:"projectId,omitempty"`
	CreatedAt    Time    `json:"createdAt,omitempty"`
	UpdatedAt    Time    `json:"updatedAt,omitempty"`
	Latency      float64 `json:"latency,omitempty"`
	TotalCost    float64 `json:"totalCost,omitempty"`
	InputCost    float64 `json:"inputCost,omitempty"`
	OutputCost   float64 `json:"outputCost,omitempty"`
	InputTokens  int     `json:"inputTokens,omitempty"`
	OutputTokens int     `json:"outputTokens,omitempty"`
	TotalTokens  int     `json:"totalTokens,omitempty"`
}
