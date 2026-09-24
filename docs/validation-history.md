# Validation history

This record counts the CI failures that each validation command caused and
gives each command a keep, merge, or retire recommendation. It is a snapshot
collected at 2026-09-24T09:09:31Z. Runs after that time are not counted. The
tier, trigger, and failure policy of each command stay in
[validation tiers](validation-tiers.md).

## History window and method

The history window runs from the first GitHub Actions run on 2026-08-15 to the
collection time. GitHub still returned the failed-step log of the first failed
run, `31874430258`, so no run in the window was lost to log retention.

The window holds 3,754 runs. The failed runs per workflow file are:

| Workflow file | Failed runs | All runs |
| --- | --- | --- |
| `validate.yml` | 12 | 501 |
| `policy-issues.yml` | 291 | 535 |
| `pr-project-status.yml` | 31 | 533 |
| `policy.yml` | 16 | 414 |
| `policy-signatures.yml` | 13 | 223 |
| `targeted.yml` | 0 | 199 |
| `security.yml` | 0 | 314 |
| `publish.yml` | 0 | 17 |

The record was collected with these commands:

```bash
gh api --paginate "repos/hidekitux/skills/actions/workflows/<file>/runs?per_page=100&status=failure"
gh api "repos/hidekitux/skills/actions/runs/<run-id>/jobs?filter=all&per_page=100"
gh run view <run-id> --log-failed
gh api --paginate "repos/hidekitux/skills/issues/events?per_page=100"
gh api graphql -f query='{repository(owner:"hidekitux",name:"skills"){issue(number:368){timelineItems(first:10,itemTypes:[ADDED_TO_PROJECT_V2_EVENT,PROJECT_V2_ITEM_STATUS_CHANGED_EVENT]){nodes{__typename ... on AddedToProjectV2Event{createdAt} ... on ProjectV2ItemStatusChangedEvent{createdAt status}}}}}}'
git log origin/main -i --grep=<command-name>
```

Each failed run is attributed to every command that its failed steps and
their first `error:` or `--- FAIL` lines name, once per command. Run
`32850226663` names two commands, so the 363 failed runs give 364
attributions. Each attributed failure has one of three kinds:

- A rule rejection: the command rejected input that broke its rule.
- A tool defect: the command failed because of a bug in the command or its
  setup, and a later commit fixed the command rather than the input.
- An infrastructure failure: a GitHub API rate limit, a missing owner type,
  or a job that stopped before any validation step ran.

A validation command is a command under `cmd/`, or a named check inside
`cmd/check-repository`, that reports a rule violation through a non-zero exit
and that `mise run validate:all` or a workflow in `.github/workflows/` runs.

Three groups of commands fall outside that definition:

- `set-issue-project-status` and `collect-badges` run in workflows but write
  Project state or badge data rather than report a rule violation.
- `check-promotion`, `compile-context`, `evaluate`, `evaluate-compaction`,
  `generate-public-status`, `publish-release`, `validate-skill-creator`, and
  `verify-release` run only from their own mise tasks, which neither
  `validate:all` nor a workflow calls.
- `actionlint`, `ruff`, `shellcheck`, `gofmt`, `go vet`, `go test`,
  `gh skill publish --dry-run`, and zizmor are not commands under `cmd/`.

## Failure history per command

| Command | Runs it | Rule rejections | Tool defects | Infrastructure | Recommendation |
| --- | --- | --- | --- | --- | --- |
| `validate-issue-project` | `policy-issues.yml` | 62 | 0 | 5 | Keep |
| `validate-plan-comment` | `pr-project-status.yml` | 25 | 0 | 0 | Keep |
| `validate-pr-commit-signatures` | `policy-signatures.yml` | 13 | 0 | 0 | Keep |
| `validate-work-item-title` | `policy.yml`, `policy-issues.yml` | 6 | 1 | 0 | Keep |
| `lint-commits` | `policy.yml` | 7 | 0 | 0 | Keep |
| `validate-branch-policy` | `policy.yml`, `check:branch-policy` | 2 | 1 | 0 | Keep |
| `validate-issue-body` | `policy-issues.yml` | 1 | 0 | 0 | Keep |
| `verify-fsl` | `verify:fsl` | 0 | 2 | 0 | Keep |
| `check-whitespace` | `check:diff` | 0 | 1 | 0 | Keep |
| The 24 checks in `check-repository` | `check:repository` | 0 | 0 | 0 | Keep |
| `validate-mise-tasks` | `check:tasks` | 0 | 0 | 0 | Keep |
| `validate-hosts` | `check:hosts` | 0 | 0 | 0 | Keep |
| `mutate-fsl` | `targeted.yml`, `publish.yml` | 0 | 0 | 0 | Keep |
| `scan-go-vuln` | `targeted.yml` | 0 | 0 | 0 | Keep |

