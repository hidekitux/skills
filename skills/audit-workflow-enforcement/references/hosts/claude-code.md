# Claude Code adaptation

Use Claude Code's task list when available, otherwise maintain the Markdown
Todo List. After deterministic checks, create at most three independent,
read-only Task subagents. Select Sonnet 5 when the host exposes that exact
model; otherwise select its `sonnet` tier or the least-cost capable model and
state the fallback in the handoff.

Do not give a subagent write authority, credentials, an expected finding, or a
request to publish. Verify cited paths and commands in the parent task before
classifying a rule.
