# SDK Consolidation & OTel Integration Proposal

## Executive Summary

The langfuse-go SDK has accumulated technical debt that undermines its reliability and usability. This proposal outlines a consolidation effort to fix critical issues, properly integrate OpenTelemetry support, and establish a maintainable architecture.

**Current State:**
- Main package build is broken
- OTel package duplicates types instead of reusing them
- Test coverage is insufficient
- "Zero dependencies" claim is false

**Target State:**
- Single, coherent API surface
- OTel export as a first-class client option
- >80% test coverage
- Honest dependency claims

---

## 1. Critical Issues

### 1.1 Build Failure

**Problem:** The main package fails to build due to unresolved dependencies in `contrib/prometheus`.

**Root Cause:** `contrib/prometheus/metrics.go` imports `github.com/prometheus/client_golang/prometheus` but the dependency isn't in `go.mod`.

**Fix Options:**

| Option | Effort | Risk |
|--------|--------|------|
| A. Add prometheus to go.mod | Low | Breaks "zero deps" claim |
| B. Move prometheus to separate module | Medium | Requires multi-module setup |
| C. Delete contrib/prometheus | Low | Loses functionality |

**Recommendation:** Option B - Create `github.com/jdziat/langfuse-go/contrib/prometheus` as a separate Go module with its own `go.mod`. This keeps the core SDK dependency-free while offering optional integrations.

### 1.2 Type Duplication

**Problem:** Core types are duplicated between packages:

```go
// types.go
type ObservationType string
type ObservationLevel string

// otel/bridge.go
type ObservationType string   // DUPLICATE
type ObservationLevel string  // DUPLICATE
```

**Impact:**
- `langfuse.ObservationType` and `otel.ObservationType` are incompatible types
- Users must convert between them
- Bug fixes must be applied twice
- API is confusing

**Fix:** The OTel package should import and use types from the main package:

```go
// otel/bridge.go
package otel

import "github.com/jdziat/langfuse-go"

// Use langfuse.ObservationType directly
type LangfuseObservation struct {
    Type langfuse.ObservationType  // Reuse, don't duplicate
    // ...
}
```

### 1.3 OTel Integration is Disconnected

**Problem:** The OTel package is standalone and doesn't integrate with the main `Client`.

**Current architecture:**
```
┌─────────────┐     ┌─────────────┐
│   Client    │     │  otel.      │
│  (langfuse) │     │  Exporter   │
└──────┬──────┘     └──────┬──────┘
       │                   │
       ▼                   ▼
   Langfuse API       OTLP Collector
```

Users must manually wire both systems together.

**Target architecture:**
```
┌─────────────────────────────────┐
│            Client               │
│  ┌─────────┐    ┌───────────┐  │
│  │Langfuse │    │   OTel    │  │
│  │Exporter │    │  Exporter │  │
│  └────┬────┘    └─────┬─────┘  │
└───────┼───────────────┼────────┘
        │               │
        ▼               ▼
   Langfuse API    OTLP Collector
```

**Fix:** Add OTel configuration directly to the main `Client`.

---

## 2. Proposed Changes

### 2.1 Module Structure

```
langfuse-go/
├── go.mod                    # Core SDK (zero deps)
├── client.go
├── types.go                  # Canonical type definitions
├── otel/                     # OTel support (same module)
│   ├── exporter.go
│   ├── bridge.go            # Uses types from parent
│   └── propagation.go
├── contrib/
│   └── prometheus/
│       ├── go.mod           # Separate module
│       └── metrics.go
└── contrib/
    └── slog/
        ├── go.mod           # Separate module
        └── adapter.go
```

### 2.2 Unified Type System

Move all shared types to a single location and have OTel import them:

```go
// types.go (main package) - CANONICAL DEFINITIONS
package langfuse

type ObservationType string
const (
    ObservationTypeSpan       ObservationType = "SPAN"
    ObservationTypeGeneration ObservationType = "GENERATION"
    ObservationTypeEvent      ObservationType = "EVENT"
)

type ObservationLevel string
const (
    ObservationLevelDebug   ObservationLevel = "DEBUG"
    ObservationLevelDefault ObservationLevel = "DEFAULT"
    ObservationLevelWarning ObservationLevel = "WARNING"
    ObservationLevelError   ObservationLevel = "ERROR"
)

// Observation is the unified observation type
type Observation struct {
    ID                  string
    TraceID             string
    ParentObservationID string
    Type                ObservationType
    Name                string
    StartTime           time.Time
    EndTime             *time.Time
    Input               interface{}
    Output              interface{}
    Metadata            map[string]interface{}
    Level               ObservationLevel
    StatusMessage       string

    // Generation-specific
    Model           string
    ModelParameters map[string]interface{}
    Usage           *Usage
}
```

