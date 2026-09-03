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

			transcriptOnly := transcriptAssertionsOf(sc)
			sandbox := t.TempDir()
			pass := evaluateAssertions(context.Background(), transcriptOnly, conforming, sandbox, nil, nil)
			fail := evaluateAssertions(context.Background(), transcriptOnly, breaking, sandbox, nil, nil)

			if got := proseFailures(pass); got != 0 {
				t.Fatalf("conforming transcript reported %d prose failure(s): %v", got, pass)
			}
			if proseFailures(fail) == 0 {
				t.Fatalf("rule-breaking transcript reported no prose failure: %v", fail)
			}
		})
	}
}

// transcriptAssertionsOf copies a scenario with only its transcript assertions
// kept. evaluateAssertions runs every command_run entry through sh, and
// implement-issue-success declares `go test ./...`, so passing a scenario
// unchanged would execute it here. The file and command assertions belong to a
// staged sandbox run, not to this test.
func transcriptAssertionsOf(sc *Scenario) *Scenario {
	return &Scenario{
		ID:   sc.ID,
		Kind: sc.Kind,
		Expectations: Expectations{
			TranscriptMustNot: sc.Expectations.TranscriptMustNot,
		},
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
