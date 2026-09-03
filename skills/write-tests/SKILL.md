---
name: write-tests
description: Design and add focused, proportionate tests for a defined feature, task, or verified failure. Choose the test level, derive test cases with completion and failure evidence, and record observable results. Use when asked to write, add, or design tests for code from requirements or an Issue. Tests only; never fix production code, and do not take over a project's full test suite. For stateful flows with an FSL spec, derive acceptance and conformance tests from `fslc scenarios` and `fslc testgen` without claiming FSL proves implementation correctness.
license: Apache-2.0
---

# Write Tests

## Todo List

1. **in progress:** Resolve the behavior under test (Issue, requirements, or verified failure scenario), the repository, and the test boundary.
2. Choose the test level and derive test cases with completion and failure evidence for each case.
3. Write and run the tests; record the command, the result, and per-case evidence.
4. Complete the list only when every case has evidence and the work is handed off, or report the remaining cases and risks in the handoff.

Keep exactly one item in progress. Mark an item complete only after its stated evidence exists. Add or revise items when the agreed scope changes. Use the host's native Todo List when available; otherwise maintain this list as a Markdown checklist in the conversation.

## Resolve and Guard

- Read repository instructions and the target's requirements, Issue, or plan before designing tests.
- Resolve what behavior is under test from the defined task: the acceptance criteria, requirements, or a verified failure scenario. Do not invent scope; this skill works from a defined task, and `plan-issue` and `implement-issue` own feature implementation.
- This is a fix-layer skill: it adds or updates tests directly and works from a defined task or Issue. Related skills: `debug-code` (isolated fix that may need a regression test), `refactor-code` (refactor verified against this skill's test baseline), and `implement-issue` (the handoff target for the resulting test changes).
- Tests only: never modify production code, and do not run or own a project's complete test suite. Fixing the feature under test belongs to `debug-code` for an isolated bug or to `implement-issue` for a governed change.
- Stop and report when the behavior under test is undefined or the failure scenario cannot be reproduced; do not guess a test target.

## Choose the test level

Pick the smallest level that reliably verifies the defined behavior:

- **Unit** — one unit in isolation, for logic with clear inputs and outputs.
- **Integration** — units through their real boundary, for contracts between modules, storage, or external interfaces.
- **End-to-end** — user-visible behavior, for flows that cross boundaries or for acceptance criteria stated in user terms.

Use one level unless the acceptance criteria genuinely span levels. Record the choice and its reason with the test cases.

## Derive test cases

Map each acceptance criterion, requirement, or failure scenario to at least one test case. Derive cases from the target's inputs and states, including boundary and empty values, state changes, error paths, retries, ordering, and duplicate execution, plus the verified failing scenario when one exists. Drop cases that do not map to a requirement or scenario.

For a stateful flow driven by an FSL specification, derive acceptance and conformance tests from the spec:

- Run `fslc scenarios <spec.fsl>` to enumerate concrete step scenarios and their per-step expected states.
- Run `fslc testgen <spec.fsl>` to emit an adapter-based conformance test skeleton for the target harness (default `pytest`; `--target` supports `vitest`, `swift`, `kotlin`, `dart`, `phpunit`; `-o` sets the output file). Wire the adapter to the real implementation. Place options after the file path; fslc 4.2.0 errors when options precede it.
- Treat the generated cases as acceptance and conformance tests for the state flow. FSL derives tests from the spec; it does not prove the implementation conforms. Only executed tests against the real implementation, with recorded results, are evidence.

## Record completion and failure evidence per case

For each test case, record an intention (which requirement or scenario it verifies), the precondition and input, the expected behavior, and the failure evidence (the command and output, or the specific assertion, that would fail). A case completes only when its evidence exists: the test file path, the run command, and the pass or fail result. For a failing case, record the failing assertion or output as failure evidence; that evidence is the deliverable, not a passing status.

## Run and record

Run tests with the repository's prescribed commands, using mise tasks when the repository exposes them. Record every command and its result; never describe an unrun check as passing. Prefer deterministic evidence — file paths, commands, and outputs — over prose claims.

## Handoff

- Report the chosen test level and reason, the test cases with intention, completion and failure evidence, the changed test file paths, the commands run with their results, and the next owner: `implement-issue`, which takes the focused tests into the verified fix or governed change.
- Never fix production code, never take over a project's whole test suite, and never create Issues or Pull Requests; those belong to other skills or later phases.

## Writing quality

The handoff report this skill writes is prose a person reads. Choose the plain word and a word people say aloud, keep one idea in one sentence, name a thing in full on first mention and reuse that exact term, make every sentence add a fact the reader did not have, and cite the file, command, or output behind every claim about the project. Where the project states its own writing guidance, that guidance governs the language of record and the terms to use; these rules are the floor when it states none.
