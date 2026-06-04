# Langfuse Go SDK Complete Refactoring Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Refactor SDK into layered pkg/ structure with facade, update docs, add GitHub Pages documentation, ensure tests pass, and enable Dependabot auto-merge.

**Architecture:** Reorganize flat 82-file root package into layered structure: thin facade at root re-exporting public API, internal packages under pkg/ for config, ingestion, api, builders, and errors. Add comprehensive documentation site using GitHub Pages with MkDocs or Hugo.

**Tech Stack:** Go 1.23+, GitHub Actions, GitHub Pages, Dependabot

---

## Phase 0: Foundation & Dead Code Cleanup

### Task 1: Remove Dead Code from ingestion.go

**Files:**
- Modify: `ingestion.go:97-106`
- Test: Run existing tests to verify no regression

**Step 1: Read and identify dead code**

Read `ingestion.go` lines 97-106 containing unused type aliases.

**Step 2: Delete unused type aliases**

Remove the following block from `ingestion.go`:
```go
type (
    createTraceEvent      = traceEvent
    updateTraceEvent      = traceEvent
    createSpanEvent       = observationEvent
    updateSpanEvent       = observationEvent
    createGenerationEvent = observationEvent
    updateGenerationEvent = observationEvent
    createEventEvent      = observationEvent
    sdkLogEvent           = sdkLogEventType
)
```

**Step 3: Run tests to verify no regression**

Run: `go test -v ./... -count=1`
Expected: All tests PASS

**Step 4: Run static analysis**

Run: `go vet ./... && staticcheck ./...`
Expected: No errors

**Step 5: Commit**

```bash
git add ingestion.go
git commit -m "refactor: remove unused type aliases from ingestion.go

Removes dead code identified in SDK refactoring proposal.
These type aliases (createTraceEvent, updateTraceEvent, etc.)
were never used in the codebase.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 2: Add Missing Critical Tests

**Files:**
- Modify: `client_test.go`
- Test: `client_test.go`

**Step 2.1: Add test for handleError**

Add test for async error handling:
```go
func TestClient_HandleError(t *testing.T) {
    // Test that handleError properly invokes error callback
    var capturedErr error
    client, _ := newTestClientWithOptions(t,
        WithAsyncErrorHandler(func(err error) {
            capturedErr = err
        }),
    )
    defer client.Shutdown(context.Background())

    testErr := errors.New("test async error")
    client.handleError(testErr)

    // Allow async processing
    time.Sleep(50 * time.Millisecond)

    if capturedErr == nil {
        t.Error("expected error to be captured by handler")
    }
    if capturedErr.Error() != testErr.Error() {
        t.Errorf("expected %v, got %v", testErr, capturedErr)
    }
}
```

**Step 2.2: Run test to verify it passes**

Run: `go test -v -run TestClient_HandleError ./...`
Expected: PASS

**Step 2.3: Add test for handleQueueFull**

```go
func TestClient_HandleQueueFull(t *testing.T) {
    var queueFullCalled bool
    client, _ := newTestClientWithOptions(t,
        WithQueueSize(1),
        WithQueueFullHandler(func() {
            queueFullCalled = true
        }),
    )
    defer client.Shutdown(context.Background())

    // Fill the queue to trigger overflow
    ctx := context.Background()
    for i := 0; i < 100; i++ {
        client.NewTrace().Name(fmt.Sprintf("trace-%d", i)).Create(ctx)
    }

    time.Sleep(100 * time.Millisecond)

    if !queueFullCalled {
        t.Error("expected queue full handler to be called")
    }
}
```

**Step 2.4: Run test to verify it passes**

Run: `go test -v -run TestClient_HandleQueueFull ./...`
Expected: PASS

**Step 2.5: Add test for CircuitBreakerState**

```go
func TestClient_CircuitBreakerState(t *testing.T) {
    client, _ := newTestClient(t)
    defer client.Shutdown(context.Background())

    state := client.CircuitBreakerState()

    // Initial state should be closed
    if state != CircuitBreakerStateClosed {
        t.Errorf("expected CircuitBreakerStateClosed, got %v", state)
    }
}
```

**Step 2.6: Run test to verify it passes**

Run: `go test -v -run TestClient_CircuitBreakerState ./...`
Expected: PASS

**Step 2.7: Add test for drainAllEvents**

```go
func TestClient_DrainAllEvents(t *testing.T) {
    ctx := context.Background()
    client, server := newTestClient(t)

    // Create several traces
    for i := 0; i < 10; i++ {
        client.NewTrace().Name(fmt.Sprintf("drain-test-%d", i)).Create(ctx)
    }

    // Shutdown should drain all events
    err := client.Shutdown(ctx)
    if err != nil {
        t.Fatalf("shutdown failed: %v", err)
    }

    // Verify events were sent
    requests := server.Requests()
    if len(requests) == 0 {
        t.Error("expected events to be drained and sent")
    }
}
```

**Step 2.8: Run all new tests**

Run: `go test -v -run "TestClient_Handle|TestClient_CircuitBreaker|TestClient_Drain" ./...`
Expected: All PASS

**Step 2.9: Check coverage improvement**

Run: `go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | grep -E "(handleError|handleQueueFull|CircuitBreakerState|drainAllEvents)"`
Expected: Coverage increased for these functions

**Step 2.10: Commit**

```bash
git add client_test.go
git commit -m "test: add tests for critical async error handling paths

