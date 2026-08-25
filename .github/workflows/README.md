# GitHub Actions workflows

Repository guidance for the GitHub Actions suite. The required status-check
contexts are enforced by the live `Require pull requests on protected branches`
ruleset; keep this document, `CONTRIBUTING.md`, and the ruleset in sync.

## Naming convention

- **Filenames**: one workflow per domain, `kebab-case.yml`, named after the
  domain (`commit-conventions.yml`, `branch-policy.yml`, `badges.yml`).
- **Workflow names** (`name:`): Title Case domain phrase matching the filename
  (`Commit conventions`, `Branch policy`, `Badge data`).
- **Job IDs**: kebab-case matching the domain.
- **Displayed job names** (`name:` on jobs): `Validate <domain>` for validation
  checks, `Audit <domain>` for security audits, and domain-only Title Case for
  publishing (`Badge data`).
- **Named steps**: sentence case, imperative verb first (`Check out
  repository`, `Set up Go`, `Validate ...`, `Run ...`, `Publish ...`).

The required status-check contexts are the displayed job names (or the job ID
when no display name is set), so required contexts must not be reworded without
updating the ruleset at the same time.

## Runtime setup

- Go-based policy checks use `.github/actions/setup-go`, which centralizes
  `actions/setup-go` (`go-version: 1.26.5` matching `go.mod` and `mise.toml`,
  caching on, `go.sum` cache key). Checkout runs as a separate named step
  before it, because GitHub loads local actions from the checked-out
  workspace.
- Task-driven workflows (`validate.yml`, `badges.yml`) run through
  `jdx/mise-action` (`install: true`, `cache: true`); the cache key includes
  the mise configuration file hash (`mise.toml`).
- `commit-signatures.yml` keeps its checkout inline with an explicit
  `ref: base.sha` because it runs on `pull_request_target` and must execute
  only trusted base code; moving the checkout into the shared action would
  hide the trusted-base checkout from the security audit.
- `workflow-security.yml` needs no external runtime setup; it runs the pinned
  zizmor action.

## Caching

- `actions/setup-go` caches the Go module and build caches keyed on `go.sum`
  (plus the Go version); `jdx/mise-action` caches mise-managed tool downloads
  keyed on the mise configuration hash. Both default to enabled at the pinned
  SHAs and are set explicitly for clarity. Cache keys invalidate when the
  corresponding dependency metadata changes.
- The fslc verifier download inside `mise run verify-fsl` is intentionally not
  cached; optimizing `mise.toml` task internals is tracked separately.

## Concurrency

- Pull-request workflows supersede older runs with
  `concurrency: group: ${{ github.workflow }}-${{ github.ref }}` and
  `cancel-in-progress: true` (limited to `pull_request` events where a
  workflow also triggers on `push` to `main`, so main runs are never
  cancelled). Publication (`badges.yml`) stays non-cancellable.

## Skipping irrelevant checks

- Workflow-level `paths:` filters are never used on required workflows: per
  GitHub's documentation, a required workflow skipped by path filtering leaves
  its checks "pending" and blocks merging.
- Irrelevant-work avoidance happens inside the job instead:
  `workflow-security.yml` gates the zizmor step on a `git diff` of
  `.github/workflows/**`, so unrelated pull requests skip the audit while the
  required check still reports success (a job whose step is conditional
  reports success).
