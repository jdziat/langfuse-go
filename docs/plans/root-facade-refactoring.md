# Root Package Facade Refactoring Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reduce root package from 79 files to ~10 facade files by moving all implementation to `pkg/` packages.

**Architecture:** Root package becomes a thin facade that re-exports types and wraps implementations from `pkg/` subpackages. Users continue importing `github.com/jdziat/langfuse-go` but implementation lives in `pkg/` for extensibility.

**Tech Stack:** Pure Go, type aliases, interface-based delegation

---

## Current State

### Root Package (79 .go files)
```
Root Package Files by Category:
├── Core Client (10 files)
│   ├── client.go          # Main client struct and methods
│   ├── api.go             # API endpoint definitions
│   ├── facade.go          # High-level facade functions
│   ├── simple_api.go      # Simplified API
│   ├── v1_api.go          # V1 API compatibility
│   ├── interfaces.go      # Interface definitions
│   ├── helpers.go         # Helper functions
│   ├── logging.go         # Logging utilities
│   ├── doc.go             # Package documentation
│   └── version.go         # Version constant
│
├── Configuration (4 files)
│   ├── config.go          # Config struct, validation
│   ├── options.go         # ConfigOption functions
│   ├── regions.go         # Region constants
│   └── env.go             # Environment variable loading
│
├── HTTP/Network (3 files)
│   ├── http.go            # HTTP client implementation
│   ├── retry.go           # Retry strategies
│   └── circuitbreaker.go  # Circuit breaker
│
├── Errors (5 files)
│   ├── errors.go          # Base error types
│   ├── errors_api.go      # API error types
│   ├── errors_async.go    # Async error types
│   ├── errors_helpers.go  # Error helper functions
│   └── errors_validation.go # Validation errors
│
├── Batching/Ingestion (4 files)
│   ├── batching.go        # Batch processing
│   ├── ingestion.go       # Ingestion event types
│   ├── queue.go           # Queue management
│   └── backpressure.go    # Backpressure handling
│
├── Builders (3 files)
│   ├── builders.go        # Fluent builders
│   ├── validated_builder.go # Validated builder pattern
│   └── validation.go      # Validation helpers
│
├── Types (1 file)
│   └── types.go           # Data types (Trace, Observation, etc.)
│
├── Sub-client Facades (7 files)
│   ├── traces.go          # TracesClient facade
│   ├── observations.go    # ObservationsClient facade
│   ├── scores.go          # ScoresClient facade
│   ├── sessions.go        # SessionsClient facade
│   ├── models.go          # ModelsClient facade
│   ├── prompts.go         # PromptsClient facade
│   └── datasets.go        # DatasetsClient facade
│
├── Domain Builders (3 files)
│   ├── trace.go           # TraceBuilder
│   ├── span.go            # SpanBuilder
│   └── generation.go      # GenerationBuilder
│
├── Lifecycle/Metrics (4 files)
│   ├── lifecycle.go       # Lifecycle management
│   ├── metrics.go         # Metrics interface
│   ├── metrics_internal.go # Internal metrics
│   └── id.go              # ID generation
│
├── Sub-client Options (4 files)
│   ├── subclient_options.go # Base options
│   ├── models_options.go  # Models options
│   ├── sessions_options.go # Sessions options
│   └── unified_options.go # Unified options
│
├── Evaluation (6 files)
│   ├── eval_generation.go
│   ├── eval_span.go
│   ├── evaluation_input.go
│   ├── evaluation_metadata.go
│   ├── evaluation_mode.go
│   └── persistence.go
│
├── Hooks (1 file)
│   └── hooks.go           # HTTP hooks
│
└── Test Files (24 files)
    └── *_test.go          # Already in tests/ or staying in root
```

