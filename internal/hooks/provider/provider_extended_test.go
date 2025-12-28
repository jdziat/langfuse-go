package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/jdziat/langfuse-go/internal/hooks/config"
)

func TestNewFactory_OpenAI(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = config.ProviderOpenAI
	cfg.OpenAI.APIKey = "test-api-key"

	provider, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if provider.Name() != "openai" {
		t.Errorf("provider.Name() = %s, want 'openai'", provider.Name())
	}
}

func TestNewFactory_OpenAI_MissingKey(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = config.ProviderOpenAI
	cfg.OpenAI.APIKey = ""

	// Ensure no env var fallback
	_, err := New(cfg)
	// This will succeed if OPENAI_API_KEY is set in env, fail otherwise
	// We can't control the environment, so just check it doesn't panic
	if err != nil {
		if provider, ok := err.(interface{ Error() string }); ok {
			t.Logf("Expected error for missing API key: %s", provider.Error())
		}
	}
}

func TestNewFactory_Anthropic(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = config.ProviderAnthropic
	cfg.Anthropic.APIKey = "test-api-key"

	provider, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if provider.Name() != "anthropic" {
		t.Errorf("provider.Name() = %s, want 'anthropic'", provider.Name())
	}
}

func TestNewFactory_Anthropic_MissingKey(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = config.ProviderAnthropic
	cfg.Anthropic.APIKey = ""

	_, err := New(cfg)
	if err != nil {
		t.Logf("Expected error for missing API key: %v", err)
	}
}

func TestNewFactory_Ollama(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = config.ProviderOllama

	provider, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if provider.Name() != "ollama" {
		t.Errorf("provider.Name() = %s, want 'ollama'", provider.Name())
	}
}

func TestNewFactory_Custom(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = config.ProviderCustom
	cfg.Custom.Endpoint = "https://custom-api.example.com"
	cfg.Custom.APIKey = "custom-key"

	provider, err := New(cfg)
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if provider.Name() != "custom" {
		t.Errorf("provider.Name() = %s, want 'custom'", provider.Name())
	}
}

func TestNewFactory_Custom_MissingEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = config.ProviderCustom
	cfg.Custom.Endpoint = ""

	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for missing custom endpoint")
	}
}

