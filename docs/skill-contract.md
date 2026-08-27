# Skill contract

Every skill states where its result goes and which skill owns the next phase. This document defines the shared handoff contract: the artifact flow from report to review, the debug and review closed loops, and the analyze-to-change boundary.

## Artifact flow

The governed change flow moves one artifact through owner skills:

report → issue → plan → implementation → pull request → review

| Stage | Owner skill | Result artifact | Handed to |
| --- | --- | --- | --- |
| Report | `analyze-*` (`analyze-project`) | Prioritized, evidence-backed findings and recommendations | `create-issue` |
| Issue | `create-issue` | Change or release Issue | `plan-issue` |
| Plan | `plan-issue` | Verified implementation plan posted as an Issue comment | `implement-issue` |
| Implementation | `implement-issue` | In-scope changes committed per task with evidence | `create-pr` |
| Pull request | `create-pr` | Issue-backed Pull Request | `review-pr` |
| Review | `review-pr` | Severity-ordered findings | `implement-issue` |

Rules:

- Each owner produces only its stage artifact and hands it to the next owner. No skill performs the next stage's work.
- A stage completes only when its artifact has observable evidence. Incomplete results return to their owner for rework.
- `create-issue` is the only skill that may create Issues. `analyze-*` reports recommendations; it never creates Issues or edits code.

## Ownership boundary

Every skill names its result, the next-owner skill, and what it must not do.

| Skill | Produces | Handoff target | Ownership boundary |
| --- | --- | --- | --- |
| `analyze-*` (`analyze-project`) | Prioritized findings report with evidence | `create-issue` | Read-only: recommends, never creates Issues, never edits code |
| `create-issue` | Compliant change or release Issue | `plan-issue` | Only Issue creator; does not plan or implement |
| `plan-issue` | Verified implementation plan (Issue comment) | `implement-issue` | Plans only; does not implement |
| `implement-issue` | In-scope edits committed per task with evidence | `create-pr` | Implements only in-scope files; does not publish a Pull Request |
| `create-pr` | Issue-backed Pull Request | `review-pr` | Opens and updates the Pull Request; does not merge or release |
| `review-pr` | Severity-ordered findings | `implement-issue` | Reviews; does not edit the branch or merge |
| `debug-code` | Reproduction, root cause, fix, and verification evidence | `write-tests` or `refactor-code`, then `implement-issue` | Fixes only the isolated bug; does not design tests or refactor |
| `write-tests` | Focused test cases with failure evidence | `implement-issue` | Tests only; does not fix production code |
| `refactor-code` | Behavior-preserving refactor verified against a test baseline | `implement-issue` | Refactors only; does not change behavior or add features |
| `bootstrap-project` | Runnable foundation with protected branch flow and FSL adoption plan | The project's governed change flow | Initialization only |
| `audit-workflow-enforcement` | Bounded subagent audit of enforcement rules | The requester and subsequent governed fixes | Audits only; fixes are governed changes |

Process-versus-fix ownership follows the criterion in
[skill-layers.md](skill-layers.md): `implement-issue` owns the implementation
stage of the artifact flow and is the process layer's only editor of target
source code, editing only in-scope files from a verified plan; fix-layer skills
edit in isolated, task-scoped work outside that stage and hand results into the
governed flow.

## Debug loop

`debug-code` owns the loop from reproduction to verification:

1. Reproduce the failure first and record the reproduction evidence.
2. Isolate and identify the root cause; do not propose a fix before the root cause has evidence.
3. Fix only the isolated defect.
4. Verify the fix against the reproduction and the project's validation.

Follow-ups:

- `write-tests` adds regression tests for the verified fix.
- `refactor-code` cleans up the root-cause area when the fix leaves related debt.
- When a fix is a governed change, the work enters the artifact flow at `create-issue` and continues through `plan-issue` and `implement-issue`.

## Review loop

`review-pr` reviews the Pull Request and returns severity-ordered findings.

1. `review-pr` returns findings to `implement-issue`.
2. `implement-issue` fixes the findings on the same Issue branch and re-runs validation.
3. The updated Pull Request is re-reviewed; the loop repeats until the findings are resolved.

The reviewer does not edit the branch; `implement-issue` owns the fixes.

## Analyze-to-change rule

- `analyze-*` skills investigate and recommend. Their report is the input to a candidate Issue, not an Issue itself.
- Only `create-issue` may turn a candidate into an Issue.
- `analyze-*` must not create Issues, document Issue-creation rules, or edit code.

## Outcome-oriented entry points

Entry points coordinate the artifact flow without changing stage ownership:

| Entry point | Outcome | Route |
| --- | --- | --- |
| `improve-project` | Improve a project | `analyze-project` → `create-issue` → `plan-issue` → `implement-issue` → `create-pr` → `review-pr` |
| `deliver-change` | Deliver an Issue-backed change | `plan-issue` → `implement-issue` → `create-pr` → `review-pr` |
| `resolve-defect` | Resolve a verified defect | `debug-code` → `write-tests` → `create-issue` → `plan-issue` → `implement-issue` → `create-pr` → `review-pr` |

Rules:

- Each phase still runs inside its owning primitive with that primitive's
  authority; the entry point only routes phases and tracks one user-visible
  progress model, then returns one cohesive final report.
- Approval boundaries and external mutation authority are unchanged: only
  `create-issue` and `create-pr` create external work items, and read-only
  phases stay read-only.
- Direct primitive invocation remains documented and functional; entry points
  are optional coordinators, not replacements.
- Entry points terminate their loops deterministically (bounded rework passes)
  and stop with actionable evidence when a handoff fails.
- Behavioral evaluation of entry-point routing and loop termination belongs to
  the behavioral evaluation system.
