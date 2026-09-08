# Review Policy

This repository operates in solo review mode unless a future repository change explicitly declares team review mode.

- Require a completed `review-pr` self-review artifact before merging.
- The self-review artifact must identify the reviewed head SHA, applied criteria, validation evidence, and findings result.
- Review every Pull Request that adds or changes human-readable prose against [docs/writing-style.md](docs/writing-style.md), even when the automated writing check passes. This criterion covers prose in documentation, skills, templates, Issues, Pull Requests, and review comments. It does not cover code, identifiers, commands, paths, or quoted output, which the writing rules exclude.
- Never treat the PR author's self-review as a third-party approval.
- Continue to enforce the live GitHub Ruleset, required status checks, protected-branch policy, and rebase-only merge policy.
- If the live Ruleset requires an eligible external approval, team-mode requirements take precedence and the merge must stop until that approval exists.
