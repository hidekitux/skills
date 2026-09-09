# Verified implementation plan for the legacy diagnostics change

## Capability landing

| Capability | Classification | Named substitute or evidence |
| --- | --- | --- |
| Branch ownership | Replaced by a named substitute | `wt list` |
| Setup-stamp comparison | Dropped | No equivalent replacement exists; `docs/worktrees.md` keeps manual `readlink` guidance. |
| Merged-state reporting | Replaced by a named substitute | `wt remove` |
| Remediation guidance | Retained | `docs/worktrees.md` |

## Ordered tasks

1. Remove `src/worktree-diagnostics.md`.
   - Evidence: the file is absent after the change.
2. Preserve `docs/worktrees.md` as the replacement guidance.
   - Evidence: the document still names the ownership, cleanup, and setup inspection procedures.
3. Reconcile the capability-landing inventory with the final change and run the required checks.
   - Evidence: the implementation handoff repeats all four rows and keeps the authorized setup-stamp drop visible.

## Out of scope

- Restoring setup-stamp comparison.
- Adding a repository check or release work.

## Residual risk

The dropped setup-stamp comparison may be missed if the handoff lists only files.

## Next-phase handoff

After implementation, hand the validated branch to the pull-request phase with
the complete capability-landing inventory.
