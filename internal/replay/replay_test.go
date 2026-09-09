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

func TestReplayAcceptsCompleteGovernedLifecycle(t *testing.T) {
	set := lifecycleSet(
		lifecycleTrace("create-issue", "plan-issue", "change-issue", "success", "user-approval", "create-issue"),
		lifecycleTrace("plan-issue", "implement-issue", "verified-plan", "success", "post-plan-comment"),
		lifecycleTrace("implement-issue", "create-pr", "implementation-commits", "success", "create-issue-branch", "write-repository", "git-commit"),
		lifecycleTrace("create-pr", "review-pr", "pull-request", "success", "user-approval", "push-issue-branch", "edit-pull-request"),
		lifecycleTrace("review-pr", "", "", "", "record-review-findings"),
	)
	report := Replay(replayRepositoryRoot(t), set)
	if !report.Valid || report.Outcome != OutcomeValid {
		t.Fatalf("valid lifecycle rejected: %#v", report)
	}
	want := []string{"create_issue", "post_plan", "record_implementation", "pass_validation", "open_pull_request", "complete_review"}
	if len(report.Observations) != len(want) {
		t.Fatalf("observations = %#v, want %v", report.Observations, want)
	}
	for index, action := range want {
		if report.Observations[index].Action != action {
			t.Fatalf("observation[%d] = %#v, want %q", index, report.Observations[index], action)
		}
	}
}

func TestReplayReportsGraphOrderViolation(t *testing.T) {
	set := lifecycleSet(
		lifecycleTrace("create-issue", "create-pr", "change-issue", "success", "user-approval", "create-issue"),
	)
	report := Replay(replayRepositoryRoot(t), set)
	if report.Valid || report.Outcome != OutcomeViolation || len(report.Findings) == 0 || report.Findings[0].Invariant != "HandoffMatchesGraph" {
		t.Fatalf("order violation not reported: %#v", report)
	}
}

func TestReplayReportsMissingEvidenceAsIncomplete(t *testing.T) {
	item := lifecycleTrace("create-issue", "plan-issue", "change-issue", "success", "user-approval", "create-issue")
	item.Trace.Events = removeReplayEvidence(item.Trace.Events)
	item.Validation = trace.Validate(item.Trace)
	report := Replay(replayRepositoryRoot(t), TraceSet{Records: []TraceRecord{item}})
	if report.Valid || report.Outcome != OutcomeIncomplete || len(report.Findings) == 0 || report.Findings[0].Invariant != "IncompleteTraceIsNotSuccess" {
		t.Fatalf("missing evidence not reported as incomplete: %#v", report)
	}
}

func TestReplayReportsMissingValidationAsIncomplete(t *testing.T) {
	item := lifecycleTrace("create-issue", "plan-issue", "change-issue", "success", "user-approval", "create-issue")
	item.Trace.Events = removeReplayValidation(item.Trace.Events)
	item.Validation = trace.Validate(item.Trace)
	report := Replay(replayRepositoryRoot(t), lifecycleSet(item))
	if report.Valid || report.Outcome != OutcomeIncomplete || len(report.Findings) == 0 || report.Findings[0].Invariant != "IncompleteTraceIsNotSuccess" {
		t.Fatalf("missing validation not reported as incomplete: %#v", report)
	}
}

func TestReplayReportsUnknownToolAsIncomplete(t *testing.T) {
	item := lifecycleTrace("create-issue", "plan-issue", "change-issue", "success", "user-approval", "create-issue", "unknown-operation")
	report := Replay(replayRepositoryRoot(t), lifecycleSet(item))
	if report.Valid || report.Outcome != OutcomeIncomplete || len(report.Findings) == 0 || report.Findings[0].Invariant != "CanonicalToolObservation" {
		t.Fatalf("unknown tool was not reported as incomplete: %#v", report)
	}
}

func TestReplayReportsMissingRequiredToolAsIncomplete(t *testing.T) {
	item := lifecycleTrace("create-issue", "plan-issue", "change-issue", "success", "create-issue")
	report := Replay(replayRepositoryRoot(t), lifecycleSet(item))
	if report.Valid || report.Outcome != OutcomeIncomplete || len(report.Findings) == 0 || report.Findings[0].Invariant != "RequiredToolObservation" {
		t.Fatalf("missing required tool was not reported as incomplete: %#v", report)
	}
}

func TestReplayRejectsReadOnlyMutation(t *testing.T) {
	item := lifecycleTrace("create-issue", "plan-issue", "change-issue", "success", "git-commit")
	report := Replay(replayRepositoryRoot(t), lifecycleSet(item))
	if report.Valid || report.Outcome != OutcomeViolation || len(report.Findings) == 0 || report.Findings[0].Invariant != "ReadOnlyPhaseHasNoMutation" {
		t.Fatalf("read-only mutation not reported: %#v", report)
	}
}

func TestReplayPreservesInterruptedOutcome(t *testing.T) {
	item := lifecycleTrace("create-issue", "", "", "")
	item.Trace.Terminal = trace.Terminal{Status: trace.StatusInterrupted, Classification: trace.ClassificationInterruption, At: replayTestTime}
	item.Trace.Events[len(item.Trace.Events)-1].Status = trace.StatusInterrupted
	item.Trace.Events[len(item.Trace.Events)-1].Terminal = &item.Trace.Terminal
	item.Validation = trace.Validate(item.Trace)
	if !item.Validation.Valid {
		t.Logf("interrupted validation findings: %v", item.Validation.Findings)
	}
	report := Replay(replayRepositoryRoot(t), lifecycleSet(item))
	if report.Valid || report.Outcome != OutcomeInterrupted {
		t.Fatalf("interruption was not preserved: %#v", report)
	}
}

