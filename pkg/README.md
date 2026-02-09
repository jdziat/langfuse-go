# Internal Package Structure

This directory contains the internal package structure for the Langfuse Go SDK.

## Purpose

The `pkg/` directory organizes SDK internals into focused, maintainable packages while maintaining 100% backward compatibility through type aliases and re-exports at the root level.

## Package Organization

### `config/`
Configuration types, constants, and utilities.
- `types.go`: Region, configuration constants
- Future: Config struct, functional options, validation

### Planned Packages

#### `errors/`
Error types and handling (planned).
- `errors.go`: Core error types
- `api.go`: API error types
- `async.go`: Async error types
- `validation.go`: Validation error types

#### `ingestion/`
Batch processing and queue management (planned).
- `queue.go`: Queue management
- `batch.go`: Batch processing

#### `api/`
HTTP client and API communication (planned).
- `http.go`: HTTP client types

#### `builders/`
Builder interfaces and implementations (planned).
- `trace.go`: Trace builder interfaces

## Backward Compatibility

All types defined in `pkg/` packages are re-exported at the root level through type aliases and constant re-exports. This ensures that existing code using `github.com/jdziat/langfuse-go` continues to work without any changes.

Example:
```go
// In pkg/config/types.go
package config

type Region string
const RegionEU Region = "eu"

// In root regions.go
package langfuse

import "github.com/jdziat/langfuse-go/pkg/config"

type Region = config.Region
const RegionEU = config.RegionEU
```

## Guidelines

1. **Never break backward compatibility**: All public types must be re-exported at root
2. **Start small**: Migrate one package at a time, verify tests pass
3. **Document changes**: Update this README as packages are added
4. **Internal use**: These packages are internal; users should import the root package
