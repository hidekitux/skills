package eval

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/hidekitux/skills/internal/trace"
)

func TestTraceForRecordAndMetricsUseStructuredFields(t *testing.T) {
	scenario := &Scenario{ID: "plan-issue-success", Skill: "plan-issue", Expectations: Expectations{Handoff: "implement-issue"}}
	record := Record{
		RunID: "run-1", Scenario: scenario.ID, Skill: scenario.Skill, Host: "codex", Model: "gpt-5",
		Commit: "0123456789abcdef0123456789abcdef01234567", Verdict: VerdictPass,
		HandoffObserved: true,
		StartedAt:       "2026-09-09T12:00:00Z", FinishedAt: "2026-09-09T12:00:02Z", ElapsedMillis: 2000,
	}
	item, err := traceForRecord(scenario, record, 1, "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if report := trace.Validate(item); !report.Valid {
		t.Fatalf("evaluation trace invalid: %v", report.Findings)
	}
	metrics := CalculateTraceMetrics([]trace.Trace{item})
	if metrics.SuccessCount != 1 || metrics.HandoffCount != 1 || metrics.ElapsedMillis != 2000 {
		t.Fatalf("unexpected metrics: %#v", metrics)
	}
	// Metrics consume the trace fields directly; changing a prose report is not
	// an input to this calculation.
	if _, err := json.Marshal(metrics); err != nil {
		t.Fatal(err)
	}
}

func TestTraceForFailedRecordDoesNotInventHandoffOrCompletion(t *testing.T) {
	scenario := &Scenario{ID: "failed", Skill: "plan-issue", Expectations: Expectations{Handoff: "implement-issue"}}
	record := Record{
		RunID: "run-failed", Scenario: scenario.ID, Skill: scenario.Skill, Host: "codex", Model: "gpt-5",
		Commit: "0123456789abcdef0123456789abcdef01234567", Verdict: VerdictFail,
		StartedAt: "2026-09-09T12:00:00Z", FinishedAt: "2026-09-09T12:00:01Z",
	}
	item, err := traceForRecord(scenario, record, 1, "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	for _, event := range item.Events {
		if event.Kind == trace.KindHandoff {
			t.Fatal("failed trace contains an unobserved handoff")
		}
		if event.Todo != nil && event.Todo.To == "completed" {
			t.Fatal("failed trace marks the Todo as completed")
		}
	}
	if report := trace.Validate(item); !report.Valid {
		t.Fatalf("evaluation trace invalid: %v", report.Findings)
	}
}

func TestTraceForInterruptedRecordUsesInterruptedTerminal(t *testing.T) {
	scenario := &Scenario{ID: "interrupted", Skill: "debug-code"}
	record := Record{
		RunID: "run-interrupted", Scenario: scenario.ID, Skill: scenario.Skill, Host: "codex", Model: "gpt-5",
		Commit: "0123456789abcdef0123456789abcdef01234567", Verdict: VerdictInterrupted,
		StartedAt: "2026-09-09T12:00:00Z", FinishedAt: "2026-09-09T12:00:01Z",
	}
	item, err := traceForRecord(scenario, record, 1, "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if item.Terminal.Status != trace.StatusInterrupted || item.Terminal.Classification != trace.ClassificationInterruption {
		t.Fatalf("terminal = %#v, want interrupted user interruption", item.Terminal)
	}
	if report := trace.Validate(item); !report.Valid {
		t.Fatalf("evaluation trace invalid: %v", report.Findings)
	}
}

func TestTraceForInfrastructureRecordUsesSafeDiagnosticIdentifiers(t *testing.T) {
	scenario := &Scenario{ID: "infrastructure", Skill: "debug-code"}
	record := Record{
		RunID: "run-infrastructure", Scenario: scenario.ID, Skill: scenario.Skill, Host: "codex", Model: "gpt-5",
		Commit: "0123456789abcdef0123456789abcdef01234567", Verdict: VerdictInfra,
		StartedAt: "2026-09-09T12:00:00Z", FinishedAt: "2026-09-09T12:00:01Z",
	}
	item, err := traceForRecord(scenario, record, 1, "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if report := trace.Validate(item); !report.Valid {
		t.Fatalf("evaluation trace invalid: %v", report.Findings)
	}
	for _, event := range item.Events {
		if event.Validation != nil && event.Validation.Result != "infrastructure-error" {
			t.Fatalf("validation result = %q, want infrastructure-error", event.Validation.Result)
		}
		if event.Validation != nil && len(event.Validation.Diagnostics) != 1 {
			t.Fatalf("diagnostics = %#v, want one diagnostic reference", event.Validation.Diagnostics)
		}
	}
}

func TestRunOptInWritesTraceAndMetrics(t *testing.T) {
	t.Setenv("EVAL_GITHUB_REPO", "hidekitux/skills")
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
	output := t.TempDir()
	var out, errOut bytes.Buffer
	opts := &Options{
		Root: root, Hosts: []string{"codex"}, ScenarioID: "plan-issue-success", TraceOutputDir: output,
		RunnerFor: func(name string) HostRunner {
			return &fakeHost{name: name, available: true, line: "handing to implement-issue"}
		},
	}
	if code := Run(context.Background(), opts, &out, &errOut); code != ExitOK {
		t.Fatalf("exit = %d\nstdout=%s\nstderr=%s", code, out.String(), errOut.String())
	}
	entries, err := os.ReadDir(output)
	if err != nil {
		t.Fatal(err)
	}
	var tracePath, metricsPath string
	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".jsonl" {
			tracePath = filepath.Join(output, entry.Name())
		}
		if filepath.Ext(entry.Name()) == ".json" {
			metricsPath = filepath.Join(output, entry.Name())
		}
	}
	if tracePath == "" || metricsPath == "" {
		t.Fatalf("trace outputs missing: %v", entries)
	}
	traces, err := trace.ReadJSONL(tracePath)
	if err != nil {
		t.Fatal(err)
	}
	if len(traces) != 1 || traces[0].SkillID != "plan-issue" || traces[0].GraphVersion != 1 {
		t.Fatalf("unexpected trace: %#v", traces)
	}
	content, err := os.ReadFile(metricsPath)
	if err != nil {
		t.Fatal(err)
	}
	var metrics TraceMetrics
	if err := json.Unmarshal(content, &metrics); err != nil {
		t.Fatal(err)
	}
	if metrics.TraceCount != 1 || metrics.SuccessCount != 1 {
		t.Fatalf("unexpected persisted metrics: %#v", metrics)
	}
}
