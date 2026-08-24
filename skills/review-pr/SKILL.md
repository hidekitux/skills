---
name: review-pr
description: Review a GitHub Pull Request systematically and return evidence-backed findings ordered by severity, prioritizing bugs, regressions, and missing tests over style. Use when asked to review, inspect, or evaluate a Pull Request or its changes before merge. Do not edit the reviewed branch, push fixes, or merge; findings return to implement-issue for fixes on the same Issue branch.
license: Apache-2.0
---

# Review Pull Request

## Todo List

1. **in progress:** Resolve the repository, the Pull Request to review, its linked Issue, its head and base branches, and the review boundary.
2. Read the Pull Request, its commits, and the complete base diff; check the change against the linked Issue's Scope and Acceptance criteria and collect evidence for every finding.
3. Produce findings ordered by severity with file and line evidence, and post the review.
4. Complete the list only when every finding has evidence and the review is handed off, or report the remaining findings and risks in the handoff.

Keep exactly one item in progress. Mark an item complete only after its stated evidence exists. Add or revise items when the agreed scope changes. Use the host's native Todo List when available; otherwise maintain this list as a Markdown checklist in the conversation.

## Resolve and Guard

- Read repository instructions and Pull Request templates before reviewing.
- Resolve the repository, the target Pull Request, and its head and base branches from local and hosting context; link it to its governing Issue. Do not guess a Pull Request to review.
- Reviews only: never edit the reviewed branch, stage files, push fixes, rebase, or merge. `implement-issue` owns and applies the fixes on the same Issue branch.
- Do not plan or implement the reviewed change (`plan-issue` and `implement-issue` own those phases) and do not create Issues (`create-issue` owns Issue creation).
- Stop and report when the target Pull Request or its diff cannot be resolved, or when the review boundary is unclear.

## Review

- Inspect the Pull Request's status, commits, and the complete base diff. Compare the change with the linked Issue's Scope and Acceptance criteria and record scope drift as a finding.
- Prioritize bugs, regressions, and missing tests over style. Check correctness, error handling, security, and the diff for credentials, tokens, private URLs, and generated noise before commenting on style; raise style only when it blocks correctness, maintenance, or review.
- Verify the Pull Request's validation claims by re-running the repository's prescribed checks where possible. Record every command and its result; never describe an unrun check as passing.
- Record evidence for every finding: the exact file path and line(s), the observed behavior, and the expected behavior. Do not report a finding without file or line evidence.

## Findings

- Order findings by severity: Critical (blocks merge: a bug, regression, security defect, or missing tests), High, Medium, then Low (style and nits).
- Group findings that share one root cause, and give each finding a required action and a suggested next step.
- Post the findings on the Pull Request or in the linked conversation so `implement-issue` can act on them.

## Handoff

- Report the Pull Request URL, the finding list with severity and file and line evidence, the validation commands and results, and the next owner: `implement-issue` fixes the findings on the same Issue branch, after which the updated Pull Request is re-reviewed until the findings are resolved.
- Never merge, release, or apply fixes; those are later phases owned by other skills or by the user, not part of this skill.