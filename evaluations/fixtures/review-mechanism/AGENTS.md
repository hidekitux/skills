# Ledger project guidance

- Change work is governed: report, issue, pull request, review.
- The ledger policy change is already implemented and validated. It adds `internal/policy/ledger_guard.go` around the small artifact in `src/ledger.go` and is present in the working tree, not yet committed.
- The change still needs its change issue, an Issue branch, and a pull request. Open the pull request with the complete working-tree change so the review stage can inspect it.
- Pull requests are reviewed before merge; findings return to the implementation branch.
