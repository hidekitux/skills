# Execution strategy

The repository selects an execution strategy from finite task signals before
host work begins. The selector is defined by
[`workflow/execution-strategy-policy.yml`](../workflow/execution-strategy-policy.yml)
and implemented by `internal/strategy`. It records a model tier, context
profile, validation tier, parallelism, retry bound, elapsed-time bound,
escalation, authority projection, reasons, and safe evidence references.

## Selection rules

The selector evaluates ordered predicates for impact, reversibility, ambiguity,
security sensitivity, state mutation, evidence quality, and validation cost.
The first matching predicate wins. Deterministic checks run before any
additional reasoning. An unknown signal selects `blocked-evidence` and asks
for the missing evidence.

The selector uses the existing skill graph for authority, the existing context
profiles for context, and the existing deliberation policy for independent
candidates. It does not grant authority, choose a provider model name, or
reimplement those contracts. A skill without a graph context profile cannot
run through the selector.

## Safety and fallback

The selected authority is the graph-derived ceiling. External mutation is
blocked when the selected skill does not declare that authority. An explicit
validation-tier choice may raise the required tier but cannot lower it. A
model-tier choice is preserved when it is valid.

When the requested model tier is unavailable, the caller may provide the
configured tier set to the selector. The selector chooses the lowest-cost
configured tier and records `model-unavailable-fallback`. If no configured tier
is available, it asks the user. External mutation never receives an automatic
mutation retry.

The selector stores identifiers and classifications only. Prompts, model
reasoning, command output, credentials, source content, and user data are not
valid selector input or decision evidence. The decision can be embedded in
the version 4 skill trace under the optional `strategy` field.

## Validation and comparison

Run `mise run validate:all` for the repository validation surface. The focused
policy check is `mise run check:repository`, and the
selector is available through `go run ./cmd/select-execution-strategy` for
JSON input and JSON or text output. The repository check invokes the focused
validator.

Behavioral evaluation scenarios may declare `execution_strategy` with a fixed
baseline. Their JSONL record stores both the adaptive decision and a
deterministic comparison of validation, retry, elapsed-time, and parallelism
bounds. Host token and cost usage remains unavailable unless the host driver
exposes it; unavailable usage is not reported as zero.
