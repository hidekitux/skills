# Release procedure

Treat one GitHub Release as one semantic version. Every skill `version` in `CATALOG.yml` must match the version portion of that release tag. For the skill-set map, layer vocabulary, and where this guide sits, see the [README](../README.md#skill-set-map).

This procedure owns a Release Issue from its creation to its closing. `plan-issue` and `deliver-change` exclude release work, so follow these steps instead. `create-issue`, `create-pr`, and `merge-pr` each run one step below.

The public status and pinned-installation claims in the README are documented only from retained release evidence (`docs/release-evidence.yml`). See [docs/public-skill-status.md](public-skill-status.md) for the evidence contract enforced by `check:repository`.

## Release flow

One Release Issue uses two Pull Requests from the same `issue/<number>` branch:

1. The preparation Pull Request aligns `CATALOG.yml` with the target version. Its body opens with `Tracks #<number>`, so its merge leaves the Release Issue open.
2. The maintainer publishes the release from an up-to-date local `main`.
3. The evidence Pull Request records the published tag and its tag target commit in `docs/release-evidence.yml`. Its body opens with `Closes #<number>`, so its merge closes the Release Issue.

The evidence must come after publication. A commit cannot record its own hash, and the repository allows only rebase merges, so a commit gets a new hash when it reaches `main`. The tag target exists only after the preparation Pull Request is merged.

## Open the Release Issue

1. Create the Release Issue with `create-issue`, titled `[Release]: vX.Y.Z`, with the changelog its [Release Issue rules](../skills/process/create-issue/references/release-issues.md) require.
2. Create `issue/<number>` from the current `origin/main`.

## Prepare the release

1. Align every `CATALOG.yml` skill version with the target `vX.Y.Z`. `check:repository` accepts a catalog version newer than the released tag in `docs/release-evidence.yml`, so the previous evidence stays in place.
2. Review the Todo List, validation results, and changes.
3. Run `mise run validate:all` from the repository root. It verifies installation for both Codex and Claude Code. When `skill-creator` is available in Codex, also run `mise run validate:skill-creator`.
4. Run `mise run check:promotion` when any catalog entry is `stable`. It checks that the retained evidence carries the input digest of the current revision (see [Evidence freshness](evaluation.md#evidence-freshness)), two recent complete runs, required scenario coverage, rubric floor, regression handling, and bounded variance. A reviewer remains responsible for the judgment behind the rubric scores.
5. Run the pinned Go vulnerability scan with `mise run check:go-vuln -- -out <retained-json-path>` and review reachable findings before release. Fix the affected Go version or module, or record an explicit reviewed exception with remediation evidence; non-reachable findings remain reported in the retained output.
6. When `specs/**/*.fsl` or `skills/**/specs/*.fsl` changed, run `mise run mutate:fsl` and review survivors. The README FSL mutation badges and test badges refresh automatically after the change reaches `main`: the Publish workflow reruns `mise run mutate:fsl` and `mise run test:all`, publishes the six shields.io payloads to the `badge-data` branch, and the endpoint badges update without a manual edit. `mise run check:repository` verifies only that the six README badges point at the `badge-data` branch payloads.
7. Commit the release contents and open the preparation Pull Request with `create-pr`. Title it `[Release]: vX.Y.Z` and start its body with an `## Issue` section that holds only `Tracks #<number>`.
8. Merge the preparation Pull Request with `merge-pr`. The Release Issue stays open.

## Publish from main

1. Switch to `main`, run `git pull --ff-only origin main`, and confirm that `git status --porcelain` prints nothing and that `git rev-parse HEAD` equals `git rev-parse origin/main`. Keep this commit hash; it is the verified commit.
2. Run `mise run verify:release -- vX.Y.Z`. It repeats the promotion check, then checks the tag format, every catalog version, the working and committed Git state, and the origin remote. It neither creates a tag nor reuses an existing local tag, and it fails when tracked or untracked changes are present.
3. Publish with `mise run publish:release -- vX.Y.Z`. The task re-runs `mise run validate:all`, the available `skill-creator` validation, and `mise run verify:release`, then runs `gh skill publish --tag vX.Y.Z`. It cannot technically prevent direct execution by a user with shell and GitHub permissions, so use this task as the standard publication entry point.
4. Run `git fetch --tags origin` and `git rev-list -n 1 vX.Y.Z`. The output is the tag target, and it must equal the verified commit from step 1.

`gh skill publish --tag` pushes the current branch and names it as the tag target. It first pushes any commits of the current branch that the remote branch lacks, as `HEAD:refs/heads/<branch>`. It then creates the GitHub Release with `target_commitish` set to the current branch name, and GitHub creates the tag at the head of that branch on `origin`. From an up-to-date `main`, the command pushes nothing, and the tag target is the `origin/main` commit. From any other branch, the command prints `Publishing from branch`, pushes that branch, and tags its head. `v0.1.0` was published from `claude/issue-366-68780c` this way. The GitHub Release `targetCommitish` field keeps the branch name, not the commit, so read the tag target with `git rev-list` instead.

When the tag target differs from the verified commit, `main` moved between steps 1 and 3. Do not record the verified commit. The tag is immutable, so report the difference on the Release Issue and decide there whether to correct it with a new patch release.

`mise run check:skills` (the `gh skill publish --dry-run` gate) warns when no active tag-target Ruleset protects release tags. Keep the `Protect release tags` Ruleset active so published `v` tags stay immutable.

## Record the evidence

1. Verify the published tag and the GitHub Release contents.
2. Install a pinned version in a consuming project for each host, for example `gh skill install hidekitux/skills create-issue --pin vX.Y.Z --agent codex --scope project` and the same command with `--agent claude-code`.
3. Recreate `issue/<number>` from `origin/main`. The repository deletes the head branch when a Pull Request is merged.
4. Update `docs/release-evidence.yml`: set `released: true`, and record the `tag`, the deterministic GitHub Release URL (`https://github.com/hidekitux/skills/releases/tag/<tag>`), and the tag target from `git rev-list -n 1 vX.Y.Z` as `commit`. Add one `pinned_installation` entry per host with the `host`, `command`, `installed_path`, `result`, and `date` of the installation from step 2. No check validates `pinned_installation`.
5. Run `mise run generate:public-status`, then `mise run validate:all`. `check:repository` fails when the released tag is newer than a catalog version or when the record is partly filled.
6. Open the evidence Pull Request with `create-pr`, titled `[Release]: vX.Y.Z`, and start its body with an `## Issue` section that holds only `Closes #<number>`.
7. Merge the evidence Pull Request with `merge-pr`. GitHub closes the Release Issue, and the Project item reaches `Done`.

## Version rules

- Use minor for backward-compatible features, patch for fixes, and major for breaking changes.
- The repository publishes every skill under one tag, so align every `CATALOG.yml` entry with the tag even when only one skill changed.
- Never overwrite a published tag; publish a new patch release to correct it.

## Correcting a release (rollback)

Release tags are immutable. The repository's active `Protect release tags` tag-target Ruleset blocks deleting or force-moving any `v*` tag. The Ruleset has deletion and non-fast-forward restrictions, active enforcement, and no bypass actors. A published version therefore always points at the same verified commit and cannot be removed to undo a release. Once a tag is published, tag replacement is not available. Correct a bad release with a new patch version under the rules below.

To correct or roll back a bad release:

1. Leave the released tag (and its GitHub Release) untouched.
2. Open a Release Issue for the next patch version, such as `v0.1.1`, and follow this procedure from [Open the Release Issue](#open-the-release-issue).
3. Install the corrected pinned version in consuming projects.

The faulty release remains installable and immutable by design; consumers migrate by pinning the corrected version. Never delete, move, or reuse a release tag.
