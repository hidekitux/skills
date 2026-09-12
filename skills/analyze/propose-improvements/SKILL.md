---
name: propose-improvements
description: Investigate a project for high-impact improvement opportunities, determine whether a major redesign is warranted, and return evidence-backed architectural proposals with alternatives, migration boundaries, risks, costs, and expected effects. Use analyze-project for a general findings report without architecture decisions and refactor-code for small behavior-preserving refactors.
license: Apache-2.0
---

# Propose Improvements

## Todo List

1. **in progress:** Define the improvement objective, scope, success criteria, and readable inputs.
2. Investigate the project and build an evidence-backed inventory of improvement opportunities.
3. Decide which opportunities need local changes, coordinated changes, or a major redesign; compare supported redesign directions.
4. Complete the proposal and handoff only when its recommendation and limits are supported, or explain why more evidence is required.

Keep exactly one item in progress. Mark an item complete only when its stated
evidence exists. Use the host's native Todo List or maintain the same list as a
Markdown checklist when no native list is available.

## Workflow

### 1. Frame the improvement question

- Do not assume that a major redesign is already justified. For an open request such as “improve this project,” define the improvement objective, intended users or operators, success criteria, constraints, and authority available for the proposal.
- State what is in scope and out of scope. Read source, manifests, tests, runtime and deployment configuration, interface documentation, history, Issues, measurements, and incident or operational records when they can support the improvement question and are available.
- Use `analyze-project` when the requester wants a general project findings report and prioritized recommendations without choosing or comparing an architectural direction. Use `analyze-codebase` for a narrow code fact, `refactor-code` for a behavior-preserving local cleanup, `bootstrap-project` for initial setup, and `plan-issue` for implementation planning.
- Stop or narrow the proposal when the request is primarily a security threat model, performance measurement, product design, or another specialist workflow.

### 2. Investigate the project

- Survey the project end to end: architecture and boundaries, important data and control flows, tests, CI and release, documentation, dependencies, operations, ownership, history, and tracked work when those areas are relevant to the objective.
- Build an improvement inventory. For each candidate, record the user or operator impact, the affected boundary, the likely root cause, the change surface, and the next useful evidence.
- Record observed facts with a file, symbol, command, test, measurement, or decision constraint for each claim.
- Label inferences and design assumptions separately from observed facts. Do not treat a preferred technology, an unmeasured benefit, or an unverified dependency as evidence.
- Identify the current architecture, boundaries, important flows, coupling, ownership, operational constraints, and current failure or change costs that explain the candidates.
- State missing evidence and unresolved questions. If the missing evidence could change the target architecture, migration order, compatibility boundary, risk, or cost, stop before selecting a direction.

### 3. Select the proposal scope

- Classify each candidate as a local behavior-preserving change, a coordinated cross-boundary change, or a major redesign. Explain the classification and prioritize candidates by impact and confidence.
- Permit a large or radical redesign when the evidence shows that the current boundaries, data model, deployment model, ownership model, or accumulated change cost blocks the stated objective. Do not propose one from a single local smell or from technology preference alone.
- If no major redesign is supported, say so and return the highest-value improvement recommendations with the appropriate next-owner skill instead of forcing an architecture proposal.
- When a major redesign is supported, describe the target architecture: components, responsibilities, boundaries, data and control flows, interfaces, operational model, and invariants.
- When a major redesign is supported, show the current-to-target gap and name the dependencies that constrain migration order.
- When a major redesign is supported, compare the current state and at least one viable alternative against explicit criteria such as compatibility, reversibility, delivery risk, operating cost, team ownership, and expected effect. Select a direction only when the evidence supports it; otherwise report that the decision remains open.
- Keep technology choices subordinate to the architecture problem. Do not recommend a tool or platform merely because it is familiar or fashionable.

### 4. Describe a reversible direction

- When a major redesign is supported, break the selected direction into stages with a goal, dependency order, owner, exit evidence, compatibility boundary, and rollback condition for each stage.
- When a major redesign is supported, explain how old and new paths coexist, how data or interfaces stay compatible, how progress is observed, and what event triggers rollback or a pause.
- Estimate the costs that apply to each recommendation. For a major redesign, include implementation, migration, operation, and coordination costs. For local or coordinated changes, estimate implementation and coordination costs. Mark estimates as assumptions and give the evidence that would refine them.
- For local or coordinated changes, describe only the dependencies, rollout concerns, and rollback conditions that apply to that scope.
- State expected effects and how they will be measured. Include risks, mitigations, affected users or systems, and failure modes.

These are proposal-level boundaries, not an implementation plan. Do not break
work into executable tickets, write migration code, or decide details that
require the later `plan-issue` investigation.

### 5. Report and hand off

Return a proposal with:

- an executive summary, improvement objective, scope, and recommendation status;
- an inventory of improvement opportunities, including candidates that do not require rearchitecture;
- an evidence table separating observed facts, inferences, assumptions, and unresolved questions;
- when a major redesign is supported, the current architecture, target architecture, and current-to-target gap;
- when a major redesign is supported, alternatives considered, decision criteria, and the selected direction or an explicit evidence blocker;
- when a major redesign is supported, migration stages in dependency order, with compatibility and rollback boundaries;
- when a major redesign is not supported, the highest-value recommendations, their priority, evidence, scope, and next-owner skill without a target architecture or migration plan;
- risks, mitigations, cost assumptions, expected effects, and measurement signals;
- the next owner and the evidence required for handoff.

For each recommendation, name the next-owner skill. Use `create-issue` for
untracked governed change candidates, `debug-code` for a reproduced defect,
`write-tests` for focused test gaps, and `refactor-code` for a behavior-
preserving cleanup. When a Change Issue exists and a major proposal is
accepted, name `plan-issue` as the owner of the verified implementation plan.
Present the proposal for an explicit decision. Do not treat the proposal as
implementation authorization.

## Read-only guardrails

- Never edit source, tests, configuration, generated files, or documentation.
- Never create, edit, close, reprioritize, or link Issues; never create branches, Pull Requests, releases, or external system changes.
- Do not write an implementation plan, implementation code, migration scripts, executable tickets, or a release plan. `plan-issue` and `implement-issue` own those later phases.
- Do not recommend a redesign from a single local smell when `refactor-code` is sufficient.
- Do not replace a security threat model, performance measurement, or product design workflow when that is the primary question.
- Stop and report the missing evidence when a high-impact candidate, safe target, migration order, compatibility boundary, rollback condition, or cost estimate cannot be supported.

## Handoff

- When the requester accepts a recommendation and no governing Change Issue exists, hand the accepted `improvement-proposal` to `create-issue` as the normal next phase. `create-issue` owns Issue creation and its mutation confirmation; this skill does not create the Issue itself.
- When the requester accepts a recommendation and an accepted Change Issue already exists, hand the proposal to `plan-issue` for the verified implementation plan.
- If the requester has not accepted the proposal, stop at the decision and report the open choice or requested revision; do not advance to `create-issue`.
- A project review that finds valuable improvements but does not justify rearchitecture hands each recommendation to its named next-owner skill; untracked governed candidates go to `create-issue`.
- An evidence-limited request names the missing inputs and the appropriate next investigation owner instead of inventing a target architecture.
- The handoff includes the proposal, evidence locations, assumptions, unresolved questions, migration gates, rollback triggers, and the next owner's required decision.

## Writing quality

Use plain, active, evidence-backed prose. Name the file, symbol, command,
measurement, or constraint behind every repository claim.
