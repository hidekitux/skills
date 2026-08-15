# Codex adaptation

The core `project-bootstrap` workflow is portable. Apply this note only when the
active host is Codex.

## Establish safe context first

- Read the applicable `AGENTS.md` instructions before inspecting, creating, or
  changing project files. Respect more specific instructions in relevant
  subdirectories.
- Inspect the repository status and existing project conventions before editing.
  Preserve unrelated dirty work and do not widen the requested scope.
- Read `mise.toml` and list available mise tasks before choosing commands. Use
  `mise run <task>` when a task exists; do not substitute an ad-hoc direct
  command merely because it is available locally.

## Select capabilities deliberately

- Use only tools, plugins, MCP servers, browser sessions, and local binaries
  that are actually available in the active Codex environment. Inspect before
  relying on a named capability.
- Prefer read-only discovery before making changes. Request approval only when
  an action needs new authority, external coordination, a network download, or
  a destructive operation beyond the project task.
- Do not create or edit `.codex/` by default. Treat it as local Codex state;
  keep portable project instructions in `AGENTS.md` and shared commands in
  `mise.toml`.

## Apply FSL without overclaiming

- Confirm that `fslc` is available before promising FSL verification. Use an
  appropriate FSL authoring skill when available; otherwise follow the
  formalization memo fallback in `references/fsl-adoption.md`.
- Preserve the formalization memo and confirmed-assumption workflow. If a
  project rule is incomplete, report the gap instead of inventing a transition,
  invariant, or approval rule.
- Run FSL checks through `mise run verify-fsl` when that task exists. Report
  whether a result is a checked specification property, generated test evidence,
  or implementation replay evidence; these are not interchangeable.

## Handoff

- Report the commands actually run, their results, files changed, and anything
  intentionally not run. Do not describe unavailable or skipped checks as
  successful.
- Keep Codex UI metadata in `agents/openai.yaml`; it augments the portable skill
  rather than replacing its workflow.
