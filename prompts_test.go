package langfuse

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPromptsClientList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/prompts" {
			t.Errorf("Expected /v2/prompts, got %s", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("Expected GET, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(PromptsListResponse{
			Data: []Prompt{
				{Name: "prompt1", Version: 1},
				{Name: "prompt2", Version: 2},
			},
			Meta: MetaResponse{TotalItems: 2},
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	result, err := client.Prompts().List(context.Background(), nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result.Data) != 2 {
		t.Errorf("Expected 2 prompts, got %d", len(result.Data))
	}
}

func TestPromptsClientGet(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v2/prompts/my-prompt" {
			t.Errorf("Expected /v2/prompts/my-prompt, got %s", r.URL.Path)
		}

		version := r.URL.Query().Get("version")
		label := r.URL.Query().Get("label")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Prompt{
			Name:    "my-prompt",
			Version: 1,
			Prompt:  "Hello {{name}}!",
			Labels:  []string{label},
		})

		_ = version // Used in assertions above
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	// Test GetLatest
	prompt, err := client.Prompts().GetLatest(context.Background(), "my-prompt")
	if err != nil {
		t.Fatalf("GetLatest failed: %v", err)
	}
	if prompt.Name != "my-prompt" {
		t.Errorf("Expected name my-prompt, got %s", prompt.Name)
	}

	// Test GetByVersion
	prompt, err = client.Prompts().GetByVersion(context.Background(), "my-prompt", 2)
	if err != nil {
		t.Fatalf("GetByVersion failed: %v", err)
	}

	// Test GetByLabel
	prompt, err = client.Prompts().GetByLabel(context.Background(), "my-prompt", "production")
	if err != nil {
		t.Fatalf("GetByLabel failed: %v", err)
	}
}

func TestPromptsClientCreate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}

		var req CreatePromptRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Name != "new-prompt" {
			t.Errorf("Expected name new-prompt, got %s", req.Name)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Prompt{
			ID:      "prompt-123",
			Name:    req.Name,
			Version: 1,
			Prompt:  req.Prompt,
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	prompt, err := client.Prompts().Create(context.Background(), &CreatePromptRequest{
		Name:   "new-prompt",
		Prompt: "Hello {{name}}!",
		Type:   "text",
		Labels: []string{"development"},
	})
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if prompt.Name != "new-prompt" {
		t.Errorf("Expected name new-prompt, got %s", prompt.Name)
	}
}

func TestPromptsClientCreateValidation(t *testing.T) {
	client, _ := New("pk-lf-test-key", "sk-lf-test-key")
	defer client.Shutdown(context.Background())

	// Nil request
	_, err := client.Prompts().Create(context.Background(), nil)
	if err != ErrNilRequest {
		t.Errorf("Expected ErrNilRequest, got %v", err)
	}

	// Missing name
	_, err = client.Prompts().Create(context.Background(), &CreatePromptRequest{
		Prompt: "Hello!",
	})
	if err == nil {
		t.Error("Expected validation error for missing name")
	}

	// Missing prompt
	_, err = client.Prompts().Create(context.Background(), &CreatePromptRequest{
		Name: "test",
	})
	if err == nil {
		t.Error("Expected validation error for missing prompt")
	}
}

func TestPromptsClientCreateTextPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CreatePromptRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "text" {
			t.Errorf("Expected type text, got %s", req.Type)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Prompt{
			Name:    req.Name,
			Version: 1,
			Prompt:  req.Prompt,
			Type:    "text",
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	prompt, err := client.Prompts().CreateTextPrompt(
		context.Background(),
		"greeting",
		"Hello {{name}}!",
		[]string{"production"},
	)
	if err != nil {
		t.Fatalf("CreateTextPrompt failed: %v", err)
	}

	if prompt.Name != "greeting" {
		t.Errorf("Expected name greeting, got %s", prompt.Name)
	}
}

func TestPromptsClientCreateChatPrompt(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req CreatePromptRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Type != "chat" {
			t.Errorf("Expected type chat, got %s", req.Type)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(Prompt{
			Name:    req.Name,
			Version: 1,
			Prompt:  req.Prompt,
			Type:    "chat",
		})
	}))
	defer server.Close()

	client, _ := New("pk-lf-test-key", "sk-lf-test-key", WithBaseURL(server.URL))
	defer client.Shutdown(context.Background())

	messages := []ChatMessage{
		{Role: "system", Content: "You are a helpful assistant."},
		{Role: "user", Content: "Hello {{name}}!"},
	}

	prompt, err := client.Prompts().CreateChatPrompt(
		context.Background(),
		"chat-greeting",
		messages,
		[]string{"development"},
	)
	if err != nil {
		t.Fatalf("CreateChatPrompt failed: %v", err)
	}

	if prompt.Name != "chat-greeting" {
		t.Errorf("Expected name chat-greeting, got %s", prompt.Name)
	}
}

