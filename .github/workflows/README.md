# GitHub Actions workflows

Repository guidance for the GitHub Actions suite. The required status-check
contexts are enforced by the live `Require pull requests on protected branches`
ruleset; keep this document, `CONTRIBUTING.md`, and the ruleset in sync.

## Naming convention

- **Filenames**: one workflow per domain, `kebab-case.yml`, named after the
  domain (`policy.yml`, `security.yml`, `publish.yml`). Specialty-trigger or
  variant workflows use `<category>-<variant>.yml` (`policy-signatures.yml`,
  `policy-issues.yml`).
- **Workflow names** (`name:`): Title Case. Validation and repository care use
  a single word (`Validate`, `Policy`, `Security`, `Publish`); a variant of a
  category uses the parenthesized form `Category (variant)` (`Policy (Pull
  request)`, `Policy (Signatures)`, `Policy (Issues)`).
- **Job IDs**: kebab-case matching the domain.
- **Displayed job names** (`name:` on jobs): verb-first — `Validate <domain>`
  for validation checks, `Audit <domain>` for security audits, and
  `Publish <domain>` for publishing (`Publish badge data`).
- **Named steps**: sentence case, imperative verb first (`Check out
  repository`, `Set up Go`, `Validate ...`, `Run ...`, `Publish ...`).

The required status-check contexts are the displayed job names (or the job ID
when no display name is set), so required contexts must not be reworded without
updating the ruleset at the same time.

## Workflow inventory

Each workflow stays its own file; they are intentionally not merged, for these
reasons:

- `Validate` (`validate.yml`) — repository validation as parallel jobs. This
  filename must stay unchanged because the README Validation badge URL embeds
  `validate.yml`.
- `Policy (Pull request)` (`policy.yml`) — `pull_request` policy checks
  (branch direction, work item title, commit conventions).
- `Policy (Signatures)` (`policy-signatures.yml`) — commit-signature
  verification; it must run on `pull_request_target` with a trusted-base
  checkout, so it stays separate for security isolation.
- `Policy (Issues)` (`policy-issues.yml`) — `issues`-event policy checks.
- `Security` (`security.yml`) — audit role (zizmor), not a validation check.
- `Targeted` (`targeted.yml`) — change-scoped Tier 2 validation: the job
  always runs, and the targeted command scopes work to the changed files
  (currently targeted FSL mutation on changed specifications and Go dependency
  security for changed Go files), so unrelated pull requests are fast and a
  future required-context promotion cannot leave checks pending. It is not a
  required context; required contexts are the ten listed in
  `CONTRIBUTING.md`.
- `Publish` (`publish.yml`) — publishing the six badge payloads and the
  retained mutation report to the `badge-data` branch; it needs
  `contents: write` and must never be cancelled, so it stays separate from the
  read-only validation checks. It runs on a weekly schedule and on demand
  (full mutation no longer runs after every `main` push).

## Runtime setup

- Go-based policy checks use `.github/actions/setup-go`, which centralizes
  `actions/setup-go` (`go-version: 1.26.6` matching `go.mod` and `mise.toml`,
  caching on, `go.sum` cache key). Checkout runs as a separate named step
  before it, because GitHub loads local actions from the checked-out
  workspace.
- Task-driven workflows (`validate.yml`, `publish.yml`) run through
  `jdx/mise-action` (`install: true`, `cache: true`); the cache key includes
  the mise configuration file hash (`mise.toml`).
- `policy-signatures.yml` keeps its checkout inline with an explicit
  `ref: base.sha` because it runs on `pull_request_target` and must execute
  only trusted base code; moving the checkout into the shared action would
  hide the trusted-base checkout from the security audit.
- `security.yml` needs no external runtime setup; it runs the pinned zizmor
  action without uploading SARIF, so the named `Audit workflow security` job
  remains the single workflow-security check.

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
  cancelled). Publication (`publish.yml`) stays non-cancellable.

## Skipping irrelevant checks

- Workflow-level `paths:` filters are never used on required workflows: per
  GitHub's documentation, a required workflow skipped by path filtering leaves
  its checks "pending" and blocks merging.
- Irrelevant-work avoidance happens inside the job instead:
  `security.yml` gates the zizmor step on a `git diff` of
  `.github/workflows/**`, so unrelated pull requests skip the audit while the
  required check still reports success (a job whose step is conditional
  reports success).
`targeted.yml` follows the same rule for Tier 2: its jobs always run, and
  each step scopes work to changed files. `mutate-fsl --changed-base <base>`
  scopes mutation to specs changed under `specs/` or `skills/**/specs/`, while
  `check:go-vuln` runs only when `go.mod`, `go.sum`, or `*.go` changed. Both
  paths remain successful no-ops when unrelated.
