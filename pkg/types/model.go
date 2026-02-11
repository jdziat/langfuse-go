package types

// Model represents a model definition in Langfuse.
type Model struct {
	ID              string         `json:"id,omitempty"`
	ModelName       string         `json:"modelName"`
	MatchPattern    string         `json:"matchPattern,omitempty"`
	StartDate       Time           `json:"startDate,omitempty"`
	InputPrice      float64        `json:"inputPrice,omitempty"`
	OutputPrice     float64        `json:"outputPrice,omitempty"`
	TotalPrice      float64        `json:"totalPrice,omitempty"`
	Unit            string         `json:"unit,omitempty"`
	Tokenizer       string         `json:"tokenizer,omitempty"`
	TokenizerConfig map[string]any `json:"tokenizerConfig,omitempty"`

	// Read-only fields
	ProjectID         string `json:"projectId,omitempty"`
	IsLangfuseManaged bool   `json:"isLangfuseManaged,omitempty"`
	CreatedAt         Time   `json:"createdAt,omitempty"`
	UpdatedAt         Time   `json:"updatedAt,omitempty"`
}
