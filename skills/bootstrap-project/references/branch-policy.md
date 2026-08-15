# Issue-based branch policy

- Name human work branches `issue/<positive-number>`; do not add type or summary.
- Require matching `Closes #<number>` in the Pull Request body.
- Protect `main` and any explicitly configured integration or release branches
  from direct updates, force-pushes, and deletion through GitHub Rulesets. Enable
  rebase merge only and automatic deletion of merged head branches in the repository
  settings.
- Require a Pull Request status check that verifies every source commit is signed
  and GitHub-verified. Do not use GitHub's `required_signatures` rule with GitHub
  rebase merge because GitHub creates unsigned replacement commits during that merge.
- Define allowed PR directions as `[[routes]]` in a project configuration file.
  A route has regular-expression head/base patterns and can require Issue linkage.
  Use `issue/* -> main` by default. Add a project-specific route only when a
  separate integration or stabilization branch is actually needed.
