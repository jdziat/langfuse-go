package langfuse

import (
	"context"
	"strconv"
	"sync"
	"time"
)

// SubClientOption is a function type for configuring sub-clients.
// This is the base type that specific sub-client option types can extend.
type SubClientOption func(interface{})

// PromptsOption is a functional option for configuring the PromptsClient.
type PromptsOption func(*promptsConfig)

type promptsConfig struct {
	defaultLabel   string
	defaultVersion int
	cacheTTL       time.Duration
	cacheEnabled   bool
}

// WithDefaultLabel sets a default label for all prompt lookups.
// This is applied when no label is explicitly provided to Get methods.
//
// Example:
//
//	prompts := client.PromptsWithOptions(
//	    langfuse.WithDefaultLabel("production"),
//	)
//	// All Get calls will use "production" label unless overridden
//	prompt, _ := prompts.Get(ctx, "my-prompt", nil)
func WithDefaultLabel(label string) PromptsOption {
	return func(c *promptsConfig) {
		c.defaultLabel = label
	}
}

// WithDefaultVersion sets a default version for prompt lookups.
// This is applied when no version is explicitly provided.
//
// Example:
//
//	prompts := client.PromptsWithOptions(
//	    langfuse.WithDefaultVersion(2),
//	)
func WithDefaultVersion(version int) PromptsOption {
	return func(c *promptsConfig) {
		c.defaultVersion = version
	}
}

// WithPromptCaching enables prompt caching with the specified TTL.
// Cached prompts are stored in memory and reused for subsequent requests.
// Use TTL of 0 to cache indefinitely until client shutdown.
//
// Example:
//
//	prompts := client.PromptsWithOptions(
//	    langfuse.WithPromptCaching(5 * time.Minute),
//	)
func WithPromptCaching(ttl time.Duration) PromptsOption {
	return func(c *promptsConfig) {
		c.cacheEnabled = true
		c.cacheTTL = ttl
	}
}

// TracesOption is a functional option for configuring the TracesClient.
type TracesOption func(*tracesConfig)

type tracesConfig struct {
	defaultMetadata Metadata
	defaultTags     []string
}

// WithDefaultMetadata sets default metadata for all traces.
// This metadata is merged with any metadata provided to individual traces.
//
// Example:
//
//	traces := client.TracesWithOptions(
//	    langfuse.WithDefaultMetadata(langfuse.Metadata{
//	        "service": "my-service",
//	        "version": "1.0",
//	    }),
//	)
func WithDefaultMetadata(metadata Metadata) TracesOption {
	return func(c *tracesConfig) {
		c.defaultMetadata = metadata
	}
}

// WithDefaultTags sets default tags for all traces.
// These tags are appended to any tags provided to individual traces.
//
// Example:
//
//	traces := client.TracesWithOptions(
//	    langfuse.WithDefaultTags([]string{"production", "v1"}),
//	)
func WithDefaultTags(tags []string) TracesOption {
	return func(c *tracesConfig) {
		c.defaultTags = tags
	}
}

// DatasetsOption is a functional option for configuring the DatasetsClient.
type DatasetsOption func(*datasetsConfig)

type datasetsConfig struct {
	defaultPageSize int
}

// WithDefaultPageSize sets the default page size for list operations.
//
// Example:
//
//	datasets := client.DatasetsWithOptions(
//	    langfuse.WithDefaultPageSize(100),
//	)
func WithDefaultPageSize(size int) DatasetsOption {
	return func(c *datasetsConfig) {
		c.defaultPageSize = size
	}
}

// ScoresOption is a functional option for configuring the ScoresClient.
type ScoresOption func(*scoresConfig)

type scoresConfig struct {
	defaultSource string
}

// WithDefaultSource sets a default source for all scores.
//
// Example:
//
//	scores := client.ScoresWithOptions(
//	    langfuse.WithDefaultSource("evaluation-pipeline"),
//	)
func WithDefaultSource(source string) ScoresOption {
	return func(c *scoresConfig) {
		c.defaultSource = source
	}
}

// ConfiguredPromptsClient wraps PromptsClient with configured defaults.
type ConfiguredPromptsClient struct {
	*PromptsClient
	config *promptsConfig

	// Cache for prompts
	cacheMu sync.RWMutex
	cache   map[string]cachedPrompt
}

type cachedPrompt struct {
	prompt    *Prompt
	expiresAt time.Time
}

