// Package evidence owns the primitives that more than one evidence producer
// needs. A producer is the package that writes a persisted artifact:
// internal/trace for a structured trace, internal/diagnostic for a validator
// diagnostic, and internal/replay for a replay report.
//
// The package holds only primitives with no producer-specific field. A type
// whose field set differs between producers stays with its producer, because
// merging it would widen a persisted contract.
package evidence

import (
	"net"
	"regexp"
	"strings"
)

// DiagnosticRef identifies a diagnostic without persisting its message,
// location, or any other content the diagnostic itself carries.
type DiagnosticRef struct {
	Producer string `json:"producer"`
	Code     string `json:"code"`
}

var (
	credentialPattern = regexp.MustCompile(`(?i)(bearer\s+|password\s*=\s*|token\s*=\s*|secret\s*=\s*|api[_-]?key\s*=\s*)([^\s,;]+)|(?:gh[pousr]_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+|sk-[A-Za-z0-9_-]+|AKIA[0-9A-Z]{16})`)
	urlPattern        = regexp.MustCompile(`https?://[^\s"']+`)
	commitSHAPattern  = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// CredentialPattern matches a credential a producer must never persist. A
// producer uses it to reject a value outright or to replace the match with a
// redaction placeholder.
func CredentialPattern() *regexp.Regexp { return credentialPattern }

// URLPattern matches an absolute HTTP or HTTPS URL. A producer uses it to find
// every URL in a value so it can check the host before persisting the value.
func URLPattern() *regexp.Regexp { return urlPattern }

// CommitSHAPattern matches a full 40-character Git commit identifier.
func CommitSHAPattern() *regexp.Regexp { return commitSHAPattern }

// IsPrivateAddress reports whether host names a private or local address. A
// producer rejects a URL with such a host, because the address is meaningful
// only inside the network that produced it.
func IsPrivateAddress(host string) bool {
	if parsed := net.ParseIP(host); parsed != nil {
		return parsed.IsPrivate() || parsed.IsLoopback()
	}
	return host == "localhost" || strings.HasSuffix(host, ".local")
}
