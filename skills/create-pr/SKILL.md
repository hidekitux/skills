---
name: create-pr
description: Create or update compliant GitHub Pull Requests from completed Issue-branch work. Use when asked to inspect changes, run repository validation, push the head branch, and open a draft or ready Pull Request without duplicating an existing one.
license: Apache-2.0
---

# Create Pull Request

## Todo List

1. **in progress:** Resolve the repository, linked Issue, head and base branches, branch policy, and any existing Pull Request.
2. Review the worktree, commits, and base diff against the linked Issue scope.
3. Run the repository-required validation and record the evidence; decide whether the Pull Request can be ready or must be draft.
4. Push the resolved head branch, then create or update the Pull Request.
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
- Confirm the branch commits follow the repository commit-message policy: a
  single-sentence header `type: summary #<number>` with the governing Issue
  number. Commits are authored by `implement-issue` during implementation.
- Check the diff for credentials, tokens, private URLs, generated noise, and accidental user data.
- Run the repository-prescribed checks. Prefer its standard task runner and required validation command; do not substitute narrower commands for an available aggregate task.
- Record every command and result, including the body and title preflight commands and their outcomes. A Pull Request may be ready only when required validation succeeds and the agreed scope is complete.
- Use a draft when the user requests one or when validation is incomplete, unavailable, or failing. State every limitation in the Pull Request body; never describe an unrun check as passing.

## Push

- Commits are authored by `implement-issue` during implementation. Do not
  stage, create, or amend commits here; push the resolved head branch as it is.
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
  repository, run `go run ./cmd/validate-branch-policy --base <base>
  --head <head> --body "$final_body"` before the GitHub API call.
- Validate the exact finalized title against the repository title convention
  before the GitHub API call. For this repository, run
  `go run ./cmd/validate-work-item-title --title "$final_title"`; do not open a
  ready Pull Request whose title fails preflight.
- If a preflight fails, do not publish a ready Pull Request or diagnose the
  body or title from the error message alone. Read the validator and its
  tests, revise the finalized body or title, and rerun the preflight; keep or
  create a draft only when the user explicitly asks to publish despite the
  unresolved failure.
- When the Pull Request opens, set the governing Issue's Project Status to `In review` using the repository-declared Project configuration and keep it there for release Issues until publication; resolve the Status field and option IDs from their declared names (`gh project field-list`) and apply them with `gh project item-edit` using the declared field and option names (one call per field). The update is idempotent with `implement-issue`. The `Policy (Project)` workflow keeps later PR events in sync (draft and ready transitions, synchronize, and unmerged close), the Project's built-in Item closed workflow moves a closed Issue to `Done`, and its built-in Item reopened workflow restores `Backlog` on reopen.
- Create a ready Pull Request only after the ready gate passes. Otherwise create or retain a draft.

## Handoff

Report the Pull Request URL, ready or draft state, base and head branches, linked Issues, the body and title validation commands and their results, the governing Issue's Project Status, and any remaining risk or follow-up. Do not merge, release, or begin review remediation unless the user separately requests it.

Opening and maintaining the Pull Request happens in this session. Merge and
release are later phases, not part of this skill.
