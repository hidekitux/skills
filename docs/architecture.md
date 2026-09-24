# Architecture record

## Purpose

This document is the architecture record that Issue #326 requires. It states
which internal module owns each responsibility of the Go implementation, which
module may import which, and where a focused test replaces a module's input.
Issue #328 establishes the ownership model and the dependency direction. Issue
#329 adds the evidence data path and the typed domain result. Issue #330 adds
the provider module and the provider ownership section. Issues #331 through
#333 extend this document; the handoff section names the section each one
changes.

`workflow/module-ownership.yml` is the machine-readable form of the model below.
The `check-module-boundaries` repository check reads that file, resolves the
imports of every package below `internal/` including a nested one, and fails on
any module edge the file does not allow. The check also fails when a package
outside the `provider` module imports `os/exec` or `net/http`. The ownership
file lists only top-level directories; a nested package such as `support/util`
belongs to the module that owns its top-level directory.
`cmd/check-repository` runs the check, so `mise run check:repository` and
`mise run validate:all` enforce the direction.

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
| `provider` | Ports and adapters for every external process and network request. | `provider` |
| `domain` | Skill domain data read from the repository tree. | `graph`, `instructions` |
| `policy` | Policy decisions about deliberation, environment, and execution strategy. | `context`, `deliberation`, `environment`, `strategy` |
| `evidence` | Evidence and reporting artifacts. | `badges`, `diagnostic`, `evidence`, `publicstatus`, `replay`, `trace` |
| `execution` | Workflow execution and release assembly. | `eval`, `release` |
| `governance` | Repository checks and validation of committed artifacts. | `check`, `commitlint`, `fsl`, `govuln`, `project`, `validate` |
| `composition` | Command-line assembly. | `cmd/**` |

One module owns each of the five responsibilities Issue #326 names: `execution`
owns workflow execution, `policy` owns policy decisions, `provider` owns
external providers, `evidence` owns evidence and reporting, and `governance`
owns the tests that validate committed artifacts. The substitution-point table
below owns the test seams.

`internal/evidence` holds the primitives more than one evidence producer needs:
`DiagnosticRef`, `IsPrivateAddress`, and the credential, URL, and commit
identifier patterns. It imports no other internal package, so `internal/trace`
and `internal/diagnostic` read one definition of each rule rather than keeping
their own. A type whose field set differs between producers stays with its
producer: `Evidence`, `RedactionSummary`, and `ValidationReport` are declared
separately in `internal/trace` and `internal/diagnostic`, because merging them
would widen a persisted contract.

`internal/strategy`, `internal/environment`, and `internal/provider` declare
their own copies of the credential or commit identifier pattern.
`internal/strategy` and `internal/environment` belong to `policy`, which may not
import `evidence`. `internal/provider` cannot import `evidence` either: the
`evidence` module may import `policy`, and `policy` imports `provider`, so the
edge would make the module graph cyclic. Removing those copies needs a separate
decision about where the patterns belong.

## Evidence data path

One producer owns each persisted artifact, and one path reaches it.

| Artifact | Producer | Input | Consumers |
| --- | --- | --- | --- |
| Structured trace | `internal/trace` | `trace.RunResult` from `internal/eval` | `cmd/validate-skill-trace`, `cmd/read-skill-trace`, `internal/replay`, `internal/eval` metrics |
| Validator diagnostic | `internal/diagnostic` | `diagnostic.Diagnostic` from `internal/validate` and `internal/fsl` | `cmd/validate-diagnostic`, `cmd/check-repository` |
| Replay report | `internal/replay` | An ordered `replay.TraceSet` read from persisted traces | `cmd/replay-skill-trace`, `cmd/check-repository` |
| Evaluation record | `internal/eval` | `eval.Record` from a host run | `cmd/evaluate`, `cmd/check-evaluation` |

`trace.RunResult` is the typed domain result. It carries what an execution
module observes: the run identity, the timestamps, the typed `trace.RunOutcome`,
the correction attempts, and the observed handoff destination. `internal/trace`
decides the schema version, the event order, the terminal classification, and
the redaction summary in `trace.FromEvaluationRun`. `internal/eval` therefore
records what a run did without writing any persisted field itself.

The `evidence` module owns both the typed result and the conversion, but they
sit in `internal/trace` rather than in `internal/evidence`. `internal/trace`
imports `internal/evidence` for the shared primitives, and `RunResult` holds a
`*trace.Deliberation`, so a conversion declared in `internal/evidence` would
close an import cycle. Moving either one there needs `Deliberation` and every
type it reaches to move first.

