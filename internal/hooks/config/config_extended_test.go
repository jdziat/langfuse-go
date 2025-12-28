package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestApplyEnvOverrides(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		provider Provider
		check    func(*Config) bool
	}{
		{
			name: "override provider",
			envVars: map[string]string{
				"LANGFUSE_HOOKS_PROVIDER": "anthropic",
			},
			provider: ProviderOpenAI,
			check: func(c *Config) bool {
				return c.Provider == ProviderAnthropic
			},
		},
		{
			name: "override model for openai",
			envVars: map[string]string{
				"LANGFUSE_HOOKS_MODEL": "gpt-4-turbo",
			},
			provider: ProviderOpenAI,
			check: func(c *Config) bool {
				return c.OpenAI.Model == "gpt-4-turbo"
			},
		},
		{
			name: "override model for anthropic",
			envVars: map[string]string{
				"LANGFUSE_HOOKS_MODEL": "claude-opus-4-20250514",
			},
			provider: ProviderAnthropic,
			check: func(c *Config) bool {
				return c.Anthropic.Model == "claude-opus-4-20250514"
			},
		},
		{
			name: "override model for ollama",
			envVars: map[string]string{
				"LANGFUSE_HOOKS_MODEL": "mistral",
			},
			provider: ProviderOllama,
			check: func(c *Config) bool {
				return c.Ollama.Model == "mistral"
			},
		},
		{
			name: "override model for custom",
			envVars: map[string]string{
				"LANGFUSE_HOOKS_MODEL": "custom-model",
			},
			provider: ProviderCustom,
			check: func(c *Config) bool {
				return c.Custom.Model == "custom-model"
			},
		},
		{
			name: "override max tokens for openai",
			envVars: map[string]string{
				"LANGFUSE_HOOKS_MAX_TOKENS": "1000",
			},
			provider: ProviderOpenAI,
			check: func(c *Config) bool {
				return c.OpenAI.MaxTokens == 1000
			},
		},
		{
			name: "override max tokens for anthropic",
			envVars: map[string]string{
				"LANGFUSE_HOOKS_MAX_TOKENS": "2000",
			},
			provider: ProviderAnthropic,
			check: func(c *Config) bool {
				return c.Anthropic.MaxTokens == 2000
			},
		},
		{
			name: "override max tokens for custom",
			envVars: map[string]string{
				"LANGFUSE_HOOKS_MAX_TOKENS": "3000",
			},
			provider: ProviderCustom,
			check: func(c *Config) bool {
				return c.Custom.MaxTokens == 3000
			},
		},
		{
			name: "invalid max tokens ignored",
			envVars: map[string]string{
				"LANGFUSE_HOOKS_MAX_TOKENS": "invalid",
			},
			provider: ProviderOpenAI,
			check: func(c *Config) bool {
				return c.OpenAI.MaxTokens == 500 // Default
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			cfg := DefaultConfig()
			cfg.Provider = tt.provider
			applyEnvOverrides(cfg)

			if !tt.check(cfg) {
				t.Errorf("config check failed for %s", tt.name)
			}
		})
	}
}