Adds tests for:
- handleError: async error callback invocation
- handleQueueFull: queue overflow handler
- CircuitBreakerState: circuit breaker state getter
- drainAllEvents: shutdown event draining

Addresses P0 test coverage gaps from refactoring proposal.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Phase 1: Architecture Improvements - File Splitting

### Task 3: Split client.go into focused modules

**Files:**
- Modify: `client.go` (reduce to ~400 lines)
- Create: `lifecycle.go` (~200 lines)
- Create: `batching.go` (~200 lines)
- Create: `queue.go` (~200 lines)
- Test: Run all tests

**Step 3.1: Read current client.go structure**

Read `client.go` to understand current organization.

**Step 3.2: Create lifecycle.go**

Extract lifecycle management methods:
- `Flush()`
- `Shutdown()`
- `Close()`
- `IsShutdown()`
- `drainAllEvents()`
- Health check methods

**Step 3.3: Create batching.go**

Extract batching logic:
- `sendBatch()`
- `processBatch()`
- Batch configuration
- Batch event types

**Step 3.4: Create queue.go**

Extract queue management:
- `enqueueEvent()`
- `handleQueueFull()`
- Queue worker goroutine
- Queue monitoring

**Step 3.5: Verify client.go is under 500 lines**

Run: `wc -l client.go`
Expected: < 500 lines

**Step 3.6: Run tests**

Run: `go test -v ./... -count=1`
Expected: All PASS

**Step 3.7: Commit**

```bash
git add client.go lifecycle.go batching.go queue.go
git commit -m "refactor: split client.go into focused modules

Extracts:
- lifecycle.go: Flush, Shutdown, Close, health checks
- batching.go: batch processing and sending logic
- queue.go: event queue management and monitoring

Reduces client.go from 1168 to ~400 lines.
All modules remain in root package for API compatibility.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 4: Split errors.go into focused modules

**Files:**
- Modify: `errors.go` (reduce to ~200 lines - core errors only)
- Create: `errors_api.go` (~200 lines)
- Create: `errors_async.go` (~150 lines)
- Create: `errors_validation.go` (~100 lines)
- Create: `errors_helpers.go` (~150 lines)
- Test: Run all tests

**Step 4.1: Read current errors.go structure**

Read `errors.go` to understand current organization.

**Step 4.2: Create errors_api.go**

Extract API error types:
- `APIError`
- `APIErrorResponse`
- `APIErrorDetail`
- Related methods

**Step 4.3: Create errors_async.go**

Extract async error types:
- `AsyncError`
- `BatchError`
- Event error types

**Step 4.4: Create errors_validation.go**

Extract validation errors:
- `ValidationError`
- Field validation helpers

**Step 4.5: Create errors_helpers.go**

Extract error helper functions:
- `IsAPIError()` / `AsAPIError()`
- `IsValidationError()` / `AsValidationError()`
- Error wrapping utilities

**Step 4.6: Verify errors.go is under 300 lines**

Run: `wc -l errors.go`
Expected: < 300 lines

**Step 4.7: Run tests**

Run: `go test -v ./... -count=1`
Expected: All PASS

**Step 4.8: Commit**

```bash
git add errors.go errors_api.go errors_async.go errors_validation.go errors_helpers.go
git commit -m "refactor: split errors.go into focused modules