`internal/trace/testdata/evaluation-run-traces.json` pins the trace that
conversion produces for every outcome. The file was captured from the assembly
`internal/eval` owned before Issue #329, so
`TestFromEvaluationRunReproducesTheRecordedTrace` fails when a field is added,
removed, or given a different value, and when the event order changes. The
comparison decodes and re-encodes both sides, which sorts the keys of a JSON
object, so it does not detect a different order of the fields inside one
object.

## Provider ownership

An external operation is a process the repository starts or a request it sends
over the network. `internal/provider` owns every one of them. A caller states
what it needs through a port, and one adapter performs it. `provider.Runner` is
the process port: `provider.OSRunner` starts a real process and `provider.Stub`
answers from recorded values without starting one. Every capability port below
is built from a Runner, so substituting the Runner substitutes every
process-backed provider at once.

| Port | Adapter | Operations | Authority required |
| --- | --- | --- | --- |
| `provider.Runner` | `OSRunner` | Any process | Whatever the command itself needs |
| `provider.Git` | `NewGit` | `git` reads and writes in a working tree | The local repository; git resolves its own remote credentials |
| `provider.GitHub` | `NewGitHub` | `gh`, including `gh skill install` and `gh skill publish` | The `gh` authentication already in the environment |
| `provider.GitHubAPI` | `NewGitHubAPI` | The pull-request commits endpoint | A token the caller passes per request |
| `provider.HostCLI` | `NewHostCLI` | One headless evaluation stage for `codex`, `claude-code`, `opencode`, or `antigravity` | The account or key the host CLI resolves |
| `provider.Mise` | `NewMise` | A `mise` task | None beyond the local toolchain |
| `provider.FSL` | `NewFSL` | The pinned `fslc` binary at an explicit path | None |
| `provider.Tool` | `NewTool` | `commitlint`, `govulncheck`, `uv`, `wt` | None beyond the local toolchain |
| `provider.Shell` | `NewShell` | One bounded `sh -c` command from a scenario file | The environment the caller passes |

The filesystem is not a port. Its adapter is the operating system, and a test
substitutes it by pointing the caller at a temporary root, the seam the
substitution table below records for every module. A Go interface would reach
about 150 call sites in 20 packages without changing what a test can already
do.

### Failure classification

Every adapter returns `*provider.Error` on failure. It carries the port, the
operation, the classification, the exit code, and a bounded detail with
credentials and private addresses removed. A nil error is the success
classification, and a product validation failure stays an ordinary error or a
report value, so `provider.KindOf` separates the two.

| Classification | Meaning | Source |
| --- | --- | --- |
| `failure` | The operation ran and returned a non-zero result | A process exit status, or an HTTP status of 400 or more |
| `timeout` | The operation exceeded the deadline the caller set | `Command.Timeout` expired |
| `interrupted` | The operation was cancelled or signalled before producing a result | A cancelled context, or a negative exit code |
| `unavailable` | The capability is absent, so the operation never started | An unresolved binary, or a process that returned no status |
| `retry_exhausted` | A caller that bounds its own attempts used the last one | `provider.RetryExhausted` |

No adapter retries. A retry stays with the caller that owns the attempt bound,
which is the behavior Issue #330 preserves rather than changes.

### Provider call map

Every external operation moved from its caller to a port. The `git` call in
`internal/support/support.go` is the one exception: repository-root resolution
needs it, and `foundation` cannot import `provider`, which imports `foundation`.
`check-module-boundaries` exempts that package by name and reports every other
process or request call outside `internal/provider`.

