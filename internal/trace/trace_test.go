package trace

import (
	"path/filepath"
	"strings"
	"testing"
)

const testSHA = "0123456789abcdef0123456789abcdef01234567"
const testTime = "2026-09-09T12:00:00Z"

func validTrace() Trace {
	terminal := Terminal{Status: "success", Classification: ClassificationNone, At: testTime}
	return Trace{
		SchemaVersion: CurrentSchemaVersion, RunID: "run-1", ScenarioID: "scenario-1",
		SkillID: "plan-issue", SkillVersion: "0.1.0", GraphVersion: 1,
		Host: "codex", Model: "gpt-5", RepositoryRevision: testSHA, StartedAt: testTime,
		Events: []Event{
			{Sequence: 1, Kind: KindRunStarted, Status: StatusStarted, At: testTime},
			{Sequence: 2, Kind: KindTodoTransition, Status: StatusSuccess, At: testTime, Todo: &TodoTransition{ItemID: "discover", From: "pending", To: "in_progress"}},
			{Sequence: 3, Kind: KindEvidence, Status: StatusSuccess, At: testTime, Evidence: &Evidence{Kind: "commit", Ref: testSHA}},
			{Sequence: 4, Kind: KindRunFinished, Status: StatusSuccess, At: testTime, Terminal: &terminal},
		},
		Terminal:  terminal,
		Redaction: RedactionSummary{Mode: "allowlist", RetentionDays: DefaultRetentionDays},
	}
}

func TestValidateAcceptsSuccessTrace(t *testing.T) {
	if report := Validate(validTrace()); !report.Valid {
		t.Fatalf("valid trace rejected: %v", report.Findings)
	}
}

func TestValidateAcceptsSafeDiagnosticReference(t *testing.T) {
	trace := validTrace()
	trace.Events[2] = Event{
		Sequence: 3, Kind: KindValidationOutcome, Status: StatusFailed, At: testTime,
		Validation: &Validation{
			Name: "check-repository", Result: "failed", Classification: ClassificationDeterministic,
			Diagnostics: []DiagnosticRef{{Producer: "check-repository", Code: "check-repository.failed"}},
		},
	}
	if report := Validate(trace); !report.Valid {
		t.Fatalf("safe diagnostic reference rejected: %v", report.Findings)
	}
}

func TestValidateRejectsUnsafeDiagnosticReference(t *testing.T) {
	trace := validTrace()
	trace.Events[2] = Event{
		Sequence: 3, Kind: KindValidationOutcome, Status: StatusFailed, At: testTime,
		Validation: &Validation{
			Name: "check-repository", Result: "failed", Classification: ClassificationDeterministic,
			Diagnostics: []DiagnosticRef{{Producer: "check-repository", Code: "token=secret"}},
		},
	}
	if report := Validate(trace); report.Valid || !contains(report.Findings, "diagnostics[0].code is invalid") {
		t.Fatalf("unsafe diagnostic reference accepted: valid=%t findings=%v", report.Valid, report.Findings)
	}
}

func TestValidateDistinguishesTerminalStates(t *testing.T) {
	cases := []struct{ name, status, classification string }{
		{"deterministic failure", "failed", ClassificationDeterministic},
		{"behavioral failure", "failed", ClassificationBehavioral},
		{"infrastructure error", "infrastructure_error", ClassificationInfrastructure},
		{"user interruption", "interrupted", ClassificationInterruption},
		{"intentional skip", "skipped", ClassificationSkip},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trace := validTrace()
			trace.Terminal = Terminal{Status: tc.status, Classification: tc.classification, At: testTime}
			trace.Events[len(trace.Events)-1].Status = map[string]string{"infrastructure_error": StatusError, "failed": StatusFailed, "interrupted": StatusInterrupted, "skipped": StatusSkipped}[tc.status]
			trace.Events[len(trace.Events)-1].Terminal = &trace.Terminal
			if report := Validate(trace); !report.Valid {
				t.Fatalf("state rejected: %v", report.Findings)
			}
		})
	}
}

func TestValidateRejectsUnsafeShape(t *testing.T) {
	cases := []struct {
		name, want string
		mutate     func(*Trace)
	}{
		{"non increasing sequence", "sequence must increase", func(trace *Trace) { trace.Events[2].Sequence = 2 }},
		{"missing terminal event", "last event must be run_finished", func(trace *Trace) { trace.Events = trace.Events[:2] }},
		{"unbounded retry", "within a positive max_attempts bound", func(trace *Trace) {
			trace.Events[2] = Event{Sequence: 3, Kind: KindRetry, Status: StatusFailed, At: testTime, Retry: &Retry{Attempt: 2, MaxAttempts: 1, Reason: "review"}}
		}},
		{"private evidence URL", "unsafe or invalid", func(trace *Trace) {
			trace.Events[2].Evidence.Ref = "https://internal.example/run/1"
			trace.Events[2].Evidence.Kind = "issue"
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			trace := validTrace()
			tc.mutate(&trace)
			if report := Validate(trace); report.Valid || !contains(report.Findings, tc.want) {
				t.Fatalf("want %q, got valid=%t findings=%v", tc.want, report.Valid, report.Findings)
			}
		})
	}
}

func TestSanitizeRedactsSecretsAndOmitsUnsafeEvidence(t *testing.T) {
	trace := validTrace()
	trace.Model = "model token=ghp_example_secret"
	trace.Events[2].Evidence.Ref = "https://internal.example/private"
	clean, err := Sanitize(trace)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(clean.Model, "ghp_example_secret") {
		t.Fatalf("secret survived redaction: %q", clean.Model)
	}
	if len(clean.Events) != 3 {
		t.Fatalf("unsafe evidence event was not omitted: %d events", len(clean.Events))
	}
	if clean.Redaction.RedactedCount == 0 || !contains(clean.Redaction.OmittedFields, "evidence.ref") {
		t.Fatalf("redaction evidence missing: %#v", clean.Redaction)
	}
	if report := Validate(clean); !report.Valid {
		t.Fatalf("sanitized trace invalid: %v", report.Findings)
	}
}

func TestWriteAndReadJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "traces.jsonl")
	if err := WriteJSONL(path, []Trace{validTrace()}); err != nil {
		t.Fatal(err)
	}
	traces, err := ReadJSONL(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(traces) != 1 || traces[0].RunID != "run-1" {
		t.Fatalf("unexpected traces: %#v", traces)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
