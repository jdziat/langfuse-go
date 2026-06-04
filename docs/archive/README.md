# Archived documentation

These files are **historical** — design proposals, refactoring plans, and roadmaps
kept for provenance. They predate the current API and are **not authoritative**.

They may reference shapes that have since been **removed or renamed**, including:

- the V1 "dual-tier" API (`TraceV1`, `NewSpanV1`, `EndV1`, …) — removed
- the `Validated*` builder hierarchy and `BuildResult[T]` — removed (validation now
  flows through `WithStrictValidation`)
- `Client.RootConfig()` — removed (`Client.Config()` returns the full config)
- `NewClient` / `MustClient` / `TryClient` — consolidated into `New` / `NewWithConfig`
  / `NewFromEnv` / `MustNew`
- the old flat root-file layout (`api.go`, `http.go`, `traces.go`, …) — migrated into
  `pkg/*`
- the `Pkg`-prefixed re-exports — removed

For current, accurate documentation see:

- the package overview on [pkg.go.dev](https://pkg.go.dev/github.com/jdziat/langfuse-go)
  (generated from `doc.go`)
- the repository `README.md` and the docs site under `content/docs/`
- `../migration-v1.md` for what changed and how to migrate
- `../TROUBLESHOOTING.md` and `../PRODUCTION.md` for operational guidance

Contents:

- `IMPROVEMENT_PROPOSAL.md`, `PROPOSAL-SDK-IMPROVEMENTS.md` — early SDK improvement proposals
- `LLM_AS_JUDGE_INTEGRATION_DESIGN.md` — LLM-as-judge design notes
- `V1_API_ROADMAP.md` — the (superseded) dual-tier v1 API roadmap
- `plans/` — the pkg-restructure and root-facade refactoring plans (completed)
- `proposals/` — assorted historical API/architecture proposals
