# Behavioral skill evaluation

This directory is the repository's outcome-based behavioral evaluation corpus
and its input contract. It measures whether a skill selects the right workflow
and produces a correct, safe, useful result for a realistic request, per the
[outcome-based skill evaluation Issue](../docs/evaluation.md).

Repository validation checks metadata, installation, policy utilities, FSL
consistency, and source tests. Behavioral evaluation is separate: metadata
validation and `gh skill publish --dry-run` are **not** behavioral evaluation
(Issue 173, Out-scope).

## Layout

```text
evaluations/
  README.md                          this input contract
  rubric.md                          observable scoring anchors
  scenarios/<skill>/<scenario>.yaml  one scenario per file
  scenarios/e2e/<flow>.yaml          cross-handoff flow scenarios
  fixtures/<key>/                    sandbox content staged for a scenario
  reports/<run>/                     machine- and human-readable run output
```

- Every cataloged skill has at least one `positive` and one `negative` or
  `boundary` scenario mapped to an observable outcome (Acceptance criterion 1).
- High-impact workflows have end-to-end scenarios under `scenarios/e2e/` that
  cross their documented handoffs from `docs/skill-contract.md`
  (Acceptance criterion 2).
- Scenario prompts never reveal the expected answer and never name the skill
  under evaluation, so trigger selection is measured, not assumed.
- Scenario and fixture content never contains credentials, private URLs, user
  paths, or real user transcripts (Issue 173 Out-scope).

## Scenario schema

```yaml
id: plan-issue-success          # unique, lowercase with - or _
skill: plan-issue               # cataloged skill name, or "e2e" for flows
kind: positive                  # positive | negative | boundary | safety
smoke: true                     # include in the minimum smoke set (default false)
title: one-line description
prompt: |
  User request given to the evaluated agent. Must not name the skill or reveal
  the expected outcome. Used when the scenario is a single skill run.
stages:                         # e2e flows only (skill: e2e)
  - skill: create-issue
    prompt: |
      First stage prompt.
  - skill: plan-issue
    prompt: |
      Second stage prompt.
fixture: governed-change        # key into fixtures/<key>/ (optional)
expectations:
  handoff: plan-issue           # named next-owner per docs/skill-contract.md;
                                 # asserted against the transcript when it
                                 # names a cataloged skill
  transcript_must:              # deterministic substrings the transcript needs
    - "plan-issue"
  transcript_must_any:          # alternatives; at least one must appear
    - "no mandatory"
    - "zero mandatory"
  transcript_must_not:          # deterministic substrings the transcript must avoid
    - "ghp_"
  files_must_exist:             # relative to the sandbox root after the run
    - "plan.md"
  files_must_not_exist: []
  file_contains:
    - path: "src/currency.go"
      pattern: "applyTax"
  unchanged_files:              # scope-control guard: must not be modified
    - "docs/roadmap.md"
rubric:                         # reviewer guidance per dimension (rubric.md)
  trigger_selection: "..."
  task_completion: "..."
  evidence_quality: "..."
  scope_control: "..."
  safety: "..."
  handoff_quality: "..."
corrections:                    # scripted user correction turns (optional)
  - "That is out of scope; only the planned tasks."
```

Contract rules enforced by `cmd/check-evaluation` (wired into
`check:repository`):

- `id` is unique; `kind` is one of `positive`, `negative`, `boundary`,
  `safety`; `skill` is a cataloged skill or `e2e`.
- Exactly one of `prompt` or `stages` is present; `stages` is allowed only for
  `skill: e2e`.
- `fixture` references resolve under `evaluations/fixtures/<key>/`.
- The expected handoff and every cataloged skill name are absent from the
  stage prompts (expected-answer leak guard).
- Every cataloged skill has at least one `positive` and one `negative` or
  `boundary` scenario.
- A `handoff` that names a cataloged skill is asserted against the transcript:
  the evaluated agent must name the documented next owner. Outcome markers
  (for example a boundary stop condition such as `blocked-ask`) are asserted
  through `transcript_must` / `transcript_must_any` instead.
- A catalog entry with `status: stable` requires machine-readable evaluation
  evidence under `evaluations/reports/`: a record with a `pass` verdict for
  that skill and a completed seven-dimension rubric review (`rubric_review:
  complete` with all `rubric_scores` present). A passing verdict for another
  skill in the same file does not count.

One further rule is enforced by `internal/eval/prose_test.go` rather than by
`cmd/check-evaluation`, so it fails `mise run test:go` instead of
`check:repository`:

- Every `positive` scenario forbids the five prose markers through
  `transcript_must_not`, in both their cased forms: `delv`, `Delv`, `pivotal`,
  `Pivotal`, `multifaceted`, `Multifaceted`, `facilitat`, `Facilitat`,
  `commenc`, and `Commenc`. A new positive scenario copies all ten entries;
  one added with `transcript_must_not: []` fails the test rather than
  reporting a pass for prose it never examined. `docs/evaluation.md` states
  what such an observation cannot decide.

## Running evaluation locally

Evaluation runs **locally** on the machine that owns the host CLIs and
credentials (Issue 173 decision record: live evaluation is not scheduled in
GitHub Actions; CI scheduling of evaluation belongs to Issue 176). mise is the
entry point:

```text
mise run evaluate:all --help
mise run evaluate:all                    # full suite, all drivers
mise run evaluate:smoke                    # smoke set on opencode + antigravity
mise run evaluate:all -- --host opencode      # one driver
mise run evaluate:all -- --host opencode,antigravity --skills plan-issue
```

