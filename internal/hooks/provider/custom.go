package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Custom implements the Provider interface for custom OpenAI-compatible endpoints.
type Custom struct {
	endpoint  string
	apiKey    string
	model     string
	maxTokens int
	client    *http.Client
}

// NewCustom creates a new Custom provider.
func NewCustom(endpoint, apiKey, model string, maxTokens int) *Custom {
	return &Custom{
		endpoint:  endpoint,
		apiKey:    apiKey,
		model:     model,
		maxTokens: maxTokens,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Name returns the provider name.
func (c *Custom) Name() string {
	return "custom"
}

// Complete sends a completion request to the custom endpoint.
// Uses OpenAI-compatible request/response format.
func (c *Custom) Complete(ctx context.Context, req *Request) (*Response, error) {
	messages := make([]openAIMessage, len(req.Messages))
	for i, m := range req.Messages {
		messages[i] = openAIMessage(m)
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = c.maxTokens
	}

	body := openAIRequest{
		Model:     c.model,
		Messages:  messages,
		MaxTokens: maxTokens,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var openAIResp openAIResponse
	if err := json.Unmarshal(respBody, &openAIResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if openAIResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", openAIResp.Error.Message)
	}

	if len(openAIResp.Choices) == 0 {
		return nil, fmt.Errorf("no completion choices returned")
	}

	return &Response{
		Content: openAIResp.Choices[0].Message.Content,
	}, nil
}