### Existing pkg/ Structure
```
pkg/
├── api/                   # ✅ Sub-client implementations (just added)
│   ├── traces/
│   ├── observations/
│   ├── scores/
│   ├── sessions/
│   ├── models/
│   ├── prompts/
│   └── datasets/
├── config/                # ✅ Partial - has types.go, env.go
├── errors/                # ✅ Has error types
├── http/                  # ✅ Has doer.go, pagination.go, retry.go, circuit.go
└── ingestion/             # ✅ Has events.go, backpressure.go, uuid.go
```

---

## Target Architecture

### Root Package (~10 files)
```
langfuse/
├── doc.go                 # Package documentation
├── version.go             # Version constant
├── client.go              # Client facade (re-exports pkg/client)
├── config.go              # Config facade (re-exports pkg/config)
├── errors.go              # Error facade (re-exports pkg/errors)
├── types.go               # Type aliases (re-exports pkg/types)
├── builders.go            # Builder facade (re-exports pkg/builders)
├── options.go             # Option facade (re-exports pkg/options)
├── subclients.go          # Sub-client facades (traces, observations, etc.)
└── evaluation.go          # Evaluation facade (re-exports pkg/evaluation)
```

### pkg/ Structure (Full Implementation)
```
pkg/
├── api/                   # ✅ Already done - API client implementations
│   ├── traces/
│   ├── observations/
│   ├── scores/
│   ├── sessions/
│   ├── models/
│   ├── prompts/
│   └── datasets/
│
├── client/                # 🆕 Core client implementation
│   ├── client.go          # Client struct and core methods
│   ├── lifecycle.go       # Lifecycle management
│   ├── batching.go        # Batch processing
│   ├── queue.go           # Queue management
│   └── hooks.go           # HTTP hooks
│
├── config/                # 🔄 Expand existing
│   ├── config.go          # Config struct
│   ├── options.go         # ConfigOption functions
│   ├── regions.go         # Region constants
│   ├── env.go             # Environment loading
│   └── defaults.go        # Default values
│
├── errors/                # 🔄 Consolidate existing
│   ├── errors.go          # Base error types
│   ├── api.go             # API errors
│   ├── async.go           # Async errors
│   ├── validation.go      # Validation errors
│   └── helpers.go         # Error helpers
│
├── http/                  # 🔄 Expand existing
│   ├── client.go          # HTTP client implementation
│   ├── doer.go            # Doer interface
│   ├── retry.go           # Retry strategies
│   ├── circuit.go         # Circuit breaker
│   └── pagination.go      # Pagination helpers
│
├── ingestion/             # 🔄 Expand existing
│   ├── events.go          # Event types
│   ├── batch.go           # Batch processing
│   ├── backpressure.go    # Backpressure handling
│   └── uuid.go            # UUID generation
│
├── types/                 # 🆕 Data types
│   ├── trace.go           # Trace type
│   ├── observation.go     # Observation type
│   ├── score.go           # Score type
│   ├── prompt.go          # Prompt type
│   ├── dataset.go         # Dataset types
│   ├── session.go         # Session type
│   ├── model.go           # Model type
│   └── metadata.go        # Metadata type
│
├── builders/              # 🆕 Builder implementations
│   ├── trace.go           # TraceBuilder
│   ├── span.go            # SpanBuilder
│   ├── generation.go      # GenerationBuilder
│   ├── score.go           # ScoreBuilder
│   ├── event.go           # EventBuilder
│   └── validation.go      # Validation helpers
│
├── options/               # 🆕 Sub-client options
│   ├── traces.go          # Traces options
│   ├── prompts.go         # Prompts options
│   ├── datasets.go        # Datasets options
│   ├── scores.go          # Scores options
│   ├── sessions.go        # Sessions options
│   └── models.go          # Models options
│
├── evaluation/            # 🆕 Evaluation support
│   ├── generation.go      # Generation evaluation
│   ├── span.go            # Span evaluation
│   ├── input.go           # Evaluation input
│   ├── metadata.go        # Evaluation metadata
│   ├── mode.go            # Evaluation mode
│   └── persistence.go     # Persistence
│
├── lifecycle/             # 🆕 Lifecycle management
│   ├── manager.go         # Lifecycle manager
│   └── metrics.go         # Metrics collection
│
└── id/                    # 🆕 ID generation
    ├── generator.go       # ID generator
    └── uuid.go            # UUID utilities
```

