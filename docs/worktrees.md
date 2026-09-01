# Worktree workflow

This repository runs Codex and Claude Code from separate Git worktrees. This
document records which worktree tool the repository uses, why it was selected,
and the commands that are supported for Issue-branch work.

## Decision

The repository uses [`worktrunk`](https://github.com/max-sixty/worktrunk) (`wt`)
as the local worktree workflow tool. It replaces the retired `diagnose:worktree`
task, which reported branch ownership and setup state from a repository-local Go
command.

`worktrunk` is pinned in `mise.toml`, so `mise install` provides the reviewed
version wherever the toolchain is set up, including CI runners. It is a
developer workflow tool, not a validation dependency: no `mise` task and no CI
job invokes `wt`, and it is not bundled into any published skill. `mise` remains
the entry point for every repository command, and native `git worktree` remains
a supported fallback.

### Reviewed version and license

| Property | Value |
| --- | --- |
| Upstream | https://github.com/max-sixty/worktrunk |
| Reviewed version | v0.75.0 (released 2026-08-27) |
| Pin | `"aqua:max-sixty/worktrunk" = "0.75.0"` in `mise.toml` |
| License | Dual `MIT OR Apache-2.0`; the repository relies on the Apache-2.0 option |
| Implementation | Rust, distributed as prebuilt platform binaries |
| Platforms | Prebuilt `apple-darwin` and `unknown-linux-musl` binaries for both x86_64 and aarch64, so the pin resolves on developer machines and the Ubuntu CI runners |
| Maintenance | Active; releases roughly every one to two weeks through 2026 |

`worktrunk` is declared in `mise.toml` `[tools]` and attested in
`TOOL_LICENSES.toml`. That registry mirrors mise-managed tools exactly — a tool
without an attestation and an attestation without a tool both fail
`check:tool-licenses` — so change the pinned version in both files together.

### Installation

`mise install` provides `worktrunk` from the repository pin, resolving it
through `aqua` and verifying the download checksum. Enable the shell
integration once per machine afterwards:

```bash
mise install
wt config shell install
```

`wt config shell install` is required for `wt switch` to change the shell's
directory; without it `wt` still creates and registers the worktree and reports
that it could not change directory. Homebrew (`brew install worktrunk`) and
Cargo (`cargo install worktrunk`) remain supported upstream, but prefer the
repository pin so every worktree uses the reviewed version.

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
| Installation and pinning | Pinned in `mise.toml` through aqua; Homebrew and Cargo also available upstream | Homebrew tap or `install.sh`, no mise registry entry |
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

- `worktrunk` is a developer workflow tool. CI installs it with the rest of the
  mise toolchain but drives Git natively; do not make any workflow or `mise`
  task depend on `wt`.
- Upstream is pre-1.0 and releases often. Its CLI, installation channels, or
  platform support can change; the pin keeps every environment on the reviewed
  version, so bump `mise.toml`, `TOOL_LICENSES.toml`, and this document
  together after re-reviewing the release.
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
