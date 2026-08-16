---
name: implement-issue
description: Implement a governed Change Issue from its verified implementation plan. Use when asked to execute an Issue-backed implementation plan, implement the tasks derived from a Change Issue's Scope and Acceptance criteria, edit only in-scope files, record observable evidence per completed task, and prepare completed work for a Pull Request.
license: Apache-2.0
---

# Implement Issue

## Todo List

1. **in progress:** Read the linked Change Issue and its implementation plan; confirm the repository, target branch, and in-scope boundary.
2. Derive or confirm every implementation task from the Issue's Scope and Acceptance criteria, and record the task list.
3. Implement each task by editing only in-scope files; record observable evidence for each completed task.
4. Run the repository-required validation and record every command and result.
5. Complete the list only when every task and validation item has evidence, or report the remaining unfinished tasks and risks in the handoff.

Keep exactly one item in progress. Mark an item complete only after its stated result or evidence exists. Add or revise items when the agreed scope changes. Use the host's native Todo List when available; otherwise maintain this list as a Markdown checklist in the conversation.

## Resolve and Guard

- Read repository instructions and relevant templates before editing.
- Resolve the linked Change Issue, the implementation plan, the current branch, and the allowed files from the plan. Do not guess a plan when one exists.
- When the Issue has no implementation plan, derive the tasks directly from its `Scope` and `Acceptance criteria` sections; treat `Out` items and unticked acceptance boxes as non-goals for this invocation.
- Implement only tasks derived from the Issue plan or the Issue's scope and acceptance criteria. Do not add unplanned features, unrelated refactors, or out-of-scope edits.
- Preserve unrelated user changes. Do not switch branches, stage unrelated files, rewrite history, or reset the worktree when doing so would include work outside the linked Issue.
- Stop and report when the plan or the Issue lacks a scoped boundary or acceptance criteria, or when an in-scope branch is unavailable.
- Do not create, plan, or release work here. Issue creation belongs to `create-issue`, planning belongs to `plan-issue`, and publishing a Pull Request belongs to `create-pr`.

## Implement

- Track every implementation task on the Todo List; keep exactly one task item in progress.
- Start from the current work branch and the plan's base revision. When the plan calls for a fresh Issue branch, create it only after confirming the Issue-backed branch convention and the upstream base.
- Edit only the files required by the task. Prefer the repository's existing patterns, local helper APIs, and module boundaries.
- For each completed task, record observable evidence: the files changed, the commands run, and the resulting output or artifact. Never claim completion from intent alone.
- Re-read repository instructions and re-check scope when the plan changes or when the base revision changes. Do not reuse a stale plan.

## Validate and Handoff

- Run the repository-prescribed checks for the completed work. Prefer the standard task runner and aggregate validation command; do not substitute narrower commands for an available aggregate task.
- Record every command and result in the handoff. State every skipped or failing check explicitly; never describe an unrun check as passing.
- Report the Issue URL, the implemented tasks, the changed files, the validation commands and results, and any remaining unfinished tasks or risks.
- Hand off completed work to `create-pr` only when the in-scope implementation and validation evidence are complete. Do not publish a Pull Request, merge, or release unless the user separately requests it.