func TestGetMaxTokens(t *testing.T) {
	tests := []struct {
		name     string
		provider Provider
		setup    func(*Config)
		expected int
	}{
		{
			name:     "openai max tokens",
			provider: ProviderOpenAI,
			setup: func(c *Config) {
				c.OpenAI.MaxTokens = 1000
			},
			expected: 1000,
		},
		{
			name:     "anthropic max tokens",
			provider: ProviderAnthropic,
			setup: func(c *Config) {
				c.Anthropic.MaxTokens = 2000
			},
			expected: 2000,
		},
		{
			name:     "custom max tokens",
			provider: ProviderCustom,
			setup: func(c *Config) {
				c.Custom.MaxTokens = 3000
			},
			expected: 3000,
		},
		{
			name:     "ollama returns default",
			provider: ProviderOllama,
			setup:    func(c *Config) {},
			expected: 500,
		},
		{
			name:     "unknown provider returns default",
			provider: "unknown",
			setup:    func(c *Config) {},
			expected: 500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Provider = tt.provider
			tt.setup(cfg)

			result := cfg.GetMaxTokens()
			if result != tt.expected {
				t.Errorf("GetMaxTokens() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestGetAPIKey_Anthropic(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = ProviderAnthropic

	// Test with empty config and env var
	os.Setenv("ANTHROPIC_API_KEY", "env-anthropic-key")
	defer os.Unsetenv("ANTHROPIC_API_KEY")

	if key := cfg.GetAPIKey(); key != "env-anthropic-key" {
		t.Errorf("GetAPIKey() = %s, want 'env-anthropic-key'", key)
	}
}

func TestGetAPIKey_Custom(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = ProviderCustom
	cfg.Custom.APIKey = "custom-api-key"

	if key := cfg.GetAPIKey(); key != "custom-api-key" {
		t.Errorf("GetAPIKey() = %s, want 'custom-api-key'", key)
	}
}

func TestLoadFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".langfuse-hooks.yaml")

	configContent := `
provider: anthropic
anthropic:
  model: claude-opus-4-20250514
  max_tokens: 1000
hooks:
  prepare-commit-msg:
    enabled: false
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := DefaultConfig()
	if err := loadFromFile(cfg, configPath); err != nil {
		t.Fatalf("loadFromFile() error = %v", err)
	}

	if cfg.Provider != ProviderAnthropic {
		t.Errorf("Provider = %s, want 'anthropic'", cfg.Provider)
	}

	if cfg.Anthropic.Model != "claude-opus-4-20250514" {
		t.Errorf("Anthropic.Model = %s, want 'claude-opus-4-20250514'", cfg.Anthropic.Model)
	}

	if cfg.Anthropic.MaxTokens != 1000 {
		t.Errorf("Anthropic.MaxTokens = %d, want 1000", cfg.Anthropic.MaxTokens)
	}

	if cfg.Hooks.PrepareCommitMsg.Enabled {
		t.Error("PrepareCommitMsg.Enabled should be false")
	}
}

func TestLoadFromFile_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, ".langfuse-hooks.yaml")

	invalidContent := `
provider: anthropic
  invalid: indentation
`

	if err := os.WriteFile(configPath, []byte(invalidContent), 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg := DefaultConfig()
	err := loadFromFile(cfg, configPath)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

func TestLoadFromFile_FileNotFound(t *testing.T) {
	cfg := DefaultConfig()
	err := loadFromFile(cfg, "/nonexistent/path/.langfuse-hooks.yaml")
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

func TestExpandEnvVars(t *testing.T) {
	os.Setenv("TEST_OPENAI_KEY", "openai-key-value")
	os.Setenv("TEST_ANTHROPIC_KEY", "anthropic-key-value")
	os.Setenv("TEST_CUSTOM_KEY", "custom-key-value")
	defer os.Unsetenv("TEST_OPENAI_KEY")
	defer os.Unsetenv("TEST_ANTHROPIC_KEY")
	defer os.Unsetenv("TEST_CUSTOM_KEY")

	cfg := DefaultConfig()
	cfg.OpenAI.APIKey = "${TEST_OPENAI_KEY}"
	cfg.Anthropic.APIKey = "$TEST_ANTHROPIC_KEY"
	cfg.Custom.APIKey = "${TEST_CUSTOM_KEY}"

	expandEnvVars(cfg)

	if cfg.OpenAI.APIKey != "openai-key-value" {
		t.Errorf("OpenAI.APIKey = %s, want 'openai-key-value'", cfg.OpenAI.APIKey)
	}

	if cfg.Anthropic.APIKey != "anthropic-key-value" {
		t.Errorf("Anthropic.APIKey = %s, want 'anthropic-key-value'", cfg.Anthropic.APIKey)
	}

	if cfg.Custom.APIKey != "custom-key-value" {
		t.Errorf("Custom.APIKey = %s, want 'custom-key-value'", cfg.Custom.APIKey)
	}
}

func TestProviderConstants(t *testing.T) {
	tests := []struct {
		provider Provider
		expected string
	}{
		{ProviderOpenAI, "openai"},
		{ProviderAnthropic, "anthropic"},
		{ProviderOllama, "ollama"},
		{ProviderCustom, "custom"},
	}

	for _, tt := range tests {
		if string(tt.provider) != tt.expected {
			t.Errorf("Provider %s = %s, want %s", tt.provider, string(tt.provider), tt.expected)
		}
	}
}

func TestHooksConfig_Defaults(t *testing.T) {
	cfg := DefaultConfig()

	// PrepareCommitMsg defaults
	if !cfg.Hooks.PrepareCommitMsg.Enabled {
		t.Error("PrepareCommitMsg.Enabled should be true by default")
	}
	if !cfg.Hooks.PrepareCommitMsg.Interactive {
		t.Error("PrepareCommitMsg.Interactive should be true by default")
	}
	if !cfg.Hooks.PrepareCommitMsg.IncludeDiff {
		t.Error("PrepareCommitMsg.IncludeDiff should be true by default")
	}
	if cfg.Hooks.PrepareCommitMsg.MaxDiffLines != 500 {
		t.Errorf("PrepareCommitMsg.MaxDiffLines = %d, want 500", cfg.Hooks.PrepareCommitMsg.MaxDiffLines)
	}

	// CommitMsg defaults
	if !cfg.Hooks.CommitMsg.Enabled {
		t.Error("CommitMsg.Enabled should be true by default")
	}
	if !cfg.Hooks.CommitMsg.ValidateFormat {
		t.Error("CommitMsg.ValidateFormat should be true by default")
	}
	if cfg.Hooks.CommitMsg.AutoFix {
		t.Error("CommitMsg.AutoFix should be false by default")
	}

	// BranchSuggest defaults
	if !cfg.Hooks.BranchSuggest.Enabled {
		t.Error("BranchSuggest.Enabled should be true by default")
	}
	if cfg.Hooks.BranchSuggest.Format != "{type}/{description}" {
		t.Errorf("BranchSuggest.Format = %s, want '{type}/{description}'", cfg.Hooks.BranchSuggest.Format)
	}
	if cfg.Hooks.BranchSuggest.MaxLength != 50 {
		t.Errorf("BranchSuggest.MaxLength = %d, want 50", cfg.Hooks.BranchSuggest.MaxLength)
	}
}

func TestContextConfig_Defaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Context.Conventions == "" {
		t.Error("Context.Conventions should not be empty by default")
	}

	if len(cfg.Context.IgnorePatterns) == 0 {
		t.Error("Context.IgnorePatterns should not be empty by default")
	}
}

func TestOllamaConfig_Defaults(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Ollama.Endpoint != "http://localhost:11434" {
		t.Errorf("Ollama.Endpoint = %s, want 'http://localhost:11434'", cfg.Ollama.Endpoint)
	}

	if cfg.Ollama.Model != "llama3.2" {
		t.Errorf("Ollama.Model = %s, want 'llama3.2'", cfg.Ollama.Model)
	}
}
