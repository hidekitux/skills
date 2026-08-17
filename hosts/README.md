# Host-specific guidance

This directory contains repository-level setup notes and configuration examples
for individual agent hosts. It is not a skill discovery directory and must not
contain publishable `SKILL.md` files.

Put portable skill behavior in `skills/`. Put a host adaptation beside the
relevant skill when the behavior changes for that host.

## Model selection

The shared model-role convention (High / Mid / Low tiers, per-project defaults,
and the fallback rule) is defined in
[docs/model-selection.md](../docs/model-selection.md). The OpenCode CLI is not
used: OpenCode Go models are served to Codex through codex-router and to Claude
Code through claude-code-router. The per-project tier variables live in
`opencode.json`; see [opencode-go/README.md](opencode-go/README.md). The Codex
and Claude Code notes in this directory record how each host applies the same
variables.
