# Contributing

## Commit messages

Use Conventional Commits for every commit and Pull Request title.

```text
type(scope): summary
```

Allowed `type` values are `build`, `chore`, `ci`, `docs`, `feat`, `fix`, `perf`, `refactor`, `revert`, and `test`. `scope` is optional, `summary` must not be empty, and the complete header must be no more than 100 characters.

```text
feat: add project bootstrap skill
fix(release): reject existing remote tags
docs: explain FSL verification boundary
ci: validate commit messages
```

Mark breaking changes with `!` or `BREAKING CHANGE:` in the body or footer. For initial setup, run `mise run setup` once to enable mise-managed Go commitlint and local hooks; the hooks stay active across branch switches without a manual re-run.

Every commit on an Issue branch must be a single sentence and end with the
Issue number in the header: `type(scope): summary #<number>`. Keep the number
as the governing Issue for that commit; one Pull Request may handle multiple
Issues, so the commit number need not match the branch name or the Pull
Request's first Issue. `cmd/validate-commit-message` enforces the
shape and commitlint validates the header. Run `mise run validate` or a local
commit to confirm the message before pushing.

One Pull Request is not required to be one commit. Split Pull Request commits
at implementation-layer boundaries or at the boundaries of the Issue's
implementation tasks. Do not create validation-only adjustment commits during
worktree development; resolve any validation failure in the intended commits
before the final commit is pushed, so the pushed history contains only the
implementation commits that belong in the Pull Request.

GitHub Actions validates commit messages and Pull Request titles. Locally, Git hooks run `mise run check:local` before commits and `mise run validate` before pushes. Fix a failed check before retrying.

## Issue and Pull Request titles

Use `[Type]: Summary` in sentence case for both Issues and Pull Requests. Type is `Feature`, `Bug`, `Improvement`, `Documentation`, `Security`, `Maintenance`, or `Release`. Begin Summary with a capitalized imperative verb and capitalize later words only when ordinary English requires it. Releases use the exception `[Release]: vX.Y.Z`; build identifiers use `[Release]: vX.Y.Z+N`. A Pull Request may close more than one Issue, so its title need not match an Issue title exactly.

Create Issues from the Change or Release template. Use `Context`, `Goal`, `Scope`, `Acceptance criteria`, and `Validation` exactly once in that order. Use one ordered `In`/`Out` pair in Scope and non-empty acceptance and validation checklists. Follow the common Release sections with `Changelog`, then `Added`, `Changed`, `Fixed`, and `Removed` exactly once in that order. Use `vX.Y.Z` for a public release and `vX.Y.Z+N` for an artifact build identifier.

## Issue planning

Every open Issue is tracked in the repository's GitHub Project declared by
`.github/issue-project.toml`. Each Issue has exactly one Project item carrying
values from the declared single-select fields:

- **Status** — `Backlog`, `Planned`, `In progress`, `In review`, `Done`.
- **Priority** — `High`, `Medium`, `Low`; confirm it with the Issue owner or
  use the declared default.
- **Scope** — `Feature`, `Bug`, `Docs`, `Maintenance`, `Improvement`,
  `Security`, `Release`; derive it from the Issue type.

The Project is the operational source of truth for planning; GitHub Issues
remain the source of truth for requirements, discussion, and closure. The
workflow skills advance Status automatically: `create-issue` adds a new Issue
with Status `Backlog` and the derived Scope and Priority, `plan-issue` moves a
planned Issue to `Planned`, `implement-issue` moves started work to
`In progress`, `create-pr` moves an Issue to `In review` when its Pull
Request opens, the Project's built-in Item closed workflow moves a closed
Issue to `Done`, and its built-in Item reopened workflow restores `Backlog`
when an Issue is reopened. Field and option IDs are resolved from the
declared names at runtime; never hard-code Project identities in skills.

Interactive `gh` authentication needs `read:project` to read Projects and
`project` to mutate them. Automation uses least-privilege PATs stored only as
repository secrets (`PROJECTS_READ_TOKEN` when the default token cannot read
Projects); never store credentials in tracked files. When Project access is
unavailable, the tooling fails safely with an actionable diagnostic and does
not mutate. The former `priority:`, `scope:`, and `phase:` triage labels are
retired; unrelated classification labels such as `dependencies`,
`github_actions`, `duplicate`, and accessibility or contributor labels
remain available and are not a second copy of Project fields.

## Branch and Pull Request flow

Create an Issue and an `issue/<number>` branch before each human change. Define allowed Pull Request directions as `[[routes]]` in `.github/branch-policy.toml`; the default is `issue/<number> -> main`. Add another route only for a project that needs a separate integration or stabilization branch. The documented automation exception is `dependabot/* -> main`, which does not require an Issue or closing reference.

Dependabot Pull Requests differ from the human flow in two ways: the `commitlint` and `work-item-title` checks exempt a pull request opened by the `dependabot[bot]` author, and commits authored by `dependabot[bot]` are skipped in push ranges, because Dependabot generates its own branch names, commit messages, and titles, which cannot carry the `#NNN` suffix or the `[Type]: Summary` shape. `.github/dependabot.yml` sets `commit-message.prefix: "ci"` so bot commits still carry a Conventional Commits type (`ci: Bump ...`) in the ecosystems where Dependabot supports commit message prefixes. Human branches keep every commit and title rule unchanged.

Rebase a work branch onto its latest upstream and push the rewritten branch with `--force-with-lease`, never plain `--force`. Do not push directly, force-push, or delete `main` or another protected branch. Use rebase merge only so the Pull Request title does not replace compliant commit messages. The `Require pull requests on protected branches` GitHub Ruleset enforces this boundary on `main` with active enforcement and no bypass actors, requiring rebase-only merging, linear history, no force-pushes or deletions, and the `Validate repository checks`, `Validate lint`, `Validate tests`, `Validate FSL specifications`, `Validate skills`, `Validate branch policy`, `Validate work item title`, `Validate commit conventions`, `Audit workflow security`, and `Validate commit signatures` status checks. The `validate` job of `validate.yml` was split into the first five parallel jobs by #155; the policy checks are grouped under `policy.yml` and the signature check under `policy-signatures.yml` by #160. Keep the required contexts in sync with the live ruleset.

Every Pull Request commit must have a GitHub-verified signature enforced by the `Validate commit signatures` required check. The `policy-signatures.yml` workflow produces that check on `pull_request_target`; GitHub's native `required_signatures` rule is not used because rebase merge creates unsigned replacement commits on `main`, and updating an Issue branch rechecks reachable base commits. Replacement commits created by GitHub rebase merge on `main` are outside the check.

Start every human Issue-backed Pull Request body with `## Issue` and no preceding prose. Put only a contiguous block of standalone references directly below it. Use `Closes #<number>` for change work and `Tracks #<number>` for a release Issue that must remain open until publication. For change work, the first reference must match the Issue number in the human work branch. Add one `Closes` line for every additional Issue handled by the same Pull Request, and keep every reference in the opening `Issue` section. Do not repeat Issue references at the end of the body.
