// Package check implements deterministic repository policy checks ported from
// the former Python scripts under scripts/check/.
//
// This package belongs to the governance module. It may import the foundation, domain, and evidence modules only. See
// docs/architecture.md and workflow/module-ownership.yml.
package check

import (
	"fmt"
	"io"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/hidekitux/skills/internal/redact"
)

type sensitivePattern struct {
	label string
	match func(line string) bool
}

// matchPattern adapts a regular expression to the sensitivePattern matcher.
func matchPattern(expression string) func(string) bool {
	return regexp.MustCompile(expression).MatchString
}

// containsPrivateNetworkURL reports whether line holds an absolute URL whose
// host redact.IsPrivateHost reports as private. It replaces the enumerated
// expression this check used before Issue #343, which matched no unique local
// IPv6 address and no link local address, so one rule now answers the private
// host question for this check and for internal/provider alike.
func containsPrivateNetworkURL(line string) bool {
	for _, raw := range redact.URLPattern().FindAllString(line, -1) {
		parsed, err := url.Parse(raw)
		if err != nil {
			continue
		}
		if host := parsed.Hostname(); host != "" && redact.IsPrivateHost(host) {
			return true
		}
	}
	return false
}

// sensitivePatterns reject tracked text that matches known credentials or
// private user context. The patterns are the Python originals with Python
// \b word boundaries translated to explicit RE2 boundary expressions.
var sensitivePatterns = []sensitivePattern{
	{
		label: "GitHub token",
		match: matchPattern(`(?:^|[^A-Za-z0-9_])(?:gh[pousr]_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,})(?:$|[^A-Za-z0-9_])`),
	},
	{
		label: "OpenAI-style API key",
		match: matchPattern(`(?:^|[^A-Za-z0-9_])sk-[A-Za-z0-9_-]{20,}(?:$|[^A-Za-z0-9_])`),
	},
	{
		label: "Slack token",
		match: matchPattern(`(?:^|[^A-Za-z0-9_])xox[baprs]-[A-Za-z0-9-]{20,}(?:$|[^A-Za-z0-9_])`),
	},
	{
		label: "private network URL",
		match: containsPrivateNetworkURL,
	},
	{
		label: "macOS user path",
		match: matchPattern(`/Users/[A-Za-z0-9._-]+(?:/|$|[^A-Za-z0-9_])`),
	},
	{
		label: "email address",
		match: matchPattern(`(?i)(?:^|[^A-Za-z0-9_])[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}(?:$|[^A-Za-z0-9_])`),
	},
}

// candidateFiles returns the tracked and untracked non-ignored files under
// root, falling back to a plain recursive walk when Git is unavailable.
func candidateFiles(root string) []string {
	out, err := gitOutput(root, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err == nil {
		var files []string
		for _, item := range strings.Split(out, "\x00") {
			if item != "" {
				files = append(files, filepath.Join(root, item))
			}
		}
		return files
	}
	var files []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files
}

// CheckSensitiveContent scans repository text for credentials and private
// user context, returning 0 on success or 1 when findings exist.
func CheckSensitiveContent(root string, out, errOut io.Writer) int {
	findings := []string{}
	for _, path := range candidateFiles(root) {
		content, err := os.ReadFile(path)
		if err != nil || !utf8.Valid(content) {
			continue
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		for number, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSuffix(line, "\r")
			for _, candidate := range sensitivePatterns {
				if candidate.match(line) {
					findings = append(findings, fmt.Sprintf("%s:%d: %s", rel, number+1, candidate.label))
				}
			}
		}
	}
	if len(findings) > 0 {
		fmt.Fprintln(errOut, "Sensitive-content check failed:")
		for _, finding := range findings {
			fmt.Fprintf(errOut, "- %s\n", finding)
		}
		return 1
	}
	fmt.Fprintln(out, "Sensitive-content check passed.")
	return 0
}
