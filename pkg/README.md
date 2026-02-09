# pkg/ Internal Packages

This directory contains internal implementation packages for the Langfuse Go SDK.

## Package Structure

- **config/** - Configuration types, regions, environment helpers
- **errors/** - Error types, helpers, async error handling
- **http/** - HTTP utilities (retry strategies, circuit breaker, pagination)
- **ingestion/** - Event ingestion utilities (backpressure, UUID)

## Usage

These packages can be imported directly for advanced use cases:

```go
import "github.com/jdziat/langfuse-go/pkg/errors"
import "github.com/jdziat/langfuse-go/pkg/http"
```

For most use cases, use the main `langfuse` package which re-exports
these types with a `Pkg` prefix (e.g., `langfuse.PkgCircuitBreaker`).

## Internal vs Public

While these packages are importable, they are considered internal implementation
details and may change between minor versions. The root `langfuse` package
provides the stable public API.
