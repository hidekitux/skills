---
name: deliver-change
description: Deliver a governed Change Issue end to end from its verified plan through a reviewed Pull Request and cohesive final report. Use when asked to take a Change Issue to a finished, reviewed change as one coordinated run, or when a plan exists and implementation, pull request, and review should proceed without selecting each skill manually.
license: Apache-2.0
---

# Deliver Change

## Todo List

1. **in progress:** Resolve the governing Change Issue, its verified plan, the base revision, and the in-scope boundary.
2. Run the planning phase with `plan-issue`, or confirm the plan already posted on the Issue, and record its URL.
3. Execute the plan with `implement-issue` and record per-task evidence.
4. Open and review the change with `create-pr` and `review-pr`, then rework with `fix-pr` until the review loop terminates.
5. Complete the list only when every phase artifact exists and one cohesive final report is delivered; hand off the Issue URL, Pull Request URL, and per-phase evidence.

Keep exactly one item in progress. Mark an item complete only after its stated evidence exists: a file, a command result, an Issue or Pull Request URL, or another observable artifact. Add or revise items when the agreed scope changes. Use the host's native Todo List when available; otherwise maintain this list as a Markdown checklist in the conversation.

## Resolve and Guard

- Resolve the repository, the linked Change Issue, the verified plan, and the Issue-backed branch created from the upstream base. Do not guess a plan; when the Issue has no plan, derive the tasks from its `Scope` and `Acceptance criteria` only.
- Route every phase through its owning primitive. This entry point coordinates phases and tracks a single progress model; it never plans, implements, opens Pull Requests, or reviews by itself.
- Do not create the Issue: a governed Change Issue already exists by definition. If no Change Issue exists, stop and report instead of inventing one.
- Preserve approval boundaries: `create-pr` opens the Pull Request only after user confirmation, `implement-issue` edits only in-scope files from the verified plan, and `fix-pr` edits only what a review finding or that same boundary justifies.
- Stop and report when the plan, scope boundary, or branch is unavailable or when a phase cannot produce its artifact; never continue past an unverified handoff.

## Route

Run the phases in order. A phase is complete only when its primitive's artifact exists with evidence.

| Phase | Primitive | Read-only | Continue when | Artifact |
| --- | --- | --- | --- | --- |
| Plan | `plan-issue` | yes | plan posted as an Issue comment | verified plan |
| Implement | `implement-issue` | no (edits in-scope files only) | per-task evidence recorded | in-scope changes |
| Pull request | `create-pr` | no | Pull Request opened after user confirmation | Pull Request URL |
| Review | `review-pr` | yes | findings returned | severity-ordered findings |
| Fix | `fix-pr` | no (edits what findings justify) | fixed head pushed and body synced | published fixed head |

## State and authority contract

- **One progress model:** maintain one Todo List for the whole run and map each primitive's own progress into it; the user never tracks a separate list per phase.
- **External mutation authority:** only `create-pr` creates the Pull Request, pausing for user confirmation; `fix-pr` updates that Pull Request but never creates one. `implement-issue` edits only files inside the verified plan's in-scope boundary.
- **Read-only phases:** plan and review never mutate repository files or external work items.
- **Handoff rule:** every phase hands its artifact to the next owning primitive; an incomplete artifact returns to its owning primitive for rework, never to the entry point's own logic.
- **Loop termination:** when `review-pr` returns findings, send them to `fix-pr`, which fixes, validates, and pushes on the same branch in one pass, then re-review. Terminate when `review-pr` reports no blocking findings, or after two complete rework passes; then stop and report the unresolved findings with evidence instead of continuing.

## Validate and Handoff

- Run the repository-prescribed checks after implementation (`mise run validate:all`, plus `mise run validate:skill-creator` when available) and record every command and result; never describe an unrun check as passing.
- Deliver one cohesive final report: the achieved outcome, the plan URL, phase artifacts and their URLs, validation commands and results, the governing Issue's label state, and any remaining risks.
- Hand off only when the change is implemented, validated, and reviewed. Do not merge, release, or expand scope unless the user separately requests it.