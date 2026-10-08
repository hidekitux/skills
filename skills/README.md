# Skill categories

The published skill library is organized by the task a user needs to complete.
Choose a category first, then choose the skill whose name matches the exact
workflow.

| Category | Use it for | Skills |
| --- | --- | --- |
| `process` | Governed Issues, plans, implementations, Pull Requests, and reviews. | `create-issue`, `plan-issue`, `implement-issue`, `create-pr`, `review-pr`, `fix-pr`, `merge-pr`, `improve-project`, `deliver-change` |
| `analyze` | Read-only codebase, project, session, and backlog investigation, and redesign proposals. | `analyze-codebase`, `analyze-project`, `propose-improvements`, `retrospect-work`, `triage-issues` |
| `fix` | Reproduced repairs, focused tests, and behavior-preserving refactors. | `debug-code`, `resolve-defect`, `write-tests`, `refactor-code` |
| `govern` | Repository setup and enforcement audits. | `bootstrap-project`, `audit-workflow-enforcement` |

Technology skills give one technology's conventions, build commands, and test
commands. They live under a technology category instead of a layer:

| Technology category | Use it for | Skills |
| --- | --- | --- |
| `language` | Programming languages and runtimes such as Go, Python, Kotlin, and Node/TypeScript. | None yet |
| `mobile` | Mobile application frameworks such as Flutter, Android/Compose, and iOS. | None yet |
| `web` | Web front-end frameworks such as React. | None yet |
| `game` | Game and game-mod platforms such as Minecraft Forge and Minecraft Fabric. | None yet |

The canonical layout is `skills/<category>/<skill-name>/SKILL.md`. The category
is a path namespace only. Installation and invocation continue to use the bare
skill name from `SKILL.md`. `refactor-code` is canonical at
`skills/fix/refactor-code`.
