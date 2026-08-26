# Change issue: apply VAT on checkout totals

## Context

The widgets checkout computes line totals without VAT, so rounded-total orders are undercharged and the discrepancy shows up in monthly reporting.

## Goal

Apply the configured VAT rate to every checkout total so reported totals match what customers pay.

## Scope

- In: update `src/currency.go` to include VAT at the configured rate; add boundary tests; keep the module green.
- Out: pricing documentation, refund flow, and release work.

## Acceptance criteria

- [ ] Checkout totals include VAT at the configured rate.
- [ ] The VAT boundary is covered by a test.
- [ ] `go test ./...` passes.

## Validation

- [ ] Run `go test ./...` and record the output.