# Architecture record

## Purpose

This document is the architecture record that Issue #326 requires. It states
which internal module owns each responsibility of the Go implementation, which
module may import which, and where a focused test replaces a module's input.
Issue #328 establishes the ownership model and the dependency direction. Issue
#329 adds the evidence data path and the typed domain result. Issue #330 adds
the provider module and the provider ownership section. Issue #331 adds the
contract decisions and Issue #332 adds the cutover and recovery procedure.
Issue #333 adds `Final validation` and the remaining risks, and it reconciles
every section above with the shipped tree. The handoff section names the
section each Sub-issue changes.

`workflow/module-ownership.yml` is the machine-readable form of the model below.
The `check-module-boundaries` repository check reads that file, resolves the
imports of every package below `internal/` including a nested one, and fails on
any module edge the file does not allow. The check also reads the
`Module ownership` table below and compares it with the file in both
directions: it fails when the file declares a module or a package the table
does not record, and when the table records a module or a package the file
does not declare. It also fails when a package outside the `provider`
module imports `os/exec` or `net/http`; that rule covers every package below
`internal/` and every package below `cmd/`. The ownership
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
- `cmd/check-repository` aggregates 21 repository checks and imports 12 internal
  packages, the widest import set in the repository.

The problem the model solves is an unrecorded and unenforced direction, not a
broken import graph.

## Module ownership

| Module | Owns | Packages |
| --- | --- | --- |
| `foundation` | Shared primitives and skill discovery. | `discover`, `hooks`, `redact`, `support` |
| `provider` | Ports and adapters for every external process and network request. | `provider` |
| `domain` | Skill domain data read from the repository tree. | `graph`, `instructions` |
| `policy` | Policy decisions about deliberation, environment, and execution strategy. | `context`, `deliberation`, `environment`, `strategy` |
| `evidence` | Evidence and reporting artifacts. | `badges`, `diagnostic`, `evidence`, `publicstatus`, `replay`, `trace` |
| `execution` | Workflow execution and release assembly. | `eval`, `release` |
| `governance` | Repository checks and validation of committed artifacts. | `check`, `commitlint`, `fsl`, `govuln`, `project`, `validate` |
| `composition` | Command-line assembly. | `cmd/**` |

`composition` is the composition root rather than a module of the ownership
file. `cmd/**` is not a directory below `internal/`, so
`workflow/module-ownership.yml` cannot own it and declares seven modules while
this table has eight rows. Every count of modules elsewhere in this document is
that seven. The check compares the other seven rows with the file and skips
this one by name.

One module owns each of the five responsibilities Issue #326 names: `execution`
owns workflow execution, `policy` owns policy decisions, `provider` owns
external providers, `evidence` owns evidence and reporting, and `governance`
owns the tests that validate committed artifacts. The substitution-point table
below owns the test seams.

`internal/evidence` holds one primitive: `DiagnosticRef`. `internal/trace` and
`internal/diagnostic` both alias it, which is why it sits in a shared package
rather than with either producer. A type whose field set differs between
producers stays with its producer: `Evidence`, `RedactionSummary`, and
`ValidationReport` are declared separately in `internal/trace` and
`internal/diagnostic`, because merging them would widen a persisted contract.

`internal/redact` holds the rules a package applies before it persists or
reports a value: `CredentialPattern`, `URLPattern`, `CommitSHAPattern`, and
`IsPrivateHost`. `internal/trace`, `internal/diagnostic`, `internal/strategy`,
`internal/provider`, and `internal/check` call them, so one definition decides
every answer.

The rules live in `foundation` rather than in `evidence` because two of those
callers cannot import the `evidence` module. `internal/strategy` belongs to
`policy`, which may not import `evidence`. `internal/provider` cannot import
`evidence` either: the `evidence` module may import `policy`, and `policy`
imports `provider`, so the edge would make the module graph cyclic. Every
module may import `foundation`, so placing the rules there removed the copies
without changing a single `may_import` list.

`IsPrivateHost` decides one question for every caller. It reports a loopback
address, an IPv4 private range, a unique local IPv6 address, a link local
unicast address, the name `localhost`, and any name ending in `.local` as
private. `internal/check` asks the same function, so the `private network URL`
rule of `check-sensitive-content` and the redaction rule of `internal/provider`
cannot disagree about a host.
Before Issue #343 the same question had two answers: `internal/provider`
matched a regular expression that covered no unique local IPv6 address, so a
URL whose host was `fd00::1` reached a diagnostic while a URL whose host was
the IPv4 loopback address did not.