```go
// otel/bridge.go - IMPORTS FROM PARENT
package otel

import (
    "github.com/jdziat/langfuse-go"
)

// ToOTelSpan converts a Langfuse observation to an OTel span
func ToOTelSpan(obs *langfuse.Observation) (*Span, error) {
    span := &Span{
        Name: obs.Name,
        Kind: mapObservationTypeToSpanKind(obs.Type), // obs.Type is langfuse.ObservationType
        // ...
    }
    return span, nil
}

func mapObservationTypeToSpanKind(t langfuse.ObservationType) SpanKind {
    switch t {
    case langfuse.ObservationTypeGeneration:
        return SpanKindClient
    case langfuse.ObservationTypeEvent:
        return SpanKindInternal
    default:
        return SpanKindInternal
    }
}
```

### 2.3 Integrated Client API

Add OTel configuration directly to the main client:

```go
// config.go
package langfuse

// OTelConfig configures OpenTelemetry export
type OTelConfig struct {
    Enabled        bool
    Endpoint       string
    Headers        map[string]string
    ServiceName    string
    ServiceVersion string
}

// WithOTelExport enables dual export to an OTLP collector
func WithOTelExport(cfg OTelConfig) ConfigOption {
    return func(c *Config) {
        c.OTel = &cfg
    }
}
```

```go
// client.go
package langfuse

import "github.com/jdziat/langfuse-go/otel"

type Client struct {
    // ... existing fields

    otelExporter *otel.Exporter  // nil if OTel disabled
}

func New(publicKey, secretKey string, opts ...ConfigOption) (*Client, error) {
    // ... existing initialization

    // Initialize OTel exporter if configured
    if cfg.OTel != nil && cfg.OTel.Enabled {
        otelCfg := otel.NewOTelConfig(
            otel.OTelEnabled(true),
            otel.OTelEndpoint(cfg.OTel.Endpoint),
            otel.OTelServiceName(cfg.OTel.ServiceName),
            otel.OTelHeaders(cfg.OTel.Headers),
        )
        exporter, err := otelCfg.BuildExporter()
        if err != nil {
            return nil, fmt.Errorf("langfuse: failed to create OTel exporter: %w", err)
        }
        exporter.Start()
        c.otelExporter = exporter
    }

    return c, nil
}
```

```go
// ingestion.go - Dual export in queueEvent
func (c *Client) queueEvent(ctx context.Context, event ingestionEvent) error {
    // ... existing Langfuse queueing

    // Also export to OTel if configured
    if c.otelExporter != nil {
        if obs := eventToObservation(event); obs != nil {
            span, err := otel.ToOTelSpan(obs)
            if err == nil {
                c.otelExporter.AddSpanContext(ctx, span)
            }
        }
    }

    return nil
}
```

**User-facing API:**

```go
// Simple: Just Langfuse
client, _ := langfuse.New(pk, sk)

// With OTel: Dual export
client, _ := langfuse.New(pk, sk,
    langfuse.WithOTelExport(langfuse.OTelConfig{
        Enabled:     true,
        Endpoint:    "http://localhost:4318/v1/traces",
        ServiceName: "my-app",
    }),
)

// Everything works automatically - no manual wiring
trace, _ := client.NewTrace().Name("request").Create()
gen, _ := trace.Generation().Model("gpt-4").Create()
gen.End()  // Exported to BOTH Langfuse and OTel automatically
```

### 2.4 Context Propagation Integration

Add trace context helpers directly to `Client`:

```go
// client.go
package langfuse

import (
    "net/http"
    "github.com/jdziat/langfuse-go/otel"
)

// TraceContextFromRequest extracts W3C trace context from HTTP headers
func (c *Client) TraceContextFromRequest(r *http.Request) *otel.TraceContext {
    ctx, tc := otel.ExtractTraceContext(r.Context(), r.Header)
    // Store in request context for later use
    *r = *r.WithContext(ctx)
    return tc
}

// NewTraceWithContext creates a trace linked to an upstream trace context
func (c *Client) NewTraceWithContext(tc *otel.TraceContext) *TraceBuilder {
    builder := c.NewTrace()
    if tc != nil {
        builder.metadata["_traceparent"] = tc.Traceparent()
        builder.metadata["_tracestate"] = tc.TraceState
    }
    return builder
}

// InjectTraceContext adds trace context headers to an outgoing request
func (t *Trace) InjectTraceContext(header http.Header) {
    if tc := t.traceContext; tc != nil {
        otel.InjectTraceContext(context.Background(), header)
    }
}
```

