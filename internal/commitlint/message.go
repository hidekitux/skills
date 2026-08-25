// Package commitlint implements commit-message validation and the commit-lint
// driver, ported from scripts/lint/validate-commit-message.py and
// scripts/lint/lint-commits.py.
package commitlint

import (
	"regexp"
	"strings"
)

// headerPattern enforces a single Conventional Commits header line ending in
// ` #NNN` with a numeric Issue number.
var headerPattern = regexp.MustCompile(`^[^:\n]+: [^\n]+ #\d+$`)

var terminalSentencePattern = regexp.MustCompile(`[.!?]\s*#\d+$`)

// splitLines mirrors Python str.splitlines for the commit-message inputs this
// package validates: a trailing single newline does not produce an empty line,
// while interior empty lines are preserved.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(s, "\n"), "\n")
}

// ValidateMessage checks the single-line and issue-number policy that
// commitlint cannot express, returning a list of human-readable problems.
// An empty list means the message is valid.
func ValidateMessage(message string) []string {
	errors := []string{}
	lines := splitLines(message)
	if len(lines) == 0 {
		return []string{"commit message must not be empty"}
	}
	if len(lines) != 1 {
		errors = append(errors, "commit message must be exactly one line")
		return errors
	}
	if !headerPattern.MatchString(lines[0]) {
		errors = append(errors, "header must be `type(scope): summary #NNN` with a numeric issue number")
		return errors
	}
	summary := strings.TrimSpace(strings.SplitN(lines[0], ":", 2)[1])
	if terminalSentencePattern.MatchString(summary) {
		errors = append(errors, "summary must be a single sentence without terminal punctuation")
	}
	return errors
}
