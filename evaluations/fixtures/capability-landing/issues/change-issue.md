# Change issue: retire legacy worktree diagnostics

## Context

The project is replacing the legacy worktree diagnostics unit with a smaller
workflow. The removed unit reports branch ownership, setup-stamp comparison,
merged-state reporting, and remediation guidance. The replacement names `wt
list` for ownership, `wt remove` for merged-state cleanup, and the worktree
document for remediation. No replacement covers setup-stamp comparison.

## Goal

Retire the legacy unit while making every capability outcome visible.

## Scope

- In: remove `src/worktree-diagnostics.md`, keep the replacement guidance in `docs/worktrees.md`, and record the landing of every removed capability.
- Out: restore setup-stamp comparison, add a repository check, and release work.

## Acceptance criteria

- [ ] The legacy unit is removed.
- [ ] The final evidence classifies every capability as retained, replaced by a named substitute, or dropped.
- [ ] The authorized setup-stamp drop remains visible in the handoff.

## Validation

- [ ] Confirm the removed file is absent and the capability-landing inventory is complete.
