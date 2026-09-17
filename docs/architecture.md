# Architecture record

## Purpose

This document is the architecture record that Issue #326 requires. It states
which internal module owns each responsibility of the Go implementation, which
module may import which, and where a focused test replaces a module's input.
Issue #328 establishes the ownership model and the dependency direction. Issues
#329 through #333 extend this document; the handoff section names the section
each one changes.

`workflow/module-ownership.yml` is the machine-readable form of the model below.
The `check-module-boundaries` repository check reads that file, resolves the
imports of every package below `internal/` including a nested one, and fails on
any module edge the file does not allow. The ownership file lists only top-level
directories; a nested package such as `support/util` belongs to the module that
owns its top-level directory. `cmd/check-repository` runs the check, so
`mise run check:repository` and `mise run validate:all` enforce the direction.

## Measured baseline

The measurements below come from commit `4cce0641bbc9bc28c9bba47522a6071b0acded69`.

- `go list ./...` returns 68 packages and reports no import cycle. The command
  needs a writable `GOCACHE`; the default local cache is not writable and fails
  with an environment permission error rather than a package error.
- `go list -f '{{.ImportPath}}:{{join .Imports ","}}' ./internal/...` shows the
  concentrated dependencies Issue #328 cites: `internal/eval` imports `context`,
  `instructions`, `strategy`, `support`, and `trace`; `internal/trace` imports
  `context`, `environment`, `graph`, and `strategy`; `internal/strategy` imports
  `deliberation`, `environment`, and `graph`.
- The 22 packages under `internal/` produce 29 module edges. Every one of them
  is allowed by the model below, so recording the direction removed no edge and
  retained all 29.
- `cmd/check-repository` aggregates 22 repository checks and imports 12 internal
  packages, the widest import set in the repository.

The problem the model solves is an unrecorded and unenforced direction, not a
broken import graph.

## Module ownership

| Module | Owns | Packages |
| --- | --- | --- |
| `foundation` | Shared primitives and skill discovery. | `discover`, `hooks`, `support` |
| `domain` | Skill domain data read from the repository tree. | `graph`, `instructions` |
| `policy` | Policy decisions about deliberation, environment, and execution strategy. | `context`, `deliberation`, `environment`, `strategy` |
| `evidence` | Evidence and reporting artifacts. | `badges`, `diagnostic`, `publicstatus`, `replay`, `trace` |
| `execution` | Workflow execution and release assembly. | `eval`, `release` |
| `governance` | Repository checks and validation of committed artifacts. | `check`, `commitlint`, `fsl`, `govuln`, `project`, `validate` |
| `composition` | Command-line assembly. | `cmd/**` |

One module owns each of the five responsibilities Issue #326 names: `execution`
owns workflow execution, `policy` owns policy decisions, `evidence` owns
evidence and reporting, `governance` owns the tests that validate committed
artifacts, and the substitution-point table below owns the test seams. No module
owns providers yet; Issue #330 introduces them.

## Dependency direction

| Module | May import |
| --- | --- |
| `foundation` | Nothing. |
| `domain` | `foundation` |
| `policy` | `foundation`, `domain` |
| `evidence` | `foundation`, `domain`, `policy` |
| `execution` | `foundation`, `domain`, `policy`, `evidence` |
| `governance` | `foundation`, `domain`, `evidence` |
| `composition` | Every module. |

An import inside one module is always allowed. Every module edge the table omits
is a forbidden reverse dependency. The rule covers these cases in particular:

- `foundation` imports no other module, so a shared helper cannot reach for a
  policy or an evidence type.
- `domain`, `policy`, and `evidence` never import `execution` or `governance`,
  so a change to workflow execution or to a repository check cannot propagate
  downward.
- `governance` never imports `execution` or `policy`, so a repository check
  validates a committed artifact rather than re-running an execution decision.
- No module imports `composition`. `cmd/**` assembles the program and is
  imported by nothing. The check reports an import of
  `github.com/hidekitux/skills/cmd/` from any internal package as a forbidden
  reverse dependency.

`check-module-boundaries` reads imports with `go/parser` rather than building the
packages. An import in a file the parser skips for a build tag is therefore not
observed. The check also ignores `_test.go` files, so a test import is not a
module edge; this keeps a test-only package such as `internal/hooks` free to
exercise any package it validates.

## Test substitution points

A focused test replaces a module's input at one of three seams. All three exist
in the code today; this table records which seam each module uses and cites one
test that uses it.

| Module | Substitution point | Test that uses it |
| --- | --- | --- |
| `foundation` | Repository-root parameter replaced by `t.TempDir()`. | `TestResolveRoot` in `internal/support/support_test.go` |
| `domain` | Repository-root parameter pointed at a written fixture tree. | `TestValidateRejectsDanglingSkillDestination` in `internal/graph/graph_test.go` |
| `policy` | Policy file read from a temporary root. | `TestSelectsDeterministicallyForRepresentativeSignals` in `internal/strategy/strategy_test.go` |
| `evidence` | Fixture file under `workflow/trace-fixtures/`. | `TestSensitiveFixtureValuesNeverReachPersistedJSON` in `internal/trace/fixture_test.go` |
| `execution` | Sandbox directory replaced by `t.TempDir()`. | `TestEvaluateAssertions` in `internal/eval/assert_test.go` |
| `governance` | Repository-root parameter plus the `out` and `errOut` writers. | `TestSensitiveContentRejectsATokenAndPrivateURL` in `internal/check/check_test.go` |
| `composition` | The `repoCheck` table passed to `run`. | `TestRunFailsAggregateAndNamesFailingCheck` in `cmd/check-repository/main_test.go` |

A module dependency is substituted by pointing the dependent at a different
repository root or fixture path, not by replacing a Go interface. Issue #330
introduces interfaces for external providers; until it lands, a provider call
stays where it is and is substituted through the same root or fixture seam.

## Handoff to the dependent Sub-issues

| Issue | Extends |
| --- | --- |
| #329 | Adds the typed evidence boundary to `Module ownership` and `Test substitution points` for the `evidence` module. |
| #330 | Adds a `provider` module to `Module ownership` and `Dependency direction`, and replaces the provider note in `Test substitution points`. |
| #331 | Adds the approved contract decision table as a new section. |
| #332 | Adds the cutover runbook and recovery procedure as a new section. |
| #333 | Records the final validation results and reconciles every section with the shipped tree. |

Each Sub-issue changes `workflow/module-ownership.yml` and this document
together. A module split that only one of the two records is a drift the
`check-module-boundaries` check reports.
