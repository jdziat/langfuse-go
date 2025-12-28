// Package config provides configuration loading for langfuse-hooks.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Provider represents the LLM provider type.
type Provider string

const (
	ProviderOpenAI    Provider = "openai"
	ProviderAnthropic Provider = "anthropic"
	ProviderOllama    Provider = "ollama"
	ProviderCustom    Provider = "custom"
)

// Config represents the complete hooks configuration.
type Config struct {
	Provider  Provider        `yaml:"provider"`
	OpenAI    OpenAIConfig    `yaml:"openai"`
	Anthropic AnthropicConfig `yaml:"anthropic"`
	Ollama    OllamaConfig    `yaml:"ollama"`
	Custom    CustomConfig    `yaml:"custom"`
	Hooks     HooksConfig     `yaml:"hooks"`
	Context   ContextConfig   `yaml:"context"`
}

// OpenAIConfig holds OpenAI-specific settings.
type OpenAIConfig struct {
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
}

// AnthropicConfig holds Anthropic-specific settings.
type AnthropicConfig struct {
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
}

// OllamaConfig holds Ollama-specific settings.
type OllamaConfig struct {
	Endpoint string `yaml:"endpoint"`
	Model    string `yaml:"model"`
}

// CustomConfig holds custom endpoint settings.
type CustomConfig struct {
	Endpoint  string `yaml:"endpoint"`
	APIKey    string `yaml:"api_key"`
	Model     string `yaml:"model"`
	MaxTokens int    `yaml:"max_tokens"`
}

// HooksConfig holds per-hook configuration.
type HooksConfig struct {
	PrepareCommitMsg PrepareCommitMsgConfig `yaml:"prepare-commit-msg"`
	CommitMsg        CommitMsgConfig        `yaml:"commit-msg"`
	BranchSuggest    BranchSuggestConfig    `yaml:"branch-suggest"`
}

// PrepareCommitMsgConfig configures the prepare-commit-msg hook.
type PrepareCommitMsgConfig struct {
	Enabled      bool `yaml:"enabled"`
	Interactive  bool `yaml:"interactive"`
	IncludeDiff  bool `yaml:"include_diff"`
	MaxDiffLines int  `yaml:"max_diff_lines"`
}

// CommitMsgConfig configures the commit-msg hook.
type CommitMsgConfig struct {
	Enabled        bool `yaml:"enabled"`
	ValidateFormat bool `yaml:"validate_format"`
	AutoFix        bool `yaml:"auto_fix"`
}

// BranchSuggestConfig configures branch name suggestions.
type BranchSuggestConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Format    string `yaml:"format"`
	MaxLength int    `yaml:"max_length"`
}

// ContextConfig provides project-specific context for prompts.
type ContextConfig struct {
	Conventions    string   `yaml:"conventions"`
	IgnorePatterns []string `yaml:"ignore_patterns"`
}

// DefaultConfig returns the default configuration.
func DefaultConfig() *Config {
	return &Config{
		Provider: ProviderOpenAI,
		OpenAI: OpenAIConfig{
			Model:     "gpt-4o",
			MaxTokens: 500,
		},
		Anthropic: AnthropicConfig{
			Model:     "claude-sonnet-4-20250514",
			MaxTokens: 500,
		},
		Ollama: OllamaConfig{
			Endpoint: "http://localhost:11434",
			Model:    "llama3.2",
		},
		Hooks: HooksConfig{
			PrepareCommitMsg: PrepareCommitMsgConfig{
				Enabled:      true,
				Interactive:  true,
				IncludeDiff:  true,
				MaxDiffLines: 500,
			},
			CommitMsg: CommitMsgConfig{
				Enabled:        true,
				ValidateFormat: true,
				AutoFix:        false,
			},
			BranchSuggest: BranchSuggestConfig{
				Enabled:   true,
				Format:    "{type}/{description}",
				MaxLength: 50,
			},
		},
		Context: ContextConfig{
			Conventions: `- Use conventional commits: feat, fix, docs, test, refactor, perf, chore
- Scope examples: client, ingestion, traces, scores, prompts
- Reference GitHub issues when applicable`,
			IgnorePatterns: []string{
				"*.generated.go",
				"vendor/*",
			},
		},
	}
}

