---
name: create-issue
description: Create GitHub change and release Issues that follow shared title, body, and changelog policy; plan-issue plans a created Change Issue next. Use before starting governed repository work or preparing a release.
license: Apache-2.0
---

# Create Issue

## Todo List

1. **in progress:** Confirm repository, outcome, and whether the Issue is a change or release.
2. Draft the title and body from the matching repository template.
3. After user confirmation, create the Issue; when the repository declares a GitHub Project, add the Issue to it with Status, Scope, and Priority.
4. Complete the list only when the Issue URL is available, and its Project item too when the repository declares a Project; report the Project Status, Scope, and Priority in the handoff, or that the repository declares no Project.

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

## User Confirmation

- Create the Issue only with user confirmation. User confirmation is an explicit instruction from the user that covers creating this Issue, given in the request or in reply to the finalized title and body.
- When no instruction covers the creation, such as when the user approved only an investigation, present the finalized title and body and wait for an explicit answer.
- Treat no answer or an ambiguous answer as no confirmation. Do not create the Issue.

## Project Triage

When the repository declares a Project in `.github/issue-project.toml`, read [Project triage](references/project-triage.md) before creating the Issue. When that file does not exist, the repository uses no Project: create the Issue without Project triage and report that no Project is declared. A missing file is not a reason to stop or to ask again.

## Change Issues

When creating a Change Issue, read [Change Issue rules](references/change-issues.md).

## Release Issues

When creating a Release Issue, read [Release Issue rules](references/release-issues.md).

## Handoff

- Report the Issue URL, the Project Status, Scope, and Priority or that no Project is declared, and the next owner. For a Change Issue, the next owner is `plan-issue`, which plans it before `implement-issue` creates the branch.
- Create only the Issue. When the request also asks to plan, create a branch, or implement, state in the words "out of scope" that this work is out of scope for `create-issue`, do not do it, and leave it to `plan-issue` and `implement-issue`.
- End the turn with this handoff after the Issue exists. Do not start `plan-issue` or another skill in the same turn, even when the request asks to continue; the next phase starts on a new request.

## Writing quality

Read [persistent prose](references/persistent-prose.md) before writing text
that outlives the conversation.
