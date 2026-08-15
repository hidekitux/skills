# Claude Code guidance

Use this directory for repository-level Claude Code setup examples only.
Publishable skills remain in `skills/`; do not duplicate their common workflow
here.

Do not assume hooks, settings, MCP servers, or a particular local binary exist.
Record any material capability difference in the relevant skill's
`references/hosts/claude-code.md`, including a safe fallback and verification
step.

## Local skill registration

Run `mise run setup` once in each Git worktree. The command sets
the worktree-local `core.hooksPath` to `.githooks` and registers each
top-level published skill under the ignored `.claude/skills/` directory for
Claude Code and `.agents/skills/` for Codex.

After that initial setup, the tracked `post-checkout` hook registers skills
whenever Git creates or switches branches. The local registration is not
committed. Verify it with `readlink .claude/skills/<skill-name>`. Claude Code
detects changes to an existing skill directory during the session; restart it
when the top-level `.claude/skills/` directory was created after startup.
