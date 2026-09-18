// Package redact owns the single rule that decides whether a host is private
// and the single definition of every pattern a package matches before it
// persists or reports a value. One definition lives here so a change to the
// rule reaches every caller at once.
//
// This package belongs to the foundation module. It imports no other internal
// module, so every module may import it. That is why the rule lives here
// rather than in internal/evidence, which the provider and policy modules
// cannot import. See docs/architecture.md and workflow/module-ownership.yml.
package redact

import (
	"net"
	"regexp"
	"strings"
)

var (
	credentialPattern = regexp.MustCompile(`(?i)(bearer\s+|password\s*=\s*|token\s*=\s*|secret\s*=\s*|api[_-]?key\s*=\s*)([^\s,;]+)|(?:gh[pousr]_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+|sk-[A-Za-z0-9_-]+|AKIA[0-9A-Z]{16})`)
	urlPattern        = regexp.MustCompile(`(?i)https?://[^\s"']+`)
	commitSHAPattern  = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// CredentialPattern matches a credential a caller must never persist or
// report. A caller uses it to reject a value outright or to replace the match
// with a redaction placeholder.
func CredentialPattern() *regexp.Regexp { return credentialPattern }

// URLPattern matches an absolute HTTP or HTTPS URL, whatever the case of its
// scheme. A caller uses it to find every URL in a value so it can check the
// host with IsPrivateHost.
func URLPattern() *regexp.Regexp { return urlPattern }

// CommitSHAPattern matches a full 40-character Git commit identifier.
func CommitSHAPattern() *regexp.Regexp { return commitSHAPattern }

// IsPrivateHost reports whether host names a private or local address. A
// caller redacts a URL with such a host, because the address is meaningful
// only inside the network that produced it.
//
// The host is the value url.URL.Hostname returns, so an IPv6 literal arrives
// without its enclosing brackets.
func IsPrivateHost(host string) bool {
	if parsed := net.ParseIP(host); parsed != nil {
		return parsed.IsLoopback() || parsed.IsPrivate() || parsed.IsLinkLocalUnicast()
	}
	lowered := strings.ToLower(host)
	return lowered == "localhost" || strings.HasSuffix(lowered, ".local")
}
