# Skill contract

Every skill states where its result goes and which skill owns the next phase. This document defines the shared handoff contract: the artifact flow from report to review, the debug and review closed loops, and the analyze-to-change boundary.

## Artifact flow

The governed change flow moves one artifact through owner skills:

report → issue → plan → implementation → pull request → review → fix → merge

| Stage | Owner skill | Result artifact | Handed to |
| --- | --- | --- | --- |
| Report | `analyze-*` (`analyze-project`) | Prioritized, evidence-backed findings and recommendations | `create-issue` |
| Issue | `create-issue` | Problem statement in a Change or release Issue | `plan-issue` |
| Plan | `plan-issue` | Investigated cause and chosen approach in a verified implementation plan posted as an Issue comment | `implement-issue` |
| Implementation | `implement-issue` | In-scope changes committed per task with evidence | `create-pr` |
| Pull request | `create-pr` | Issue-backed Pull Request | `review-pr` |
| Review | `review-pr` | Severity-ordered findings | `fix-pr` |
| Fix | `fix-pr` | Fixed, validated commits pushed as the Pull Request head | `review-pr`, then `merge-pr` |
| Merge | `merge-pr` | Merged Pull Request and linked-work outcome evidence | Post-merge verification or release publication |

Rules:

- Each owner produces only its stage artifact and hands it to the next owner. No skill performs the next stage's work.
- A stage completes only when its artifact has observable evidence. Incomplete results return to their owner for rework.
- `create-issue` is the only skill that may create Issues. `analyze-*` reports recommendations; it never creates Issues or edits code.

## Ownership boundary

Every skill names its result, the next-owner skill, and what it must not do.

| Skill | Produces | Handoff target | Ownership boundary |
| --- | --- | --- | --- |
| `analyze-*` (`analyze-project`) | Prioritized findings report with evidence | `create-issue` | Read-only: recommends, never creates Issues, never edits code |
| `create-issue` | Problem statement in a compliant change or release Issue | `plan-issue` | Only Issue creator; records the problem and boundaries, but does not investigate the cause, choose an approach, or implement |
| `plan-issue` | Verified implementation plan with investigated cause and resolved approach (Issue comment) | `implement-issue` | Investigates premises and surfaces decisions; plans only and does not implement |
| `implement-issue` | In-scope edits committed per task with evidence | `create-pr` | Implements only in-scope files; does not publish a Pull Request |
| `create-pr` | Issue-backed Pull Request | `review-pr` | Opens and updates the Pull Request; does not merge or release |
| `review-pr` | Severity-ordered findings | `fix-pr` | Reviews; does not edit the branch or merge |
| `fix-pr` | Fixed, validated commits pushed as the Pull Request head, with the body synced | `review-pr` | Fixes an open Pull Request from review findings; does not open the first Pull Request, merge, or release |
| `merge-pr` | Merged Pull Request and linked-work outcome evidence | Post-merge verification or release publication | Merges only after review and required checks; may resolve narrowly scoped conflicts after explicit authorization, but does not apply substantive fixes or publish releases |
| `debug-code` | Reproduction, root cause, fix, and verification evidence | `write-tests` or `refactor-code`, then `implement-issue` | Fixes only the isolated bug; does not design tests or refactor |
| `write-tests` | Focused test cases with failure evidence | `implement-issue` | Tests only; does not fix production code |
| `refactor-code` | Behavior-preserving refactor verified against a test baseline | `implement-issue` | Refactors only; does not change behavior or add features |
| `bootstrap-project` | Runnable foundation with protected branch flow and FSL adoption plan | The project's governed change flow | Initialization only |
| `audit-workflow-enforcement` | Bounded subagent audit of enforcement rules | The requester and subsequent governed fixes | Audits only; fixes are governed changes |

Process-versus-fix ownership follows the criterion in
[skill-layers.md](skill-layers.md): `implement-issue` owns the implementation
stage of the artifact flow and `fix-pr` owns the post-review stage, and they are
the process layer's only editors of target source code. Which of the two applies
is decided by one observable condition: when the branch has no open Pull
Request, `implement-issue` then `create-pr` apply; when one exists, `fix-pr`
applies. `implement-issue` edits only in-scope files from a verified plan and
`fix-pr` edits only what a review finding or that same boundary justifies;
fix-layer skills edit in isolated, task-scoped work outside both stages and hand
results into the governed flow.

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

1. `review-pr` returns findings to `fix-pr`.
2. `fix-pr` fixes the findings on the same Issue branch, tidies unpushed
   commits, re-runs validation, pushes, and syncs the Pull Request body in one
   invocation.
3. The updated Pull Request is re-reviewed; the loop repeats until the findings are resolved.

The reviewer does not edit the branch; `fix-pr` owns the fixes. Because `fix-pr`
also owns pushing, the commit the next review examines is always the Pull
Request head: the loop cannot leave a fix committed but unpublished, which is
the state in which `merge-pr` blocks.

`implement-issue` is not part of this loop. It derives its tasks from the
Issue's `Scope` and `Acceptance criteria`, which is the wrong task source for
review findings, and it does not publish.

## Analyze-to-change rule

- `analyze-*` skills investigate and recommend. Their report is the input to a candidate Issue, not an Issue itself.
- Only `create-issue` may turn a candidate into an Issue.
- `analyze-*` must not create Issues, document Issue-creation rules, or edit code.

## Outcome-oriented entry points

Entry points coordinate the artifact flow without changing stage ownership:

| Entry point | Outcome | Route |
| --- | --- | --- |
| `improve-project` | Improve a project | `analyze-project` → `create-issue` → `plan-issue` → `implement-issue` → `create-pr` → `review-pr` → `fix-pr` |
| `deliver-change` | Deliver an Issue-backed change | `plan-issue` → `implement-issue` → `create-pr` → `review-pr` → `fix-pr` |
| `resolve-defect` | Resolve a verified defect | `debug-code` → `write-tests` → `create-issue` → `plan-issue` → `implement-issue` → `create-pr` → `review-pr` → `fix-pr` |

Rules:

- Each phase still runs inside its owning primitive with that primitive's
  authority; the entry point only routes phases and tracks one user-visible
  progress model, then returns one cohesive final report.
- Approval boundaries and external mutation authority are unchanged: only
  `create-issue` and `create-pr` create external work items, `fix-pr` updates
  an existing Pull Request but never creates one, and read-only phases stay
  read-only.
- Direct primitive invocation remains documented and functional; entry points
  are optional coordinators, not replacements.
- Entry points terminate their loops deterministically (bounded rework passes)
  and stop with actionable evidence when a handoff fails.
- Behavioral evaluation of entry-point routing and loop termination belongs to
  the behavioral evaluation system.