---

## Migration Tasks

### Task 1: Create pkg/types/ Package

**Files:**
- Create: `pkg/types/trace.go`
- Create: `pkg/types/observation.go`
- Create: `pkg/types/score.go`
- Create: `pkg/types/prompt.go`
- Create: `pkg/types/dataset.go`
- Create: `pkg/types/session.go`
- Create: `pkg/types/model.go`
- Create: `pkg/types/metadata.go`
- Create: `pkg/types/doc.go`
- Modify: `types.go` (root) → thin re-exports

**Step 1:** Extract type definitions from root `types.go` to individual files in `pkg/types/`

**Step 2:** Create type aliases in root `types.go`:
```go
package langfuse

import "github.com/jdziat/langfuse-go/pkg/types"

// Type aliases for backward compatibility
type (
    Trace       = types.Trace
    Observation = types.Observation
    Score       = types.Score
    Prompt      = types.Prompt
    Dataset     = types.Dataset
    Session     = types.Session
    Model       = types.Model
    Metadata    = types.Metadata
    // ... etc
)
```

**Step 3:** Run tests to verify: `go test ./...`

---

### Task 2: Consolidate pkg/errors/

**Files:**
- Modify: `pkg/errors/errors.go` - add missing types from root
- Modify: `pkg/errors/api.go` - consolidate API errors
- Modify: `pkg/errors/async.go` - consolidate async errors
- Modify: `pkg/errors/validation.go` - consolidate validation
- Modify: `pkg/errors/helpers.go` - consolidate helpers
- Delete: `errors.go` (root) → replace with facade
- Delete: `errors_api.go` (root)
- Delete: `errors_async.go` (root)
- Delete: `errors_helpers.go` (root)
- Delete: `errors_validation.go` (root)
- Create: `errors.go` (root) - thin facade

**Step 1:** Compare root error files with pkg/errors/ to identify missing pieces

**Step 2:** Move any missing types/functions to pkg/errors/

**Step 3:** Create facade in root:
```go
package langfuse

import "github.com/jdziat/langfuse-go/pkg/errors"

// Error type aliases
type (
    APIError        = errors.APIError
    ValidationError = errors.ValidationError
    AsyncError      = errors.AsyncError
    // ... etc
)

// Error variables
var (
    ErrNilRequest    = errors.ErrNilRequest
    ErrClientClosed  = errors.ErrClientClosed
    // ... etc
)

// Error helper functions
var (
    NewAPIError        = errors.NewAPIError
    NewValidationError = errors.NewValidationError
    AsAPIError         = errors.AsAPIError
    // ... etc
)
```

**Step 4:** Delete old root error files

**Step 5:** Run tests: `go test ./...`

---

### Task 3: Consolidate pkg/config/

**Files:**
- Modify: `pkg/config/config.go` - add Config struct
- Modify: `pkg/config/options.go` - add ConfigOption functions
- Create: `pkg/config/regions.go` - move regions
- Modify: `pkg/config/env.go` - ensure complete
- Delete: `config.go` (root) → replace with facade
- Delete: `options.go` (root)
- Delete: `regions.go` (root)
- Delete: `env.go` (root)
- Create: `config.go` (root) - thin facade

**Step 1:** Move Config struct and all related types to pkg/config/

**Step 2:** Move ConfigOption functions to pkg/config/options.go

**Step 3:** Move region constants to pkg/config/regions.go