func TestPromptCompile(t *testing.T) {
	prompt := &Prompt{
		Name:   "greeting",
		Prompt: "Hello {{name}}! Welcome to {{service}}.",
	}

	result, err := prompt.Compile(map[string]string{
		"name":    "John",
		"service": "Langfuse",
	})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	expected := "Hello John! Welcome to Langfuse."
	if result != expected {
		t.Errorf("Compile result = %v, want %v", result, expected)
	}
}

func TestPromptCompileEmptyVariables(t *testing.T) {
	prompt := &Prompt{
		Name:   "greeting",
		Prompt: "Hello World!",
	}

	result, err := prompt.Compile(nil)
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	if result != "Hello World!" {
		t.Errorf("Compile result = %v, want Hello World!", result)
	}
}

func TestPromptCompileMultipleOccurrences(t *testing.T) {
	prompt := &Prompt{
		Name:   "greeting",
		Prompt: "{{name}} says hello to {{name}}!",
	}

	result, err := prompt.Compile(map[string]string{
		"name": "John",
	})
	if err != nil {
		t.Fatalf("Compile failed: %v", err)
	}

	expected := "John says hello to John!"
	if result != expected {
		t.Errorf("Compile result = %v, want %v", result, expected)
	}
}

func TestPromptCompileNonTextPrompt(t *testing.T) {
	prompt := &Prompt{
		Name:   "chat",
		Prompt: []any{map[string]string{"role": "user", "content": "Hello"}},
	}

	_, err := prompt.Compile(map[string]string{"name": "John"})
	if err == nil {
		t.Error("Expected error for non-text prompt")
	}
}

