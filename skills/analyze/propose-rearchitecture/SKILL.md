---
name: propose-rearchitecture
description: Turn evidence about a system into a bounded proposal for a major redesign, architectural refactor, or foundational migration, including alternatives, staged migration, compatibility, rollback, risks, costs, and expected effects. Use analyze-codebase for local code facts, analyze-project for whole-project investigation, and refactor-code for small behavior-preserving refactors.
license: Apache-2.0
---

# Propose Rearchitecture

## Todo List

1. **in progress:** Define the redesign question, scope, decision authority, and readable inputs.
2. Build and validate the evidence record, separating observed facts, inferences, assumptions, and unresolved questions.
3. Compare the current state with a target architecture, alternatives, migration stages, dependencies, compatibility, rollback, risks, costs, and expected effects.
4. Complete the proposal and handoff only when the decision and its limits are supported, or explain why more evidence is required.

Keep exactly one item in progress. Mark an item complete only when its stated
evidence exists. Use the host's native Todo List or maintain the same list as a
Markdown checklist when no native list is available.

## Workflow

### 1. Define the decision

- Confirm that the request concerns a major redesign, architectural refactor, or foundational migration that crosses module, data, deployment, or ownership boundaries.
- State what is in scope and out of scope, the decision to be made, the constraints, and the authority available for the proposal. Read source, manifests, tests, runtime and deployment configuration, interface documentation, history, measurements, and incident or operational records only when they are in scope and available.
- Route ordinary local refactoring to `refactor-code`, focused codebase facts to `analyze-codebase`, a whole-project governance review to `analyze-project`, initial project setup to `bootstrap-project`, and implementation planning to `plan-issue`.
- Stop or narrow the proposal when the request is primarily a security threat model, performance measurement, product design, or another specialist workflow.

### 2. Build the evidence record

- Record observed facts with a file, symbol, command, test, measurement, or decision constraint for each claim.
- Label inferences and design assumptions separately from observed facts. Do not treat a preferred technology, an unmeasured benefit, or an unverified dependency as evidence.
- Identify the current architecture, boundaries, important flows, coupling, ownership, operational constraints, and current failure or change costs.
- State missing evidence and unresolved questions. If the missing evidence could change the target architecture, migration order, compatibility boundary, risk, or cost, stop before selecting a direction.

### 3. Design and compare

- Describe the target architecture: components, responsibilities, boundaries, data and control flows, interfaces, operational model, and invariants.
- Show the current-to-target gap and name the dependencies that constrain migration order.
- Compare the current state and at least one viable alternative against explicit criteria such as compatibility, reversibility, delivery risk, operating cost, team ownership, and expected effect. Select a direction only when the evidence supports it; otherwise report that the decision remains open.
- Keep technology choices subordinate to the architecture problem. Do not recommend a tool or platform merely because it is familiar or fashionable.

### 4. Make the migration reversible

- Break the selected direction into stages with a goal, dependency order, owner, exit evidence, compatibility boundary, and rollback condition for each stage.
- Explain how old and new paths coexist, how data or interfaces stay compatible, how progress is observed, and what event triggers rollback or a pause.
- Estimate implementation, migration, operation, and coordination costs. Mark estimates as assumptions and give the evidence that would refine them.
- State expected effects and how they will be measured. Include risks, mitigations, affected users or systems, and failure modes.

### 5. Report and hand off

Return a proposal with:

- an executive summary, decision question, scope, and recommendation status;
- an evidence table separating observed facts, inferences, assumptions, and unresolved questions;
- the current architecture, target architecture, and current-to-target gap;
- alternatives considered, decision criteria, and the selected direction or an explicit evidence blocker;
- migration stages in dependency order, with compatibility and rollback boundaries;
- risks, mitigations, cost assumptions, expected effects, and measurement signals;
- the next owner and the evidence required for handoff.

When no governed Change Issue exists, name `create-issue` as the owner of the
next governed change. When a Change Issue exists and the proposal is accepted,
name `plan-issue` as the owner of the verified implementation plan. Do not
imply that the proposal authorizes implementation.

## Read-only guardrails

- Never edit source, tests, configuration, generated files, or documentation.
- Never create, edit, close, reprioritize, or link Issues; never create branches, Pull Requests, releases, or external system changes.
- Do not write an implementation plan, implementation code, migration scripts, or a release plan. `plan-issue` and `implement-issue` own those later phases.
- Do not recommend a redesign from a single local smell when `refactor-code` is sufficient.
- Do not replace a security threat model, performance measurement, or product design workflow when that is the primary question.
- Stop and report the missing evidence when a safe target, migration order, compatibility boundary, rollback condition, or cost estimate cannot be supported.

## Handoff

- A supported proposal hands its `rearchitecture-proposal` to `create-issue` when governed work is not yet tracked, or to `plan-issue` when an accepted Change Issue already exists.
- An evidence-limited request names the missing inputs and the appropriate next investigation owner instead of inventing a target architecture.
- The handoff includes the proposal, evidence locations, assumptions, unresolved questions, migration gates, rollback triggers, and the next owner's required decision.

## Writing quality

Use plain, active, evidence-backed prose. Name the file, symbol, command,
measurement, or constraint behind every repository claim.
