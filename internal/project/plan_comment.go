package project

import (
	"fmt"
	"strconv"
	"strings"
)

const planCommentMarkerFormat = "<!-- skills:plan-issue issue=%d -->"

var requiredPlanCommentHeadings = []string{
	"## Implementation plan",
	"## Out of scope",
	"## Residual risk",
	"## Next-phase handoff",
}

// IsAuthoritativePlanComment reports whether body is a complete plan comment
// emitted for issueNumber. The marker must be the first line and every required
// handoff section must be present as an exact heading line.
func IsAuthoritativePlanComment(body string, issueNumber int64) bool {
	if issueNumber <= 0 {
		return false
	}
	lines := strings.Split(strings.ReplaceAll(body, "\r\n", "\n"), "\n")
	if len(lines) == 0 || lines[0] != fmt.Sprintf(planCommentMarkerFormat, issueNumber) {
		return false
	}
	seen := map[string]bool{}
	for _, line := range lines[1:] {
		if _, required := requiredHeadingSet[line]; required {
			seen[line] = true
		}
	}
	for _, heading := range requiredPlanCommentHeadings {
		if !seen[heading] {
			return false
		}
	}
	return true
}

var requiredHeadingSet = func() map[string]struct{} {
	set := make(map[string]struct{}, len(requiredPlanCommentHeadings))
	for _, heading := range requiredPlanCommentHeadings {
		set[heading] = struct{}{}
	}
	return set
}()

// PlanCommentMarker returns the marker that plan-issue must put on the first
// line of its authoritative plan comment.
func PlanCommentMarker(issueNumber int64) string {
	return fmt.Sprintf(planCommentMarkerFormat, issueNumber)
}

// ParsePlanCommentIssue extracts the Issue number from a valid marker. It is
// useful to callers that receive only the marker line.
func ParsePlanCommentIssue(marker string) (int64, error) {
	const prefix = "<!-- skills:plan-issue issue="
	const suffix = " -->"
	if !strings.HasPrefix(marker, prefix) || !strings.HasSuffix(marker, suffix) {
		return 0, fmt.Errorf("invalid plan comment marker")
	}
	value := strings.TrimSuffix(strings.TrimPrefix(marker, prefix), suffix)
	issue, err := strconv.ParseInt(value, 10, 64)
	if err != nil || issue <= 0 {
		return 0, fmt.Errorf("invalid plan comment Issue number")
	}
	return issue, nil
}
