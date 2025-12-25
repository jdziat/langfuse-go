package langfuse

import (
	"context"
	"fmt"
	"net/url"
)

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
	err := c.client.http.get(ctx, "/scores", query, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a single score by ID.
func (c *ScoresClient) Get(ctx context.Context, scoreID string) (*Score, error) {
	var result Score
	err := c.client.http.get(ctx, fmt.Sprintf("/scores/%s", scoreID), nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// CreateScoreRequest represents a request to create a score.
type CreateScoreRequest struct {
	TraceID       string                 `json:"traceId"`
	ObservationID string                 `json:"observationId,omitempty"`
	Name          string                 `json:"name"`
	Value         interface{}            `json:"value"`
	StringValue   string                 `json:"stringValue,omitempty"`
	DataType      ScoreDataType          `json:"dataType,omitempty"`
	Comment       string                 `json:"comment,omitempty"`
	ConfigID      string                 `json:"configId,omitempty"`
	Environment   string                 `json:"environment,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
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
	err := c.client.http.post(ctx, "/scores", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a score by ID.
func (c *ScoresClient) Delete(ctx context.Context, scoreID string) error {
	return c.client.http.delete(ctx, fmt.Sprintf("/scores/%s", scoreID), nil)
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
