package eval

import (
	"bytes"
	"strings"
	"testing"
)

// The injection tests implement the Issue validation item "Inject one known
// contract violation and confirm the appropriate scenario fails" at the
// deterministic level: a dropped handoff fails the scenario assertions, and a
// dropped failure scenario fails the corpus check. They run without any host
// CLI or model access.

// TestInjectionContractViolationDetected removes the expected handoff from the
// agent transcript and confirms the scenario's deterministic assertions
// fail. The scenario declares the handoff only, with no transcript_must
// duplication, so the handoff assertion itself must catch the dropped
// handoff.
func TestInjectionContractViolationDetected(t *testing.T) {
	sc := &Scenario{
		ID:     "implement-issue-success",
		Skill:  "implement-issue",
		Kind:   KindPositive,
		Prompt: "Execute the plan and stop before a pull request.",
		Expectations: Expectations{
			Handoff: "create-pr",
		},
	}
	// Injected violation: the transcript names no handoff to create-pr.
	host := &fakeHost{name: "claude-code", available: true, line: "plan executed"}
	record := runOneForTest(t, sc, host, &Options{HandoffNames: map[string]bool{"create-pr": true}})
	if record.Verdict != VerdictFail {
		t.Fatalf("verdict = %s, want fail", record.Verdict)
	}
	if len(record.Failures) == 0 || !strings.Contains(record.Failures[0], `does not name the expected handoff "create-pr"`) {
		t.Fatalf("failures = %v, want missing-handoff finding", record.Failures)
	}
}

// TestInjectionMissingFailureScenarioBlocked removes the failure or boundary
// scenarios of a cataloged skill and confirms the corpus check fails with a
// coverage finding.
func TestInjectionMissingFailureScenarioBlocked(t *testing.T) {
	root := scaffoldEval(t,
		[]map[string]string{skillEntry("plan-issue", "experimental")},
		[]*Scenario{baseScenario()},
		nil,
	)
	var out, errOut bytes.Buffer
	if code := CheckCorpus(root, &out, &errOut); code != 1 {
		t.Fatalf("expected corpus check to fail, got %d", code)
	}
	if !strings.Contains(errOut.String(), "has no failure or boundary scenario") {
		t.Fatalf("missing coverage finding:\n%s", errOut.String())
	}
}

// TestInjectionPromptLeakDetected confirms that a prompt naming the skill
// under evaluation fails the corpus check instead of leaking the expected
// trigger.
func TestInjectionPromptLeakDetected(t *testing.T) {
	leaky := baseScenario()
	leaky.Prompt = "Run the plan-issue workflow on this change."
	root := scaffoldEval(t,
		[]map[string]string{skillEntry("plan-issue", "experimental")},
		[]*Scenario{leaky},
		nil,
	)
	var out, errOut bytes.Buffer
	if code := CheckCorpus(root, &out, &errOut); code != 1 {
		t.Fatalf("expected corpus check to fail, got %d", code)
	}
	if !strings.Contains(errOut.String(), "revealing trigger selection") {
		t.Fatalf("missing leak finding:\n%s", errOut.String())
	}
}