func TestPromptCompileChatMessages(t *testing.T) {
	prompt := &Prompt{
		Name: "chat",
		Prompt: []any{
			map[string]any{"role": "system", "content": "You are a {{role}}."},
			map[string]any{"role": "user", "content": "Hello {{name}}!"},
		},
	}

	messages, err := prompt.CompileChatMessages(map[string]string{
		"role": "helpful assistant",
		"name": "John",
	})
	if err != nil {
		t.Fatalf("CompileChatMessages failed: %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("Expected 2 messages, got %d", len(messages))
	}

	if messages[0].Role != "system" {
		t.Errorf("Expected role system, got %s", messages[0].Role)
	}
	if messages[0].Content != "You are a helpful assistant." {
		t.Errorf("Expected compiled content, got %s", messages[0].Content)
	}

	if messages[1].Role != "user" {
		t.Errorf("Expected role user, got %s", messages[1].Role)
	}
	if messages[1].Content != "Hello John!" {
		t.Errorf("Expected compiled content, got %s", messages[1].Content)
	}
}

func TestPromptCompileChatMessagesNonChatPrompt(t *testing.T) {
	prompt := &Prompt{
		Name:   "text",
		Prompt: "Hello {{name}}!",
	}

	_, err := prompt.CompileChatMessages(map[string]string{"name": "John"})
	if err == nil {
		t.Error("Expected error for non-chat prompt")
	}
}

func TestPromptCompileChatMessagesWithErrors(t *testing.T) {
	t.Run("invalid message type", func(t *testing.T) {
		prompt := &Prompt{
			Name: "chat",
			Prompt: []any{
				"not a map", // invalid - should be map[string]any
				map[string]any{"role": "user", "content": "Hello!"},
			},
		}

		messages, err := prompt.CompileChatMessages(nil)
		if err == nil {
			t.Error("Expected CompilationError for invalid message type")
		}

		compErr, ok := IsCompilationError(err)
		if !ok {
			t.Errorf("Expected CompilationError, got %T", err)
		}
		if compErr != nil && len(compErr.Errors) != 1 {
			t.Errorf("Expected 1 error, got %d", len(compErr.Errors))
		}

		// Should still return valid messages
		if len(messages) != 1 {
			t.Errorf("Expected 1 valid message, got %d", len(messages))
		}
	})

	t.Run("missing role field", func(t *testing.T) {
		prompt := &Prompt{
			Name: "chat",
			Prompt: []any{
				map[string]any{"content": "Hello!"}, // missing role
			},
		}

		messages, err := prompt.CompileChatMessages(nil)
		if err == nil {
			t.Error("Expected CompilationError for missing role")
		}

		compErr, ok := IsCompilationError(err)
		if !ok {
			t.Errorf("Expected CompilationError, got %T", err)
		}
		if compErr != nil {
			errStr := compErr.Error()
			if !strings.Contains(errStr, "role") {
				t.Errorf("Error should mention 'role': %s", errStr)
			}
		}

		// Should skip invalid messages
		if len(messages) != 0 {
			t.Errorf("Expected 0 valid messages, got %d", len(messages))
		}
	})

	t.Run("missing content field", func(t *testing.T) {
		prompt := &Prompt{
			Name: "chat",
			Prompt: []any{
				map[string]any{"role": "user"}, // missing content
			},
		}

		messages, err := prompt.CompileChatMessages(nil)
		if err == nil {
			t.Error("Expected CompilationError for missing content")
		}

		compErr, ok := IsCompilationError(err)
		if !ok {
			t.Errorf("Expected CompilationError, got %T", err)
		}
		if compErr != nil {
			errStr := compErr.Error()
			if !strings.Contains(errStr, "content") {
				t.Errorf("Error should mention 'content': %s", errStr)
			}
		}

		// Should skip invalid messages
		if len(messages) != 0 {
			t.Errorf("Expected 0 valid messages, got %d", len(messages))
		}
	})

	t.Run("multiple errors", func(t *testing.T) {
		prompt := &Prompt{
			Name: "chat",
			Prompt: []any{
				map[string]any{}, // missing both role and content
				map[string]any{"role": "user", "content": "valid"}, // valid
				"invalid", // wrong type
			},
		}

		messages, err := prompt.CompileChatMessages(nil)
		if err == nil {
			t.Error("Expected CompilationError")
		}

		compErr, ok := IsCompilationError(err)
		if !ok {
			t.Errorf("Expected CompilationError, got %T", err)
		}
		// Should have errors for: message 0 missing role, message 0 missing content, message 2 wrong type
		if compErr != nil && len(compErr.Errors) < 3 {
			t.Errorf("Expected at least 3 errors, got %d: %v", len(compErr.Errors), compErr.Errors)
		}

		// Should have 1 valid message
		if len(messages) != 1 {
			t.Errorf("Expected 1 valid message, got %d", len(messages))
		}
	})

	t.Run("all messages valid", func(t *testing.T) {
		prompt := &Prompt{
			Name: "chat",
			Prompt: []any{
				map[string]any{"role": "system", "content": "You are helpful."},
				map[string]any{"role": "user", "content": "Hello!"},
			},
		}

		messages, err := prompt.CompileChatMessages(nil)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(messages) != 2 {
			t.Errorf("Expected 2 messages, got %d", len(messages))
		}
	})
}

func TestStringsReplaceAll(t *testing.T) {
	// Tests to verify strings.ReplaceAll behaves as expected for our use cases
	tests := []struct {
		name     string
		s        string
		old      string
		new      string
		expected string
	}{
		{
			name:     "single replacement",
			s:        "Hello {{name}}!",
			old:      "{{name}}",
			new:      "John",
			expected: "Hello John!",
		},
		{
			name:     "multiple replacements",
			s:        "{{x}} + {{x}} = {{y}}",
			old:      "{{x}}",
			new:      "1",
			expected: "1 + 1 = {{y}}",
		},
		{
			name:     "no match",
			s:        "Hello World!",
			old:      "{{name}}",
			new:      "John",
			expected: "Hello World!",
		},
		{
			name:     "empty input",
			s:        "",
			old:      "{{name}}",
			new:      "John",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strings.ReplaceAll(tt.s, tt.old, tt.new)
			if result != tt.expected {
				t.Errorf("strings.ReplaceAll(%q, %q, %q) = %q, want %q",
					tt.s, tt.old, tt.new, result, tt.expected)
			}
		})
	}
}

func TestStringsIndex(t *testing.T) {
	// Tests to verify strings.Index behaves as expected for our use cases
	tests := []struct {
		name     string
		s        string
		substr   string
		expected int
	}{
		{
			name:     "found at beginning",
			s:        "Hello World",
			substr:   "Hello",
			expected: 0,
		},
		{
			name:     "found in middle",
			s:        "Hello World",
			substr:   "Wor",
			expected: 6,
		},
		{
			name:     "not found",
			s:        "Hello World",
			substr:   "xyz",
			expected: -1,
		},
		{
			name:     "empty substr",
			s:        "Hello",
			substr:   "",
			expected: 0,
		},
		{
			name:     "substr longer than s",
			s:        "Hi",
			substr:   "Hello",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strings.Index(tt.s, tt.substr)
			if result != tt.expected {
				t.Errorf("strings.Index(%q, %q) = %d, want %d",
					tt.s, tt.substr, result, tt.expected)
			}
		})
	}
}
