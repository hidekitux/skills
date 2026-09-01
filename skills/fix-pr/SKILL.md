---
name: fix-pr
description: Fix review findings on an existing Pull Request and publish the result in one invocation, ending with the reviewed commit pushed as the Pull Request head. Use when a review returned findings on an open Pull Request and asked for them to be addressed, when a fix was committed but never pushed, or when unpushed commits on a reviewed branch need tidying before publication. Do not use it to implement a plan on a branch with no Pull Request, to create the first Pull Request, or to merge or release.
license: Apache-2.0
---

# Fix Pull Request

## Todo List

1. **in progress:** Resolve the repository, the open Pull Request, its head and base branches, the governing Issue, and the review artifact with its findings.
2. Record one task per finding, ordered by severity, with the fix each finding requires and the evidence that will show it resolved.
3. Fix each finding by editing only files the finding and the Issue scope justify, and commit the completed fix on the Pull Request branch.
4. Tidy unpushed commits so the history to be published is the history the plan and the findings justify.
5. Run the repository-required validation and record every command and result.
6. Push the branch and sync the Pull Request body to the published state.
7. Complete the list only when the pushed head equals the commit the next review will identify and every finding has a recorded disposition; hand off all evidence.

Keep exactly one item in progress. Mark an item complete only after its stated evidence exists. Add or revise items when the review returns new findings. Use the host's native Todo List when available; otherwise maintain this list as a Markdown checklist in the conversation.

## Resolve and Guard

- Read repository instructions, the Pull Request template, and the governing Issue before editing.
- Resolve the repository, the open Pull Request number, its exact head and base branches, and the current head SHA from local and hosting context. Do not guess a Pull Request; when none is open for the branch, stop and apply the boundary in `Phase Boundary`.
- Resolve the review artifact: the review comments, the finding list with severity, and the head SHA the review examined. Findings are the input to this skill; when no review artifact exists, stop and route to `review-pr` instead of inventing findings.
- Confirm the review examined the current head. When the head moved after the review, treat the older findings as possibly stale, say so, and re-review before fixing rather than fixing against an outdated diff.
- Fix only what a finding or the governing Issue's scope justifies. Do not add unplanned features, unrelated refactors, or out-of-scope edits, and do not silently drop a finding; every finding needs a disposition of fixed, rejected with a reason, or deferred to a named follow-up.
- Preserve unrelated user changes. Do not switch branches, stage unrelated files, or reset the worktree when doing so would include work outside the Pull Request.
- Do not create Issues, open a second Pull Request, merge, or release. `create-issue` owns Issue creation, `merge-pr` owns merging, and publication belongs to the release flow.

## Phase Boundary

The owning skill is decided by one checkable condition: whether the branch already has an open Pull Request.

```sh
gh pr list --head "$(git rev-parse --abbrev-ref HEAD)" --state open --json number --jq 'length'
```

- `0` — no open Pull Request. `implement-issue` owns the edits and `create-pr` owns first publication. Stop here and route to them; this skill has no Pull Request to fix or sync.
- `1` — an open Pull Request exists. This skill owns the work: fixing, tidying unpushed commits, validation, pushing, and body sync happen in this one invocation.
- More than `1` — ambiguous. Stop and report the Pull Request numbers instead of guessing which one to fix.

Read the condition on the branch's state, not on how the request was phrased. A request to "implement the plan" on a branch that already has an open Pull Request is still this skill's work, and a request to "fix the review findings" on a branch with no Pull Request still belongs to `implement-issue` and `create-pr`.

`implement-issue` derives its tasks from the Issue's `Scope` and `Acceptance criteria`; this skill derives its tasks from review findings. That difference in task source is why the two are separate skills and must not be merged.

## Fix

- Track one Todo item per finding and keep exactly one in progress. Address blocking findings before non-blocking ones.
- Edit only the files the finding identifies or the fix demonstrably requires. Prefer the repository's existing patterns, local helper APIs, and module boundaries.
- Commit each completed fix on the Pull Request branch at finding granularity so the history mirrors the review. Stage only that fix's files and keep unrelated changes unstaged.
- Follow the repository commit policy: a single-sentence header `type: summary #<number>`, where `type` comes from the repository commitlint enum and `<number>` is the governing Issue number. Write what changed, not that a review asked for it; `fix: address review comments #123` is not acceptable.
- Record observable evidence per finding: the files changed, the commands run, the resulting output, and the commit hash. Never claim a finding resolved from intent alone.
- When a finding reveals that the change needs work outside the Issue's scope, stop and report it as a follow-up candidate for `create-issue` instead of widening this Pull Request.

## Tidy Unpushed History

- Tidying applies only to commits that are not yet pushed. Confirm what is unpushed by comparing the local branch against its remote tracking ref before rewriting anything.
- Drop, reword, or combine unpushed commits when doing so makes the published history match the findings and the plan: a no-op commit, a commit that a later fix fully reverts, or a header that violates the commit policy.
- Never rewrite a commit that is already the Pull Request head or an ancestor of it on the remote, unless the requester explicitly authorizes that rewrite in this invocation. An already-reviewed commit is review evidence.
- After tidying, inspect the complete diff against the base and confirm that only the intended content changed. Tidying must not alter the tree that validation and review will see, except where a finding required it.
- Record the pre-tidy and post-tidy head SHAs and the reason for each dropped or reworded commit.

## Validate

- Run the repository-prescribed checks for the fixed work. Prefer the standard task runner and aggregate validation command; do not substitute narrower commands for an available aggregate task.
- Resolve validation failures inside the fix commit they belong to. Never create a validation-only adjustment commit.
- Record every command and result. State every skipped or failing check explicitly; never describe an unrun check as passing.
- Do not push a branch whose required validation is failing. Report the failure and stop instead of publishing a known-broken head.

## Push and Sync

- Push the resolved head branch to its exact remote after validation succeeds. Use `--force-with-lease` when tidying rewrote unpushed commits; never use plain `--force` and never push the protected base branch.
- After pushing, sync the Pull Request body so it describes the published state: update the `Validation` evidence to the commands actually run on this head and keep the `Issue` reference block first and unchanged.
- Validate the exact finalized body against the repository's Pull Request-body validator before the API call. For this repository, run `go run ./cmd/validate-branch-policy --base <base> --head <head> --body "$final_body"`.
- Do not re-resolve the Pull Request template from scratch, re-run duplicate-Pull-Request lookup, or re-validate the title unless the title itself changed. Those belong to first publication in `create-pr`.
- Do not update the governing Issue's Project Status. The trusted `Policy (Project)` workflow owns Pull Request-observable transitions.

## Handoff

Report the Pull Request URL and number, the head and base branches, the review artifact and the head SHA it examined, every finding with its disposition and commit hash, the pre-tidy and post-tidy head SHAs when history was tidied, the validation commands and results, the pushed head SHA, and the Pull Request body sync evidence.

This skill is complete only when the pushed head equals the commit the next review will identify, so a fix cannot be left unpushed. Verify it, do not assume it:

```sh
test "$(git rev-parse HEAD)" = "$(gh pr view <number> --json headRefOid --jq .headRefOid)"
```

If the two SHAs differ, the invocation is not finished: push the remaining commits and re-check. Never hand off a branch whose local head is ahead of the Pull Request head; that is the state in which `merge-pr` blocks on an unpushed fix.

The next owner is `review-pr`, which re-reviews the pushed head. Do not merge or release; `merge-pr` owns merging after the re-review passes.
