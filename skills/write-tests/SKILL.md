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

## Decide property-based testing

Use property-based testing when a behavior has a stable invariant and a useful input or state space. Look for parsers and serializers that must preserve a grammar or a round trip, conversions that preserve units or bounds, state machines with legal transitions, validators with a broad invalid domain, ordering and deduplication, and transformations whose examples cover only a few combinations.

Keep example tests when the behavior is ordinary CRUD wiring, a thin wrapper, unstable external behavior, visual output, or a case whose setup and review cost exceeds the risk it reduces. Do not add a property because a tool makes one possible.

Before writing a property, record the invariant, generated input domain, expected relation, and reason examples are insufficient. Configure a bounded generator and shrink strategy. Retain the smallest failing input or seed, tool version, and reproduction command. A passing property is evidence only when its invariant and input domain are explicit.

## Decide implementation mutation

Implementation mutation changes production code and checks whether the test suite detects each changed behavior. It is separate from specification mutation: an FSL mutation changes a specification and remains owned by the FSL workflow. Do not claim implementation coverage from FSL mutation results.

Inspect the target project's manifest, lockfile, `mise.toml`, test configuration, and existing mutation reports before proposing a tool. Use the existing tool when it meets the result contract. If no tool exists, propose a development-only dependency only after checking its license, pinning its version, and recording its cost and support boundary.

Normalize the mutation report into these outcomes:

| Outcome | Meaning | Completion rule |
| --- | --- | --- |
| killed | The test suite detects the mutant. | Record the test command and result. |
| survived | The test suite accepts the changed behavior. | Triage the mutant with a reason and a fix plan or accepted disposition. |
| skipped | The tool did not run the mutant by selection or policy. | Report the skip; it is not a pass. |
| timed out | The mutation run exceeded its wall-clock limit. | Report the limit and treat the result as incomplete. |
| infrastructure error | Setup, execution, parsing, or reporting failed. | Report the error and do not claim test effectiveness. |

Do not collapse these outcomes into a percentage or aggregate green status. A survivor needs explicit review before handoff. Record whether the survivor is equivalent, redundant, accepted with a reason, or assigned a fix. Use `needs-review` for an unresolved survivor and keep the run incomplete.

## Route optional checks through mise

Add `test:property` and `test:mutation` only when the adoption decision selects them. Route both tasks through `mise` and include them in the project's aggregate check only when they apply. Keep the ordinary test task separate so an unrelated change does not pay for mutation analysis.

For a Go project, use its existing fuzz target or property package and configured implementation mutation runner. For a Python project, use the existing test runner and configured property or mutation package. For a JavaScript or TypeScript project, use the existing test runner and configured property or mutation package. In each case, pin the development tool in the project's `mise.toml`, package manifest, or lockfile, then record the license and exact command in the task.

Example task names are stable across these technologies, but command lines are project-specific:

```toml
[tasks."test:property"]
run = "<pinned property runner>"

[tasks."test:mutation"]
run = "<pinned implementation mutation runner>"
```

Do not present the placeholders as runnable commands. Replace them only after the target project's tool and version are known.

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

- Report the chosen test level and reason, the adoption decision for property-based testing and implementation mutation, the test cases with intention, completion and failure evidence, the changed test file paths, the commands run with their results, and the next owner: `implement-issue`, which takes the focused tests into the verified fix or governed change.
- Never fix production code, never take over a project's whole test suite, and never create Issues or Pull Requests; those belong to other skills or later phases.

## Writing quality

Use plain, active, evidence-backed prose in the final handoff. Keep headings in
sentence case and name the file, command, or output behind each repository claim.
