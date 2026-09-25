# Public skill status

This document defines the authoritative fields and evidence for the
release-backed public skill status published in [../README.md](../README.md).
It is the contract enforced by the `check:repository` **check-public-status**
check and regenerated with `mise run generate:public-status`.

The repository keeps two documentation views with different owners:

- **Contributor-facing inventory** — how the repository is laid out today:
  current publishable-skill inventory, layers, and repository statuses. It is
  owned by Issue 182 and driven by `CATALOG.yml` through the `check-catalog-docs`
  check (the README skill-set map and `docs/skill-layers.md`).
- **Public release-backed status** — what users can rely on from a verified
  release: the evaluated outcome-oriented entry points, lifecycle status,
  version, preview stability, and pinned installation. This document owns that
  view.

The two views are separate: a skill can be present in the contributor-facing
inventory while the public view has not yet verified its release or
installation behavior.

## Authority and evidence

The public status section is generated from two committed sources, each with
a single owner:

| Field | Authority | Evidence |
| --- | --- | --- |
| Entry-point set | The three outcome-oriented entry points named by Issue 175: `improve-project`, `deliver-change`, and `resolve-defect`. | merged entry-point contract and behavioral evaluation (`docs/evaluation.md`) |
| Entry-point summary, lifecycle status, version | `CATALOG.yml` `summary`, `status`, and `version` fields of the entry-point skills. | committed catalog |
| Lifecycle status of every cataloged skill | `CATALOG.yml` `status` field. | committed catalog; promotion to `stable` additionally requires retained behavioral-evaluation reports (`docs/evaluation.md`) |
| Verified release, preview stability, pinned installation | `docs/release-evidence.yml` (`released`, `tag`, `release_url`, `commit`, and the optional `pinned_installation`). | the evidence Pull Request of the release procedure ([docs/releasing.md](releasing.md)) records these after `mise run publish:release` succeeds |

## Fields and evidence rules

- **Lifecycle status**: the catalog `status` value. `experimental` is a
  preview; `stable` requires retained behavioral-evaluation evidence enforced
  by `check-evaluation`.
- **Version**: the catalog `version` value of each entry point. When a
  release exists, every catalog version must equal or follow the version
  portion of the released tag. A merged preparation Pull Request raises the
  catalog before the next tag exists, and `mise run verify:release` checks
  that the next tag equals every catalog version.
- **Public entry-point status**: the catalog `status` of each entry-point
  skill. The three entry points are always listed, and direct primitive
  invocation stays documented for advanced or partial workflows (Issue 175).
- **Preview stability**: derived from the catalog statuses and the release
  evidence. Without `released: true`, the documentation must not claim a
  verified release or pinned installation behavior.
- **Pinned installation**: documented only from retained release evidence.
  The pinned commands use the verified release tag.
- **Pinned-installation results**: the evidence Pull Request records one
  `pinned_installation` entry per host in `docs/release-evidence.yml`, with
  the `host`, `command`, `installed_path`, `result`, and `date` of a pinned
  installation for Codex and for Claude Code. The key is optional, and no
  check validates it: `check-public-status` neither reads it nor renders it.

## Generation and rejection rules

1. Run `mise run generate:public-status` to rewrite the README
   `<!-- BEGIN generated: public-status -->` block from the two authorities.
   The command is idempotent: rerunning it produces no diff.
2. `check:repository` fails when the committed block differs from a fresh
   render, when `docs/public-skill-status.md` is missing, or when the release
   evidence contradicts the catalog (for example a `released: true` tag newer
   than a catalog version, or a partially filled evidence record).
3. Stale derived documentation is rejected; checks never silently repair it.

## Out of scope

- Reimplementing the contributor-facing inventory, layer, or contract
  documentation owned by Issue 182.
- Conceptual guidance that requires editorial judgment.
- Marking skills stable without behavioral and release evidence.
- Changing skill behavior, creating entry points, or publishing a release
  (release work belongs to the release flow and Issue 174).