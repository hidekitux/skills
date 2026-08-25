# Validation task timing report

Measured during the implementation of the mise validation acceleration
change (Issue 156). All runs executed `mise run <task>` from the repository
root on macOS Apple Silicon with mise 2026.8.12 and the pinned tools from
`mise.toml`. Logs were captured to temporary files outside the repository;
the per-task "Finished in" lines come from the mise task output.

## Baseline (before the change)

| Task | Cache | Total | Slowest contributors |
| --- | --- | --- | --- |
| `mise run validate` | warm | 2.16s | `check:skills` 2.13s, `check:repository` 694ms, `check:hosts` 566ms, `check:diff` 510ms |
| `mise run validate` | cold | 22.14s | `check:repository` 22.11s, `test` 18.75s, `check:branch-policy` 17.94s, `check:hosts` 17.94s |
| `mise run check:repository` | warm | 0.48s | six sequential `go run ./cmd/...` commands |
| `mise run lint` | warm | 0.31s | parallel lint tasks |
| `mise run test` | warm | 0.33s in-graph | `go test ./...` |
| `mise run verify-fsl` | warm | 0.18s in-graph | `go run ./cmd/verify-fsl` + `fslc` |

Cold-cache runs used a fresh `GOCACHE`, `GOMODCACHE`, `RUFF_CACHE_DIR`, and
`FSLC_BIN_DIR` so Go, Ruff, and the pinned `fslc` verifier all started
empty. `mise`-managed tools were already installed (they are setup, not a
per-run cache).

Warm `mise run validate` is bounded by `check:skills` (the `gh skill publish
--dry-run` invocation, ~2.1s). The `mise` task graph already runs the
top-level dependencies concurrently; the critical path is the slowest
task, not a sum of task times.

## After the change

| Task | Cache | Total | Slowest contributors |
| --- | --- | --- | --- |
| `mise run validate` | warm | 2.14s | `check:skills` 2.11s (unchanged, external `gh` invocation), `test:go` 715ms, `check:hosts` 421ms |
| `mise run validate` | cold | 18.84s | `test:go` 18.73s, `check:repository` 15.97s, `check:branch-policy` 15.49s |
| `mise run check:repository` | warm | 0.36s | single `go run ./cmd/check-repository` dispatcher |
| `mise run lint` | warm | 0.21s | parallel lint tasks |
| `mise run test` | warm | 0.20s | delegates to `test:go` |
| `mise run verify-fsl` | warm | 0.29s | `go run ./cmd/verify-fsl` + `fslc` |

### `check:repository` in-graph timing (warm and cold)

| Cache | Before | After |
| --- | --- | --- |
| warm | 694ms | 299ms |
| cold | 22.11s | 15.97s |

The dispatcher replaces six sequential `go run` commands with one
compilation whose six checks run concurrently; at cold cache this removes
five redundant package links and reduces build-cache lock contention with
the other Go-consuming tasks in the graph.

## What changed

- `cmd/check-repository` (new) runs the six independent repository checks
  (`validate-repository`, `check-tool-licenses`, `validate-script-tests`,
  `check-sensitive-content`, `check-mutation-badges`, `check-analyze-readonly`)
  concurrently, prints each result under a labeled section, and exits 1 if
  any check fails. The six individual commands remain and are unchanged.
- `mise.toml`: `check:repository` now runs `go run ./cmd/check-repository`.
- `mise.toml`: `test` now delegates to `test:go` (`depends = ["test:go"]`)
  instead of redefining the same `go test ./...` command; `test:json`
  (used by the badge workflow) is unchanged.
- `SCRIPT_TESTS.toml`: `cmd/check-repository` maps to its own test package.
- `docs/validation-timings.md`: this report.

## Decisions and evidence

- **Dispatcher over pre-built binaries.** Consolidating the six checks
  behind one command measurably reduces repeated compilation (six `go run`
  links to one). Building all validation binaries once per run was
  evaluated and not adopted: Go already reuses the shared `GOCACHE` across
  `go run`, `go test`, and `go vet`, the remaining Go tasks run
  concurrently and off the warm critical path (bounded by `check:skills`),
  and cold `mise run validate` is bounded by the full `go test` compile
  (`test:go` 18.73s) regardless; pre-built binaries would add staleness and
  state management for a sub-second cold gain.
- **Deterministic output.** The dispatcher buffers concurrent check output
  and prints it in a fixed order with `[check-name]` prefixes, so a failure
  stays attributable. A failure-injection run confirmed the aggregate task
  fails with `FAIL: check-sensitive-content` and the summary line
  `check:repository: FAILED (1 of 6 repository checks failed:
  check-sensitive-content)`; the other five checks still ran and reported.
- **FSL installation.** `mise` de-duplicates dependency tasks in one graph:
  `mise run verify-fsl ::: fsl:install` executed the `fslc` install script
  once. The installer is already idempotent (checksum check, exit 0 when
  present), so `fsl:install` needs no change.
- **Duplicate tests.** `test` and `test:go` defined the identical command;
  `test` now delegates to `test:go`, so one task graph runs the suite once.
  `test:json` remains for the badge workflow.
- **Ruff cache.** `lint:python` keeps `RUFF_CACHE_DIR` under
  `RUNNER_TEMP`/`TMPDIR`, so local warm runs reuse the cache (lint:python
  finishes in ~50-70ms). Making the cache persist across CI runs would need
  workflow-level caching, which is out of scope.
- **Lint review.** No avoidable repeated scans were found: `lint:go`
  (gofmt + vet) and `lint:python` (check + format with the same cache) are
  unchanged; `go vet` and `go test` share `GOCACHE` and do not recompile
  shared packages twice.
- **Documented bottleneck.** Warm `mise run validate` (2.14s) is bounded by
  `check:skills` at ~2.1s, an external `gh skill publish --dry-run`
  invocation. Changing how that check works is outside this Issue's scope
  (workflow/tooling changes are tracked separately), so it is the
  documented unchanged bottleneck.

## Validation evidence

- `mise run validate`, `mise run lint`, `mise run test`,
  `mise run check:repository`, `mise run check:local`, and
  `mise run verify-fsl` all pass locally before and after the change.
- Failure injection (untracked file containing a GitHub token pattern)
  fails `mise run check:repository` and names the failing check; removal
  restores a passing run.
- Existing task names (`validate`, `lint`, `test`, `check:*`, `verify-fsl`,
  `mutate-fsl`, `check:local`, `fsl:install`) work unchanged.
- GitHub Actions on `ubuntu-latest` invokes `mise run validate`; the
  optimized graph is exercised by the same command locally. The CI run
  itself is confirmed after the branch is pushed by `create-pr`.