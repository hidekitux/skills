# Issue-based branch policy

- Name human work branches `issue/<positive-number>`; do not add type or summary.
- Make `## Issue` the first Pull Request body section with no preceding prose.
- Put only a contiguous block of standalone Issue references immediately below
  it. Require the first `Closes #<number>` reference to match the Issue number
  in the human work branch. Allow following `Closes` lines for additional
  Issues handled by the same Pull Request, but reject every reference outside
  the opening `Issue` section. Use `Tracks #<number>` only for release work
  that closes after publication.
- Protect `main` and any explicitly configured integration or release branches
  from direct updates, force-pushes, and deletion through GitHub Rulesets. Enable
  rebase merge only and automatic deletion of merged head branches in the repository
  settings.
- Require GitHub-verified signatures for every pull-request commit through the
  `Validate signed pull-request commits` required check. Run it on
  `pull_request_target`, use GitHub's pull-request commits API, and check out the
  base revision so a PR cannot weaken its own verifier. Do not target `main` or
  `issue/*` with `required_signatures`: GitHub rebase merge creates unsigned
  replacement commits, and updating an Issue branch rechecks reachable base
  commits. Do not add a non-fast-forward rule to `issue/*`; authors must be able
  to rebase and push with `--force-with-lease`.
- Define allowed PR directions as `[[routes]]` in a project configuration file.
  A route has regular-expression head/base patterns and can require Issue linkage.
  Use `issue/* -> main` by default. Add a project-specific route only when a
  separate integration or stabilization branch is actually needed.
