package replay

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/trace"
)

const (
	replayTestSHA  = "0123456789abcdef0123456789abcdef01234567"
	replayTestTime = "2026-09-10T12:00:00Z"
)

func replayRepositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func sampleReplayTrace() trace.Trace {
	terminal := trace.Terminal{Status: trace.StatusSuccess, Classification: trace.ClassificationNone, At: replayTestTime}
	return trace.Trace{
		SchemaVersion:      trace.CurrentSchemaVersion,
		RunID:              "replay-run-1",
		SkillID:            "create-issue",
		SkillVersion:       "0.1.0",
		GraphVersion:       1,
		Host:               "codex",
		Model:              "gpt-5",
		RepositoryRevision: replayTestSHA,
		StartedAt:          replayTestTime,
		Events: []trace.Event{
			{Sequence: 1, Kind: trace.KindRunStarted, Status: trace.StatusStarted, At: replayTestTime},
			{Sequence: 2, Kind: trace.KindEvidence, Status: trace.StatusSuccess, At: replayTestTime, Evidence: &trace.Evidence{Kind: "issue", Ref: "https://github.com/hidekitux/skills/issues/196"}},
			{Sequence: 3, Kind: trace.KindRunFinished, Status: trace.StatusSuccess, At: replayTestTime, Terminal: &terminal},
		},
		Terminal:  terminal,
		Redaction: trace.RedactionSummary{Mode: "allowlist", OmittedFields: []string{}, RetentionDays: trace.DefaultRetentionDays},
	}
}

func TestReadJSONLReadsCurrentTraceFixtures(t *testing.T) {
	path := filepath.Join(replayRepositoryRoot(t), "workflow", "trace-fixtures", "representative.jsonl")
	set, err := ReadJSONL(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Records) != 5 {
		t.Fatalf("record count = %d, want 5", len(set.Records))
	}
	for _, record := range set.Records {
		if !record.Validation.Valid {
			t.Fatalf("fixture record at line %d is invalid: %v", record.Line, record.Validation.Findings)
		}
	}
}

func TestReadJSONLPreservesPartialTraceForIncompleteClassification(t *testing.T) {
	item := sampleReplayTrace()
	item.Events = item.Events[:2]
	item.Terminal = trace.Terminal{}
	path := filepath.Join(t.TempDir(), "partial.jsonl")
	writeReplayJSONL(t, path, item)

	set, err := ReadJSONL(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Records) != 1 {
		t.Fatalf("record count = %d, want 1", len(set.Records))
	}
	if set.Records[0].Validation.Valid {
		t.Fatal("partial trace was reported as valid")
	}
	if !containsReplayFinding(set.Records[0].Validation.Findings, "last event must be run_finished") {
		t.Fatalf("partial trace lost terminal finding: %v", set.Records[0].Validation.Findings)
	}
}

func TestReadJSONLRejectsUnknownFieldsWithoutEchoingInput(t *testing.T) {
	path := filepath.Join(t.TempDir(), "unknown.jsonl")
	if err := os.WriteFile(path, []byte(`{"prompt":"private fixture content"}`+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := ReadJSONL(path)
	var inputErr *InputError
	if !errors.As(err, &inputErr) {
		t.Fatalf("error type = %T, want InputError", err)
	}
	if strings.Contains(err.Error(), "private fixture content") {
		t.Fatalf("input content leaked in error: %v", err)
	}
}

func TestReportJSONIsDeterministic(t *testing.T) {
	report := Report{
		Valid:        false,
		Outcome:      OutcomeViolation,
		Spec:         "CrossSkillWorkflow",
		StepsChecked: 2,
		Observations: []Observation{{Action: "create_issue", SkillID: "create-issue", TraceIndex: 0, Line: 1, EventSequence: 2}},
		Findings:     []Finding{{Invariant: "HandoffMatchesGraph", Category: "ordering", Message: "handoff does not match graph", TraceIndex: 0, Line: 1, EventSequence: 2}},
	}
	first, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("report encoding changed: %s != %s", first, second)
	}
}

func writeReplayJSONL(t *testing.T, path string, item trace.Trace) {
	t.Helper()
	data, err := json.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
}

func containsReplayFinding(findings []string, want string) bool {
	for _, finding := range findings {
		if strings.Contains(finding, want) {
			return true
		}
	}
	return false
}
