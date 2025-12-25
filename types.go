package langfuse

import (
	"encoding/json"
	"time"
)

// Time is a custom time type that handles JSON marshaling/unmarshaling.
type Time struct {
	time.Time
}

// MarshalJSON implements json.Marshaler.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time.Format(time.RFC3339Nano))
}

// UnmarshalJSON implements json.Unmarshaler.
func (t *Time) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		// Try parsing as a number (Unix timestamp)
		var ts float64
		if err := json.Unmarshal(data, &ts); err != nil {
			return err
		}
		t.Time = time.Unix(int64(ts), int64((ts-float64(int64(ts)))*1e9))
		return nil
	}
	if s == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339Nano, s)
	if err != nil {
		// Try other formats
		parsed, err = time.Parse(time.RFC3339, s)
		if err != nil {
			parsed, err = time.Parse("2006-01-02T15:04:05.000Z", s)
			if err != nil {
				return err
			}
		}
	}
	t.Time = parsed
	return nil
}

// Now returns the current time as a Time.
func Now() Time {
	return Time{Time: time.Now()}
}

// ObservationType represents the type of observation.
type ObservationType string

const (
	ObservationTypeSpan       ObservationType = "SPAN"
	ObservationTypeGeneration ObservationType = "GENERATION"
	ObservationTypeEvent      ObservationType = "EVENT"
)

// ObservationLevel represents the severity level of an observation.
type ObservationLevel string

const (
	ObservationLevelDebug   ObservationLevel = "DEBUG"
	ObservationLevelDefault ObservationLevel = "DEFAULT"
	ObservationLevelWarning ObservationLevel = "WARNING"
	ObservationLevelError   ObservationLevel = "ERROR"
)

// ScoreDataType represents the data type of a score.
type ScoreDataType string

const (
	ScoreDataTypeNumeric     ScoreDataType = "NUMERIC"
	ScoreDataTypeCategorical ScoreDataType = "CATEGORICAL"
	ScoreDataTypeBoolean     ScoreDataType = "BOOLEAN"
)

// ScoreSource represents the source of a score.
type ScoreSource string

const (
	ScoreSourceAPI        ScoreSource = "API"
	ScoreSourceAnnotation ScoreSource = "ANNOTATION"
	ScoreSourceEval       ScoreSource = "EVAL"
)

