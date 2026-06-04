# Review Remediation Plan

This plan addresses the findings from the adversarial two-persona review (Design
Visionary + 10x Engineer) of `github.com/jdziat/langfuse-go`. Every item below
traces to a finding that was **confirmed at source** (24 confirmed, 1 refuted).

## Policy decisions (set by maintainer)

1. **Backward compatibility: deprecate, stay 1.x-safe.** No existing exported
   symbol is removed or has its signature changed. Redundant surface gets
   `// Deprecated:` markers pointing to the canonical path; new correct APIs are
   *added* alongside the old ones. Downstream code keeps compiling.
2. **Dead options: wire them up for real.** `WithStrictValidation`,
   `WithClassifiedHooks`, `WithMetricsRecorder`, and `WithOnAsyncError` get real,
   tested behavior instead of being removed.
3. **Git: dedicated branch `fix/review-remediation` off `main`, one commit per
   reviewed packet, no push.** Each packet is implemented, built/vetted/tested,
   signed off by a 10x-engineer reviewer, then committed.

## Execution model

Packets run **sequentially** in dependency order (the core files — `client.go`,
`options.go`, `builders.go`, `doc.go` — are touched by many packets, so parallel
edits would conflict). Each packet:

1. **Implement** — edit, `gofmt -w`, `go build ./...`, `go vet`, targeted tests.
2. **10x review** — an elite reviewer re-runs build/vet/`go test -race` on the
   affected packages and checks the diff against the acceptance criteria. It
   returns `approved` or `changes_required` with blocking issues.
3. **Iterate** up to 3× on `changes_required`.
4. **Commit** on approval (`LANGFUSE_HOOKS_DISABLED=true git commit -m "<conv>"`),
   or `git reset --hard HEAD` to restore the last good state on failure.

A **final integration phase** runs the full `go test -race ./...`, `staticcheck`,
builds every example, runs the new doc-snippet build gate, and does a 10x review
of the whole `git diff main..HEAD`.

---

## Packets

### P1 — Circuit breaker can never re-close (CRITICAL)
- **Files:** `pkg/http/circuit.go`, `pkg/client/http.go`, `pkg/http/http_test.go`, `tests/circuitbreaker_test.go`
- **Fix:** Make half-open recovery work through `Execute()` only. Decrement the in-flight `halfOpenRequests` counter when a probe completes in `Record()` (so successive probes are admitted up to `SuccessThreshold`), and/or set `DefaultCircuitBreakerConfig.HalfOpenMaxRequests >= SuccessThreshold`. Add a guard in `NewCircuitBreaker` that repairs/rejects `SuccessThreshold > HalfOpenMaxRequests`.
- **Also:** default `IsFailure` (wired in `newHTTPClient`) to `APIError.IsRetryable()` so non-retryable 4xx (401/403/404/422) do **not** trip the breaker.
- **Acceptance:** a new test drives recovery **only** through `Execute()` against a healthy backend with the shipped default config and the circuit re-closes; 5 consecutive 404s do not open the circuit; existing circuit tests pass; build+vet clean.

### P2 — Robust retry detection with `errors.As` (MEDIUM)
- **Files:** `pkg/http/retry.go`, `pkg/http/http_test.go`
- **Fix:** Replace bare `err.(RetryableError)` (lines ~181/286/331) and `err.(RetryAfterError)` (line ~199) with `errors.As`.
- **Acceptance:** a wrapped (`fmt.Errorf("...: %w", apiErr)`) retryable 503 with a `Retry-After` is still retried with the server delay; existing tests pass.

### P3 — Persistence file permissions (MEDIUM)
- **Files:** `pkg/evaluation/persistence.go`, new `pkg/evaluation/persistence_perms_test.go`
- **Fix:** `os.WriteFile` mode `0600`, `os.MkdirAll` mode `0700`. Document cleartext storage.
- **Acceptance:** test (skipped on Windows) verifies file is `0600` and dir `0700`; build+vet clean.

