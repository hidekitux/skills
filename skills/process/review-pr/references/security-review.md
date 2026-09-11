# Security review reference

Load this reference only when the PR content touches authentication, secrets,
permissions, untrusted input, or data exposure. It shapes failure hypotheses on
the traced paths; it is not a checklist to run on every PR.

## Focus areas

- Credentials without protection: tokens, keys, and secrets introduced, logged, or
  exposed; hardcoded defaults that users inherit.
- Authentication and authorization: any path that reaches protected data without
  an authentication check, or that checks the wrong principal or scope.
- Untrusted input: injection, deserialization, path traversal, or unsafe
  construction from input the change makes reachable.
- Data exposure: private URLs, personal data, or secrets written to logs,
  responses, or generated artifacts the change produces.
- Forbidden states: no secret data before authentication; no state where a caller
  can observe another user's data.

## Applying the rules

- Report only a security issue the PR introduces or makes reachable, with a
  concrete reproduction scenario (input or state, path, observable breakage, why
  existing defenses do not prevent it, and the causing diff location).
- Drop a candidate when existing defenses on the traced path already prevent it,
  or when there is no realistic execution path from the changed lines.
