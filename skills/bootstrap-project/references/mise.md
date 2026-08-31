# Mise project convention

Use mise as the standard entry point for project tools and repeatable commands.
Create or update `mise.toml` at the project root before documenting development
commands.

## Tools

Pin only tools the project truly requires. Preserve an existing pin unless the
user asks to upgrade it. If the language, runtime, package manager, or version is
not known, ask or inspect existing project files rather than selecting one.

## Tasks

Define only tasks that apply to the project:

- `format` for formatting;
- `lint` for static analysis;
- `test` for automated tests;
- `verify:fsl` for FSL checks when FSL is in scope; and
- `check` to compose the applicable validation tasks.

Use `mise run <task>` in documentation, automation, and handoff notes. Do not
invent empty tasks merely to match this list. Let a task fail on its first failing
command so the failure is visible.

Keep complex shell logic in a repository script and invoke it from a mise task.
This preserves shell syntax highlighting and testability while leaving mise as the
single user-facing command entry point.
