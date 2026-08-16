---
name: plan-issue
description: Read a governed change Issue and produce a verified implementation plan before coding begins. Use when asked to plan, break down, or scope an Issue into ordered tasks, or before implementing a Change Issue. Do not write or edit code and do not execute the plan.
license: Apache-2.0
---

# Plan Issue

## Todo List

1. **in progress:** Resolve the repository, the governing change Issue, and its `Scope` and `Acceptance criteria`.
2. Derive plan tasks that cover the Issue scope and acceptance criteria; order them for implementation.
3. Record every task in a Todo List with observable completion evidence and a handoff.
4. Complete the list only when the plan tasks, their ordering, and their evidence are available; hand off the plan without writing or executing code.

Keep exactly one item in progress. Mark an item complete only after its stated evidence exists. Add or revise items when the agreed scope changes.

## Read the Issue

- Read the governing change Issue first. Use `Context`, `Goal`, `Scope`, and `Acceptance criteria` as the authority for what must be planned.
- Confirm the repository, branch, and any repository instructions before deriving tasks.
- Read `Scope` markers and `Acceptance criteria` exactly. The plan must cover included work and must not silently add excluded work.
- Treat a missing `Scope` or `Acceptance criteria` as a blocker: stop, explain what is missing, and ask to complete the Issue before planning.
- When the Issue references other Issues, Pull Requests, or release work, include what the Change flow needs and defer release work to the release Issue.

## Derive Plan Tasks

- Decompose the change into small, ordered tasks that map to the `Scope` and `Acceptance criteria`.
- Add a discovery task first when repository context or the Change flow requires it; add a validation task when the repository requires checks.
- Keep each task small enough to complete and verify independently. Prefer tasks whose evidence is a file, command result, Issue update, or other observable artifact.
- Order tasks so each depends only on tasks before it. Split or merge tasks when the Issue scope changes.
- Do not include writing or editing code: implementation and validation of the code belong to `implement-issue`. A planning skill plans; it does not code.
- Do not include release work, which belongs to the release flow.

## Record the Todo List

- Follow the repository Todo List contract: keep exactly one item in progress, complete an item only after its observable result exists, and hand off when every item is complete or explicitly explained.
- Use the host's native Todo List when available; otherwise present and maintain an equivalent Markdown checklist.
- Write every task with a verb-led, observable completion condition such as `Add`, `Formalize`, or `Verify`.
- Prefer task evidence that is deterministic: a path, command, test result, artifact, or other concrete signal. Do not rely on prose alone.

## Handoff

- Hand off the plan only. Report the governing Issue, the ordered tasks with their completion evidence, and any remaining risk or required context. Do not write code, create commits, or execute the plan unless separately requested.
- Simple Issues may skip this planning step. When a plan is needed, the next
  session uses `implement-issue` to execute it; `create-pr` opens the Pull
  Request in a later session.
