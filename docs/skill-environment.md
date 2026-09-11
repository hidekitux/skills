# Skill execution environment

Issue #199 defines the environment boundary that sits between skill selection and skill execution. The environment is a local workspace plus a command policy, setup result, branch-ownership result, and cleanup disposition. It describes authority; it does not grant authority.

## Authority projection

The provisioner reads the selected skill from `workflow/skill-graph.yml` and projects its declared authority into one profile:

- `read_only` uses a detached Git snapshot. It permits observation and denies repository writes, Git writes, GitHub writes, and external mutation.
- `repository_write` uses an existing `issue/<number>` worktree. It permits repository and Git writes but no external mutation.
- `external_mutation` records the graph's declared external-mutation class. It follows the repository and Git workspace rule, but it does not provide credentials or skip the owning skill's approval boundary.

The graph remains the only authority source. A profile cannot add permission that the graph does not declare.

## Provisioning gates

Provisioning resolves the requested repository revision before execution. A read-only profile creates a detached snapshot and applies read-only filesystem permissions without changing the shared Git directory. A profile that can write the repository or Git state requires an existing governed Issue, the matching `issue/<number>` branch, and a registered worktree owned by that branch. The provisioner stops on a detached, wrong, missing, or concurrently owned branch.

The provisioner runs `mise run setup:all` before execution. A missing command or non-zero result stops the selected skill. `worktrunk` remains a local operator convenience. Use `wt list` to inspect ownership when available; native `git worktree list --porcelain` is the machine-readable source.

## Command and host boundaries

The generated command policy denies repository-writing Git operations and remote-writing GitHub operations in a read-only environment. It rejects unknown mutation commands instead of guessing. The policy is defense in depth. It does not replace the active host's sandbox or approval prompts, and it never passes a bypass-approval option or creates external work items.

Codex maps `read_only` to its read-only workspace and approval settings. When the setting is unavailable, the adapter keeps the command policy and stops for user direction rather than enabling a bypass. Claude Code maps `read_only` to its planning or read-only permission mode and maps repository writes to its edit-capable mode only inside the verified Issue worktree. When a mode is unavailable, it uses the same fail-closed fallback. Both adapters record the same semantic environment fields and omit provider-specific paths, arguments, output, prompts, reasoning, and unknown fields.

## Manifest and trace fields

`workflow/skill-environment.schema.json` defines the privacy-safe manifest. It records the environment identifier, skill and graph identity, profile, workspace kind, repository revision, Issue branch identity when present, declared permissions, setup status, ownership status, and cleanup disposition. It never records absolute paths, credentials, command output, prompts, reasoning, source contents, or user data.

The manifest is an optional `environment` object in trace schema version 2. Trace validation keeps the existing allowlist and redaction rules. The manifest records what provisioning observed; it does not prove that a host enforced the boundary or that a remote branch remained unchanged after provisioning.

## Cleanup

Cleanup inspects active worktree ownership, uncommitted changes, and upstream divergence before removal. It retains active or material worktrees and refuses force removal. A clean inactive environment is removed only after explicit review confirmation. Every path records one of `not_requested`, `retained`, `removed_after_review`, `blocked_material`, or `blocked_active`.
