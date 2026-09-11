---
name: retrospect-work
description: Review a completed or interrupted development session for recurring workarounds, unresolved causes, duplicate follow-ups, and evidence gaps, then return a read-only recommendation for durable next work. Use when asked to review what happened during a task or retain a workaround; use analyze-project for whole-project investigation and triage-issues for backlog ordering.
license: Apache-2.0
---

# Retrospect work

## Todo List

1. **in progress:** State the retrospective objective and boundary; inventory the readable session, repository, tracker, and evaluation evidence.
2. Gather sanitized evidence for changes, failures, workarounds, retries, corrections, unresolved causes, and existing follow-up work.
3. Classify each observation as an observed problem, hypothesis, transient condition, tracked work, no finding, or unavailable evidence.
4. Connect repeated repository-owned failures to the failure promotion process and name the next responsible owner and recurrence check.
5. Complete the list only when the report and evidence are handed off, or every evidence limitation and absence of warranted follow-up is explained.

Keep exactly one item in progress. Complete an item only when its stated evidence
exists. Use the host's native task list when available; otherwise maintain an
equivalent Markdown checklist.

## Workflow

### 1. Define the review

- State whether the requester wants a review of a finished task, an interrupted
  task, a workaround, or a suspected recurring failure.
- Define the session boundary and list the inputs that are available: user
  summary, structured trace, repository status or diff, command output, Issue
  or Pull Request references, failure records, evaluation results, and logs.
- Keep the review read-only. Do not infer inaccessible session history, and do
  not retain or reproduce private transcripts.

### 2. Build the evidence record

For each observation, record:

- The source file, command, trace event, Issue, Pull Request, or user-provided
  fact and its freshness.
- The expected result and observed result, when both are available.
- The workaround, retry, user correction, or manual step that allowed progress.
- The remaining cause, uncertainty, and whether the evidence supports one
  observation or recurrence.
- Any sensitive content that must be omitted from the report.

Separate observed facts from hypotheses. Sanitize paths, inputs, logs, and
transcripts so the report keeps only the evidence needed to support a decision.

### 3. Classify and connect findings

- **Observed problem:** The session contains a concrete failure or undesirable
  behavior with a source and an observable effect.
- **Hypothesis:** A possible cause lacks enough evidence to claim a defect.
  Keep it as an investigation question with the evidence needed to test it.
- **Transient condition:** An external service, permission, dependency, or
  infrastructure condition interrupted the work. Do not turn it into a
  repository rule without evidence of a repository-owned cause.
- **Tracked work:** An existing Issue, Pull Request, failure record, or
  evaluation already owns the remaining work. Reference it and preserve any
  requirement it does not cover.
- **No finding:** The workaround was expected, the cause was resolved, or the
  evidence supports no durable improvement.
- **Unavailable evidence:** The session history or source needed for a claim
  cannot be read. State the limitation instead of filling the gap.

When a repository-owned failure recurs, use
`docs/failure-promotion.md` for the reproduction threshold, cause class,
lowest sufficient enforcement layer, regression asset, and owner. Do not copy
its record schema or evaluation infrastructure into this workflow. Recommend
`create-issue` only when the evidence supports durable repository work and no
existing work covers it. Keep duplicate candidates tied to the existing work.

### 4. Validate the recommendation

- Check the cited path, command, or work item again before reporting it.
- Distinguish one failed attempt from recurrence. Two minimal reproductions or
  one deterministic failure with a clear expected result support promotion.
- Name the workaround, remaining cause, next responsible owner, and the test,
  evaluation, or observation that can detect recurrence for every actionable
  finding.
- Report that no follow-up is warranted when the evidence supports no change.

## Report

Return these sections:

- **Executive summary:** The session outcome and whether durable follow-up is
  warranted.
- **Evidence boundary:** Readable sources, commands, freshness limits, and
  unavailable evidence.
- **Findings:** For each item, state the classification, evidence, workaround,
  remaining cause, confidence, next owner, and recurrence detection.
- **Existing work and duplicates:** Name tracked work and explain any
  requirements it does not cover; do not create or edit it.
- **Handoff:** Route a supported repository change to `create-issue`, an
  existing governed change to its current owner, or state that no handoff is
  warranted. State that the retrospective made no external changes.

## Read-only boundary

- Never edit repository files, memory, Issues, Pull Requests, Projects, labels,
  links, or evaluation records during a retrospective.
- Never create an Issue, branch, Pull Request, failure record, or regression
  test. Those actions belong to their owning workflows.
- Do not copy credentials, tokens, private URLs, personal data, or private
  transcript content into the report.
- Stop when the objective or evidence boundary is unclear and report the
  missing input.

## Handoff

The retrospective produces a sanitized, evidence-backed report. A supported
repository change goes to `create-issue`; an existing governed change returns
to its current owner; a transient condition, duplicate, or resolved session
may end with no repository handoff. The retrospective itself makes no
external change.

## Writing quality

Use plain, active, evidence-backed prose. Name the file, command, trace,
Issue, Pull Request, or output behind every repository claim, and distinguish
facts from hypotheses.
