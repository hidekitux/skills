---
name: plan-issue
description: Read a governed change Issue and produce a verified implementation plan before coding begins. Use when asked to plan, break down, or scope an Issue into ordered tasks, or before implementing a Change Issue. Do not write or edit code and do not execute the plan.
license: Apache-2.0
---

# Plan Issue

## Todo List

1. **in progress:** Resolve the repository, the governing change Issue, its `Scope` and `Acceptance criteria`, and the plan comment.
2. Test the Issue's `Context` as a hypothesis, surface decisions and undefined terms, and resolve them with the requester before deriving tasks.
3. Derive plan tasks that cover the Issue scope and acceptance criteria; order them for implementation.
4. Record every task in a Todo List with observable completion evidence.
5. Complete the list only when the plan is posted as a comment on the governing Change Issue, its comment URL is available, and the governing Issue's Project Status is `Planned`; do not write or execute code.

Keep exactly one item in progress. Mark an item complete only after its stated evidence exists. Add or revise items when the agreed scope changes.

## Read the Issue

- Read the governing change Issue first. Use `Context`, `Goal`, `Scope`, and `Acceptance criteria` as the authority for what must be planned.
- Confirm the repository, branch, and any repository instructions before deriving tasks.
- Treat the Issue's `Context` as a hypothesis: inspect the repository and linked evidence to test its stated current state, problem, and cause. If the premise is false or incomplete, report that result and stop rather than planning around it.
- Before deriving tasks, enumerate every decision the change requires. For each decision, provide more than one defensible option, the trade-offs, and a recommendation; present unresolved decisions to the requester and record the answer. Do not write a recommendation into the plan as though it were an answer.
- Identify every undefined term on which the acceptance criteria depend. Define each term from repository evidence or raise it as a decision using the same options-and-trade-offs process.
- A plan is "verified" only when its premises have been tested and all decisions have been resolved; task-shape review alone is insufficient.
- Read `Scope` markers and `Acceptance criteria` exactly. The plan must cover included work and must not silently add excluded work.
- Treat a missing `Scope` or `Acceptance criteria` as a blocker: stop, explain what is missing, and ask to complete the Issue before planning.
- When the Issue references other Issues, Pull Requests, or release work, include what the Change flow needs and defer release work to the release Issue.

## Derive Plan Tasks

- Decompose the change into small, ordered tasks that map to the `Scope` and `Acceptance criteria`, after the premise and decision gates above are complete.
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

- Post the plan comment with the machine-readable marker `<!-- skills:plan-issue issue=<number> -->` as the first line (replacing `<number>` with the governing Issue number), followed by the required ordered plan sections. The trusted `Policy (Project)` workflow advances the governing Project item to `Planned` from this marker; do not issue a second Agent-side Status mutation.
- Deliver the plan as a comment on the governing Change Issue. Keep the comment self-contained: the tested premise and evidence, resolved decisions and requester answers, defined terms, ordered tasks with completion evidence, out-of-scope items, residual risk, and the next-phase handoff. Report the Issue URL and the comment URL; the Project workflow's `Planned` update may be asynchronous and must be verified before handoff.
- Do not hand off through a temporary or local file, and do not rely on the host's native task tracking to carry the plan.
- Complete the Todo List only after the comment is posted and its URL is available. Do not write code, create commits, or execute the plan unless separately requested.
- Simple Issues may skip this planning step. When a plan is needed, the next
  session uses `implement-issue` to execute it; `create-pr` opens the Pull
  Request in a later session.

## Writing quality

These rules bind the prose this skill writes into anything a person reads later: an Issue body, a Pull Request body, a comment, a commit message body, or a file added to the project. Code, identifiers, commands, paths, and quoted output are exempt. Where the project states its own writing guidance, that guidance governs the language of record and the terms to use; these rules are the floor when it states none.

- Choose the plain word, and choose a word people say aloud. Write `use` rather than `utilize` and `is` rather than `serves as`; a replacement nobody says fails this rule too.
- Keep one idea in one sentence. Split a sentence that makes the reader hold the first idea while parsing the second.
- Name a thing in full on first mention and reuse that exact term to the last. Define a short form before using it.
- Make every sentence add a fact the reader did not have. Delete each sentence in turn; one that loses nothing does not belong.
- Cite the file, command, or output behind every claim about the project.
- State a position and give its reason. Do not present two options and commit to neither.
- Write headings in sentence case, and use a list only for items a reader counts.
