# Package Reorganization Proposal

## Overview

This proposal outlines a reorganization of the `langfuse-go` SDK following Go best practices from the [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments), [Effective Go](https://go.dev/doc/effective_go), and community conventions established by projects like the AWS SDK for Go v2, Google Cloud Go, and Stripe Go.

## Current Structure Analysis

### Current Layout
```
langfuse-go/
├── client.go           (638 lines) - Client, TraceBuilder, TraceContext
├── config.go           (348 lines) - Config, ConfigOption
├── datasets.go         (220 lines) - DatasetsClient
├── errors.go           (107 lines) - Error types
├── evaluation.go       (205 lines) - Evaluation input/output types
├── evaluation_builders.go (719 lines) - RAG/QA/etc. trace builders
├── http.go             (315 lines) - HTTP client
├── ingestion.go        (979 lines) - Span/Generation/Event builders
├── models.go           (83 lines)  - ModelsClient
├── observations.go     (96 lines)  - ObservationsClient
├── prompts.go          (187 lines) - PromptsClient
├── retry.go            (196 lines) - Retry logic
├── scores.go           (141 lines) - ScoresClient
├── sessions.go         (86 lines)  - SessionsClient
├── testing.go          (281 lines) - Test utilities
├── traces.go           (54 lines)  - TracesClient
├── types.go            (360 lines) - Common types
├── validation.go       (372 lines) - Validation utilities
├── version.go          (5 lines)   - Version constant
├── cmd/langfuse-hooks/ - CLI tool
├── examples/           - Usage examples
├── internal/hooks/     - Internal CLI packages
└── otel/               - OpenTelemetry bridge
```

### Identified Issues

1. **Flat package with 15+ files** - All core functionality is in the root package, making it harder to navigate and maintain.

2. **Large files** - `ingestion.go` (979 lines) and `evaluation_builders.go` (719 lines) are too large for comfortable reading and maintenance.

3. **Mixed concerns** - `client.go` contains both the client lifecycle AND trace builders (638 lines).

4. **Internal types exposed** - HTTP client and ingestion event types are in the public API but are implementation details.

5. **Test utilities in main package** - `testing.go` with `MockClient` pollutes the main package.

6. **Inconsistent sub-client pattern** - Some clients return builders (`NewTrace()`), others make direct API calls (`Datasets().List()`).

---

## Proposed Structure

Following Go conventions:
- Keep the root package small and focused on the main client
- Use internal packages for implementation details
- Create subpackages only when there's a clear domain boundary
- Put test utilities in a separate `langfusetest` package

### Proposed Layout

```
langfuse-go/
├── doc.go                    # Package documentation
├── client.go                 # Client struct, New(), Shutdown(), Flush()
├── config.go                 # Config, ConfigOption, Region constants
├── errors.go                 # Public error types and sentinels
├── types.go                  # Public shared types (Time, Usage, etc.)
├── version.go                # Version constant
│
├── trace.go                  # TraceBuilder, TraceContext, TraceUpdateBuilder
├── span.go                   # SpanBuilder, SpanContext
├── generation.go             # GenerationBuilder, GenerationContext
├── event.go                  # EventBuilder
├── score.go                  # ScoreBuilder
│
├── evaluation/               # Evaluation-ready tracing (subpackage)
│   ├── doc.go                # Package documentation
│   ├── types.go              # RAGInput, RAGOutput, QAInput, etc.
│   ├── validation.go         # ValidateForEvaluator, EvaluatorRequirements
│   ├── rag.go                # RAGTraceBuilder, RAGTraceContext
│   ├── qa.go                 # QATraceBuilder, QATraceContext
│   ├── summarization.go      # SummarizationTraceBuilder
│   └── classification.go     # ClassificationTraceBuilder
│
├── api/                      # API resource clients (subpackage)
│   ├── traces.go             # TracesClient (List, Get)
│   ├── observations.go       # ObservationsClient
│   ├── scores.go             # ScoresClient
│   ├── prompts.go            # PromptsClient
│   ├── datasets.go           # DatasetsClient
│   ├── sessions.go           # SessionsClient
│   └── models.go             # ModelsClient
│
├── internal/
│   ├── http/                 # HTTP client implementation
│   │   ├── client.go         # httpClient, request handling
│   │   └── retry.go          # Retry logic with exponential backoff
│   │
│   ├── ingestion/            # Ingestion event types
│   │   ├── types.go          # ingestionEvent, createTraceEvent, etc.
│   │   └── batch.go          # Batch processing logic
│   │
│   └── hooks/                # (existing) CLI tool internals
│       ├── config/
│       ├── git/
│       ├── prompt/
│       └── provider/
│
├── langfusetest/             # Test utilities (separate package)
│   ├── mock.go               # MockClient, MockServer
│   └── helpers.go            # Test helper functions
│
├── otel/                     # (existing) OpenTelemetry bridge
│   ├── bridge.go
│   ├── propagation.go
│   └── types.go
│
├── cmd/langfuse-hooks/       # (existing) CLI tool
│
└── examples/                 # (existing) Usage examples
```

---

## Detailed Changes

### 1. Split `client.go` (638 lines)

**Current:** Contains Client, TraceBuilder, TraceContext, TraceUpdateBuilder

**Proposed:**
- `client.go` (~200 lines): Client struct, New(), Shutdown(), Flush(), sub-client accessors
- `trace.go` (~200 lines): TraceBuilder, TraceContext, TraceUpdateBuilder
- Move background goroutine management to internal/ingestion

### 2. Split `ingestion.go` (979 lines)

**Current:** Contains all observation builders (Span, Generation, Event, Score) plus ingestion types

**Proposed:**
- `span.go` (~200 lines): SpanBuilder, SpanContext, SpanUpdateBuilder
- `generation.go` (~250 lines): GenerationBuilder, GenerationContext, GenerationUpdateBuilder
- `event.go` (~100 lines): EventBuilder
- `score.go` (~100 lines): ScoreBuilder
- `internal/ingestion/types.go`: ingestionEvent, event type constants
- `internal/ingestion/batch.go`: Batch queue management

### 3. Create `evaluation/` Subpackage

**Rationale:** Evaluation-ready tracing is a distinct feature set that:
- Has its own domain vocabulary (RAG, QA, Summarization, Classification)
- Is optional for users who don't need Langfuse evaluators
- Has 1300+ lines of code that can be independently versioned

**Usage change:**
```go
// Before
trace, _ := client.NewRAGTrace("name").Query("...").Create()
langfuse.ValidateForEvaluator(input, output, langfuse.RAGEvaluator)

// After
import "github.com/jdziat/langfuse-go/evaluation"

trace, _ := evaluation.NewRAGTrace(client, "name").Query("...").Create()
evaluation.ValidateFor(input, output, evaluation.RAGEvaluator)
```

### 4. Create `api/` Subpackage

**Rationale:** API resource clients (Traces, Datasets, Prompts, etc.) are:
- Read-oriented (GET/LIST operations)
- Distinct from the builder pattern used for tracing
- Optional for users who only need ingestion

**Usage change:**
```go
// Before
traces, _ := client.Traces().List(ctx, params)
datasets, _ := client.Datasets().List(ctx, params)

// After (Option A: Subpackage)
import "github.com/jdziat/langfuse-go/api"
traces, _ := api.Traces(client).List(ctx, params)

// After (Option B: Keep on client, just reorganize files)
traces, _ := client.Traces().List(ctx, params)  // unchanged
```

**Recommendation:** Option B - keep the API unchanged but reorganize files into `api/` for internal organization. The `Client` struct can embed or delegate to these.

### 5. Move HTTP/Retry to `internal/`

**Current:** `http.go` and `retry.go` expose implementation details

**Proposed:** Move to `internal/http/`
- Users don't need to interact with the HTTP layer directly
- Retry configuration stays in public `Config` struct
- Internal implementation is hidden

### 6. Create `langfusetest/` Package

**Current:** `testing.go` in main package pollutes the API

**Proposed:** Separate `langfusetest` package following the pattern of:
- `net/http/httptest`
- `database/sql/sqltest`
- AWS SDK's `smithyhttp` test utilities

**Usage:**
```go
import "github.com/jdziat/langfuse-go/langfusetest"

func TestMyApp(t *testing.T) {
    client := langfusetest.NewMockClient(t)
    // or
    server := langfusetest.NewServer(t)
    client, _ := langfuse.New(key, secret, langfuse.WithBaseURL(server.URL))
}
```

---

## Migration Strategy

### Phase 1: Internal Reorganization (Non-breaking)

1. Move HTTP client to `internal/http/`
2. Move ingestion types to `internal/ingestion/`
3. Split large files (`ingestion.go`, `client.go`)
4. Create `langfusetest/` package
5. Keep all public APIs unchanged via re-exports

### Phase 2: Subpackage Creation (Minor version)

1. Create `evaluation/` subpackage
2. Create `api/` subpackage (if pursuing Option A)
3. Add deprecation notices to root package types
4. Provide migration guide

### Phase 3: Cleanup (Major version)

1. Remove deprecated re-exports from root package
2. Update all examples and documentation

---

## File Size Guidelines

Following Go community conventions:
- Target 200-400 lines per file
- Never exceed 800 lines (current `ingestion.go` at 979 is too large)
- Group related types/functions together
- One major type per file when the type has many methods

---

## Package Documentation

Add `doc.go` files with package-level documentation:

```go
// Package langfuse provides a Go SDK for Langfuse LLM observability platform.
//
// Basic usage:
//
//     client, _ := langfuse.New(publicKey, secretKey)
//     defer client.Shutdown(ctx)
//
//     trace, _ := client.NewTrace().Name("my-trace").Create()
//     gen, _ := trace.Generation().Model("gpt-4").Create()
//     gen.End()
//
// For evaluation-ready tracing, see the evaluation subpackage.
// For direct API access (list traces, manage datasets), see the api subpackage.
package langfuse
```

---

## Import Path Conventions

Following Go best practices:
- Main package: `github.com/jdziat/langfuse-go`
- Evaluation: `github.com/jdziat/langfuse-go/evaluation`
- Test utilities: `github.com/jdziat/langfuse-go/langfusetest`
- OpenTelemetry: `github.com/jdziat/langfuse-go/otel`

Avoid deep nesting like `github.com/jdziat/langfuse-go/pkg/core/trace` - this is an anti-pattern in Go.

---

## Compatibility Considerations

### Backward Compatibility

- Phase 1 is fully backward compatible
- Phase 2 uses deprecation warnings
- Major version bump only for Phase 3

### Go Version Support

- Maintain Go 1.21+ support (current: 1.23)
- Use generics sparingly and only where they improve API clarity

---

## Benefits

1. **Discoverability** - Clearer package boundaries help users find what they need
2. **Maintainability** - Smaller files are easier to review and modify
3. **Testability** - Internal packages can be tested independently
4. **Flexibility** - Users import only what they need
5. **Documentation** - Package docs can focus on specific domains

---

## References

- [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [Effective Go](https://go.dev/doc/effective_go)
- [Standard Go Project Layout](https://github.com/golang-standards/project-layout)
- [AWS SDK for Go v2 Structure](https://github.com/aws/aws-sdk-go-v2)
- [Google Cloud Go Structure](https://github.com/googleapis/google-cloud-go)
