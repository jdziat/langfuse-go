package langfuse

import (
	"testing"
	"time"
)

func TestPromptsOptions(t *testing.T) {
	t.Run("WithDefaultLabel", func(t *testing.T) {
		cfg := &promptsConfig{}
		opt := WithDefaultLabel("production")
		opt(cfg)

		if cfg.defaultLabel != "production" {
			t.Errorf("defaultLabel = %q, want %q", cfg.defaultLabel, "production")
		}
	})

	t.Run("WithDefaultVersion", func(t *testing.T) {
		cfg := &promptsConfig{}
		opt := WithDefaultVersion(5)
		opt(cfg)

		if cfg.defaultVersion != 5 {
			t.Errorf("defaultVersion = %d, want %d", cfg.defaultVersion, 5)
		}
	})

	t.Run("WithPromptCaching", func(t *testing.T) {
		cfg := &promptsConfig{}
		opt := WithPromptCaching(5 * time.Minute)
		opt(cfg)

		if !cfg.cacheEnabled {
			t.Error("cacheEnabled should be true")
		}
		if cfg.cacheTTL != 5*time.Minute {
			t.Errorf("cacheTTL = %v, want %v", cfg.cacheTTL, 5*time.Minute)
		}
	})
}

func TestTracesOptions(t *testing.T) {
	t.Run("WithDefaultMetadata", func(t *testing.T) {
		cfg := &tracesConfig{}
		opt := WithDefaultMetadata(Metadata{"env": "prod"})
		opt(cfg)

		if cfg.defaultMetadata["env"] != "prod" {
			t.Errorf("defaultMetadata[env] = %v, want %v", cfg.defaultMetadata["env"], "prod")
		}
	})

	t.Run("WithDefaultTags", func(t *testing.T) {
		cfg := &tracesConfig{}
		opt := WithDefaultTags([]string{"production", "v1"})
		opt(cfg)

		if len(cfg.defaultTags) != 2 {
			t.Errorf("defaultTags length = %d, want %d", len(cfg.defaultTags), 2)
		}
		if cfg.defaultTags[0] != "production" {
			t.Errorf("defaultTags[0] = %q, want %q", cfg.defaultTags[0], "production")
		}
	})
}

func TestDatasetsOptions(t *testing.T) {
	t.Run("WithDefaultPageSize", func(t *testing.T) {
		cfg := &datasetsConfig{}
		opt := WithDefaultPageSize(100)
		opt(cfg)

		if cfg.defaultPageSize != 100 {
			t.Errorf("defaultPageSize = %d, want %d", cfg.defaultPageSize, 100)
		}
	})
}

func TestScoresOptions(t *testing.T) {
	t.Run("WithDefaultSource", func(t *testing.T) {
		cfg := &scoresConfig{}
		opt := WithDefaultSource("evaluation-pipeline")
		opt(cfg)

		if cfg.defaultSource != "evaluation-pipeline" {
			t.Errorf("defaultSource = %q, want %q", cfg.defaultSource, "evaluation-pipeline")
		}
	})
}

func TestConfiguredPromptsClient_ApplyDefaults(t *testing.T) {
	client := &ConfiguredPromptsClient{
		config: &promptsConfig{
			defaultLabel:   "production",
			defaultVersion: 3,
		},
	}

	t.Run("applies defaults when params is nil", func(t *testing.T) {
		params := client.applyDefaults(nil)
		if params.Label != "production" {
			t.Errorf("Label = %q, want %q", params.Label, "production")
		}
		if params.Version != 3 {
			t.Errorf("Version = %d, want %d", params.Version, 3)
		}
	})

	t.Run("applies defaults when params has empty values", func(t *testing.T) {
		params := client.applyDefaults(&GetPromptParams{})
		if params.Label != "production" {
			t.Errorf("Label = %q, want %q", params.Label, "production")
		}
		if params.Version != 3 {
			t.Errorf("Version = %d, want %d", params.Version, 3)
		}
	})

	t.Run("does not override explicit values", func(t *testing.T) {
		params := client.applyDefaults(&GetPromptParams{
			Label:   "staging",
			Version: 5,
		})
		if params.Label != "staging" {
			t.Errorf("Label = %q, want %q", params.Label, "staging")
		}
		if params.Version != 5 {
			t.Errorf("Version = %d, want %d", params.Version, 5)
		}
	})
}

func TestConfiguredPromptsClient_CacheKey(t *testing.T) {
	client := &ConfiguredPromptsClient{
		config: &promptsConfig{},
	}

	tests := []struct {
		name     string
		prompt   string
		params   *GetPromptParams
		expected string
	}{
		{
			name:     "name only",
			prompt:   "my-prompt",
			params:   nil,
			expected: "my-prompt",
		},
		{
			name:     "with label",
			prompt:   "my-prompt",
			params:   &GetPromptParams{Label: "production"},
			expected: "my-prompt:label=production",
		},
		{
			name:     "with version",
			prompt:   "my-prompt",
			params:   &GetPromptParams{Version: 5},
			expected: "my-prompt:version=5",
		},
		{
			name:     "with label and version",
			prompt:   "my-prompt",
			params:   &GetPromptParams{Label: "production", Version: 5},
			expected: "my-prompt:label=production:version=5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := client.getCacheKey(tt.prompt, tt.params)
			if key != tt.expected {
				t.Errorf("getCacheKey() = %q, want %q", key, tt.expected)
			}
		})
	}
}

