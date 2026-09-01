# Skill layers

## Purpose

Every skill in this repository belongs to one of four layers: process, analyze,
fix, or govern. The layer states the skill's naming pattern, what it may and may
not do, and where its results go. Published skills declare their `layer` and
`related` skills in `CATALOG.yml`; planned skills follow the same vocabulary in
their feature Issues. The current inventory, layer, and status claims in this
document and the README derive from `CATALOG.yml`: presence in its `skills:`
list is the current publishable inventory, and each entry's `layer` and
`status` fields drive the layer and status documentation. A skill is planned
only when it is absent from the catalog. Issue #98 establishes this model for
the whole skill set.

## The four layers

### process

Skills that drive the governed change workflow: issues, plans, implementations,
pull requests, and reviews. They create and update work items, branches, and
pull requests. Two stages own substantive changes to the process layer's
target-project source code: the implementation stage (`implement-issue`) before
a Pull Request exists, and the post-review stage (`fix-pr`) once one is open.
`merge-pr` may make narrowly scoped conflict-resolution edits while rebasing an
explicitly authorized Pull Request, but it must not invent feature behavior or
apply review fixes; substantive drift returns to `fix-pr`.

- Published: `create-issue`, `plan-issue`, `implement-issue`, `create-pr`,
  `review-pr`, `fix-pr`, `merge-pr`
- Entry points: `improve-project`, `deliver-change` ([#175](https://github.com/hidekitux/skills/issues/175))

### analyze

Read-only investigation skills that discover, prioritize, and report
evidence-backed findings. They never modify files and never create Issues or
Pull Requests; candidates for change are recommendations only.

- Published: `analyze-project` ([#76](https://github.com/hidekitux/skills/issues/76)),
  which owns the analysis area and folds in error, tests, dependencies, docs,
  performance, and security investigation modes (the separate proposals
  [#77](https://github.com/hidekitux/skills/issues/77)-
  [#82](https://github.com/hidekitux/skills/issues/82) are superseded by
  [#112](https://github.com/hidekitux/skills/issues/112))

### fix

Skills that modify code or artifacts directly in isolated, task-scoped work:
repairing a reproduced defect, writing tests for a defined target, or
refactoring against a test baseline without changing behavior. They work from a
defined task or Issue and hand their result to the next owner or into the
governed flow at `create-issue` instead of inventing scope.

- Published: `debug-code`, `write-tests` ([#69](https://github.com/hidekitux/skills/issues/69)),
  `refactor-code` ([#70](https://github.com/hidekitux/skills/issues/70))
- Entry points: `resolve-defect` ([#175](https://github.com/hidekitux/skills/issues/175))

### govern

Skills that establish or verify repository rules and their enforcement. They
create project governance or audit existing enforcement and report what is
missing; they do not implement the audited rules themselves.

- Published: `bootstrap-project`, `audit-workflow-enforcement`

## Skill-set mapping

| Layer | Skill | Status |
| --- | --- | --- |
| process | create-issue | experimental |
| process | plan-issue | experimental |
| process | implement-issue | experimental |
| process | create-pr | experimental |
| process | review-pr | experimental |
| process | fix-pr | experimental |
| process | merge-pr | experimental |
| process | improve-project | experimental |
| process | deliver-change | experimental |
| analyze | analyze-project | experimental |
| fix | debug-code | experimental |
| fix | resolve-defect | experimental |
| fix | write-tests | experimental |
| fix | refactor-code | experimental |
| govern | bootstrap-project | experimental |
| govern | audit-workflow-enforcement | experimental |

## Outcome-oriented entry points

Entry points coordinate the primitives from a user outcome to a complete,
observable result without changing any primitive's layer authority. They belong
to the layer of their primary outcome:

| Entry point | Outcome | Primary layer | Route |
| --- | --- | --- | --- |
| `improve-project` | Improve a project | process | `analyze-project` → `create-issue` → `plan-issue` → `implement-issue` → `create-pr` → `review-pr` → `fix-pr` |
| `deliver-change` | Deliver an Issue-backed change | process | `plan-issue` → `implement-issue` → `create-pr` → `review-pr` → `fix-pr` |
| `resolve-defect` | Resolve a verified defect | fix | `debug-code` → `write-tests` → `create-issue` → `plan-issue` → `implement-issue` → `create-pr` → `review-pr` → `fix-pr` |

An entry point keeps one user-visible progress model across handoffs and
returns one cohesive final report. It coordinates phases and never performs a
phase's work itself; direct primitive invocation remains available for advanced
or partial workflows. See [skill-contract.md](skill-contract.md) for the
routing rules.

## Process versus fix

`implement-issue` and `fix-pr` are the only process-layer skills that edit the
target project's source code, and they never both own a branch at once: the
observable condition is whether the branch has an open Pull Request. The
deciding ownership criterion is observable:

1. **Stage in the artifact flow** — `implement-issue` owns the *implementation*
   stage of report → issue → plan → implementation → pull request → review, and
   `fix-pr` owns the post-review stage that returns a fixed head to review;
   fix-layer skills operate outside both stages.
2. **Edit mandate artifact** — `implement-issue` edits only files within a
   governed Issue's in-scope boundary from its verified plan, and `fix-pr` edits
   only what a review finding or that same boundary justifies; fix-layer edits
   are justified by a task-scoped artifact such as reproduction evidence, a
   defined test target, or a test baseline.
3. **Handoff target** — `implement-issue` hands its in-scope edits to
   `create-pr`, and `fix-pr` hands a pushed head back to `review-pr`; fix-layer
   skills hand to the next fix owner or into the governed flow at
   `create-issue`.

Despite its name, `fix-pr` is `process`, not `fix`: it owns a stage of the
governed change flow and its task source is the review artifact, not an
isolated task-scoped one.

Classify governed implementation as `process`; classify isolated debugging,
test writing, or refactoring as `fix`.

## Naming pattern

- Analysis skills are named `analyze-<scope>` and follow the common contract in
  [analysis-skill-common.md](analysis-skill-common.md).
- Process and fix skills use a verb-first name (`create-issue`, `debug-code`,
  `write-tests`).
- Governance skills name the governed artifact or action (`bootstrap-project`,
  `audit-workflow-enforcement`).

## Boundaries

- `analyze` recommends; only `create-issue` turns candidates into Issues.
- `fix` changes the target project in isolated, task-scoped work; `process`
  changes it only through the governed implementation stage (`implement-issue`)
  and the post-review stage (`fix-pr`); `govern` does not change the target
  project.
- `govern` establishes and verifies rules; `analyze` and `fix` do not change
  governance.
- Every skill states its layer, related skills, and handoff target when it is
  authored or updated.
