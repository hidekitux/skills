---
name: create-issue
description: Create GitHub change and release Issues that follow shared title, body, and changelog policy. Branch setup is the next phase. Use before starting governed repository work or preparing a release.
license: Apache-2.0
---

# Create Issue

## Todo List

1. **in progress:** Confirm repository, outcome, and whether the Issue is a change or release.
2. Draft the title and body from the matching repository template.
3. Add the Issue to the declared GitHub Project with Status, Scope, and Priority, then create it.
4. Complete the list only when the Issue URL and its Project item are available; report the Project Status, Scope, and Priority in the handoff.

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

## Project Triage

- Read the repository-declared Project configuration at `.github/issue-project.toml` before creating the Issue. It names the Project and the Status, Priority, and Scope fields with their options and the default Priority. A missing or invalid configuration is a blocker; report it instead of guessing.
- After the Issue exists, add it to the declared Project exactly once. Confirm it has no existing item with `gh project item-list`, then add it by URL with `gh project item-add`; when an item already exists, reuse it and never create a duplicate.
- Set Status to `Backlog`. Derive Scope from the Issue type: Feature→`Feature`, Bug→`Bug`, Documentation→`Docs`, Maintenance→`Maintenance`, Improvement→`Improvement`, Security→`Security`, Release→`Release`. Set the user-selected Priority, or the declared default when the user has no preference.
- Resolve Project number, field IDs, and option IDs from the declared names at runtime with `gh project list` and `gh project field-list`; never hard-code this repository's Project identity or IDs.
- Apply the values with `gh project field-set`. Fail safely when Project access is unavailable or the configuration is ambiguous: do not mutate, and report the exact diagnostic.
- Report the Issue URL plus the resulting Project item and its Status, Priority, and Scope values in the handoff.

## Change Issues

- Use `[Type]: Summary` in sentence case. Type is `Feature`, `Bug`, `Improvement`, `Documentation`, `Security`, or `Maintenance`; Summary begins with a capitalized imperative verb. Capitalize later words only when ordinary English requires it, such as for proper nouns or abbreviations.
- Use `Context`, `Goal`, `Scope`, `Acceptance criteria`, and `Validation` in that exact order.
- Do not create the `issue/<number>` branch here. Branch creation and rebase
  belong to `implement-issue`, which is the next session for change work.

## Release Issues

- Use `[Release]: vX.Y.Z`. Follow the common headings with `Changelog`, then use `Added`, `Changed`, `Fixed`, and `Removed` in that exact order as level-three headings.
- Add one or more entries below every changelog heading; write `- None.` when a category is intentionally empty.
- Public releases use `vX.Y.Z`; build identifiers use `vX.Y.Z+N`.
- Link a release PR with `Tracks #<number>` and close the Issue only after publication succeeds.