`internal/environment` still declares its own copy of the commit identifier
pattern at `internal/environment/environment.go:98`. That copy screens a branch
and revision value rather than evidence a producer persists, and Issue #343 did
not include it.

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
imports `internal/evidence` for `DiagnosticRef`, and `RunResult` holds a
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

## Contract decisions

Issue #326 requires every contract to be classified as preserved or changed
before the coordinated cutover. Issue #331 records that classification here.
A contract is a surface an outside consumer can depend on without reading the
Go implementation: a command name and its flags, a `mise` task name, a
persisted JSONL shape, an FSL specification, the skill graph and replay data, a
committed fixture, the skill layout, a diagnostic code, or a redaction rule for
privacy-sensitive evidence.

`workflow/contract-decisions.yml` is the machine-readable form of the table
below. The `check-contract-decisions` repository check reads that file and
fails on a missing field, an incomplete change record, an evidence path absent
from the tree, an entry this document does not describe, and a version
identifier Issue #326 has not separately approved. `cmd/check-repository` runs
the check, so `mise run check:repository` and `mise run validate:all` enforce
the record.

`preserved` means the surface behaves at the branch head as it did at the
measured baseline `4cce0641bbc9bc28c9bba47522a6071b0acded69` and its existing
check still passes. `changed` means an observable difference exists and the
entry records the old behavior, the new behavior, the reason, the requester
confirmation, the migration condition, the compatibility impact, and the
validation observation.

| Contract | Surface | Decision |
| --- | --- | --- |
| `cli-command-names` | Command names and flags under `cmd/`. | preserved |
| `repository-check-list` | The checks `cmd/check-repository` prints and its total. | changed |
| `mise-task-names` | Task names in `mise.toml`. | preserved |
| `skill-trace-jsonl` | The persisted skill-trace JSONL shape. | preserved |
| `validator-diagnostic-jsonl` | The validator-diagnostic JSONL shape and its codes. | preserved |
| `failure-record-jsonl` | The failure-record JSONL shape. | preserved |
| `skill-environment-json` | The skill-environment report shape. | preserved |
| `fsl-specifications` | The specification sources under `specs/`. | preserved |
| `fsl-verifier-failure-classification` | How a caller learns the verifier could not judge a specification. | changed |
| `skill-graph` | The skill graph data and its schema version. | preserved |
| `replay-fixtures` | The replay fixtures and the outcome each asserts. | preserved |
| `evidence-fixtures` | The committed trace, diagnostic, context, and failure-record fixtures. | preserved |
| `skill-layout` | The `skills/<category>/<skill-name>/SKILL.md` layout, its catalog entry, and where a per-skill asset lives. | changed |
| `evaluation-corpus` | The evaluation scenarios and their assertions. | preserved |
| `execution-policy-files` | The deliberation and execution-strategy policies. | preserved |
| `evidence-redaction` | The rules that keep a credential, a private URL, and a user path out of evidence. | preserved |

Thirteen of the sixteen contracts are preserved. Issue #346 changed
`skill-layout` after the redesign, as its section below records. The redesign moved Go package
boundaries and routed external calls through provider ports; it renamed no
command, no task, no schema field, and no skill path. `git diff --stat
4cce0641..HEAD -- mise.toml specs/ workflow/ CATALOG.yml skills/ .github/ cmd/`
shows changes only in the three Project command call sites, one added line in
`cmd/check-repository/main.go`, this document, and the two machine-readable
files this redesign adds.

### Changed: the repository check list

`cmd/check-repository` ran 21 checks at the baseline and printed
`check:repository: all 21 repository checks passed.` on its final line. It now
runs 24 and prints the matching total. Issue #328 added
`check-module-boundaries` so the recorded dependency direction is enforced,
Issue #331 added `check-contract-decisions` so this table is enforced the same
way, and Issue #332 added `check-cutover-record` so the cutover record is too.
An unenforced record drifts, which `docs/validation-tiers.md` demonstrated by
carrying a stale check total until Issue #331 reconciled it.

The requester confirmed the change while planning Issue #331 and confirmed the
third added check while planning Issue #332. A consumer that reads the printed
list by name keeps working, because the list grows and no existing name was
removed or renamed; a consumer that asserts an exact total updates that total
once. The exit status contract is unchanged: zero when every check passes and
one when any check fails. `go run ./cmd/check-repository`
prints the 24 named checks and the matching total, and
`cmd/check-repository/main_test.go` covers the aggregate result.

### Changed: the FSL verifier failure classification

