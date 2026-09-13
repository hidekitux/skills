# Codex guidance

Use this directory for repository-level Codex setup examples only. Publishable
skills remain in `skills/`; Codex-facing UI metadata belongs in each relevant
skill's `agents/openai.yaml`.

Do not assume a particular plugin, MCP server, browser session, or local binary
exists. Record any material capability difference in the relevant skill's
`references/hosts/codex.md`, including a safe fallback and verification step.

## Trace adapter

The Codex adapter accepts the provider event shape in `trace-fixture.json` and
maps only its safe semantic fields into `workflow/skill-trace.schema.json`.
Arguments, output, prompts, reasoning, and unknown provider fields are not
persisted. Verify the mapping with `go test ./internal/trace`.

## Execution environment

Codex maps the `read_only` profile to its read-only workspace and approval
settings. It runs `repository_write` only in the verified `issue/<number>`
worktree. The `external_mutation` profile records the graph declaration but
does not grant credentials, bypass approval, or remote-write permission.

When the host setting is unavailable, keep the generated command policy,
stop before execution, and ask for host direction. Verify the boundary with
`git worktree list --porcelain`, the manifest's `ownership.status`, and the
read-only mutation fixture in `go test ./internal/environment`. Never pass a
bypass-approval option to make a denied operation run.

## Model selection

Skills select role-specific models from the shared convention in
[docs/model-selection.md](../../docs/model-selection.md). The per-project
variables live in `opencode.json` under `agent.high.model`, `agent.mid.model`,
and `agent.low.model`; see
[docs/model-routing.md](../../docs/model-routing.md) for the OpenCode provider
setup. In Codex the selectors are served through
codex-router (for example `opencode-go/deepseek-v4-flash`): the default agent
model is set once in the host config, and subagents inherit it or take an
explicit selector when the host allows a model override. When Codex cannot
read the variable, use the documented default for the tier and the fallback
rule, and name the fallback in the handoff.

## Local skill registration

Run `mise run setup:all` once in each Git worktree. The command bootstraps the
worktree-local `core.hooksPath`, shared commitlint, and validator, then
registers each published skill under the ignored `.agents/skills/` directory
for Codex and `.claude/skills/` for Claude Code. The ignored
`.agents/setup-state` marker records the revision and bootstrap inputs after
all stages succeed.

After that initial setup, the tracked `post-checkout` hook reruns `mise run
setup:refresh` whenever Git creates or switches branches. It refreshes
revision-dependent local skills synchronously and does not repeat unchanged
bootstrap work. A failed refresh reports the failed stage and leaves the prior
ready marker unchanged. The local registration is not committed. The command
reconciles repository-owned links after a skill path or name change, removes
links for skills that are no longer published, and keeps both host directories
aligned. Verify it with `readlink .agents/skills/<skill-name>` and restart Codex
if a newly registered skill does not appear.

The command preserves external symbolic links and regular files at registration
paths. It prints the path and the action required when one conflicts with a
published skill. Remove or rename the conflicting entry, then rerun `mise run
setup:refresh`. A second run after a successful refresh makes no registration
changes.

Worktrees share the pinned commitlint binary through the common Git directory,
so setting up a new worktree does not rebuild it. The local skill registrations
are revision-dependent: the `post-checkout` hook refreshes them when the branch
changes, and `wt list` reports which worktree owns a branch. Confirm the
registration for a worktree with the `readlink` check above. See
`docs/worktrees.md` for the `worktrunk` workflow and its safe-removal rules.

The enabled hooks run `mise run check:local` before commits and `mise run
validate:all` before pushes. Fix a reported failure before retrying the commit or
push.

For `bootstrap-project`, see
`skills/govern/bootstrap-project/references/hosts/codex.md` for the Codex-specific
execution and handoff rules.
