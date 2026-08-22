# Model selection

This document is the shared convention for role-specific model selection. Skills and host notes reference it instead of hard-coding model names, so a project can change any role's model in one place.

## Role tiers

The role tiers are the stable contract; the model names are overridable per-project defaults.

- High: complex planning and directing.
- Mid: moderate planning and implementation.
- Low: implementation, validation, and simple Subagent tasks.

## Per-project configuration

The default models live in `opencode.json` in the project root. Each tier is one variable that a project can override:

- `agent.high.model`
- `agent.mid.model`
- `agent.low.model`

The current defaults are:

| Tier | Default model | Alternative examples |
| --- | --- | --- |
| High | `gpt-5.6-terra` | `claude-opus-5` |
| Mid | `opencode-go/deepseek-v4-pro` | |
| Low | `opencode-go/deepseek-v4-flash` | `gpt-5.6-luna`, `claude-sonnet-5` |

The key names were verified against the OpenCode config schema at <https://opencode.ai/config.json>. `model` is a string and the schema rejects arbitrary top-level keys, so `model.high` is not a valid key; the tiers are declared as custom agents (`agent.<tier>.model`), which the schema allows. Treat the examples as today's preferences, not a fixed contract: changing a tier means editing its variable in the project config, never each `SKILL.md` or handoff.

## Runtime

The OpenCode CLI is not part of this setup. The models are provided by OpenCode and served to the agents through routers: Codex reaches the selectors through codex-router, and Claude Code reaches them through claude-code-router. `opencode.json` is the shared per-project variable contract those hosts read; it remains schema-valid if the CLI is ever used, but no skill depends on the CLI.

## Subagent use

Use Subagents only for work that is independent, read-only, and parallelizable with bounded context. Do not use them for deterministic checks, single-session edits, or work that must share mutable state.

## Fallback

When the configured model selector is unavailable on a host, use the host's lowest-cost capable model and name the fallback in the handoff. A host that does not expose the config variable applies the same rule with the documented default for the tier.

Host-specific reading notes are in `hosts/codex/README.md` and `hosts/claude-code/README.md`. Router consumption and verification guidance is in [model-routing.md](model-routing.md).
