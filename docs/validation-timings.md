# Validation task timing report

Measured during the implementation of the mise validation acceleration
change (Issue 156). All runs executed `mise run <task>` from the repository
root on macOS Apple Silicon with mise 2026.8.12 and the pinned tools from
`mise.toml`. Logs were captured to temporary files outside the repository;
the per-task "Finished in" lines come from the mise task output.

## Baseline (before the change)

| Task | Cache | Total | Slowest contributors |
| --- | --- | --- | --- |
| `mise run validate:all` | warm | 2.16s | `check:skills` 2.13s, `check:repository` 694ms, `check:hosts` 566ms, `check:diff` 510ms |
| `mise run validate:all` | cold | 22.14s | `check:repository` 22.11s, `test:all` 18.75s, `check:branch-policy` 17.94s, `check:hosts` 17.94s |
| `mise run check:repository` | warm | 0.48s | six sequential `go run ./cmd/...` commands |
| `mise run lint:all` | warm | 0.31s | parallel lint tasks |
| `mise run test:all` | warm | 0.33s in-graph | `go test ./...` |
| `mise run verify:fsl` | warm | 0.18s in-graph | `go run ./cmd/verify-fsl` + `fslc` |

Cold-cache runs used a fresh `GOCACHE`, `GOMODCACHE`, `RUFF_CACHE_DIR`, and
`FSLC_BIN_DIR` so Go, Ruff, and the pinned `fslc` verifier all started
empty. `mise`-managed tools were already installed (they are setup, not a
per-run cache).

Warm `mise run validate:all` is bounded by `check:skills` (the `gh skill publish
--dry-run` invocation, ~2.1s). The `mise` task graph already runs the
top-level dependencies concurrently; the critical path is the slowest
task, not a sum of task times.

## After the change

| Task | Cache | Total | Slowest contributors |
| --- | --- | --- | --- |
| `mise run validate:all` | warm | 2.14s | `check:skills` 2.11s (unchanged, external `gh` invocation), `test:go` 715ms, `check:hosts` 421ms |
| `mise run validate:all` | cold | 18.84s | `test:go` 18.73s, `check:repository` 15.97s, `check:branch-policy` 15.49s |
| `mise run check:repository` | warm | 0.36s | single `go run ./cmd/check-repository` dispatcher |
| `mise run lint:all` | warm | 0.21s | parallel lint tasks |
| `mise run test:all` | warm | 0.20s | delegates to `test:go` |
| `mise run verify:fsl` | warm | 0.29s | `go run ./cmd/verify-fsl` + `fslc` |

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
- `mise.toml`: `test:all` now delegates to `test:go` (`depends = ["test:go"]`)
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
  and cold `mise run validate:all` is bounded by the full `go test` compile
  (`test:go` 18.73s) regardless; pre-built binaries would add staleness and
  state management for a sub-second cold gain.
- **Deterministic output.** The dispatcher buffers concurrent check output
  and prints it in a fixed order with `[check-name]` prefixes, so a failure
  stays attributable. A failure-injection run confirmed the aggregate task
  fails with `FAIL: check-sensitive-content` and the summary line
  `check:repository: FAILED (1 of 6 repository checks failed:
  check-sensitive-content)`; the other five checks still ran and reported.
- **FSL installation.** `mise` de-duplicates dependency tasks in one graph:
  `mise run verify:fsl ::: install:fsl` executed the `fslc` install script
  once. The installer is already idempotent (checksum check, exit 0 when
  present), so `install:fsl` needs no change.
- **Duplicate tests.** `test:all` and `test:go` define the identical command;
  `test:all` delegates to `test:go`, so one task graph runs the suite once.
  `test:json` remains for the badge workflow.
- **Ruff cache.** `lint:python` keeps `RUFF_CACHE_DIR` under
  `RUNNER_TEMP`/`TMPDIR`, so local warm runs reuse the cache (lint:python
  finishes in ~50-70ms). Making the cache persist across CI runs would need
  workflow-level caching, which is out of scope.
- **Lint review.** No avoidable repeated scans were found: `lint:go`
  (gofmt + vet) and `lint:python` (check + format with the same cache) are
  unchanged; `go vet` and `go test` share `GOCACHE` and do not recompile
  shared packages twice.
- **Documented bottleneck.** Warm `mise run validate:all` (2.14s) is bounded by
  `check:skills` at ~2.1s, an external `gh skill publish --dry-run`
  invocation. Changing how that check works is outside this Issue's scope
  (workflow/tooling changes are tracked separately), so it is the
  documented unchanged bottleneck.

## Validation evidence

- `mise run validate:all`, `mise run lint:all`, `mise run test:all`,
  `mise run check:repository`, `mise run check:local`, and
  `mise run verify:fsl` all pass locally before and after the change.
- Failure injection (untracked file containing a GitHub token pattern)
  fails `mise run check:repository` and names the failing check; removal
  restores a passing run.
- Canonical task names (`validate:all`, `lint:all`, `test:all`, `check:*`,
  `verify:fsl`, `mutate:fsl`, `check:local`, `install:fsl`) work as documented.
- GitHub Actions on `ubuntu-latest` invokes `mise run validate:all`; the
  optimized graph is exercised by the same command locally. The CI run
  itself is confirmed after the branch is pushed by `create-pr`.

## Risk-tiered validation (Issue 176)

Measured during the Issue 176 change on the same macOS Apple Silicon
workstation with the same mise toolchain. `git diff`-targeted runs used
`origin/main` as the base revision and the mutation depth remains 8.

### Tier 2 targeted mutation

| Run | Cache | Total | Notes |
| --- | --- | --- | --- |
| `mise run mutate:fsl-changed -- origin/main` (no spec changed) | warm | ~0.9s | prints "No FSL specifications selected for mutation." and exits 0 |
| `mise run mutate:fsl-changed -- origin/main` (one spec touched) | warm | ~1.1s | mutates only `specs/branch-flow.fsl` |
| `mise run mutate:fsl -- --report <path>` (all eight specs) | warm | ~37s | full Tier 3 / release run with retained report (861 killed / 203 survivors measured on the post-#173/#205 `main`) |

### Tier 1 unchanged surface

| Task | Cache | Before (Issue 156) | After (Issue 176) |
| --- | --- | --- | --- |
| `mise run validate:all` | warm | 2.14s | 2.30s (bounded by `check:skills` 2.28s) |
| `mise run validate:all` | cold | 18.84s | 18.91s (a fresh `GOCACHE`, `GOMODCACHE`, `RUFF_CACHE_DIR`, and `FSLC_BIN_DIR`) |

### What changed

- `mutate:fsl` accepts `--changed-base <rev>` (Tier 2: changed specs only,
  no-op success when none match) and `--report <path>` (retained report that
  distinguishes killed, survived, invalid, and infrastructure-error results);
  `mise run mutate:fsl-changed` wraps the changed-base mode.
- `check:repository` gained `check-mutation-triage` (validates
  `docs/mutation-triage.md`, eleven checks total including the #173 evaluation,
  #182 catalog-docs, and #178 Dependabot-config checks that landed on `main`).
- `publish.yml` moved from every `main` push to a weekly `schedule` plus
  `workflow_dispatch`, and now publishes the retained `fsl-mutation-report.json`
  alongside the six badge payloads.
- `targeted.yml` added the always-running Tier 2 mutation job; it is not a
  required context, so the ten required checks stay stable.

### GitHub Actions timing

The Tier 2 job timing on `ubuntu-latest` is recorded from the first CI run
after the Issue 176 branch is pushed (`create-pr` session); the local warm
measurements above bound the expected delta of the always-success no-op path.
