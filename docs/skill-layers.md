# Skill layers

## Purpose

Every skill in this repository belongs to one of four layers: process, analyze,
fix, or govern. The layer states the skill's naming pattern, what it may and may
not do, and where its results go. Published skills declare their `layer` and
`related` skills in `CATALOG.yml`; planned skills follow the same vocabulary in
their feature Issues. Issue #98 establishes this model for the whole skill set.

## The four layers

### process

Skills that drive the governed change workflow: issues, plans, implementations,
pull requests, and reviews. They create and update work items, branches, and
pull requests but never edit the target project's source code.

- Published: `create-issue`, `plan-issue`, `implement-issue`, `create-pr`
- Planned: `review-pr` ([#67](https://github.com/hidekitux/skills/issues/67))

### analyze

Read-only investigation skills that discover, prioritize, and report
evidence-backed findings. They never modify files and never create Issues or
Pull Requests; candidates for change are recommendations only.

- Planned: `analyze-project` ([#76](https://github.com/hidekitux/skills/issues/76)),
  which owns the analysis area and folds in error, tests, dependencies, docs,
  performance, and security investigation modes (the separate proposals
  [#77](https://github.com/hidekitux/skills/issues/77)-
  [#82](https://github.com/hidekitux/skills/issues/82) are superseded by
  [#112](https://github.com/hidekitux/skills/issues/112))

### fix

Skills that modify code or artifacts directly to repair, test, or restructure
them. They work from a defined task or Issue and hand their result to the next
owner instead of inventing scope.

- Published: `debug-code`
- Planned: `write-tests` ([#69](https://github.com/hidekitux/skills/issues/69)),
  `refactor-code` ([#70](https://github.com/hidekitux/skills/issues/70))

### govern

Skills that establish or verify repository rules and their enforcement. They
create project governance or audit existing enforcement and report what is
missing; they do not implement the audited rules themselves.

- Published: `bootstrap-project`, `audit-workflow-enforcement`

## Skill-set mapping

| Layer | Skill | Status | Issue |
| --- | --- | --- | --- |
| process | create-issue | published | |
| process | plan-issue | published | |
| process | implement-issue | published | |
| process | create-pr | published | |
| process | review-pr | planned | #67 |
| analyze | analyze-project | planned | #76 |
| fix | debug-code | published | |
| fix | write-tests | planned | #69 |
| fix | refactor-code | planned | #70 |
| govern | bootstrap-project | published | |
| govern | audit-workflow-enforcement | published | |

## Naming pattern

- Analysis skills are named `analyze-<scope>` and follow the common contract in
  [analysis-skill-common.md](analysis-skill-common.md).
- Process and fix skills use a verb-first name (`create-issue`, `debug-code`,
  `write-tests`).
- Governance skills name the governed artifact or action (`bootstrap-project`,
  `audit-workflow-enforcement`).

## Boundaries

- `analyze` recommends; only `create-issue` turns candidates into Issues.
- `fix` changes the target project; `process` and `govern` do not.
- `govern` establishes and verifies rules; `analyze` and `fix` do not change
  governance.
- Every skill states its layer, related skills, and handoff target when it is
  authored or updated.
