---
name: analyze-codebase
description: Assess a codebase's structure, module boundaries, dependencies, tests, build or runtime entry points, and technical debt with read-only, evidence-backed findings. Use when asked to understand code, test organization, dependency relationships, or concrete codebase debt; use analyze-project for full project governance or release analysis.
license: Apache-2.0
---

# Analyze Codebase

## Todo List

1. **in progress:** State the assessment objective and scope; inventory the readable code, manifests, tests, and build or runtime inputs.
2. Assess structure, dependencies, tests, entry points, and technical debt; record file, symbol, command, or test evidence for every finding.
3. Validate the evidence, label inferences, and record unavailable checks or inputs as limitations.
4. Prioritize findings, name the next-owner skill for each recommendation, and hand off the report.

Keep exactly one item in progress. Mark an item complete only after its stated
evidence exists. Use the host's native Todo List or maintain the same list as a
Markdown checklist when no native list is available.

## Workflow

### 1. Discover

- State what the assessment must explain and what it excludes.
- Inventory source directories, module boundaries, dependency manifests and lockfiles, test directories, build scripts, runtime entry points, and recent relevant changes.
- List the inputs the assessment may read: repository paths, symbols, command output, logs, manifests, and test or build results.
- Separate a codebase request from a full project review. Route CI, release, documentation, Issue-tracker, license, or FSL drift to `analyze-project` unless a short reference is needed to explain a local code finding.

### 2. Investigate

Cover the areas that the objective includes:

- **Structure:** Map modules, public boundaries, ownership signals, and important call or import relationships. Cite paths and symbols.
- **Dependencies:** Compare declared and locked dependencies, identify relationships that affect build or runtime behavior, and run available dependency commands when they are safe. Do not claim a vulnerability or license risk without current command evidence.
- **Tests:** Map suite organization and runnable test commands. Report a gap only with a concrete missed scenario and explain why the current tests do not exercise it. Do not write tests.
- **Build and runtime:** Identify executable entry points, configuration inputs, generated code, and commands that establish whether the code can build or run. Do not change configuration or generated output.
- **Technical debt:** Report concrete dead code, duplicated logic, stale interfaces, TODO markers, or risky patterns only when a path, symbol, command, or test supports the claim.

Distinguish observed facts from inferences. Label an inference and keep it below
an observed finding unless later evidence confirms it.

### 3. Validate findings

- Re-run cited commands when cheap and confirm that cited paths and symbols still exist.
- Drop an unverifiable finding or label it as an inference with the missing evidence.
- Record failed, skipped, unavailable, or environment-dependent checks. An unrun check is not a passing result.
- Check that every retained finding has a severity, confidence, evidence location, and observable impact.

### 4. Prioritize

- Rank findings by impact and confidence.
- Separate quick, local improvements from structural risks.
- Group findings that share one cause, but keep separate next owners when the work belongs to different skills.
- Name the next-owner skill for every recommendation: `write-tests` for test additions, `debug-code` for reproduced failures, `refactor-code` for behavior-preserving cleanup, `analyze-project` for broader project analysis, or `create-issue` for governed change candidates.
- When a test failure is already reproduced, name `debug-code` as the next owner. Use `write-tests` only for a concrete missing test scenario that does not require fixing a reproduced failure.

### 5. Report

Return:

- An executive summary of the objective, scope, and highest-priority outcomes.
- Findings with severity, confidence, observed or inferred status, concrete impact, and file, symbol, command, or test evidence.
- Concrete missed test scenarios without writing tests or claiming coverage that was not measured.
- Prioritized recommendations with their next-owner skills.
- Commands run and their results, plus skipped or blocked investigations and their limitations.

## Read-only guardrails

- Never edit source, tests, configuration, generated files, or documentation.
- Never run mutating commands or create, close, edit, reprioritize, or link Issues or Pull Requests.
- Do not fix a finding. Reproduction and fixing belong to `debug-code`, test additions belong to `write-tests`, behavior-preserving cleanup belongs to `refactor-code`, and governed implementation belongs to the `create-issue` flow.
- Do not duplicate `analyze-project`'s full project investigation, `review-pr`'s Pull Request review, or `audit-workflow-enforcement`'s enforcement audit.
- Do not design a large-scale redesign. That work belongs to `propose-improvements` when it is available and separately requested.
- Stop and report when the objective, scope, or evidence needed for a finding is unavailable.

## Handoff

- Hand governed change candidates to `create-issue`; the report never creates the Issue or Pull Request.
- Keep local codebase findings separate from broader project-governance findings and name the owner for both.
- End with the report, evidence, validation results, limitations, and next-owner skills. Do not claim that the codebase is safe or complete from an incomplete inspection.

## Writing quality

Use plain, active, evidence-backed prose. Name the file, symbol, command, or
test behind every repository claim.
