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

Mark breaking changes with `!` or `BREAKING CHANGE:` in the body or footer. For initial setup, run `mise run setup` to enable mise-managed Go commitlint and local hooks.

GitHub Actions validates commit messages and Pull Request titles. Locally, Git hooks run `mise run check:local` before commits and `mise run validate` before pushes. Fix a failed check before retrying.

## Issue and Pull Request titles

Use `[Type]: Summary` in sentence case for both Issues and Pull Requests. Type is `Feature`, `Bug`, `Improvement`, `Documentation`, `Security`, `Maintenance`, or `Release`. Begin Summary with a capitalized imperative verb and capitalize later words only when ordinary English requires it. Releases use the exception `[Release]: vX.Y.Z`. A Pull Request may close more than one Issue, so its title need not match an Issue title exactly.

Create Issues from the Change or Release template. Use `Context`, `Goal`, `Scope`, `Acceptance criteria`, and `Validation` exactly once in that order. Use one ordered `In`/`Out` pair in Scope and non-empty acceptance and validation checklists. Follow the common Release sections with `Changelog`, then `Added`, `Changed`, `Fixed`, and `Removed` exactly once in that order. Use `vX.Y.Z` for a public release and `vX.Y.Z+N` for an artifact build identifier.

## Branch and Pull Request flow

Create an Issue and an `issue/<number>` branch before each human change. Define allowed Pull Request directions as `[[routes]]` in `.github/branch-policy.toml`; the default is `issue/<number> -> main`. Add another route only for a project that needs a separate integration or stabilization branch. The documented automation exception is `dependabot/* -> main`, which does not require an Issue or closing reference.

Rebase a work branch onto its latest upstream and push the rewritten branch with `--force-with-lease`, never plain `--force`. Do not push directly, force-push, or delete `main` or another protected branch. Use rebase merge only so the Pull Request title does not replace compliant commit messages. Every Pull Request commit must have a GitHub-verified signature enforced by the `Validate signed pull-request commits` required check. Replacement commits created by GitHub rebase merge on `main` are outside that check.

Start every human Issue-backed Pull Request body with `## Issue` and no preceding prose. Put only a contiguous block of standalone references directly below it. Use `Closes #<number>` for change work and `Tracks #<number>` for a release Issue that must remain open until publication. For change work, the first reference must match the Issue number in the human work branch. Add one `Closes` line for every additional Issue handled by the same Pull Request, and keep every reference in the opening `Issue` section. Do not repeat Issue references at the end of the body.
