# Bounded multi-agent deliberation

This policy decides when independent reasoning is likely to improve a high-risk
or uncertain decision. Its machine-readable source is
[`workflow/deliberation-policy.yml`](../workflow/deliberation-policy.yml).

## Default route

Run deterministic checks first. Use one agent for ordinary work, including
deterministic validation, a single-session edit, and work that shares mutable
state. The one-agent route is the required fallback when no trigger is present,
evidence cannot be separated, a required bound is unavailable, or a host cannot
provide the configured role tier.

Use deliberation only when at least one risk or uncertainty signal is present
and the evidence can be inspected independently. The signals are architectural
ambiguity, security sensitivity, high-risk migration, conflicting hypotheses,
and independently reviewable evidence. A signal is an observed task fact, not a
model score or a reason to expand authority.

| Signal | Use it when | Do not use it when |
| --- | --- | --- |
| Architectural ambiguity | More than one design changes behavior or structure. | The repository contract names one implementation. |
| Security sensitivity | Independent review can find an abuse path or data-handling error. | The canonical security check already decides the question. |
| High-risk migration | A format, state, dependency, or compatibility change is costly to roll back. | The change is a reversible local edit. |
| Conflicting hypotheses | Separate evidence supports competing explanations or findings. | The failure has one reproduced cause. |
| Independently reviewable evidence | Candidates can inspect isolated, read-only evidence. | Candidates need shared mutable state or hidden peer context. |

## Bounded patterns

Every pattern declares its authority, concurrency, independence, termination,
agent, token, retry, elapsed-time, and cost bounds in the policy source.

### Independent candidates

Run at most two read-only candidates in isolated contexts. Withhold expected
findings and other candidates' conclusions. The parent task owns any later
mutation and applies it serially after a judge decision. Stop when both
candidates finish or any declared bound is reached.

### Fan-out investigation

Run at most three read-only candidates in parallel, each with a disjoint
investigation scope. Give each candidate only its assigned evidence and the
shared immutable task contract. Stop when all candidates finish or a bound is
reached. The parent task serializes all state changes.

### Judge

Run one read-only judge after candidates finish. Give the judge sanitized
candidate result labels and safe evidence references, never raw prompts,
reasoning, tool output, source content, credentials, private URLs, or user
data. The judge must cite task evidence and may choose, reject, or return a
blocked result. A majority is a signal to inspect, not a decision rule.

## Authority and cost

Candidates and judges never receive more repository, Git, GitHub, external, or
data authority than the parent task. Concurrent shared-state mutation is
prohibited. Destructive and external mutations are never retried automatically.
If a candidate needs a mutation, it returns an evidence-backed proposal to the
parent task; the parent task decides and performs it through the owning
workflow.

The policy bounds maximum agents, retries, elapsed milliseconds, input tokens,
output tokens, and cost in micro-units. A missing cost meter is recorded as
unavailable, not as zero. Reaching any bound stops the pattern or returns a
blocked result. A host fallback uses the lowest-cost capable configured role
tier and records the fallback; the policy never names a provider or model.

## Trace and evaluation

When deliberation runs, the opt-in structured trace records the selected
signals and reason, pattern, independence, authority, concurrency, bounds,
sanitized candidate results, judge evidence, and measured marginal cost. The
trace does not persist raw model content. Single-agent traces record the
default route when the caller supplies a deliberation summary.

Issue #173 evaluates each retained trigger against the same single-agent
baseline. It compares deterministic quality and false positives, rubric
quality, context and output tokens, elapsed time, retries, and marginal cost.
An evaluation that lacks required bounds or evidence is not a pass. Issue #204
may combine this policy with task risk, context budgets, validation tiers, and
model tiers; it does not redefine these patterns.
