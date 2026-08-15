# Issue-based branch policy

- Name human work branches `issue/<positive-number>`; do not add type or summary.
- Require matching `Closes #<number>` in the Pull Request body.
- Protect `develop`, `main`, and release branches from direct updates,
  force-pushes, and deletion through GitHub Rulesets.
- Define allowed PR directions as `[[routes]]` in a project configuration file.
  A route has regular-expression head/base patterns and can require Issue linkage.
  Use `issue/* -> develop -> release/vX.Y.Z -> main` when releases need a
  stabilization branch, or `issue/* -> develop -> main` when they do not.