At the baseline, an absent `fslc` binary, an expired deadline, and an
interrupted run surfaced through the same path as a verifier that ran and
rejected the specification, so an environment fault read as an invalid
specification. `internal/fsl/run.go` now reports on the specification only when
the verifier ran and returned a status, and reports every other outcome as a
verifier infrastructure error.

Issue #330 gave every external call a failure kind, and a specification that
was never judged must not be reported as one that failed, because the two need
different responses from the reader. The requester confirmed the change while
planning Issue #331. A caller that treated any non-zero verifier result as an
invalid specification now separates the two; no committed specification
changes, and a passing run is byte-identical. `internal/fsl/run_test.go`
exercises the absent, timed-out, and interrupted verifier against a provider
stub, and `mise run verify:fsl` passes on the committed specifications.

### Changed: the skill layout

At the baseline the repository stated no rule for where a per-skill asset, a
file that belongs to exactly one skill, lives. Evaluation scenarios sat in
`evaluations/scenarios/<skill-name>/` with no checked link to `skills/`, so a
scenario directory named after a retired skill, a scenario filed under another
skill's directory, or a catalog entry without a skill directory passed
`check-evaluation`. `AGENTS.md` and `evaluations/README.md` now state that every
per-skill asset lives below its skill root and name the evaluation inputs,
scenarios and fixtures, as the one exception. `check-evaluation` enforces the link the exception leaves: every
scenario directory except `e2e` names a cataloged skill, every scenario's
`skill` field equals its directory name, and every cataloged skill resolves to
`skills/<category>/<skill-name>/SKILL.md`.

The evaluation inputs stay central because `gh skill install` copies the
whole skill directory to every user and into the evaluation sandbox
(`internal/eval/run.go`), where a scenario would show its expectations to the
agent under evaluation. A fixture is staged only for the scenario that names
it. The requester confirmed the exception, the three relationships, and this
classification while planning Issue #346 and extended the exception to the
fixtures during review of Pull Request #355. No file moves and installation output
is unchanged; only `check-evaluation` rejects trees it previously accepted.
`check-evaluation` still reports 77 scenarios for 20 cataloged skills, and
`internal/eval/corpus_test.go` fails each broken relationship.

### No version identifier

No entry in the table assigns `v1`, `v2`, or any other version identifier.
Issue #326 allows one only after a separate recorded decision that names the
identifier, its compatibility meaning, and its migration conditions, and the
`check-contract-decisions` check fails on an entry that names one without a
`version_decision` field. The `schema_version` field inside an existing schema
is not a new identifier; this Issue changes no such value.

## Cutover and recovery

Issue #326 requires one cutover point for the selected boundary model and the
approved contract changes, with pre-cutover readiness conditions, a recovery
procedure, a migration order, and post-cutover validation. Issue #332 records
them here. `workflow/cutover-record.yml` is the machine-readable form, and the
`check-cutover-record` repository check enforces it.

The cutover point is `9f04db42a68e5140e4dc4067a42fdc8b02b230f0`, the commit
Issue #331 produced. Every approved contract change is active at that commit
and at no earlier one: `check-module-boundaries` lands with Issue #328, the
typed evidence boundary with Issue #329, the provider ports with Issue #330,
and `check-contract-decisions` with Issue #331. The readiness conditions were
observed at that commit rather than before it, because the four Sub-issues
merged one at a time through the ordinary review path. The record therefore
proves that the conditions hold at the recorded point; it does not claim a gate
ran ahead of it.

### Migration order

| Order | Issue | Delivers | Depends on |
| --- | --- | --- | --- |
| 1 | #328 | Module ownership and dependency direction, enforced by `check-module-boundaries`. | none |
| 2 | #329 | The typed evidence boundary: `internal/evidence` and `trace.RunResult`. | #328 |
| 3 | #330 | Provider ports and adapters in `internal/provider` with one failure classification. | #328, #329 |
| 4 | #331 | The contract decisions and `check-contract-decisions`. | #328, #329, #330 |

Each slice depends only on slices before it. `check-cutover-record` rejects a
`depends_on` entry that names an Issue further down the list, so the file
states an order rather than a set.

### Readiness and recovery matrix

Every component, consumer, schema, fixture, workflow, and document the cutover
touches is one row. Each row names the module that owns it, the readiness
condition, the command that observes the condition, and what recovery does to
it. The sixteen contracts of `workflow/contract-decisions.yml` are each covered
by at least one row, and every module in `workflow/module-ownership.yml` owns
at least one row. `check-cutover-record` fails when either is not covered, so a
later module or contract cannot land without a matrix row.

