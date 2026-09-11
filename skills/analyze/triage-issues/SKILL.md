---
name: triage-issues
description: Review an Issue backlog for duplicates, stale premises, completed work, dependencies, and the next useful order, then return a read-only evidence-backed triage report. Use when asked to review existing Issues, clean up backlog decisions, or choose the next Issue; use analyze-project for whole-project investigation.
license: Apache-2.0
---

# Triage Issues

## Todo List

1. **in progress:** State the triage objective and scope; inventory the readable Issues, repository paths, and related changes.
2. Gather current tracker and repository evidence for each candidate; record the command or artifact behind every claim.
3. Compare candidates for duplicate requirements, stale premises, completed work, explicit dependencies, and suggested sequencing.
4. Validate the proposed order, label inference and unavailable evidence, and prepare the report with an owner for every recommendation.
5. Complete the list only when the report and its evidence are handed off, or every blocked investigation is explained.

Keep exactly one item in progress. Complete an item only when its stated evidence
exists. Use the host's native task list when available; otherwise maintain an
equivalent Markdown checklist.

## Workflow

### 1. Define the review

- State the decision the requester needs, the Issue set under review, and the
  output boundary.
- Include only existing Issues and their related repository evidence. Keep new
  implementation, unrelated project audits, and external mutations out of the
  review.
- List the inputs that are available: Issue state, title, body, labels, Project
  fields, links, comments, repository files, commit history, merged changes,
  tests, and command output.
- If tracker or code access is unavailable, record that limitation before
  drawing conclusions.

### 2. Build the evidence record

For each candidate Issue, record:

- The current state and declared Project fields.
- The goal, scope, acceptance criteria, dependencies, and requested owner.
- Relevant paths, symbols, commits, merged Pull Requests, tests, or commands.
- The source and freshness of every observation.

Separate observed facts from inferences. Cite a file, command, Issue, Pull
Request, or other readable artifact for every finding. Do not treat Issue age,
an unchecked checklist item, a stale command, or an open state as proof by
itself.

### 3. Classify Issue relationships

- **Duplicate candidate:** Report only evidenced overlap in the goal, scope, or
  acceptance criteria. Preserve requirements that occur in only one Issue and
  name the proposed canonical Issue. Treat title or keyword similarity alone as
  an unconfirmed lead.
- **Completed candidate:** Require code or merged-change evidence that covers
  the requested behavior. Compare the evidence with the full scope and
  acceptance criteria, and list any uncovered requirement.
- **Stale premise:** Identify the specific premise that no longer matches
  current code, history, tracker state, or command behavior. Explain whether the
  Issue needs a scope correction, a completion check, or more evidence.
- **Explicit dependency:** Record a stated prerequisite, Issue link, blocking
  relationship, or repository dependency as explicit and preserve its source.
- **Suggested sequencing:** Keep impact, risk, effort, priority, and readiness
  recommendations separate from explicit dependencies. State the reason for the
  suggested order without presenting it as a tracker fact.

Do not collapse duplicate, completed, stale, or dependency findings into one
label when more than one applies. Keep each proposed action tied to its
evidence and name the responsible next-owner skill or role.

### 4. Prioritize and validate

- Rank the candidate work by impact, confidence, readiness, and dependency
  order. Respect explicit user priorities over an inferred ranking.
- Preserve unique requirements when suggesting consolidation. A canonical Issue
  must cover the retained requirements, and any missing requirement must remain
  visible as a follow-up.
- Re-run cheap evidence commands and confirm cited paths or change references
  before reporting a finding.
- Label an unverified conclusion as inference. Label unavailable evidence as a
  limitation. A report may recommend no action when the evidence does not
  support a change.

## Report

Return these sections:

- **Executive summary:** The decision, scope, top findings, and recommended
  next Issue.
- **Evidence boundary:** Readable sources, commands, unavailable inputs, and
  any freshness limit.
- **Issue assessment:** For each candidate, state the classification, evidence,
  confidence, unique requirements, and proposed action.
- **Dependency and order:** Separate explicit dependencies from suggested
  sequencing and explain the recommended order.
- **Handoff:** Name the next-owner skill or role for each action. Hand new work
  candidates to `create-issue`; hand ready governed work to the existing change
  flow. State that the triage run made no external changes.

Use severity only when it explains impact. Use confidence separately to show
how strongly the evidence supports the conclusion. Keep recommendations
actionable and avoid presenting a preference as a duplicate, completion, or
blocking dependency.

## Read-only boundary

- Never edit repository files, Issue bodies, Project fields, labels, links, or
  other tracker state during triage.
- Never run a command whose purpose is to mutate repository or external state.
- Do not implement a recommendation, write tests, create a branch, or open a
  Pull Request. Those actions belong to their owning workflows.
- Do not retain private transcripts or reproduce sensitive Issue content. Keep
  only the sanitized evidence needed to support the report.
- Stop when the objective, candidate set, or evidence boundary is unclear, and
  report the missing input instead of guessing.

## Handoff

The report is a read-only analysis artifact. Each recommendation names its
next owner, and each claim points to its evidence. Change candidates go to
`create-issue`; an existing governed change continues through its plan and
implementation owners. The handoff ends the triage run.

## Writing quality

Use plain, active, evidence-backed prose. Name the file, command, Issue, Pull
Request, or output behind every repository claim, and distinguish facts from
inferences.
