# FSL adoption guide

Use this guide when bootstrapping a project that may benefit from FSL. FSL is a
living specification and verification tool, not a generic project template.

## Decide fit before creating a specification

Classify a flow in this order.

1. **State machine:** Can its states and operations be drawn as a finite set of
   boxes and arrows? If not, FSL is out of scope for the core behavior.
2. **Payoff:** Is there a forbidden state reachable through interaction, or would
   the project benefit from keeping a requirements or design document as a
   checkable source of truth? Either payoff is sufficient.
3. **Finite and discrete:** If the core depends on real-time quantities,
   probability, continuous values, or free-text meaning, use FSL only for a
   discrete abstraction and say what remains outside the model.

Mark a flow as `in scope now`, `thin/low priority`, or `out of scope`. A simple
flow that would be documented anyway is often a thin FSL candidate; do not label
it out of scope merely because it has low verification risk.

## Add FSL proportionately

For a project with confirmed FSL scope:

- Put source specifications in `specs/`.
- Expose a `mise run verify-fsl` task that runs `fslc check` and `fslc verify --depth 8`.
- Start with one real, prioritized flow rather than a placeholder specification.
- Re-run the FSL checks on changes to the covered flow. Add CI only when `fslc`
  is available in that environment.
- For high-risk specifications, add induction, mutation testing, and positive
  reachability or acceptance checks so a passing property is not vacuous.

Use one specification for a localized design risk. Use connected business,
requirements, and design layers only when those layers already exist and
cross-layer alignment is valuable. Do not manufacture layers solely to use
refinement.

## Formalization memo

Before writing or changing a `.fsl` specification derived from natural language,
present a memo in the conversation and obtain confirmation for behavior-changing
choices. Include:

- candidate states, actions, actors, and finite domains;
- each rule's trigger, constraint, exception, and boundary interpretation;
- confirmed assumptions versus representation-only modeling choices; and
- unresolved decisions about lifecycle, retry, ownership, priority, and timing.

Use an appropriate FSL authoring skill when one is available. If one is not
available, this guide and the confirmed memo are the fallback procedure; do not
invent missing behavior merely to proceed.

Do not create the memo as a separate tracked file. Keep confirmed assumptions
with the specification as comments or typed requirement/assumption annotations.
If a required decision is missing, ask rather than invent a guard, invariant, or
transition.

## Verification boundary

`fslc` establishes properties of the specification, not automatically of the
implementation. Where implementation conformance matters, add generated tests
with an adapter or replay mapped execution evidence. Read counterexamples and
action coverage after every repair; a stronger guard can accidentally make useful
behavior unreachable.
