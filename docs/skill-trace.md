# Skill execution trace

`workflow/skill-trace.schema.json` defines the versioned record for one skill
run. The trace is host-neutral. A host adapter maps provider events into the
semantic events in this schema. Schema version 3 may include the privacy-safe
context manifest produced by `cmd/compile-context` and the deliberation
summary defined by [multi-agent deliberation](multi-agent-deliberation.md).

## Persistence boundary

Trace persistence is opt-in. A caller enables it by supplying a destination to
the trace writer. Ordinary skill execution and ordinary evaluation runs do not
write traces. The writer emits one validated JSON object per line, so an
interrupted writer can leave the last run identifiable without storing a raw
transcript.

The writer uses an allowlist. It stores run identity, skill and graph versions,
host and model identifiers, repository revision, ordered event outcomes, safe
evidence references, usage numbers, context provenance, deliberation routing
and cost provenance, and terminal state. It omits prompts,
model reasoning, tool arguments, tool output, source contents, credentials,
user data, and unknown host fields.

The context manifest is provenance only. It records selected module IDs, source
paths, activation decisions, budgets, and fixed-encoding token counts; it never
records selected module content.

Permitted strings are checked again for credentials and URLs. Public GitHub
Issue, Pull Request, and commit references may remain; other URLs are replaced
with a redaction marker before persistence. A redaction failure rejects the
trace. The writer records the number of redactions but never records the
rejected value. The default retention window is 30 days, and callers must
delete local traces within that window.

## Semantic events

Each event has a strictly increasing sequence number, a UTC timestamp, a kind,
and a status. The event kind selects one payload:

- `run_started` identifies the beginning of the run.
- `todo_transition` records one Todo List item state change and an optional
  safe evidence reference.
- `tool_outcome` records the tool identifier, result, failure class, and
  retryable flag. It never records arguments or output.
- `validation_outcome` records the validation identifier, result, failure class,
  and optional diagnostic references. It never records command output or a
  diagnostic message.
- `evidence` records a safe path, command identifier, public Issue or Pull
  Request reference, commit, or validation reference.
- `handoff` records the destination skill and named artifact.
- `retry` records a bounded attempt and a reason code.
- `run_finished` records the terminal status.

## Deliberation summary

The optional `deliberation` summary records the routing signals and reason,
selected pattern, independence, authority, concurrency, resource bounds,
sanitized candidate result labels, judge evidence, and marginal cost. A
multi-agent record is invalid without isolated or candidate-only context,
read-only authority, explicit bounds, candidate evidence, judge evidence, and
measured marginal cost. Raw candidate output, prompts, reasoning, and peer
conclusions are not trace fields.

The terminal status is one of `success`, `failed`, `skipped`, `interrupted`,
or `infrastructure_error`. Failure classifications distinguish deterministic
failure, behavioral failure, infrastructure error, user interruption, and
intentional skip. A trace with no terminal event is invalid for persistence;
the writer records an interrupted terminal event when the caller cancels a
run.

## Correlation and consumers

`run_id` correlates all events from one run. `skill_id` and `skill_version`
identify the cataloged skill, `graph_version` identifies the
`workflow/skill-graph.yml` schema version, and `repository_revision` identifies
the source revision. `scenario_id` is optional evaluation context.

Issue #173 consumes the JSONL trace and existing structured evaluation records.
It does not parse Markdown reports or host transcripts. Trace validation proves
schema, ordering, classification, and privacy invariants. It does not prove
that a host emitted every event or that a skill followed its `SKILL.md` prose.

## Cross-skill replay

`cmd/replay-skill-trace` reads the ordered JSONL records for one Issue-backed
change and checks each handoff against `workflow/skill-graph.yml`. The replay
requires a completed Todo item, a safe evidence reference, and a successful
validation before a successful handoff. It also checks the closed canonical
tool map, so a read-only skill cannot use a write operation owned by another
skill. An unknown or missing required operation returns `incomplete_evidence`.

Use the repository fixture command with a representative trace:

```text
go run ./cmd/replay-skill-trace --root . --input workflow/replay-fixtures/valid.jsonl --format json
```

The JSON report contains normalized actions and numeric source boundaries. The
public FSL input contains only those actions. It does not contain prompts,
reasoning, tool arguments, tool output, source contents, credentials, private
URLs, or user data. `incomplete_evidence` means that the records cannot prove a
required transition; `interrupted` records an explicit interrupted terminal;
`retry_exhausted` records the graph's bounded review outcome. None of these
outcomes is a successful replay. A successful review may terminate the replay
or hand off to `merge-pr`; a successful merge is recorded as the terminal
`complete_merge` action.

Validation events may reference a diagnostic by its stable `producer` and
`code` pair. The reference is safe to persist because it contains no message,
observed value, command output, or evidence content. The diagnostic contract is
defined in [validator-diagnostics.md](validator-diagnostics.md).

## Host boundary

Codex and Claude Code may expose different event names and payload shapes.
Their adapters normalize only observable fields that map to the semantic
contract. An unknown event fails closed or is reported as an adapter finding;
the adapter never guesses a handoff, outcome, or evidence reference.
