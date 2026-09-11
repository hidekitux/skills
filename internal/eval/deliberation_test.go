package eval

import (
	"context"
	"io"
	"path/filepath"
	"sync"
	"testing"

	"github.com/hidekitux/skills/internal/trace"
)

type recordingHost struct {
	mu        sync.Mutex
	name      string
	available bool
	sandboxes []string
}

func (h *recordingHost) Name() string          { return h.name }
func (h *recordingHost) BinaryAvailable() bool { return h.available }
func (h *recordingHost) InstallSkills(context.Context, string, string, io.Writer, io.Writer) error {
	return nil
}

func (h *recordingHost) Run(_ context.Context, sandboxDir, _ string, out io.Writer) error {
	h.mu.Lock()
	h.sandboxes = append(h.sandboxes, sandboxDir)
	h.mu.Unlock()
	_, err := io.WriteString(out, "handing the verified result to write-tests")
	return err
}

func deliberationForTest() *DeliberationSpec {
	return &DeliberationSpec{
		Pattern:         "independent-candidates",
		Signals:         []string{"conflicting-hypotheses"},
		Reason:          "independent-evidence-needed",
		Independence:    "isolated-context",
		Authority:       "read-only",
		Concurrency:     "parallel-read-only",
		Bounds:          DeliberationBounds{MaxAgents: 2, MaxRetries: 1, MaxElapsedMillis: 5000, MaxInputTokens: 1000, MaxOutputTokens: 1000, MaxCostMicros: 10000},
		CandidateCount:  2,
		CompareBaseline: true,
		Judge:           "evidence-required-not-majority",
	}
}

func TestRunOneDeliberationUsesIndependentSandboxesAndRecordsComparison(t *testing.T) {
	scenario := &Scenario{
		ID:           "deliberation",
		Skill:        "debug-code",
		Kind:         KindPositive,
		Prompt:       "Investigate the reported failure and hand off the verified result.",
		Deliberation: deliberationForTest(),
		Expectations: Expectations{Handoff: "write-tests", TranscriptMust: []string{"write-tests"}},
	}
	host := &recordingHost{name: "codex", available: true}
	record := runOneForTest(t, scenario, host, &Options{Commit: "test-commit"})
	if record.Verdict != VerdictPass {
		t.Fatalf("verdict = %s, failures = %v", record.Verdict, record.Failures)
	}
	if record.Comparison == nil || record.Comparison.CandidateCount != 2 || record.Comparison.BaselineVerdict != VerdictPass {
		t.Fatalf("comparison = %#v, want baseline plus two candidates", record.Comparison)
	}
	if record.Deliberation == nil || record.Deliberation.Judge.Decision != "candidate-a" {
		t.Fatalf("deliberation = %#v, want evidence-backed first passing candidate", record.Deliberation)
	}
	host.mu.Lock()
	defer host.mu.Unlock()
	if len(host.sandboxes) != 3 {
		t.Fatalf("sandbox count = %d, want baseline plus two candidates", len(host.sandboxes))
	}
	seen := map[string]bool{}
	for _, sandbox := range host.sandboxes {
		seen[filepath.Clean(sandbox)] = true
	}
	if len(seen) != 3 {
		t.Fatalf("sandboxes are not isolated: %v", host.sandboxes)
	}
}

func TestRunOneOrdinaryScenarioRemainsSingleAgent(t *testing.T) {
	scenario := &Scenario{
		ID: "ordinary", Skill: "debug-code", Kind: KindPositive,
		Prompt:       "Investigate the reported failure and hand off the verified result.",
		Expectations: Expectations{Handoff: "write-tests", TranscriptMust: []string{"write-tests"}},
	}
	host := &recordingHost{name: "codex", available: true}
	record := runOneForTest(t, scenario, host, &Options{Commit: "test-commit"})
	if record.Verdict != VerdictPass || len(host.sandboxes) != 1 {
		t.Fatalf("verdict = %s, sandbox count = %d, want one single-agent run", record.Verdict, len(host.sandboxes))
	}
	if record.Comparison != nil || record.Deliberation != nil {
		t.Fatalf("ordinary scenario unexpectedly recorded deliberation: %#v", record.Comparison)
	}
}

func TestJudgeDeliberationDoesNotUseMajority(t *testing.T) {
	baseline := Record{Verdict: VerdictFail}
	candidates := []Record{{Verdict: VerdictFail}, {Verdict: VerdictFail}, {Verdict: VerdictPass}}
	index, selected := judgeDeliberation(baseline, candidates)
	if index != 2 || selected.Verdict != VerdictPass {
		t.Fatalf("judge selected candidate %d with verdict %s, want the evidence-backed passing candidate", index, selected.Verdict)
	}
}

func TestDeliberationTraceIsValidAndMetricsIncludeMarginalCost(t *testing.T) {
	scenario := &Scenario{ID: "deliberation", Skill: "debug-code", Expectations: Expectations{Handoff: "write-tests"}}
	spec := deliberationForTest()
	deliberation := deliberationTrace(spec, Record{Verdict: VerdictPass, ElapsedMillis: 1000}, []Record{{Verdict: VerdictPass}, {Verdict: VerdictFail}}, 0)
	record := Record{
		RunID: "run-deliberation", Scenario: scenario.ID, Skill: scenario.Skill, Host: "codex", Model: "gpt-5",
		Commit: "0123456789abcdef0123456789abcdef01234567", Verdict: VerdictPass,
		StartedAt: "2026-09-09T12:00:00Z", FinishedAt: "2026-09-09T12:00:02Z", ElapsedMillis: 2000,
		Deliberation: &deliberation,
	}
	item, err := traceForRecord(scenario, record, 1, "0.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if report := trace.Validate(item); !report.Valid {
		t.Fatalf("deliberation trace invalid: %v", report.Findings)
	}
	metrics := CalculateTraceMetrics([]trace.Trace{item})
	if metrics.DeliberationCount != 1 || metrics.CandidateCount != 2 || metrics.MarginalRetries != 0 {
		t.Fatalf("unexpected deliberation metrics: %#v", metrics)
	}
}
