# pkg/ Restructure Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Reorganize SDK into layered pkg/ structure with thin root facade while maintaining 100% backward compatibility.

**Architecture:** Move all implementation into pkg/ packages (config, errors, http, ingestion, tracing, evaluation, resources). Root becomes thin facade with type aliases and re-exports. Three-phase migration: create pkg/, migrate internals, cleanup root.

**Tech Stack:** Go 1.25, standard library only (zero dependencies)

---

## Phase 1: Infrastructure Packages

### Task 1: Create pkg/errors Package

**Files:**
- Create: `pkg/errors/errors.go`
- Create: `pkg/errors/api.go`
- Create: `pkg/errors/async.go`
- Create: `pkg/errors/validation.go`
- Create: `pkg/errors/helpers.go`
- Create: `pkg/errors/doc.go`
- Test: `pkg/errors/errors_test.go`

**Step 1: Create pkg/errors directory and doc.go**

```bash
mkdir -p pkg/errors
```

```go
// pkg/errors/doc.go
// Package errors provides error types and helpers for the Langfuse SDK.
package errors
```

**Step 2: Create pkg/errors/errors.go with base types**

Copy from root `errors.go`:
- `ErrorCode` type and constants
- `LangfuseError` interface
- `CodedError` interface
- Sentinel errors (`ErrClientClosed`, `ErrMissingPublicKey`, etc.)
- `ShutdownError` type
- `CompilationError` type

**Step 3: Create pkg/errors/api.go**

Copy from root `errors_api.go`:
- `APIError` struct and methods
- `IngestionError`, `IngestionResult`, `IngestionSuccess`
- Sentinel API errors (`ErrNotFound`, `ErrUnauthorized`, etc.)

**Step 4: Create pkg/errors/async.go**

Copy from root `errors_async.go`:
- `AsyncError` and `AsyncErrorOperation`
- `AsyncErrorHandler` and related types

**Step 5: Create pkg/errors/validation.go**

Copy from root `errors_validation.go`:
- `ValidationError` type and methods

**Step 6: Create pkg/errors/helpers.go**

Copy from root `errors_helpers.go`:
- `AsAPIError`, `AsValidationError`, etc.
- `IsRetryable`, `RetryAfter`
- `WrapError`, `WrapErrorf`

**Step 7: Create pkg/errors/errors_test.go**

```go
package errors

import (
    "testing"
)

func TestAPIError(t *testing.T) {
    err := &APIError{StatusCode: 404, Message: "not found"}
    if err.Error() == "" {
        t.Error("expected non-empty error message")
    }
}

func TestAsAPIError(t *testing.T) {
    apiErr := &APIError{StatusCode: 400}
    extracted, ok := AsAPIError(apiErr)
    if !ok || extracted.StatusCode != 400 {
        t.Error("expected to extract APIError")
    }
}
```

**Step 8: Run tests**

```bash
go test -v ./pkg/errors/...
```
Expected: PASS

**Step 9: Commit**

```bash
git add pkg/errors/
git commit -m "feat(pkg/errors): create errors package

Moves error types to pkg/errors:
- Base error types and sentinels
- APIError and ingestion errors
- AsyncError and handler
- ValidationError
- Helper functions (As*, IsRetryable, etc.)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Create pkg/config Package (Expand Existing)

**Files:**
- Modify: `pkg/config/types.go`
- Create: `pkg/config/config.go`
- Create: `pkg/config/options.go`
- Create: `pkg/config/env.go`
- Test: `pkg/config/config_test.go`

**Step 1: Create pkg/config/config.go with Config struct**

Copy `Config` struct and related types from root `config.go`.

**Step 2: Create pkg/config/options.go with functional options**

Copy all `With*` option functions from root `config.go` and `options.go`.

**Step 3: Create pkg/config/env.go**

Copy environment variable helpers from root `env.go`.

**Step 4: Create pkg/config/config_test.go**

```go
package config

import "testing"

