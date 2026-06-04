# Langfuse Go SDK Refactoring Proposal

## Executive Summary

This proposal outlines a comprehensive refactoring plan for the langfuse-go SDK to improve code maintainability, developer experience, and API consistency. After thorough analysis of the codebase, we've identified opportunities to reduce complexity, eliminate dead code, improve test coverage, and enhance the overall architecture while maintaining backward compatibility where possible.

The refactoring focuses on four key areas:
1. **Immediate cleanup** of dead code and fixing documented technical debt (P0)
2. **Architecture improvements** to reduce complexity and improve consistency (P1)
3. **API enhancements** for better developer experience (P2)
4. **Long-term structural improvements** for maintainability (P3)

## Current State Analysis

### Strengths
- **Zero dependencies**: Pure Go implementation with stdlib only
- **Type-safe API**: Strong typing throughout the codebase
- **Concurrent-safe**: Proper mutex usage and thread safety
- **Comprehensive feature set**: Full API coverage for all Langfuse features
- **Good documentation**: Well-documented README with examples

### Architecture Overview
The SDK follows a clear architectural pattern:
```
Client (main entry point)
├── HTTP Client (with retry/circuit breaker)
├── Event Queue (async batching)
├── Sub-clients (Traces, Observations, etc.)
└── Builders (fluent API for entity creation)
```

### Code Metrics
- **Total coverage**: 77.1% (target: >85%)
- **Cyclomatic complexity**: Several methods exceed 10 (client.go, http.go)
- **File count**: 100+ files (could be better organized)
- **Lines per file**: Some files exceed 1000 lines (client.go: 1168, errors.go: 897)

## Identified Issues

### P0 - Critical Issues (Immediate Fix Required)

#### 1. Dead Code: Unused Type Aliases (SEVERITY: HIGH)
**Location**: `ingestion.go` lines 97-106
```go
type (
    createTraceEvent      = traceEvent       // Never used
    updateTraceEvent      = traceEvent       // Never used
    createSpanEvent       = observationEvent // Never used
    // ... 5 more unused aliases
)
```
**Impact**: Confuses developers, bloats codebase
**Fix**: Delete entire block

#### 2. Test Coverage Gaps on Critical Paths (SEVERITY: HIGH)
**Functions with 0% coverage**:
- `handleError()` - Async error handling could silently fail
- `handleQueueFull()` - Data loss potential
- `CircuitBreakerState()` - New public API untested
- `drainAllEvents()` - Shutdown reliability at risk

**Impact**: Silent failures, data loss, unreliable shutdown
**Fix**: Add comprehensive tests

#### 3. Circuit Breaker Partial Protection (SEVERITY: HIGH)
**Issue**: Only protects ingestion, not read operations
```go
// Protected:
sendBatch() -> circuitBreaker.Execute()

// NOT Protected:
client.Prompts().Get() -> direct HTTP call
```
**Impact**: Read operations can overwhelm failing API
**Fix**: Wrap all HTTP operations with circuit breaker

### P1 - Major Issues (High Priority)

#### 1. File Size and Complexity (SEVERITY: MEDIUM)
**Problem Files**:
- `client.go`: 1168 lines, 30+ methods, mixed concerns
- `errors.go`: 897 lines, multiple error types mixed
- `ingestion.go`: Contains both data types and ID generation

**Impact**: Hard to navigate, high cognitive load
**Fix**: Split into focused modules

#### 2. Inconsistent Error Handling Patterns (SEVERITY: MEDIUM)
**Issues**:
- Functions named `Is*` return `(*T, bool)` (should be `As*`)
- Redundant error helpers (3 ways to check same condition)
- Deprecated APIs still in use (`Temporary()`)

**Impact**: Confusing API, potential bugs
**Fix**: Standardize on Go conventions

#### 3. Missing Structured Subclient Pattern (SEVERITY: MEDIUM)
**Issue**: Some sub-clients lack consistent patterns
```go
// Good pattern (prompts.go):
PromptsWithOptions() -> ConfiguredPromptsClient

// Missing pattern:
Sessions, Models lack configured variants
```
**Impact**: Inconsistent API experience
**Fix**: Add WithOptions() for all sub-clients

### P2 - Enhancement Opportunities (Medium Priority)

#### 1. Builder API Improvements (SEVERITY: LOW)
**Issues**:
- No compile-time validation for required fields
- Error handling deferred to runtime
- No way to validate before sending

