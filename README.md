# hidekitux/skills

![FSL mutants killed](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ffsl-killed.json)
![FSL kill rate](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ffsl-kill-rate.json)
![FSL surviving mutants](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ffsl-survived.json)
![License](https://img.shields.io/github/license/hidekitux/skills)
![FSL verifier](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ffslc-version.json)
![Validation](https://img.shields.io/github/actions/workflow/status/hidekitux/skills/validate.yml?branch=main)
![Tests status](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ftests-status.json)
![Tests run](https://img.shields.io/endpoint?url=https%3A%2F%2Fraw.githubusercontent.com%2Fhidekitux%2Fskills%2Fbadge-data%2Ftests-run.json)

This repository is the source of reusable Agent Skills for individuals and teams. Skills follow the [Agent Skills](https://agentskills.io/specification) format and are published and installed with `gh skill`.

## License

The repository and published skills use the [Apache License 2.0](LICENSE). The `license: Apache-2.0` field in each `SKILL.md` and the root `LICENSE` are authoritative. [NOTICE](NOTICE) records the copyright owner and covered years. Keep the standard `LICENSE` text unchanged; update `NOTICE` only when copyrightable material is added or materially updated.

## Development

Use [mise](https://mise.jdx.dev/) as the standard command entry point. Trust the configuration and run `setup:all` once to prepare local skills for Codex and Claude Code, Git hooks, and project-local commitlint. `setup:all` is safe to rerun, and it does not need to be run again by hand on every branch switch: the tracked `post-checkout` hook reruns it automatically.

```bash
mise trust
mise run setup:all
mise tasks ls
```

| Workflow | Command |
| --- | --- |
| Initial setup | `mise run setup:all` |
| Full repository validation | `mise run validate:all` |
| Fast local-change check | `mise run check:local` |
| Static analysis | `mise run lint:all` |
| FSL verification | `mise run verify:fsl` |
| FSL mutation check | `mise run mutate:fsl` |
| Behavioral evaluation smoke set | `mise run evaluate:smoke` |
| Behavioral evaluation suite | `mise run evaluate:all` |
| Release-candidate verification | `mise run verify:release -- vX.Y.Z` |
| Publish a verified release | `mise run publish:release -- vX.Y.Z` |

`setup:all` enables `.githooks`. It reruns automatically on branch checkout; `check:local` runs before commits and `validate:all` before pushes. A failed check blocks the corresponding commit or push.

## Worktrees

Codex and Claude Code worktrees cannot check out the same branch more than once. The primary worktree owns `main`, so creating another worktree on `main` fails. `mise run setup:all` registers skills for the checked-out snapshot and reuses the pinned commitlint from the shared Git directory, so worktrees setting up in parallel do not rebuild or conflict with each other; the tracked `post-checkout` hook runs it for every new worktree.

The repository uses [`worktrunk`](https://github.com/max-sixty/worktrunk) (`wt`) as the worktree tool. Install it once per machine with `mise use -g worktrunk` and run `wt config shell install`; it is a local developer convenience, not a repository or CI dependency. See [docs/worktrees.md](docs/worktrees.md) for the worktree policy and commands.

```bash
wt switch --create issue/<number>   # create the Issue branch and its worktree
wt list                             # show which worktree owns which branch
wt remove issue/<number>            # remove an inspected, inactive worktree
```

No worktree is removed automatically. Inspect changes with `git status` first, and remove one only after deciding it is no longer active. `wt remove` refuses a worktree with uncommitted changes and keeps an unmerged branch; never reach for `wt remove --force` or `wt remove -D` to work around either. Do not run development commands from a bare repository entry point; use a registered non-bare worktree from `wt list`.

Native `git worktree` remains supported when `worktrunk` is unavailable. Use a detached worktree for a read-only `main` snapshot. For changes, create a branch from an existing Issue instead of checking out `main` again.

```bash
git worktree add --detach <path> origin/main
git worktree add -b issue/<number> <path> origin/main
```

Never use `--force` to check out `main` in multiple worktrees.

`mise run validate:all` validates temporary installation for both Codex and Claude Code. When `skill-creator` is available in Codex, also run `mise run validate:skill-creator`. Linux x64 and macOS Apple Silicon are supported for full validation because it includes FSL verification.

## Layout and skill contract

```text
skills/<skill-name>/SKILL.md
skills/<namespace>/<skill-name>/SKILL.md
```

Every skill requires `SKILL.md`; its `name` matches the parent directory and uses lowercase letters, digits, and hyphens. Add `scripts/`, `references/`, and `assets/` only when reusable resources are needed. Repository automation is implemented as Go commands under `cmd/` with shared packages under `internal/`; the retained shell helpers live under `scripts/fsl/` and `scripts/setup/`.

Every published skill creates and maintains a Todo List at invocation start. Include discovery, scope confirmation, implementation, validation, and handoff where applicable. Use a host-native list when available, otherwise an equivalent Markdown checklist. Complete an item only when evidence exists and explain unfinished items at handoff.

Every skill belongs to one of four layers — process, analyze, fix, or govern. See [Skill layers](docs/skill-layers.md) for the layer model and the skill-set mapping, and [Analysis skill common contract](docs/analysis-skill-common.md) for the shared analyze-* core design. Outcome-oriented entry points (`improve-project`, `deliver-change`, `resolve-defect`) coordinate the primitives from a user outcome to a complete result; direct primitive invocation remains available for advanced or partial workflows.

## Skill-set map

The repository publishes 15 skills today and tracks 0 planned next-generation skills. Presence in the `skills:` list of [`CATALOG.yml`](CATALOG.yml) is the current publishable inventory; each entry's `layer` and `status` fields drive the layer and status documentation below and in [Skill layers](docs/skill-layers.md). [Skill layers](docs/skill-layers.md) is the authoritative layer model with the full mapping and feature Issues; the table below summarizes which layer every current skill belongs to. Use the layer vocabulary — process, analyze, fix, and govern — consistently in Issues, docs, and the authoring brief.

| Skill | Layer | Status |
| --- | --- | --- |
| create-issue | process | experimental |
| plan-issue | process | experimental |
| implement-issue | process | experimental |
| create-pr | process | experimental |
| review-pr | process | experimental |
| merge-pr | process | experimental |
| improve-project | process | experimental |
| deliver-change | process | experimental |
| analyze-project | analyze | experimental |
| debug-code | fix | experimental |
| resolve-defect | fix | experimental |
| write-tests | fix | experimental |
| refactor-code | fix | experimental |
| bootstrap-project | govern | experimental |
| audit-workflow-enforcement | govern | experimental |

Where the related guides live:

- [docs/skill-layers.md](docs/skill-layers.md) — the layer model, naming pattern, boundaries, and the skill-set mapping.
- [docs/analysis-skill-common.md](docs/analysis-skill-common.md) — the shared analyze-* core contract.
- [docs/skill-contract.md](docs/skill-contract.md) — the cross-skill handoff and boundary rules: which skill owns each phase, where every skill sends its result, and the debug, review, and analyze-to-change loops.
- [docs/evaluation.md](docs/evaluation.md) — outcome-based behavioral evaluation, the promotion threshold for catalog status, and how regressions block promotion.
- [docs/skill-brief-template.md](docs/skill-brief-template.md) — the authoring brief, including boundaries, related skills, and handoff targets.
- [docs/releasing.md](docs/releasing.md) — the release procedure.
- [docs/worktrees.md](docs/worktrees.md) — the worktree policy, commands, and setup.
- [docs/fsl.md](docs/fsl.md) — the FSL specification boundary and verification.
- [docs/model-selection.md](docs/model-selection.md) — role-tier model selection.
- [docs/model-routing.md](docs/model-routing.md) — how each host consumes and verifies the selected models.

<!-- BEGIN generated: public-status -->

## Public status

This section is generated from `CATALOG.yml` and `docs/release-evidence.yml` by `mise run generate:public-status` and mechanically verified by `check:repository`. Do not edit by hand. See [`docs/public-skill-status.md`](docs/public-skill-status.md) for the authority and evidence contract.

### Outcome-oriented entry points

The evaluated entry points coordinate the primitive skills from a user request to a complete observable result; direct primitive invocation remains available for advanced or partial workflows.

| Entry point | Outcome | Status | Version |
| --- | --- | --- | --- |
| `improve-project` | Improve a project end to end from read-only analysis through an Issue-backed, reviewed change. | experimental | 0.1.0 |
| `deliver-change` | Deliver a governed Change Issue end to end from its verified plan through a reviewed Pull Request. | experimental | 0.1.0 |
| `resolve-defect` | Resolve a verified defect from reproduction through fix, regression tests, and any required governed change. | experimental | 0.1.0 |

### Preview stability

Every cataloged skill is `experimental` until behavioral and release evidence qualify promotion ([docs/evaluation.md](docs/evaluation.md)).

No verified release exists yet ([Issue #174](https://github.com/hidekitux/skills/issues/174)); until the preview release is published, any `@vX.Y.Z` installation resolves to the default branch head.

### Pinned installation

Pinned installation is documented from retained release evidence only. No verified release exists yet, so no pinned installation is claimed; the release flow ([Issue #174](https://github.com/hidekitux/skills/issues/174)) records the preview tag, commit, and Codex and Claude Code installation results in `docs/release-evidence.yml` before this section can state them.
<!-- END generated: public-status -->

## Development workflow

1. Add `skills/<skill-name>/SKILL.md`.
2. Record its purpose, owner, and supported agents in `CATALOG.yml`.
3. Run `mise run validate:all` before publishing.
4. Run `mise run validate:skill-creator` when it is available in Codex.
5. Follow the [release procedure](docs/releasing.md) after review.

`mise run check:repository` checks catalog entries, Apache-2.0 metadata, host adapters, the Todo List contract, known secrets, private URLs, user paths, tool-license evidence, the script-to-test mapping, and catalog-versus-documentation drift. Use `skill-creator` for new or substantially updated skills when available; otherwise complete the [skill creation brief](docs/skill-brief-template.md) and run the common validation.

## Installation and compatibility

`gh skill` supports Codex and Claude Code. Install the latest catalog snapshot for both hosts:

```bash
gh skill install hidekitux/skills --all --agent codex --scope user
gh skill install hidekitux/skills --all --agent claude-code --scope user
```

Use `--scope project` for a project install and `skill-name@vX.Y.Z` to pin a release. `--agent` selects one host, so run each command to install for both.

Pinning changes how an install resolves:

- **Unpinned** (`hidekitux/skills`) resolves to the default branch head, so it tracks the latest unreleased catalog snapshot.
- **Pinned** (`skill-name@vX.Y.Z`) resolves to the immutable release tag, so it is reproducible and stable.

Each released tag matches a GitHub Release at `https://github.com/hidekitux/skills/releases` and the `CATALOG.yml` versions for that release. Published `v` release tags are immutable: the repository's active `Protect release tags` tag-target Ruleset blocks deleting or force-moving them, so a released version always points at the same verified commit. To correct a bad release, publish a new patch release; never replace or reuse an existing tag. See [docs/releasing.md](docs/releasing.md) for the release procedure.

Keep one canonical skill under `skills/`. Do not duplicate shared `SKILL.md` content for a host. When a material execution difference exists, add `references/hosts/<host>.md` that states the available capability, preferred path, fallback, and verification method. `agents/openai.yaml` is Codex UI metadata only. The tracked `hosts/` directory contains repository-level examples; `.codex/`, `.claude/`, and `.agents/` are generated local state and are not tracked.

## FSL

FSL verifies state transitions and publication conditions, not `SKILL.md` prose. Place a skill-owned source in `skills/<skill-name>/specs/*.fsl` and expose it with a relative symbolic link at `specs/<skill-name>/`; place repository-owned or cross-cutting sources directly in `specs/`. Confirm a formalization memo before adding or changing a specification, then run:

```bash
mise run verify:fsl
mise run mutate:fsl
```

The FSL and test badges at the top of this README are published to the `badge-data` branch by the [Publish workflow](.github/workflows/publish.yml); they auto-refresh after spec or test changes reach `main`. Do not hand-edit them.

See [docs/fsl.md](docs/fsl.md) for the specification boundary and [docs/releasing.md](docs/releasing.md) for release requirements.
