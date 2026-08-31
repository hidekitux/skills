---
name: bootstrap-project
description: Initialize or assess a software project with a minimal runnable foundation, protected GitHub pull-request workflow, proportionate validation, and an FSL adoption plan. Use when asked to start a project, scaffold a prototype, establish project conventions, require PRs and prevent direct pushes to protected branches, or introduce FSL to a new or existing codebase.
license: Apache-2.0
---

# Project Bootstrap

## Overview

Create only the smallest project foundation justified by the request. Establish how
the project is run and verified, then decide deliberately whether and how FSL
should document or verify its stateful workflows.

## Workflow

### 0. Create and maintain the Todo List

- Before inspecting or changing the project, create a Todo List. Include
  **establish boundary**, **build foundation**, **decide FSL adoption**,
  **verify**, and **handoff**; remove only items confirmed as not applicable.
- Use the active host's native Todo List or task-tracking capability. If none is
  available, show and maintain an equivalent Markdown checklist in the
  conversation.
- Keep exactly one item in progress. Mark each item complete only when its
  required evidence exists. Update the list whenever the user changes scope or
  discovery changes the plan.
- Before handoff, resolve every item or explain why it remains open. Keep the
  handoff report aligned with the final Todo List.

### 1. Establish the project boundary

- Inspect the existing repository before adding files. Identify its language,
  package manager, existing commands, conventions, and uncommitted work.
- Inspect `mise.toml` first when it exists. Treat mise as the project command
  entry point rather than relying on globally installed tools or ad-hoc commands.
- When the active host has material capability differences, read its note in
  `references/hosts/` before selecting tools or editing host-specific files.
- For a new project, confirm the intended users, primary command or interface,
  target runtime, and acceptance criteria. Ask only for choices that materially
  change the architecture.
- State the selected scope and assumptions before generating a non-trivial
  scaffold. Do not silently choose a framework, service, or deployment target.

### 2. Build a minimal, runnable foundation

- Prefer the smallest conventional structure for the chosen ecosystem.
- Add a clear entry point, dependency manifest or lockfile when applicable, and
  one representative test or verification command.
- Establish Conventional Commits for the project. Read
  [the Conventional Commits guide](references/conventional-commits.md), add the
  policy to contributor documentation, and add automated commit and Pull Request
  title checks when the project has a CI provider. Treat branch protection as
  part of enforcement and report it separately from a passing workflow.
- Use shared Issue and Pull Request titles in the form `[Type]: Summary` in
  sentence case. Define a small common Type list and require Summary to begin
  with a capitalized imperative verb; capitalize later words only when ordinary
  English requires it, such as for proper nouns or abbreviations, and do not
  maintain a finite verb list. Permit the explicit release exception
  `[Release]: vX.Y.Z` and build identifiers `[Release]: vX.Y.Z+N`. Validate
  both Issue and Pull Request titles in CI. Copy the shared validator from
  `templates/github/scripts/validate-work-item-title.py` into the generated
  repository's `scripts/`, and install
  `templates/github/work-item-title.yml` and
  `templates/github/issue-title-policy.yml` under `.github/workflows/`. Do not
  require a PR title to equal any linked Issue because a PR may close more than
  one Issue.
- Require an Issue number at the end of the commit header when the project
  uses issue-based delivery: `type(scope): summary #<number>`. Keep the number
  as the governing Issue for that commit; one Pull Request may handle multiple
  Issues, so the number need not match the branch or another Issue listed in
  the Pull Request. Require a single-sentence message and enforce the shape
  with commitlint; copy the shipped validator template
  `templates/github/scripts/validate-commit-message.py` into the generated
  repository's `scripts/lint/validate-commit-message.py` and run it in parallel
  with commitlint. Document the format in the contributor guide.
- Provision Change and Release Issue templates. Require exactly one `Context`,
  `Goal`, `Scope`, `Acceptance criteria`, and `Validation` heading in that
  order. Define each section's purpose, require ordered `In`/`Out` scope
  markers and actionable checklists, and reject empty sections. Follow the
  common Release headings with `Changelog` and exactly one ordered
  `Added`/`Changed`/`Fixed`/`Removed` heading. Define public releases as
  `vX.Y.Z` and build identifiers as `vX.Y.Z+N`.