Extracts:
- errors_api.go: APIError and HTTP error types
- errors_async.go: async and batch error types
- errors_validation.go: validation error types
- errors_helpers.go: Is*/As* helper functions

Reduces errors.go from 897 to ~200 lines.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 5: Standardize error helpers to Go conventions

**Files:**
- Modify: `errors_helpers.go`
- Modify: `README.md` (update examples)
- Test: `errors_test.go`

**Step 5.1: Add As* functions alongside Is* functions**

Add standard Go convention functions:
```go
// AsAPIError extracts an APIError from err if present.
// This follows Go's standard errors.As() convention.
func AsAPIError(err error) (*APIError, bool) {
    var apiErr *APIError
    if errors.As(err, &apiErr) {
        return apiErr, true
    }
    return nil, false
}

// AsValidationError extracts a ValidationError from err if present.
func AsValidationError(err error) (*ValidationError, bool) {
    var valErr *ValidationError
    if errors.As(err, &valErr) {
        return valErr, true
    }
    return nil, false
}
```

**Step 5.2: Deprecate Is* extraction functions**

Add deprecation notices:
```go
// Deprecated: Use AsAPIError instead. IsAPIError returns extraction semantics
// but uses Is* naming which conventionally returns bool only.
func IsAPIError(err error) (*APIError, bool) {
    return AsAPIError(err)
}
```

**Step 5.3: Add tests for new As* functions**

```go
func TestAsAPIError(t *testing.T) {
    apiErr := &APIError{StatusCode: 400, Message: "bad request"}
    wrappedErr := fmt.Errorf("wrapped: %w", apiErr)

    extracted, ok := AsAPIError(wrappedErr)
    if !ok {
        t.Fatal("expected to extract APIError")
    }
    if extracted.StatusCode != 400 {
        t.Errorf("expected status 400, got %d", extracted.StatusCode)
    }
}
```

**Step 5.4: Run tests**

Run: `go test -v -run "TestAs" ./...`
Expected: All PASS

**Step 5.5: Commit**

```bash
git add errors_helpers.go errors_test.go
git commit -m "refactor: add Go-conventional As* error extraction functions

Adds:
- AsAPIError(): extracts APIError following errors.As() convention
- AsValidationError(): extracts ValidationError
- Deprecation notices on Is* functions that return (*T, bool)

Maintains backward compatibility while encouraging Go conventions.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 6: Add WithOptions patterns to Sessions and Models

**Files:**
- Modify: `sessions.go`
- Modify: `models.go`
- Create: `sessions_options.go`
- Create: `models_options.go`
- Test: Add tests for new patterns

**Step 6.1: Create sessions_options.go**

```go
// SessionsOption configures the sessions client.
type SessionsOption func(*sessionsConfig)

type sessionsConfig struct {
    timeout time.Duration
    retries int
}

// WithSessionsTimeout sets request timeout for sessions operations.
func WithSessionsTimeout(d time.Duration) SessionsOption {
    return func(c *sessionsConfig) {
        c.timeout = d
    }
}

