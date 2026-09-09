# Tidy and publish a fix

Read this reference when unpushed history needs cleanup or after a fix passes validation.

## Tidy unpushed history

- Compare the local branch with its remote tracking ref before rewriting. Drop, reword, or combine only unpushed commits that no longer match the plan or findings.
- Never rewrite a reviewed remote commit without explicit authorization. Record pre-tidy and post-tidy heads and the reason for each change.
- Inspect the complete base diff after tidying and confirm that only the intended review fix remains.

## Push and synchronize

- Push the resolved head with `--force-with-lease` when tidying rewrote unpushed commits; never use plain `--force` or push a protected base branch.
- After pushing, synchronize the Pull Request body so it describes the published head. Keep the opening Issue-reference block unchanged and first.
- Validate the exact body with `go run ./cmd/validate-branch-policy --base <base> --head <head> --body "$final_body"` before the API call.
- Do not update Project Status; the trusted `Policy (Project)` workflow owns Pull Request-observable transitions.
- Verify `git rev-parse HEAD` equals `gh pr view <number> --json headRefOid --jq .headRefOid` before handoff. The next owner is `review-pr`.