- When the project is hosted on GitHub and the user wants PR-only protected
  branches, first create and run its CI workflows. Identify their completed job
  names, then use `scripts/configure-github-ruleset.py` with an explicit
  `--repo`, every protected `--branch`, and every `--required-check`.
  The script prints its payload unless `--apply` is supplied, creates or updates
  its named ruleset through `gh api`, and reads it back to verify the result.
  Require explicit user authorization immediately before `--apply`; it changes
  live repository policy. The default target is `main`; add every additional
  protected branch explicitly with `--branch`. The script also enables rebase merging
  only, disables merge-commit and squash merges, and automatically deletes merged head
  branches. Include `Validate signed pull-request commits` with every
  `--required-check`. Use one approval, stale-approval dismissal, last-push
  approval, resolved conversations, rebase merging, linear history, no force
  pushes or deletions, and no bypass actors. Add a `pull_request_target`
  workflow that checks every PR-head commit through GitHub's API and runs the
  verifier from the base revision; do not run PR-provided verification code with
  the token. The script removes its deprecated `issue/*` signature Ruleset when
  applied, because GitHub rechecks reachable base commits when an Issue branch
  is updated. Use
  `--allow-last-push-approval` and `--approvals 0` only for a confirmed solo
  workflow. Add `--require-code-owner-review` only after creating `CODEOWNERS`.
  Do not add a restrictive `update` rule: the pull-request rule is what forbids
  direct pushes while allowing GitHub to merge accepted pull requests.
- For issue-based delivery, use `issue/<number>` for human work branches. Make
  `## Issue` the first Pull Request body section and put only a contiguous block
  of standalone Issue references immediately below it. Require the first
  `Closes #<number>` reference to match the Issue number in the human work
  branch, allow one following `Closes` line per additional Issue handled by the
  same Pull Request, and reject references outside that section. Use
  `Tracks #<number>` only for release work that closes after publication.
  Keep the commit header suffix `#<number>` as the governing Issue for that
  commit so every commit traces to an Issue without requiring the suffix to
  match the work branch number.
  Protect `main` and any
  explicitly configured integration or release branches, then validate PR
  direction in CI. Default flow: `issue/* -> main`; document every automated
  exception explicitly. See [branch policy](references/branch-policy.md).
- Synchronize an Issue branch by rebasing it onto the latest target branch.
  Push the rewritten author-owned branch only with `--force-with-lease`; never
  use plain `--force`. Protected branches must never be force-pushed.
- Add or update `mise.toml`. Pin only the tools the project actually needs and
  expose applicable `format:all`, `lint:all`, `test:all`, and `check:all` tasks. Make `check:all`
  compose the relevant validations rather than duplicating their commands.
- Document only the commands a contributor needs to run, test, and understand
  the project. Document `mise run <task>` as the standard invocation. Do not add
  aspirational infrastructure or unused tooling.
- Execute the relevant commands and repair failures before handoff.

### 3. Decide the FSL adoption level

- Read [the FSL adoption guide](references/fsl-adoption.md) when the project has
  business rules, lifecycles, approvals, retries, permissions, queues, or other
  stateful behavior.
- Classify each candidate flow as **in scope now**, **thin/low priority**, or
  **out of scope**. Do not equate low priority with out of scope.
- If FSL fits, add only the verified foundation justified by confirmed rules:
  the `specs/` location, a `mise run verify:fsl` task wrapping repeatable `fslc`
  checks, and one
  prioritized flow. Do not create empty or invented `.fsl` specifications.
- Before authoring a `.fsl` file, use an appropriate FSL authoring skill when
  one is available. Otherwise, follow the formalization memo procedure in
  [the FSL adoption guide](references/fsl-adoption.md). In either case, present
  the memo for human confirmation, keep it in the conversation, and retain
  confirmed assumptions as comments or tags in the spec.
- Use a single spec for an isolated hard problem. Create business, requirements,
  and design layers only when those real layers exist and their alignment is the
  value being verified.

### 4. Verify and hand off

- Run the project's narrowest relevant checks through `mise run`. Report commands, outcomes, and
  anything not verified.
- When FSL is in scope, run `fslc check` and bounded `fslc verify` for each
  changed specification; use induction and mutation testing where the workflow's
  risk justifies them.
- Hand off a concise inventory: created or changed files, run/test commands,
  Conventional Commits policy and enforcement evidence, GitHub Ruleset ID and
  verification result when applicable, FSL decision and candidate flows,
  confirmed assumptions, and open questions.

## Guardrails

- Do not present FSL verification as proof that the implementation conforms to a
  specification. Use generated tests, adapters, or replay evidence when
  implementation conformance is required.
- Do not force FSL onto continuous values, probability, wall-clock behavior, or
  free-text semantics. Use ordinary tests or another suitable method instead.
- Do not weaken an accepted requirement merely to make a specification pass.
- Preserve existing project conventions unless the request explicitly changes
  them.