// ConfiguredSessionsClient wraps SessionsClient with pre-configured options.
type ConfiguredSessionsClient struct {
    client *SessionsClient
    config sessionsConfig
}
```

**Step 6.2: Add SessionsWithOptions to client**

Add to `client.go`:
```go
// SessionsWithOptions returns a configured sessions client.
func (c *Client) SessionsWithOptions(opts ...SessionsOption) *ConfiguredSessionsClient {
    cfg := sessionsConfig{
        timeout: 30 * time.Second,
        retries: 3,
    }
    for _, opt := range opts {
        opt(&cfg)
    }
    return &ConfiguredSessionsClient{
        client: c.Sessions(),
        config: cfg,
    }
}
```

**Step 6.3: Create models_options.go**

Similar pattern for Models client.

**Step 6.4: Add ModelsWithOptions to client**

Similar pattern for Models.

**Step 6.5: Add tests**

```go
func TestSessionsWithOptions(t *testing.T) {
    client, _ := newTestClient(t)
    defer client.Shutdown(context.Background())

    configured := client.SessionsWithOptions(
        WithSessionsTimeout(10 * time.Second),
    )

    if configured == nil {
        t.Fatal("expected configured client")
    }
}
```

**Step 6.6: Run tests**

Run: `go test -v -run "TestSessionsWithOptions|TestModelsWithOptions" ./...`
Expected: All PASS

**Step 6.7: Commit**

```bash
git add sessions.go models.go sessions_options.go models_options.go client.go *_test.go
git commit -m "feat: add WithOptions pattern to Sessions and Models clients

Adds consistent API patterns:
- client.SessionsWithOptions(opts...) -> ConfiguredSessionsClient
- client.ModelsWithOptions(opts...) -> ConfiguredModelsClient

Matches existing pattern from PromptsWithOptions.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Phase 2: API Enhancements

### Task 7: Enhance Metadata type with utility methods

**Files:**
- Modify: `types.go`
- Create: `metadata.go`
- Test: `metadata_test.go`

**Step 7.1: Create metadata.go with utility methods**

```go
package langfuse

// Get retrieves a value from metadata by key.
func (m Metadata) Get(key string) (any, bool) {
    if m == nil {
        return nil, false
    }
    v, ok := m[key]
    return v, ok
}

// GetString retrieves a string value, returning empty string if not found or wrong type.
func (m Metadata) GetString(key string) string {
    v, ok := m.Get(key)
    if !ok {
        return ""
    }
    s, _ := v.(string)
    return s
}

// GetInt retrieves an int value, returning 0 if not found or wrong type.
func (m Metadata) GetInt(key string) int {
    v, ok := m.Get(key)
    if !ok {
        return 0
    }
    switch n := v.(type) {
    case int:
        return n
    case int64:
        return int(n)
    case float64:
        return int(n)
    default:
        return 0
    }
}

// GetBool retrieves a bool value, returning false if not found or wrong type.
func (m Metadata) GetBool(key string) bool {
    v, ok := m.Get(key)
    if !ok {
        return false
    }
    b, _ := v.(bool)
    return b
}

// Merge combines this metadata with another, with other taking precedence.
func (m Metadata) Merge(other Metadata) Metadata {
    result := make(Metadata, len(m)+len(other))
    for k, v := range m {
        result[k] = v
    }
    for k, v := range other {
        result[k] = v
    }
    return result
}

// Filter returns a new Metadata containing only the specified keys.
func (m Metadata) Filter(keys ...string) Metadata {
    result := make(Metadata, len(keys))
    for _, k := range keys {
        if v, ok := m[k]; ok {
            result[k] = v
        }
    }
    return result
}

// Keys returns all keys in the metadata.
func (m Metadata) Keys() []string {
    keys := make([]string, 0, len(m))
    for k := range m {
        keys = append(keys, k)
    }
    return keys
}
```

**Step 7.2: Create metadata_test.go**

```go
func TestMetadata_Get(t *testing.T) {
    m := Metadata{"key": "value", "num": 42}

    v, ok := m.Get("key")
    if !ok || v != "value" {
        t.Errorf("expected 'value', got %v", v)
    }

    _, ok = m.Get("missing")
    if ok {
        t.Error("expected missing key to return false")
    }
}

func TestMetadata_GetString(t *testing.T) {
    m := Metadata{"str": "hello", "num": 42}

    if s := m.GetString("str"); s != "hello" {
        t.Errorf("expected 'hello', got %s", s)
    }
    if s := m.GetString("num"); s != "" {
        t.Errorf("expected empty string for non-string, got %s", s)
    }
}

func TestMetadata_Merge(t *testing.T) {
    m1 := Metadata{"a": 1, "b": 2}
    m2 := Metadata{"b": 3, "c": 4}

    merged := m1.Merge(m2)

    if merged["a"] != 1 || merged["b"] != 3 || merged["c"] != 4 {
        t.Errorf("unexpected merge result: %v", merged)
    }
}

func TestMetadata_Filter(t *testing.T) {
    m := Metadata{"a": 1, "b": 2, "c": 3}

    filtered := m.Filter("a", "c")

    if len(filtered) != 2 || filtered["a"] != 1 || filtered["c"] != 3 {
        t.Errorf("unexpected filter result: %v", filtered)
    }
}
```

