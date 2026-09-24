# Validator diagnostics

`workflow/validator-diagnostic.schema.json` defines the versioned contract for
one validator diagnostic. A diagnostic gives a consumer the smallest safe set
of facts needed to route a failure: its producer, stable code, category, source
command, location, rule or invariant, expected and observed state, evidence,
retryability, and remediation category.

## Categories and codes

Each diagnostic has exactly one category:

- `validation_failure` means the input or repository violated a named rule.
- `infrastructure_error` means the validator or its dependency could not run.
- `unsupported_input` means the producer cannot handle the supplied input.
- `incomplete_evidence` means the result cannot establish the required fact.

The `producer` and `code` pair is stable across prose changes. Evaluation
fixtures and consumers use that pair instead of matching the human message.
The source command identifies the command or validator that emitted the
diagnostic. A location is a repository-relative path with optional line and
column numbers.

## Two renderings from one value

The structured rendering is one JSON diagnostic. A JSONL stream contains one
diagnostic per line and keeps the order in which a producer found failures.
The human rendering is concise text produced from the same value. It includes
the producer, code, category, message, and every present routing field. A
renderer never invents facts or changes the stable code.

Detailed command output remains available beside the diagnostic. The
diagnostic does not replace logs, tool output, host transcripts, prompts,
reasoning, source contents, or user data.

## Privacy boundary

Persistence uses an allowlist. The diagnostic writer redacts credentials,
private URLs, local addresses, and private evidence before it emits JSON or
JSONL. It omits unsafe evidence and fails closed when it cannot validate a
value. Public GitHub Issue, Pull Request, and commit references may remain only
when they have no query, fragment, credentials, or private host.

The redaction summary records the mode, count, and omitted field names. It
never records a rejected value. Trace validation events store only the
diagnostic `producer` and `code` pair. The #198 context compiler selects
diagnostics by structured fields and receives safe evidence references without
parsing the human rendering.

## Producer boundary

The first repository adapters cover branch-policy and Issue-body validation,
host installation validation, repository-check aggregation, and FSL verification
or mutation reports. They preserve existing exit codes and detailed output.
Issue #196 owns cross-skill FSL replay, so this contract exposes the mapping
seam without claiming that current FSL reports are replay conformance evidence.
