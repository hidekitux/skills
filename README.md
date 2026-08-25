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

Use [mise](https://mise.jdx.dev/) as the standard command entry point. Trust the configuration and run `setup` once to prepare local skills for Codex and Claude Code, Git hooks, and project-local commitlint. `setup` is safe to rerun, and it does not need to be run again by hand on every branch switch: the tracked `post-checkout` hook reruns it automatically.

```bash
mise trust
mise run setup
mise tasks ls
```

| Workflow | Command |
| --- | --- |
| Initial setup | `mise run setup` |
| Full repository validation | `mise run validate` |
| Fast local-change check | `mise run check:local` |
| Worktree diagnosis | `mise run worktree:diagnose -- --branch issue/123` |
| Static analysis | `mise run lint` |
| FSL verification | `mise run verify-fsl` |
| FSL mutation check | `mise run mutate-fsl` |
| Release-candidate verification | `mise run verify-release -- vX.Y.Z` |
| Publish a verified release | `mise run release:publish -- vX.Y.Z` |

`setup` enables `.githooks`. It reruns automatically on branch checkout; `check:local` runs before commits and `validate` before pushes. A failed check blocks the corresponding commit or push.

## Worktrees

Codex and Claude Code worktrees cannot check out the same branch more than once. The primary worktree owns `main`, so creating another worktree on `main` fails. `mise run setup` registers skills for the checked-out snapshot and reuses the pinned commitlint from the shared Git directory, so worktrees setting up in parallel do not rebuild or conflict with each other. Inspect owner and setup state first:

```bash
mise run worktree:diagnose -- --branch issue/123
```

The diagnostic never removes a worktree automatically. Inspect changes with `git status`, and run `git worktree remove <path>` only after deciding it is no longer active. Do not run development commands from a bare repository entry point; use a registered non-bare worktree reported by the diagnostic.

`post-checkout` records the worktree setup result so a later worktree
diagnostic can report whether local setup is current.

Use a detached worktree for a read-only `main` snapshot. For changes, create a branch from an existing Issue instead of checking out `main` again.

```bash
git worktree add --detach <path> origin/main
git worktree add -b issue/<number> <path> origin/main
```

Never use `--force` to check out `main` in multiple worktrees.

`mise run validate` validates temporary installation for both Codex and Claude Code. When `skill-creator` is available in Codex, also run `mise run validate-skill-creator`. Linux x64 and macOS Apple Silicon are supported for full validation because it includes FSL verification.

## Layout and skill contract

```text
skills/<skill-name>/SKILL.md
skills/<namespace>/<skill-name>/SKILL.md
```

Every skill requires `SKILL.md`; its `name` matches the parent directory and uses lowercase letters, digits, and hyphens. Add `scripts/`, `references/`, and `assets/` only when reusable resources are needed. Repository automation is organized by responsibility in `scripts/check/`, `diagnose/`, `fsl/`, `lint/`, `release/`, `setup/`, and `validate/`.

Every published skill creates and maintains a Todo List at invocation start. Include discovery, scope confirmation, implementation, validation, and handoff where applicable. Use a host-native list when available, otherwise an equivalent Markdown checklist. Complete an item only when evidence exists and explain unfinished items at handoff.

Every skill belongs to one of four layers — process, analyze, fix, or govern. See [Skill layers](docs/skill-layers.md) for the layer model and the skill-set mapping, and [Analysis skill common contract](docs/analysis-skill-common.md) for the shared analyze-* core design.

## Skill-set map

The repository publishes seven skills today and tracks three planned next-generation skills. [Skill layers](docs/skill-layers.md) is the authoritative layer model with the full mapping and feature Issues; the table below summarizes which layer every current and planned skill belongs to. Use the layer vocabulary — process, analyze, fix, and govern — consistently in Issues, docs, and the authoring brief.

| Skill | Layer | Status |
| --- | --- | --- |
| create-issue | process | published |
| plan-issue | process | published |
| implement-issue | process | published |
| create-pr | process | published |
| review-pr | process | planned |
| analyze-project | analyze | planned |
| debug-code | fix | published |
| write-tests | fix | planned |
| refactor-code | fix | planned |
| bootstrap-project | govern | published |
| audit-workflow-enforcement | govern | published |

Where the related guides live:

- [docs/skill-layers.md](docs/skill-layers.md) — the layer model, naming pattern, boundaries, and the skill-set mapping.
- [docs/analysis-skill-common.md](docs/analysis-skill-common.md) — the shared analyze-* core contract.
- [docs/skill-contract.md](docs/skill-contract.md) — the cross-skill handoff and boundary rules: which skill owns each phase, where every skill sends its result, and the debug, review, and analyze-to-change loops.
- [docs/skill-brief-template.md](docs/skill-brief-template.md) — the authoring brief, including boundaries, related skills, and handoff targets.
- [docs/releasing.md](docs/releasing.md) — the release procedure.
- [docs/fsl.md](docs/fsl.md) — the FSL specification boundary and verification.
- [docs/model-selection.md](docs/model-selection.md) — role-tier model selection.
- [docs/model-routing.md](docs/model-routing.md) — how each host consumes and verifies the selected models.

## Development workflow

1. Add `skills/<skill-name>/SKILL.md`.
2. Record its purpose, owner, and supported agents in `CATALOG.yml`.
3. Run `mise run validate` before publishing.
4. Run `mise run validate-skill-creator` when it is available in Codex.
5. Follow the [release procedure](docs/releasing.md) after review.

`mise run check:repository` checks catalog entries, Apache-2.0 metadata, host adapters, the Todo List contract, known secrets, private URLs, user paths, tool-license evidence, and the script-to-test mapping. Use `skill-creator` for new or substantially updated skills when available; otherwise complete the [skill creation brief](docs/skill-brief-template.md) and run the common validation.

## Installation and compatibility

`gh skill` supports Codex and Claude Code:

```bash
gh skill install hidekitux/skills --all --agent codex --scope user
gh skill install hidekitux/skills --all --agent claude-code --scope user
```

Use `--scope project` for a project install and `skill-name@vX.Y.Z` to pin a release. `--agent` selects one host, so run each command to install for both.

Keep one canonical skill under `skills/`. Do not duplicate shared `SKILL.md` content for a host. When a material execution difference exists, add `references/hosts/<host>.md` that states the available capability, preferred path, fallback, and verification method. `agents/openai.yaml` is Codex UI metadata only. The tracked `hosts/` directory contains repository-level examples; `.codex/`, `.claude/`, and `.agents/` are generated local state and are not tracked.

## FSL

FSL verifies state transitions and publication conditions, not `SKILL.md` prose. Place a skill-owned source in `skills/<skill-name>/specs/*.fsl` and expose it with a relative symbolic link at `specs/<skill-name>/`; place repository-owned or cross-cutting sources directly in `specs/`. Confirm a formalization memo before adding or changing a specification, then run:

```bash
mise run verify-fsl
mise run mutate-fsl
```

The FSL and test badges at the top of this README are published to the `badge-data` branch by the [Badge data workflow](.github/workflows/badges.yml); they auto-refresh after spec or test changes reach `main`. Do not hand-edit them.

See [docs/fsl.md](docs/fsl.md) for the specification boundary and [docs/releasing.md](docs/releasing.md) for release requirements.