**Step 7.3: Run tests**

Run: `go test -v -run "TestMetadata" ./...`
Expected: All PASS

**Step 7.4: Commit**

```bash
git add metadata.go metadata_test.go
git commit -m "feat: add utility methods to Metadata type

Adds convenience methods:
- Get(key): retrieves value with ok bool
- GetString/GetInt/GetBool: type-safe getters
- Merge(other): combines metadata maps
- Filter(keys...): extracts subset of keys
- Keys(): returns all keys

Reduces boilerplate for common metadata operations.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Phase 3: Package Restructuring

### Task 8: Create pkg/ directory structure

**Files:**
- Create: `pkg/config/config.go`
- Create: `pkg/config/options.go`
- Create: `pkg/errors/errors.go`
- Create: `pkg/errors/api.go`
- Create: `pkg/ingestion/queue.go`
- Create: `pkg/ingestion/batch.go`
- Create: `pkg/api/client.go`
- Create: `pkg/builders/trace.go`
- Modify: Root files to re-export

**Step 8.1: Create pkg/config package**

Create `pkg/config/config.go`:
```go
// Package config provides configuration types for the Langfuse SDK.
package config

import "time"

// Config holds all configuration for the Langfuse client.
type Config struct {
    PublicKey     string
    SecretKey     string
    BaseURL       string
    Region        Region

    // Batching configuration
    Batching BatchingConfig

    // Network configuration
    Network NetworkConfig

    // Feature flags
    Features FeaturesConfig
}

// BatchingConfig controls event batching behavior.
type BatchingConfig struct {
    Size          int
    FlushInterval time.Duration
    QueueSize     int
}

// NetworkConfig controls HTTP client behavior.
type NetworkConfig struct {
    Timeout     time.Duration
    MaxRetries  int
    RetryDelay  time.Duration
}

// FeaturesConfig enables/disables optional features.
type FeaturesConfig struct {
    CircuitBreaker bool
    Metrics        bool
    Debug          bool
}
```

**Step 8.2: Create pkg/errors package**

Create `pkg/errors/errors.go` with core error types.

**Step 8.3: Create pkg/ingestion package**

Create `pkg/ingestion/queue.go` and `pkg/ingestion/batch.go`.

**Step 8.4: Create pkg/api package**

Create `pkg/api/client.go` with HTTP client logic.

**Step 8.5: Create pkg/builders package**

Create `pkg/builders/trace.go` with builder types.

**Step 8.6: Update root package to re-export**

Add type aliases and re-exports in root for backward compatibility:
```go
// Re-exports from pkg/config for backward compatibility
type Config = config.Config
type BatchingConfig = config.BatchingConfig
```

**Step 8.7: Run tests**

Run: `go test -v ./... -count=1`
Expected: All PASS

**Step 8.8: Commit**

```bash
git add pkg/ *.go
git commit -m "refactor: create layered pkg/ structure

Introduces organized package structure:
- pkg/config: configuration types and options
- pkg/errors: error types and helpers
- pkg/ingestion: event queue and batching
- pkg/api: HTTP client implementation
- pkg/builders: fluent builder types

Root package re-exports all public types for backward compatibility.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Phase 4: Documentation

### Task 9: Update README.md

**Files:**
- Modify: `README.md`

**Step 9.1: Update README with new features**

Update to include:
- New package structure documentation
- Enhanced Metadata API examples
- New error handling patterns with As* functions
- WithOptions patterns for all sub-clients
- Evaluation workflow examples
- LLM-as-a-Judge integration

**Step 9.2: Add architecture diagram**

Add ASCII diagram of new package structure.

**Step 9.3: Update code examples**

Update all examples to use new patterns.

**Step 9.4: Commit**

