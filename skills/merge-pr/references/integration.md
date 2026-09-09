# Integration and post-merge rules

Read this reference when a Pull Request conflicts or after the merge gate passes.

## Resolve conflicts

- Resolve conflicts only with explicit requester authorization. Fetch the exact remote head and base, record the pre-rewrite head, and use an isolated clean worktree.
- Rebase onto the latest allowed base and push only with `--force-with-lease`. Resolve each conflict from the Issue scope and review evidence; never use blanket `ours` or `theirs`.
- Inspect the complete new diff, run repository validation, and obtain a fresh review when the diff or commit graph changed. Restart the merge gate after the update.

## Merge and verify

- Use rebase merge only: `gh pr merge <number> --rebase` after all gates pass. Never use `--admin` or bypass required checks.
- After merging, verify `merged=true`, timestamp, merge commit, base branch, and expected head ancestry. Do not retry an ambiguous merge.
- Do not publish a release, create tags, edit release notes, or close a Release Issue here.

## Reconcile linked work

- For a change Pull Request using `Closes`, verify GitHub closed the Issue and the Project item reached `Done`; report automation delay instead of manually forcing it.
- For a release Pull Request using `Tracks`, verify the Release Issue remains open and its Project Status remains non-terminal until publication.
- Verify each linked Issue independently and report mismatches as follow-up work.