func TestNewConfig(t *testing.T) {
    cfg := &Config{
        PublicKey: "pk-test",
        SecretKey: "sk-test",
    }
    if cfg.PublicKey != "pk-test" {
        t.Error("expected PublicKey to be set")
    }
}

func TestWithRegion(t *testing.T) {
    // Test option functions
}
```

**Step 5: Run tests**

```bash
go test -v ./pkg/config/...
```
Expected: PASS

**Step 6: Commit**

```bash
git add pkg/config/
git commit -m "feat(pkg/config): expand config package

Adds:
- Config struct and types
- Functional options (With*)
- Environment variable helpers

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 3: Create pkg/http Package

**Files:**
- Create: `pkg/http/client.go`
- Create: `pkg/http/retry.go`
- Create: `pkg/http/circuit.go`
- Create: `pkg/http/doc.go`
- Test: `pkg/http/client_test.go`

**Step 1: Create pkg/http directory and doc.go**

```bash
mkdir -p pkg/http
```

```go
// pkg/http/doc.go
// Package http provides HTTP client functionality for the Langfuse SDK.
package http
```

**Step 2: Create pkg/http/client.go**

Copy HTTP client wrapper from root `http.go`.

**Step 3: Create pkg/http/retry.go**

Copy retry logic from root `retry.go`.

**Step 4: Create pkg/http/circuit.go**

Copy circuit breaker from root `circuitbreaker.go`.

**Step 5: Create pkg/http/client_test.go**

```go
package http

import "testing"

func TestHTTPClient(t *testing.T) {
    // Basic client creation test
}

func TestRetryLogic(t *testing.T) {
    // Retry behavior test
}
```

**Step 6: Run tests**

```bash
go test -v ./pkg/http/...
```
Expected: PASS

**Step 7: Commit**

```bash
git add pkg/http/
git commit -m "feat(pkg/http): create HTTP client package

Moves HTTP functionality to pkg/http:
- HTTP client wrapper
- Retry logic with backoff
- Circuit breaker implementation

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Create pkg/ingestion Package

**Files:**
- Create: `pkg/ingestion/queue.go`
- Create: `pkg/ingestion/batch.go`
- Create: `pkg/ingestion/backpressure.go`
- Create: `pkg/ingestion/events.go`
- Create: `pkg/ingestion/persistence.go`
- Create: `pkg/ingestion/doc.go`
- Test: `pkg/ingestion/ingestion_test.go`

**Step 1: Create pkg/ingestion directory and doc.go**

```bash
mkdir -p pkg/ingestion
```

```go
// pkg/ingestion/doc.go
// Package ingestion provides event queue and batch processing for the Langfuse SDK.
package ingestion
```

**Step 2: Create pkg/ingestion/events.go**

Copy event types from root `ingestion.go`:
- `traceEvent`, `observationEvent`, `scoreEvent`
- Event body interfaces

**Step 3: Create pkg/ingestion/queue.go**

Copy queue logic from root `queue.go`.

**Step 4: Create pkg/ingestion/batch.go**

Copy batch processor from root `batching.go`.

**Step 5: Create pkg/ingestion/backpressure.go**

Copy backpressure handling from root `backpressure.go`.

**Step 6: Create pkg/ingestion/persistence.go**

Copy persistence logic from root `persistence.go`.

**Step 7: Create pkg/ingestion/ingestion_test.go**

```go
package ingestion

import "testing"

func TestEventQueue(t *testing.T) {
    // Queue behavior test
}

func TestBatchProcessor(t *testing.T) {
    // Batch processing test
}
```

**Step 8: Run tests**

```bash
go test -v ./pkg/ingestion/...
```
Expected: PASS

**Step 9: Commit**

```bash
git add pkg/ingestion/
git commit -m "feat(pkg/ingestion): create ingestion package

Moves event ingestion to pkg/ingestion:
- Event types and queue
- Batch processor
- Backpressure handling
- Persistence for recovery

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Phase 2: Domain Packages

