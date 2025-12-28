package langfuse

import (
	"encoding/json"
	"time"
)

// JSON is an alias for any, representing any JSON value.
// Use this for input/output fields that accept arbitrary JSON data.
//
// Example:
//
//	trace.Generation().
//	    Input(langfuse.JSON("What is Go?")).
//	    Output(langfuse.JSON(map[string]any{"answer": "Go is..."})).
//	    Create()
type JSON = any

// JSONObject is an alias for map[string]any, representing a JSON object.
// Use this for metadata and structured data fields.
//
// Example:
//
//	trace.Generation().
//	    Metadata(langfuse.JSONObject{"model": "gpt-4", "temperature": 0.7}).
//	    Create()
type JSONObject = map[string]any

// Time is a custom time type that handles JSON marshaling/unmarshaling.
// When the time is zero, it marshals to JSON null.
// Note: The omitempty tag does NOT prevent zero times from being marshaled.
// If you need true omitempty behavior, use *Time (pointer) instead.
type Time struct {
	time.Time
}

// IsZero returns true if the time is the zero value.
// This method is used by encoding/json for omitempty checks in Go 1.18+.
func (t Time) IsZero() bool {
	return t.Time.IsZero()
}

// MarshalJSON implements json.Marshaler.
// Zero times are marshaled as JSON null.
func (t Time) MarshalJSON() ([]byte, error) {
	if t.Time.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(t.Time.Format(time.RFC3339Nano))
}

// TimePtr returns a pointer to a Time value.
// Use this when you need true omitempty behavior with JSON marshaling.
func TimePtr(t time.Time) *Time {
	return &Time{Time: t}
}

// TimeNow returns a pointer to the current time.
// Convenience function for TimePtr(time.Now()).
func TimeNow() *Time {
	return &Time{Time: time.Now()}
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

// String returns the string representation of the observation type.
func (o ObservationType) String() string { return string(o) }

// ObservationLevel represents the severity level of an observation.
type ObservationLevel string

const (
	ObservationLevelDebug   ObservationLevel = "DEBUG"
	ObservationLevelDefault ObservationLevel = "DEFAULT"
	ObservationLevelWarning ObservationLevel = "WARNING"
	ObservationLevelError   ObservationLevel = "ERROR"
)

// String returns the string representation of the observation level.
func (l ObservationLevel) String() string { return string(l) }

// ScoreDataType represents the data type of a score.
type ScoreDataType string

const (
	ScoreDataTypeNumeric     ScoreDataType = "NUMERIC"
	ScoreDataTypeCategorical ScoreDataType = "CATEGORICAL"
	ScoreDataTypeBoolean     ScoreDataType = "BOOLEAN"
)

// String returns the string representation of the score data type.
func (s ScoreDataType) String() string { return string(s) }

// ScoreSource represents the source of a score.
type ScoreSource string

const (
	ScoreSourceAPI        ScoreSource = "API"
	ScoreSourceAnnotation ScoreSource = "ANNOTATION"
	ScoreSourceEval       ScoreSource = "EVAL"
)

// String returns the string representation of the score source.
func (s ScoreSource) String() string { return string(s) }

// Common environment constants.
// Use these with the Environment() builder methods for consistency.
const (
	EnvProduction  = "production"
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvTest        = "test"
)

// Common prompt label constants.
// Use these with GetByLabel() for consistency.
const (
	LabelProduction  = "production"
	LabelDevelopment = "development"
	LabelStaging     = "staging"
	LabelLatest      = "latest"
)

// Common model name constants.
// These are provided for convenience and discoverability.
const (
	// OpenAI models
	ModelGPT4          = "gpt-4"
	ModelGPT4Turbo     = "gpt-4-turbo"
	ModelGPT4o         = "gpt-4o"
	ModelGPT4oMini     = "gpt-4o-mini"
	ModelGPT35Turbo    = "gpt-3.5-turbo"
	ModelO1            = "o1"
	ModelO1Mini        = "o1-mini"
	ModelO1Preview     = "o1-preview"
	ModelO3Mini        = "o3-mini"
	ModelTextEmbedding = "text-embedding-3-small"

	// Anthropic models
	ModelClaude3Opus    = "claude-3-opus"
	ModelClaude3Sonnet  = "claude-3-sonnet"
	ModelClaude3Haiku   = "claude-3-haiku"
	ModelClaude35Sonnet = "claude-3.5-sonnet"
	ModelClaude35Haiku  = "claude-3.5-haiku"
	ModelClaude4Opus    = "claude-opus-4"
	ModelClaude4Sonnet  = "claude-sonnet-4"

	// Google models
	ModelGeminiPro     = "gemini-pro"
	ModelGemini15Pro   = "gemini-1.5-pro"
	ModelGemini15Flash = "gemini-1.5-flash"
	ModelGemini20Flash = "gemini-2.0-flash"
)

// PromptType represents the type of a prompt.
type PromptType string

const (
	PromptTypeText = "text"
	PromptTypeChat = "chat"
)

// String returns the string representation of the prompt type.
func (p PromptType) String() string { return string(p) }

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

// Prompt represents a prompt in Langfuse.
type Prompt struct {
	Name     string         `json:"name"`
	Version  int            `json:"version,omitempty"`
	Prompt   any            `json:"prompt"`
	Type     string         `json:"type,omitempty"`
	Config   map[string]any `json:"config,omitempty"`
	Labels   []string       `json:"labels,omitempty"`
	Tags     []string       `json:"tags,omitempty"`
	IsActive bool           `json:"isActive,omitempty"`

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
