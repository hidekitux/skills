# GitHub Actions workflows

Repository guidance for the GitHub Actions suite. The required status-check
contexts are enforced by the live `Require pull requests on protected branches`
ruleset; keep this document, `CONTRIBUTING.md`, and the ruleset in sync.

## Naming convention

- **Filenames**: one workflow per domain, `kebab-case.yml`, named after the
  domain (`policy.yml`, `security.yml`, `publish.yml`). Specialty-trigger or
  variant workflows use `<category>-<variant>.yml` (`policy-signatures.yml`,
  `policy-issues.yml`). Filenames are independent of display names; the
  required status-check contexts follow the displayed job names below.
- **Workflow names are one-word categories.** Title Case. Name each workflow
  after the domain it owns in a single word — `Validate`, `Policy`,
  `Security`, `Publish` — and aggregate that domain's jobs under it instead of
  minting a new name per job or topic (`Validate` runs `Validate tests`,
  `Validate FSL specifications`, and `Validate repository checks` in one
  workflow). Do not name a workflow after a mechanism, tier, or event:
  `Targeted` and `Pull Request Project Status` are rejected.
- **Variants reuse the category word with a parenthesized role.** When one
  domain must span more than one workflow — for example when security
  isolation or a different trigger requires a separate file — keep the
  category word and append a parenthesized role that names the distinguishing
  trigger or scope: `Policy (Pull request)`, `Policy (Signatures)`,
  `Policy (Issues)`.
- **Job IDs**: kebab-case matching the domain.
- **Job display names are verb-first, sentence-case, and short.** Each job
  name starts with the class verb that states what the check establishes —
  `Validate` for validation and policy checks, `Audit` for security audits,
  `Publish` for publication — followed by the domain it checks in lowercase,
  in at most four words (`Validate commit signatures`, `Audit workflow
  security`, `Validate issue project status`). A job name must never be a bare
  tool name (`zizmor`) or describe the mechanism instead of the check
  (`Mutate changed FSL specifications`, `Sync Issue Project Status`).
- **Named steps**: sentence case, imperative verb first (`Check out
  repository`, `Set up Go`, `Validate ...`, `Run ...`, `Publish ...`).
- **The job name is the check identity.** Required status-check contexts are
  the job display names (or the job ID when no display name is set); never
  rely on a tool-generated check name the repository cannot rename. The
  zizmor step uploads SARIF (`advanced-security: true`), which adds GitHub's
  `Code scanning results / zizmor` check to pull requests that run the audit;
  that check is an accepted informational side effect that is not — and must
  never be promoted to — a required context. `Audit workflow security`
  remains the required check.

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
- `Policy (Project)` (`pr-project-status.yml`) — keeps the governing
  Issue's Project Status in sync with every pull_request state transition on
  `pull_request_target`, and advances an Issue to `Planned` from an
  authoritative `plan-issue` comment on `issue_comment`; only comments from
  the repository owner qualify, and PR and comment runs use separate
  concurrency groups. It checks out only trusted repository content and never
  executes comment or Pull Request code.
- `Security` (`security.yml`) — audit role (zizmor), not a validation check.
- `Validate (Targeted)` (`targeted.yml`) — change-scoped Tier 2 validation:
  the jobs `Validate FSL mutation` and `Audit Go dependency security` always
  run, and each command scopes work to changed files (FSL specifications or
  Go source and module files), so unrelated pull requests are fast and a
  future required-context promotion cannot leave checks pending. It is not a
  required context; required contexts are the ten listed in `CONTRIBUTING.md`.
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
  action and uploads SARIF. The resulting `Code scanning results / zizmor`
  check is informational, while the named `Audit workflow security` job remains
  the required workflow-security check.

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
- `targeted.yml` follows the same rule for Tier 2: its jobs always run, and
  each step scopes work to changed files. `mutate-fsl --changed-base <base>`
  scopes mutation to specs changed under `specs/` or `skills/**/specs/`, while
  `check:go-vuln` runs only when `go.mod`, `go.sum`, or `*.go` changed. Both
  paths remain successful no-ops when unrelated.
