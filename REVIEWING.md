# Review Policy

This repository operates in solo review mode unless a future repository change explicitly declares team review mode.

- Require a completed `review-pr` self-review artifact before merging.
- The self-review artifact must identify the reviewed head SHA, applied criteria, validation evidence, and findings result.
- Never treat the PR author's self-review as a third-party approval.
- Continue to enforce the live GitHub Ruleset, required status checks, protected-branch policy, and rebase-only merge policy.
- If the live Ruleset requires an eligible external approval, team-mode requirements take precedence and the merge must stop until that approval exists.
