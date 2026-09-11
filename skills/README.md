# Skill categories

The published skill library is organized by the task a user needs to complete.
Choose a category first, then choose the skill whose name matches the exact
workflow.

| Category | Use it for | Skills |
| --- | --- | --- |
| `process` | Governed Issues, plans, implementations, Pull Requests, and reviews. | `create-issue`, `plan-issue`, `implement-issue`, `create-pr`, `review-pr`, `fix-pr`, `merge-pr`, `improve-project`, `deliver-change` |
| `analyze` | Read-only project, session, and backlog investigation. | `analyze-project`; `retrospect-work` and `triage-issues` are reserved for Issues #285 and #286 |
| `fix` | Reproduced repairs, focused tests, and behavior-preserving refactors. | `debug-code`, `resolve-defect`, `write-tests`, `refactor-code` |
| `govern` | Repository setup and enforcement audits. | `bootstrap-project`, `audit-workflow-enforcement` |

The usual layout is `skills/<category>/<skill-name>/SKILL.md`, but an Issue may
approve a direct layout such as `skills/refactor-code/SKILL.md`. The category is
a path namespace only. Installation and invocation continue to use the bare
skill name from `SKILL.md`. `refactor-code` is canonical at
`skills/refactor-code` under Issue #287 while remaining a `fix`-layer skill.
