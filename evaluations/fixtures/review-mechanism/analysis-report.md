# Analysis report: ledger policy enforcement

Prioritized findings for the ledger project:

1. High: ledger entries need a policy guard that rejects negative entries before totals are used. The guard belongs in `internal/policy/ledger_guard.go` and protects `src/ledger.go`.
2. Medium: the policy guard must cover the last entry in the input, because an unchecked final entry can reach the total.

Recommendation: create a governed change issue for finding 1 and review the complete enforcement change before merge.
