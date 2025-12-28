package provider

import (
	"fmt"

	"github.com/jdziat/langfuse-go/internal/hooks/config"
)

// New creates a provider based on the configuration.
func New(cfg *config.Config) (Provider, error) {
	switch cfg.Provider {
	case config.ProviderOpenAI:
		apiKey := cfg.GetAPIKey()
		if apiKey == "" {
			return nil, fmt.Errorf("OpenAI API key not configured. Set OPENAI_API_KEY or configure in .langfuse-hooks.yaml")
		}
		return NewOpenAI(apiKey, cfg.GetModel(), cfg.GetMaxTokens()), nil

	case config.ProviderAnthropic:
		apiKey := cfg.GetAPIKey()
		if apiKey == "" {
			return nil, fmt.Errorf("Anthropic API key not configured. Set ANTHROPIC_API_KEY or configure in .langfuse-hooks.yaml")
		}
		return NewAnthropic(apiKey, cfg.GetModel(), cfg.GetMaxTokens()), nil

	case config.ProviderOllama:
		return NewOllama(cfg.Ollama.Endpoint, cfg.GetModel()), nil

	case config.ProviderCustom:
		if cfg.Custom.Endpoint == "" {
			return nil, fmt.Errorf("custom endpoint not configured")
		}
		// For custom endpoints, use OpenAI-compatible API
		return NewCustom(cfg.Custom.Endpoint, cfg.GetAPIKey(), cfg.GetModel(), cfg.GetMaxTokens()), nil

	default:
		return nil, fmt.Errorf("unknown provider: %s", cfg.Provider)
	}
}
