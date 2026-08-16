---
name: create-issue
description: Create GitHub change and release Issues that follow shared title, body, branch, and changelog policy. Use before starting governed repository work or preparing a release.
license: Apache-2.0
---

# Create Issue

## Todo List

1. **in progress:** Confirm repository, outcome, and whether the Issue is a change or release.
2. Draft the title and body from the matching repository template.
3. Create the Issue; for change work fetch the upstream default branch and create
   `issue/<number>` from the fetched upstream base.
4. Complete the list only when the Issue URL is available and, for change work,
   the branch name and its upstream base commit are available; include every
   applicable result in the handoff.

Keep exactly one item in progress. Do not complete an item without its observable result.

## Draft Prose

- Begin ordinary English sentences and list items with a capital letter, such as `Add`, `Formalize`, or `Register`.
- Preserve canonical lowercase or mixed-case names such as `iPhone`, `npm`, and `eBay`. Also preserve literal commands, paths, code, and identifiers instead of capitalizing them mechanically.
- Review the rendered title and body for accidental lowercase prose before creating the Issue.

## Body Structure

- Use each required heading exactly once and in the prescribed order. Do not insert other level-two or level-three headings.
- Fill every section with concrete content; remove template comments and do not leave empty checklist items.
- Write `Context` as the current state and reason for change, and `Goal` as one observable desired outcome.
- Write `Scope` with `- In:` followed by `- Out:`. State included work and explicit boundaries under the matching marker.
- Write `Acceptance criteria` as observable checkboxes that define completion.
- Write `Validation` as checkboxes naming the commands or evidence that will prove the criteria.
- Before creation, review the rendered body for heading order, duplicate sections, empty content, and unresolved placeholders.

## Change Issues

- Use `[Type]: Summary` in sentence case. Type is `Feature`, `Bug`, `Improvement`, `Documentation`, `Security`, or `Maintenance`; Summary begins with a capitalized imperative verb. Capitalize later words only when ordinary English requires it, such as for proper nouns or abbreviations.
- Use `Context`, `Goal`, `Scope`, `Acceptance criteria`, and `Validation` in that exact order.
- Fetch the upstream default branch before creating the issue branch so the
  branch starts from the latest base. Record the upstream base commit and open
  `issue/<number>` from that base with a branch that will close `#<number>`.
- To update the branch, rebase it onto the latest default branch and push with `--force-with-lease`. Never use plain `--force`.

## Release Issues

- Use `[Release]: vX.Y.Z`. Follow the common headings with `Changelog`, then use `Added`, `Changed`, `Fixed`, and `Removed` in that exact order as level-three headings.
- Add one or more entries below every changelog heading; write `- None.` when a category is intentionally empty.
- Public releases use `vX.Y.Z`; build identifiers use `vX.Y.Z+N`.
- Link a release PR with `Tracks #<number>` and close the Issue only after publication succeeds.
