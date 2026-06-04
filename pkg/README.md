# pkg/ Building Blocks

This directory holds the implementation packages that make up the Langfuse Go
SDK. They are importable building blocks: the root `langfuse` package is a thin
facade composed from them.

## Package layout

- **api/** — per-resource read API sub-clients (`api/traces`, `api/observations`,
  `api/scores`, `api/prompts`, `api/datasets`, `api/sessions`, `api/models`).
- **builders/** — fluent builder helpers, interfaces, and input validation.
- **client/** — the core client: HTTP wiring, configuration, batching,
  ingestion, lifecycle, and options. The root `langfuse.Client` embeds
  `*client.Client`.
- **config/** — configuration types, regions, and environment helpers.
- **errors/** — error types, sentinels, and `errors.Is`/`errors.As` helpers.
- **evaluation/** — evaluation result types, flattening, and persistence.
- **http/** — HTTP utilities: retry strategies, circuit breaker, pagination,
  hooks, and the request `Doer`.
- **id/** — ID generation.
- **ingestion/** — event ingestion utilities (event structs, backpressure, UUID).
- **lifecycle/** — lifecycle manager and metrics.
- **types/** — shared data types (`Trace`, `Observation`, `Score`, `Prompt`,
  `Dataset`, `Session`, `Time`, `Metadata`, enums, and constants).

## How the root facade uses these packages

The root `langfuse` package does not duplicate these definitions. Instead it:

- **Embeds `*pkgclient.Client`** (from `pkg/client`) in `langfuse.Client`, which
  promotes the core client's exported methods (HTTP, batching, lifecycle).
- **Re-exports types via Go type aliases**, so the root names are identical
  types to the underlying ones. For example:

  ```go
  type Trace = types.Trace             // from pkg/types
  type Metadata = types.Metadata       // from pkg/types
  type BatchResult = pkgclient.BatchResult // from pkg/client
  ```

  There is no `Pkg`-prefixed naming scheme; the root names match their public
  documentation (`langfuse.Trace`, `langfuse.Metadata`, etc.).

Because these are aliases, a value produced by a `pkg/*` package and a value of
the corresponding root type are interchangeable.

## Importing directly

You may import these packages directly for advanced use cases:

```go
import (
    pkgclient "github.com/jdziat/langfuse-go/pkg/client"
    "github.com/jdziat/langfuse-go/pkg/errors"
    "github.com/jdziat/langfuse-go/pkg/http"
)
```

## Compatibility expectations

The root `langfuse` package is the stable, recommended public API and follows
the module's semantic-versioning guarantees. The `pkg/*` packages are part of the
public module and are importable, but they exist primarily to support the root
facade; treat their surface as more likely to evolve than the root package. When
in doubt, depend on the root `langfuse` types and methods.