// Get retrieves a prompt by name, applying configured defaults.
// If caching is enabled, cached prompts are returned when available.
func (c *ConfiguredPromptsClient) Get(ctx context.Context, name string, params *GetPromptParams) (*Prompt, error) {
	// Apply defaults
	effectiveParams := c.applyDefaults(params)

	// Check cache if enabled
	if c.config.cacheEnabled {
		if prompt := c.getFromCache(name, effectiveParams); prompt != nil {
			return prompt, nil
		}
	}

	// Fetch from API
	prompt, err := c.PromptsClient.Get(ctx, name, effectiveParams)
	if err != nil {
		return nil, err
	}

	// Cache if enabled
	if c.config.cacheEnabled {
		c.addToCache(name, effectiveParams, prompt)
	}

	return prompt, nil
}

func (c *ConfiguredPromptsClient) applyDefaults(params *GetPromptParams) *GetPromptParams {
	if params == nil {
		params = &GetPromptParams{}
	}

	// Only apply defaults if not explicitly set
	if params.Label == "" && c.config.defaultLabel != "" {
		params.Label = c.config.defaultLabel
	}
	if params.Version == 0 && c.config.defaultVersion > 0 {
		params.Version = c.config.defaultVersion
	}

	return params
}

func (c *ConfiguredPromptsClient) getCacheKey(name string, params *GetPromptParams) string {
	key := name
	if params != nil {
		if params.Label != "" {
			key += ":label=" + params.Label
		}
		if params.Version > 0 {
			key += ":version=" + strconv.Itoa(params.Version)
		}
	}
	return key
}

func (c *ConfiguredPromptsClient) getFromCache(name string, params *GetPromptParams) *Prompt {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()

	if c.cache == nil {
		return nil
	}

	key := c.getCacheKey(name, params)
	cached, ok := c.cache[key]
	if !ok {
		return nil
	}

	// Check expiration (TTL of 0 means no expiration)
	if c.config.cacheTTL > 0 && time.Now().After(cached.expiresAt) {
		return nil
	}

	return cached.prompt
}

func (c *ConfiguredPromptsClient) addToCache(name string, params *GetPromptParams, prompt *Prompt) {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()

	if c.cache == nil {
		c.cache = make(map[string]cachedPrompt)
	}

	key := c.getCacheKey(name, params)
	expiresAt := time.Time{} // Zero time means no expiration
	if c.config.cacheTTL > 0 {
		expiresAt = time.Now().Add(c.config.cacheTTL)
	}

	c.cache[key] = cachedPrompt{
		prompt:    prompt,
		expiresAt: expiresAt,
	}
}

// ClearCache clears the prompt cache.
func (c *ConfiguredPromptsClient) ClearCache() {
	c.cacheMu.Lock()
	defer c.cacheMu.Unlock()
	c.cache = nil
}

// CacheSize returns the number of cached prompts.
func (c *ConfiguredPromptsClient) CacheSize() int {
	c.cacheMu.RLock()
	defer c.cacheMu.RUnlock()
	return len(c.cache)
}

// ConfiguredTracesClient wraps TracesClient with configured defaults.
type ConfiguredTracesClient struct {
	*TracesClient
	config *tracesConfig
}

// Get retrieves a trace by ID.
func (c *ConfiguredTracesClient) Get(ctx context.Context, id string) (*Trace, error) {
	return c.TracesClient.Get(ctx, id)
}

// DefaultMetadata returns the configured default metadata.
func (c *ConfiguredTracesClient) DefaultMetadata() Metadata {
	return c.config.defaultMetadata
}

// DefaultTags returns the configured default tags.
func (c *ConfiguredTracesClient) DefaultTags() []string {
	return c.config.defaultTags
}

// ConfiguredDatasetsClient wraps DatasetsClient with configured defaults.
type ConfiguredDatasetsClient struct {
	*DatasetsClient
	config *datasetsConfig
}

// DefaultPageSize returns the configured default page size.
func (c *ConfiguredDatasetsClient) DefaultPageSize() int {
	return c.config.defaultPageSize
}

// ConfiguredScoresClient wraps ScoresClient with configured defaults.
type ConfiguredScoresClient struct {
	*ScoresClient
	config *scoresConfig
}

// DefaultSource returns the configured default source.
func (c *ConfiguredScoresClient) DefaultSource() string {
	return c.config.defaultSource
}
