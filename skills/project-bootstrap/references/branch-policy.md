# Issue-based branch policy

- Name human work branches `issue/<positive-number>`; do not add type or summary.
- Require matching `Closes #<number>` in the Pull Request body.
- Protect `main` and any explicitly configured integration or release branches
  from direct updates, force-pushes, and deletion through GitHub Rulesets.
- Define allowed PR directions as `[[routes]]` in a project configuration file.
  A route has regular-expression head/base patterns and can require Issue linkage.
  Use `issue/* -> main` by default. Add a project-specific route only when a
  separate integration or stabilization branch is actually needed.
