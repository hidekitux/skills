package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func writeRecords(t *testing.T, path string, records []Record) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	encoder := json.NewEncoder(file)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			file.Close()
			t.Fatal(err)
		}
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestCompareReportsAcceptsPreservedPassAndRubric(t *testing.T) {
	dir := t.TempDir()
	fullPath := filepath.Join(dir, "full.jsonl")
	compactPath := filepath.Join(dir, "compact.jsonl")
	scores := map[string]int{
		"trigger_selection": 4, "task_completion": 5, "evidence_quality": 4,
		"scope_control": 4, "safety": 5, "user_correction_count": 4, "handoff_quality": 4,
	}
	base := Record{Scenario: "demo", Skill: "debug-code", Host: "codex", PromptSHA: "same", Verdict: VerdictPass, RubricReview: RubricComplete, RubricScores: scores}
	writeRecords(t, fullPath, []Record{base})
	base.RubricScores = map[string]int{
		"trigger_selection": 4, "task_completion": 5, "evidence_quality": 4,
		"scope_control": 4, "safety": 4, "user_correction_count": 4, "handoff_quality": 4,
	}
	writeRecords(t, compactPath, []Record{base})
	report, err := CompareReports(fullPath, compactPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Results) != 1 || report.Results[0].Status != "pass" {
		t.Fatalf("comparison = %+v, want one pass", report.Results)
	}
}

func TestCompareReportsAcceptsEqualDeterministicOutcomesWithoutRubric(t *testing.T) {
	dir := t.TempDir()
	fullPath := filepath.Join(dir, "full.jsonl")
	compactPath := filepath.Join(dir, "compact.jsonl")

	for _, verdict := range []string{VerdictPass, VerdictFail} {
		t.Run(verdict, func(t *testing.T) {
			base := Record{
				Scenario: "demo", Skill: "debug-code", Host: "codex",
				PromptSHA: "same", Verdict: verdict, RubricReview: RubricNA,
			}
			writeRecords(t, fullPath, []Record{base})
			writeRecords(t, compactPath, []Record{base})

			report, err := CompareReports(fullPath, compactPath)
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Results) != 1 || report.Results[0].Status != "pass" {
				t.Fatalf("comparison = %+v, want one pass", report.Results)
			}
		})
	}
}

func TestCompareReportsMarksInterruptedSourcesInconclusive(t *testing.T) {
	tests := []struct {
		name           string
		fullVerdict    string
		compactVerdict string
		fullReason     string
		compactReason  string
	}{
		{
			name:           "full source",
			fullVerdict:    VerdictInterrupted,
			compactVerdict: VerdictPass,
			fullReason:     "full source was interrupted",
		},
		{
			name:           "compact source",
			fullVerdict:    VerdictPass,
			compactVerdict: VerdictInterrupted,
			compactReason:  "compact source was interrupted",
		},
		{
			name:           "both sources",
			fullVerdict:    VerdictInterrupted,
			compactVerdict: VerdictInterrupted,
			fullReason:     "full source was interrupted",
			compactReason:  "compact source was interrupted",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			fullPath := filepath.Join(dir, "full.jsonl")
			compactPath := filepath.Join(dir, "compact.jsonl")
			full := Record{
				Scenario: "demo", Skill: "debug-code", Host: "codex",
				PromptSHA: "same", Verdict: test.fullVerdict, RubricReview: RubricNA,
			}
			compact := full
			compact.Verdict = test.compactVerdict
			writeRecords(t, fullPath, []Record{full})
			writeRecords(t, compactPath, []Record{compact})

			report, err := CompareReports(fullPath, compactPath)
			if err != nil {
				t.Fatal(err)
			}
			result := report.Results[0]
			if result.Status != "inconclusive" {
				t.Fatalf("status = %q, want inconclusive; reasons = %v", result.Status, result.Reasons)
			}
			if result.FullVerdict != test.fullVerdict || result.CompactVerdict != test.compactVerdict {
				t.Fatalf("verdicts = (%q, %q), want (%q, %q)", result.FullVerdict, result.CompactVerdict, test.fullVerdict, test.compactVerdict)
			}
			for _, reason := range []string{test.fullReason, test.compactReason} {
				if reason != "" && !containsReason(result.Reasons, reason) {
					t.Fatalf("reasons = %v, want %q", result.Reasons, reason)
				}
			}
		})
	}
}

func TestCompareReportsRejectsDeterministicRegression(t *testing.T) {
	dir := t.TempDir()
	fullPath := filepath.Join(dir, "full.jsonl")
	compactPath := filepath.Join(dir, "compact.jsonl")
	base := Record{Scenario: "demo", Skill: "plan-issue", Host: "claude-code", PromptSHA: "same", Verdict: VerdictPass, RubricReview: RubricNA}
	writeRecords(t, fullPath, []Record{base})
	base.Verdict = VerdictFail
	base.Failures = []string{"handoff missing"}
	writeRecords(t, compactPath, []Record{base})
	report, err := CompareReports(fullPath, compactPath)
	if err != nil {
		t.Fatal(err)
	}
	if report.Results[0].Status != "fail" {
		t.Fatalf("comparison = %+v, want fail for deterministic regression", report.Results[0])
	}
	base.RubricReview = RubricComplete
	base.RubricScores = map[string]int{
		"trigger_selection": 3, "task_completion": 3, "evidence_quality": 3,
		"scope_control": 3, "safety": 3, "user_correction_count": 3, "handoff_quality": 3,
	}
	writeRecords(t, fullPath, []Record{{
		Scenario: "demo", Skill: "plan-issue", Host: "claude-code", PromptSHA: "same",
		Verdict: VerdictPass, RubricReview: RubricComplete, RubricScores: map[string]int{
			"trigger_selection": 5, "task_completion": 5, "evidence_quality": 5,
			"scope_control": 5, "safety": 5, "user_correction_count": 5, "handoff_quality": 5,
		},
	}})
	writeRecords(t, compactPath, []Record{base})
	report, err = CompareReports(fullPath, compactPath)
	if err != nil {
		t.Fatal(err)
	}
	if report.Results[0].Status != "fail" {
		t.Fatalf("comparison = %+v, want fail", report.Results[0])
	}
}
