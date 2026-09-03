---
name: debug-code
description: Debug a failure from reproduction through root cause, fix, and verification. Use when asked to debug, reproduce, isolate, root-cause, or fix a defect; complete each workflow step only with observable evidence.
license: Apache-2.0
---

# Debug Code

## Todo List

1. **in progress:** Reproduce the failure and record its reproduction evidence.
2. Isolate the failure and confirm the root cause with evidence before proposing a fix.
3. Hypothesize a cause and reduce the failure to a minimal reproduction that tests it.
4. Apply the minimal fix for the isolated defect.
5. Verify the fix against the reproduction and the project's validation, then hand off.

Keep exactly one item in progress. Mark an item complete only after its stated result or evidence exists: a command and its output, a failing and then passing check, or another observable artifact. Add or revise items when the observed failure changes. Use the host's native Todo List when available; otherwise maintain this list as a Markdown checklist in the conversation.

## Resolve and Guard

- Reproduce the failure first. Record how to trigger it, what happened, and what should have happened. Do not propose a fix before a reproduction exists.
- Isolate before fixing: narrow the failure to a single defect using binary search, logging, boundaries, or input reduction. The root cause must have evidence before a fix is proposed.
- Push each hypothesis to a minimal reproduction so the fix can be verified deterministically. Prefer the smallest input, call path, or configuration that still fails.
- Fix only the isolated defect. Do not design tests or refactor code: regression tests belong to `write-tests`, cleanup of the root-cause area belongs to `refactor-code`, and a governed fix enters the artifact flow at `create-issue` and continues through `plan-issue` and `implement-issue`.
- Preserve unrelated user changes. Do not rework areas outside the isolated defect, add features, or change behavior you were not asked to fix.
- Stop and report when the failure cannot be reproduced, the root cause cannot be evidenced, or the project's validation cannot run; never claim completion from intent alone.

## Debug

### Reproduce

- Run the failing command or scenario and record the trigger, the observed output, and the expected behavior. The reproduction is the baseline every later step is measured against.

### Isolate

- Narrow the failure to a single root cause: bisect inputs or commits, enable tracing at suspected boundaries, and confirm each eliminated candidate with a check. Record the evidence that identifies the defect.

### Hypothesize and minimize

- State the suspected cause as a testable hypothesis, then build or run the smallest reproduction that confirms it. Record the minimal reproduction command and its output; a hypothesis without a minimal reproduction is unconfirmed.

### Fix

- Apply the minimal change that addresses the isolated defect. Keep the change scoped: no test creation (`write-tests`), no restructuring (`refactor-code`), no scope expansion.

### Verify

- Re-run the reproduction and confirm the failure is gone. Then run the project's prescribed validation for the change and record every command and result. Verification evidence must come from actual runs.

## Validate and Handoff

- Run the repository-prescribed checks for the completed work. Record every command and result; state skipped or failing checks explicitly and never describe an unrun check as passing.
- Report the reproduction, the root-cause evidence, the minimal reproduction, the fix, the verification results, and any remaining risks or unfinished steps.
- Hand off follow-ups to `write-tests` (regression tests for the verified fix) or `refactor-code` (related debt left in the root-cause area). When the fix is a governed change, hand the reproduction and root-cause evidence to `implement-issue`; issue creation and planning happened in earlier sessions with `create-issue` and `plan-issue`. Do not create Issues, open Pull Requests, or release unless the user separately requests it.

## Writing quality

The handoff report this skill writes is prose a person reads. Choose the plain word and a word people say aloud, keep one idea in one sentence, name a thing in full on first mention and reuse that exact term, make every sentence add a fact the reader did not have, and cite the file, command, or output behind every claim about the project. Where the project states its own writing guidance, that guidance governs the language of record and the terms to use; these rules are the floor when it states none.
