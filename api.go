package langfuse

// API version prefixes.
const (
	// apiV2 is the prefix for v2 API endpoints.
	apiV2 = "/v2"
)

// endpoints defines all API endpoint paths.
// Using a struct ensures type safety and enables IDE autocompletion.
var endpoints = struct {
	// Core endpoints (no version prefix)
	Health    string
	Ingestion string

	// Trace endpoints
	Traces string

	// Observation endpoints
	Observations string

	// Score endpoints
	Scores string

	// Session endpoints
	Sessions string

	// Model endpoints
	Models string

	// v2 API endpoints
	Prompts      string
	Datasets     string
	DatasetItems string
	DatasetRuns  string
}{
	// Core endpoints
	Health:    "/health",
	Ingestion: "/ingestion",

	// Resource endpoints
	Traces:       "/traces",
	Observations: "/observations",
	Scores:       "/scores",
	Sessions:     "/sessions",
	Models:       "/models",

	// v2 endpoints
	Prompts:      apiV2 + "/prompts",
	Datasets:     apiV2 + "/datasets",
	DatasetItems: "/dataset-items",
	DatasetRuns:  "/dataset-run-items",
}
