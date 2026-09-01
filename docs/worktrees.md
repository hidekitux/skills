# Worktree workflow

This repository runs Codex and Claude Code from separate Git worktrees. This
document records which worktree tool the repository uses, why it was selected,
and the commands that are supported for Issue-branch work.

## Decision

The repository uses [`worktrunk`](https://github.com/max-sixty/worktrunk) (`wt`)
as the local worktree workflow tool. It replaces the retired `diagnose:worktree`
task, which reported branch ownership and setup state from a repository-local Go
command.

`worktrunk` is a local developer convenience. It is **not** pinned in
`mise.toml`, not required by CI, and not bundled into any published skill.
`mise` remains the entry point for every repository command, and native
`git worktree` remains a supported fallback.

### Reviewed version and license

| Property | Value |
| --- | --- |
| Upstream | https://github.com/max-sixty/worktrunk |
| Reviewed version | v0.75.0 (released 2026-08-27) |
| Minimum version | v0.75.0 |
| License | Dual `MIT OR Apache-2.0`; the repository relies on the Apache-2.0 option |
| Implementation | Rust, distributed as prebuilt platform binaries |
| Maintenance | Active; releases roughly every one to two weeks through 2026 |

`worktrunk` is not added to `mise.toml` `[tools]`, so it has no
`TOOL_LICENSES.toml` attestation. That registry mirrors mise-managed tools
exactly, and an attestation without a matching mise tool fails
`check:tool-licenses`. Record any future decision to pin `worktrunk` in both
files together.

### Installation

Install `worktrunk` once per machine. Any of the upstream channels is
acceptable; the reviewed installation used `mise`, which resolves the tool
through `aqua` and verifies the download checksum.

```bash
mise use -g "aqua:max-sixty/worktrunk@0.75.0"
wt config shell install
```

Homebrew (`brew install worktrunk`) and Cargo (`cargo install worktrunk`) are
also supported upstream. `wt config shell install` is required for `wt switch`
to change the shell's directory; without it `wt` still creates and registers the
worktree and reports that it could not change directory.

### Rejected alternative

`coderabbitai/git-worktree-runner` (`gtr`, Apache-2.0, Bash) was rejected. It
meets the licensing and macOS requirements, but its cleanup model conflicts with
this repository's rule that no worktree is removed without inspection: `gtr
clean --merged` performs bulk removal and can ignore local changes. `worktrunk`
refuses removal when a worktree is dirty and keeps an unmerged branch unless
deletion is requested explicitly, which matches the rule the repository already
enforced by hand.

### Requirement comparison

| Requirement | `worktrunk` | `git-worktree-runner` |
| --- | --- | --- |
| Issue-branch isolation | `wt switch --create issue/<number>` derives the path from the branch name | `git gtr new <branch>` |
| Creation and discovery | `wt switch`, `wt list`, `wt list --format json` | `git gtr new`, `git gtr list` |
| Safe cleanup | Refuses a dirty worktree; retains an unmerged branch | `git gtr rm`/`git gtr clean --merged` support bulk and forced removal |
| Setup hooks | `wt hook` lifecycle hooks; not needed here because `post-checkout` already runs setup | Configuration copying and dependency install hooks |
| macOS support | Prebuilt `aarch64-apple-darwin` binary, verified | Bash 3.2+, supported |
| Non-interactive use | Verified: every command below ran without a TTY; `-y` skips approval prompts | Supported |
| Installation and pinning | mise/aqua, Homebrew, Cargo; version pinnable | Homebrew tap or `install.sh` |
| Maintenance | Active, frequent releases | Active |
| Licensing | `MIT OR Apache-2.0` | Apache-2.0 |
| Failure diagnostics | Names the blocking condition and the files involved | Not verified |

## Workflow

The primary worktree owns `main`. Git refuses to check out `main` in a second
worktree, and `worktrunk` does not bypass that; never use `--force` to work
around it. Create a branch from an existing Issue instead of checking out `main`
again.

### Create or locate an Issue worktree

```bash
wt switch --create issue/<number>   # create the branch and its worktree
wt switch issue/<number>            # switch to an existing worktree
```

`--base <branch>` selects a base other than the default branch. The worktree
path is derived from the branch name, so `issue/232` becomes a
`repo.issue-232`-style directory.

### Run repository setup

The tracked `post-checkout` hook runs `mise run setup:all` whenever Git creates
or switches a branch, including `git worktree add` and `wt switch --create`, so
a new worktree is normally set up already. Run it by hand from the worktree if
the hook was skipped or reported a failure:

```bash
mise run setup:all
```

No `worktrunk` hook is configured for setup, because that would duplicate the
`post-checkout` hook, which works for every worktree regardless of how it was
created.

### Inspect worktree state

```bash
wt list                  # branch, path, and status for every worktree
wt list --format json    # same data for scripts
```

`wt list` reports which worktree owns a branch, replacing the ownership half of
the retired diagnostic. Local skill registration is snapshot-dependent and is
refreshed by `post-checkout`; verify it directly with `readlink
.claude/skills/<skill-name>` or `readlink .agents/skills/<skill-name>`.

### Remove an inactive worktree

Removal is an explicit, user-reviewed operation. Inspect the worktree with
`git status` and confirm the work is pushed before removing anything.

```bash
wt remove issue/<number>
```

`wt remove` fails when the worktree has uncommitted changes and lists the
offending files. It removes the branch only when the branch is merged; an
unmerged branch is kept and reported.

Do not use `wt remove --force` (removes a dirty worktree, discarding staged,
modified, and untracked files) or `wt remove -D` (deletes an unmerged branch).
Commit, push, or stash the work instead. `--reap` is experimental and kills
processes running under the worktree; do not use it as part of the standard
workflow.

### Read-only snapshot

Use a detached worktree for a read-only `main` snapshot:

```bash
git worktree add --detach <path> origin/main
```

## Verified behavior

Exercised against `worktrunk` v0.75.0 on macOS (Apple Silicon) in a disposable
repository, non-interactively:

| Scenario | Result |
| --- | --- |
| `wt switch --create issue/232` | Created branch and worktree; reported the missing shell integration without failing |
| `wt list` | Listed the primary and Issue worktrees with branch, path, and status |
| `wt remove issue/232` with uncommitted changes | Refused, exit status 1, listed the modified and untracked files |
| `wt remove issue/232` after cleaning | Removed the worktree and the merged branch |
| `wt remove issue/999` with unmerged commits | Removed the worktree, kept the branch, reported that `-D` would delete it |
| `git worktree add <path> main` while `main` is checked out | Failed, as the repository requires |
| `git worktree add --detach <path> main` | Created a read-only snapshot |

## Limitations and fallback

- `worktrunk` is a local tool. CI uses native Git only; do not make any workflow
  or `mise` task depend on `wt`.
- Upstream is pre-1.0 and releases often. Its CLI, installation channels, or
  platform support can change; review the reviewed version above before
  upgrading, and update this document with the change.
- No tool can infer whether uncommitted or unpushed work matters. Inspect state
  and decide before removing a worktree; automatic removal stays prohibited.
- Everything here has a native equivalent — `git worktree add`,
  `git worktree list`, and `git worktree remove` — which remains supported when
  `worktrunk` is unavailable or misbehaves.

## Migration note

The `diagnose:worktree` task and its `cmd/diagnose-worktree` command were
removed in favor of this workflow. `wt list` covers branch ownership, and the
`post-checkout` hook plus `readlink` verification covers setup state. The task
name is retired and is rejected by `check:tasks`.
