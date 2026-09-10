# Compiled skill context

`workflow/skill-graph.yml` owns the context contract. Each cataloged skill has
a profile with four independent token budgets, critical invariants, and
declared modules. A module is either a repository-relative file or a signal
adapter. The compiler does not infer rules from prose or select a model.

## Compile a package

`cmd/compile-context` reads the selected skill, profile, and observable task
signals, then emits a JSON package:

```text
mise run generate:context -- --root . --skill plan-issue --signals task.json
```

The compiler counts every module with the pinned `cl100k_base` tokenizer from
`docs/skill-instruction-inventory.yml`. Required core instructions, critical
invariants, and required repository modules are admitted first. Optional
modules are ordered by category, priority, and module ID. A module that does
not match every declared activation group is excluded with a reason. An
optional module that does not fit is excluded as `budget-evicted`.

When required content exceeds one category budget, compilation stops and
returns `stop-and-escalate`. It never truncates a required file or silently
replaces it. The manifest reports category measurements, total measurements,
budgets, and every inclusion or exclusion decision.

## Signals and privacy

The input contract accepts task kind, repository-relative paths, diagnostic
producer and code pairs, FSL result codes, and prior-evidence kinds. It does
not accept prompts, command output, reasoning, credentials, or user content.
Signal-backed modules synthesize concise context from those structured fields.

`Manifest` is the trace-safe explanation of a package. It records module IDs,
source paths, reasons, required flags, and token counts, but never module
content. `workflow/skill-trace.schema.json` carries the manifest as optional
schema version 2 context provenance while the package returned to the caller
contains the selected content.

## Evaluation

Issue #173 evaluation records may use `context_mode: compiled`. The record
stores the manifest, and the paired full/compact report compares its measured
context tokens without parsing Markdown or host transcripts. Representative
selection and overflow cases are checked by `check-context`, which is part of
`check:repository`.