| Participant | Kind | Owner | Readiness observed by |
| --- | --- | --- | --- |
| `module-foundation` | module | foundation | `go run ./cmd/check-repository` |
| `module-provider` | module | provider | `go run ./cmd/check-repository` |
| `module-domain` | module | domain | `go run ./cmd/validate-skill-graph` |
| `module-policy` | module | policy | `go run ./cmd/validate-execution-strategy` |
| `module-evidence` | module | evidence | `go run ./cmd/check-repository` |
| `module-execution` | module | execution | `go run ./cmd/check-evaluation` |
| `module-governance` | module | governance | `go run ./cmd/check-repository` |
| `command-surface` | command | composition | `go run ./cmd/validate-script-tests` |
| `mise-tasks` | workflow | composition | `go run ./cmd/validate-mise-tasks` |
| `continuous-integration` | workflow | composition | `go run ./cmd/check-repository` |
| `fsl-specifications` | schema | domain | `mise run verify:fsl` |
| `evidence-schemas` | schema | evidence | `go run ./cmd/check-repository` |
| `skill-graph-data` | consumer | domain | `go run ./cmd/validate-skill-graph` |
| `replay-consumers` | consumer | evidence | `go run ./cmd/check-repository` |
| `evidence-fixtures` | fixture | evidence | `go run ./cmd/check-repository` |
| `execution-policies` | fixture | policy | `go run ./cmd/validate-deliberation-policy` |
| `published-skills` | consumer | domain | `go run ./cmd/check-repository` |
| `evaluation-reports` | consumer | execution | `go run ./cmd/check-evaluation` |
| `architecture-record` | document | composition | `go run ./cmd/check-repository` |

`composition` is the owner for a surface that lives in `cmd/` or in the
repository configuration rather than inside one internal module. It is the same
owner name `workflow/contract-decisions.yml` uses, and `check-cutover-record`
accepts it alongside the seven module identifiers in
`workflow/module-ownership.yml`.

### Recovery

The documented safe state is the tree of the measured baseline
`4cce0641bbc9bc28c9bba47522a6071b0acded69`. Recovery reaches it by reverting
every commit from the `main` head back to the baseline, newest first, on the
Issue branch of a recovery Issue. Recovery never rewrites published history, so
it needs no force-push and no GitHub Ruleset change, both of which Issue #326
excludes.

The range starts at the `main` head rather than at the cutover point. The
planning step of Issue #353 checked out the cutover point, reverted
`4cce0641..9f04db4`, and ran `git merge origin/main` with `main` at
`f049bf8`, 48 commits past the cutover point. The merge left 18 files in
conflict, and GitHub runs no `pull_request` check on a Pull Request with a
merge conflict. Reverting a linear history newest first cannot conflict. The
cost is that the work that landed after the cutover point is reverted too, and
it must land again after recovery.

Three conditions start recovery: a readiness condition that fails at the
cutover point, a post-cutover check that fails after it, and a contract drift
that `check-contract-decisions` or `check-cutover-record` reports. The third
condition has two answers, and the record states which applies: reconcile the
record when the tree is correct, and revert when the tree is not.

The procedure was rehearsed twice. The first rehearsal ran for Issue #332 in a
throwaway worktree checked out at the cutover point, so the branch under review
and the published history stayed untouched. The observations:

- `git revert --no-commit 4cce0641..9f04db4` reverted all 20 commits of the
  range without a conflict.
- `git diff --stat 4cce0641` reported no difference afterwards, so the reverted
  tree is the baseline tree rather than something close to it.
- `go build ./...` exited zero.
- `go test ./...` exited zero with 30 packages reporting `ok`.
- `go run ./cmd/check-repository` reported `all 21 repository checks passed`,
  the baseline total. The three checks this redesign adds are removed by the
  same revert that removes the three records they enforce.

That rehearsal showed that the revert produces a building, passing tree at
the baseline. It did not exercise the review path a real recovery Pull Request
takes.

The second rehearsal ran for Issue #353 and took that review path without
merging:

- `issue/353` was created from the `main` head
  `f049bf8ae49183577fa0d94a6bba3eb50a7d8fe7`, and all 68 commits back to the
  baseline reverted newest first without a conflict, one signed revert commit
  each. `git diff --stat 4cce0641` reported no difference.
- On the reverted tree, `go build ./...` and `go test ./...` exited zero,
  `go run ./cmd/check-repository` reported `all 21 repository checks passed`,
  and `mise run validate:all` exited zero with `FSLC_BIN_DIR` set.
