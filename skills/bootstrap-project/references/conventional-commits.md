# Conventional Commits

Use Conventional Commits as the default contribution contract for bootstrapped
projects. The contract is `type(scope): summary`, with an optional scope and a
non-empty summary. Allow the standard types `build`, `chore`, `ci`, `docs`,
`feat`, `fix`, `perf`, `refactor`, `revert`, and `test`; document any project-
specific additions explicitly.

## Enforcement

- Add the policy to the existing contributor documentation, or create a small
  `CONTRIBUTING.md` when no contributor guide exists.
- If the project has GitHub Actions, add a required CI check for commit messages
  and Pull Request titles. Pin third-party Actions to immutable commit SHAs.
- For the CI implementation, prefer `actions/setup-go` plus a pinned
  `github.com/conventionalcommit/commitlint` version over a Node-only action.
- Prefer the Go implementation `github.com/conventionalcommit/commitlint` for
  local hooks when the project does not already have a Node toolchain. Pin the
  version, install it through the project's tool manager when possible, and
  keep the hook in the repository. If the project already has a Node toolchain,
  the JavaScript commitlint implementation is also acceptable.
- If no CI provider is available, document that Conventional Commits are the
  project policy and report that automated enforcement is not yet available.
- Configure every protected branch through a GitHub Ruleset to require the
  commit and Pull Request title checks before merge. Use
  `scripts/configure-github-ruleset.py` after the checks have run once so their
  completed job names are known. Do not claim enforcement is complete until the
  script has read the Ruleset back successfully.

## Verification

Check at least one valid and one invalid message against the selected linter,
then run the project's normal check command. Report whether the evidence is a
local hook, CI result, branch protection setting, or documentation only.
