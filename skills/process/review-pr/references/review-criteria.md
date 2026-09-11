# Review criteria and their sources

`review-pr` judges findings against review criteria. This reference defines the
standardized criteria-file declaration and the resolution order.

## Standardized criteria file

- The standardized criteria-file name is `REVIEWING.md` at the repository root,
  next to `AGENTS.md`.
- A project declares its review criteria by adding `REVIEWING.md`. The file is
  free text with these conventions:
  - A short purpose statement scoping what this project wants reviews to focus on.
  - A prioritized list of review concerns that overrides or supplements the
    built-in baseline (for example, "security over correctness", "never suggest
    dependency changes", "require a test for every changed branch").
  - Explicit non-goals: concerns the project does not want reported.
  - Scope notes stating whether a concern is business-wide (all PRs) or limited to
    certain paths or systems.
- Keep the file short; it supplements, not replaces, the baseline. The project may
  reference additional policy documents by relative path.

## Resolution order

Resolve the applied criteria in this order, stopping at the first source that
decides a concern:

1. Explicit requester input: criteria the requester states in the review request.
2. The project criteria file (`REVIEWING.md`), when the repository declares it.
3. Criteria derived from the change: contract obligations implied by the PR body,
   the linked Issue, specifications, and existing code.
4. The built-in baseline: bugs, regressions, and missing tests over style, with
   severity ordered by impact and blast radius.

Ask the requester when criteria are ambiguous or conflict between sources instead
of silently choosing.

## Reporting

- List every applied criterion and its source at the top of the review report,
  for example `Built-in baseline (built-in)`, `Guardrails from REVIEWING.md
  (repository criteria file)`, or `PR requirement aligned to Issue #123
  (requester input)`.
- When a candidate crosses a criterion boundary, record the applicable one; do not
  report the same finding under multiple standards.