### Task 5: Create pkg/tracing Package

**Files:**
- Create: `pkg/tracing/trace.go`
- Create: `pkg/tracing/span.go`
- Create: `pkg/tracing/generation.go`
- Create: `pkg/tracing/event.go`
- Create: `pkg/tracing/builders.go`
- Create: `pkg/tracing/types.go`
- Create: `pkg/tracing/doc.go`
- Test: `pkg/tracing/tracing_test.go`

**Step 1: Create pkg/tracing directory and doc.go**

```bash
mkdir -p pkg/tracing
```

```go
// pkg/tracing/doc.go
// Package tracing provides trace, span, and generation types for the Langfuse SDK.
package tracing
```

**Step 2: Create pkg/tracing/types.go**

Copy shared types from root `types.go`:
- `Metadata`, `Usage`, `ObservationLevel`
- Helper types

**Step 3: Create pkg/tracing/trace.go**

Copy Trace type and builder from root `trace.go`.

**Step 4: Create pkg/tracing/span.go**

Copy Span type and builder from root `span.go`.

**Step 5: Create pkg/tracing/generation.go**

Copy Generation type and builder from root `generation.go`.

**Step 6: Create pkg/tracing/builders.go**

Copy builder utilities from root `builders.go`:
- `MetadataBuilder`, `TagsBuilder`, `UsageBuilder`

**Step 7: Create pkg/tracing/tracing_test.go**

```go
package tracing

import "testing"

func TestTraceBuilder(t *testing.T) {
    // Trace builder test
}

func TestSpanBuilder(t *testing.T) {
    // Span builder test
}
```

**Step 8: Run tests**

```bash
go test -v ./pkg/tracing/...
```
Expected: PASS

**Step 9: Commit**

```bash
git add pkg/tracing/
git commit -m "feat(pkg/tracing): create tracing package

Moves tracing types to pkg/tracing:
- Trace, Span, Generation, Event types
- Builder pattern implementations
- Shared types (Metadata, Usage, etc.)

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 6: Create pkg/evaluation Package

**Files:**
- Create: `pkg/evaluation/scores.go`
- Create: `pkg/evaluation/mode.go`
- Create: `pkg/evaluation/input.go`
- Create: `pkg/evaluation/metadata.go`
- Create: `pkg/evaluation/doc.go`
- Test: `pkg/evaluation/evaluation_test.go`

**Step 1: Create pkg/evaluation directory and doc.go**

```bash
mkdir -p pkg/evaluation
```

```go
// pkg/evaluation/doc.go
// Package evaluation provides scoring and evaluation functionality for the Langfuse SDK.
package evaluation
```

**Step 2: Create pkg/evaluation/scores.go**

Copy Score type from root `scores.go`.

**Step 3: Create pkg/evaluation/mode.go**

Copy evaluation mode from root `evaluation_mode.go`.

**Step 4: Create pkg/evaluation/input.go**

Copy evaluation input from root `evaluation_input.go`.

**Step 5: Create pkg/evaluation/metadata.go**

Copy evaluation metadata from root `evaluation_metadata.go`.

**Step 6: Create pkg/evaluation/evaluation_test.go**

```go
package evaluation

import "testing"

func TestScoreBuilder(t *testing.T) {
    // Score builder test
}
```

**Step 7: Run tests**

```bash
go test -v ./pkg/evaluation/...
```
Expected: PASS

**Step 8: Commit**

```bash
git add pkg/evaluation/
git commit -m "feat(pkg/evaluation): create evaluation package

Moves evaluation types to pkg/evaluation:
- Score type and builder
- Evaluation mode support
- Input and metadata helpers

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 7: Create pkg/resources Package

**Files:**
- Create: `pkg/resources/prompts.go`
- Create: `pkg/resources/datasets.go`
- Create: `pkg/resources/models.go`
- Create: `pkg/resources/sessions.go`
- Create: `pkg/resources/traces.go`
- Create: `pkg/resources/observations.go`
- Create: `pkg/resources/doc.go`
- Test: `pkg/resources/resources_test.go`

