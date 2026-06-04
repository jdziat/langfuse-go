# AGENTS.md - AI Coding Assistant Guide

This document provides guidance for AI coding assistants working with the Langfuse Go SDK.

## Project Overview

This is a pure Go SDK for [Langfuse](https://langfuse.com/), an open-source LLM observability platform. The SDK provides:

- Zero external dependencies (stdlib only)
- Type-safe API with fluent builders
- Automatic event batching and async flushing
- Thread-safe concurrent operations
- Full API coverage for traces, observations, scores, prompts, datasets, and more

**Module**: `github.com/jdziat/langfuse-go`
**Go Version**: 1.23+
**License**: MIT

## Repository Structure

The SDK is organized as a thin root facade composed from importable building
blocks under `pkg/`. There are no longer any monolithic root files such as
`api.go`, `http.go`, `traces.go`, or `ingestion.go`; that code now lives in the
relevant `pkg/*` subpackages.

```
langfuse-go/
├── *.go                    # Root facade package "langfuse" (see below)
├── *_test.go               # Root facade tests
├── pkg/                     # Importable building blocks (see "Package Layout")
│   ├── api/                 # Per-resource read sub-clients (traces, scores, ...)
│   ├── builders/            # Fluent builder helpers + validation
│   ├── client/              # Core client: HTTP, config, batching, lifecycle
│   ├── config/              # Config types, regions, env helpers
│   ├── errors/              # Error types, sentinels, errors.Is/As helpers
│   ├── evaluation/          # Evaluation result types, flattening, persistence
│   ├── http/                # Retry, circuit breaker, pagination, hooks, Doer
│   ├── id/                  # ID generation
│   ├── ingestion/           # Event structs, backpressure, UUID
│   ├── lifecycle/           # Lifecycle manager + metrics
│   └── types/               # Shared data types and enums
├── evaluation/              # Root "evaluation" package: QA/RAG/classification helpers
├── langfusetest/            # Test doubles: mock client, mock server
├── examples/                # Runnable examples (basic, advanced, evaluation, ...)
├── cmd/langfuse-hooks/      # Git hooks helper command
├── internal/                # Internal-only helpers
├── docs/, content/, layouts/ # Hugo documentation site
├── .github/workflows/       # CI/CD pipelines
├── .goreleaser.yaml         # Release automation
└── README.md                # User documentation
```

### Root facade files

The root package `langfuse` (module path `github.com/jdziat/langfuse-go`) is a
facade. It embeds `*pkgclient.Client` (from `pkg/client`) and re-exports types
via Go type aliases (e.g. `type Trace = types.Trace`). There is no `Pkg`-prefixed
re-export scheme.

| File | Purpose |
|------|---------|
| `client.go` | Root `Client` (embeds `*pkgclient.Client`), `New`, sub-client wiring |
| `config.go` | Root `Config`, `ConfigOption`s, conversion to `pkgclient.Config` |
| `types.go` | Type aliases re-exporting `pkg/types` (Trace, Observation, Score, ...) |
| `builders.go` | Root-facing fluent builders and contexts |
| `simple_api.go` | Convenience "Simple API" methods on `Client` |
| `subclients.go` | API sub-clients (Traces, Scores, Prompts, Datasets, ...) |
| `lifecycle.go` | Lifecycle/queue glue, re-exported sentinels (`ErrBackpressure`, ...) |
| `options.go` | Additional functional options |
| `evaluation.go` | Root wiring for the evaluation feature |
| `doc.go` | Package overview and canonical usage documentation |

### Package layout (`pkg/`)

| Package | Purpose |
|---------|---------|
| `pkg/client` | Core client: HTTP wiring, config, batching, ingestion, lifecycle, options |
| `pkg/types` | Shared data types (Trace, Observation, Score, Prompt, Dataset, Session, Time, Metadata), enums, constants |
| `pkg/api/*` | Per-resource read sub-clients: `traces`, `observations`, `scores`, `prompts`, `datasets`, `sessions`, `models` |
| `pkg/builders` | Fluent builder helpers, interfaces, input validation |
| `pkg/config` | Configuration types, regions, environment helpers |
| `pkg/errors` | Error types, sentinel errors, `errors.Is`/`errors.As` helpers |
| `pkg/evaluation` | Evaluation result types, flattening, persistence |
| `pkg/http` | HTTP utilities: retry, circuit breaker, pagination, hooks, request `Doer` |
| `pkg/id` | ID generation |
| `pkg/ingestion` | Event structs, backpressure, UUID generation |
| `pkg/lifecycle` | Lifecycle manager and metrics |

## Coding Patterns

### Fluent Builder Pattern

All entity creation uses fluent builders:

```go
trace, err := client.NewTrace().
    Name("operation").
    UserID("user-123").
    Input(data).
    Tags([]string{"prod"}).
    Create()
```

- Builders return `*BuilderType` for method chaining
- `.Create()` or `.Apply()` finalizes and queues the event
- Builders wrap internal event structs

### Context Hierarchy

Observations nest via embedded contexts:

```
TraceContext
├── SpanContext (embeds TraceContext)
│   └── Child spans, generations, events
├── GenerationContext
└── EventContext
```

### Sub-Client Pattern

Each API area has a dedicated sub-client:

```go
client.Traces()      // *TracesClient
client.Scores()      // *ScoresClient
client.Prompts()     // *PromptsClient
// etc.
```

Sub-clients are wired in `subclients.go` and use the embedded core client's
HTTP client and event queue. The per-resource read implementations live under
`pkg/api/*`.

### Error Handling

- Return explicit errors from all public methods
- Use sentinel errors for configuration validation (`ErrMissingPublicKey`, etc.)
- Use `errors.Is()` for checking specific error types
- `APIError` wraps HTTP responses with helper methods (`IsRetryable()`, etc.)

### JSON Serialization

- Use `json:"fieldName,omitempty"` tags
- Custom `Time` type handles RFC3339Nano format
- Nullable fields use pointers
- Omitempty prevents sending empty values

## Testing Requirements

### Before Committing

Always run:

```bash
go test -v -race ./...
go vet ./...
go fmt ./...
```

### Test Patterns

- **Mock HTTP**: Use `httptest.NewServer()` for API tests
- **Table-driven tests**: For validation and configuration scenarios
- **Race detection**: All tests run with `-race` flag in CI

### Test File Naming

Each package keeps its tests alongside its source in `*_test.go` files:
- `client.go` → `client_test.go` (root facade)
- `pkg/ingestion/events.go` → `pkg/ingestion/ingestion_test.go`
- `pkg/http/circuit.go` → `pkg/http/http_test.go`

## Thread Safety

The client uses `sync.Mutex` to protect:
- `pendingEvents` slice
- `flushTimer`
- `closed` flag

All public API methods must acquire locks when accessing shared state.

## Event Batching

Events are batched automatically:

1. Events queued via `queueEvent()` to `pendingEvents[]`
2. Flush triggers:
   - Batch reaches `BatchSize` (default 100) → async flush
   - `FlushInterval` timer fires (default 5s) → background flush
   - Explicit `client.Flush(ctx)` → synchronous flush
   - `client.Shutdown(ctx)` → final synchronous flush

## Common Tasks

### Adding a New Builder Type

1. Define the event struct in `pkg/ingestion/events.go`
2. Add the builder struct with fluent methods (root builders live in `builders.go`)
3. Add the context struct for observation management
4. Wire to parent context (TraceContext or SpanContext)
5. Add tests with httptest mock server

### Adding a New API Sub-Client

1. Add the read implementation under `pkg/api/<resource>/`
2. Add the root-facing sub-client struct holding `*Client` in `subclients.go`
3. Implement CRUD methods using the embedded core client's HTTP client
4. Add an accessor method to the root `Client` in `subclients.go`
5. Create corresponding `*_test.go` files
6. Update README.md with examples

### Adding New Configuration Options

1. Add the field to the core `Config` in `pkg/client/config.go` (and to the root
   `Config` in `config.go` if it is a root-only option such as evaluation)
2. Create the `ConfigOption` function (e.g., `WithNewOption`) in `config.go`/`options.go`
3. Map the field through `convertToPkgClientConfig` in `client.go` when needed
4. Set a sensible default and add validation if needed
5. Document in README.md

## Do's and Don'ts

### Do

- Follow the fluent builder pattern for new entity types
- Use `context.Context` for all HTTP operations
- Return errors explicitly (no panics)
- Use table-driven tests
- Protect shared state with mutex
- Use `json:",omitempty"` for optional fields
- Add doc comments on exported types and methods
- Run `go fmt` before committing

### Don't

- Add external dependencies (keep stdlib only)
- Block on network I/O in builder methods
- Expose internal event structs
- Use global state
- Skip the race detector in tests
- Forget to update README.md for new features

## Commit Message Convention

Follow conventional commits for automated changelog generation:

- `feat:` - New features
- `fix:` - Bug fixes
- `perf:` - Performance improvements
- `docs:` - Documentation changes
- `test:` - Test additions/changes
- `refactor:` - Code refactoring

## CI/CD

### CI Pipeline (`.github/workflows/ci.yml`)

- Tests on Go 1.23 and 1.24
- Race detector enabled
- Coverage uploaded to Codecov
- Linting: `go vet`, `gofmt`, `staticcheck`

### Release Pipeline (`.github/workflows/release.yml`)

- Triggered by version tags (`v*`)
- Uses GoReleaser for semantic versioning
- Auto-generates changelog from commits

## Configuration Defaults

| Setting | Default | Description |
|---------|---------|-------------|
| `BatchSize` | 100 | Events per batch |
| `FlushInterval` | 5s | Auto-flush interval |
| `Timeout` | 30s | HTTP request timeout |
| `MaxRetries` | 3 | Retry attempts |
| `RetryDelay` | 1s | Initial retry delay |

## Regions

```go
RegionUS   = "us"      // https://us.cloud.langfuse.com
RegionEU   = "eu"      // https://cloud.langfuse.com
RegionHIPAA = "hipaa"  // https://hipaa.cloud.langfuse.com
```

## Useful Commands

```bash
# Run all tests with race detection
go test -v -race ./...

# Run tests with coverage
go test -v -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Check formatting
gofmt -l .

# Static analysis
go vet ./...
staticcheck ./...

# Build examples
go build -o /dev/null ./examples/basic
go build -o /dev/null ./examples/advanced

# Tidy dependencies
go mod tidy
```

---

## Codebase State & Working Notes

This SDK has been through a structured remediation effort (tracked in
`docs/review-remediation-plan.md`). The monolithic root source files referenced
by older versions of this guide (`api.go`, `http.go`, `traces.go`,
`observations.go`, `scores.go`, `prompts.go`, `datasets.go`, `sessions.go`,
`models.go`, `types_generic.go`, `ingestion.go`) no longer exist; that code was
migrated into the `pkg/*` packages described under "Package layout" above. When
older notes, comments, or examples reference those files, treat them as stale.

There is no standing backlog of fabricated technical debt to chase. Make changes
against the actual layout, keep the public API stable, and follow the
conventions below.

### Backward compatibility

This is a released `1.x` module. Treat the public API as stable:

- Do **not** remove an exported symbol or change an existing exported signature.
- You may **add** new exported APIs, and you may mark obsolete ones with a
  `// Deprecated:` doc comment that points to the replacement.
- The root `langfuse` package is the stable surface. It re-exports `pkg/*` types
  via Go type aliases (e.g. `type Trace = types.Trace`) and embeds
  `*pkgclient.Client`. There is no `Pkg`-prefixed naming scheme.

### Error handling conventions

- Sentinel errors support `errors.Is` (e.g. `errors.Is(err, langfuse.ErrNotFound)`).
- Extraction helpers follow the `As*` shape, returning `(*T, bool)`.
- `APIError` carries the HTTP response and exposes helpers such as
  `IsRetryable()`, `IsNotFound()`, and `IsUnauthorized()`.
- Network classification lives in `pkg/errors`; prefer typed checks
  (`net.OpError`, `net.DNSError`, `Timeout()`) over string matching.

### Circuit breaker & HTTP resilience

The circuit breaker and retry strategies live in `pkg/http`. HTTP operations go
through the core client's request `Doer`, so resilience behavior is centralized
rather than re-implemented per sub-client. When adding HTTP calls, route them
through the existing core client rather than constructing ad-hoc requests.

### Metadata

`langfuse.Metadata` (alias of `pkg/types.Metadata`) is the type for
metadata maps in public APIs. Prefer it over raw `map[string]any` when adding
new public fields.

### Validation checklist

Before opening a PR:

```bash
# Format, build, vet, and test the whole module.
gofmt -l .              # Expected: no output
go build ./...
go vet ./...
go test -race ./...

# No interface{} in source (use any).
grep -rn "interface{}" --include="*.go" . | grep -v _test.go
# Expected: no output

# Coverage (informational).
go test -coverprofile=cov.out ./... && go tool cover -func=cov.out | grep total
```

### When adding new features

1. Put implementation in the appropriate `pkg/*` package; keep the root package a
   thin facade.
2. Re-export new public types from the root via an alias only when they are part
   of the stable surface.
3. Preserve backward compatibility (add, never break).
4. Test error paths, not just the happy path.
5. Use `Metadata` instead of raw `map[string]any` in public APIs.
6. Follow the `As*` shape for new error-extraction helpers.