**Enhancement**: Add validated builder pattern
```go
// Current:
trace, err := client.NewTrace().Name("").Create() // Empty name caught at runtime

// Proposed:
result := client.NewTraceStrict().
    Build() // Returns BuildResult[TraceBuilder] with validation
```

#### 2. Metadata Type Lacks Utility (SEVERITY: LOW)
**Current**: `type Metadata map[string]any`
**Missing**: Get(), GetString(), GetInt(), Merge() methods
**Impact**: Developers reimplement common operations
**Fix**: Add utility methods

#### 3. Configuration Explosion (SEVERITY: LOW)
**Issue**: 25+ config fields, growing complexity
**Impact**: Hard to discover options, easy to misconfigure
**Fix**: Group related options into sub-configs

### P3 - Long-term Improvements (Low Priority)

#### 1. Package Structure Reorganization
**Current**: Single flat package with 100+ files
**Proposed**:
```
langfuse/
├── client.go          # Core client only
├── config/            # Configuration types
├── ingestion/         # Event types and batching
├── api/               # Sub-clients
│   ├── traces.go
│   ├── prompts.go
│   └── ...
├── builders/          # Fluent builders
├── errors/            # Error types
└── internal/          # Implementation details
```

#### 2. Interface Segregation
**Issue**: Large interfaces, tight coupling
**Fix**: Define minimal interfaces for testing/mocking

#### 3. Metrics and Observability
**Current**: Basic metrics interface
**Enhancement**: OpenTelemetry integration option

## Proposed Changes

### Phase 1: Critical Fixes (Week 1)

1. **Delete dead code**
   - Remove unused type aliases in `ingestion.go`
   - Remove any other unreferenced code

2. **Fix critical test gaps**
   ```go
   // client_test.go
   func TestHandleError(t *testing.T) { /* test async error handling */ }
   func TestHandleQueueFull(t *testing.T) { /* test queue overflow */ }
   func TestCircuitBreakerState(t *testing.T) { /* test breaker state */ }
   func TestDrainAllEvents(t *testing.T) { /* test shutdown drain */ }
   ```

3. **Complete circuit breaker integration**
   - Add `doWithCircuitBreaker()` to httpClient
   - Update all sub-clients to use it
   - Test circuit breaker with read operations

### Phase 2: Architecture Improvements (Week 2-3)

1. **Split large files**
   ```
   client.go -> client.go (300 lines)
             -> lifecycle.go (200 lines)
             -> batching.go (300 lines)
             -> queue.go (200 lines)
   
   errors.go -> errors/api.go (200 lines)
             -> errors/async.go (200 lines)
             -> errors/validation.go (100 lines)
             -> errors/helpers.go (100 lines)
   ```

2. **Standardize error helpers**
   ```go
   // Rename Is* to As* for extraction functions
   func AsAPIError(err error) (*APIError, bool)
   func AsValidationError(err error) (*ValidationError, bool)
   
   // Remove redundant package-level functions
   // Keep only: errors.Is(err, sentinel) and As* functions
   ```

3. **Add missing WithOptions patterns**
   ```go
   client.SessionsWithOptions(opts ...SessionsOption) *ConfiguredSessionsClient
   client.ModelsWithOptions(opts ...ModelsOption) *ConfiguredModelsClient
   ```

### Phase 3: API Enhancements (Week 4-5)

1. **Implement validated builders**
   ```go
   // Validated builder with compile-time safety
   type TraceBuilderStrict struct {
       name string // required
       // ...
   }
   
   func (b *TraceBuilderStrict) Build() BuildResult[*Trace] {
       if b.name == "" {
           return BuildResult[*Trace]{Err: ErrMissingName}
       }
       return BuildResult[*Trace]{Value: &Trace{Name: b.name}}
   }
   ```

2. **Enhance Metadata type**
   ```go
   func (m Metadata) Get(key string) (any, bool)
   func (m Metadata) GetString(key string) string
   func (m Metadata) GetInt(key string) int
   func (m Metadata) Merge(other Metadata) Metadata
   func (m Metadata) Filter(keys ...string) Metadata
   ```

3. **Simplify configuration**
   ```go
   type Config struct {
       // Core settings
       Credentials CredentialsConfig
       Network     NetworkConfig
       Batching    BatchingConfig
       Lifecycle   LifecycleConfig
       
       // Optional features
       CircuitBreaker *CircuitBreakerConfig
       Metrics        MetricsConfig
       // ...
   }
   ```