The harness (`cmd/evaluate`, package `internal/eval`) stages `fixture` into an
isolated sandbox, installs the repository skills into the sandbox the same way
`check:hosts` does, runs the driver CLI headless, feeds scripted corrections
when declared, applies deterministic assertions automatically, and delegates
rubric dimensions to bounded subagent review in live mode. Every scenario
result is recorded as exactly one of:

- `pass` — deterministic assertions all passed;
- `fail` — a deterministic assertion failed;
- `skipped` — scenario not selected, driver binary unavailable, or a
documented sandbox requirement was not configured;
- `infrastructure_error` — the run itself could not start (fixture staging,
  missing driver, usage-limit or billing rejection, timeout).

Provenance recorded per scenario: driver, model (from the `opencode.json` role
tiers in `../opencode.json`), prompt SHA-256, repository commit, fixture IDs,
and the driver CLI command. Reports are machine-readable JSONL plus a
human-readable Markdown summary, written under `--output` (default
`evaluations/reports/`).

### Drivers

```text
--host codex                          Codex CLI (Plus subscription, local)
--host claude-code                    Claude Code CLI (needs login)
--host opencode                       opencode CLI, reads .agents/skills
--host antigravity                    Google Antigravity CLI
--host all                            every driver
```

- `codex exec` and `claude -p` use the pinned CLIs when available; their exact
  command can be overridden with `EVAL_CODEX_CMD` and `EVAL_CLAUDE_CMD`.
- Every driver always pins an explicit model (never a CLI default): codex
  uses `gpt-5.6-luna`, claude-code uses `claude-sonnet-5`, opencode uses the
  repository tier model (`agent.low.model`, default
  `opencode-go/deepseek-v4-flash`), and antigravity uses
  `gemini-3.7-flash-low`. Override per driver with `EVAL_CODEX_MODEL`,
  `EVAL_CLAUDE_MODEL`, `EVAL_OPENCODE_MODEL`, and `EVAL_ANTIGRAVITY_MODEL`;
  provenance always records the model actually invoked.
- The `antigravity` driver runs `antigravity --print --add-dir <sandbox>` and
  uses the **logged-in Google account by default** (the CLI's keyring/sign-in
  flow). For headless runs without an account session, opt into Gemini API
  key authentication with `EVAL_ANTIGRAVITY_KEY_MODE=1` plus
  `GEMINI_API_KEY`: the harness writes `modelProvider: gemini` into the
  sandbox `settings.json` and isolates the CLI's HOME to the sandbox, so
  your global Antigravity configuration stays untouched (official install
  guide: antigravity.google/docs/cli/install). The sandbox is fully
  disposable, so the driver uses `--dangerously-skip-permissions` to run
  headless.

  Account mode keeps the real HOME, so the evaluated agent sees the
  developer's global `~/.gemini` skills, plugins, and built-ins (for example
  the firebase and chrome-devtools plugin skills) in its available-skill
  list. That pollutes trigger-selection context and makes account-mode runs
  differ from an isolated key-mode run; prefer the opt-in key mode for
  reproducible runs, or restrict global skills.

Because the evaluated model can read its own environment, credential-like
variables (`*KEY*`, `*TOKEN*`, `*SECRET*`, `*PASSWORD*`, `*CREDENTIAL*`) are
filtered from every driver child process. Only the antigravity key-mode
process receives `GEMINI_API_KEY`, and only from the harness's own process;
commit nothing that would let a model echo a secret into a transcript.

When the driver binary is absent or, for antigravity, the Gemini project
behind `GEMINI_API_KEY` has exhausted its credits (HTTP 429), scenarios record
`skipped` or `infrastructure_error` with the concrete reason instead of a
failure.

### Either-pass policy across drivers

Running multiple drivers (`--host opencode,antigravity`) executes them
concurrently per scenario. The scenario gate passes when **at least one driver
produced a deterministic pass**; deterministic failures in the report stay
recorded and visible but block the gate only when no driver passed;
infrastructure errors surface when neither driver passed nor failed. This
keeps local verification usable when one provider is rate-limited or down
(Issue 173 decision record).

### GitHub-dependent scenarios

Process-layer skills (`create-issue`, `implement-issue`, `create-pr`,
`review-pr`, `fix-pr`) interact with GitHub. Live evaluation of those
scenarios and of the `artifact-flow` e2e stages requires a documented sandbox
repository and is otherwise recorded as `skipped` (reason
`sandbox_repo_not_configured`). A sandbox repository is an environment
concern, not part of this corpus.

Set `EVAL_GITHUB_REPO=owner/repo` to configure one: the harness registers
the repository as the sandbox Git origin (`origin` →
`https://github.com/<owner/repo>.git`) and passes `GH_REPO` to the driver
children, so `git` and `gh` resolve the same target. The sandbox itself
stays a throwaway local clone of the fixtures; skills and their prompts
never receive credentials, and the harness's own `gh` calls keep the
developer's authentication.

## Deterministic versus rubric

Deterministic assertions (`expectations`) are machine-checked and gate the
verdict. Rubric dimensions (`rubric.md`) are reviewed by a bounded subagent in
live runs, scored 1–5, and recorded separately; they never convert a
deterministic failure into a pass. A re-run of an unchanged scenario must
produce identical deterministic verdicts and bounded rubric variance.