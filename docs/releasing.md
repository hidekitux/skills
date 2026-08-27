# Release procedure

Treat one GitHub Release as one semantic version. Every skill `version` in `CATALOG.yml` must match the version portion of that release tag. For the skill-set map, layer vocabulary, and where this guide sits, see the [README](../README.md#skill-set-map).

The public status and pinned-installation claims in the README are documented only from retained release evidence (`docs/release-evidence.yml`); the steps below record that evidence as part of the release. See [docs/public-skill-status.md](public-skill-status.md) for the evidence contract enforced by `check:repository`.

## Release steps

1. Align every `CATALOG.yml` skill version with the target `vX.Y.Z`.
2. Review the Todo List, validation results, and changes.
3. Run `mise run validate` from the repository root. It verifies installation for both Codex and Claude Code. When `skill-creator` is available in Codex, also run `mise run validate-skill-creator`.
4. When `specs/**/*.fsl` or `skills/**/specs/*.fsl` changed, run `mise run mutate-fsl` and review survivors. The README FSL mutation badges and test badges refresh automatically after the change reaches `main`: the Publish workflow reruns `mise run mutate-fsl` and `mise run test`, publishes the six shields.io payloads to the `badge-data` branch, and the endpoint badges update without a manual edit. `mise run check:repository` verifies only that the six README badges point at the `badge-data` branch payloads.
5. Update `docs/release-evidence.yml` to `released: true` and record the verified `tag`, the deterministic GitHub Release URL (`https://github.com/hidekitux/skills/releases/tag/<tag>`), and the verified `commit`. `check:repository` then fails unless the release tag version matches every catalog version.
6. Commit the release contents.
7. Run `mise run verify-release -- vX.Y.Z` from the committed state. It checks the tag format, every catalog version, the working and committed Git state, and the origin remote. It neither creates a tag nor reuses an existing local tag, and it fails when tracked or untracked changes are present.
8. Publish with `mise run release:publish -- vX.Y.Z`. The task re-runs `mise run validate`, the available `skill-creator` validation, and `mise run verify-release`, then runs `gh skill publish --tag vX.Y.Z`. It cannot technically prevent direct execution by a user with shell and GitHub permissions, so use this task as the standard publication entry point.

## After publication

Verify the published tag and GitHub Release contents, and install a pinned version such as `skill-name@vX.Y.Z` in consuming projects for Codex and Claude Code. Record the pinned-installation results in `docs/release-evidence.yml` so the README public status section can document them from retained evidence.

`mise run check:skills` (the `gh skill publish --dry-run` gate) warns when no active tag-target Ruleset protects release tags. Keep the `Protect release tags` Ruleset active so published `v` tags stay immutable.

## Version rules

- Use minor for backward-compatible features, patch for fixes, and major for breaking changes.
- The repository publishes every skill under one tag, so align every `CATALOG.yml` entry with the tag even when only one skill changed.
- Never overwrite a published tag; publish a new patch release to correct it.

## Correcting a release (rollback)

Release tags are immutable. The repository's active `Protect release tags` tag-target Ruleset blocks deleting or force-moving any `v*` tag (deletion and non-fast-forward restrictions, active enforcement, no bypass actors), so a published version always points at the same verified commit and cannot be removed to undo a release. Once a tag is published, tag replacement is not available; a bad release is corrected with a new patch version under the rules below.

To correct or roll back a bad release:

1. Leave the released tag (and its GitHub Release) untouched.
2. Align every `CATALOG.yml` version to the next patch version and commit.
3. Run the pre-release checks and `mise run release:publish -- vX.Y.Z` for the new version (for example `v0.1.1`).
4. Install the corrected pinned version in consuming projects.

The faulty release remains installable and immutable by design; consumers migrate by pinning the corrected version. Never delete, move, or reuse a release tag.
