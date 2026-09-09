# Failure promotion

This document defines how a confirmed agent failure becomes durable regression
evidence and, when justified, an enforcement rule. The process applies to
failures observed in a skill, evaluation, validator, execution trace, or
repository workflow. It does not store private transcripts or change a rule
without review.

## Start with a sanitized reproduction

Record a failure only after a minimal reproduction produces the same observable
problem twice or produces one deterministic failure with a clear expected
outcome. The reproduction must use synthetic or public input, omit credentials
and private paths, and name the fixture, commands, or trace needed to run it.
The record also names the source evidence, the accountable owner, and the
regression asset that will detect a recurrence.

Classify the cause before choosing an enforcement layer:

| Cause | Meaning |
| --- | --- |
| `instruction` | The written contract is missing, contradictory, or too vague. |
| `context_selection` | The agent loaded the wrong task, reference, or repository context. |
| `tool_use` | The agent selected or used a tool incorrectly even though the contract and context were sufficient. |
| `model_behavior` | The same contract and tools produce an incorrect model decision. |
| `repository_design` | The repository leaves a required invariant unenforced or ambiguous. |
| `external_state` | A dependency, service, permission, or remote work item changed outside the repository. |
| `infrastructure` | The host, runner, credential setup, or execution environment prevented the run. |

An `external_state` or `infrastructure` classification is not a repository
defect by itself. Keep its evidence for measurement, but do not promote it to a
permanent product rule unless a new reproduction proves a repository-owned
cause.

## Choose the lowest sufficient layer

Try the following layers in order and stop at the first one that expresses the
invariant and prevents the reproduced failure:

1. An evaluation fixture for a behavioral contract that must stay observable.
2. A static check for a fixed repository shape or metadata rule.
3. A type constraint for a value or state that must be impossible to represent.
4. A focused test or property test for executable behavior.
5. An FSL state invariant for a finite workflow transition.
6. Runtime isolation for a boundary that code or static checks cannot safely enforce.
7. Concise natural-language guidance when the behavior depends on judgment or context.

The order is a decision aid, not an instruction to add every layer. A promoted
record must name its layer and owner. A mechanical addition must also say
whether it replaces, compresses, or intentionally retains the instruction that
led to the failure. Keep the instruction when removing it would hide a safety
boundary, change the host-neutral contract, or make the expected outcome harder
to discover.

## Connect evaluation and instruction reduction

Issue #173 supplies the behavioral evidence. Its deterministic assertions must
fail when the known failure is reintroduced and pass after the correction.
Rubric scores describe quality but never turn a deterministic failure into a
pass. Records retain host, model, skill, repository revision, run identifier,
and outcome so repeated failures can be compared across revisions.

For Issue #197, compare a full synthetic instruction fixture with its compact
form. Remove or compress one instruction only when the compact form preserves
the required deterministic assertions, safety boundaries, handoff, and output,
and retained evaluation evidence shows no material regression. Keep the full
instruction when the paired evidence is incomplete or when the compact form
depends on an unstated host behavior.

## Review and measurement

The machine-readable records under `workflow/failure-records/` are validated by
`check-evaluation`. A promoted record needs a sanitized reproduction, expected
outcome, owner, source evidence, regression asset, and at least two observations
before it can represent a repeated failure. Candidate and not-promoted records
remain useful evidence without claiming that a repository rule is warranted.

The check distinguishes deterministic failure, behavioral failure, skipped case,
infrastructure error, and resolved outcome. The human review decides whether a
failure is causal, whether the selected layer is sufficient, and whether an
instruction may be removed. No check changes instructions, catalog status, or
external work items automatically.
