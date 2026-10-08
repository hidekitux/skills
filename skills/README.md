# Skill categories

The published skill library is organized by the task a user needs to complete.
Choose a category first, then choose the skill whose name matches the exact
workflow.

| Category | Use it for |
| --- | --- |
| `process` | Governed Issues, plans, implementations, Pull Requests, and reviews. |
| `analyze` | Read-only codebase, project, session, and backlog investigation, and redesign proposals. |
| `fix` | Reproduced repairs, focused tests, and behavior-preserving refactors. |
| `maintain` | Dependency, toolchain, and version-pin updates. |
| `document` | Project documentation that matches the code. |
| `govern` | Repository setup and enforcement audits. |

Technology skills give one technology's conventions, build commands, and test
commands. They live under a technology category instead of a layer:

| Technology category | Use it for |
| --- | --- |
| `language` | Programming languages and runtimes such as Go, Python, Kotlin, and Node/TypeScript. |
| `mobile` | Mobile application frameworks such as Flutter, Android/Compose, and iOS. |
| `web` | Web front-end frameworks such as React. |
| `game` | Game and game-mod platforms such as Minecraft Forge and Minecraft Fabric. |

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

The canonical layout is `skills/<category>/<skill-name>/SKILL.md`. The category
is a path namespace only. Installation and invocation continue to use the bare
skill name from `SKILL.md`. `refactor-code` is canonical at
`skills/fix/refactor-code`.
