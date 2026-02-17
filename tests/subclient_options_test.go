package langfuse_test

// Note: Most tests from the original subclient_options_test.go have been removed
// because they test unexported types and functions:
// - promptsConfig, tracesConfig, datasetsConfig, scoresConfig, sessionsConfig, modelsConfig
// - applyDefaults, getCacheKey, addToCache, getFromCache
// - ConfiguredPromptsClient internal cache implementation
//
// These internal implementation details cannot be tested from external test packages.
// The remaining tests focus on the public API.
//
// If you need to test internal behavior, consider:
// 1. Keeping internal tests in the langfuse package
// 2. Exposing test hooks via internal packages
// 3. Testing behavior through the public API instead

import (
	"testing"
	"time"

	"github.com/jdziat/langfuse-go"
)

// TestPromptsOptionTypes verifies that prompts options are valid function types.
func TestPromptsOptionTypes(t *testing.T) {
	// These should compile without errors - verifying option functions exist
	var _ langfuse.PromptsOption = langfuse.WithDefaultLabel("production")
	var _ langfuse.PromptsOption = langfuse.WithDefaultVersion(5)
	var _ langfuse.PromptsOption = langfuse.WithPromptCaching(5 * time.Minute)
}

// TestTracesOptionTypes verifies that traces options are valid function types.
func TestTracesOptionTypes(t *testing.T) {
	// These should compile without errors - verifying option functions exist
	var _ langfuse.TracesOption = langfuse.WithDefaultMetadata(langfuse.Metadata{"env": "prod"})
	var _ langfuse.TracesOption = langfuse.WithDefaultTags([]string{"production", "v1"})
}

// TestDatasetsOptionTypes verifies that datasets options are valid function types.
func TestDatasetsOptionTypes(t *testing.T) {
	// These should compile without errors - verifying option functions exist
	var _ langfuse.DatasetsOption = langfuse.WithDefaultPageSize(100)
}

// TestScoresOptionTypes verifies that scores options are valid function types.
func TestScoresOptionTypes(t *testing.T) {
	// These should compile without errors - verifying option functions exist
	var _ langfuse.ScoresOption = langfuse.WithDefaultSource("evaluation-pipeline")
}

// TestSessionsOptionTypes verifies that sessions options are valid function types.
func TestSessionsOptionTypes(t *testing.T) {
	// These should compile without errors - verifying option functions exist
	var _ langfuse.SessionsOption = langfuse.WithSessionsTimeout(10 * time.Second)
}

// TestModelsOptionTypes verifies that models options are valid function types.
func TestModelsOptionTypes(t *testing.T) {
	// These should compile without errors - verifying option functions exist
	var _ langfuse.ModelsOption = langfuse.WithModelsTimeout(10 * time.Second)
}

// TestConfiguredClientTypes verifies that configured client types exist.
func TestConfiguredClientTypes(t *testing.T) {
	// These type assertions verify the types are exported
	// Note: We can't instantiate them without a real client
	t.Run("ConfiguredPromptsClient", func(t *testing.T) {
		// Type exists - verified by compiler
		var _ *langfuse.ConfiguredPromptsClient
	})

	t.Run("ConfiguredTracesClient", func(t *testing.T) {
		// Type exists - verified by compiler
		var _ *langfuse.ConfiguredTracesClient
	})

	t.Run("ConfiguredDatasetsClient", func(t *testing.T) {
		// Type exists - verified by compiler
		var _ *langfuse.ConfiguredDatasetsClient
	})

	t.Run("ConfiguredScoresClient", func(t *testing.T) {
		// Type exists - verified by compiler
		var _ *langfuse.ConfiguredScoresClient
	})

	t.Run("ConfiguredModelsClient", func(t *testing.T) {
		// Type exists - verified by compiler
		var _ *langfuse.ConfiguredModelsClient
	})

	t.Run("ConfiguredSessionsClient", func(t *testing.T) {
		// Type exists - verified by compiler
		var _ *langfuse.ConfiguredSessionsClient
	})
}