The 24 checks in `check-repository` are the entries of `repoChecks` in
`cmd/check-repository/main.go`: `validate-repository`,
`check-tool-licenses`, `validate-script-tests`, `check-sensitive-content`,
`check-writing-quality`, `check-mutation-badges`, `check-mutation-triage`,
`check-analyze-readonly`, `check-guided-paths`,
`check-instruction-inventory`, `check-catalog-docs`,
`check-dependabot-config`, `check-module-boundaries`,
`check-contract-decisions`, `check-cutover-record`, `check-public-status`,
`validate-diagnostic`, `check-evaluation`, `check-context`,
`validate-skill-graph`, `validate-skill-trace`, `validate-replay-fixtures`,
`validate-deliberation-policy`, and `validate-execution-strategy`.

The Git-history search found no commit that names one of these commands as the
failure it fixed. Commit messages in this repository describe the change, not
the check that flagged it, so the search cannot count local catches. The
matching commits change the commands or their workflows instead, for example
`3ee5c95` and `1afb081` (`verify-fsl`), `9b825f2` (`check-sensitive-content`),
`1ef692a` (a `lint-commits` test), and `042f6ba` (the `policy.yml` jobs).

## Evidence behind each recommendation

- `validate-issue-project`: 25 rejections report that the Issue has no Project
  item, 34 that its Status is empty, and 3 that its Scope or Priority is
  empty. At least some of these rejections come from ordering. Issue #368 was
  created at 2026-09-24T08:25:28Z. Run `35975157261` started at 08:25:30Z, and
  the check reported the empty Status at 08:26:10Z. The Status became
  `Backlog` at 08:26:33Z. No later event re-ran the check. No other workflow
  step runs `validate-issue-project` on Issue events, so the recommendation is
  to keep the check and fix its timing in a follow-up Issue.
- `validate-plan-comment`: 25 rejections of comments that carried the plan
  marker, across 21 Issues. Issues #232, #234, #283, and #286 each have two
  rejections, which shows authors posting the plan again after a rejection.
- `validate-pr-commit-signatures`: 13 runs on 12 Pull Requests found
  unverified commits. Branch `issue/318` failed twice. No other check reads
  commit signatures.
- `validate-work-item-title` and `lint-commits`: 13 rejections of Pull Request
  titles and commit messages. The local `commit-msg` hook runs
  `validate-commit-message`, not `lint-commits`, so CI is the only point that
  checks a pushed commit series.
- `validate-branch-policy`: 2 rejections, both on branch
  `claude/issue-332-96af6f`, whose Pull Request did not begin with the
  required Issue section. The tool defect is shared with
  `validate-work-item-title`: run `32850226663`, a push to `main`, ran both
  Pull Request-only steps. The title was empty, and `validate-branch-policy`
  printed `error: --base and --head are required`. `042f6ba` then skipped
  those jobs on push events.
- `validate-issue-body`: 1 rejection, in run `33495989699`, of an Issue whose
  level-two headings were out of order.
- `verify-fsl` and `check-whitespace`: every CI failure was a tool defect.
  `verify-fsl` could not find `fslc` on `PATH` until `3ee5c95`. The first push
  to `main` failed in the `diff-check` task, which ran
  `scripts/check-whitespace.sh` before `cmd/check-whitespace` replaced it,
  because the push had no merge base. Both commands still guard their rules,
  and their tool defects are fixed, so the recommendation is keep.
- The zero-failure commands: `.githooks/pre-push` runs `mise run validate:all`
  before every push. A rule rejection from `check:repository`, `check:tasks`,
  or `check:hosts` therefore stops the push on the author's machine and never
  reaches CI. A zero count shows that the local gate works. It does not show
  that the command catches nothing. The recommendation is keep until a record
  of local failures exists. In `targeted.yml`, `mutate-fsl` and `scan-go-vuln`
  run only when their paths change, and neither failed in 199 runs.
  `publish.yml` runs the full `mutate-fsl` weekly regardless of paths, and
  none of its 17 runs failed.

