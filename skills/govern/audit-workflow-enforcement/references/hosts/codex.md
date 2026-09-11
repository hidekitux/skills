# Codex adaptation

Use Codex's native task list for the Todo List. After deterministic checks,
create at most three read-only subagents with independent prompts. Request
the project's Low tier model from `agent.low.model` in `opencode.json` (see
[docs/model-selection.md](../../../../docs/model-selection.md)); the selector
is served through codex-router. If that selector is unavailable, use the
lowest-cost capable model shown by the host and name the fallback in the
handoff.

Do not give a subagent write authority, credentials, an expected finding, or a
request to publish. Verify cited paths and commands in the parent task before
classifying a rule.
