package langfuse

import (
	"context"
	"net/url"
	"strconv"

	"github.com/jdziat/langfuse-go/pkg/api/datasets"
	"github.com/jdziat/langfuse-go/pkg/api/models"
	"github.com/jdziat/langfuse-go/pkg/api/observations"
	"github.com/jdziat/langfuse-go/pkg/api/prompts"
	"github.com/jdziat/langfuse-go/pkg/api/scores"
	"github.com/jdziat/langfuse-go/pkg/api/sessions"
	"github.com/jdziat/langfuse-go/pkg/api/traces"
)

// ============================================================================
// Traces Client
// ============================================================================

// TracesClient handles trace-related API operations.
type TracesClient struct {
	impl *traces.Client
}

// newTracesClient creates a new TracesClient.
func newTracesClient(client *Client) *TracesClient {
	return &TracesClient{
		impl: traces.New(client.HTTP()),
	}
}

// TracesListParams represents parameters for listing traces.
type TracesListParams struct {
	PaginationParams
	FilterParams
}

// TracesListResponse represents the response from listing traces.
type TracesListResponse struct {
	Data []Trace      `json:"data"`
	Meta MetaResponse `json:"meta"`
}

// List retrieves a list of traces.
func (c *TracesClient) List(ctx context.Context, params *TracesListParams) (*TracesListResponse, error) {
	var query = make(map[string][]string)
	if params != nil {
		query = mergeQuery(params.PaginationParams.ToQuery(), params.FilterParams.ToQuery())
	}

	var result TracesListResponse
	if err := c.impl.List(ctx, query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single trace by ID.
func (c *TracesClient) Get(ctx context.Context, traceID string) (*Trace, error) {
	var result Trace
	if err := c.impl.Get(ctx, traceID, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a trace by ID.
func (c *TracesClient) Delete(ctx context.Context, traceID string) error {
	return c.impl.Delete(ctx, traceID)
}

// ============================================================================
// Observations Client
// ============================================================================

// ObservationsClient handles observation-related API operations.
type ObservationsClient struct {
	impl *observations.Client
}

// newObservationsClient creates a new ObservationsClient.
func newObservationsClient(client *Client) *ObservationsClient {
	return &ObservationsClient{
		impl: observations.New(client.HTTP()),
	}
}

// ObservationsListParams represents parameters for listing observations.
type ObservationsListParams struct {
	PaginationParams
	FilterParams
	ParentObservationID string
}

// ObservationsListResponse represents the response from listing observations.
type ObservationsListResponse struct {
	Data []Observation `json:"data"`
	Meta MetaResponse  `json:"meta"`
}

// List retrieves a list of observations.
func (c *ObservationsClient) List(ctx context.Context, params *ObservationsListParams) (*ObservationsListResponse, error) {
	query := url.Values{}
	if params != nil {
		query = mergeQuery(params.PaginationParams.ToQuery(), params.FilterParams.ToQuery())
		if params.ParentObservationID != "" {
			query.Set("parentObservationId", params.ParentObservationID)
		}
	}

	var result ObservationsListResponse
	if err := c.impl.List(ctx, query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single observation by ID.
func (c *ObservationsClient) Get(ctx context.Context, observationID string) (*Observation, error) {
	var result Observation
	if err := c.impl.Get(ctx, observationID, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListByTrace retrieves all observations for a specific trace.
func (c *ObservationsClient) ListByTrace(ctx context.Context, traceID string, params *PaginationParams) (*ObservationsListResponse, error) {
	query := url.Values{}
	query.Set("traceId", traceID)
	if params != nil {
		query = mergeQuery(query, params.ToQuery())
	}

	var result ObservationsListResponse
	if err := c.impl.List(ctx, query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ListSpans retrieves all spans.
func (c *ObservationsClient) ListSpans(ctx context.Context, params *ObservationsListParams) (*ObservationsListResponse, error) {
	if params == nil {
		params = &ObservationsListParams{}
	}
	params.Type = string(ObservationTypeSpan)
	return c.List(ctx, params)
}

// ListGenerations retrieves all generations.
func (c *ObservationsClient) ListGenerations(ctx context.Context, params *ObservationsListParams) (*ObservationsListResponse, error) {
	if params == nil {
		params = &ObservationsListParams{}
	}
	params.Type = string(ObservationTypeGeneration)
	return c.List(ctx, params)
}

// ListEvents retrieves all events.
func (c *ObservationsClient) ListEvents(ctx context.Context, params *ObservationsListParams) (*ObservationsListResponse, error) {
	if params == nil {
		params = &ObservationsListParams{}
	}
	params.Type = string(ObservationTypeEvent)
	return c.List(ctx, params)
}

// ============================================================================
// Scores Client
// ============================================================================

// ScoresClient handles score-related API operations.
type ScoresClient struct {
	impl *scores.Client
}

// newScoresClient creates a new ScoresClient.
func newScoresClient(client *Client) *ScoresClient {
	return &ScoresClient{
		impl: scores.New(client.HTTP()),
	}
}

// ScoresListParams represents parameters for listing scores.
type ScoresListParams struct {
	PaginationParams
	Name          string
	UserID        string
	TraceID       string
	ObservationID string
	ConfigID      string
	DataType      ScoreDataType
	Source        ScoreSource
	Environment   string
}

// ScoresListResponse represents the response from listing scores.
type ScoresListResponse struct {
	Data []Score      `json:"data"`
	Meta MetaResponse `json:"meta"`
}

// List retrieves a list of scores.
func (c *ScoresClient) List(ctx context.Context, params *ScoresListParams) (*ScoresListResponse, error) {
	query := url.Values{}
	if params != nil {
		query = params.PaginationParams.ToQuery()
		if params.Name != "" {
			query.Set("name", params.Name)
		}
		if params.UserID != "" {
			query.Set("userId", params.UserID)
		}
		if params.TraceID != "" {
			query.Set("traceId", params.TraceID)
		}
		if params.ObservationID != "" {
			query.Set("observationId", params.ObservationID)
		}
		if params.ConfigID != "" {
			query.Set("configId", params.ConfigID)
		}
		if params.DataType != "" {
			query.Set("dataType", string(params.DataType))
		}
		if params.Source != "" {
			query.Set("source", string(params.Source))
		}
		if params.Environment != "" {
			query.Set("environment", params.Environment)
		}
	}

	var result ScoresListResponse
	if err := c.impl.List(ctx, query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single score by ID.
func (c *ScoresClient) Get(ctx context.Context, scoreID string) (*Score, error) {
	var result Score
	if err := c.impl.Get(ctx, scoreID, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateScoreRequest represents a request to create a score.
type CreateScoreRequest struct {
	TraceID       string        `json:"traceId"`
	ObservationID string        `json:"observationId,omitempty"`
	Name          string        `json:"name"`
	Value         any           `json:"value"`
	StringValue   string        `json:"stringValue,omitempty"`
	DataType      ScoreDataType `json:"dataType,omitempty"`
	Comment       string        `json:"comment,omitempty"`
	ConfigID      string        `json:"configId,omitempty"`
	Environment   string        `json:"environment,omitempty"`
	Metadata      Metadata      `json:"metadata,omitempty"`
}

// Create creates a new score directly via the API (not batched).
func (c *ScoresClient) Create(ctx context.Context, req *CreateScoreRequest) (*Score, error) {
	if req == nil {
		return nil, ErrNilRequest
	}
	if req.TraceID == "" {
		return nil, NewValidationError("traceId", "trace ID is required")
	}
	if req.Name == "" {
		return nil, NewValidationError("name", "score name is required")
	}

	var result Score
	if err := c.impl.Create(ctx, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a score by ID.
func (c *ScoresClient) Delete(ctx context.Context, scoreID string) error {
	return c.impl.Delete(ctx, scoreID)
}

// ListByTrace retrieves all scores for a specific trace.
func (c *ScoresClient) ListByTrace(ctx context.Context, traceID string, params *PaginationParams) (*ScoresListResponse, error) {
	p := &ScoresListParams{
		TraceID: traceID,
	}
	if params != nil {
		p.PaginationParams = *params
	}
	return c.List(ctx, p)
}

// ListByObservation retrieves all scores for a specific observation.
func (c *ScoresClient) ListByObservation(ctx context.Context, observationID string, params *PaginationParams) (*ScoresListResponse, error) {
	p := &ScoresListParams{
		ObservationID: observationID,
	}
	if params != nil {
		p.PaginationParams = *params
	}
	return c.List(ctx, p)
}

// ============================================================================
// Score Builder (for ingestion API)
// ============================================================================

// ScoreBuilder provides a fluent interface for creating scores.
//
// ScoreBuilder is NOT safe for concurrent use. Each builder instance should
// be created, configured, and used within a single goroutine. All setter
// methods modify the builder in place and return the same pointer for method
// chaining.
//
// Validation is performed both on set (for early error detection) and at
// Create() time. Use HasErrors() to check for validation errors before
// calling Create(), or let Create() return the combined errors.
//
// Example:
//
//	err := trace.Score().
//	    Name("quality").
//	    NumericValue(0.95).
//	    Create(ctx)
type ScoreBuilder struct {
	ctx       *TraceContext
	score     *createScoreEvent
	validator Validator
}

// ID sets the score ID.
func (b *ScoreBuilder) ID(id string) *ScoreBuilder {
	b.score.ID = id
	return b
}

// Name sets the score name.
// Name is required and must not be empty.
func (b *ScoreBuilder) Name(name string) *ScoreBuilder {
	if err := ValidateName("name", name, MaxNameLength); err != nil {
		b.validator.AddError(err)
	}
	b.score.Name = name
	return b
}

// Value sets the score value.
func (b *ScoreBuilder) Value(value any) *ScoreBuilder {
	b.score.Value = value
	return b
}

// NumericValue sets a numeric score value.
func (b *ScoreBuilder) NumericValue(value float64) *ScoreBuilder {
	b.score.Value = value
	b.score.DataType = ScoreDataTypeNumeric
	return b
}

// CategoricalValue sets a categorical score value.
// Value must not be empty for categorical scores.
func (b *ScoreBuilder) CategoricalValue(value string) *ScoreBuilder {
	if value == "" {
		b.validator.AddFieldError("value", "categorical value cannot be empty")
	}
	b.score.StringValue = value
	b.score.DataType = ScoreDataTypeCategorical
	return b
}

// BooleanValue sets a boolean score value.
func (b *ScoreBuilder) BooleanValue(value bool) *ScoreBuilder {
	if value {
		b.score.Value = 1
	} else {
		b.score.Value = 0
	}
	b.score.DataType = ScoreDataTypeBoolean
	return b
}

// Comment sets the comment.
func (b *ScoreBuilder) Comment(comment string) *ScoreBuilder {
	b.score.Comment = comment
	return b
}

// ConfigID sets the score config ID.
func (b *ScoreBuilder) ConfigID(id string) *ScoreBuilder {
	b.score.ConfigID = id
	return b
}

// Environment sets the environment.
func (b *ScoreBuilder) Environment(env string) *ScoreBuilder {
	b.score.Environment = env
	return b
}

// Metadata sets the metadata.
// Validates that metadata keys are not empty.
func (b *ScoreBuilder) Metadata(metadata Metadata) *ScoreBuilder {
	if err := ValidateMetadata("metadata", metadata); err != nil {
		b.validator.AddError(err)
	}
	b.score.Metadata = metadata
	return b
}

// ObservationID sets the observation ID.
func (b *ScoreBuilder) ObservationID(id string) *ScoreBuilder {
	b.score.ObservationID = id
	return b
}

// HasErrors returns true if there are any validation errors.
func (b *ScoreBuilder) HasErrors() bool {
	return b.validator.HasErrors()
}

// Errors returns all accumulated validation errors.
func (b *ScoreBuilder) Errors() []error {
	return b.validator.Errors()
}

// Validate validates the score builder configuration.
// It returns any errors accumulated during setting plus final validation checks.
func (b *ScoreBuilder) Validate() error {
	// Check accumulated errors from setters
	if b.validator.HasErrors() {
		return b.validator.CombinedError()
	}

	// Final validation checks
	if b.score.Name == "" {
		return NewValidationError("name", "score name is required")
	}
	if b.score.TraceID == "" {
		return NewValidationError("traceId", "trace ID cannot be empty")
	}
	return nil
}

// Create creates the score.
func (b *ScoreBuilder) Create(ctx context.Context) error {
	if err := b.Validate(); err != nil {
		return err
	}

	event := ingestionEvent{
		ID:        generateID(),
		Type:      eventTypeScoreCreate,
		Timestamp: Now(),
		Body:      b.score,
	}

	return b.ctx.client.queueEvent(ctx, event)
}

// ============================================================================
// Sessions Client
// ============================================================================

// SessionsClient handles session-related API operations.
type SessionsClient struct {
	impl   *sessions.Client
	client *Client // kept for cross-client calls (GetWithTraces)
}

// newSessionsClient creates a new SessionsClient.
func newSessionsClient(client *Client) *SessionsClient {
	return &SessionsClient{
		impl:   sessions.New(client.HTTP()),
		client: client,
	}
}

// SessionsListParams represents parameters for listing sessions.
type SessionsListParams struct {
	PaginationParams
	FromTimestamp string
	ToTimestamp   string
}

// SessionsListResponse represents the response from listing sessions.
type SessionsListResponse struct {
	Data []Session    `json:"data"`
	Meta MetaResponse `json:"meta"`
}

// List retrieves a list of sessions.
func (c *SessionsClient) List(ctx context.Context, params *SessionsListParams) (*SessionsListResponse, error) {
	query := url.Values{}
	if params != nil {
		query = params.PaginationParams.ToQuery()
		if params.FromTimestamp != "" {
			query.Set("fromTimestamp", params.FromTimestamp)
		}
		if params.ToTimestamp != "" {
			query.Set("toTimestamp", params.ToTimestamp)
		}
	}

	var result SessionsListResponse
	if err := c.impl.List(ctx, query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a session by ID.
func (c *SessionsClient) Get(ctx context.Context, sessionID string) (*Session, error) {
	var result Session
	if err := c.impl.Get(ctx, sessionID, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// SessionWithTraces represents a session with its traces.
type SessionWithTraces struct {
	Session
	Traces []Trace `json:"traces"`
}

// GetWithTraces retrieves a session with all its traces.
func (c *SessionsClient) GetWithTraces(ctx context.Context, sessionID string) (*SessionWithTraces, error) {
	// First get the session
	session, err := c.Get(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	// Then get traces for this session
	tracesResp, err := c.client.Traces().List(ctx, &TracesListParams{
		FilterParams: FilterParams{
			SessionID: sessionID,
		},
	})
	if err != nil {
		return nil, err
	}

	return &SessionWithTraces{
		Session: *session,
		Traces:  tracesResp.Data,
	}, nil
}

// ============================================================================
// Models Client
// ============================================================================

// ModelsClient handles model-related API operations.
type ModelsClient struct {
	impl *models.Client
}

// newModelsClient creates a new ModelsClient.
func newModelsClient(client *Client) *ModelsClient {
	return &ModelsClient{
		impl: models.New(client.HTTP()),
	}
}

// ModelsListParams represents parameters for listing models.
type ModelsListParams struct {
	PaginationParams
}

// ModelsListResponse represents the response from listing models.
type ModelsListResponse struct {
	Data []Model      `json:"data"`
	Meta MetaResponse `json:"meta"`
}

// List retrieves a list of models.
func (c *ModelsClient) List(ctx context.Context, params *ModelsListParams) (*ModelsListResponse, error) {
	query := url.Values{}
	if params != nil {
		query = params.PaginationParams.ToQuery()
	}

	var result ModelsListResponse
	if err := c.impl.List(ctx, query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a model by ID.
func (c *ModelsClient) Get(ctx context.Context, modelID string) (*Model, error) {
	var result Model
	if err := c.impl.Get(ctx, modelID, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateModelRequest represents a request to create a model.
type CreateModelRequest struct {
	ModelName       string         `json:"modelName"`
	MatchPattern    string         `json:"matchPattern,omitempty"`
	StartDate       Time           `json:"startDate,omitempty"`
	InputPrice      float64        `json:"inputPrice,omitempty"`
	OutputPrice     float64        `json:"outputPrice,omitempty"`
	TotalPrice      float64        `json:"totalPrice,omitempty"`
	Unit            string         `json:"unit,omitempty"`
	Tokenizer       string         `json:"tokenizer,omitempty"`
	TokenizerConfig map[string]any `json:"tokenizerConfig,omitempty"`
}

// Create creates a new model definition.
func (c *ModelsClient) Create(ctx context.Context, req *CreateModelRequest) (*Model, error) {
	if req == nil {
		return nil, ErrNilRequest
	}
	if req.ModelName == "" {
		return nil, NewValidationError("modelName", "model name is required")
	}

	var result Model
	if err := c.impl.Create(ctx, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a model by ID (only user-defined models can be deleted).
func (c *ModelsClient) Delete(ctx context.Context, modelID string) error {
	return c.impl.Delete(ctx, modelID)
}

// ============================================================================
// Prompts Client
// ============================================================================

// PromptsClient handles prompt-related API operations.
type PromptsClient struct {
	impl *prompts.Client
}

// newPromptsClient creates a new PromptsClient.
func newPromptsClient(client *Client) *PromptsClient {
	return &PromptsClient{
		impl: prompts.New(client.HTTP()),
	}
}

// PromptsListParams represents parameters for listing prompts.
type PromptsListParams struct {
	PaginationParams
	Name  string
	Label string
	Tag   string
}

// PromptsListResponse represents the response from listing prompts.
type PromptsListResponse struct {
	Data []Prompt     `json:"data"`
	Meta MetaResponse `json:"meta"`
}

// List retrieves a list of prompts.
func (c *PromptsClient) List(ctx context.Context, params *PromptsListParams) (*PromptsListResponse, error) {
	query := url.Values{}
	if params != nil {
		query = params.PaginationParams.ToQuery()
		if params.Name != "" {
			query.Set("name", params.Name)
		}
		if params.Label != "" {
			query.Set("label", params.Label)
		}
		if params.Tag != "" {
			query.Set("tag", params.Tag)
		}
	}

	var result PromptsListResponse
	if err := c.impl.List(ctx, query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPromptParams represents parameters for getting a prompt.
type GetPromptParams struct {
	Version int
	Label   string
}

// Get retrieves a prompt by name with optional version or label.
func (c *PromptsClient) Get(ctx context.Context, name string, params *GetPromptParams) (*Prompt, error) {
	query := url.Values{}
	if params != nil {
		if params.Version > 0 {
			query.Set("version", strconv.Itoa(params.Version))
		}
		if params.Label != "" {
			query.Set("label", params.Label)
		}
	}

	var result Prompt
	if err := c.impl.Get(ctx, name, query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetLatest retrieves the latest version of a prompt by name.
func (c *PromptsClient) GetLatest(ctx context.Context, name string) (*Prompt, error) {
	return c.Get(ctx, name, nil)
}

// GetByVersion retrieves a specific version of a prompt.
func (c *PromptsClient) GetByVersion(ctx context.Context, name string, version int) (*Prompt, error) {
	return c.Get(ctx, name, &GetPromptParams{Version: version})
}

// GetByLabel retrieves a prompt by name and label (e.g., "production").
func (c *PromptsClient) GetByLabel(ctx context.Context, name string, label string) (*Prompt, error) {
	return c.Get(ctx, name, &GetPromptParams{Label: label})
}

// CreatePromptRequest represents a request to create a prompt.
type CreatePromptRequest struct {
	Name   string         `json:"name"`
	Prompt any            `json:"prompt"`
	Type   string         `json:"type,omitempty"`
	Config map[string]any `json:"config,omitempty"`
	Labels []string       `json:"labels,omitempty"`
	Tags   []string       `json:"tags,omitempty"`
}

// Create creates a new prompt.
func (c *PromptsClient) Create(ctx context.Context, req *CreatePromptRequest) (*Prompt, error) {
	if req == nil {
		return nil, ErrNilRequest
	}
	if req.Name == "" {
		return nil, NewValidationError("name", "prompt name is required")
	}
	if req.Prompt == nil {
		return nil, NewValidationError("prompt", "prompt content is required")
	}

	var result Prompt
	if err := c.impl.Create(ctx, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateTextPrompt creates a new text prompt.
func (c *PromptsClient) CreateTextPrompt(ctx context.Context, name string, prompt string, labels []string) (*Prompt, error) {
	return c.Create(ctx, &CreatePromptRequest{
		Name:   name,
		Prompt: prompt,
		Type:   "text",
		Labels: labels,
	})
}

// CreateChatPrompt creates a new chat prompt.
func (c *PromptsClient) CreateChatPrompt(ctx context.Context, name string, messages []ChatMessage, labels []string) (*Prompt, error) {
	return c.Create(ctx, &CreatePromptRequest{
		Name:   name,
		Prompt: messages,
		Type:   "chat",
		Labels: labels,
	})
}

// Note: Compile and CompileChatMessages methods are defined on the Prompt type
// in pkg/types/prompt.go and are available via the type alias.

// ============================================================================
// Datasets Client
// ============================================================================

// DatasetsClient handles dataset-related API operations.
type DatasetsClient struct {
	impl *datasets.Client
}

// newDatasetsClient creates a new DatasetsClient.
func newDatasetsClient(client *Client) *DatasetsClient {
	return &DatasetsClient{
		impl: datasets.New(client.HTTP()),
	}
}

// DatasetsListParams represents parameters for listing datasets.
type DatasetsListParams struct {
	PaginationParams
}

// DatasetsListResponse represents the response from listing datasets.
type DatasetsListResponse struct {
	Data []Dataset    `json:"data"`
	Meta MetaResponse `json:"meta"`
}

// List retrieves a list of datasets.
func (c *DatasetsClient) List(ctx context.Context, params *DatasetsListParams) (*DatasetsListResponse, error) {
	query := url.Values{}
	if params != nil {
		query = params.PaginationParams.ToQuery()
	}

	var result DatasetsListResponse
	if err := c.impl.List(ctx, query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a dataset by name.
func (c *DatasetsClient) Get(ctx context.Context, datasetName string) (*Dataset, error) {
	var result Dataset
	if err := c.impl.Get(ctx, datasetName, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateDatasetRequest represents a request to create a dataset.
type CreateDatasetRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Metadata    Metadata `json:"metadata,omitempty"`
}

// Create creates a new dataset.
func (c *DatasetsClient) Create(ctx context.Context, req *CreateDatasetRequest) (*Dataset, error) {
	if req == nil {
		return nil, ErrNilRequest
	}
	if req.Name == "" {
		return nil, NewValidationError("name", "dataset name is required")
	}

	var result Dataset
	if err := c.impl.Create(ctx, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DatasetItemsListParams represents parameters for listing dataset items.
type DatasetItemsListParams struct {
	PaginationParams
	DatasetName         string
	SourceTraceID       string
	SourceObservationID string
}

// DatasetItemsListResponse represents the response from listing dataset items.
type DatasetItemsListResponse struct {
	Data []DatasetItem `json:"data"`
	Meta MetaResponse  `json:"meta"`
}

// ListItems retrieves items in a dataset.
func (c *DatasetsClient) ListItems(ctx context.Context, params *DatasetItemsListParams) (*DatasetItemsListResponse, error) {
	query := url.Values{}
	if params != nil {
		query = params.PaginationParams.ToQuery()
		if params.DatasetName != "" {
			query.Set("datasetName", params.DatasetName)
		}
		if params.SourceTraceID != "" {
			query.Set("sourceTraceId", params.SourceTraceID)
		}
		if params.SourceObservationID != "" {
			query.Set("sourceObservationId", params.SourceObservationID)
		}
	}

	var result DatasetItemsListResponse
	if err := c.impl.ListItems(ctx, query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetItem retrieves a dataset item by ID.
func (c *DatasetsClient) GetItem(ctx context.Context, itemID string) (*DatasetItem, error) {
	var result DatasetItem
	if err := c.impl.GetItem(ctx, itemID, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateDatasetItemRequest represents a request to create a dataset item.
type CreateDatasetItemRequest struct {
	DatasetName         string   `json:"datasetName"`
	Input               any      `json:"input,omitempty"`
	ExpectedOutput      any      `json:"expectedOutput,omitempty"`
	Metadata            Metadata `json:"metadata,omitempty"`
	SourceTraceID       string   `json:"sourceTraceId,omitempty"`
	SourceObservationID string   `json:"sourceObservationId,omitempty"`
	Status              string   `json:"status,omitempty"`
	ID                  string   `json:"id,omitempty"`
}

// CreateItem creates a new dataset item.
func (c *DatasetsClient) CreateItem(ctx context.Context, req *CreateDatasetItemRequest) (*DatasetItem, error) {
	if req == nil {
		return nil, ErrNilRequest
	}
	if req.DatasetName == "" {
		return nil, NewValidationError("datasetName", "dataset name is required")
	}

	var result DatasetItem
	if err := c.impl.CreateItem(ctx, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteItem deletes a dataset item by ID.
func (c *DatasetsClient) DeleteItem(ctx context.Context, itemID string) error {
	return c.impl.DeleteItem(ctx, itemID)
}

// DatasetRunsListResponse represents the response from listing dataset runs.
type DatasetRunsListResponse struct {
	Data []DatasetRun `json:"data"`
	Meta MetaResponse `json:"meta"`
}

// ListRuns retrieves runs for a dataset.
func (c *DatasetsClient) ListRuns(ctx context.Context, datasetName string, params *PaginationParams) (*DatasetRunsListResponse, error) {
	query := url.Values{}
	if params != nil {
		query = params.ToQuery()
	}

	var result DatasetRunsListResponse
	if err := c.impl.ListRuns(ctx, datasetName, query, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetRun retrieves a dataset run by name.
func (c *DatasetsClient) GetRun(ctx context.Context, datasetName string, runName string) (*DatasetRun, error) {
	var result DatasetRun
	if err := c.impl.GetRun(ctx, datasetName, runName, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DeleteRun deletes a dataset run.
func (c *DatasetsClient) DeleteRun(ctx context.Context, datasetName string, runName string) error {
	return c.impl.DeleteRun(ctx, datasetName, runName)
}

// CreateDatasetRunItemRequest represents a request to create a dataset run item.
type CreateDatasetRunItemRequest struct {
	DatasetItemID  string   `json:"datasetItemId"`
	RunName        string   `json:"runName"`
	RunDescription string   `json:"runDescription,omitempty"`
	TraceID        string   `json:"traceId,omitempty"`
	ObservationID  string   `json:"observationId,omitempty"`
	Metadata       Metadata `json:"metadata,omitempty"`
}

// CreateRunItem creates a dataset run item (links a trace/observation to a dataset item).
func (c *DatasetsClient) CreateRunItem(ctx context.Context, req *CreateDatasetRunItemRequest) (*DatasetRunItem, error) {
	if req == nil {
		return nil, ErrNilRequest
	}
	if req.DatasetItemID == "" {
		return nil, NewValidationError("datasetItemId", "dataset item ID is required")
	}
	if req.RunName == "" {
		return nil, NewValidationError("runName", "run name is required")
	}

	var result DatasetRunItem
	if err := c.impl.CreateRunItem(ctx, req, &result); err != nil {
		return nil, err
	}
	return &result, nil
}