func TestReplayRejectsReviewLoopBeyondGraphBound(t *testing.T) {
	set := lifecycleSet(
		lifecycleTrace("create-issue", "plan-issue", "change-issue", "success", "user-approval", "create-issue"),
		lifecycleTrace("plan-issue", "implement-issue", "verified-plan", "success", "post-plan-comment"),
		lifecycleTrace("implement-issue", "create-pr", "implementation-commits", "success", "create-issue-branch", "write-repository", "git-commit"),
		lifecycleTrace("create-pr", "review-pr", "pull-request", "success", "user-approval", "push-issue-branch", "edit-pull-request"),
		reviewFindingsTrace(),
		fixTrace(1),
		reviewFindingsTrace(),
		fixTrace(2),
		reviewFindingsTrace(),
	)
	report := Replay(replayRepositoryRoot(t), set)
	if report.Valid || report.Outcome != OutcomeViolation || len(report.Findings) == 0 || report.Findings[0].Invariant != "ReviewLoopBounded" {
		t.Fatalf("review loop violation not reported: %#v", report)
	}
}

func lifecycleSet(items ...TraceRecord) TraceSet {
	for index := range items {
		items[index].Line = index + 1
	}
	return TraceSet{Records: items}
}

func lifecycleTrace(skill, destination, artifact, outcome string, tools ...string) TraceRecord {
	item := sampleReplayTrace()
	item.RunID = "run-" + skill
	item.SkillID = skill
	item.Events = []trace.Event{
		{Sequence: 1, Kind: trace.KindRunStarted, Status: trace.StatusStarted, At: replayTestTime},
		{Sequence: 2, Kind: trace.KindTodoTransition, Status: trace.StatusSuccess, At: replayTestTime, Todo: &trace.TodoTransition{ItemID: "phase", From: "in_progress", To: "completed", EvidenceRef: "phase-evidence"}},
		{Sequence: 3, Kind: trace.KindValidationOutcome, Status: trace.StatusSuccess, At: replayTestTime, Validation: &trace.Validation{Name: "phase-validation", Result: "success"}},
		{Sequence: 4, Kind: trace.KindEvidence, Status: trace.StatusSuccess, At: replayTestTime, Evidence: &trace.Evidence{Kind: "path", Ref: "evidence/phase"}},
	}
	sequence := 5
	for _, tool := range tools {
		item.Events = append(item.Events, trace.Event{Sequence: sequence, Kind: trace.KindToolOutcome, Status: trace.StatusSuccess, At: replayTestTime, Tool: &trace.ToolOutcome{Name: tool, Result: trace.StatusSuccess}})
		sequence++
	}
	if destination != "" {
		item.Events = append(item.Events, trace.Event{Sequence: sequence, Kind: trace.KindHandoff, Status: trace.StatusSuccess, At: replayTestTime, Handoff: &trace.Handoff{Destination: destination, Artifact: artifact, Outcome: outcome}})
		sequence++
	}
	item.Terminal = trace.Terminal{Status: trace.StatusSuccess, Classification: trace.ClassificationNone, At: replayTestTime}
	item.Events = append(item.Events, trace.Event{Sequence: sequence, Kind: trace.KindRunFinished, Status: trace.StatusSuccess, At: replayTestTime, Terminal: &item.Terminal})
	return TraceRecord{Trace: item, Line: 1, Validation: trace.Validate(item)}
}

func reviewFindingsTrace() TraceRecord {
	return lifecycleTrace("review-pr", "fix-pr", "review-findings", "findings", "record-review-findings")
}

func fixTrace(attempt int) TraceRecord {
	item := lifecycleTrace("fix-pr", "review-pr", "fixed-pull-request", "success", "write-repository", "git-commit", "push-issue-branch")
	finished := item.Trace.Events[len(item.Trace.Events)-1]
	item.Trace.Events = append(item.Trace.Events[:len(item.Trace.Events)-2],
		trace.Event{Sequence: finished.Sequence, Kind: trace.KindRetry, Status: trace.StatusFailed, At: replayTestTime, Retry: &trace.Retry{Attempt: attempt, MaxAttempts: 2, Reason: "review"}},
		trace.Event{Sequence: finished.Sequence + 1, Kind: trace.KindHandoff, Status: trace.StatusSuccess, At: replayTestTime, Handoff: &trace.Handoff{Destination: "review-pr", Artifact: "fixed-pull-request", Outcome: "success"}},
		trace.Event{Sequence: finished.Sequence + 2, Kind: trace.KindRunFinished, Status: trace.StatusSuccess, At: replayTestTime, Terminal: &item.Trace.Terminal},
	)
	item.Validation = trace.Validate(item.Trace)
	return item
}

func removeReplayEvidence(events []trace.Event) []trace.Event {
	result := make([]trace.Event, 0, len(events))
	for _, event := range events {
		if event.Evidence != nil {
			continue
		}
		result = append(result, event)
	}
	for index := range result {
		result[index].Sequence = index + 1
	}
	return result
}

func removeReplayValidation(events []trace.Event) []trace.Event {
	result := make([]trace.Event, 0, len(events))
	for _, event := range events {
		if event.Validation != nil {
			continue
		}
		result = append(result, event)
	}
	for index := range result {
		result[index].Sequence = index + 1
	}
	return result
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
