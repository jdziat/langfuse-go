package langfuse

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
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
	err := c.client.http.get(ctx, endpoints.Prompts, query, &result)
	if err != nil {
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
	err := c.client.http.get(ctx, fmt.Sprintf("%s/%s", endpoints.Prompts, name), query, &result)
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
	err := c.client.http.post(ctx, endpoints.Prompts, req, &result)
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
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result, nil
}

// CompileChatMessages compiles chat messages with variables.
// Returns all successfully compiled messages along with any errors encountered.
// If errors occurred, a CompilationError is returned containing all individual errors.
func (p *Prompt) CompileChatMessages(variables map[string]string) ([]ChatMessage, error) {
	messages, ok := p.Prompt.([]any)
	if !ok {
		return nil, NewValidationError("prompt", "prompt is not a chat prompt")
	}

	var errs []error
	result := make([]ChatMessage, 0, len(messages))

	for i, msg := range messages {
		msgMap, ok := msg.(map[string]any)
		if !ok {
			errs = append(errs, fmt.Errorf("message %d: expected object, got %T", i, msg))
			continue
		}

		role, roleOk := msgMap["role"].(string)
		if !roleOk {
			errs = append(errs, fmt.Errorf("message %d: missing or invalid 'role' field", i))
		}

		content, contentOk := msgMap["content"].(string)
		if !contentOk {
			errs = append(errs, fmt.Errorf("message %d: missing or invalid 'content' field", i))
		}

		// Skip messages with missing required fields
		if !roleOk || !contentOk {
			continue
		}

		// Apply variable substitution
		for key, value := range variables {
			placeholder := "{{" + key + "}}"
			content = strings.ReplaceAll(content, placeholder, value)
		}

		result = append(result, ChatMessage{
			Role:    role,
			Content: content,
		})
	}

	if len(errs) > 0 {
		return result, &CompilationError{Errors: errs}
	}
	return result, nil
}