| Operation | Before | After |
| --- | --- | --- |
| `git` output for repository-root resolution | `support.GitOutputIn` | Unchanged, now unexported as `support.gitOutput` |
| `git` reads for whitespace, writing, and sensitive-content checks | `support.GitOutputIn` and `exec.Command` in `internal/check` | `check.gitPort` on `provider.Git` |
| `git` reads for changed FSL specifications | `exec.Command` in `internal/fsl/changed.go` | `fsl.gitPort` on `provider.Git` |
| `git` reads and `commitlint` for commit linting | `commitlint.commandRunner` | `provider.Git` and `provider.Tool` |
| `git` reads and `git ls-remote` for release verification | `release.commandRunner` | `release.releasePorts` on `provider.Git` |
| `git` provenance for an evaluation run | `support.GitOutputIn` in `internal/eval/run.go` | `eval.gitPort` on `provider.Git` |
| `git`, `wt`, and `bash` for worktree provisioning | `environment.CommandRunner` | `provider.Runner` through `environment.runCombined` |
| `gh` for Project reads and mutations | `exec.Command` in `internal/project/client.go` | `provider.GitHub` |
| `gh skill install` for host validation | `exec.Command` in `internal/validate/hosts.go` | `validate.githubPort` on `provider.GitHub` |
| `gh skill install` for an evaluation sandbox | `exec.CommandContext` in `internal/eval/host.go` | `provider.HostCLI` using `provider.GitHub` |
| `mise` tasks and `gh skill publish` for a release | `release.commandRunner` | `provider.Mise` and `provider.GitHub` |
| `fslc` check and verify | `exec.Command` in `internal/fsl/run.go` | `fsl.fslPort` on `provider.FSL` |
| `govulncheck` scan | `govuln.runner` | `provider.Tool` |
| `uv` for the skill-creator validator | `exec.Command` in `internal/validate/skillcreator.go` | `validate.toolPort` on `provider.Tool` |
| One headless host CLI stage | `eval.HostRunner` | `provider.HostCLI` |
| `sh -c` for an assertion command and a rubric reviewer | `exec.CommandContext` in `internal/eval` | `eval.shellPort` on `provider.Shell` |
| The pull-request commits endpoint | `net/http` in `internal/validate/prsignatures.go` | `validate.apiPort` on `provider.GitHubAPI` |

## Dependency direction

| Module | May import |
| --- | --- |
| `foundation` | Nothing. |
| `provider` | `foundation` |
| `domain` | `foundation` |
| `policy` | `foundation`, `provider`, `domain` |
| `evidence` | `foundation`, `domain`, `policy` |
| `execution` | `foundation`, `provider`, `domain`, `policy`, `evidence` |
| `governance` | `foundation`, `provider`, `domain`, `evidence` |
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
- `provider` imports only `foundation`, so an adapter cannot reach for a policy,
  a domain type, or an evidence type. `domain` and `evidence` do not import
  `provider`, because neither reads an external system.
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
| `provider` | `provider.Runner` replaced by `provider.Stub`. | `TestStubSubstitutesEveryProcessBackedPort` in `internal/provider/provider_test.go` |
| `domain` | Repository-root parameter pointed at a written fixture tree. | `TestValidateRejectsDanglingSkillDestination` in `internal/graph/graph_test.go` |
| `policy` | Policy file read from a temporary root. | `TestSelectsDeterministicallyForRepresentativeSignals` in `internal/strategy/strategy_test.go` |
| `evidence` | Fixture file under `workflow/trace-fixtures/`, or a `trace.RunResult` value passed to the conversion. | `TestSensitiveFixtureValuesNeverReachPersistedJSON` in `internal/trace/fixture_test.go`; `TestFromEvaluationRunReproducesTheRecordedTrace` in `internal/trace/run_result_test.go` |
| `execution` | Sandbox directory replaced by `t.TempDir()`. | `TestEvaluateAssertions` in `internal/eval/assert_test.go` |
| `governance` | Repository-root parameter plus the `out` and `errOut` writers. | `TestSensitiveContentRejectsATokenAndPrivateURL` in `internal/check/check_test.go` |
| `composition` | The `repoCheck` table passed to `run`. | `TestRunFailsAggregateAndNamesFailingCheck` in `cmd/check-repository/main_test.go` |

A module dependency is substituted by pointing the dependent at a different
repository root or fixture path. An external provider is substituted by
replacing a Go interface instead: a test assigns a port built on
`provider.Stub`, and no process starts, no credential is read, and no ambient
repository state is touched. `TestOSRunnerClassifiesEveryFailureKind` covers
failure, timeout, interruption, and unavailable capability against the real
adapter, and `TestErrorMessageNeverCarriesACredential` checks that a diagnostic
carries no credential.

## Handoff to the dependent Sub-issues

| Issue | Extends |
| --- | --- |
| #329 | Landed. Added `Evidence data path`, the `internal/evidence` package to `Module ownership`, and the typed-result seam to `Test substitution points`. |
| #330 | Landed. Added the `provider` module to `Module ownership` and `Dependency direction`, added `Provider ownership`, and replaced the provider note in `Test substitution points`. |
| #331 | Adds the approved contract decision table as a new section. |
| #332 | Adds the cutover runbook and recovery procedure as a new section. |
| #333 | Records the final validation results and reconciles every section with the shipped tree. |

Each Sub-issue changes `workflow/module-ownership.yml` and this document
together. A module split that only one of the two records is a drift the
`check-module-boundaries` check reports.
