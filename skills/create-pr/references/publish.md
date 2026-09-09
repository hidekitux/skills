# Publish and synchronize

Read this reference after validation passes and the branch is ready to publish.

## Push

- Commits are authored by `implement-issue` during implementation. Do not stage, create, or amend commits here; push the resolved head branch as it is.
- Push the resolved Issue branch before creating or updating the Pull Request. Never push to a protected base branch.
- If an author-owned branch was rebased, use `--force-with-lease` only after resolving the exact remote branch and obtaining any required approval. Never use plain `--force`.
- After a rebase changes the base revision, reread repository instructions, the Pull Request template, and branch policy before drafting the body.

## Create or update

- Use the repository Pull Request template. If none exists, use `Issue`, `Summary`, and `Validation` in that order.
- Use `[Type]: Summary` in sentence case, with an imperative first verb. Use `[Release]: vX.Y.Z` only for releases.
- Make `## Issue` the first section. Put standalone Issue references directly below it, with the branch Issue first and `Closes #<number>` for each change Issue. Use `Tracks #<number>` only for release work.
- Summarize observable behavior and scope, then list exact validation commands and outcomes. Begin ordinary English sentences and list items with a capital letter while preserving literal names and commands.
- Include conditional checklists only when applicable, and mark allowed non-applicable items instead of hiding evidence.
- Validate the exact body with `go run ./cmd/validate-branch-policy --base <base> --head <head> --body "$final_body"` and the exact title with `go run ./cmd/validate-work-item-title --title "$final_title"` before the GitHub API call. Read the validators and tests before diagnosing a preflight failure.
- Do not update Project Status here; the trusted `Policy (Project)` workflow owns Pull Request transitions. Create a ready Pull Request only after the ready gate passes; otherwise create or retain a draft and state every limitation.