**Step 1: Create pkg/resources directory and doc.go**

```bash
mkdir -p pkg/resources
```

```go
// pkg/resources/doc.go
// Package resources provides API resource clients for the Langfuse SDK.
package resources
```

**Step 2: Create pkg/resources/prompts.go**

Copy PromptsClient from root `prompts.go`.

**Step 3: Create pkg/resources/datasets.go**

Copy DatasetsClient from root `datasets.go`.

**Step 4: Create pkg/resources/models.go**

Copy ModelsClient from root `models.go`.

**Step 5: Create pkg/resources/sessions.go**

Copy SessionsClient from root `sessions.go`.

**Step 6: Create pkg/resources/traces.go**

Copy TracesClient from root `traces.go`.

**Step 7: Create pkg/resources/observations.go**

Copy ObservationsClient from root `observations.go`.

**Step 8: Create pkg/resources/resources_test.go**

```go
package resources

import "testing"

func TestPromptsClient(t *testing.T) {
    // Prompts client test
}

func TestDatasetsClient(t *testing.T) {
    // Datasets client test
}
```

**Step 9: Run tests**

```bash
go test -v ./pkg/resources/...
```
Expected: PASS

**Step 10: Commit**

```bash
git add pkg/resources/
git commit -m "feat(pkg/resources): create resources package

Moves API clients to pkg/resources:
- PromptsClient
- DatasetsClient
- ModelsClient
- SessionsClient
- TracesClient
- ObservationsClient

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Phase 3: Root Facade

### Task 8: Create Root Facade

**Files:**
- Create: `facade.go`
- Modify: `client.go`
- Create: `compat_test.go`

**Step 1: Create facade.go with type aliases**

```go
// facade.go
package langfuse

import (
    "github.com/jdziat/langfuse-go/pkg/config"
    "github.com/jdziat/langfuse-go/pkg/errors"
    "github.com/jdziat/langfuse-go/pkg/tracing"
    "github.com/jdziat/langfuse-go/pkg/evaluation"
)

// Config re-exports
type Config = config.Config
type Region = config.Region
type Option = config.Option

const (
    RegionUS   = config.RegionUS
    RegionEU   = config.RegionEU
    RegionHIPAA = config.RegionHIPAA
)

// Option re-exports
var (
    WithRegion        = config.WithRegion
    WithBaseURL       = config.WithBaseURL
    WithTimeout       = config.WithTimeout
    WithBatchSize     = config.WithBatchSize
    WithFlushInterval = config.WithFlushInterval
    // ... all other options
)

// Tracing re-exports
type Trace = tracing.Trace
type Span = tracing.Span
type Generation = tracing.Generation
type Metadata = tracing.Metadata
type Usage = tracing.Usage

// Error re-exports
type APIError = errors.APIError
type ValidationError = errors.ValidationError

var (
    ErrNotFound     = errors.ErrNotFound
    ErrUnauthorized = errors.ErrUnauthorized
    AsAPIError      = errors.AsAPIError
    IsRetryable     = errors.IsRetryable
    // ... all other error helpers
)

// Evaluation re-exports
type Score = evaluation.Score
```

**Step 2: Update client.go to use pkg/ imports**

Modify client.go to import from pkg/ and delegate.

**Step 3: Create compat_test.go**

```go
package langfuse

import (
    "context"
    "testing"
)

func TestBackwardCompatibility_Types(t *testing.T) {
    // Verify all types are accessible
    var _ Client
    var _ Trace
    var _ Span
    var _ Generation
    var _ Config
    var _ APIError
    var _ Region
}

func TestBackwardCompatibility_Constants(t *testing.T) {
    if RegionUS == "" {
        t.Error("RegionUS should be accessible")
    }
    if RegionEU == "" {
        t.Error("RegionEU should be accessible")
    }
}

