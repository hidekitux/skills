package eval

import (
	"context"
	"strings"
	"testing"
)

// proseScenarios lists the scenario files that carry the prose assertions, one
// per skill whose prose outlives the conversation. docs/skill-contract.md holds
// the tier table these seven come from.
var proseScenarios = []string{
	"bootstrap-project/bootstrap-project-success.yaml",
	"create-issue/create-issue-success.yaml",
	"create-pr/create-pr-success.yaml",
	"fix-pr/fix-pr-success.yaml",
	"implement-issue/implement-issue-success.yaml",
	"plan-issue/plan-issue-success.yaml",
	"review-pr/review-pr-success.yaml",
}

// TestProseAssertionsDistinguishBreakingRun runs each scenario's own assertions
// against a conforming transcript and against one that breaks a stated writing
// rule, and requires the two verdicts to differ. Without this, a scenario could
// carry an empty marker list and report a pass for prose it never examined.
//
// The transcripts are written here rather than staged as a fixture on purpose.
// evaluateAssertions matches TranscriptMustNot over the whole transcript, so a
// fixture holding the marker words would fail a conforming run the moment the
// evaluated agent read the file.
func TestProseAssertionsDistinguishBreakingRun(t *testing.T) {
	const conforming = "The plan starts the run and uses the recorded evidence."
	const breaking = "The plan delves into the pivotal work and will commence shortly."

	for _, rel := range proseScenarios {
		t.Run(rel, func(t *testing.T) {
			sc, err := LoadScenario("../../" + scenarioBase + "/" + rel)
			if err != nil {
				t.Fatal(err)
			}
			if len(sc.Expectations.TranscriptMustNot) == 0 {
				t.Fatal("scenario carries no prose marker; a run cannot observe its writing rules")
			}

			sandbox := t.TempDir()
			pass := evaluateAssertions(context.Background(), sc, conforming, sandbox, nil, nil)
			fail := evaluateAssertions(context.Background(), sc, breaking, sandbox, nil, nil)

			if proseFailures(pass) != 0 {
				t.Fatalf("conforming transcript reported a prose failure: %v", pass)
			}
			if got := proseFailures(fail); got == 0 {
				t.Fatalf("rule-breaking transcript reported no prose failure: %v", fail)
			}
		})
	}
}

// proseFailures counts the assertion failures a forbidden marker produced, so a
// scenario's unrelated assertions do not decide the result.
func proseFailures(failures []string) int {
	count := 0
	for _, failure := range failures {
		if strings.HasPrefix(failure, "transcript contains forbidden") {
			count++
		}
	}
	return count
}
