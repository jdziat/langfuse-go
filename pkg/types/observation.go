package types

// Observation represents a span, generation, or event in a trace.
type Observation struct {
	ID                  string           `json:"id"`
	TraceID             string           `json:"traceId,omitempty"`
	Type                ObservationType  `json:"type"`
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

	// Read-only fields
	ProjectID            string  `json:"projectId,omitempty"`
	CreatedAt            Time    `json:"createdAt,omitempty"`
	UpdatedAt            Time    `json:"updatedAt,omitempty"`
	Latency              float64 `json:"latency,omitempty"`
	TimeToFirstToken     float64 `json:"timeToFirstToken,omitempty"`
	TotalCost            float64 `json:"totalCost,omitempty"`
	InputCost            float64 `json:"inputCost,omitempty"`
	OutputCost           float64 `json:"outputCost,omitempty"`
	CalculatedTotalCost  float64 `json:"calculatedTotalCost,omitempty"`
	CalculatedInputCost  float64 `json:"calculatedInputCost,omitempty"`
	CalculatedOutputCost float64 `json:"calculatedOutputCost,omitempty"`
}

// Usage represents token usage for a generation.
type Usage struct {
	Input      int     `json:"input,omitempty"`
	Output     int     `json:"output,omitempty"`
	Total      int     `json:"total,omitempty"`
	Unit       string  `json:"unit,omitempty"`
	InputCost  float64 `json:"inputCost,omitempty"`
	OutputCost float64 `json:"outputCost,omitempty"`
	TotalCost  float64 `json:"totalCost,omitempty"`
}
