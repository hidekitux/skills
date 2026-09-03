package eval

import (
	"context"
	"strings"
	"testing"
)

// proseMarkers are the words a scenario must forbid for its run to observe the
// prose the skill produced. docs/writing-style.md names the first three an
// inflated style word and the last two Latinate padding; each appears in both
// cased forms because the assertion is a case-sensitive substring match.
var proseMarkers = []string{"delv", "pivotal", "multifaceted", "facilitat", "commenc"}

// TestEveryPositiveScenarioForbidsTheProseMarkers requires each positive
// scenario in the corpus to carry every marker, so a skill added later cannot
// reach a passing run with its prose unobserved. The list is read from the
// corpus rather than written here, because a hand-maintained copy of the tier
// table in docs/skill-contract.md would drift from it silently.
func TestEveryPositiveScenarioForbidsTheProseMarkers(t *testing.T) {
	for _, sc := range positiveScenarios(t) {
		t.Run(sc.ID, func(t *testing.T) {
			forbidden := map[string]bool{}
			for _, pattern := range sc.Expectations.TranscriptMustNot {
				forbidden[pattern] = true
			}
			for _, marker := range proseMarkers {
				for _, cased := range []string{marker, strings.ToUpper(marker[:1]) + marker[1:]} {
					if !forbidden[cased] {
						t.Errorf("scenario does not forbid %q, so its run cannot observe that marker", cased)
					}
				}
			}
		})
	}
}

// TestProseAssertionsDistinguishBreakingRun runs each positive scenario's own
// assertions against a conforming transcript and against one that breaks a
// stated writing rule, and requires the two verdicts to differ. Without this, a
// scenario could carry an empty marker list and report a pass for prose it
// never examined.
//
// The transcripts are written here rather than staged as a fixture on purpose.
// evaluateAssertions matches TranscriptMustNot over the whole transcript, so a
// fixture holding the marker words would fail a conforming run the moment the
// evaluated agent read the file.
func TestProseAssertionsDistinguishBreakingRun(t *testing.T) {
	const conforming = "The plan starts the run and uses the recorded evidence."
	const breaking = "The plan delves into the pivotal work and will commence shortly."

	for _, sc := range positiveScenarios(t) {
		t.Run(sc.ID, func(t *testing.T) {
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

// positiveScenarios loads every positive scenario in the tracked corpus.
func positiveScenarios(t *testing.T) []*Scenario {
	t.Helper()
	all, err := LoadAllScenarios("../..")
	if err != nil {
		t.Fatal(err)
	}
	var positives []*Scenario
	for _, sc := range all {
		if sc.Kind == "positive" {
			positives = append(positives, sc)
		}
	}
	if len(positives) == 0 {
		t.Fatal("corpus holds no positive scenario")
	}
	return positives
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
