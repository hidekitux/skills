---
name: update-dependencies
description: Update a project's dependencies, toolchains, and pinned tool versions in verified groups, with the release-note evidence, verification result, and license effect of every version change, and an explicit list of updates held back and why. Use it whenever a user asks to update, upgrade, or bump dependencies, packages, libraries, lockfiles, Go modules, Gradle versions, npm packages, mise tools, or GitHub Action pins, or to act on outdated or vulnerable dependency reports. Hand the verified change to implement-issue; do not use it to rewrite application code for a new API.
license: Apache-2.0
---

# Update Dependencies

## Todo List

1. **in progress:** Resolve the update scope, the ecosystems and manifests in it, and the project's update policy.
2. Inventory the available updates and divide them into update groups.
3. Apply and verify one update group at a time, and record the evidence for each version change.
4. Complete the list only when every update group is verified or held back with a reason; hand off the update record to `implement-issue`.

Keep exactly one item in progress. Mark an item complete only when its
evidence exists. Use the host's native Todo List, or keep the same list as a
Markdown checklist when no native list is available.

## Resolve and Guard

- This is a maintain-layer skill. It changes manifests, lockfiles, toolchain
  files, and version pins in the working tree, and it hands the verified change
  to `implement-issue`, which commits it on the Issue branch. It does not commit,
  create a branch, an Issue, or a Pull Request, or merge anything.
- Resolve the scope from the request or its Issue: which ecosystems, which
  manifests, and whether major versions are allowed. Do not widen it. An update
  outside the scope is a held-back update, not a change.
- Read the project's update policy before changing anything: a document such as
  `docs/tool-update-policy.md`, comments next to pins in `mise.toml` or
  workflow files, and `dependabot.yml` or `renovate.json`. The policy decides
  pinning style, grouping, and which versions are allowed.
- Use the matching technology skill, such as `develop-go` or `develop-kotlin`,
  for the ecosystem's commands and verification.
- Change no application source code. When a version needs a source change, such
  as a renamed API or a new required argument, hold the update back and record
  the change it needs; that source change is implementation work that
  `implement-issue` owns.

## Inventory and group

- List what is outdated with the ecosystem's own command, such as
  `go list -m -u all`, `npm outdated`, `mise outdated`, or the project's Gradle
  version task. Record versions you could not check, such as when the network is
  unavailable.
- Put security fixes first. Form update groups so that a failure points to one
  cause:
  - each major version change on its own;
  - patch and minor changes in one ecosystem together;
  - a toolchain or runtime change, such as Go, the JDK, Node.js, or Gradle, on
    its own.

## Apply and verify each group

For each update group, in order:

1. Read the release notes or changelog for every version between the current
   and the new version. Note each breaking change and deprecation that touches
   the project.
2. Check whether the new version changes the license. A license that the
   project's policy does not allow holds the update back.
3. Apply the change with the ecosystem's command, so the lockfile or checksum
   file changes with the manifest, and keep pin formats, such as a commit SHA
   with a version comment, as the project already writes them.
4. Run the project's verification for that ecosystem. Revert the group and hold
   it back when verification fails because of the update.

## Handoff

Report one row per version change: name, old version, new version, update
group, release-note evidence, license effect, and verification result. List
every held-back update with its reason and the change it would need. Hand the
working-tree change and this record to `implement-issue`; report a blocked
result instead when no group could be verified.

## Writing quality

Use plain, active, evidence-backed prose in the handoff. Name the file,
command, or output behind every claim about the project.
