package langfuse

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
)

// PromptsClient handles prompt-related API operations.
type PromptsClient struct {
	client *Client
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
	err := c.client.http.get(ctx, "/v2/prompts", query, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// GetParams represents parameters for getting a prompt.
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
	err := c.client.http.get(ctx, fmt.Sprintf("/v2/prompts/%s", name), query, &result)
	if err != nil {
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
	Name   string                 `json:"name"`
	Prompt interface{}            `json:"prompt"`
	Type   string                 `json:"type,omitempty"`
	Config map[string]interface{} `json:"config,omitempty"`
	Labels []string               `json:"labels,omitempty"`
	Tags   []string               `json:"tags,omitempty"`
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
	err := c.client.http.post(ctx, "/v2/prompts", req, &result)
	if err != nil {
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

// Compile compiles a text prompt with variables.
func (p *Prompt) Compile(variables map[string]string) (string, error) {
	promptStr, ok := p.Prompt.(string)
	if !ok {
		return "", NewValidationError("prompt", "prompt is not a text prompt")
	}

	result := promptStr
	for key, value := range variables {
		placeholder := "{{" + key + "}}"
		result = replaceAll(result, placeholder, value)
	}
	return result, nil
}

// CompileChatMessages compiles chat messages with variables.
func (p *Prompt) CompileChatMessages(variables map[string]string) ([]ChatMessage, error) {
	messages, ok := p.Prompt.([]interface{})
	if !ok {
		return nil, NewValidationError("prompt", "prompt is not a chat prompt")
	}

	result := make([]ChatMessage, 0, len(messages))
	for _, msg := range messages {
		msgMap, ok := msg.(map[string]interface{})
		if !ok {
			continue
		}
		role, _ := msgMap["role"].(string)
		content, _ := msgMap["content"].(string)

		for key, value := range variables {
			placeholder := "{{" + key + "}}"
			content = replaceAll(content, placeholder, value)
		}

		result = append(result, ChatMessage{
			Role:    role,
			Content: content,
		})
	}
	return result, nil
}

// replaceAll replaces all occurrences of old with new in s.
func replaceAll(s, old, new string) string {
	if old == "" {
		return s
	}
	result := ""
	for {
		i := indexOf(s, old)
		if i == -1 {
			return result + s
		}
		result += s[:i] + new
		s = s[i+len(old):]
	}
}

// indexOf returns the index of substr in s, or -1 if not found.
func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
