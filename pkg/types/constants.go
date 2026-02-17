package types

// Common environment constants.
// Use these with the Environment() builder methods for consistency.
const (
	EnvProduction  = "production"
	EnvDevelopment = "development"
	EnvStaging     = "staging"
	EnvTest        = "test"
)

// Common prompt label constants.
// Use these with GetByLabel() for consistency.
const (
	LabelProduction  = "production"
	LabelDevelopment = "development"
	LabelStaging     = "staging"
	LabelLatest      = "latest"
)

// Common model name constants.
// These are provided for convenience and discoverability.
const (
	// OpenAI models
	ModelGPT4          = "gpt-4"
	ModelGPT4Turbo     = "gpt-4-turbo"
	ModelGPT4o         = "gpt-4o"
	ModelGPT4oMini     = "gpt-4o-mini"
	ModelGPT35Turbo    = "gpt-3.5-turbo"
	ModelO1            = "o1"
	ModelO1Mini        = "o1-mini"
	ModelO1Preview     = "o1-preview"
	ModelO3Mini        = "o3-mini"
	ModelTextEmbedding = "text-embedding-3-small"

	// Anthropic models
	ModelClaude3Opus    = "claude-3-opus"
	ModelClaude3Sonnet  = "claude-3-sonnet"
	ModelClaude3Haiku   = "claude-3-haiku"
	ModelClaude35Sonnet = "claude-3.5-sonnet"
	ModelClaude35Haiku  = "claude-3.5-haiku"
	ModelClaude4Opus    = "claude-opus-4"
	ModelClaude4Sonnet  = "claude-sonnet-4"

	// Google models
	ModelGeminiPro     = "gemini-pro"
	ModelGemini15Pro   = "gemini-1.5-pro"
	ModelGemini15Flash = "gemini-1.5-flash"
	ModelGemini20Flash = "gemini-2.0-flash"
)
