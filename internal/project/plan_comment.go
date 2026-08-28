package project

import (
	"fmt"
	"strconv"
	"strings"
)

const planCommentMarkerFormat = "<!-- skills:plan-issue issue=%d -->"

// requiredPlanCommentHeadingGroups keeps the emitted heading compatible with
// both the original contract and the heading currently produced by plan-issue.
// Each group must contribute exactly one heading, in this order.
var requiredPlanCommentHeadingGroups = [][]string{
	{"## Implementation plan", "## Ordered implementation plan"},
	{"## Out of scope"},
	{"## Residual risk"},
	{"## Next-phase handoff"},
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
	headingCounts := map[string]int{}
	for _, line := range lines[1:] {
		headingCounts[line]++
	}
	positions := make([]int, 0, len(requiredPlanCommentHeadingGroups))
	for _, group := range requiredPlanCommentHeadingGroups {
		position := 0
		matches := 0
		for _, heading := range group {
			if headingCounts[heading] != 1 {
				continue
			}
			matches++
			for index := 1; index < len(lines); index++ {
				if lines[index] == heading {
					position = index
					break
				}
			}
		}
		if matches != 1 || position == 0 {
			return false
		}
		positions = append(positions, position)
	}
	if len(positions) != len(requiredPlanCommentHeadingGroups) {
		return false
	}
	for index, position := range positions {
		if index > 0 && position <= positions[index-1] {
			return false
		}
	}
	if strings.TrimSpace(strings.Join(lines[1:positions[0]], "\n")) != "" {
		return false
	}
	for index, position := range positions {
		end := len(lines)
		if index+1 < len(positions) {
			end = positions[index+1]
		}
		if strings.TrimSpace(strings.Join(lines[position+1:end], "\n")) == "" {
			return false
		}
	}
	return true
}

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
