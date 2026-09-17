package trace

import (
	"encoding/json"
	"os"
	"testing"
)

// TestFromEvaluationRunReproducesTheRecordedTrace pins the persisted shape this
// conversion produces. The golden file was captured from the assembly that
// internal/eval owned before Issue #329 moved it here, so an added, removed, or
// differently valued field, and a different event order, is a change to a
// persisted contract. equalJSON sorts the keys of a JSON object, so the order of
// the fields inside one object is not compared.
func TestFromEvaluationRunReproducesTheRecordedTrace(t *testing.T) {
	data, err := os.ReadFile("testdata/evaluation-run-traces.json")
	if err != nil {
		t.Fatalf("read golden traces: %v", err)
	}
	var golden map[string]json.RawMessage
	if err := json.Unmarshal(data, &golden); err != nil {
		t.Fatalf("decode golden traces: %v", err)
	}
	outcomes := []RunOutcome{RunPassed, RunFailed, RunSkipped, RunInterrupted, RunInfrastructureError, "unknown_verdict"}
	if len(golden) != len(outcomes) {
		t.Fatalf("golden covers %d outcomes, the test covers %d", len(golden), len(outcomes))
	}
	for _, outcome := range outcomes {
		want, ok := golden[string(outcome)]
		if !ok {
			t.Fatalf("golden has no trace for outcome %q", outcome)
		}
		got, err := json.MarshalIndent(FromEvaluationRun(representativeRun(outcome)), "", "  ")
		if err != nil {
			t.Fatalf("marshal trace for %q: %v", outcome, err)
		}
		if equalJSON(t, got, want) {
			continue
		}
		t.Fatalf("trace for %q differs from the recorded shape:\ngot:  %s\nwant: %s", outcome, got, want)
	}
}

// TestFromEvaluationRunDefaultsTheMissingTimestamps covers the two inputs the
// golden file cannot carry, because a captured record always has both times.
func TestFromEvaluationRunDefaultsTheMissingTimestamps(t *testing.T) {
	item := FromEvaluationRun(RunResult{RunID: "run-1", SkillID: "plan-issue", Outcome: RunPassed})
	if item.StartedAt != "1970-01-01T00:00:00Z" {
		t.Fatalf("expected the epoch as the default start, got %q", item.StartedAt)
	}
	if item.Terminal.At != item.StartedAt {
		t.Fatalf("expected the finish to default to the start, got %q", item.Terminal.At)
	}
}

// TestFromEvaluationRunOmitsTheOptionalEvents checks the three conditional
// events, because the representative run always includes all three.
func TestFromEvaluationRunOmitsTheOptionalEvents(t *testing.T) {
	result := representativeRun(RunFailed)
	result.CorrectionAttempts = 0
	result.HandoffDestination = ""
	item := FromEvaluationRun(result)
	for _, event := range item.Events {
		if event.Kind == KindRetry || event.Kind == KindHandoff {
			t.Fatalf("expected no %s event without a correction or a handoff", event.Kind)
		}
	}
	if report := Validate(item); !report.Valid {
		t.Fatalf("expected a valid trace, got findings %v", report.Findings)
	}
}

func representativeRun(outcome RunOutcome) RunResult {
	return RunResult{
		RunID:              "run-" + string(outcome),
		ScenarioID:         "scenario-1",
		SkillID:            "plan-issue",
		SkillVersion:       "0.1.0",
		GraphVersion:       1,
		Host:               "codex",
		Model:              "gpt-5",
		RepositoryRevision: "0123456789abcdef0123456789abcdef01234567",
		StartedAt:          "2026-09-09T12:00:00Z",
		FinishedAt:         "2026-09-09T12:00:05Z",
		ElapsedMillis:      5000,
		Outcome:            outcome,
		CorrectionAttempts: 2,
		HandoffDestination: "implement-issue",
	}
}

func equalJSON(t *testing.T, left, right []byte) bool {
	t.Helper()
	var a, b any
	if err := json.Unmarshal(left, &a); err != nil {
		t.Fatalf("decode produced trace: %v", err)
	}
	if err := json.Unmarshal(right, &b); err != nil {
		t.Fatalf("decode recorded trace: %v", err)
	}
	x, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("re-encode produced trace: %v", err)
	}
	y, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("re-encode recorded trace: %v", err)
	}
	return string(x) == string(y)
}
