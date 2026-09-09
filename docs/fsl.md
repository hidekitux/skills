# FSL operations

## Purpose

FSL does not directly assess skill prose or Markdown quality. This repository uses it to verify skill lifecycle transitions and high-risk workflows as state machines. For the skill-set map and layer vocabulary, see the [README](../README.md#skill-set-map).

Suitable targets include:

- Creation, review, validation, publication, update, and deprecation transitions.
- Invariants such as only a successfully validated version may be published.
- Skill workflows containing approval, rollback, retry, authority, or duplicate execution.

Use `gh skill publish --dry-run` for prose, file-name, and frontmatter checks.

## Location and verification

Keep a specification beside its owner; do not copy it.

- A published skill workflow belongs in `skills/<skill-name>/specs/<topic>.fsl`. Create a **relative symbolic link** to that source at `specs/<skill-name>/<topic>.fsl` so the repository can verify and reference it.
- A release gate or cross-cutting branch policy that belongs to no individual published skill belongs in `specs/<topic>.fsl` as a regular file.

Issue #29 confirmed this packaging memo: `create-issue` owns `issue-creation.fsl`, `create-pr` owns `pull-request-creation.fsl`, and `branch-flow.fsl` and `release-gate.fsl` remain repository-owned. After adding a specification, run the following from the repository root:

```text
mise run verify:fsl
```

The task discovers root specifications and `skills/**/specs/*.fsl` sources; downloads and checksum-verifies the official `fslc` v4.2.0 release; then runs `fslc check` and `fslc verify --depth 8` for each source. It uses a temporary cache under `RUNNER_TEMP` in CI and `TMPDIR` locally, and repository validation checks symlink integrity without running sources twice. Override the depth when needed, for example `FSL_DEPTH=12 mise run verify:fsl`. Supported platforms are GitHub Actions Linux x64 and development macOS Apple Silicon.

## Cross-skill trace replay

`specs/cross-skill-workflow.fsl` models the governed path from `create-issue` through `review-pr`, including evidence, authority, interruption, incomplete evidence, and the two-pass review limit. `internal/replay` checks the observed structured traces against `workflow/skill-graph.yml` before it emits normalized actions. `cmd/replay-skill-trace` then passes those actions to `fslc replay`.

Run a replay with an ordered JSONL trace set:

```text
go run ./cmd/replay-skill-trace --root . --input workflow/replay-fixtures/valid.jsonl --format json
```

The command returns `valid` only when graph conformance and FSL replay both pass. `violation` names an observed invariant failure. `incomplete_evidence`, `interrupted`, and `retry_exhausted` are explicit non-success outcomes; the command never treats missing events as a pass. The repository fixture check covers these outcomes under `workflow/replay-fixtures/`.

FSL replay checks the finite action model. It does not prove that a host emitted every event, that a branch or GitHub object has the recorded state, that `SKILL.md` prose was followed, or that the implementation is correct. Behavioral evaluation remains a separate claim.

## Authoring commitment

Before writing an FSL specification, confirm a formalization memo in the conversation. It must include states, actions, prohibited states, boundary conditions, modeling assumptions, and open questions. Do not guess an open decision that affects behavior.

After a specification verifies, run `mise run mutate:fsl` to confirm that important properties can detect faults. Review mutation survivors; they do not fail the command automatically. A passing FSL result shows that the specification is internally consistent, not that an implementation or a skill body conforms to it automatically.

## Publication-gate specification

`specs/release-gate.fsl` models catalog update, `mise run validate:all`, commit, `mise run verify:release -- vX.Y.Z`, and publication order. Publication is allowed only from a clean validated commit, and existing tags cannot be reused. It does not prove the GitHub API publication result; it verifies the preconditions for using the publication command.
