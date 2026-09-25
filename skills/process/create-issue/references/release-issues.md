# Release Issue rules

- Use `[Release]: vX.Y.Z`. Follow the common headings with `Changelog`, then use `Added`, `Changed`, `Fixed`, and `Removed` in that exact order as level-three headings.
- Add one or more entries below every changelog heading; write `- None.` when a category is intentionally empty.
- Public releases use `vX.Y.Z`; build identifiers use `vX.Y.Z+N`.
- Link a release Pull Request merged before publication with `Tracks #<number>`. Close the Issue only after publication succeeds, through a release Pull Request merged after publication that uses `Closes #<number>`.
