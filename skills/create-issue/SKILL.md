---
name: create-issue
description: Create GitHub change and release Issues that follow shared title, body, and changelog policy. Branch setup is the next phase. Use before starting governed repository work or preparing a release.
license: Apache-2.0
---

# Create Issue

## Todo List

1. **in progress:** Confirm repository, outcome, and whether the Issue is a change or release.
2. Draft the title and body from the matching repository template.
3. Choose the triage labels and create the Issue with them.
4. Complete the list only when the Issue URL is available; report the applied labels in the handoff.

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

## Triage Labels

- Create every Issue with exactly one `phase:`, one `scope:`, and one `priority:` label; all labels must come from the repository label set.
- Set `phase:backlog` on new Issues.
- Derive `scope:` from the Issue type: Feature→`scope:feature`, Bug→`scope:bug`, Documentation→`scope:docs`, Maintenance→`scope:maintenance`, Improvement→`scope:improvement`, Release→`scope:release`. Security uses `scope:bug` until a dedicated label exists.
- Choose one `priority:` label with the user; default to `priority:medium` when the user has no preference.
- Report the applied `priority:`, `scope:`, and `phase:` labels in the handoff with the Issue URL.

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
