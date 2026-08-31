# Skill creation brief

When requesting a new or substantially updated skill, complete this brief and pass it to the available authoring workflow. Use `skill-creator` in Codex when available. Name the skill's layer, boundaries, related skills, and handoff target per the [skill-set map in README](../README.md#skill-set-map) and [docs/skill-layers.md](../docs/skill-layers.md). Leave unresolved decisions as questions; do not fill them by assumption.

```markdown
Create or update a publishable skill in skills/. When the host provides
skill-creator, use it; otherwise follow this brief and the repository
validation workflow.

## Name
- Proposed skill name:

## Goal and triggers
- What result must this skill produce?
- Example user requests that should trigger it:
- Requests it must not handle:

## Contract
- Required inputs and available tools/data:
- Expected output and completion criteria:
- Safety, approval, or privacy constraints:
- License: Apache-2.0 (project standard)
- Copyright attribution: use the confirmed repository owner in `NOTICE`; update covered years only when copyrightable material changes.

## Layer, boundaries, and handoff
- Layer (process, analyze, fix, or govern) per the skill-set map in README:
- Related skills in the same or adjacent layers, and how they share or hand off state:
- Intentional boundaries — work this skill must not do, and the adjacent skill that owns that work:
- Handoff target — next-owner skill that receives this skill's output, and what the handoff must contain:

## Todo List
- Initial Todo List items (include discovery, implementation, validation, and handoff when applicable):
- Evidence required to complete each item:
- Host-native Todo List capability and Markdown-checklist fallback, if any:

## Development environment
- Required tools and exact versions to pin in `mise.toml`:
- Applicable mise tasks (`format:all`, `lint:all`, `test:all`, `check:all`, `verify:fsl`):

## Reusable resources
- Deterministic scripts needed:
- References needed:
- Assets/templates needed:

## Validation
- Representative success case:
- Failure or edge case:
- Commands or observable evidence required before handoff:

## Host differences
- Codex-specific capability or fallback, if any:
- Claude Code-specific capability or fallback, if any:

## FSL
- Is there a stateful workflow, invariant, approval, retry, or publication rule worth formalizing? If yes, name the source rules and open decisions.
```

Keep core behavior portable, create host notes only for material execution differences, expose project workflows through mise, and validate the result before handoff.
