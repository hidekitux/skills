# Mise task naming

Repository tasks use the form `verb:task-name`.

- `verb` is exactly one lowercase word from the repository's action vocabulary.
- `task-name` is one or more lowercase words separated by hyphens.
- Both components start with a lowercase letter and contain only lowercase letters, digits, and hyphens where permitted.
- A task must describe an action. The approved verb vocabulary is `check`, `evaluate`, `format`, `generate`, `install`, `lint`, `mutate`, `publish`, `setup`, `test`, `validate`, and `verify`.
- Top-level convenience tasks are not exceptions: aggregate operations use an explicit task name such as `validate:all` or `test:all`.
- A task name is a public repository interface. Renaming one requires updating every tracked invocation, dependency, example, workflow, hook, and document in the same change.
- The task validator rejects malformed declarations and references to retired names.

## Canonical task inventory

| Category | Task names |
| --- | --- |
| check | `check:all`, `check:branch-policy`, `check:diff`, `check:go-vuln`, `check:hosts`, `check:local`, `check:repository`, `check:skills` |
| evaluate | `evaluate:all`, `evaluate:smoke` |
| generate | `generate:public-status` |
| install | `install:fsl` |
| lint | `lint:all`, `lint:actions`, `lint:go`, `lint:python`, `lint:shell` |
| mutate | `mutate:fsl`, `mutate:fsl-changed` |
| publish | `publish:release` |
| setup | `setup:all`, `setup:commitlint`, `setup:local-skills` |
| test | `test:all`, `test:go`, `test:json` |
| validate | `validate:all`, `validate:skill-creator` |
| verify | `verify:fsl`, `verify:release` |