// Trace represents a trace in Langfuse.
type Trace struct {
	ID          string                 `json:"id"`
	Timestamp   Time                   `json:"timestamp,omitempty"`
	Name        string                 `json:"name,omitempty"`
	UserID      string                 `json:"userId,omitempty"`
	Input       interface{}            `json:"input,omitempty"`
	Output      interface{}            `json:"output,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	SessionID   string                 `json:"sessionId,omitempty"`
	Release     string                 `json:"release,omitempty"`
	Version     string                 `json:"version,omitempty"`
	Public      bool                   `json:"public,omitempty"`
	Environment string                 `json:"environment,omitempty"`

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

// Observation represents a span, generation, or event in a trace.
type Observation struct {
	ID                  string                 `json:"id"`
	TraceID             string                 `json:"traceId,omitempty"`
	Type                ObservationType        `json:"type"`
	Name                string                 `json:"name,omitempty"`
	StartTime           Time                   `json:"startTime,omitempty"`
	EndTime             Time                   `json:"endTime,omitempty"`
	CompletionStartTime Time                   `json:"completionStartTime,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	Level               ObservationLevel       `json:"level,omitempty"`
	StatusMessage       string                 `json:"statusMessage,omitempty"`
	ParentObservationID string                 `json:"parentObservationId,omitempty"`
	Version             string                 `json:"version,omitempty"`
	Input               interface{}            `json:"input,omitempty"`
	Output              interface{}            `json:"output,omitempty"`
	Environment         string                 `json:"environment,omitempty"`

	// Generation-specific fields
	Model           string                 `json:"model,omitempty"`
	ModelParameters map[string]interface{} `json:"modelParameters,omitempty"`
	Usage           *Usage                 `json:"usage,omitempty"`
	PromptName      string                 `json:"promptName,omitempty"`
	PromptVersion   int                    `json:"promptVersion,omitempty"`

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

// Score represents a score attached to a trace or observation.
type Score struct {
	ID            string                 `json:"id,omitempty"`
	TraceID       string                 `json:"traceId"`
	ObservationID string                 `json:"observationId,omitempty"`
	Name          string                 `json:"name"`
	Value         interface{}            `json:"value"`
	StringValue   string                 `json:"stringValue,omitempty"`
	DataType      ScoreDataType          `json:"dataType,omitempty"`
	Source        ScoreSource            `json:"source,omitempty"`
	Comment       string                 `json:"comment,omitempty"`
	ConfigID      string                 `json:"configId,omitempty"`
	Environment   string                 `json:"environment,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`

	// Read-only fields
	ProjectID    string `json:"projectId,omitempty"`
	Timestamp    Time   `json:"timestamp,omitempty"`
	CreatedAt    Time   `json:"createdAt,omitempty"`
	UpdatedAt    Time   `json:"updatedAt,omitempty"`
	AuthorUserID string `json:"authorUserId,omitempty"`
}

// Prompt represents a prompt in Langfuse.
type Prompt struct {
	Name     string                 `json:"name"`
	Version  int                    `json:"version,omitempty"`
	Prompt   interface{}            `json:"prompt"`
	Type     string                 `json:"type,omitempty"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Labels   []string               `json:"labels,omitempty"`
	Tags     []string               `json:"tags,omitempty"`
	IsActive bool                   `json:"isActive,omitempty"`

	// Read-only fields
	ID        string `json:"id,omitempty"`
	ProjectID string `json:"projectId,omitempty"`
	CreatedAt Time   `json:"createdAt,omitempty"`
	UpdatedAt Time   `json:"updatedAt,omitempty"`
	CreatedBy string `json:"createdBy,omitempty"`
}

// TextPrompt represents a text-based prompt.
type TextPrompt struct {
	Prompt
}

// ChatPrompt represents a chat-based prompt with messages.
type ChatPrompt struct {
	Prompt
	Messages []ChatMessage `json:"prompt"`
}

// ChatMessage represents a message in a chat prompt.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Session represents a session in Langfuse.
type Session struct {
	ID        string `json:"id"`
	CreatedAt Time   `json:"createdAt,omitempty"`
	ProjectID string `json:"projectId,omitempty"`
}

// Dataset represents a dataset in Langfuse.
type Dataset struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`

	// Read-only fields
	ProjectID string `json:"projectId,omitempty"`
	CreatedAt Time   `json:"createdAt,omitempty"`
	UpdatedAt Time   `json:"updatedAt,omitempty"`
}

// DatasetItem represents an item in a dataset.
type DatasetItem struct {
	ID                  string                 `json:"id,omitempty"`
	DatasetName         string                 `json:"datasetName,omitempty"`
	Input               interface{}            `json:"input,omitempty"`
	ExpectedOutput      interface{}            `json:"expectedOutput,omitempty"`
	Metadata            map[string]interface{} `json:"metadata,omitempty"`
	SourceTraceID       string                 `json:"sourceTraceId,omitempty"`
	SourceObservationID string                 `json:"sourceObservationId,omitempty"`
	Status              string                 `json:"status,omitempty"`

	// Read-only fields
	DatasetID string `json:"datasetId,omitempty"`
	ProjectID string `json:"projectId,omitempty"`
	CreatedAt Time   `json:"createdAt,omitempty"`
	UpdatedAt Time   `json:"updatedAt,omitempty"`
}

// DatasetRun represents a run against a dataset.
type DatasetRun struct {
	ID          string                 `json:"id,omitempty"`
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	DatasetID   string                 `json:"datasetId,omitempty"`
	DatasetName string                 `json:"datasetName,omitempty"`

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

// Model represents a model definition in Langfuse.
type Model struct {
	ID              string                 `json:"id,omitempty"`
	ModelName       string                 `json:"modelName"`
	MatchPattern    string                 `json:"matchPattern,omitempty"`
	StartDate       Time                   `json:"startDate,omitempty"`
	InputPrice      float64                `json:"inputPrice,omitempty"`
	OutputPrice     float64                `json:"outputPrice,omitempty"`
	TotalPrice      float64                `json:"totalPrice,omitempty"`
	Unit            string                 `json:"unit,omitempty"`
	Tokenizer       string                 `json:"tokenizer,omitempty"`
	TokenizerConfig map[string]interface{} `json:"tokenizerConfig,omitempty"`

	// Read-only fields
	ProjectID         string `json:"projectId,omitempty"`
	IsLangfuseManaged bool   `json:"isLangfuseManaged,omitempty"`
	CreatedAt         Time   `json:"createdAt,omitempty"`
	UpdatedAt         Time   `json:"updatedAt,omitempty"`
}

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
