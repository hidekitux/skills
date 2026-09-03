# Behavioral skill evaluation

This document defines how the repository measures the user outcome of every
published skill and what the evidence means for catalog status. The corpus,
input contract, and harness live under [evaluations/](../evaluations/README.md).

Metadata validation and `gh skill publish --dry-run` are **not** behavioral
evaluation: they check shape, not outcome. Behavioral evaluation runs a
realistic request against a skill on a real host and checks whether the agent
selects the right workflow and produces a correct, safe, useful result
(Issue 173).

## What counts as evidence

A scenario run records, for each result, exactly one verdict:

- `pass` — deterministic assertions passed;
- `fail` — a deterministic assertion failed;
- `skipped` — not selected, host CLI unavailable, or a documented sandbox
  requirement was not configured;
- `infrastructure_error` — the run itself could not start (fixture staging,
  missing host, auth failure, timeout).

Provenance is recorded with every result: host, model (from `opencode.json`),
prompt SHA-256, repository commit, and fixture IDs, so any result can be
reproduced from its documented inputs. Reports are machine-readable JSONL plus
a human-readable Markdown summary under `evaluations/reports/`.

### What a prose observation cannot decide

Each skill carries its own writing rules in its `SKILL.md`. Every positive
scenario in the corpus asserts that the transcript holds none of five markers.
`docs/writing-style.md` calls `delve`, `pivotal`, and `multifaceted` an
inflated style word, and it calls `facilitate` and `commence` Latinate padding.

`internal/eval/prose_test.go` runs those assertions against a conforming
transcript and a rule-breaking one, and requires the verdicts to differ. It
reads the scenario list from the corpus, so an emptied marker list and a newly
added positive scenario without markers both fail `mise run test:go` instead of
reporting a silent pass.

A passing verdict is not conformance. It reports the absence of five markers
from one transcript, and it cannot decide whether the prose keeps one idea in
one sentence, keeps one term per concept, varies its sentence length, states a
position with its reason, or cites the file behind each claim. Five further
limits bound it:

- Most of these scenarios do not run at all in a default checkout. Every
  scenario that declares `github_sandbox: true` records `skipped` with
  `skip_reason: sandbox_repo_not_configured` until `EVAL_GITHUB_REPO` names a
  sandbox repository, and that gate is what decides whether a prose
  observation happens. It covered 12 of the 19 positive scenarios when this
  paragraph was written, leaving 7 that run. Recount the gated ones with
  `grep -l '^kind: positive' evaluations/scenarios/*/*.yaml |
  xargs grep -l '^github_sandbox: true' | wc -l`, which intersects the two
  sets. A second command,
  `grep -l '^github_sandbox: true' evaluations/scenarios/*/*.yaml | wc -l`,
  counts the flag across all 37 scenarios whatever their kind, and it was 23.
  Expect the two to differ, because only the first restricts the count to
  positive scenarios. Set `EVAL_GITHUB_REPO` before reading a prose result as
  evidence about a gated skill.
- The match is a case-sensitive substring, so only the two cased forms each
  scenario lists are observed.
- `utilize` and `serves as` are absent from the marker lists on purpose. Both
  appear in the carried rules as the example of what to avoid, so asserting
  them would fail every run in which an agent quotes its own instructions.
- `not just` is absent too. The rule it marks is the `not just X, but Y`
  frame, and the bare phrase is common enough in correct prose to fail a
  conforming run.
- The either-pass policy in `evaluations/README.md` passes a scenario when at
  least one host produced a deterministic pass, so a prose result may rest on
  one host and one model.

The rubric wording in those scenarios names the judgment-based rules, and
`evaluations/rubric.md` records rubric scores as opinion with bounded variance.
No rubric score gates a verdict.

## Behavioral threshold for promotion

A cataloged skill may be promoted from `experimental` to `stable` only when
all of the following hold, verified against a retained evaluation run:

1. **No deterministic failures.** Every success and failure/boundary scenario
   of the skill passes its deterministic assertions on the smoke run for the
   evaluated drivers. `fail` verdicts of any kind block promotion.
2. **Rubric floor.** Reviewed scenarios score the seven rubric dimensions
   (`evaluations/rubric.md`) with no dimension below 3 and a mean of at least
   4.0. Rubric scores are opinion and never override deterministic failures.
3. **Regression-free.** The same scenario set passed on the two most recent
   consecutive runs: a deterministic assertion that passed before and fails
   now is a regression that blocks promotion and triggers demotion review.
4. **Bounded variance.** Re-running an unchanged scenario produces identical
   deterministic verdicts and rubric scores within ±1 per dimension.
