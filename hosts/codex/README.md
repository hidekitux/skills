# Codex guidance

Use this directory for repository-level Codex setup examples only. Publishable
skills remain in `skills/`; Codex-facing UI metadata belongs in each relevant
skill's `agents/openai.yaml`.

Do not assume a particular plugin, MCP server, browser session, or local binary
exists. Record any material capability difference in the relevant skill's
`references/hosts/codex.md`, including a safe fallback and verification step.

## Local skill registration

Run `mise run setup` once in each Git worktree. The command sets
the worktree-local `core.hooksPath` to `.githooks` and registers each
top-level published skill under the ignored `.agents/skills/` directory for
Codex and `.claude/skills/` for Claude Code.

After that initial setup, the tracked `post-checkout` hook registers skills
whenever Git creates or switches branches. The local registration is not
committed. Verify it with `readlink .agents/skills/<skill-name>` and restart
Codex if a newly registered skill does not appear.

For `bootstrap-project`, see
`skills/bootstrap-project/references/hosts/codex.md` for the Codex-specific
execution and handoff rules.
