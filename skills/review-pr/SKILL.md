---
name: review-pr
description: Review a GitHub Pull Request and report only findings the change introduces, grounded in the change's contract and backed by a concrete failure scenario, ordered by severity (bugs, regressions, and missing tests over style), applying standardized project-specific criteria when a project declares them. Use when asked to review, inspect, or evaluate a Pull Request or its changes before merge. Do not edit the reviewed branch, push fixes, or merge; findings return to implement-issue for fixes on the same Issue branch.
license: Apache-2.0
---

# Review Pull Request

## Todo List

1. **in progress:** Resolve the repository, the Pull Request to review, its linked Issue, its head and base branches, and the review boundary.
2. Assemble the change contract, map the diff and its impact scope, trace high-risk paths, and verify failure hypotheses; collect evidence for every candidate finding.
3. Filter candidates through the adoption gate, produce minimal findings in severity order, and state the applied criteria and their sources.
4. Complete the list only when every finding has evidence and the review is handed off, or report the remaining findings and risks in the handoff.

Keep exactly one item in progress. Mark an item complete only after its stated evidence exists. Add or revise items when the agreed scope changes. Use the host's native Todo List when available; otherwise maintain this list as a Markdown checklist in the conversation.

## Resolve and Guard

- Read repository instructions and Pull Request templates before reviewing.
- Resolve the repository, the target Pull Request, and its head and base branches from local and hosting context; link it to its governing Issue. Do not guess a Pull Request to review. Fetch or check out the base branch before judging causality.
- Reviews only: never edit the reviewed branch, stage files, push fixes, rebase, or merge. `implement-issue` owns and applies the fixes on the same Issue branch.
- Do not plan or implement the reviewed change (`plan-issue` and `implement-issue` own those phases) and do not create Issues (`create-issue` owns Issue creation).
- Stop and report when the target Pull Request or its diff cannot be resolved, or when the review boundary is unclear.

## Review

Review in this order and follow this flow rather than a pre-existing checklist.

### 1. Assemble the change contract

- Build the contract the change must satisfy from the PR body, the linked Issue, specifications, and the existing code before reviewing. Report drift as a finding when the diff implements a different feature than intended or deviates from the Issue's acceptance criteria.
- Apply the standards the review uses. Resolve criteria in this order: explicit requester input, the project criteria file, criteria derived from the change, then the built-in baseline. Ask the requester when criteria are ambiguous or conflict. See [review criteria](references/review-criteria.md).
- List the applied criteria and their sources at the top of the review report.

### 2. Map the diff and its impact scope

- Read the complete diff from head to base. Only report problems this PR introduces or makes reachable; verify causality against the base branch and normally exclude pre-existing issues.
- Check callers, plugins, generated code, serialization formats, schemas, defaults, environment variables, documentation, and samples outside the changed files; confirm existing users can satisfy the new assumptions.

### 3. Trace high-risk execution paths and state changes

- Trace input through validation, state change, persistence, external effects, and response. Check empty and boundary values, retries, partial failure, caller and callee contracts, forbidden states, async ordering, duplicate execution, and cross-store inconsistency. Catch problems where each line is correct but the combination breaks.
- Identify invariants that must always hold (for example, no secret data before authentication, no success state on failure, idempotent retries, preserved public API compatibility, resources released on all paths) and check the diff against them.
- Load specialized checks only when the PR content calls for them, from the dedicated reference for that topic, instead of always loading full checklists:
  - [Security review](references/security-review.md): authentication, secrets, permissions, untrusted input, data exposure.
  - [Performance review](references/performance-review.md): hot paths, loops, I/O, queries, large data handling.
  - [Accessibility review](references/accessibility-review.md): user-facing HTML, UI, interaction semantics.

### 4. Verify failure hypotheses

- For each candidate, state the input or state, the path, what actually breaks, why existing defenses and tests do not prevent it, and which part of the diff causes it. Drop candidates that only say "it might happen"; a finding without a minimal reproduction scenario is an investigation note, not a review finding.
- Evaluate tests by whether they would actually fail on regression (shared wrong assumptions, assertion depth, over-mocking, missing branches and failure paths), not by existence or coverage numbers. Report a missing test only when it lets a concrete missed bug through.
- Verify the Pull Request's validation claims by re-running the repository's prescribed checks where possible. Record every command and its result; never describe an unrun check as passing. This evidence is separate from failure-hypothesis verification.

### 5. Filter through the adoption gate

- Exclude candidates that are not PR-introduced, have no realistic execution path, have no observable adverse effect, have no concrete code location, are not actionable at developer granularity, are a preference, refactor request, or explanation request, or are already prevented by tests or existing mechanisms. Accept "no issues" as a valid outcome.

### 6. Output minimal findings

- Keep the fixed baseline ordering: bugs, regressions, and missing tests over style. Assign each accepted finding a severity tag, Critical (blocks merge: a bug, regression, security defect, or missing test) through High, Medium, and Low (style and nits), so severity sorts to a stable order.
- Report severity (impact and blast radius) and confidence (evidence strength from code path, tests, and execution results) separately; do not present a high-impact hypothesis as a certain critical bug.

## Findings

- Each finding is short and self-contained: severity tag, confidence, one-line claim, trigger scenario, cause, and file and line location. Omit long review summaries, diff summaries, praise, and full fix code. Group findings that share one root cause and give each a required action.
- Report the applied criteria and their sources at the top, then the findings, then the validation evidence.
- Post the findings on the Pull Request or in the linked conversation so `implement-issue` can act on them.

## Handoff

- Report the Pull Request URL, the applied criteria and their sources, the finding list with severity and confidence and file and line evidence, the validation commands and results, and the next owner: `implement-issue` fixes the findings on the same Issue branch, after which the updated Pull Request is re-reviewed until the findings are resolved.
- Never merge, release, or apply fixes; those are later phases owned by other skills or by the user, not part of this skill.