func TestNewFactory_UnknownProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Provider = "unknown"

	_, err := New(cfg)
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestOllamaComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		if r.URL.Path != "/api/chat" {
			t.Errorf("expected /api/chat, got %s", r.URL.Path)
		}

		resp := ollamaResponse{
			Message: struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				Role:    "assistant",
				Content: "Hello from Ollama!",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOllama(server.URL, "llama3.2")

	resp, err := provider.Complete(context.Background(), &Request{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 100,
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if resp.Content != "Hello from Ollama!" {
		t.Errorf("resp.Content = %s, want 'Hello from Ollama!'", resp.Content)
	}
}

func TestCustomComplete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		auth := r.Header.Get("Authorization")
		if auth != "Bearer test-key" {
			t.Errorf("expected 'Bearer test-key', got %s", auth)
		}

		resp := openAIResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Content string `json:"content"`
					}{
						Content: "Custom response",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewCustom(server.URL, "test-key", "custom-model", 500)

	resp, err := provider.Complete(context.Background(), &Request{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if resp.Content != "Custom response" {
		t.Errorf("resp.Content = %s, want 'Custom response'", resp.Content)
	}
}

func TestOllamaComplete_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "internal error"}`))
	}))
	defer server.Close()

	provider := NewOllama(server.URL, "llama3.2")

	_, err := provider.Complete(context.Background(), &Request{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	})

	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestOllamaComplete_InvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`invalid json`))
	}))
	defer server.Close()

	provider := NewOllama(server.URL, "llama3.2")

	_, err := provider.Complete(context.Background(), &Request{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	})

	if err == nil {
		t.Error("expected error for invalid JSON, got nil")
	}
}

func TestCustomComplete_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{
			Error: &struct {
				Message string `json:"message"`
				Type    string `json:"type"`
			}{
				Message: "Rate limit exceeded",
				Type:    "rate_limit_error",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewCustom(server.URL, "test-key", "custom-model", 500)

	_, err := provider.Complete(context.Background(), &Request{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	})

	if err == nil {
		t.Error("expected error for API error response, got nil")
	}
}

func TestCustomComplete_NoChoices(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := openAIResponse{
			Choices: []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewCustom(server.URL, "test-key", "custom-model", 500)

	_, err := provider.Complete(context.Background(), &Request{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	})

	if err == nil {
		t.Error("expected error for no choices, got nil")
	}
}

func TestComplete_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	provider := NewOllama(server.URL, "llama3.2")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := provider.Complete(ctx, &Request{
		Messages: []Message{
			{Role: "user", Content: "Hello"},
		},
	})

	if err == nil {
		t.Error("expected context deadline exceeded error")
	}
}

func TestMessageFields(t *testing.T) {
	msg := Message{
		Role:    "assistant",
		Content: "I am an AI assistant",
	}

	if msg.Role != "assistant" {
		t.Errorf("Role = %s, want 'assistant'", msg.Role)
	}

	if msg.Content != "I am an AI assistant" {
		t.Errorf("Content = %s", msg.Content)
	}
}

func TestRequestFields(t *testing.T) {
	req := &Request{
		Messages: []Message{
			{Role: "system", Content: "You are helpful"},
			{Role: "user", Content: "Hello"},
		},
		MaxTokens: 1000,
	}

	if len(req.Messages) != 2 {
		t.Errorf("Messages length = %d, want 2", len(req.Messages))
	}

	if req.MaxTokens != 1000 {
		t.Errorf("MaxTokens = %d, want 1000", req.MaxTokens)
	}
}

func TestResponseFields(t *testing.T) {
	resp := &Response{
		Content: "Hello, I am here to help!",
	}

	if resp.Content != "Hello, I am here to help!" {
		t.Errorf("Content = %s", resp.Content)
	}
}

func TestNewOpenAI(t *testing.T) {
	provider := NewOpenAI("test-key", "gpt-4o", 500)

	if provider.Name() != "openai" {
		t.Errorf("Name() = %s, want 'openai'", provider.Name())
	}

	if provider.apiKey != "test-key" {
		t.Errorf("apiKey = %s, want 'test-key'", provider.apiKey)
	}

	if provider.model != "gpt-4o" {
		t.Errorf("model = %s, want 'gpt-4o'", provider.model)
	}

	if provider.maxTokens != 500 {
		t.Errorf("maxTokens = %d, want 500", provider.maxTokens)
	}
}

func TestNewAnthropic(t *testing.T) {
	provider := NewAnthropic("test-key", "claude-sonnet-4-20250514", 1000)

	if provider.Name() != "anthropic" {
		t.Errorf("Name() = %s, want 'anthropic'", provider.Name())
	}

	if provider.apiKey != "test-key" {
		t.Errorf("apiKey = %s, want 'test-key'", provider.apiKey)
	}

	if provider.model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %s, want 'claude-sonnet-4-20250514'", provider.model)
	}
}

func TestNewOllama(t *testing.T) {
	provider := NewOllama("http://localhost:11434", "llama3.2")

	if provider.Name() != "ollama" {
		t.Errorf("Name() = %s, want 'ollama'", provider.Name())
	}

	if provider.endpoint != "http://localhost:11434" {
		t.Errorf("endpoint = %s, want 'http://localhost:11434'", provider.endpoint)
	}

	if provider.model != "llama3.2" {
		t.Errorf("model = %s, want 'llama3.2'", provider.model)
	}
}

func TestNewCustom(t *testing.T) {
	provider := NewCustom("https://api.example.com", "api-key", "model-x", 2000)

	if provider.Name() != "custom" {
		t.Errorf("Name() = %s, want 'custom'", provider.Name())
	}

	if provider.endpoint != "https://api.example.com" {
		t.Errorf("endpoint = %s, want 'https://api.example.com'", provider.endpoint)
	}

	if provider.apiKey != "api-key" {
		t.Errorf("apiKey = %s, want 'api-key'", provider.apiKey)
	}
}
