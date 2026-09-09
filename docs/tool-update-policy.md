# Pinned tool update policy

The repository owner, `@hidekitux`, reviews every pinned tool in the first week of each month. The owner starts an out-of-cycle review when an upstream release or security advisory reports a fix that affects a pinned tool. The review covers the version pins in [`mise.toml`](../mise.toml) and the version and checksums in [`scripts/fsl/install-fslc.sh`](../scripts/fsl/install-fslc.sh).

## Review procedure

1. Read the authoritative release source for each pinned tool and record the current pin and latest stable release in the inventory below.
2. Read the newer release notes and public security advisories. Record whether a newer security fix affects the current pin.
3. Read the upstream license and source record. For a mise-managed tool, keep the matching entry in [`TOOL_LICENSES.toml`](../TOOL_LICENSES.toml). A version bump must update that attestation when the license or source changes and must state in the Pull Request when the attestation remains valid.
4. Run `mise run check:repository` and `mise run check:go-vuln`. Run `mise run validate:all` before publishing the bump. Record the scanner and vulnerability-database versions from `mise run check:go-vuln`.
5. For fslc, record the upstream license and release source with the bump and update the checksum for every supported platform. `mise run check:repository` checks mise-managed tools and direct Go modules; it does not check fslc because fslc is installed by a script.

The Pull Request for a bump must link the upstream release or advisory, name the old and new pins, include the validation results, and describe the license-attestation decision. This policy review does not bump a tool version.

## Current inventory

The owner checked the following sources on 2026-09-09. “No fix identified” means that the reviewed release notes and public advisory source did not identify a security fix after the current pin. It does not claim that the tool has no undisclosed or unrelated vulnerability.

| Tool | Pin and current version | Latest stable release checked | Security result | Authoritative source |
| --- | --- | --- | --- | --- |
| Go | [`mise.toml`](../mise.toml), `1.26.6` | `1.27.1`; latest `1.26` patch is `1.26.8` | No later security fix identified. `1.26.6` itself includes security fixes. | [Go release history](https://go.dev/doc/devel/release) |
| actionlint | [`mise.toml`](../mise.toml), `1.7.12` | `1.7.12` | No fix identified. The current pin is latest. | [actionlint v1.7.12](https://github.com/rhysd/actionlint/releases/tag/v1.7.12) |
| Ruff | [`mise.toml`](../mise.toml), `0.16.0` | `0.16.6` | No fix identified in the reviewed release notes. | [Ruff 0.16.6](https://github.com/astral-sh/ruff/releases/tag/0.16.6) |
| ShellCheck | [`mise.toml`](../mise.toml), `0.11.0` | `0.11.0` | No fix identified. The current pin is latest. | [ShellCheck v0.11.0](https://github.com/koalaman/shellcheck/releases/tag/v0.11.0) |
| GitHub CLI | [`mise.toml`](../mise.toml), `2.97.0` | `2.100.0` | **Security follow-up required.** `CVE-2026-72924` is fixed in `2.98.0`; the current pin is older. | [GitHub CLI v2.100.0](https://github.com/cli/cli/releases/tag/v2.100.0), [GHSA-vfhh-p7hm-pxfh](https://github.com/cli/cli/security/advisories/GHSA-vfhh-p7hm-pxfh) |
| antigravity-cli | [`mise.toml`](../mise.toml), `1.1.19` | `1.1.28` | No security fix identified in the reviewed release notes. | [antigravity-cli 1.1.28](https://github.com/google-antigravity/antigravity-cli/releases/tag/1.1.28) |
| OpenCode | [`mise.toml`](../mise.toml), `1.18.22` | `1.18.30` | No newer fix identified. The current pin is later than the patched versions in the reviewed advisories. | [OpenCode v1.18.30](https://github.com/anomalyco/opencode/releases/tag/v1.18.30), [GHSA-c83v-7274-4vgp](https://github.com/anomalyco/opencode/security/advisories/GHSA-c83v-7274-4vgp), [GHSA-vxw4-wv6m-9hhh](https://github.com/anomalyco/opencode/security/advisories/GHSA-vxw4-wv6m-9hhh) |
| govulncheck | [`mise.toml`](../mise.toml), `v1.7.0` | `v1.8.0` | No security fix identified in the reviewed module release record. Update the scanner when its freshness is required. | [golang.org/x/vuln v1.8.0](https://pkg.go.dev/golang.org/x/vuln@v1.8.0) |
| fslc | [`install-fslc.sh`](../scripts/fsl/install-fslc.sh), `4.2.0` | `4.4.1` | No security fix identified in the reviewed release notes. | [fsl v4.4.1](https://github.com/ymm-oss/fsl/releases/tag/v4.4.1) |

The GitHub CLI finding remains intentionally unfixed because Issue #253 excludes version bumps. The owner must review and prioritize that bump in a separate change before relying on a newer release for the security fix.
