---
name: resolve-defect
description: Resolve a verified defect end to end from reproduction through fix, regression tests, and, when the fix is a governed change, a reviewed update. Use when asked to debug and fix a defect and see it through to a finished, verified result as one coordinated run instead of selecting each skill manually.
license: Apache-2.0
---

# Resolve Defect

## Todo List

1. **in progress:** Resolve the defect report, the reproduction boundary, and whether the fix is a governed change.
2. Run the debug loop with `debug-code` and record reproduction, root-cause, fix, and verification evidence.
3. Add regression tests for the verified fix with `write-tests`.
4. When the fix is a governed change, continue through `create-issue`, `plan-issue`, `implement-issue`, `create-pr`, and `review-pr`.
5. Complete the list only when the defect is verified fixed and one cohesive final report is delivered; hand off the evidence and any Issue or Pull Request URLs.

Keep exactly one item in progress. Mark an item complete only after its stated evidence exists: a file, a command result, an Issue or Pull Request URL, or another observable artifact. Add or revise items when the observed failure changes. Use the host's native Todo List when available; otherwise maintain this list as a Markdown checklist in the conversation.

## Resolve and Guard

- Reproduce the defect first and record the reproduction evidence before any fix is proposed. Never skip to a fix without a reproduction.
- Route each step through its owning primitive: `debug-code` owns reproduction through fix, `write-tests` owns regression tests, and the governed flow owns any permanent change.
- When a fix is not a governed change (isolated, task-scoped work), stop after debug and tests and hand off; do not force the fix into Issue governance.
- Preserve approval boundaries: external mutations (Issue, Pull Request) happen only through the owning primitives with user confirmation.
- Stop and report when the defect cannot be reproduced, the root cause cannot be evidenced, or the fix does not verify; never claim completion from intent alone.

## Route

Run the steps in order. A step is complete only when its primitive's artifact exists with evidence.

| Phase | Primitive | Read-only | Continue when | Artifact |
| --- | --- | --- | --- | --- |
| Debug | `debug-code` | no (fixes isolated defect only) | fix verified against reproduction | reproduction, root-cause, fix, verification evidence |
| Tests | `write-tests` | no (adds tests only) | regression tests recorded with failure evidence | focused test cases |
| Issue | `create-issue` | no | Issue created after user confirmation | change Issue |
| Plan | `plan-issue` | yes | plan posted as an Issue comment | verified plan |
| Implement | `implement-issue` | no (edits in-scope files only) | per-task evidence recorded | in-scope changes |
| Pull request | `create-pr` | no | Pull Request opened after user confirmation | Pull Request URL |
| Review | `review-pr` | yes | findings returned | severity-ordered findings |

## State and authority contract

- **One progress model:** maintain one Todo List for the whole run and map each primitive's own progress into it; the user never tracks a separate list per phase.
- **External mutation authority:** only `create-issue` creates Issues and only `create-pr` creates Pull Requests, each pausing for user confirmation. `debug-code` and `write-tests` change the project only in isolated, task-scoped work justified by reproduction or test-target evidence.
- **Read-only phases:** plan and review never mutate repository files or external work items.
- **Handoff rule:** every step hands its artifact to the next owning primitive; an incomplete artifact returns to its owning primitive for rework, never to the entry point's own logic.
- **Loop termination:** the debug loop terminates when `debug-code` verifies the fix against the reproduction. The review loop terminates when `review-pr` reports no blocking findings, or after two complete rework passes; then stop and report the unresolved findings with evidence instead of continuing.

## Validate and Handoff

- Run the repository-prescribed checks after the fix and tests (`mise run validate`, plus `mise run validate-skill-creator` when available) and record every command and result; never describe an unrun check as passing.
- Deliver one cohesive final report: reproduction, root cause, fix, verification results, test results, and, when the fix is governed, the Issue and Pull Request URLs with the governing Issue's label state.
- Hand off follow-ups as the flow directs. Do not merge, release, or expand scope unless the user separately requests it.