---
name: create-issue
description: Create GitHub change and release Issues that follow shared title, body, branch, and changelog policy. Use before starting governed repository work or preparing a release.
license: Apache-2.0
---

# Create Issue

## Todo List

1. **in progress:** Confirm repository, outcome, and whether the Issue is a change or release.
2. Draft the title and body from the matching repository template.
3. Create the Issue; for change work create `issue/<number>` from the default branch.
4. Complete the list only when the Issue URL and branch name are available; handoff both.

Keep exactly one item in progress. Do not complete an item without its observable result.

## Change Issues

- Use `[Type]: Verb Summary`. Type is `Feature`, `Bug`, `Improvement`, `Documentation`, `Security`, or `Maintenance`; Summary begins with a capitalized imperative verb.
- Include `Context`, `Goal`, `Scope`, `Acceptance criteria`, and `Validation`.
- Open `issue/<number>` back to the default branch with `Closes #<number>`.
- To update the branch, rebase it onto the latest default branch and push with `--force-with-lease`. Never use plain `--force`.

## Release Issues

- Use `[Release]: vX.Y.Z` and include the common headings plus `Changelog` with `Added`, `Changed`, `Fixed`, and `Removed`.
- Public releases use `vX.Y.Z`; build identifiers use `vX.Y.Z+N`.
- Link a release PR with `Tracks #<number>` and close the Issue only after publication succeeds.
