# Worktree workflow

Codex and Claude Code run from separate Git worktrees. This is the operating
policy for creating, using, and removing them.

## Policy

- The primary worktree owns `main`. Git refuses to check out `main` in a second
  worktree; never use `--force` to work around that.
- Changes go on an `issue/<number>` branch created from an existing Issue.
- `mise` is the entry point for every repository command.
- No worktree is removed without inspecting it first. Never force-remove a
  worktree that holds uncommitted work, and never delete an unmerged branch to
  make a removal succeed.
- CI drives Git natively. No workflow or `mise` task depends on the worktree
  tool.

## Tooling

[`worktrunk`](https://github.com/max-sixty/worktrunk) (`wt`) is the worktree
tool. It is dual-licensed `MIT OR Apache-2.0`, and the repository relies on the
Apache-2.0 option. Install it once per machine; it is a local developer
convenience, not a repository or CI dependency, and it is not bundled into any
published skill.

```bash
mise use -g worktrunk
wt config shell install
```

`wt config shell install` enables directory switching. Without it `wt` still
creates and registers the worktree, but reports that it could not change
directory.

## Commands

```bash
wt switch --create issue/<number>   # create the Issue branch and its worktree
wt switch issue/<number>            # switch to an existing worktree
wt list                             # branch, path, and status per worktree
wt remove issue/<number>            # remove an inspected, inactive worktree
```

`--base <branch>` selects a base other than the default branch. The worktree
path is derived from the branch name, so `issue/232` becomes a
`repo.issue-232`-style directory.

`wt list` reports which worktree owns a branch. Do not run development commands
from a bare repository entry point; use a registered non-bare worktree.

`wt remove` fails when the worktree has uncommitted changes, and it removes the
branch only when the branch is merged. Do not reach for `wt remove --force`
(`-f`), which discards staged, modified, and untracked files, or
`wt remove -D` (`--force-delete`), which deletes an unmerged branch. Commit,
push, or stash the work instead. `--reap` is experimental; it is not part of
this workflow.

## Setup

The tracked `post-checkout` hook runs `mise run setup:all` whenever Git creates
or switches a branch, including `wt switch --create`, so a new worktree is
normally ready to use. Run it by hand from the worktree when the hook was
skipped or reported a failure:

```bash
mise run setup:all
```

Local skill registration is snapshot-dependent. Verify it with `readlink
.claude/skills/<skill-name>` or `readlink .agents/skills/<skill-name>`.

## Native Git

Native `git worktree` remains supported. Use a detached worktree for a
read-only `main` snapshot, and create an Issue branch rather than checking out
`main` again:

```bash
git worktree add --detach <path> origin/main
git worktree add -b issue/<number> <path> origin/main
```

The removal policy applies equally here: never `git worktree remove --force` a
worktree that holds work.