### P4 — Single version source of truth (HIGH)
- **Files:** `VERSION` (existing), `doc.go`, `client.go`, `pkg/client/client.go`, `pkg/client/config.go`, new version assertion test
- **Fix:** Root package `//go:embed VERSION` into `Version` (trimmed), replacing the `"1.0.0"` literal. Inject the version into the core client (add a `Version` field to `pkgclient.Config`, set by the root) so the User-Agent in `pkg/client/http.go:171` reports the real version; keep the `pkg/client` constant only as a fallback default. (`pkg/client` cannot `//go:embed ../VERSION`, hence injection.)
- **Acceptance:** `langfuse.Version == strings.TrimSpace(<VERSION file>)`; an outgoing request's User-Agent contains that version; a test asserts the equality so a release can't drift; build+vet clean.

### P5 — Config converter completeness + drift guard (HIGH)
- **Files:** `client.go` (`convertToPkgClientConfig`), new converter test
- **Fix:** Forward every root `Config` field that has a direct `pkgclient.Config` counterpart (e.g. `ClassifiedHooks`). Add a reflection-based test that iterates `pkgclient.Config` fields and fails if any is neither set by the converter nor on an explicit allow-list of intentionally-unmapped fields — so future fields can't be silently dropped.
- **Depends on:** none. **Blocks:** P6, P7.
- **Acceptance:** converter forwards all directly-mappable fields; drift test passes and would fail on a new unmapped field; build+vet+tests clean.

### P6 — Wire ClassifiedHooks + MetricsRecorder + OnAsyncError (HIGH, feature)
- **Files:** `client.go`, `config.go`, `options.go`, `pkg/client/*` as needed, tests
- **Fix:**
  - **ClassifiedHooks:** end-to-end forward so a classified hook actually fires on requests.
  - **EnableMetricsRecorder:** when true, construct the internal metrics recorder (`pkg/lifecycle/metrics.go`) and set `pkgclient.Config.Metrics`, so queue/HTTP metrics are emitted.
  - **OnAsyncError / AsyncErrorConfig:** bridge to `pkgclient.Config.ErrorHandler` so a background batch-send failure invokes the user's handler (wrapped as `*AsyncError`).
- **Depends on:** P5.
- **Acceptance:** one passing test per option proving behavior changes when enabled (hook invoked; a metric recorded; async-error handler called on a simulated background failure); no signature changes; `go test -race` on affected packages clean.

### P7 — Wire StrictValidation (HIGH, feature)
- **Files:** `builders.go`, `client.go`, `config.go`, tests
- **Fix:** When `StrictValidation` is configured, the standard fluent builders run validation on `Create()`/`Apply()` and return an error on invalid input, reusing the existing `Validated*` validation logic. Off by default (no behavior change).
- **Depends on:** P5.
- **Acceptance:** with strict mode on, an invalid build returns a validation error; with it off, behavior is unchanged; `go test -race` on affected packages clean.

### P8 — Embed leak: add correct accessors (HIGH, non-breaking)
- **Files:** `client.go`, `lifecycle.go`, docs
- **Fix:** Keep the `*pkgclient.Client` embed (removing it is breaking). **Add** `func (c *Client) RootConfig() *langfuse.Config` returning the complete, correctly-typed root config. Add doc comments (and `// Deprecated:` where appropriate) on the promoted `Config()`/`HTTP()`/`QueueEvent()` clarifying they expose internals and pointing to `RootConfig()`.
- **Acceptance:** `client.RootConfig()` returns the full root config; godoc clarifies the promoted internal methods; no signature changes; build+vet+tests clean.

### P9 — Populate `EndResult.Duration` (MEDIUM, non-breaking)
- **Files:** `builders.go`, test
- **Fix:** Record the observation start time at `Create()` and compute `time.Since(start)` in the `EndWith`/`End` paths so `EndResult.Duration` is real.
- **Depends on:** none (ordered before P10 to avoid `builders.go` churn).
- **Acceptance:** a span/generation that ran for a measurable interval reports `Duration > 0` ≈ elapsed; build+vet clean.

