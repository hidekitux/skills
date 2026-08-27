# Security coverage

## Two distinct surfaces

The repository audits security in two places that must not be conflated:
workflow security and Go dependency security. They use different tools, cover
different inputs, and have different failure policies.

| Surface | Tool / command | Covers | Where it runs | Failure policy |
| --- | --- | --- | --- | --- |
| Workflow security | `zizmor` (`security.yml`, job `Audit workflow security`) | GitHub Actions workflow files (`.github/workflows/**`) for unsafe constructs | Every pull request and `main` push; the step runs only when workflow files changed | Blocking: findings fail the required `Audit workflow security` check |
| Go dependency security | `govulncheck` (`check:go-vuln`; `targeted.yml` job `Audit Go dependency security`) | The Go module graph and Go sources for known vulnerabilities in dependencies and the standard library | Tier 2: every pull request, step runs only when Go sources or module files changed; Tier 4: run before release per `docs/releasing.md` | Reachable findings fail; non-reachable findings are reported; infrastructure errors fail |

`zizmor` never scans Go code or modules, and the Go dependency scan never
audits workflow files. Check each surface with its own command.

## Go dependency scan (`check:go-vuln`)

- **Pinned tool.** `govulncheck` v1.7.0 is pinned in `mise.toml`
  (`go:golang.org/x/vuln/cmd/govulncheck`) with its license attested in
  `TOOL_LICENSES.toml` (BSD-3-Clause). Updating the scanner is a manual mise
  bump; Dependabot does not cover mise-managed tools.
- **Invocation.** `mise run check:go-vuln` runs
  `govulncheck -json -mode source ./...` through `cmd/scan-go-vuln`. Pass
  `-- -out <file>` to retain the raw JSON stream as machine-readable evidence.
- **Reported versions.** The scan prints the JSON header fields: scanner name
  and version, the vulnerability database URL and last-modified time, and the
  Go version used for the analysis, so a result stays attributable to a
  specific tool and database state.
- **Failure policy (Issue #178).** Defined by `cmd/scan-go-vuln`:
  - **Reachable findings** (a vulnerable symbol is called from scanned code)
    fail the command with exit 1.
  - **Non-reachable findings** (the vulnerable module is required or imported
    but no vulnerable symbol is called) are reported and do not fail (exit 0).
  - **Infrastructure errors** (govulncheck failure, database fetch failure,
    malformed output) fail with exit 2.
  The tier where the scan runs fails on reachable findings; non-reachable
  findings stay visible in the report and the retained output.
- **Exceptions.** A reachable finding can be accepted only with an explicit,
  reviewed exception recorded on the change (for example a tracked remediation
  follow-up). An exception never silences the scan: the finding remains in the
  retained output and the change notes.
- **Remediation.** Update the affected module, or the pinned Go version for
  standard-library findings, and re-run the scan until the finding is gone or
  an explicit reviewed exception is recorded.

## What this does not claim

- An available dependency update is not itself a vulnerability, and a version
  update is not a security finding. Dependabot update pull requests run the
  normal repository and behavioral validation — branch policy, commit
  conventions, commit signatures, tests, lint, and the Go dependency scan when
  Go files change — and are never automatically merged.
- The Go dependency scan reports known vulnerabilities; it does not prove the
  absence of unknown ones, and it covers the code the module graph and Go
  sources reach with the pinned Go version, as described by the `govulncheck`
  tool's own limitations.