**Step 4:** Create facade in root:
```go
package langfuse

import "github.com/jdziat/langfuse-go/pkg/config"

// Config type alias
type Config = config.Config

// Region constants
const (
    RegionUS = config.RegionUS
    RegionEU = config.RegionEU
)

// ConfigOption type and functions
type ConfigOption = config.Option

var (
    WithBaseURL      = config.WithBaseURL
    WithRegion       = config.WithRegion
    WithBatchSize    = config.WithBatchSize
    // ... etc
)
```

**Step 5:** Delete old root config files

**Step 6:** Run tests: `go test ./...`

---

### Task 4: Consolidate pkg/http/

**Files:**
- Create: `pkg/http/client.go` - move httpClient from root
- Modify: `pkg/http/retry.go` - consolidate retry strategies
- Modify: `pkg/http/circuit.go` - consolidate circuit breaker
- Delete: `http.go` (root) → replace with facade
- Delete: `retry.go` (root)
- Delete: `circuitbreaker.go` (root)

**Step 1:** Move httpClient struct and methods to pkg/http/client.go

**Step 2:** Ensure retry strategies are complete in pkg/http/retry.go

**Step 3:** Ensure circuit breaker is complete in pkg/http/circuit.go

**Step 4:** Update root to use pkg/http:
```go
package langfuse

import pkghttp "github.com/jdziat/langfuse-go/pkg/http"

// HTTP types (internal, not exported but used by Client)
type httpClient = pkghttp.Client

// Exported types
type (
    RetryStrategy     = pkghttp.RetryStrategy
    CircuitBreaker    = pkghttp.CircuitBreaker
    CircuitState      = pkghttp.CircuitState
    // ... etc
)
```

**Step 5:** Delete old root HTTP files

**Step 6:** Run tests: `go test ./...`

---

### Task 5: Consolidate pkg/ingestion/

**Files:**
- Modify: `pkg/ingestion/events.go` - consolidate event types
- Create: `pkg/ingestion/batch.go` - move batching logic
- Create: `pkg/ingestion/queue.go` - move queue logic
- Modify: `pkg/ingestion/backpressure.go` - ensure complete
- Delete: `ingestion.go` (root)
- Delete: `batching.go` (root)
- Delete: `queue.go` (root)
- Delete: `backpressure.go` (root)

**Step 1:** Move all ingestion event types to pkg/ingestion/events.go

**Step 2:** Move batch processing logic to pkg/ingestion/batch.go

**Step 3:** Move queue management to pkg/ingestion/queue.go

**Step 4:** Root client.go will import from pkg/ingestion

**Step 5:** Delete old root ingestion files

**Step 6:** Run tests: `go test ./...`

---

### Task 6: Create pkg/builders/

**Files:**
- Create: `pkg/builders/trace.go`
- Create: `pkg/builders/span.go`
- Create: `pkg/builders/generation.go`
- Create: `pkg/builders/score.go`
- Create: `pkg/builders/event.go`
- Create: `pkg/builders/validation.go`
- Create: `pkg/builders/doc.go`
- Delete: `builders.go` (root) → replace with facade
- Delete: `validated_builder.go` (root)
- Delete: `validation.go` (root)
- Delete: `trace.go` (root)
- Delete: `span.go` (root)
- Delete: `generation.go` (root)
- Create: `builders.go` (root) - thin facade

**Step 1:** Move builder implementations to pkg/builders/

**Step 2:** Create facade in root:
```go
package langfuse

import "github.com/jdziat/langfuse-go/pkg/builders"

type (
    TraceBuilder      = builders.TraceBuilder
    SpanBuilder       = builders.SpanBuilder
    GenerationBuilder = builders.GenerationBuilder
    ScoreBuilder      = builders.ScoreBuilder
    EventBuilder      = builders.EventBuilder
)
```

**Step 3:** Delete old root builder files

**Step 4:** Run tests: `go test ./...`

---

### Task 7: Create pkg/lifecycle/

