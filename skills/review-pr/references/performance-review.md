# Performance review reference

Load this reference only when the PR content touches hot paths, loops, I/O,
queries, or large data handling. Do not speculate about performance without a
concrete path and observable breakage.

## Focus areas

- Unbounded work: loops or recursion whose cost grows with input with no bound,
  allocation on every request, or re-fetching data the change can reuse.
- N+1 and redundant I/O: repeated queries or network calls inside loops that the
  change introduces or makes reachable.
- Query and schema impact: changed queries, indexes, serialization formats, or
  defaults that degrade existing callers.
- Resource exhaustion: paths that hold or grow memory, connections, or files
  without release, with a concrete trigger.

## Applying the rules

- Report only regressions the PR introduces or makes reachable, backed by the
  exact path and the observable effect (for example, "each request now issues N
  queries where the previous code issued one").
- Drop candidates without a realistic execution path or with no measurable adverse
  effect. Under-reporting is preferred over hypothesis-only noise.