// Load reads configuration from file and environment variables.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try to find config file
	configPath := findConfigFile()
	if configPath != "" {
		if err := loadFromFile(cfg, configPath); err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
	}

	// Apply environment variable overrides
	applyEnvOverrides(cfg)

	// Expand environment variables in API keys
	expandEnvVars(cfg)

	return cfg, nil
}

// findConfigFile searches for the configuration file.
func findConfigFile() string {
	candidates := []string{
		".langfuse-hooks.yaml",
		".langfuse-hooks.yml",
	}

	// Start from current directory and walk up
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		for _, name := range candidates {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

// loadFromFile reads configuration from a YAML file.
func loadFromFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, cfg)
}

// applyEnvOverrides applies environment variable overrides.
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("LANGFUSE_HOOKS_PROVIDER"); v != "" {
		cfg.Provider = Provider(v)
	}

	if v := os.Getenv("LANGFUSE_HOOKS_MODEL"); v != "" {
		switch cfg.Provider {
		case ProviderOpenAI:
			cfg.OpenAI.Model = v
		case ProviderAnthropic:
			cfg.Anthropic.Model = v
		case ProviderOllama:
			cfg.Ollama.Model = v
		case ProviderCustom:
			cfg.Custom.Model = v
		}
	}

	if v := os.Getenv("LANGFUSE_HOOKS_MAX_TOKENS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			switch cfg.Provider {
			case ProviderOpenAI:
				cfg.OpenAI.MaxTokens = n
			case ProviderAnthropic:
				cfg.Anthropic.MaxTokens = n
			case ProviderCustom:
				cfg.Custom.MaxTokens = n
			}
		}
	}
}

// expandEnvVars expands ${VAR} references in configuration values.
func expandEnvVars(cfg *Config) {
	cfg.OpenAI.APIKey = expandEnvVar(cfg.OpenAI.APIKey)
	cfg.Anthropic.APIKey = expandEnvVar(cfg.Anthropic.APIKey)
	cfg.Custom.APIKey = expandEnvVar(cfg.Custom.APIKey)
}

// expandEnvVar expands a single environment variable reference.
func expandEnvVar(s string) string {
	if s == "" {
		return s
	}

	// Match ${VAR} or $VAR patterns
	re := regexp.MustCompile(`\$\{?([A-Za-z_][A-Za-z0-9_]*)\}?`)
	return re.ReplaceAllStringFunc(s, func(match string) string {
		// Extract variable name
		name := strings.TrimPrefix(match, "${")
		name = strings.TrimPrefix(name, "$")
		name = strings.TrimSuffix(name, "}")
		return os.Getenv(name)
	})
}

// IsDisabled returns true if hooks are globally disabled.
func IsDisabled() bool {
	v := os.Getenv("LANGFUSE_HOOKS_DISABLED")
	return v == "true" || v == "1"
}

// GetAPIKey returns the API key for the configured provider.
func (c *Config) GetAPIKey() string {
	switch c.Provider {
	case ProviderOpenAI:
		if c.OpenAI.APIKey != "" {
			return c.OpenAI.APIKey
		}
		return os.Getenv("OPENAI_API_KEY")
	case ProviderAnthropic:
		if c.Anthropic.APIKey != "" {
			return c.Anthropic.APIKey
		}
		return os.Getenv("ANTHROPIC_API_KEY")
	case ProviderCustom:
		return c.Custom.APIKey
	default:
		return ""
	}
}

// GetModel returns the model for the configured provider.
func (c *Config) GetModel() string {
	switch c.Provider {
	case ProviderOpenAI:
		return c.OpenAI.Model
	case ProviderAnthropic:
		return c.Anthropic.Model
	case ProviderOllama:
		return c.Ollama.Model
	case ProviderCustom:
		return c.Custom.Model
	default:
		return ""
	}
}

// GetMaxTokens returns the max tokens for the configured provider.
func (c *Config) GetMaxTokens() int {
	switch c.Provider {
	case ProviderOpenAI:
		return c.OpenAI.MaxTokens
	case ProviderAnthropic:
		return c.Anthropic.MaxTokens
	case ProviderCustom:
		return c.Custom.MaxTokens
	default:
		return 500
	}
}
