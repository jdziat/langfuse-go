package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOpenAIProvider(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("expected Authorization header to be 'Bearer test-key', got %s", r.Header.Get("Authorization"))
		}

		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type to be application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Return mock response
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
						Content: "test response",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Note: We can't easily test OpenAI with mock server without modifying the code
	// to accept custom endpoints. This test is more of a compilation check.

	provider := NewOpenAI("test-key", "gpt-4o", 500)

	if provider.Name() != "openai" {
		t.Errorf("expected name to be 'openai', got %s", provider.Name())
	}
}

func TestOllamaProvider(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/chat" {
			t.Errorf("expected path to be /api/chat, got %s", r.URL.Path)
		}

		// Return mock response
		resp := ollamaResponse{
			Message: struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			}{
				Role:    "assistant",
				Content: "test response",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewOllama(server.URL, "llama3.2")

	if provider.Name() != "ollama" {
		t.Errorf("expected name to be 'ollama', got %s", provider.Name())
	}

	// Test completion
	resp, err := provider.Complete(context.Background(), &Request{
		Messages: []Message{
			{Role: "user", Content: "test"},
		},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if resp.Content != "test response" {
		t.Errorf("expected content to be 'test response', got %s", resp.Content)
	}
}

func TestAnthropicProvider(t *testing.T) {
	provider := NewAnthropic("test-key", "claude-sonnet-4-20250514", 500)

	if provider.Name() != "anthropic" {
		t.Errorf("expected name to be 'anthropic', got %s", provider.Name())
	}
}

func TestCustomProvider(t *testing.T) {
	// Create a mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer custom-key" {
			t.Errorf("expected Authorization header, got %s", r.Header.Get("Authorization"))
		}

		// Return mock response
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
						Content: "custom response",
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	provider := NewCustom(server.URL, "custom-key", "custom-model", 500)

	if provider.Name() != "custom" {
		t.Errorf("expected name to be 'custom', got %s", provider.Name())
	}

	// Test completion
	resp, err := provider.Complete(context.Background(), &Request{
		Messages: []Message{
			{Role: "user", Content: "test"},
		},
	})

	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}

	if resp.Content != "custom response" {
		t.Errorf("expected content to be 'custom response', got %s", resp.Content)
	}
}

func TestMessage(t *testing.T) {
	msg := Message{
		Role:    "user",
		Content: "test content",
	}

	if msg.Role != "user" {
		t.Errorf("expected role to be 'user', got %s", msg.Role)
	}

	if msg.Content != "test content" {
		t.Errorf("expected content to be 'test content', got %s", msg.Content)
	}
}

func TestRequest(t *testing.T) {
	req := Request{
		Messages: []Message{
			{Role: "user", Content: "test"},
		},
		MaxTokens: 500,
	}

	if len(req.Messages) != 1 {
		t.Errorf("expected 1 message, got %d", len(req.Messages))
	}

	if req.MaxTokens != 500 {
		t.Errorf("expected max tokens to be 500, got %d", req.MaxTokens)
	}
}
