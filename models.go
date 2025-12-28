package langfuse

import (
	"context"
	"fmt"
	"net/url"
)

// ModelsClient handles model-related API operations.
type ModelsClient struct {
	client *Client
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
	err := c.client.http.get(ctx, endpoints.Models, query, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Get retrieves a model by ID.
func (c *ModelsClient) Get(ctx context.Context, modelID string) (*Model, error) {
	var result Model
	err := c.client.http.get(ctx, fmt.Sprintf("%s/%s", endpoints.Models, modelID), nil, &result)
	if err != nil {
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
	err := c.client.http.post(ctx, endpoints.Models, req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Delete deletes a model by ID (only user-defined models can be deleted).
func (c *ModelsClient) Delete(ctx context.Context, modelID string) error {
	return c.client.http.delete(ctx, fmt.Sprintf("%s/%s", endpoints.Models, modelID), nil)
}
