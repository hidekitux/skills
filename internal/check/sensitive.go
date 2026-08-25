// Package check implements deterministic repository policy checks ported from
// the former Python scripts under scripts/check/.
package check

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/hidekitux/skills/internal/support"
)

type sensitivePattern struct {
	label   string
	pattern *regexp.Regexp
}

// sensitivePatterns reject tracked text that matches known credentials or
// private user context. The patterns are the Python originals with Python
// \b word boundaries translated to explicit RE2 boundary expressions.
var sensitivePatterns = []sensitivePattern{
	{
		label:   "GitHub token",
		pattern: regexp.MustCompile(`(?:^|[^A-Za-z0-9_])(?:gh[pousr]_[A-Za-z0-9_]{20,}|github_pat_[A-Za-z0-9_]{20,})(?:$|[^A-Za-z0-9_])`),
	},
	{
		label:   "OpenAI-style API key",
		pattern: regexp.MustCompile(`(?:^|[^A-Za-z0-9_])sk-[A-Za-z0-9_-]{20,}(?:$|[^A-Za-z0-9_])`),
	},
	{
		label:   "Slack token",
		pattern: regexp.MustCompile(`(?:^|[^A-Za-z0-9_])xox[baprs]-[A-Za-z0-9-]{20,}(?:$|[^A-Za-z0-9_])`),
	},
	{
		label:   "private network URL",
		pattern: regexp.MustCompile(`(?i)https?://(?:localhost|127(?:\.\d{1,3}){3}|10(?:\.\d{1,3}){3}|192\.168(?:\.\d{1,3}){2}|172\.(?:1[6-9]|2\d|3[0-1])(?:\.\d{1,3}){2})(?::\d+)?(?:[/?#]|$|[^A-Za-z0-9_])`),
	},
	{
		label:   "macOS user path",
		pattern: regexp.MustCompile(`/Users/[A-Za-z0-9._-]+(?:/|$|[^A-Za-z0-9_])`),
	},
	{
		label:   "email address",
		pattern: regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9_])[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}(?:$|[^A-Za-z0-9_])`),
	},
}

// candidateFiles returns the tracked and untracked non-ignored files under
// root, falling back to a plain recursive walk when Git is unavailable.
func candidateFiles(root string) []string {
	out, err := support.GitOutputIn(root, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
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
				if candidate.pattern.MatchString(line) {
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