**Files:**
- Create: `pkg/lifecycle/manager.go`
- Create: `pkg/lifecycle/metrics.go`
- Create: `pkg/lifecycle/doc.go`
- Delete: `lifecycle.go` (root)
- Delete: `metrics.go` (root)
- Delete: `metrics_internal.go` (root)

**Step 1:** Move lifecycle management to pkg/lifecycle/

**Step 2:** Move metrics to pkg/lifecycle/metrics.go

**Step 3:** Root client imports from pkg/lifecycle

**Step 4:** Delete old root files

**Step 5:** Run tests: `go test ./...`

---

### Task 8: Create pkg/id/

**Files:**
- Create: `pkg/id/generator.go`
- Create: `pkg/id/doc.go`
- Delete: `id.go` (root)

**Step 1:** Move ID generation to pkg/id/

**Step 2:** Root imports from pkg/id

**Step 3:** Delete old root file

**Step 4:** Run tests: `go test ./...`

---

### Task 9: Create pkg/options/

**Files:**
- Create: `pkg/options/traces.go`
- Create: `pkg/options/prompts.go`
- Create: `pkg/options/datasets.go`
- Create: `pkg/options/scores.go`
- Create: `pkg/options/sessions.go`
- Create: `pkg/options/models.go`
- Create: `pkg/options/doc.go`
- Delete: `subclient_options.go` (root)
- Delete: `models_options.go` (root)
- Delete: `sessions_options.go` (root)
- Delete: `unified_options.go` (root)
- Create: `options.go` (root) - thin facade

**Step 1:** Move all sub-client options to pkg/options/

**Step 2:** Create facade in root

**Step 3:** Delete old root files

**Step 4:** Run tests: `go test ./...`

---

### Task 10: Create pkg/evaluation/

**Files:**
- Create: `pkg/evaluation/generation.go`
- Create: `pkg/evaluation/span.go`
- Create: `pkg/evaluation/input.go`
- Create: `pkg/evaluation/metadata.go`
- Create: `pkg/evaluation/mode.go`
- Create: `pkg/evaluation/persistence.go`
- Create: `pkg/evaluation/doc.go`
- Delete: `eval_generation.go` (root)
- Delete: `eval_span.go` (root)
- Delete: `evaluation_input.go` (root)
- Delete: `evaluation_metadata.go` (root)
- Delete: `evaluation_mode.go` (root)
- Delete: `persistence.go` (root)
- Create: `evaluation.go` (root) - thin facade

**Step 1:** Move all evaluation code to pkg/evaluation/

**Step 2:** Create facade in root

**Step 3:** Delete old root files

**Step 4:** Run tests: `go test ./...`

---

### Task 11: Create pkg/client/

**Files:**
- Create: `pkg/client/client.go` - core client logic
- Create: `pkg/client/api.go` - API endpoints
- Create: `pkg/client/hooks.go` - HTTP hooks
- Create: `pkg/client/doc.go`
- Modify: `client.go` (root) - thin facade
- Delete: `api.go` (root)
- Delete: `hooks.go` (root)
- Delete: `facade.go` (root) - merge into client facade
- Delete: `simple_api.go` (root) - merge into client facade
- Delete: `v1_api.go` (root) - merge into client facade
- Delete: `helpers.go` (root) - distribute to relevant packages
- Delete: `logging.go` (root) - move to pkg/client or pkg/lifecycle
- Delete: `interfaces.go` (root) - distribute to relevant packages

**Step 1:** Move core client implementation to pkg/client/

**Step 2:** Create thin facade in root client.go:
```go
package langfuse

import "github.com/jdziat/langfuse-go/pkg/client"

// Client is the main Langfuse client
type Client = client.Client

// New creates a new Langfuse client
func New(publicKey, secretKey string, opts ...ConfigOption) (*Client, error) {
    return client.New(publicKey, secretKey, opts...)
}

// NewWithConfig creates a new client from Config
func NewWithConfig(cfg *Config) (*Client, error) {
    return client.NewWithConfig(cfg)
}
```

