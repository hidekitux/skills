# Verified implementation plan for the VAT change issue

Ordered tasks, each with observable evidence:

1. Add an `applyTax` helper to `src/currency.go` that applies the configured VAT rate.
   Evidence: symbol present in the committed file.
2. Add a VAT boundary test to `src/currency_test.go`.
   Evidence: test case committed.
3. Run `go test ./...` and record the output.
   Evidence: passing test log.

Out of scope: `docs/roadmap.md`, the README tagline, and any release work.