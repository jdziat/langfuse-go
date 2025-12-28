package langfuse

import (
	"context"
	"fmt"
	"net/url"
)

// ============================================================================
// Scores API Client
// ============================================================================

// ScoresClient handles score-related API operations.
type ScoresClient struct {
	client *Client
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
	err := c.client.http.get(ctx, endpoints.Scores, query, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single score by ID.
func (c *ScoresClient) Get(ctx context.Context, scoreID string) (*Score, error) {
	var result Score
	err := c.client.http.get(ctx, fmt.Sprintf("%s/%s", endpoints.Scores, scoreID), nil, &result)
	if err != nil {
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
	err := c.client.http.post(ctx, endpoints.Scores, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a score by ID.
func (c *ScoresClient) Delete(ctx context.Context, scoreID string) error {
	return c.client.http.delete(ctx, fmt.Sprintf("%s/%s", endpoints.Scores, scoreID), nil)
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
