package types

// Dataset represents a dataset in Langfuse.
type Dataset struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Metadata    Metadata `json:"metadata,omitempty"`

	// Read-only fields
	ProjectID string `json:"projectId,omitempty"`
	CreatedAt Time   `json:"createdAt,omitempty"`
	UpdatedAt Time   `json:"updatedAt,omitempty"`
}

// DatasetItem represents an item in a dataset.
type DatasetItem struct {
	ID                  string   `json:"id,omitempty"`
	DatasetName         string   `json:"datasetName,omitempty"`
	Input               any      `json:"input,omitempty"`
	ExpectedOutput      any      `json:"expectedOutput,omitempty"`
	Metadata            Metadata `json:"metadata,omitempty"`
	SourceTraceID       string   `json:"sourceTraceId,omitempty"`
	SourceObservationID string   `json:"sourceObservationId,omitempty"`
	Status              string   `json:"status,omitempty"`

	// Read-only fields
	DatasetID string `json:"datasetId,omitempty"`
	ProjectID string `json:"projectId,omitempty"`
	CreatedAt Time   `json:"createdAt,omitempty"`
	UpdatedAt Time   `json:"updatedAt,omitempty"`
}

// DatasetRun represents a run against a dataset.
type DatasetRun struct {
	ID          string   `json:"id,omitempty"`
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Metadata    Metadata `json:"metadata,omitempty"`
	DatasetID   string   `json:"datasetId,omitempty"`
	DatasetName string   `json:"datasetName,omitempty"`

	// Read-only fields
	ProjectID string `json:"projectId,omitempty"`
	CreatedAt Time   `json:"createdAt,omitempty"`
	UpdatedAt Time   `json:"updatedAt,omitempty"`
}

// DatasetRunItem represents an item in a dataset run.
type DatasetRunItem struct {
	ID             string `json:"id,omitempty"`
	DatasetRunID   string `json:"datasetRunId,omitempty"`
	DatasetRunName string `json:"datasetRunName,omitempty"`
	DatasetItemID  string `json:"datasetItemId,omitempty"`
	TraceID        string `json:"traceId,omitempty"`
	ObservationID  string `json:"observationId,omitempty"`

	// Read-only fields
	ProjectID string `json:"projectId,omitempty"`
	CreatedAt Time   `json:"createdAt,omitempty"`
	UpdatedAt Time   `json:"updatedAt,omitempty"`
}
