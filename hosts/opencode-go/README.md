# OpenCode Go guidance

Use this directory for repository-level OpenCode Go setup examples only. Publishable skills remain in `skills/`.

## Model source and routers

This project consumes models provided by OpenCode Go. The OpenCode CLI is not part of the runtime: Codex reaches the model selectors through codex-router, and Claude Code reaches them through claude-code-router.

The shared role tiers and their per-project variables are defined in [docs/model-selection.md](../../docs/model-selection.md). Each tier is one variable in the project root `opencode.json`, for example `agent.low.model`.

## Reading the variable from a skill

Read the tier variable from the project root config, then select that exact model selector on the host. When the config is absent or the selector is unavailable, use the documented per-project default for the tier, then the host's lowest-cost capable model, and name the fallback in the handoff.

## Router consumption

- Codex: codex-router exposes the OpenCode Go selectors (for example `opencode-go/deepseek-v4-flash`). The default agent model is set once in the host config; subagents inherit the parent routed model or take an explicit selector when the host allows a model override.
- Claude Code: claude-code-router maps Claude Code requests to the OpenCode Go provider. Select the tier model in the router's provider/profile configuration.

## Verification

Compare the configured variable with the model the host actually used:

```bash
jq -r '.agent.low.model' opencode.json
```

- Codex: spawn a read-only subagent with an explicit model override equal to `agent.low.model` and ask it to report the exact model it runs as. The resolved model is also recorded in the codex-router meter at `~/.codex/codex-router/usage-events.jsonl`.
- Claude Code: select the tier through the claude-code-router provider profile and confirm the resolved provider and model in the router logs.