func TestBackwardCompatibility_Functions(t *testing.T) {
    // Verify option functions work
    _ = WithRegion(RegionUS)
    _ = WithTimeout(30)
}
```

**Step 4: Run all tests**

```bash
go test -v ./...
```
Expected: ALL PASS

**Step 5: Commit**

```bash
git add facade.go client.go compat_test.go
git commit -m "feat: create root facade with re-exports

Adds facade.go re-exporting all public types from pkg/:
- Config types and options
- Tracing types (Trace, Span, Generation)
- Error types and helpers
- Evaluation types

Maintains 100% backward compatibility.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 9: Cleanup Root (Remove Duplicates)

**Files:**
- Delete: `errors.go`, `errors_api.go`, `errors_async.go`, `errors_validation.go`, `errors_helpers.go`
- Delete: `http.go`, `retry.go`, `circuitbreaker.go`
- Delete: `queue.go`, `batching.go`, `backpressure.go`, `ingestion.go`, `persistence.go`
- Delete: `trace.go`, `span.go`, `generation.go`, `builders.go`, `types.go`
- Delete: `scores.go`, `evaluation_mode.go`, `evaluation_input.go`, `evaluation_metadata.go`
- Delete: `prompts.go`, `datasets.go`, `models.go`, `sessions.go`, `traces.go`, `observations.go`

**Step 1: Verify pkg/ tests pass**

```bash
go test -v ./pkg/...
```
Expected: ALL PASS

**Step 2: Remove duplicate root files one group at a time**

```bash
# Remove error files (now in pkg/errors/)
rm errors.go errors_api.go errors_async.go errors_validation.go errors_helpers.go

# Verify build
go build ./...
```

**Step 3: Continue removing groups**

Remove each group, verifying build after each:
- HTTP files
- Ingestion files
- Tracing files
- Evaluation files
- Resource files

**Step 4: Run full test suite**

```bash
go test -v -race ./...
```
Expected: ALL PASS

**Step 5: Commit**

```bash
git add -A
git commit -m "refactor: remove duplicate root files

Root now contains only:
- client.go (thin wrapper)
- facade.go (re-exports)
- doc.go
- version.go
- test files

All implementation moved to pkg/.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 10: Final Verification

**Step 1: Run full test suite with coverage**

```bash
go test -v -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out | tail -5
```
Expected: Coverage ≥ 68.5%

**Step 2: Verify build**

```bash
go build -v ./...
```
Expected: Success

**Step 3: Verify examples build**

```bash
go build -v ./examples/...
```
Expected: Success

**Step 4: Count root files**

```bash
ls *.go | wc -l
```
Expected: ~5 files (client.go, facade.go, doc.go, version.go, compat_test.go)

**Step 5: Final commit**

```bash
git add -A
git commit -m "chore: complete pkg/ restructure

Final state:
- Root: thin facade (~5 files)
- pkg/: all implementation (~37 files)
- 100% backward compatibility
- All tests passing

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Summary

| Phase | Tasks | Description |
|-------|-------|-------------|
| Phase 1 | Tasks 1-4 | Infrastructure packages (errors, config, http, ingestion) |
| Phase 2 | Tasks 5-7 | Domain packages (tracing, evaluation, resources) |
| Phase 3 | Tasks 8-10 | Root facade and cleanup |

**Total: 10 Tasks, ~10 Commits**

## Success Criteria

- [ ] pkg/errors created with all error types
- [ ] pkg/config expanded with full configuration
- [ ] pkg/http created with HTTP client
- [ ] pkg/ingestion created with queue/batch
- [ ] pkg/tracing created with trace types
- [ ] pkg/evaluation created with score types
- [ ] pkg/resources created with API clients
- [ ] Root facade re-exports all public types
- [ ] Duplicate root files removed
- [ ] All tests pass
- [ ] Coverage ≥ 68.5%
- [ ] Examples build
