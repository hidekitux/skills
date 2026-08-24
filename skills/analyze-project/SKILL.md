---
name: analyze-project
description: Investigate a repository or project end to end (architecture, tests, CI/release, documentation drift, technical debt, dependencies, and FSL-spec drift) and return prioritized, evidence-backed findings and improvement opportunities, with investigation modes for errors, tests, dependencies, docs, performance, and security. Use when asked to analyze, audit, or review a whole project, find risks or gaps, or decide what to improve next.
license: Apache-2.0
---

# Analyze Project

## Todo List

1. **in progress:** State the analysis objective and scope; build the candidate-area inventory and list the readable inputs.
2. Investigate the in-scope areas and modes; record file or command evidence for every finding.
3. Prioritize the findings and draft the report with severity, evidence, locations, and a next-owner skill per recommendation.
4. Complete the list only when every finding has evidence and the report is handed off, or the remaining findings and risks are explained in the handoff.

Keep exactly one item in progress. Mark an item complete only after its stated evidence exists. Add or revise items when the agreed scope changes. Use the host's native Todo List when available; otherwise maintain this list as a Markdown checklist in the conversation.

## Workflow

Run the shared flow for every analysis; use investigation modes for domain targets.

### 1. Discover

- State the analysis objective and scope before investigating; record what is in and out of scope.
- Build an inventory of candidate areas from the repository layout, recent history, the issue tracker, and CI runs.
- List the inputs the analysis may read: repository paths, command output, logs, dependency manifests, and test or build results.

### 2. Investigate

Cover the candidate areas across these axes:

- **Architecture:** module boundaries, layering, component dependencies, configuration drift.
- **Tests:** suite structure, failure patterns, coverage signals, case-design gaps (see the tests mode).
- **CI/release:** workflow correctness, gate coverage, drift between documented and enforced checks.
- **Documentation drift:** command references, setup and release instructions, configuration examples, architecture claims, required outputs.
- **Technical debt:** dead code, TODO markers, duplicated logic, outdated patterns.
- **Dependencies:** declared versus locked versions, compatibility and license risks, vulnerability warnings from current tool output.
- **FSL-spec drift:** drift between formalized specifications and the workflows they describe.

Evidence rules:

- Every finding cites file or command evidence; a finding without evidence is not reported.
- Record the commands run and their resulting output or artifacts.
- Distinguish observed facts from inferences, and label inference as such.

Investigation modes: when the objective targets one domain, load that mode from [references/modes.md](references/modes.md) and run the shared flow inside it: errors, tests, dependencies, docs, performance, security. Multi-mode runs are allowed; the mode guardrails still bind every finding.

### 3. Validate findings

- Re-run cited commands when cheap and confirm cited paths exist; drop unverifiable findings or re-label them as inference.
- Check that every finding carries a severity, evidence, and location before it enters the report.

### 4. Prioritize

- Rank findings by impact and confidence; separate quick wins from structural issues.
- Group findings that share one root cause so recommendations stay minimal.

### 5. Report

Produce a report with:

- An executive summary of the objective, scope, and top outcomes.
- Findings, each with severity, evidence, and location.
- Prioritized recommendations, each naming its next-owner skill.
- A handoff section listing the commands run, skipped or blocked investigations, and open risks.

Change candidates are recommendations only; converting them into governed work items belongs to `create-issue`. The report never contains Issue-creation or Pull-request-creation instructions.

## Read-only guardrails

- Never edit files or run mutating commands.
- Never author issues or pull requests, and never document Issue-creation or Pull-request-creation rules.
- Stop and report when the objective or scope is unclear, or when an in-scope area cannot be investigated.
- The analysis ends at recommendations: refactoring, test-writing, and fixing belong to `debug-code`, `write-tests`, `refactor-code`, and the governed Change flow.
- Do not duplicate `review-pr` (per-pull-request review), `audit-workflow-enforcement` (enforcement audits), or release validation; the CI/release axis here is drift discovery only.

## Handoff

- Report the executive summary, the findings with severity and evidence and location, the prioritized recommendations with next-owner skills, the commands run with their results, and any skipped or blocked investigations.
- Hand change candidates to `create-issue`; only `create-issue` may turn candidates into governed work items. The handoff ends the analysis; the analysis never edits files or runs mutating commands.