- Draft Pull Request [#360](https://github.com/hidekitux/skills/pull/360)
  opened from `issue/353` to `main` at the head
  `747602c36a1e1ae8476dedc507460a95e9cd6166`.
- All 10 status checks the `main` ruleset requires passed, and
  `gh pr view 360 --json mergeStateStatus` reported `CLEAN`.
- The [review](https://github.com/hidekitux/skills/pull/360#issuecomment-5807903929)
  reported no findings. Each revert commit has the `git patch-id` of the
  reverse of the commit it names, and the named commits equal
  `git rev-list 4cce0641..f049bf8` exactly.
- Pull Request #360 was closed without a merge, and `main` stayed at
  `f049bf8`.

Pull Request #360 retired the `Remaining risks` entry that Issue #353 owned:
the recovery procedure has now passed the required checks and a review as a
recovery Pull Request.

### Post-cutover observations

The checks below ran at the branch head. Each one covers an aspect Issue #332
names, and `check-cutover-record` fails when an aspect has no check.

| Aspect | Command | Result |
| --- | --- | --- |
| public behavior | `go run ./cmd/check-repository` | All 24 named checks ran and the final line reported the matching total. |
| public behavior | `go run ./cmd/validate-script-tests` | 43 commands and 12 scripts mapped; no command name or flag changed. |
| privacy | `go run ./cmd/check-sensitive-content` | Passed; no credential, private address, or user path reached a committed artifact. |
| authority | `go run ./cmd/check-analyze-readonly` | Passed; no `analyze-*` skill instructs an Issue or Pull Request mutation. |
| provider failure | `go test ./internal/provider/` | Passed; success, failure, timeout, interruption, unavailable capability, and retry exhaustion stay separate. |
| provider failure | `go test ./internal/fsl/` | Passed; an absent, timed-out, or interrupted verifier reports a verifier infrastructure error. |
| terminal outcome | `go test ./internal/trace/` | Passed; the pass, fail, skipped, interrupted, and infrastructure-error outcomes stay separate. |
| deterministic replay | `go test ./internal/replay/` | Passed; and `check-repository` returned the recorded outcome for all 10 replay fixtures. |
| compatibility | `go run ./cmd/validate-skill-graph` | 20 skills, schema version 1, unchanged. |
| compatibility | `go run ./cmd/check-evaluation` | 77 scenarios for 20 cataloged skills passed. |
| compatibility | `mise run verify:fsl` | Verified every committed specification once `FSLC_BIN_DIR` was set. Without it the task reports an unavailable environment. See below. |
| compatibility | `mise run validate:all` | Exit status 0 from the branch tip, with `FSLC_BIN_DIR` set. |

`mise run verify:fsl` first failed in the worktree that produced this record,
because the task does not export `FSLC_BIN_DIR`, which
`scripts/setup/environment-state.sh` sets for a provisioned environment. With
that variable set the same task verifies every committed specification and
`mise run validate:all` exits zero. The first failure is an unavailable
environment, which `internal/fsl/run.go` has classified separately since Issue
#330; it is neither a product failure nor a contract change. `Final validation`
records the same outcome at the branch head.

## Final validation

Issue #333 closes the redesign with one validation record. Every command below
ran in the worktree for the `issue/333` branch, whose only change to the tree
is the documentation this Issue adds. The one exception is run
`20260924T033735Z` under `Behavioral evaluation`, which ran for Issue #352 on
its own branch. The record keeps four outcomes apart: a
**product outcome** comes from the repository's own code and committed
artifacts, an
**infrastructure outcome** is a fault in the tool or its dependency rather than
in the artifact under check, an **unavailable-environment outcome** is a check
that cannot start because a host capability or a provisioned variable is
absent, and an **interrupted outcome** is a run stopped before its terminal
status.

### Repository validation

| Command | Outcome | Classification |
| --- | --- | --- |
| `mise run validate:all` with `FSLC_BIN_DIR` set | Exit status 0. It ran `check:repository`, `check:branch-policy`, `check:diff`, `check:tasks`, `check:skills`, `check:hosts`, `lint:actions`, `lint:go`, `lint:python`, `lint:shell`, `install:fsl`, `verify:fsl`, and `test:go`. | product |
| `go run ./cmd/check-repository` | `check:repository: all 24 repository checks passed.` | product |
| `go test ./...` inside `test:go` | 32 packages reported `ok`; no package failed. | product |
| `go list ./...` | 70 packages, no import cycle. `go list ./internal/...` returns 24. | product |
| `go run ./cmd/validate-skill-graph` | `skill graph valid: 20 skills, schema version 1`. | product |
| `go run ./cmd/check-evaluation` | `Evaluation corpus check passed: 77 scenario(s) for 20 cataloged skill(s).` | product |
| `go run ./cmd/validate-script-tests` | `Script-test mapping check passed: 43 command(s) and 12 script(s) mapped.` | product |
| `go run ./cmd/verify-fsl` with `FSLC_BIN_DIR` set | `Verified 9 FSL spec(s).` | product |
| `mise run verify:fsl` without `FSLC_BIN_DIR` | Exit status 1 after `Checking specs/branch-flow.fsl`. | unavailable environment |
| `go run ./cmd/verify-fsl -diagnostic-format json` without `FSLC_BIN_DIR` | `{"code":"fsl.verify.infrastructure","category":"infrastructure_error",...,"retryable":true,"remediation":"retry_operation"}` | unavailable environment |
| The `check-cutover-record` check, run by `go run ./cmd/check-repository` | `cutover record valid: 19 participant(s), 6 recovery step(s), 12 post-cutover check(s).` | product |
| The `check-contract-decisions` check, run by `go run ./cmd/check-repository` | `contract decisions valid: 16 contract(s), 14 preserved, 2 changed.` | product |
| The `check-module-boundaries` check, run by `go run ./cmd/check-repository` | `module boundaries valid: 24 packages in 7 modules, 35 allowed module edges, 43 command packages scanned.` | product |
| `go run ./cmd/check-sensitive-content` | `Sensitive-content check passed.` | product |

The text form of `verify-fsl` prints `exit status 1` and names no
classification. The JSON form carries it. A reader who needs to separate an
absent verifier from an invalid specification must pass
`-diagnostic-format json`; the row above records both forms so the distinction
is not inferred from the exit status alone.

No command in this record was interrupted. The interrupted outcome has no
committed command that produces it on demand, so it is covered by the fixture
and the package tests named in the coverage table below rather than by a run
recorded here.

### Behavioral evaluation

`mise run evaluate:smoke` ran three times. The first two runs are on
repository commit `facf9e641f8020e35c10e0dab515e3352f81d40e`; the third, for
Issue #352, is on `c51e94856641dc7cb7e3fd852c85bec65e27d695` with
`EVAL_GITHUB_REPO` set to a private sandbox repository.
`docs/validation-tiers.md` places live behavioral evaluation outside the CI
tiers, so no run blocks a pull request.

| Run | Outcome | Classification |
| --- | --- | --- |
| `20260918T044954Z` | Every driver returned `infrastructure_error` or `skipped`; no scenario produced a behavioral verdict. The `antigravity` driver was not signed in and the `opencode` driver returned `UnknownError` `err_34aa405f` from its server. | infrastructure, unavailable environment |
| `20260918T045712Z` | After the `antigravity` sign-in, four scenarios passed, two were `skipped` with `sandbox_repo_not_configured`, and `triage-issues-success` failed on the one driver that ran: the transcript did not name the expected handoff `create-issue`. The `opencode` driver still returned `infrastructure_error` for every scenario it attempted. | product for the five scenarios that reached a verdict, unavailable environment for the two `skipped` scenarios, infrastructure for the `opencode` driver |
| `20260924T033735Z` | Both drivers ran all seven scenarios: 6 records passed, 4 failed, 4 returned `infrastructure_error`, and none was `skipped`. No `opencode` record carries `UnknownError`. The four `infrastructure_error` records are the 5-minute stage timeout on `audit-workflow-enforcement-boundary` and `plan-issue-success`, on both drivers. | product for the ten records that reached a verdict, infrastructure for the four timeouts |

The `triage-issues-success` failure was a skill defect, not a model limit.
Issue #351 reproduced it with `mise run evaluate:all -- --scenario
triage-issues-success` on the drivers each row names and read each transcript.
The harness keeps no transcript, so each run set `EVAL_CLAUDE_CMD` and
`EVAL_ANTIGRAVITY_CMD` to a wrapper that copied the driver output to a local
file outside the repository.

| Run | Commit | `claude-code`, `claude-sonnet-5` | `antigravity`, `gemini-3.7-flash-low` |
| --- | --- | --- | --- |
| `20260924T023443Z` | `8b5249f` | `infrastructure_error`, not signed in | `fail`: sends the ready Issue #44 to `plan-issue` and never names `create-issue` |
| `20260924T023541Z` | `8b5249f` | `infrastructure_error`, not signed in | `pass`: names `create-issue` as the owner of merging #41 into the existing #42 |
| `20260924T023925Z` | `8b5249f` | `pass`: names `create-issue` for a correction that "may" need a new or corrected Issue | not run |
| `20260924T024155Z` | `8b5249f` | `pass`: names `create-issue` as one of two owners for the #41 and #42 merge | not run |
| `20260924T024729Z` | `2d3f9ee` | `pass` | `pass` |
| `20260924T025013Z` | `2d3f9ee` | `pass` | `pass` |

Every transcript at `8b5249f` handed the ready Issues to `plan-issue`. The
failing transcript followed `skills/analyze/triage-issues/SKILL.md`, which sent
ready governed work to the existing change flow. The three passing transcripts
named `create-issue` only as a conditional or shared owner of a tracker
correction, which `create-issue` does not perform because it only creates
Issues. `workflow/skill-graph.yml` gives `triage-issues` one success
transition, `handoff-change-issue` to `create-issue`, so `SKILL.md` offered a
route the contract lacks and named no owner for a tracker correction.

Commit `2d3f9ee` states in `SKILL.md` that every report goes to `create-issue`,
that the report says so when no finding calls for new work, and that a ready
Issue goes to `plan-issue` and a tracker correction to the Issue's maintainer.
All four runs at `2d3f9ee` passed. Each transcript names `create-issue` in its
handoff and states that no finding calls for a new Issue. Only the
`antigravity` transcript of run `20260924T024729Z` also states that the report
goes to `create-issue`. The other three name `create-issue` as the owner of new
work, and the `claude-code` transcript of run `20260924T025013Z` adds that a
new investigation task, if one is needed, would go to `create-issue`.

Run `20260924T033735Z` retired two conditions of the first two runs and
recorded the third as an environment limit:

- The `opencode` driver failure was a missing OpenCode Go credential, not a
  host service fault. `opencode run --print-logs` showed
  `ProviderModelNotFoundError` for `opencode-go/deepseek-v4-flash` behind the
  `UnknownError`. After `opencode auth login`, the run recorded seven
  `opencode` records: five verdicts, two stage timeouts, and no
  `UnknownError`. `docs/evaluation.md` and
  `evaluations/README.md` name the credential as a driver prerequisite.
- The two `sandbox_repo_not_configured` skips are gone: with
  `EVAL_GITHUB_REPO` set, `plan-issue-success` and `implement-issue-negative`
  ran on both drivers. `implement-issue-negative` reached a verdict, and
  `plan-issue-success` ended in the stage timeout.
- The `(could not read directory)` listing is still printed. It comes from
  `gh skill install --from-local`, which reads a different path for its
  post-install file tree than the one it installs to. `docs/evaluation.md`
  records it as an environment limit with the evidence that installation is
  complete.

### Outcome coverage

The second acceptance criterion of Issue #333 names six outcomes. Each one has
a committed asset that observes it.

| Outcome | Asset | Observation |
| --- | --- | --- |
| success | `workflow/replay-fixtures/valid.jsonl`, `workflow/replay-fixtures/valid-merge.jsonl`; 41 `positive` evaluation scenarios | `validate-replay-fixtures` reports `valid` for both files. |
| failure | `workflow/replay-fixtures/invalid-order.jsonl`, `read-only-mutation.jsonl`, `wrong-branch-owner.jsonl` (`violation`), `review-loop-beyond-bound.jsonl` (`retry_exhausted`), `missing-evidence.jsonl`, `missing-validation.jsonl`, `partial.jsonl` (`incomplete_evidence`); 10 `negative` and 22 `boundary` evaluation scenarios | Each fixture asserts its own non-success outcome, and the check reports the recorded outcome rather than a pass. |
| interruption | `workflow/replay-fixtures/interrupted.jsonl` under invariant `InterruptedIsNotSuccess`; `internal/trace` and `internal/provider` package tests | The fixture asserts `interrupted`, and the tests keep `interrupted` apart from `failed` and `infrastructure_error`. |
| unavailable environment | `mise run verify:fsl` without `FSLC_BIN_DIR`; `evaluations/fixtures/retrospect-unavailable` and `evaluations/fixtures/triage-unavailable`; `internal/eval/strategy.go` | The command reports `fsl.verify.infrastructure`, and the fixtures drive a skill that must state the limit instead of inventing a finding. |
| privacy | `go run ./cmd/check-sensitive-content`; `evaluations/scenarios/analyze-project/analyze-project-safety.yaml`; the `evidence-redaction` contract | The check passes, and the scenario asserts that a private context log stays out of a public report. |
| deterministic replay | The 10 files under `workflow/replay-fixtures/`; `cmd/replay-skill-trace` | `validate-replay-fixtures` returns the recorded outcome for all 10 files on every run. |

No approved contract change required a new observation. The two changed
contracts in `workflow/contract-decisions.yml` are `repository-check-list` and
`fsl-verifier-failure-classification`. The first changes a printed total inside
`cmd/check-repository`, and the second changes how `internal/fsl/run.go`
classifies a verifier that never judged a specification. Neither changes a
skill behavior an evaluation scenario observes, so the evaluation corpus stays
at 77 scenarios and no fixture changes.

### Consumer agreement

The consumers below each read the same artifact and authority contracts. They
agree at the branch head.

| Consumer | Command | Result |
| --- | --- | --- |
| FSL specifications | `mise run verify:fsl` with `FSLC_BIN_DIR` set | Verified 9 specifications. |
| graph validation | `go run ./cmd/validate-skill-graph` | 20 skills, schema version 1. |
| replay validation | `validate-replay-fixtures` inside `check-repository` | 10 fixtures, each returning its recorded outcome. |
| trace validation | `validate-skill-trace` and `validate-diagnostic` inside `check-repository` | The committed trace and diagnostic fixtures match their schemas. |
| evaluation report | `go run ./cmd/check-evaluation` | 77 scenarios for 20 cataloged skills. |
| module boundaries | `check-module-boundaries` inside `check-repository` | 24 packages in 7 modules, 35 allowed edges, 43 command packages scanned. |

### Documentation alignment

Each document that states an ownership, failure, or contract claim now points
at this record instead of restating it: `docs/skill-contract.md`,
`docs/fsl.md`, `docs/evaluation.md`, `docs/skill-graph.md`,
`docs/skill-trace.md`, `docs/validator-diagnostics.md`,
`docs/validation-tiers.md`, and `CONTRIBUTING.md`. One record holds the
boundary; a second copy would drift.

The three `21 repository checks` figures in this document are baseline figures
and stay as they are. The `Measured baseline` section and the revert rehearsal
describe commit `4cce0641bbc9bc28c9bba47522a6071b0acded69`, where the total was
21, and the `Changed: the repository check list` section states both the
baseline 21 and the current 24.

## Remaining risks

Each entry names the Issue that owns it. An entry leaves this list only with
the evidence that retired it.

No entry remains.

## Handoff to the dependent Sub-issues

| Issue | Extends | Completed evidence |
| --- | --- | --- |
| #328 | Landed. Added `Module ownership`, `Dependency direction`, and `Test substitution points`. | `workflow/module-ownership.yml`; `check-module-boundaries` reports 24 packages in 7 modules and 35 allowed edges. |
| #329 | Landed. Added `Evidence data path`, the `internal/evidence` package to `Module ownership`, and the typed-result seam to `Test substitution points`. | `internal/evidence` holds `DiagnosticRef`, which `internal/trace` and `internal/diagnostic` alias; `go test ./internal/trace/` and `go test ./internal/diagnostic/` pass inside `test:go`. Issue #343 moved the pattern tests to `internal/redact` with the patterns, so `internal/evidence` has none of its own. |
| #330 | Landed. Added the `provider` module to `Module ownership` and `Dependency direction`, added `Provider ownership`, and replaced the provider note in `Test substitution points`. | `internal/provider`; `check-module-boundaries` fails a use of `os/exec` or `net/http` outside the module, and `go test ./internal/provider/` keeps every failure kind apart. |
| #331 | Landed. Added `Contract decisions`, `workflow/contract-decisions.yml`, and the `check-contract-decisions` repository check. | `check-contract-decisions` reports 16 contracts, 14 preserved and 2 changed. |
| #332 | Landed. Added `Cutover and recovery`, `workflow/cutover-record.yml`, and the `check-cutover-record` repository check. | `check-cutover-record` reports 19 participants, 6 recovery steps, and 12 post-cutover checks. |
| #333 | Landed. Added `Final validation` and `Remaining risks`, and reconciled every section above with the shipped tree. | `mise run validate:all` exits 0 with `FSLC_BIN_DIR` set; `Final validation` holds every command, its outcome, and its classification. |

Each Sub-issue changes `workflow/module-ownership.yml` and this document
together. A module split that only one of the two records is a drift the
`check-module-boundaries` check reports.
