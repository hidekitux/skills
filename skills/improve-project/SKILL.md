---
name: improve-project
description: Improve a project end to end from read-only analysis through an Issue-backed change, reviewed Pull Request, and cohesive final report. Use when asked to improve, assess, and act on a project as one coordinated run, or to reach a finished improvement without selecting each governed-change skill manually.
license: Apache-2.0
---

# Improve Project

## Todo List

1. **in progress:** Resolve the repository, the target improvement, and the scope boundary; confirm the approvals each phase requires.
2. Run the read-only analysis phase with `analyze-project` and record its findings report.
3. Convert the agreed finding into a governed change through `create-issue`, `plan-issue`, and `implement-issue`.
4. Open and review the change through `create-pr` and `review-pr`, reworking until the review loop terminates.
5. Complete the list only when every phase artifact exists and one cohesive final report is delivered; hand off the Issue URL, Pull Request URL, and per-phase evidence.

Keep exactly one item in progress. Mark an item complete only after its stated evidence exists: a file, a command result, an Issue or Pull Request URL, or another observable artifact. Add or revise items when the agreed scope changes. Use the host's native Todo List when available; otherwise maintain this list as a Markdown checklist in the conversation.

## Resolve and Guard

- Resolve the repository, the user's target improvement, and the boundary of that improvement before any phase starts. Do not expand the improvement beyond what the user requested.
- Route every phase through its owning primitive skill. This entry point coordinates phases and tracks a single progress model; it never performs a phase's work itself and never duplicates a primitive's instructions.
- Preserve approval boundaries: read-only phases stay read-only, and every external mutation (Issue creation, Pull Request creation) happens only through the owning primitive with the user's confirmation.
- Keep direct primitive invocation available: do not remove or shadow the primitives this entry point coordinates.
- Stop and report when a phase cannot produce its artifact or approval is withheld; never continue past an unverified handoff.

## Route

Run the phases in order. A phase is complete only when its primitive's artifact exists with evidence.

| Phase | Primitive | Read-only | Continue when | Artifact |
| --- | --- | --- | --- | --- |
| Analyze | `analyze-project` | yes | findings report exists | prioritized findings |
| Issue | `create-issue` | no | Issue created after user confirmation | change Issue |
| Plan | `plan-issue` | yes | plan posted as an Issue comment | verified plan |
| Implement | `implement-issue` | no (edits in-scope files only) | per-task evidence recorded | in-scope changes |
| Pull request | `create-pr` | no | Pull Request opened after user confirmation | Pull Request URL |
| Review | `review-pr` | yes | findings returned | severity-ordered findings |

## State and authority contract

- **One progress model:** maintain one Todo List for the whole run and map each primitive's own progress into it; the user never tracks a separate list per phase.
- **External mutation authority:** only `create-issue` creates Issues and only `create-pr` creates Pull Requests, each pausing for user confirmation. `implement-issue` edits only files inside the verified plan's in-scope boundary.
- **Read-only phases:** analyze, plan, and review never mutate repository files or external work items.
- **Handoff rule:** every phase hands its artifact to the next owning primitive; an incomplete artifact returns to its owning primitive for rework, never to the entry point's own logic.
- **Loop termination:** when `review-pr` returns findings, send them back to `implement-issue` on the same branch, update the Pull Request through `create-pr`, and re-review. Terminate when `review-pr` reports no blocking findings, or after two complete rework passes; then stop and report the unresolved findings with evidence instead of continuing.

## Validate and Handoff

- Run the repository-prescribed checks after implementation (`mise run validate`, plus `mise run validate-skill-creator` when available) and record every command and result; never describe an unrun check as passing.
- Deliver one cohesive final report: the achieved outcome, phase artifacts and their URLs, validation commands and results, the governing Issue's label state, and any remaining risks.
- When the improvement is complete, no further skill handoff is required. If the user requested only a partial workflow, stop at that boundary and hand off the partial artifacts. Do not merge, release, or expand scope unless the user separately requests it.