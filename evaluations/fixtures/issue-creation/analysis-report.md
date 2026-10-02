# Analysis report: VAT on checkout totals

Prioritized findings for the widgets project (read-only investigation).

1. High: `src/currency.go` computes checkout totals without VAT, so orders are undercharged.
   Evidence: the VAT case added below fails under `go test ./...`.
2. Medium: `docs/roadmap.md` still describes the legacy pricing model and contradicts `src/currency.go`.
3. Low: no regression tests cover the VAT boundary.

Recommendation: create a governed change issue for finding 1; keep findings 2 and 3 as follow-ups.

```go
// VAT case expected to fail today:
// package currency
// func TestTotalIncludesVAT(t) { ... }
```