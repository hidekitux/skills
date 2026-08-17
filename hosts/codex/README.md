# Codex guidance

Use this directory for repository-level Codex setup examples only. Publishable
skills remain in `skills/`; Codex-facing UI metadata belongs in each relevant
skill's `agents/openai.yaml`.

Do not assume a particular plugin, MCP server, browser session, or local binary
exists. Record any material capability difference in the relevant skill's
`references/hosts/codex.md`, including a safe fallback and verification step.

## Model selection

Skills select role-specific models from the shared convention in
[docs/model-selection.md](../../docs/model-selection.md). The per-project
variables live in `opencode.json` under `agent.high.model`, `agent.mid.model`,
and `agent.low.model`; see [opencode-go/README.md](../opencode-go/README.md)
for the OpenCode Go setup. In Codex the selectors are served through
codex-router (for example `opencode-go/deepseek-v4-flash`): the default agent
model is set once in the host config, and subagents inherit it or take an
explicit selector when the host allows a model override. When Codex cannot
read the variable, use the documented default for the tier and the fallback
rule, and name the fallback in the handoff.

## Local skill registration

Run `mise run setup` once in each Git worktree. The command sets
the worktree-local `core.hooksPath` to `.githooks` and registers each
top-level published skill under the ignored `.agents/skills/` directory for
Codex and `.claude/skills/` for Claude Code.

After that initial setup, the tracked `post-checkout` hook reruns `mise run
setup` whenever Git creates or switches branches. It refreshes local skills,
Git Hooks, and commitlint without blocking checkout if setup fails. The local
registration is not committed. Verify it with `readlink
.agents/skills/<skill-name>` and restart Codex if a newly registered skill does
not appear.

Worktrees share the pinned commitlint binary through the common Git directory,
so setting up a new worktree does not rebuild it. The local skill registrations
are snapshot-dependent: the `post-checkout` hook refreshes them when the branch
changes, and `mise run worktree:diagnose -- --branch issue/<number>` reports
which worktree owns a branch and whether its setup is current.

The enabled hooks run `mise run check:local` before commits and `mise run
validate` before pushes. Fix a reported failure before retrying the commit or
push.

For `bootstrap-project`, see
`skills/bootstrap-project/references/hosts/codex.md` for the Codex-specific
execution and handoff rules.
