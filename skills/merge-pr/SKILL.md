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
- Do not merge a Pull Request that is missing its governing Issue, has scope drift, has unresolved review findings, lacks the required review evidence or approval for its mode, or is still awaiting required checks. Route unresolved review findings to `fix-pr`, and missing implementation or review work to `implement-issue`, `create-pr`, or `review-pr`, instead of compensating here.
- Preserve unrelated work. Do not modify unrelated files, create substantive feature fixes, close Issues manually, or delete branches. Conflict resolution is allowed only after the requester explicitly authorizes it for this merge; if authorization is absent, stop and ask before rewriting the Pull Request branch.

## Merge Gate

- Inspect the complete Pull Request metadata, commits, diff, linked Issues, review threads, and latest check runs. Check the current head SHA immediately before merging and use it as the expected revision where the host supports that guard.
- Apply the resolved review mode: team mode requires explicit approval from a permitted reviewer other than the PR author; solo mode requires a completed self-review artifact identifying the reviewed head SHA, applied criteria, validation evidence, and findings result. A comment, a dismissed approval, an approval from an insufficiently privileged reviewer, or an approval that became stale after synchronization is not sufficient.
- Require all repository-mandated checks to pass on the current head. This includes branch-policy, title, commit convention, signature, repository validation, lint, tests, FSL, and workflow-security checks when the repository declares them. Do not treat skipped, pending, cancelled, stale, or unavailable checks as passing.
- Confirm the Pull Request is mergeable and up to date with its base when the repository requires that condition. If it conflicts, follow `Conflict Resolution` instead of bypassing the gate. If the head changes during inspection, restart the gate from the metadata and check verification steps; never merge a stale inspection result.
- Check for unresolved conversations, blocking labels, merge freezes, required deployment or environment approvals, and repository-specific release gates. When policy or permissions are ambiguous, stop and report the exact blocker rather than guessing.
- Record each inspected source, command, current head SHA, and result. Never claim a check, approval, or policy condition was satisfied without observable evidence.

## Conflict Resolution

When the Pull Request conflicts, read [integration and post-merge rules](references/integration.md).

## Merge

After the merge gate passes, read [integration and post-merge rules](references/integration.md).

## Linked Work and Project State

After merging, follow the linked-work rules in [integration and post-merge rules](references/integration.md).

## Handoff

Report the Pull Request URL and number. Report the repository, head and base branches, and resolved review mode. Report whether conflict resolution was performed. Report the pre-rewrite and pre-merge head SHAs and the resulting merge commit SHA. Report conflict files and resolution evidence. Report self-review or external approval evidence, required-check evidence, linked Issue outcomes, Project Status evidence, and any release publication or automation follow-up. State clearly whether the merge completed, was blocked, or has an ambiguous result. The next owner for a merged change is the repository's post-merge verification or release workflow. The next owner for a tracked release Pull Request is the release publication process. Do not apply substantive review fixes or publish a release in this skill.

## Writing quality

Use plain, active, evidence-backed prose in the final handoff. Keep headings in
sentence case and name the file, command, or output behind each repository claim.