No command in scope has a merge or retire recommendation, so no enforcement
rule loses coverage.

## Failures outside the validation commands

| Source | Failed runs | Kind |
| --- | --- | --- |
| `validate-issue-labels` (retired) | 219 | Rule rejections during a label migration |
| `go test ./...` and the earlier Python tests | 7 | Test failures |
| `gh skill publish --dry-run`, then the `validate-skills` task | 2 | Tool defect: it walked a `ruff` cache directory while `ruff` deleted it |
| `set-issue-project-status` | 5 | Infrastructure (GitHub API rate limit) |
| Jobs with no failed validation step | 5 | Infrastructure |

`validate-issue-labels` was retired in `5ab3a78` (#205). All 219 of its
failures fell between 2026-08-27T02:02:50Z and 02:17:44Z. In that window the
repository's Issue event log shows 217 `unlabeled` events, as labels were
removed in favor of Project fields. `set-issue-project-status` writes Project
state rather than validating it, so it is not a validation command.

## Run IDs

- `validate-issue-project` rule rejections: `33074457320`, `33094442957`,
  `33112941390`, `33117066321`, `33121652432`, `33136708643`, `33138511159`,
  `33454030120`, `33495136674`, `33495174647`, `33495207514`, `33496060204`,
  `33501749578`, `33652914077`, `33663978491`, `33663986643`, `33664428805`,
  `33664463568`, `33664948964`, `33665363320`, `33665666430`, `33665891577`,
  `33668085110`, `34500355422`, `34500355489`, `34500358474`, `34502497240`,
  `34502501957`, `34502503902`, `34502507026`, `34502508644`, `34547073188`,
  `34547074703`, `34605526268`, `34610457097`, `34720183433`, `34723093217`,
  `34724334271`, `34724334661`, `34724334924`, `34870241707`, `34870264206`,
  `34870266961`, `34870270859`, `34870277084`, `34870280597`, `34870286747`,
  `35319555768`, `35319564754`, `35319566600`, `35319568495`, `35319569821`,
  `35319572409`, `35319616193`, `35944626259`, `35944627357`, `35944628155`,
  `35950109536`, `35962136748`, `35975147352`, `35975154520`, `35975157261`.
- `validate-issue-project` infrastructure: `33040409943`, `33577297906`,
  `33668109579`, `34293523694`, `34293523990`.
- `validate-plan-comment`: `33468704169`, `33473044846`, `33506049307`,
  `33532634675`, `33653969631`, `33783105997`, `34294550283`, `34299735702`,
  `34304078948`, `34310113532`, `34313983207`, `34315454134`, `34315554888`,
  `34320184827`, `34334785893`, `34337024879`, `34601990600`, `34611271964`,
  `34611451814`, `34622653778`, `34622850157`, `34630169135`, `34718299521`,
  `34869312422`, `35956200373`.
- `validate-pr-commit-signatures`: `33048881645`, `33093843006`,
  `33094404170`, `34261348183`, `34574974917`, `34603619795`, `34608155187`,
  `34620168515`, `34737520261`, `34768090126`, `34769594706`, `35941145787`,
  `35964153500`.
- `validate-work-item-title` rule rejections: `32868033264`, `32868145040`,
  `32868170643`, `32870787365`, `32870962711`, `32872563765`.
- `validate-work-item-title` and `validate-branch-policy` tool defect:
  `32850226663`.
- `lint-commits`: `34617833291`, `34617970171`, `34618124543`, `34618459456`,
  `34618587004`, `34769595809`, `34769670210`.
- `validate-branch-policy` rule rejections: `35306007222`, `35306226131`.
- `validate-issue-body`: `33495989699`.
- `verify-fsl`: `32811688191`, `32812690604`.
- `gh skill publish --dry-run`: `31878114506`, `31889492541`.
- `check-whitespace`: `31874430258`.
- Tests: `31925057359`, `31939298672`, `31953815807`, `31955022328`,
  `32813143520`, `34779886998`, `35294120870`.
- `set-issue-project-status`: `33670258620`, `34294827958`, `34602122953`,
  `34608155134`, `34610305320`.
- Jobs with no failed validation step: `32985505322`, `32985583281`,
  `32985594073`, `32985740514`, `34770184872`.
