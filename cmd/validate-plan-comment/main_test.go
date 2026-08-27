package main

import (
	"testing"

	"github.com/hidekitux/skills/internal/project"
)

func TestPlanCommentValidationContract(t *testing.T) {
	body := project.PlanCommentMarker(226) + `

## Implementation plan

1. Complete the implementation.

## Out of scope

- Release work.

## Residual risk

- Events may be delayed.

## Next-phase handoff

Use implement-issue.`
	if !project.IsAuthoritativePlanComment(body, 226) {
		t.Fatal("expected the representative plan comment to be accepted")
	}
}
