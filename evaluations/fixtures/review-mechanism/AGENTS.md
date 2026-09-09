# Ledger project guidance

- Change work is governed: report, issue, plan, implementation, pull request, review.
- The ledger policy change adds `internal/policy/ledger_guard.go` around the small artifact in `src/ledger.go`.
- Only the implementation stage edits source files. Keep the policy guard in scope so the review stage can inspect the complete change.
- Pull requests are reviewed before merge; findings return to the implementation branch.
