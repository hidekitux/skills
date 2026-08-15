# Release procedure

Treat one GitHub Release as one semantic version. Every skill `version` in `CATALOG.yml` must match the version portion of that release tag.

## Before release

1. Align every `CATALOG.yml` skill version with the target `vX.Y.Z`.
2. Review the Todo List, validation results, and changes.
3. Run `mise run validate` from the repository root. It verifies installation for both Codex and Claude Code. When `skill-creator` is available in Codex, also run `mise run validate-skill-creator`.
4. When `specs/**/*.fsl` or `skills/**/specs/*.fsl` changed, run `mise run mutate-fsl` and review survivors.
5. Commit the changes.
6. Publish with `mise run release:publish -- vX.Y.Z`. This entry point runs the available `skill-creator` validation, verifies the tag format, catalog versions, and committed state, then publishes.

`verify-release` neither creates a tag nor reuses an existing local tag. It also fails when tracked or untracked changes are present.

## Publish

Run the following from the verified commit:

```bash
mise run release:publish -- vX.Y.Z
```

The task finally runs `gh skill publish --tag vX.Y.Z`. It cannot technically prevent direct execution by a user with shell and GitHub permissions, so use this task as the standard publication entry point. After publication, verify the tag and Release contents, and install a pinned version such as `skill-name@vX.Y.Z` in consuming projects.

## Version rules

- Use minor for backward-compatible features, patch for fixes, and major for breaking changes.
- The repository publishes every skill under one tag, so align every `CATALOG.yml` entry with the tag even when only one skill changed.
- Never overwrite a published tag; publish a new patch release to correct it.