5. **Retained evidence.** A machine-readable report recording a qualifying
   pass for the skill exists under `evaluations/reports/`: a record whose
   `verdict` is `pass` for that skill **and** whose `rubric_review` is
   `complete` with all seven `rubric_scores` present (a pass for another
   skill in the same file, or a pass without a completed rubric review, is
   not retained evidence). This is enforced by `check-evaluation` (part of
   `check:repository`): a catalog entry with `status: stable` and no
   qualifying evidence fails repository validation. The remaining threshold
   items (rubric floor, regression-free, bounded variance) are verified by
   the release flow against the retained runs.

Skills also contract to **name the next owner**: when a scenario's `handoff`
is a cataloged skill name (per `docs/skill-contract.md`), the transcript
must contain it, so a flow that stops before its documented handoff fails
deterministically. Outcome markers such as boundary stop conditions are
asserted through `transcript_must` / `transcript_must_any`.

Changing a skill's actual `status` is release-flow work (and later Sub-issues
of Issue 165); this document defines the threshold that the release flow must
satisfy, and `specs/evaluation-gate.fsl` models the transition.

## Regressions block promotion

- A deterministic failure in a required scenario blocks promotion from
  `experimental` regardless of rubric scores.
- A documented regression that demotes a `stable` skill returns it to
  `experimental` (see `specs/evaluation-gate.fsl`); it may not be re-promoted
  until a clean retained run satisfies the threshold again.
- `skipped` and `infrastructure_error` verdicts neither prove nor disprove an
  outcome; promotion requires `pass` verdicts on the required scenarios for
  the evaluated drivers.

## Running evaluation locally

Live evaluation runs **locally** (Issue 173 decision record: it is not
scheduled in GitHub Actions; CI scheduling belongs to Issue 176). The static
corpus check runs in CI through `check:repository` on every pull request and
push, free of model calls.

```text
mise run evaluate:all --help
mise run evaluate:smoke      # smoke set on opencode + antigravity
mise run evaluate:all -- --host codex
mise run evaluate:all -- --host opencode,antigravity --skills plan-issue
```

Drivers: `codex` (OpenAI ChatGPT tier via Plus; default model `gpt-5.6-luna`),
`claude-code` (needs login; default `claude-sonnet-5`), `opencode` (reads
`.agents/skills`; default tier model `opencode-go/deepseek-v4-flash`), and
`antigravity` (uses the logged-in Google account by default; opt into Gemini
API key auth for headless runs with `EVAL_ANTIGRAVITY_KEY_MODE=1` plus
`GEMINI_API_KEY`, which makes the harness write `modelProvider: gemini` into
the sandbox settings and isolate HOME; default model `gemini-3.7-flash-low`).
Every driver pins an explicit model.
Driver commands are overridable through `EVAL_CODEX_CMD`, `EVAL_CLAUDE_CMD`,
`EVAL_OPENCODE_CMD`, and `EVAL_ANTIGRAVITY_CMD`; per-driver models through
`EVAL_CODEX_MODEL`, `EVAL_CLAUDE_MODEL`, `EVAL_OPENCODE_MODEL`, and
`EVAL_ANTIGRAVITY_MODEL`; a review hook through `--reviewer-cmd` for rubric
scoring. Running several drivers together applies the either-pass policy: the
scenario gate passes when at least one driver produced a deterministic pass,
so a rate-limited or credit-depleted provider (HTTP 429) does not block the
run. GitHub-dependent scenarios require a sandbox repository via
`EVAL_GITHUB_REPO`; without it they record `skipped` with reason
`sandbox_repo_not_configured`. When set, the harness registers the repository
as the sandbox Git origin and passes `GH_REPO` to the driver children so `git`
and `gh` resolve the same target. Driver processes never inherit
credential-like environment variables (`*KEY*`, `*TOKEN*`, `*SECRET*`,
`*PASSWORD*`, `*CREDENTIAL*`): the evaluated model can read its own
environment, and only the antigravity key-mode process receives
`GEMINI_API_KEY`. Account mode for antigravity keeps the real HOME, so the
evaluated agent also sees the developer's global `~/.gemini` skills; prefer
the opt-in key mode for reproducible runs.

## Validation of this system (Issue 173)

- Smoke suite run locally for the supported drivers with retained reports.
- Contract-violation injection (a dropped handoff, a dropped failure scenario,
  a leaked prompt) fails the appropriate scenario or corpus check
  deterministically; see `go test ./internal/eval -run Injection`.
- Re-running an unchanged scenario keeps deterministic verdicts stable;
  rubric variance is bounded by the reviewer procedure in `evaluations/rubric.md`.
- `mise run validate:all` covers the corpus statically through the
  `check-evaluation` repository check.