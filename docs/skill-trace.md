# Skill execution trace

`workflow/skill-trace.schema.json` defines the versioned record for one skill
run. The trace is host-neutral. A host adapter maps provider events into the
semantic events in this schema.

## Persistence boundary

Trace persistence is opt-in. A caller enables it by supplying a destination to
the trace writer. Ordinary skill execution and ordinary evaluation runs do not
write traces. The writer emits one validated JSON object per line, so an
interrupted writer can leave the last run identifiable without storing a raw
transcript.

The writer uses an allowlist. It stores run identity, skill and graph versions,
host and model identifiers, repository revision, ordered event outcomes, safe
evidence references, usage numbers, and terminal state. It omits prompts,
model reasoning, tool arguments, tool output, source contents, credentials,
user data, and unknown host fields.

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

Validation events may reference a diagnostic by its stable `producer` and
`code` pair. The reference is safe to persist because it contains no message,
observed value, command output, or evidence content. The diagnostic contract is
defined in [validator-diagnostics.md](validator-diagnostics.md).

## Host boundary

Codex and Claude Code may expose different event names and payload shapes.
Their adapters normalize only observable fields that map to the semantic
contract. An unknown event fails closed or is reported as an adapter finding;
the adapter never guesses a handoff, outcome, or evidence reference.