func TestConfiguredPromptsClient_Cache(t *testing.T) {
	client := &ConfiguredPromptsClient{
		config: &promptsConfig{
			cacheEnabled: true,
			cacheTTL:     5 * time.Minute,
		},
	}

	t.Run("cache starts empty", func(t *testing.T) {
		if client.CacheSize() != 0 {
			t.Errorf("CacheSize() = %d, want 0", client.CacheSize())
		}
	})

	t.Run("add and get from cache", func(t *testing.T) {
		prompt := &Prompt{Name: "test-prompt"}
		params := &GetPromptParams{Label: "production"}

		client.addToCache("test-prompt", params, prompt)

		if client.CacheSize() != 1 {
			t.Errorf("CacheSize() = %d, want 1", client.CacheSize())
		}

		cached := client.getFromCache("test-prompt", params)
		if cached == nil {
			t.Error("getFromCache() returned nil, want prompt")
		} else if cached.Name != "test-prompt" {
			t.Errorf("cached.Name = %q, want %q", cached.Name, "test-prompt")
		}
	})

	t.Run("cache miss for different params", func(t *testing.T) {
		cached := client.getFromCache("test-prompt", &GetPromptParams{Label: "staging"})
		if cached != nil {
			t.Error("getFromCache() should return nil for different params")
		}
	})

	t.Run("clear cache", func(t *testing.T) {
		client.ClearCache()
		if client.CacheSize() != 0 {
			t.Errorf("CacheSize() after ClearCache() = %d, want 0", client.CacheSize())
		}
	})
}

func TestConfiguredPromptsClient_CacheExpiration(t *testing.T) {
	client := &ConfiguredPromptsClient{
		config: &promptsConfig{
			cacheEnabled: true,
			cacheTTL:     10 * time.Millisecond, // Very short TTL for testing
		},
	}

	prompt := &Prompt{Name: "test-prompt"}
	params := &GetPromptParams{Label: "production"}

	client.addToCache("test-prompt", params, prompt)

	// Should get from cache immediately
	cached := client.getFromCache("test-prompt", params)
	if cached == nil {
		t.Error("getFromCache() should return prompt before expiration")
	}

	// Wait for expiration
	time.Sleep(15 * time.Millisecond)

	// Should not get from cache after expiration
	cached = client.getFromCache("test-prompt", params)
	if cached != nil {
		t.Error("getFromCache() should return nil after expiration")
	}
}

func TestConfiguredPromptsClient_NoExpirationWithZeroTTL(t *testing.T) {
	client := &ConfiguredPromptsClient{
		config: &promptsConfig{
			cacheEnabled: true,
			cacheTTL:     0, // No expiration
		},
	}

	prompt := &Prompt{Name: "test-prompt"}
	params := &GetPromptParams{Label: "production"}

	client.addToCache("test-prompt", params, prompt)

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Should still get from cache with TTL of 0
	cached := client.getFromCache("test-prompt", params)
	if cached == nil {
		t.Error("getFromCache() should return prompt with TTL of 0 (no expiration)")
	}
}

func TestConfiguredTracesClient(t *testing.T) {
	client := &ConfiguredTracesClient{
		config: &tracesConfig{
			defaultMetadata: Metadata{"env": "prod"},
			defaultTags:     []string{"production"},
		},
	}

	t.Run("DefaultMetadata", func(t *testing.T) {
		metadata := client.DefaultMetadata()
		if metadata["env"] != "prod" {
			t.Errorf("DefaultMetadata()[env] = %v, want %v", metadata["env"], "prod")
		}
	})

	t.Run("DefaultTags", func(t *testing.T) {
		tags := client.DefaultTags()
		if len(tags) != 1 || tags[0] != "production" {
			t.Errorf("DefaultTags() = %v, want [production]", tags)
		}
	})
}

func TestConfiguredDatasetsClient(t *testing.T) {
	client := &ConfiguredDatasetsClient{
		config: &datasetsConfig{
			defaultPageSize: 50,
		},
	}

	if client.DefaultPageSize() != 50 {
		t.Errorf("DefaultPageSize() = %d, want 50", client.DefaultPageSize())
	}
}

func TestConfiguredScoresClient(t *testing.T) {
	client := &ConfiguredScoresClient{
		config: &scoresConfig{
			defaultSource: "evaluation",
		},
	}

	if client.DefaultSource() != "evaluation" {
		t.Errorf("DefaultSource() = %q, want %q", client.DefaultSource(), "evaluation")
	}
}