**Step 3:** Delete old root files

**Step 4:** Run tests: `go test ./...`

---

### Task 12: Consolidate Sub-client Facades

**Files:**
- Modify: `subclients.go` (root) - combine all sub-client facades
- Delete: `traces.go` (root)
- Delete: `observations.go` (root)
- Delete: `scores.go` (root)
- Delete: `sessions.go` (root)
- Delete: `models.go` (root)
- Delete: `prompts.go` (root)
- Delete: `datasets.go` (root)

**Step 1:** Combine all sub-client code into single `subclients.go`

**Step 2:** Delete individual sub-client facade files

**Step 3:** Run tests: `go test ./...`

---

### Task 13: Final Cleanup

**Files:**
- Verify: `doc.go` (root) - update package documentation
- Verify: `version.go` (root) - keep as-is
- Delete: any remaining unused files
- Update: imports throughout codebase

**Step 1:** Review root package - should have ~10 files

**Step 2:** Update doc.go with new architecture description

**Step 3:** Run full test suite: `go test ./...`

**Step 4:** Run linter: `golangci-lint run`

**Step 5:** Verify backward compatibility with example code

---

## Final Root Package Structure

After all tasks complete:
```
langfuse/
├── doc.go           # Package documentation
├── version.go       # Version constant
├── client.go        # Client facade (~50 lines)
├── config.go        # Config facade (~100 lines)
├── errors.go        # Errors facade (~100 lines)
├── types.go         # Type aliases (~150 lines)
├── builders.go      # Builder facade (~50 lines)
├── options.go       # Options facade (~100 lines)
├── subclients.go    # Sub-client facades (~200 lines)
├── evaluation.go    # Evaluation facade (~100 lines)
└── tests/           # Test files (already moved)
```

**Total: ~10 files, ~850 lines** (down from 79 files, ~15,000+ lines)

---

## Backward Compatibility

All existing code continues to work:
```go
// This still works - no changes needed
import "github.com/jdziat/langfuse-go"

client, _ := langfuse.New("pk", "sk")
trace := client.NewTrace().Name("test").Create(ctx)
```

Users who want to customize can now access pkg/:
```go
// Advanced users can import specific packages
import (
    "github.com/jdziat/langfuse-go"
    "github.com/jdziat/langfuse-go/pkg/http"
    "github.com/jdziat/langfuse-go/pkg/builders"
)

// Custom HTTP client
customHTTP := &myHTTPClient{}
// Use pkg/http.Doer interface to provide custom implementation
```

---

## Execution Order

Recommended execution order (dependencies considered):

1. **Task 1: pkg/types/** - No dependencies, foundational
2. **Task 2: pkg/errors/** - No dependencies
3. **Task 3: pkg/config/** - No dependencies
4. **Task 8: pkg/id/** - No dependencies
5. **Task 4: pkg/http/** - Depends on errors, config
6. **Task 5: pkg/ingestion/** - Depends on types, errors
7. **Task 7: pkg/lifecycle/** - Depends on config
8. **Task 6: pkg/builders/** - Depends on types, errors
9. **Task 9: pkg/options/** - Depends on types
10. **Task 10: pkg/evaluation/** - Depends on types, builders
11. **Task 11: pkg/client/** - Depends on all above
12. **Task 12: Sub-client facades** - Depends on client
13. **Task 13: Final cleanup** - Last

---

## Risk Mitigation

1. **Run tests after each task** - `go test ./...`
2. **Commit after each task** - Easy rollback
3. **Keep type aliases** - Backward compatibility
4. **Feature branch** - Don't merge until complete

---

## Success Criteria

- [ ] Root package has ≤10 .go files (excluding tests)
- [ ] All tests pass
- [ ] No breaking changes to public API
- [ ] Examples still compile and work
- [ ] pkg/ packages are independently usable
