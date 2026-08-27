---
name: merge-pr
description: Merge an approved, validated GitHub Pull Request through the repository's protected-branch policy, resolving rebase conflicts when explicitly authorized and safe. Use when asked to merge a Pull Request after review and required checks pass; do not use it to create, perform substantive feature fixes, or release a Pull Request.
license: Apache-2.0
---

# Merge Pull Request

## Todo List

1. **in progress:** Resolve the repository, target Pull Request, linked Issues, head and base branches, and the requested merge authority.
2. Verify the Pull Request is merge-ready: the repository's review mode, required review evidence or approval, no blocking or unresolved findings, required checks, branch freshness, permissions, and repository merge policy.
3. If the Pull Request conflicts, obtain or confirm explicit authorization for a branch rewrite, resolve only integration conflicts, revalidate the changed Pull Request, and obtain a fresh review when the diff changed.
4. Merge the Pull Request using the repository-approved method and verify the resulting merged state and commit.
5. Reconcile linked Issue and Project outcomes, including the different handling of change and release Issues.
6. Complete the list only when the merge result, commit, linked Issue effects, and any remaining release follow-up are observable; hand off all evidence.

Keep exactly one item in progress. Mark an item complete only after its stated evidence exists. Use the host-native Todo List when available; otherwise maintain this list as a Markdown checklist in the conversation.

## Resolve and Guard

- Read repository instructions, branch-policy configuration, Pull Request templates, and the Pull Request's linked Issue before mutating GitHub state. Resolve the exact repository and Pull Request from user input or unambiguous local context; never guess a Pull Request.
- Confirm the target is an open, non-draft Pull Request with the expected head and base branches. Require an allowed branch-policy route and do not merge into a protected branch except through the repository's approved Pull Request mechanism.
- Confirm the Pull Request is linked to the governing Issue. For human change work, require the Issue number encoded in the `issue/<number>` branch and the opening `Closes #<number>` reference. A release Pull Request uses `Tracks #<number>` and must not close the Release Issue.
- Resolve the repository's review mode from explicit requester input, repository review policy, or the live Ruleset before applying the approval gate. In team mode, require approval from an eligible reviewer other than the PR author. In solo mode, require a completed self-review artifact while continuing to enforce the live Ruleset and every required check.
- Do not infer solo mode merely because no reviewer is available. If the live Ruleset requires an approval that cannot be satisfied, stop and report the exact blocker; never use an administrative bypass.
- Do not merge a Pull Request that is missing its governing Issue, has scope drift, has unresolved review findings, lacks the required review evidence or approval for its mode, or is still awaiting required checks. Route missing implementation or review work to `implement-issue`, `create-pr`, or `review-pr` instead of compensating here.
- Preserve unrelated work. Do not modify unrelated files, create substantive feature fixes, close Issues manually, or delete branches. Conflict resolution is allowed only after the requester explicitly authorizes it for this merge; if authorization is absent, stop and ask before rewriting the Pull Request branch.

## Merge Gate

- Inspect the complete Pull Request metadata, commits, diff, linked Issues, review threads, and latest check runs. Check the current head SHA immediately before merging and use it as the expected revision where the host supports that guard.
- Apply the resolved review mode: team mode requires explicit approval from a permitted reviewer other than the PR author; solo mode requires a completed self-review artifact identifying the reviewed head SHA, applied criteria, validation evidence, and findings result. A comment, a dismissed approval, an approval from an insufficiently privileged reviewer, or an approval that became stale after synchronization is not sufficient.
- Require all repository-mandated checks to pass on the current head. This includes branch-policy, title, commit convention, signature, repository validation, lint, tests, FSL, and workflow-security checks when the repository declares them. Do not treat skipped, pending, cancelled, stale, or unavailable checks as passing.
- Confirm the Pull Request is mergeable and up to date with its base when the repository requires that condition. If it conflicts, follow `Conflict Resolution` instead of bypassing the gate. If the head changes during inspection, restart the gate from the metadata and check verification steps; never merge a stale inspection result.
- Check for unresolved conversations, blocking labels, merge freezes, required deployment or environment approvals, and repository-specific release gates. When policy or permissions are ambiguous, stop and report the exact blocker rather than guessing.
- Record each inspected source, command, current head SHA, and result. Never claim a check, approval, or policy condition was satisfied without observable evidence.