### Phase 4: Package Restructuring (Week 6)

1. **Reorganize packages** (see structure above)
2. **Define public interfaces**
   ```go
   type Client interface {
       NewTrace() TraceBuilder
       Flush(context.Context) error
       Shutdown(context.Context) error
   }
   
   type TraceBuilder interface {
       Name(string) TraceBuilder
       Create(context.Context) (Trace, error)
   }
   ```

3. **Add integration tests** for new structure

## Implementation Roadmap

### Week 1: Foundation
- [ ] Set up feature branch
- [ ] Delete dead code (P0)
- [ ] Add critical tests
- [ ] Fix circuit breaker gaps
- [ ] Run full regression tests

### Week 2-3: Core Refactoring
- [ ] Split large files
- [ ] Standardize error handling
- [ ] Add missing patterns
- [ ] Update documentation
- [ ] Achieve 85%+ coverage

### Week 4-5: API Improvements
- [ ] Implement validated builders
- [ ] Enhance Metadata type
- [ ] Refactor configuration
- [ ] Add examples for new APIs

### Week 6: Package Structure
- [ ] Create new package layout
- [ ] Move files gradually
- [ ] Update imports
- [ ] Final testing

## Breaking Changes Assessment

### Definitely Breaking
1. **Error helper renaming**: `IsAPIError()` -> `AsAPIError()`
   - **Migration**: Simple find/replace
   - **Justification**: Follows Go conventions

2. **Removed functions**: Redundant error helpers
   - **Migration**: Use `errors.Is()` or `As*()` functions
   - **Justification**: Reduces API surface

### Potentially Breaking (with compatibility layer)
1. **Package restructuring**
   - **Migration**: Import aliases for backward compatibility
   - **Timeline**: Deprecate flat structure in v2.0

2. **Configuration changes**
   - **Migration**: Accept both old and new config formats
   - **Timeline**: Deprecate flat config in v2.0

## Migration Guide Considerations

### For Error Helper Changes
```go
// Old code:
if apiErr, ok := langfuse.IsAPIError(err); ok {
    // handle
}

// New code:
if apiErr, ok := langfuse.AsAPIError(err); ok {
    // handle
}

// Or use errors.As directly:
var apiErr *langfuse.APIError
if errors.As(err, &apiErr) {
    // handle
}
```

### For Configuration Changes
```go
// Old code (still works):
cfg := &langfuse.Config{
    PublicKey: "pk-xxx",
    SecretKey: "sk-xxx",
    BatchSize: 100,
}

// New code (recommended):
cfg := &langfuse.Config{
    Credentials: langfuse.CredentialsConfig{
        PublicKey: "pk-xxx",
        SecretKey: "sk-xxx",
    },
    Batching: langfuse.BatchingConfig{
        Size: 100,
    },
}
```

## Success Metrics

### Code Quality
- [ ] Test coverage > 85% (from 77.1%)
- [ ] No functions > 50 lines
- [ ] No files > 500 lines
- [ ] Cyclomatic complexity < 10 for all functions

### Performance
- [ ] No performance regression in benchmarks
- [ ] Memory usage stable or improved
- [ ] Batch processing latency unchanged

### Developer Experience
- [ ] Reduced time to first successful API call
- [ ] Fewer support issues about error handling
- [ ] Positive feedback on new builder APIs
- [ ] Clear migration path for breaking changes

### Maintenance
- [ ] Reduced time to add new features
- [ ] Easier onboarding for contributors
- [ ] Cleaner git history with focused modules
- [ ] Simplified debugging with better structure

## Risk Mitigation

1. **Backward Compatibility**
   - Maintain aliases for moved types
   - Gradual deprecation with warnings
   - Comprehensive migration guide

2. **Testing Strategy**
   - Feature flags for new behaviors
   - Parallel testing of old/new code paths
   - Extended beta period for major changes

3. **Rollback Plan**
   - Git tags at each phase completion
   - Feature toggles for risky changes
   - Monitoring for regression signals

## Conclusion

This refactoring plan addresses both immediate technical debt and long-term architectural improvements. By following this phased approach, we can improve code quality and developer experience while minimizing disruption to existing users. The focus on test coverage, consistent patterns, and modular structure will make the SDK more maintainable and easier to extend with new Langfuse features.

The estimated 6-week timeline allows for careful implementation and testing of each phase, with clear checkpoints for validation and potential course corrections. Success will be measured not just in code metrics, but in improved developer satisfaction and reduced maintenance burden.
