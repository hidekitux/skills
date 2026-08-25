package validate

import (
	"fmt"
	"io"
	"strings"
)

var (
	labelPriorities = []string{"high", "medium", "low"}
	labelScopes     = []string{"feature", "bug", "docs", "maintenance", "improvement", "release"}
	labelPhases     = []string{"backlog", "planned", "in-progress"}
)

func labelAllowed(prefix, value string) bool {
	var allowed []string
	switch prefix {
	case "priority":
		allowed = labelPriorities
	case "scope":
		allowed = labelScopes
	case "phase":
		allowed = labelPhases
	default:
		return false
	}
	for _, candidate := range allowed {
		if value == candidate {
			return true
		}
	}
	return false
}

// labelErrors returns the validation errors for an Issue triage label set.
func labelErrors(labels []string) []string {
	errors := []string{}
	if len(labels) == 0 {
		return []string{"at least one label is required (priority:, scope:, phase:)"}
	}
	counts := map[string]int{"priority": 0, "scope": 0, "phase": 0}
	for _, label := range labels {
		prefix, value, hasColon := strings.Cut(label, ":")
		if !hasColon || !(prefix == "priority" || prefix == "scope" || prefix == "phase") {
			errors = append(errors, fmt.Sprintf("unknown label: %q", label))
			continue
		}
		if !labelAllowed(prefix, value) {
			errors = append(errors, fmt.Sprintf("unknown label value for %s: %q", prefix, label))
			continue
		}
		counts[prefix]++
	}
	for _, dimension := range []string{"priority", "scope", "phase"} {
		if counts[dimension] != 1 {
			errors = append(errors, fmt.Sprintf("exactly one %s: label is required; found %d", dimension, counts[dimension]))
		}
	}
	return errors
}

// CheckIssueLabels validates the triage label set on an open Issue, returning
// 0 on success or 1 when the label set is invalid.
func CheckIssueLabels(labels []string, out, errOut io.Writer) int {
	errors := labelErrors(labels)
	if len(errors) > 0 {
		fmt.Fprintf(errOut, "error: %s\n", strings.Join(errors, "; "))
		return 1
	}
	fmt.Fprintln(out, "Issue triage labels are valid.")
	return 0
}