## Conflict Resolution

- Resolve conflicts only when the requester has explicitly authorized conflict resolution in this invocation. Do not infer authorization merely from a request to merge.
- Resolve the exact remote head and base branches, fetch both, and record the pre-rewrite head SHA. Confirm the worktree is clean or use an isolated worktree; never mix unrelated local changes into the Pull Request branch.
- Rebase the Pull Request branch onto the latest allowed base branch. For this repository, use the branch's exact remote and `--force-with-lease` after resolution; never use plain `--force` and never push the protected base branch.
- Resolve each conflict from the Pull Request's Issue scope, existing behavior, and review evidence. Do not use blanket `ours` or `theirs`, discard review fixes, or invent feature behavior. If intent is ambiguous, abort the rebase and report the files and decision needed.
- After resolving conflicts, inspect the complete new diff against the base, verify that only the intended conflict-resolution changes occurred, run the repository-prescribed validation, and record every command and result. Continue only if validation succeeds.
- Because conflict resolution changes the reviewed commit graph or diff, require a fresh review or explicit repository-approved re-review evidence before merging. If the review identifies substantive drift, return the branch to `implement-issue` instead of proceeding.
- Re-read the Pull Request metadata and check runs after the force-with-lease update. The merge gate starts over against the new head SHA.

## Merge

- Use the repository's declared merge method. For this repository, use **rebase merge only** so compliant commit messages remain the commits reaching `main`; never use merge commits or squash merge unless the repository policy is explicitly changed.
- Prefer the host's guarded merge operation with the resolved Pull Request number and current head SHA. With GitHub CLI, use `gh pr merge <number> --rebase` only after the merge gate passes; do not use `--admin`, bypass required checks, or bypass branch protection.
- Do not merge drafts, still-conflicted Pull Requests, or Pull Requests with failing, pending, or missing required checks. If the merge operation is rejected, capture the exact host diagnostic and do not retry by bypassing policy.
- After the operation, re-read the Pull Request and the resulting commit. Confirm `merged=true`, the merged timestamp, the merge commit SHA, the base branch, and the expected head commit ancestry. If the result is ambiguous, stop and report it; do not issue a second merge request.
- Do not publish a release, create or move tags, edit release notes, or close a Release Issue. Those actions belong to the release workflow after a release Pull Request has merged and its publication gates pass.

## Linked Work and Project State

- For a change Pull Request using `Closes`, verify that GitHub closed the linked change Issue and that its Project item reached `Done` through the repository's configured automation. If automation is delayed, report the observed state and do not manually force a terminal state without explicit repository ownership and policy.
- For a release Pull Request using `Tracks`, verify that the Release Issue remains open and its Project Status remains non-terminal (`In review` or the repository's documented equivalent) until publication succeeds. Hand off to the release/publishing owner with the merge commit and remaining publication gate.
- If multiple Issues are linked, verify each closing or tracking behavior independently. Report any Issue or Project mismatch as a follow-up rather than silently changing it.

## Handoff

Report the Pull Request URL and number, repository, head and base branches, resolved review mode, whether conflict resolution was performed, pre-rewrite and pre-merge head SHAs, resulting merge commit SHA, conflict files and resolution evidence, self-review or external approval evidence, required-check evidence, linked Issue outcomes, Project Status evidence, and any release publication or automation follow-up. State clearly whether the merge completed, was blocked, or has an ambiguous result. The next owner for a merged change is the repository's post-merge verification or release workflow; the next owner for a tracked release Pull Request is the release publication process. Do not apply substantive review fixes or publish a release in this skill.
