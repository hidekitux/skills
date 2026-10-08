# Skill layers

## Purpose

Every workflow skill in this repository belongs to one of six layers: process,
analyze, fix, maintain, document, or govern. A workflow skill is a catalog entry whose `kind` is
absent or `workflow`; technology skills have no layer and are described in
[Technology skills](#technology-skills). The layer states the skill's naming pattern, what it may and may
not do, and where its results go. Published skills declare their `layer` and
`related` skills in `CATALOG.yml`; planned skills follow the same vocabulary in
their feature Issues. The current inventory, layer, and status claims in this
document and the README derive from `CATALOG.yml`: presence in its `skills:`
list is the current publishable inventory, and each entry's `layer` and
`status` fields drive the layer and status documentation. A skill is planned
only when it is absent from the catalog. This model applies to every workflow
skill.

## Directory hierarchy

Published workflow skills use their layer as the first directory below `skills/`:
`skills/<layer>/<skill-name>/SKILL.md`. The layer is a navigation category, not
part of the public skill name. The category answers the user's task before the
skill name identifies the exact workflow.

| Category | Choose this category when |
| --- | --- |
| `process` | You need to move a governed change through Issues, plans, Pull Requests, or reviews. |
| `analyze` | You need read-only evidence about a codebase, a project, a completed session, an Issue backlog, or a major redesign. |
| `fix` | You need a task-scoped repair, test, or behavior-preserving refactor. |
| `maintain` | You need to keep a project's dependencies, toolchains, and version pins current. |
| `document` | You need to write or update a project's documentation so it matches the code. |
| `govern` | You need to establish or audit repository rules and their enforcement. |

Use these representative requests to navigate without repository history:

| User task | Category | Recommended workflow |
| --- | --- | --- |
| Understand code structure, tests, dependencies, and technical debt. | `analyze` | `analyze-codebase` |
| Understand a repository and its technical debt. | `analyze` | `analyze-project` |
| Investigate project-wide improvements and decide whether a major redesign is warranted. | `analyze` | `propose-improvements` |
| Review a completed session for a recurring workaround. | `analyze` | retrospect-work |
| Review the backlog and choose the next Issue. | `analyze` | triage-issues |
| Plan and deliver a governed repository change. | `process` | `deliver-change` |
| Repair a reproduced defect. | `fix` | `debug-code` |
| Resolve a verified defect end to end. | `fix` | `resolve-defect` |
| Add focused tests for a defined target. | `fix` | `write-tests` |
| Refactor against a passing baseline. | `fix` | `refactor-code` |
| Update dependencies, toolchains, or version pins. | `maintain` | `update-dependencies` |
| Write or update project documentation so it matches the code. | `document` | `write-docs` |

`refactor-code` is at `skills/fix/refactor-code`. This category location keeps
the public name as `refactor-code` while placing the skill with the other
repair workflows.

## The six layers

### process

Skills that drive the governed change workflow: issues, plans, implementations,
pull requests, and reviews. They create and update work items, branches, and
pull requests. Two stages own substantive changes to the process layer's
target-project source code: the implementation stage (`implement-issue`) before
a Pull Request exists, and the post-review stage (`fix-pr`) once one is open.
`merge-pr` may make narrowly scoped conflict-resolution edits while rebasing an
explicitly authorized Pull Request, but it must not invent feature behavior or
apply review fixes; substantive drift returns to `fix-pr`.

- Entry points: `improve-project`, `deliver-change`

### analyze

Read-only investigation skills that discover, prioritize, and report
evidence-backed findings. They never modify files and never create Issues or
Pull Requests; candidates for change are recommendations only.

`analyze-codebase` owns focused codebase assessment, `analyze-project` owns
whole-project findings without architecture decisions, `propose-improvements`
owns project-wide improvement investigation and bounded redesign proposals when
structural change is warranted, `retrospect-work` owns session review, and
`triage-issues` owns Issue backlog comparison and ordering.

### fix

Skills that modify code or artifacts directly in isolated, task-scoped work:
repairing a reproduced defect, writing tests for a defined target, or
refactoring against a test baseline without changing behavior. They work from a
defined task or Issue and hand their result to the next owner or into the
governed flow at `create-issue` instead of inventing scope.

- Entry points: `resolve-defect`

### maintain

Skills that keep a working project current without changing what it does:
dependency, toolchain, and version-pin updates. They change manifests,
lockfiles, and pins in the working tree, verify each change, and hand the
verified change to `implement-issue`, which commits it on the Issue branch.
They change no application code; an update that needs a source change is held
back and recorded for `implement-issue`.

### document

Skills that write or update a project's documentation. They trace every claim
to its source in the repository, follow the project's writing rules, verify
commands, paths, and links, and hand the change to `implement-issue`, which
commits it on the Issue branch. They change no code or configuration; a
requested behavior change is recorded and handed to `implement-issue`.

### govern

Skills that establish or verify repository rules and their enforcement. They
create project governance or audit existing enforcement and report what is
missing; they do not implement the audited rules themselves.

## Technology skills

A technology skill is a catalog entry with `kind: stack`. It gives the
conventions, build commands, and test commands for one technology, such as Go
or Flutter. It creates no Issue, Pull Request, branch, or release, and it hands
off to no skill. Workflow skills keep the contract described in the rest of
this document.

A technology skill lives under one of four technology categories, the first
directory below `skills/`:

| Technology category | Choose this category when |
| --- | --- |
| `language` | The skill covers a programming language or runtime used across project types, such as Go, Python, Kotlin, or Node/TypeScript. |
| `mobile` | The skill covers a mobile application framework, such as Flutter, Android/Compose, or iOS. |
| `web` | The skill covers a web front-end framework, such as React. |
| `game` | The skill covers a game or game-mod platform, such as Minecraft Forge or Minecraft Fabric. |

A technology skill differs from a workflow skill in what the repository checks
require:

- It declares no `layer` in `CATALOG.yml`; `check:repository` rejects one.
- It must not appear in `workflow/skill-graph.yml`, because the graph is where a
  skill declares authority and handoff transitions; the skill graph check
  rejects a node for it.
- It is not listed in `docs/skill-instruction-inventory.yml`.
- It keeps the Todo List, frontmatter, license, and catalog requirements of
  every published skill, and it needs a positive scenario and a negative or
  boundary scenario under `evaluations/scenarios/<skill-name>/`.
- Its row in the generated skill list shows its technology category in the
  "Layer or technology category" column.

A workflow skill must not live under a technology category directory. A
technology skill describes a technology; it can refer to another technology
skill by name, for example `develop-kotlin` from a Compose skill, instead of
copying shared files.

## Skill-set mapping

<!-- BEGIN generated: skill-list -->

This list is generated from `CATALOG.yml` by `mise run generate:skill-lists` and checked by `check:repository`. Do not edit it by hand.

The repository publishes 28 skills: 22 workflow skills and 6 technology skills.

| Skill | Layer or technology category | Status |
| --- | --- | --- |
| `create-issue` | process | stable |
| `plan-issue` | process | experimental |
| `implement-issue` | process | experimental |
| `create-pr` | process | experimental |
| `review-pr` | process | experimental |
| `fix-pr` | process | experimental |
| `merge-pr` | process | experimental |
| `improve-project` | process | experimental |
| `deliver-change` | process | experimental |
| `analyze-codebase` | analyze | experimental |
| `analyze-project` | analyze | experimental |
| `propose-improvements` | analyze | experimental |
| `retrospect-work` | analyze | experimental |
| `triage-issues` | analyze | experimental |
| `debug-code` | fix | experimental |
| `resolve-defect` | fix | experimental |
| `write-tests` | fix | experimental |
| `refactor-code` | fix | experimental |
| `update-dependencies` | maintain | experimental |
| `write-docs` | document | experimental |
| `bootstrap-project` | govern | experimental |
| `audit-workflow-enforcement` | govern | experimental |
| `develop-go` | language | experimental |
| `develop-kotlin` | language | experimental |
| `develop-typescript` | language | experimental |
| `develop-java` | language | experimental |
| `develop-dart` | language | experimental |
| `develop-swift` | language | experimental |

<!-- END generated: skill-list -->

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
- Maintenance skills use a verb-first name (`update-dependencies`).
- Documentation skills use a verb-first name (`write-docs`).
- Governance skills name the governed artifact or action (`bootstrap-project`,
  `audit-workflow-enforcement`).
- Technology skills are named `develop-<technology>` (`develop-go`,
  `develop-kotlin`).

## Boundaries

- `analyze` recommends; only `create-issue` turns candidates into Issues.
- `fix` changes the target project in isolated, task-scoped work; `process`
  changes it only through the governed implementation stage (`implement-issue`)
  and the post-review stage (`fix-pr`); `govern` does not change the target
  project.
- `maintain` updates dependencies, toolchains, and pins and hands the change to
  `implement-issue`; it does not change application code.
- `document` writes documentation and hands the change to `implement-issue`; it
  does not change code or configuration.
- `govern` establishes and verifies rules; `analyze` and `fix` do not change
  governance.
- Every workflow skill states its layer, related skills, and handoff target
  when it is authored or updated. A technology skill states its technology
  category and related skills.
