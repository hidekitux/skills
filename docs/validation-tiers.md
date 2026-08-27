# Validation tiers

Validation is tiered by change risk so pull requests receive fast, actionable
gates; expensive analyses run at the boundary that pays for them; and release
evidence stays comprehensive. Every validation command belongs to exactly one
tier. This document is the authority for the tier assignments; keep it in sync
when a command, workflow, or trigger changes.

## Tier definitions

- **Tier 1 — fast pull-request.** Runs on every pull request (and on pushes to
  `main` where the workflow also triggers there) as required checks. Fast,
  deterministic, and blocking: a failure makes the pull request unmergeable.
- **Tier 2 — targeted high-risk.** Runs inside the pull request, scoped to the
  files a change actually touches. Jobs always run and report a result so a
  future required-context promotion cannot leave a pull request pending; their
  steps skip when the change is unrelated. Evidence appears on the pull
  request before merge. Survivors from these runs feed the triage register, not
  a hidden green aggregate.
- **Tier 3 — scheduled full.** Runs the full mutation and badge pipeline on a
  schedule and on demand, instead of after every unrelated `main` push. Output
  is observability: the retained report distinguishes killed mutants,
  surviving mutants, invalid (skipped) mutants, and infrastructure errors.
- **Tier 4 — release.** Runs the complete validation surface at the release
  boundary (see `specs/release-gate.fsl` and `docs/releasing.md`). Blocking:
  publication is allowed only from a clean, validated commit.

Failure policy values: **blocking** (a failure blocks the pull request or the
release) and **observability** (results are published; they inform review but
do not by themselves block).

## Tier assignments

| Command / job | Tier | Trigger | Owner | Expected duration | Failure policy |
| --- | --- | --- | --- | --- | --- |
| `check:repository` (ten repository checks, `cmd/check-repository`) | 1 | every PR / `main` push / local | repository owner | ~0.4s warm (see `docs/validation-timings.md`) | blocking |
| `check:hosts` (`cmd/validate-hosts`) | 1 | every PR / `main` push | repository owner | ~0.4s warm | blocking |
| `check:branch-policy` (`cmd/validate-branch-policy`) | 1 | every PR / `main` push | repository owner | ~0.2s | blocking |
| `check:diff` (`cmd/check-whitespace`) | 1 | every PR / `main` push | repository owner | ~0.5s | blocking |
| `lint` (workflows, Python, shell, Go) | 1 | every PR / `main` push | repository owner | ~0.2s warm | blocking |
| `test` (`go test ./...`) | 1 | every PR / `main` push | repository owner | ~0.2s warm / ~18.7s cold | blocking |
| `verify-fsl` (`cmd/verify-fsl`) | 1 | every PR / `main` push | repository owner | ~0.3s warm | blocking |
| `check:skills` (`gh skill publish --dry-run`) | 1 | every PR / `main` push | repository owner | ~2.1s warm | blocking |
| `Validate branch policy` / `Validate work item title` / `Validate commit conventions` (`policy.yml`) | 1 | every PR | repository owner | seconds | blocking |
| `Audit workflow security` (zizmor, `security.yml`) | 1 | every PR / `main` push; step is scoped to `.github/workflows/**` changes | repository owner | seconds | blocking |
| `Validate commit signatures` (`policy-signatures.yml`) | 1 | every PR (`pull_request_target`) | repository owner | seconds | blocking |
| `mutate-fsl --changed-base <rev>` (Tier 2 targeted mutation, `targeted.yml`) | 2 | every PR; step runs only when `specs/**` or `skills/**/specs/**` changes | repository owner | ~1s per changed spec, no-op in <1s when none match | blocking on infrastructure errors; surviving mutants are triaged, not a silent pass |
| behavioral smoke for skill changes (`targeted.yml`) | 2 | every PR; step runs only when `skills/**` changes | repository owner | defined by the #173 evaluation harness (not yet wired into a job) | blocking per the #173 smoke contract |
| `mutate-fsl` (full) + `test:json` + `collect-badges` (`publish.yml`, badge-data) | 3 | weekly `schedule` + `workflow_dispatch` (not every `main` push) | repository owner | ~37s for full mutation (measured at depth 8) + `go test` | observability; the retained report distinguishes categories |
| `mise run validate` (full Tier 1 surface) | 4 | release | repository owner | ~2.1s warm / ~19s cold | blocking |
| `verify-release` / `release:publish` | 4 | release | repository owner | seconds | blocking |

## Change classes

A change selects tiers by the paths it touches. The targeted tier never treats
folder scope alone as a pass: the step runs (or explicitly reports no matching
files), so evidence is present either way.

| Change class | Paths | Selected tiers |
| --- | --- | --- |
| Documentation-only | `README.md`, `docs/**`, `CONTRIBUTING.md` | Tier 1 only |
| Workflow / CI | `.github/**` | Tier 1 (including zizmor audit) |
| Skill change | `skills/**` (excluding specs) | Tier 1 + Tier 2 smoke (when #173 provides the entry point) |
| FSL change | `specs/**/*.fsl`, `skills/**/specs/*.fsl` (+ symlink exposures) | Tier 1 + Tier 2 targeted mutation |
| Release candidate | release tag / Release | Tier 1 + Tier 4 |

## Constraints

- Workflow-level `paths:` filters are never used on required workflows: a
  skipped required workflow leaves its checks pending and blocks merging. Tier
  2 jobs therefore always run and scope the work in a step (see
  `.github/workflows/README.md`).
- Tier 3 never hides survivors: the retained `fsl-mutation-report.json`
  distinguishes killed, survived, invalid (skipped), and infrastructure
  errors, and the triage register (`docs/mutation-triage.md`) records each
  surviving mutant.
- Tier 2 currently adds no new required contexts: the ten required checks
  listed in `CONTRIBUTING.md` stay stable, and the Ruleset is updated only
  together with (or after) a workflow change lands on `main`, never before,
  so no pull request waits forever on a context that does not exist yet.
- Behavioral smoke in Tier 2 is not wired into a job yet: Issue #173 landed
the evaluation corpus and harness, and integrating the harness into a Tier 2
smoke job on skill changes is a tracked follow-up of #176. Until then, skill
changes are covered by Tier 1.