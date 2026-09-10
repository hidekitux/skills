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

## Body Structure

- Use each required heading exactly once and in the prescribed order. Do not insert other level-two or level-three headings.
- Fill every section with concrete content; remove template comments and do not leave empty checklist items.
- Write `Context` as the current state, the reason to act, and the problem to investigate; do not prescribe a solution before investigation.
- Write `Scope` with `- In:` followed by `- Out:`. Under `- In:`, state the part of the system and problem boundary covered, not work to perform; never scope an unmade decision as work. State explicit exclusions under `- Out:`.
- Write `Acceptance criteria` as observable checkboxes that define the outcome regardless of which defensible approach is chosen.
- Write `Validation` as checkboxes naming how the outcome will be observed, rather than prescribing implementation commands.
- Begin ordinary English sentences and list items with a capital letter, such as `Add`, `Formalize`, or `Register`.
- Preserve canonical lowercase or mixed-case names such as `iPhone`, `npm`, and `eBay`. Also preserve literal commands, paths, code, and identifiers instead of capitalizing them mechanically.
- Before creation, review the rendered title and body for heading order, duplicate sections, empty content, unresolved placeholders, and accidental lowercase prose.

## Project Triage

When the Issue must enter the repository Project, read [Project triage](references/project-triage.md) before creating it.

## Change Issues

When creating a Change Issue, read [Change Issue rules](references/change-issues.md).

## Release Issues

When creating a Release Issue, read [Release Issue rules](references/release-issues.md).

## Writing quality

Read [persistent prose](references/persistent-prose.md) before writing text
that outlives the conversation.
