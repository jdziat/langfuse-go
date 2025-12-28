package config

import (
	"os"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Provider != ProviderOpenAI {
		t.Errorf("expected default provider to be openai, got %s", cfg.Provider)
	}

	if cfg.OpenAI.Model != "gpt-4o" {
		t.Errorf("expected default OpenAI model to be gpt-4o, got %s", cfg.OpenAI.Model)
	}

	if cfg.OpenAI.MaxTokens != 500 {
		t.Errorf("expected default max tokens to be 500, got %d", cfg.OpenAI.MaxTokens)
	}

	if !cfg.Hooks.PrepareCommitMsg.Enabled {
		t.Error("expected prepare-commit-msg hook to be enabled by default")
	}

	if !cfg.Hooks.CommitMsg.Enabled {
		t.Error("expected commit-msg hook to be enabled by default")
	}

	if !cfg.Hooks.BranchSuggest.Enabled {
		t.Error("expected branch-suggest to be enabled by default")
	}

	if cfg.Hooks.BranchSuggest.MaxLength != 50 {
		t.Errorf("expected branch max length to be 50, got %d", cfg.Hooks.BranchSuggest.MaxLength)
	}
}

func TestExpandEnvVar(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		envKey   string
		envValue string
		expected string
	}{
		{
			name:     "dollar brace syntax",
			input:    "${TEST_VAR}",
			envKey:   "TEST_VAR",
			envValue: "test-value",
			expected: "test-value",
		},
		{
			name:     "dollar syntax",
			input:    "$TEST_VAR",
			envKey:   "TEST_VAR",
			envValue: "test-value",
			expected: "test-value",
		},
		{
			name:     "empty input",
			input:    "",
			envKey:   "TEST_VAR",
			envValue: "test-value",
			expected: "",
		},
		{
			name:     "no env var",
			input:    "plain-text",
			envKey:   "TEST_VAR",
			envValue: "test-value",
			expected: "plain-text",
		},
		{
			name:     "mixed content",
			input:    "prefix-${TEST_VAR}-suffix",
			envKey:   "TEST_VAR",
			envValue: "middle",
			expected: "prefix-middle-suffix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv(tt.envKey, tt.envValue)
			defer os.Unsetenv(tt.envKey)

			result := expandEnvVar(tt.input)
			if result != tt.expected {
				t.Errorf("expandEnvVar(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsDisabled(t *testing.T) {
	tests := []struct {
		name     string
		envValue string
		expected bool
	}{
		{"true string", "true", true},
		{"1 string", "1", true},
		{"false string", "false", false},
		{"0 string", "0", false},
		{"empty string", "", false},
		{"other value", "yes", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				os.Setenv("LANGFUSE_HOOKS_DISABLED", tt.envValue)
				defer os.Unsetenv("LANGFUSE_HOOKS_DISABLED")
			} else {
				os.Unsetenv("LANGFUSE_HOOKS_DISABLED")
			}

			result := IsDisabled()
			if result != tt.expected {
				t.Errorf("IsDisabled() = %v, want %v (env=%q)", result, tt.expected, tt.envValue)
			}
		})
	}
}

func TestGetAPIKey(t *testing.T) {
	cfg := DefaultConfig()

	// Test with config value
	cfg.OpenAI.APIKey = "config-key"
	if key := cfg.GetAPIKey(); key != "config-key" {
		t.Errorf("expected config-key, got %s", key)
	}

	// Test with env fallback
	cfg.OpenAI.APIKey = ""
	os.Setenv("OPENAI_API_KEY", "env-key")
	defer os.Unsetenv("OPENAI_API_KEY")

	if key := cfg.GetAPIKey(); key != "env-key" {
		t.Errorf("expected env-key, got %s", key)
	}

	// Test Anthropic
	cfg.Provider = ProviderAnthropic
	cfg.Anthropic.APIKey = "anthropic-key"
	if key := cfg.GetAPIKey(); key != "anthropic-key" {
		t.Errorf("expected anthropic-key, got %s", key)
	}

	// Test Ollama (no API key needed)
	cfg.Provider = ProviderOllama
	if key := cfg.GetAPIKey(); key != "" {
		t.Errorf("expected empty key for ollama, got %s", key)
	}
}

func TestGetModel(t *testing.T) {
	cfg := DefaultConfig()

	// OpenAI
	if model := cfg.GetModel(); model != "gpt-4o" {
		t.Errorf("expected gpt-4o, got %s", model)
	}

	// Anthropic
	cfg.Provider = ProviderAnthropic
	if model := cfg.GetModel(); model != "claude-sonnet-4-20250514" {
		t.Errorf("expected claude-sonnet-4-20250514, got %s", model)
	}

	// Ollama
	cfg.Provider = ProviderOllama
	if model := cfg.GetModel(); model != "llama3.2" {
		t.Errorf("expected llama3.2, got %s", model)
	}
}
