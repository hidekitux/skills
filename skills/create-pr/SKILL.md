---
name: create-pr
description: Create or update compliant GitHub Pull Requests from completed Issue-branch work. Use when asked to inspect changes, run repository validation, commit and push the intended files, and open a draft or ready Pull Request without duplicating an existing one.
license: Apache-2.0
---

# Create Pull Request

## Todo List

1. **in progress:** Resolve the repository, linked Issue, head and base branches, branch policy, and any existing Pull Request.
2. Review the worktree, commits, and base diff against the linked Issue scope.
3. Run the repository-required validation and record the evidence; decide whether the Pull Request can be ready or must be draft.
4. Commit and push only the intended changes, then create or update the Pull Request.
5. Complete the list only when the Pull Request URL, base and head branches, linked Issues, and validation evidence are available; hand off all of them.

Keep exactly one item in progress. Mark an item complete only after its stated evidence exists. Add or revise items when the agreed scope changes.

## Resolve and Guard

- Read repository instructions and Pull Request templates before drafting.
- Resolve the repository, current branch, target branch, linked Issue, and allowed route from local and hosting context. Do not guess a target branch.
- Require the repository's Issue-backed branch convention for human work. If the required Issue or branch is absent, stop and route to Issue creation rather than creating an unlinked Pull Request.
- Search for an open Pull Request with the same head and base. Update and hand off that Pull Request instead of creating a duplicate.
- Preserve unrelated user changes. Do not switch branches, stage files, or rewrite history when doing so would include or overwrite work outside the linked Issue.

## Review and Validate

- Inspect status, commits, and the complete base diff. Compare them with the Issue scope and acceptance criteria; report scope drift before publishing it.
- Check the diff for credentials, tokens, private URLs, generated noise, and accidental user data.
- Run the repository-prescribed checks. Prefer its standard task runner and required validation command; do not substitute narrower commands for an available aggregate task.
- Record every command and result. A Pull Request may be ready only when required validation succeeds and the agreed scope is complete.
- Use a draft when the user requests one or when validation is incomplete, unavailable, or failing. State every limitation in the Pull Request body; never describe an unrun check as passing.

## Commit and Push

- Stage only the intended files and follow the repository's commit-message policy. Keep unrelated changes unstaged.
- End every single-sentence commit message with ` #<number>` in the header, using the governing Issue for that commit; one Pull Request may handle multiple Issues, so the number need not match the branch name. Do not omit the suffix.
- Push the resolved head branch before creating the Pull Request.
- When an author-owned Issue branch must be rebased, resolve the exact remote branch and obtain any approval required by the host before rewriting it. Push only with `--force-with-lease`; never use plain `--force`.
- After a rebase changes the base revision, re-read the current repository
  instructions, Pull Request template, and branch-policy rules before drafting
  or updating the Pull Request body. Do not reuse a body layout inferred from
  the pre-rebase base revision.
- Do not push to a protected base branch or bypass repository protections.

## Create or Update

- Use the repository Pull Request template as the authority. When no template exists, include `Issue`, `Summary`, and `Validation` sections in that order.
- Use the repository title convention. For this repository, use `[Type]: Summary` in sentence case: begin the imperative verb with a capital letter and capitalize later words only when ordinary English requires it, such as for proper nouns or abbreviations. Use `[Release]: vX.Y.Z` only for releases.
- Make `## Issue` the first section with no prose before it. Put every Issue reference immediately below it as a contiguous block of standalone lines, before `Summary` or any other section; do not repeat references at the end.
- Link every change Issue with its own `Closes #123` line. Put the Issue encoded in the human work branch first, then add any other Issues the same Pull Request closes; keep every line in the opening `Issue` section. Use the repository's non-closing release keyword, such as `Tracks #123`, when publication must close the Release Issue later.
- Summarize observable behavior and scope, not the editing process. Include exact validation commands and outcomes.
- Begin ordinary English sentences and list items, including Summary bullets, with a capital letter, such as `Add`, `Formalize`, or `Register`.
- Preserve canonical lowercase or mixed-case names such as `iPhone`, `npm`, and `eBay`. Also preserve literal commands, paths, code, and identifiers instead of capitalizing them mechanically.
- Include repository-specific conditional checklists only when they apply; mark an allowed item not applicable instead of silently deleting required evidence.
- Review the rendered body and confirm the Issue-reference block is first, complete, and uses the correct closing behavior.
- Before creating or updating an Issue-backed Pull Request, validate the exact
  finalized body against any repository-provided Pull Request-body or
  branch-policy validator, using the resolved base and head branches. For this
  repository, run `python3 scripts/validate/validate-branch-policy.py --base <base>
  --head <head> --body "$final_body"` before the GitHub API call.
- If that preflight fails, do not publish a ready Pull Request or diagnose the
  body from the error message alone. Read the validator and its tests, revise
  the finalized body, and rerun the preflight; keep or create a draft only when
  the user explicitly asks to publish despite the unresolved failure.
- Create a ready Pull Request only after the ready gate passes. Otherwise create or retain a draft.

## Handoff

Report the Pull Request URL, ready or draft state, base and head branches, linked Issues, validation commands and results, and any remaining risk or follow-up. Do not merge, release, or begin review remediation unless the user separately requests it.