### P10 — Deprecate redundant surface + "Start Here" (HIGH, non-breaking)
- **Files:** `builders.go`, `simple_api.go`, `evaluation.go`, `doc.go`
- **Fix:** Add `// Deprecated:` comments to the one-shot `X()` methods, the `…V1` methods, and the `Validated*` builder hierarchy, each redirecting to the canonical fluent builder. Update internal usages (tests/examples) to the canonical path so `staticcheck` stays clean. Add a "Start Here" section to `doc.go` naming the ~5 canonical symbols + one canonical example.
- **Depends on:** P7, P9 (also edit `builders.go`).
- **Acceptance:** deprecated symbols carry redirecting `// Deprecated:`; `doc.go` has a Start Here block; no internal use of deprecated APIs; build+vet+staticcheck clean.

### P11 — `doc.go` + README truth (CRITICAL)
- **Files:** `doc.go`, `README.md`
- **Fix:** Rewrite the `doc.go` Quick Start to compile (thread `ctx` through `Create(ctx)`; replace the fictional `gen.End().Output().Usage().Apply()` chain with the real `EndWithUsage(ctx, ...)`). Delete the phantom `otel` subpackage paragraph. Replace `WithQueueSize` → `WithBatchQueueSize`. Fix README config/prompts snippets (don't pass `TraceOption` into `New`; add `ctx` to `Create`), remove references to non-existent `WithPrompts*/WithSessions*/WithModels*` options, and fix the `pk_`/`sk_` vs `pk-lf-`/`sk-lf-` key format.
- **Acceptance:** every Go fenced block in `doc.go` and `README.md` compiles under the doc-build gate (P13); build+vet clean.

### P12 — Rewrite `content/docs/*` to the real API (CRITICAL)
- **Files:** `content/docs/getting-started.md`, `tracing.md`, `evaluation.md`, `configuration.md`, `migration.md`, `api-reference.md`
- **Fix:** Replace the non-existent API (`github.com/jdziat/langfuse-go/langfuse` import path, `WithPublicKey/WithSecretKey`, `TraceParams/SpanParams/GenerationParams{Output,Usage}`, `client.Trace(params)`, `client.Evaluator()`) with the real ctx-threaded fluent/option API, mirroring the corrected README.
- **Depends on:** P11 (canonical examples).
- **Acceptance:** every Go fenced block in `content/docs/**` compiles under the doc-build gate.

### P13 — CI doc-snippet build gate (HIGH)
- **Files:** new `scripts/check-doc-snippets.*`, `.github/workflows/ci.yml`
- **Fix:** A script that extracts every ```` ```go ```` fenced block from `README.md`, `doc.go`, and `content/docs/**`, wraps/asssembles them, and `go build`s them; wire it into CI as a required check so doc rot becomes a red build.
- **Depends on:** P11, P12 (snippets must be correct first).
- **Acceptance:** the gate passes locally on the corrected docs and fails if a snippet is broken; CI job added.

### P14 — Hygiene (LOW)
- **Files:** `.gitignore`, `.hugo_build.lock`, `AGENTS.md`, `pkg/README.md`
- **Fix:** `git rm --cached .hugo_build.lock` and ignore it; fix the malformed `.gitignore` line `evaluationevaluation` and ignore root-built example binaries; rewrite the stale `AGENTS.md` "Key Source Files" + "Technical Debt" sections to match the `pkg/` layout; correct `pkg/README.md`'s false "internal"/`Pkg`-prefix claims.
- **Acceptance:** `.hugo_build.lock` untracked and ignored; `.gitignore` valid; `AGENTS.md`/`pkg/README.md` reflect reality; build unaffected.

---

---

# Round 2 — Push to straight A's (breaking changes allowed, stay on v1)

**Policy change (maintainer):** the library has a single user (the owner). Backward
compatibility and semver are explicitly waived to make this a standout SDK — but it
stays on the **v1 module path** (no `/v2`). Round 1's deprecations become deletions.

Round 1 grades: API Design **B**, Engineering **A−**, Docs/Trust **A−**, Overall **B+**.
Goal: **A in every section.** Same execution model (sequential packets, per-packet
implement → strict 10x signoff → commit). Continues on branch `fix/review-remediation`.

## Engineering → A

### R1 — Version: true single source (no residual lie)
- **Files:** new `internal/version/`, `doc.go`, `client.go`, `pkg/client/client.go`, `pkg/client/http.go`, `.releaserc.json`, CI/test.
- **Fix:** put the version in one embedded file (e.g. `internal/version/version.txt`) imported by both the root and `pkg/client`, so direct `pkg/client` users no longer get the stale `0.1.0`. Update `.releaserc.json` `prepareCmd` to write that file (keep `VERSION` mirrored with a CI equality assertion, or make it the sole source). Delete the duplicate `Version` constants.
- **Acceptance:** one writer; root and `pkg/client` both report the real version; a test asserts equality so release can't drift; build/vet/tests green.

### R2 — HTTP hardening: real MaxDelay ceiling + panic-safe state callback
- **Files:** `pkg/http/retry.go`, `pkg/http/circuit.go`, `pkg/http/http_test.go`.
- **Fix:** cap the backoff **after** applying jitter so `MaxDelay` is a true maximum (currently jitter can push it to ~1.5×). Make `OnStateChange` dispatch panic-safe (recover) and ordering-sane.
- **Acceptance:** tests prove delay never exceeds `MaxDelay`; a panicking `OnStateChange` cannot crash the breaker; build/vet/`-race` green.

### R3 — Close critical-path test coverage
- **Files:** `pkg/client/*_test.go`, `tests/`, optionally `.github/workflows/ci.yml`.
- **Fix:** add tests for the previously-0%-coverage critical paths (`handleError`, `handleQueueFull`, async logging). Optionally add a coverage floor to CI.
- **Acceptance:** new tests exercise those paths and fail if the behavior regresses; `-race` green.

## API Design → A (breaking)

### R4 — Replace the anonymous embed with a curated client
- **Files:** `client.go`, `lifecycle.go`, `subclients.go`, `export_test.go`, tests.
- **Fix:** replace anonymous `*pkgclient.Client` with a named `core` field; hand-write only the genuinely-public methods (Flush/Shutdown/Close/Health/State/…). `Config()` returns `*langfuse.Config`. Remove leaked `HTTP()`/`QueueEvent()` from the public surface (or hide behind an explicit `Advanced()`).
- **Acceptance:** no internal types promoted onto the public client; `Config()` returns the root type; build/vet/`-race` green.

### R5 — Delete the V1 surface
- **Files:** `simple_api.go`, `builders.go`, `tests/v1_api_test.go`, `examples/v1_api/`.
- **Fix:** remove `…V1` methods + `TraceContextV1`/`WrapTraceContext` and their tests/example.
- **Acceptance:** symbols gone; nothing references them; build/`-race`/staticcheck green.

### R6 — Delete the Validated* hierarchy
- **Files:** `builders.go`, `tests/validated_builder_test.go`, related.
- **Fix:** remove the `Validated*` builder tree (redundant now that `StrictValidation` is wired into the standard builders); migrate any unique coverage to strict-mode tests.
- **Acceptance:** symbols gone; strict validation still covered; build/`-race`/staticcheck green.

### R7 — Collapse remaining redundant idioms + naming + package collision
- **Files:** `options.go`, `simple_api.go`, `builders.go`, `pkg/evaluation/` (rename), importers.
- **Fix:** settle on ONE canonical creation idiom (fluent builders) plus at most one documented convenience; remove duplicate verbs; fix `apply2`/`apply3`; rename `pkg/evaluation` to remove the `package evaluation` name collision (or move under `internal/`).
- **Acceptance:** one obvious way to create each object; no `applyN`; no two importable packages named `evaluation`; build/`-race`/staticcheck green.

## Docs / Trust → A (after the API settles)

### R8 — Re-sync all docs to the collapsed API
- **Files:** `doc.go`, `README.md`, `content/docs/*`.
- **Fix:** update every example and reference to the new single API; remove anything referencing deleted V1/Validated; tighten the Start Here.
- **Depends on:** R4–R7.

### R9 — Runnable godoc Examples for the canonical path
- **Files:** `example_test.go` (root + key packages).
- **Fix:** add compile-checked `Example…` functions that render on pkg.go.dev.
- **Depends on:** R4–R7.

### R10 — Strengthen the doc-snippet gate
- **Files:** `scripts/check-doc-snippets.sh`, `.github/workflows/ci.yml`.
- **Fix:** auto-wrap fragments so most of the (previously 97 skipped) snippets actually compile; drive the skip count down; keep CI green.
- **Depends on:** R8, R9.

## Final
Full validation (`build`/`vet`/`-race`/`staticcheck`/examples/doc gate) **plus a fresh
adversarial re-grade panel** (Design Visionary + 10x Engineer, independent of the
implementers) scoring against A criteria.

## Still out of scope
- Moving to a `/v2` module path (staying on v1 by choice).

---

# Round 3 — Close the last gaps to straight A's

Round 2 independent re-grade: **API Design A−, Engineering A, Docs/Trust B+ (regressed), Overall A−.**
Goal: a clean **A** in every section. The graders gave an exact, confirmed roadmap.

## API Design (A− → A)

### R3-A1 — Prune client constructors + drop the RootConfig alias
- **Files:** `client.go`, `lifecycle.go`, `options.go`, referencing tests/examples/docs.
- **Fix:** consolidate the five constructors (`New`, `NewWithConfig`, `NewClient`, `MustClient`, `TryClient`) to a tight canonical set (keep `New` + `NewWithConfig`; remove or fold the rest — breaking OK). Remove the deprecated `RootConfig` alias now that `Config()` returns the root type. Update every reference.
- **Acceptance:** minimal constructor set; no `RootConfig`; no dangling refs; build/vet/staticcheck/`-race` green.

### R3-A2 — DRY the Simple API + rank it clearly secondary
- **Files:** `simple_api.go`, `doc.go`.
- **Fix:** extract shared apply helpers so the ~1600 lines of near-identical per-context (`Trace`/`Span`/`Generation`/`Event`) option-apply loops collapse to one path; the `WithXxx` duplication shrinks. Make godoc unambiguous that the fluent builders are canonical and `Trace(ctx,…)`/helpers are convenience.
- **Acceptance:** simple_api.go materially smaller with no behavior change; one clearly-canonical path in docs; green gates.

## Docs / Trust (B+ → A)

### R3-D1 — Close the doc-gate blind spot + fix all compile rot
- **Files:** `scripts/check-doc-snippets.sh`, `evaluation/doc.go`, any other `**/doc.go`, `.github/workflows/ci.yml`.
- **Fix:** extend `DOC_GLOBS` to scan subpackage `doc.go` files (`**/doc.go`); fix every `go` block it surfaces (e.g. `evaluation/doc.go` `Create(ctx)`/`UpdateOutput(ctx,…)`). Demonstrate the gate would have failed before the fix.
- **Depends on:** R3-A1, R3-A2.
- **Acceptance:** gate covers subpackage doc.go and passes; the previously-broken examples now compile under it.

### R3-D2 — Fix prose/structural doc inaccuracies
- **Files:** `README.md`, `content/docs/*.md`.
- **Fix:** rewrite README "Package Structure" to the real tree (no phantom `queue.go`/`errors_*.go`/`helpers.go`); audit documented defaults/values against source and correct them (`WithFlushInterval` → 5s, etc.).
- **Depends on:** R3-A1, R3-A2.
- **Acceptance:** file map matches the real tree; documented defaults match code.

## Final
Full validation + an independent Design-Visionary + 10x re-grade. Confirm Engineering stays A and API/Docs reach a clean A.
