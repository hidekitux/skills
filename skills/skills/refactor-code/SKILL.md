---
name: refactor-code
description: Perform small, behavior-preserving code refactorings verified against a test baseline. Use when asked to refactor, restructure, clean up, or rename code under a defined task or Issue while keeping observable behavior unchanged, or when a debug or review step leaves related code debt. Requires a runnable test baseline before and after the change; never changes behavior, adds features, or fixes unrelated defects.
license: Apache-2.0
---

# Refactor Code

## Todo List

1. **in progress:** Resolve the refactor task, its in-scope files, and the project's test baseline command.
2. Scope one small refactor and state the behavior that must be preserved.
3. Make the mechanical changes on the in-scope files.
4. Verify the before/after baseline results and the diff; record the evidence.
5. Complete the list only when the baseline passes before and after and the handoff is reported.

Keep exactly one item in progress. Mark an item complete only after its stated evidence exists. Add or revise items when the agreed scope changes. Use the host's native Todo List when available; otherwise maintain this list as a Markdown checklist in the conversation.

## Resolve and Guard

- Read repository instructions and the governing task or Issue before refactoring.
- Resolve the refactor target, its in-scope files, and the project's test command from the repository. Do not guess a target.
- Refactors only: never change behavior, add features, or fix unrelated defects. Feature work and defect fixes belong to separate governed work (`plan-issue` and `implement-issue`); test design belongs to `write-tests`; defect fixing belongs to `debug-code`.
- Stop and report when the task lacks a scoped boundary, the baseline cannot run, or the baseline does not pass before the change.

## Scope the Refactor

- Choose a single small refactor with a stated purpose, for example extracting a function, renaming an identifier, or simplifying a structure.
- State the behavior that must be preserved and how the test baseline observes it.
- Do not expand the refactor into a larger rewrite or combine it with unrelated work.

## Make Mechanical Changes

- Edit only the in-scope files and prefer the repository's existing patterns.
- Keep each change small and reviewable; prefer many small verified steps over one large change.

## Verify the Baseline

- Run the project's test baseline before the change and record the result; the baseline must pass before refactoring starts.
- Run the same baseline after the change and record the result.
- Review the diff to confirm that only structure changed. A failing baseline or a behavior-changing diff stops the refactor and is reported.

## Handoff

- Report the refactor target, the changed files, the before/after baseline results, and the next owner: `implement-issue` integrates the refactor into the governed change; report to the requester when the refactor is not part of a governed change.
- Never add features, fix defects, or open a Pull Request; those are separate phases.