```bash
git add README.md
git commit -m "docs: update README with new SDK features and patterns

Updates documentation for:
- New layered package structure
- Enhanced Metadata utility methods
- Go-conventional As* error helpers
- WithOptions patterns for all sub-clients
- Evaluation workflows and LLM-as-a-Judge

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 10: Create GitHub Pages documentation

**Files:**
- Create: `docs/index.md`
- Create: `docs/getting-started.md`
- Create: `docs/configuration.md`
- Create: `docs/tracing.md`
- Create: `docs/evaluation.md`
- Create: `docs/api-reference.md`
- Create: `docs/migration.md`
- Create: `mkdocs.yml`
- Create: `.github/workflows/docs.yml`

**Step 10.1: Create mkdocs.yml**

```yaml
site_name: Langfuse Go SDK
site_url: https://jdziat.github.io/langfuse-go
repo_url: https://github.com/jdziat/langfuse-go
repo_name: jdziat/langfuse-go

theme:
  name: material
  palette:
    - scheme: default
      primary: indigo
      accent: indigo
  features:
    - navigation.tabs
    - navigation.sections
    - navigation.expand
    - search.suggest
    - content.code.copy

nav:
  - Home: index.md
  - Getting Started: getting-started.md
  - Configuration: configuration.md
  - Tracing:
    - Overview: tracing/overview.md
    - Traces: tracing/traces.md
    - Spans: tracing/spans.md
    - Generations: tracing/generations.md
  - Evaluation:
    - Overview: evaluation/overview.md
    - LLM-as-a-Judge: evaluation/llm-as-judge.md
    - Workflows: evaluation/workflows.md
  - API Reference: api-reference.md
  - Migration Guide: migration.md

markdown_extensions:
  - pymdownx.highlight:
      anchor_linenums: true
  - pymdownx.superfences
  - admonition
  - tables
```

**Step 10.2: Create docs/index.md**

```markdown
# Langfuse Go SDK

Welcome to the official documentation for the Langfuse Go SDK.

## Features

- **Zero Dependencies**: Pure Go implementation
- **Type-Safe API**: Strongly typed interfaces
- **Automatic Batching**: Efficient event batching
- **Concurrent-Safe**: Thread-safe operations
- **Full API Coverage**: Traces, spans, generations, scores, prompts, datasets

## Quick Start

```go
client, _ := langfuse.New(
    os.Getenv("LANGFUSE_PUBLIC_KEY"),
    os.Getenv("LANGFUSE_SECRET_KEY"),
)
defer client.Shutdown(context.Background())

trace, _ := client.NewTrace().
    Name("my-trace").
    Create(context.Background())
```

## Installation

```bash
go get github.com/jdziat/langfuse-go
```
```

**Step 10.3: Create GitHub Actions workflow for docs**

Create `.github/workflows/docs.yml`:
```yaml
name: Deploy Documentation

on:
  push:
    branches: [main]
    paths:
      - 'docs/**'
      - 'mkdocs.yml'
  workflow_dispatch:

permissions:
  contents: read
  pages: write
  id-token: write

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Setup Python
        uses: actions/setup-python@v5
        with:
          python-version: '3.x'

      - name: Install MkDocs
        run: pip install mkdocs-material

      - name: Build docs
        run: mkdocs build

      - name: Upload artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: site/

  deploy:
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - name: Deploy to GitHub Pages
        id: deployment
        uses: actions/deploy-pages@v4
```

**Step 10.4: Create remaining documentation pages**

Create all documentation pages with comprehensive content.

**Step 10.5: Commit**

```bash
git add docs/ mkdocs.yml .github/workflows/docs.yml
git commit -m "docs: add GitHub Pages documentation site

Creates comprehensive documentation using MkDocs Material:
- Getting started guide
- Configuration reference
- Tracing documentation
- Evaluation workflows
- API reference
- Migration guide

Adds GitHub Actions workflow for automatic deployment.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 11: Enable Dependabot auto-merge

**Files:**
- Modify: `.github/dependabot.yml`
- Create: `.github/workflows/dependabot-auto-merge.yml`

**Step 11.1: Update dependabot.yml**

