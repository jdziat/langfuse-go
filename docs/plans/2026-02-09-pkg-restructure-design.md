# Langfuse Go SDK: pkg/ Restructure Design

## Goal

Reorganize the SDK into a clean layered architecture with:
- **Root facade**: Thin public API (re-exports only)
- **pkg/**: All implementation in structured domain packages

## Package Structure

```
langfuse-go/
├── client.go           # Client struct + New() (~100 lines)
├── facade.go           # Type aliases for all public types (~150 lines)
├── doc.go              # Package documentation
├── version.go          # Version info
│
├── pkg/
│   ├── config/         # Configuration, options, regions
│   │   ├── config.go
│   │   ├── options.go
│   │   ├── regions.go
│   │   └── defaults.go
│   │
│   ├── errors/         # All error types and helpers
│   │   ├── errors.go
│   │   ├── api.go
│   │   ├── async.go
│   │   ├── validation.go
│   │   └── helpers.go
│   │
│   ├── http/           # HTTP client, retry, circuit breaker
│   │   ├── client.go
│   │   ├── retry.go
│   │   ├── circuit.go
│   │   └── request.go
│   │
│   ├── ingestion/      # Event queue, batching, backpressure
│   │   ├── queue.go
│   │   ├── batch.go
│   │   ├── backpressure.go
│   │   ├── events.go
│   │   └── persistence.go
│   │
│   ├── tracing/        # Trace, Span, Generation, Event + builders
│   │   ├── trace.go
│   │   ├── span.go
│   │   ├── generation.go
│   │   ├── event.go
│   │   ├── observation.go
│   │   ├── builders.go
│   │   └── types.go
│   │
│   ├── evaluation/     # Scores, evaluation workflows
│   │   ├── scores.go
│   │   ├── mode.go
│   │   ├── input.go
│   │   ├── metadata.go
│   │   └── workflows.go
│   │
│   └── resources/      # API resource clients
│       ├── prompts.go
│       ├── datasets.go
│       ├── models.go
│       ├── sessions.go
│       ├── traces.go
│       └── observations.go
```

## Dependency Graph

```
                    ┌─────────────┐
                    │   config    │  ← No dependencies
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │   errors    │  ← Depends on config
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │    http     │  ← Depends on config, errors
                    └──────┬──────┘
                           │
                    ┌──────▼──────┐
                    │  ingestion  │  ← Depends on http, errors, config
                    └──────┬──────┘
                           │
          ┌────────────────┼────────────────┐
          │                │                │
   ┌──────▼──────┐  ┌──────▼──────┐  ┌──────▼──────┐
   │   tracing   │  │ evaluation  │  │  resources  │
   └─────────────┘  └─────────────┘  └─────────────┘
          │                │                │
          └────────────────┼────────────────┘
                           │
                    ┌──────▼──────┐
                    │   (root)    │  ← Thin facade
                    │  langfuse   │
                    └─────────────┘
```

**Key rules:**
- Lower packages never import higher packages
- Peer packages (tracing, evaluation, resources) cannot import each other
- Root facade imports everything and re-exports

## Root Facade Design

### client.go
```go
package langfuse

import (
    "github.com/jdziat/langfuse-go/pkg/config"
    "github.com/jdziat/langfuse-go/pkg/tracing"
    "github.com/jdziat/langfuse-go/pkg/resources"
)

type Client struct {
    *tracing.TracingClient
    prompts  *resources.PromptsClient
    datasets *resources.DatasetsClient
    // ...
}

func New(publicKey, secretKey string, opts ...config.Option) (*Client, error) {
    cfg := config.New(publicKey, secretKey, opts...)
    // Initialize and return client
}
```

### facade.go
```go
package langfuse

import (
    "github.com/jdziat/langfuse-go/pkg/config"
    "github.com/jdziat/langfuse-go/pkg/errors"
    "github.com/jdziat/langfuse-go/pkg/tracing"
)

// Config re-exports
type Config = config.Config
type Region = config.Region
type Option = config.Option

const (
    RegionUS = config.RegionUS
    RegionEU = config.RegionEU
)

// Tracing re-exports
type Trace = tracing.Trace
type Span = tracing.Span
type Generation = tracing.Generation

// Error re-exports
type APIError = errors.APIError
var ErrNotFound = errors.ErrNotFound
var AsAPIError = errors.AsAPIError

// Option re-exports
var WithRegion = config.WithRegion
var WithTimeout = config.WithTimeout
// ...
```

## Migration Strategy

### Phase 1: Create pkg/ Structure (No Breaking Changes)
1. Create all pkg/ directories
2. Copy (not move) types to pkg/ packages
3. Add comprehensive tests for pkg/ packages
4. Root facade re-exports from pkg/ using type aliases
5. **Result:** Two copies exist temporarily, API unchanged

### Phase 2: Internal Migration
1. Update internal references to use pkg/ imports
2. Root files become thin wrappers calling pkg/
3. Run full test suite after each file migration
4. **Result:** Logic lives in pkg/, root delegates

### Phase 3: Cleanup
1. Remove duplicate code from root (now just facade)
2. Update documentation
3. Final test pass + coverage check
4. **Result:** Clean separation achieved

## File Movement Summary

| From Root | To pkg/ | File Count |
|-----------|---------|------------|
| `config.go`, `options.go`, `regions.go`, `env.go` | `pkg/config/` | 4 |
| `errors*.go` | `pkg/errors/` | 5 |
| `http.go`, `retry.go`, `circuitbreaker.go` | `pkg/http/` | 3 |
| `queue.go`, `batching.go`, `backpressure.go`, `ingestion.go`, `persistence.go` | `pkg/ingestion/` | 5 |
| `trace.go`, `span.go`, `generation.go`, `builders.go`, etc. | `pkg/tracing/` | 8 |
| `scores.go`, `eval_*.go`, `evaluation_*.go` | `pkg/evaluation/` | 6 |
| `prompts.go`, `datasets.go`, `models.go`, `sessions.go`, etc. | `pkg/resources/` | 6 |

**Total:** ~37 files moved into pkg/, root reduced to ~5 files

## Testing Strategy

### Per-Package Tests
Each pkg/ package gets its own test file:
- `pkg/config/config_test.go`
- `pkg/errors/errors_test.go`
- `pkg/http/client_test.go`
- `pkg/ingestion/queue_test.go`
- `pkg/tracing/trace_test.go`
- `pkg/evaluation/scores_test.go`
- `pkg/resources/prompts_test.go`

### Validation Checkpoints
1. After each package migration: `go build ./...` + `go test ./pkg/[package]/...`
2. After all migrations: Full test suite with race detection
3. Final: Coverage must stay ≥ 68.5%

### Backward Compatibility Tests
```go
// compat_test.go
func TestBackwardCompatibility(t *testing.T) {
    // Verify all public types are accessible via root package
    var _ langfuse.Client
    var _ langfuse.Trace
    var _ langfuse.Config
    var _ langfuse.APIError

    // Verify constructors work
    _, err := langfuse.New("pk", "sk", langfuse.WithRegion(langfuse.RegionUS))
    // ...
}
```

## Backward Compatibility

**100% API compatible** - existing code continues to work unchanged:
```go
// This still works exactly as before
client, _ := langfuse.New(publicKey, secretKey, langfuse.WithRegion(langfuse.RegionUS))
trace, _ := client.NewTrace().Name("my-trace").Create(ctx)
```

## Success Criteria

- [ ] All 37 files moved to pkg/
- [ ] Root reduced to ~5 files (facade only)
- [ ] All tests pass
- [ ] Coverage ≥ 68.5%
- [ ] Zero breaking changes to public API
- [ ] Clear dependency hierarchy (no cycles)
- [ ] Documentation updated
