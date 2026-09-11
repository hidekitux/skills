# Claude Code adaptation

Use Claude Code's task list when available, otherwise maintain the Markdown
Todo List. After deterministic checks, create at most three independent,
read-only Task subagents. Select the project's Low tier model from
`agent.low.model` in `opencode.json` (see
[docs/model-selection.md](../../../../docs/model-selection.md)); the request
is routed through claude-code-router. If that selector is unavailable, use the
least-cost capable model and state the fallback in the handoff.

Do not give a subagent write authority, credentials, an expected finding, or a
request to publish. Verify cited paths and commands in the parent task before
classifying a rule.