Add auto-merge labels:
```yaml
version: 2
updates:
  - package-ecosystem: "gomod"
    directory: "/"
    schedule:
      interval: "weekly"
    groups:
      minor-and-patch:
        patterns:
          - "*"
        update-types:
          - "minor"
          - "patch"
    labels:
      - "dependencies"
      - "automerge"

  - package-ecosystem: "github-actions"
    directory: "/"
    schedule:
      interval: "weekly"
    groups:
      actions:
        patterns:
          - "*"
    labels:
      - "dependencies"
      - "automerge"
```

**Step 11.2: Create auto-merge workflow**

Create `.github/workflows/dependabot-auto-merge.yml`:
```yaml
name: Dependabot Auto-Merge

on:
  pull_request:
    types: [opened, synchronize, reopened, labeled]

permissions:
  contents: write
  pull-requests: write

jobs:
  auto-merge:
    runs-on: ubuntu-latest
    if: github.actor == 'dependabot[bot]'

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Wait for CI
        uses: lewagon/wait-on-check-action@v1.3.4
        with:
          ref: ${{ github.event.pull_request.head.sha }}
          check-name: 'Test'
          repo-token: ${{ secrets.GITHUB_TOKEN }}
          wait-interval: 30

      - name: Wait for Lint
        uses: lewagon/wait-on-check-action@v1.3.4
        with:
          ref: ${{ github.event.pull_request.head.sha }}
          check-name: 'Lint'
          repo-token: ${{ secrets.GITHUB_TOKEN }}
          wait-interval: 30

      - name: Wait for Build
        uses: lewagon/wait-on-check-action@v1.3.4
        with:
          ref: ${{ github.event.pull_request.head.sha }}
          check-name: 'Build'
          repo-token: ${{ secrets.GITHUB_TOKEN }}
          wait-interval: 30

      - name: Auto-merge Dependabot PRs
        run: gh pr merge --auto --squash "$PR_URL"
        env:
          PR_URL: ${{ github.event.pull_request.html_url }}
          GH_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

**Step 11.3: Commit**

```bash
git add .github/dependabot.yml .github/workflows/dependabot-auto-merge.yml
git commit -m "ci: enable Dependabot auto-merge on successful tests

Configures Dependabot to:
- Group minor and patch updates
- Add automerge label to PRs
- Auto-merge after CI passes (Test, Lint, Build)

Reduces maintenance burden for dependency updates.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

### Task 12: Final verification and test coverage

**Files:**
- All modified files
- Test: Full test suite

**Step 12.1: Run full test suite**

Run: `go test -v -race -coverprofile=coverage.out ./...`
Expected: All PASS, coverage > 80%

**Step 12.2: Run linters**

Run: `go vet ./... && staticcheck ./...`
Expected: No errors

**Step 12.3: Run security scan**

Run: `govulncheck ./...`
Expected: No vulnerabilities

**Step 12.4: Build all packages**

Run: `go build -v ./...`
Expected: Success

**Step 12.5: Build examples**

Run: `go build -v ./examples/...`
Expected: Success

**Step 12.6: Generate final coverage report**

Run: `go tool cover -html=coverage.out -o coverage.html`

**Step 12.7: Commit any final fixes**

```bash
git add -A
git commit -m "chore: final verification and cleanup

- All tests passing
- Coverage > 80%
- No linter warnings
- No security vulnerabilities
- All packages build successfully

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>"
```

---

## Summary

| Phase | Tasks | Estimated Commits |
|-------|-------|-------------------|
| Phase 0: Foundation | Tasks 1-2 | 2 |
| Phase 1: Architecture | Tasks 3-6 | 4 |
| Phase 2: API Enhancements | Task 7 | 1 |
| Phase 3: Restructuring | Task 8 | 1 |
| Phase 4: Documentation | Tasks 9-12 | 4 |
| **Total** | **12 Tasks** | **12 Commits** |

## Success Criteria

- [ ] All dead code removed
- [ ] Test coverage > 85%
- [ ] All files < 500 lines
- [ ] All tests passing
- [ ] pkg/ structure implemented
- [ ] GitHub Pages docs live
- [ ] Dependabot auto-merge enabled
- [ ] README updated
