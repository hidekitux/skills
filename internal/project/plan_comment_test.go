package project

import "testing"

func TestIsAuthoritativePlanComment(t *testing.T) {
	body := PlanCommentMarker(226) + `

## Implementation plan

1. Do the work.

## Out of scope

- Do not release.

## Residual risk

- Events can be delayed.

## Next-phase handoff

Use implement-issue.`
	if !IsAuthoritativePlanComment(body, 226) {
		t.Fatal("expected complete plan comment to be authoritative")
	}
}

func TestIsAuthoritativePlanCommentRejectsSpoofedOrIncompleteComments(t *testing.T) {
	valid := PlanCommentMarker(226) + "\n## Implementation plan\n## Out of scope\n## Residual risk\n## Next-phase handoff"
	cases := map[string]string{
		"wrong issue":        PlanCommentMarker(225) + valid[len(PlanCommentMarker(226)):],
		"marker not first":   "intro\n" + valid,
		"missing heading":    PlanCommentMarker(226) + "\n## Implementation plan\n## Out of scope\n## Residual risk",
		"edited marker text": PlanCommentMarker(226) + " extra\n## Implementation plan\n## Out of scope\n## Residual risk\n## Next-phase handoff",
	}
	for name, body := range cases {
		t.Run(name, func(t *testing.T) {
			if IsAuthoritativePlanComment(body, 226) {
				t.Fatal("expected comment to be rejected")
			}
		})
	}
}

func TestParsePlanCommentIssue(t *testing.T) {
	issue, err := ParsePlanCommentIssue(PlanCommentMarker(226))
	if err != nil || issue != 226 {
		t.Fatalf("got issue=%d err=%v, want 226", issue, err)
	}
	if _, err := ParsePlanCommentIssue("<!-- skills:plan-issue issue=x -->"); err == nil {
		t.Fatal("expected malformed marker to fail")
	}
}
