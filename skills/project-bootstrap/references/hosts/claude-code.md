# Claude Code adaptation

The core `project-bootstrap` workflow is portable. Apply this note only when the
active host is Claude Code.

## Project context

- Read the repository's `CLAUDE.md` files, if present, before changing project
  structure or commands. Treat them as project instructions alongside the
  repository's existing conventions.
- Keep `mise.toml` and `mise run <task>` as the standard tool and task entry
  point. Do not replace them with Claude Code-specific shell aliases or settings.

## Capability boundaries

- Do not assume a hook, MCP server, browser session, plugin, or local binary is
  available. Inspect the current environment before relying on it.
- Do not create `.claude/` merely to bootstrap a project. Add Claude-specific
  configuration only when the user requests it or the project has a confirmed
  need, and keep shared workflow rules in portable project files.
- Ignore `agents/openai.yaml`; it is Codex UI metadata and does not alter the
  portable `SKILL.md` workflow.

## Verification and handoff

- Run the applicable `mise run` tasks and report their exact result.
- If a required capability is unavailable, state the missing capability and use
  the documented project fallback. Do not claim that an unrun check passed.