---

## 3. Migration Path

### Phase 1: Fix Build (1 day)

1. Move `contrib/prometheus` to separate module
2. Verify `go build ./...` passes
3. Add CI check to prevent future breakage

### Phase 2: Type Consolidation (2-3 days)

1. Remove duplicate types from `otel/bridge.go`
2. Update OTel package to import from main package
3. Add type aliases for backwards compatibility if needed:
   ```go
   // otel/compat.go (temporary)
   package otel

   import "github.com/jdziat/langfuse-go"

   // Deprecated: Use langfuse.ObservationType instead
   type ObservationType = langfuse.ObservationType
   ```

### Phase 3: Client Integration (3-5 days)

1. Add `OTelConfig` to `Config`
2. Add `WithOTelExport()` option
3. Initialize OTel exporter in `New()`
4. Add dual export to `queueEvent()`
5. Add `Shutdown()` cleanup for OTel exporter
6. Add context propagation helpers

### Phase 4: Testing (2-3 days)

1. Fix main package tests
2. Add integration tests for dual export
3. Achieve >80% coverage on core paths
4. Add CI coverage gates

### Phase 5: Documentation (1-2 days)

1. Update README with OTel usage
2. Add migration guide from standalone otel package
3. Document breaking changes

---

## 4. Breaking Changes

### 4.1 Required Changes

| Change | Impact | Mitigation |
|--------|--------|------------|
| `otel.ObservationType` removed | Code using OTel types breaks | Type alias for 1 release |
| `otel.LangfuseObservation` changes | Field types change | Use `langfuse.Observation` |
| `contrib/prometheus` moves | Import path changes | Document new import |

### 4.2 Deprecation Timeline

```
v0.X.0 - Add type aliases, mark old types deprecated
v0.X+1.0 - Remove type aliases, require new types
v1.0.0 - Stable API
```

---

## 5. Success Criteria

| Metric | Current | Target |
|--------|---------|--------|
| Build status | FAIL | PASS |
| Main package coverage | Unknown | >80% |
| OTel package coverage | 83% | >85% |
| Duplicate type definitions | 4 | 0 |
| Manual wiring required | Yes | No |
| API surface (public types) | ~50 | ~35 |

---

## 6. Alternatives Considered

### 6.1 Keep OTel Separate

**Pros:** No breaking changes
**Cons:** Permanent architectural debt, confusing API, duplicate maintenance

**Decision:** Rejected. The cost of maintaining parallel systems exceeds the cost of one-time migration.

### 6.2 Full Rewrite

**Pros:** Clean slate
**Cons:** High effort, abandons working code, delays delivery

**Decision:** Rejected. Incremental consolidation achieves goals with lower risk.

### 6.3 Deprecate OTel Package

**Pros:** Simplest option
**Cons:** Loses OTel functionality users may depend on

**Decision:** Rejected. OTel support is valuable; integrate it properly instead.

---

## 7. Implementation Order

```
Week 1:
├── Day 1: Fix build (Phase 1)
├── Day 2-3: Type consolidation (Phase 2)
└── Day 4-5: Begin client integration (Phase 3)

Week 2:
├── Day 1-2: Complete client integration
├── Day 3-4: Testing (Phase 4)
└── Day 5: Documentation (Phase 5)
```

---

## 8. Appendix: File Changes

### Files to Modify

| File | Change |
|------|--------|
| `go.mod` | Remove prometheus (moved to contrib) |
| `config.go` | Add `OTelConfig`, `WithOTelExport()` |
| `client.go` | Add `otelExporter` field, init logic |
| `ingestion.go` | Add dual export in `queueEvent()` |
| `otel/bridge.go` | Remove duplicate types, import from parent |
| `otel/config.go` | Simplify (client handles integration) |

### Files to Add

| File | Purpose |
|------|---------|
| `contrib/prometheus/go.mod` | Separate module for prometheus |
| `otel/compat.go` | Temporary type aliases |

### Files to Remove

| File | Reason |
|------|--------|
| `otel/integration.go` | Functionality moves to client.go |
| Duplicate type definitions | Consolidated in types.go |

---

## 9. Open Questions

1. **Versioning:** Should this be a minor or major version bump?
   - Recommendation: Minor with deprecation warnings, major for final removal

2. **Feature flags:** Should OTel be opt-in or opt-out?
   - Recommendation: Opt-in via `WithOTelExport()` for backwards compatibility

3. **Error handling:** If OTel export fails, should Langfuse export continue?
   - Recommendation: Yes, log error but don't fail Langfuse export
