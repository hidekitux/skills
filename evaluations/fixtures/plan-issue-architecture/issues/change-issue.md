# Change issue: keep checkout ingestion responsive

## Context

The checkout service processes payment events in the request path at
`src/handler.go`. Large events can make the request exceed the response-time
target. The repository also contains a queue boundary in `src/worker.go`, but
the change has not selected whether the handler should enqueue events or keep
processing in the request path with a bounded handoff. The change has also not
selected whether processing failures should remain in the checkout service or
move to a separate failure-recording boundary.

## Goal

Keep checkout requests responsive while preserving one processing attempt per
event and making processing failures observable.

## Scope

- In: the checkout ingestion and processing boundary in `src/handler.go` and
  `src/worker.go`, with tests for response behavior, event processing, and
  failure visibility.
- Out: payment-provider selection, database migrations, deployment changes
  outside the ingestion boundary, and release work.

## Acceptance criteria

- [ ] Checkout requests meet the response-time target for large events.
- [ ] Each event has one observable processing outcome.
- [ ] Processing failures remain visible to operators.
- [ ] Tests cover the response, processing, and failure boundaries.

## Validation

- [ ] Run `go test ./...` and record the output.
