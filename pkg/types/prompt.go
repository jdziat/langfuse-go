package types

import (
	"fmt"
	"strings"
)

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

// CompilationError contains errors encountered during prompt compilation.
type CompilationError struct {
	Errors []error
}

// Error implements the error interface.
func (e *CompilationError) Error() string {
	if len(e.Errors) == 0 {
		return "compilation error"
	}
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}
	return fmt.Sprintf("%d compilation errors: %v", len(e.Errors), e.Errors[0])
}

// Compile compiles a text prompt with variables.
func (p *Prompt) Compile(variables map[string]string) (string, error) {
	promptStr, ok := p.Prompt.(string)
	if !ok {
		return "", fmt.Errorf("prompt is not a text prompt")
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
		return nil, fmt.Errorf("prompt is not a chat prompt")
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
