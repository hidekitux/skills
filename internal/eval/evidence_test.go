package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func retainedTestRecord() Record {
	return Record{
		RunID: "run-1", Scenario: "plan-issue-success", Skill: "plan-issue", Kind: KindPositive,
		Host: "claude-code", Model: "claude-sonnet-5", Commit: promotionTestRevision,
		InputDigest: promotionTestDigest, PromptSHA: "prompt-plan-issue-success", Verdict: VerdictPass,
		RubricReview: RubricComplete, RubricScores: promotionScores(4),
		Failures: []string{"private failure detail"}, InfraError: "private infrastructure detail",
		CorrectionsUsed: 1, FinishedAt: "2026-10-02T12:00:00Z",
	}
}

func TestWriteRetainedEvidenceKeepsOnlyRetainedFields(t *testing.T) {
	root := t.TempDir()
	if err := writeRetainedEvidence(root, "run-1", []Record{retainedTestRecord()}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(root, "evaluations", "evidence", "plan-issue", "run-1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	var record map[string]any
	if err := json.Unmarshal(content, &record); err != nil {
		t.Fatal(err)
	}
	allowed := map[string]bool{}
	for _, field := range retainedFields {
		allowed[field] = true
	}
	for field := range record {
		if !allowed[field] {
			t.Errorf("retained record carries %q, want only retained fields", field)
		}
	}
	for _, field := range requiredRetainedFields {
		if _, ok := record[field]; !ok {
			t.Errorf("retained record is missing required field %q", field)
		}
	}
	if strings.Contains(string(content), "private") {
		t.Fatalf("retained record leaks report detail: %s", content)
	}
}

func TestCheckRetainedEvidenceRejectsInvalidRecords(t *testing.T) {
	cases := []struct {
		name, path, content, want string
	}{
		{"extra field", "evaluations/evidence/plan-issue/run-1.jsonl",
			`{"run_id":"run-1","scenario":"s","skill":"plan-issue","host":"h","input_digest":"d","prompt_sha256":"p","verdict":"pass","rubric_review":"complete","finished_at":"2026-10-02T12:00:00Z","failures":["x"]}`,
			`carries field "failures" outside the retained set`},
		{"missing field", "evaluations/evidence/plan-issue/run-1.jsonl",
			`{"run_id":"run-1","scenario":"s","skill":"plan-issue","host":"h","prompt_sha256":"p","verdict":"pass","rubric_review":"complete","finished_at":"2026-10-02T12:00:00Z"}`,
			`missing required field "input_digest"`},
		{"wrong directory", "evaluations/evidence/debug-code/run-1.jsonl",
			`{"run_id":"run-1","scenario":"s","skill":"plan-issue","host":"h","input_digest":"d","prompt_sha256":"p","verdict":"pass","rubric_review":"complete","finished_at":"2026-10-02T12:00:00Z"}`,
			`records skill "plan-issue" under directory "debug-code"`},
		{"null field", "evaluations/evidence/plan-issue/run-1.jsonl",
			`{"run_id":"run-1","scenario":"s","skill":"plan-issue","host":"h","input_digest":null,"prompt_sha256":"p","verdict":"pass","rubric_review":"complete","finished_at":"2026-10-02T12:00:00Z"}`,
			`missing required field "input_digest"`},
		{"malformed line", "evaluations/evidence/plan-issue/run-1.jsonl", `not json`, "is not a JSON object"},
		{"not jsonl", "evaluations/evidence/plan-issue/run-1.md", "# report", "is not a .jsonl file"},
		{"flat file", "evaluations/evidence/run-1.jsonl", "", "is not at evaluations/evidence/<skill>/<run_id>.jsonl"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestFile(t, root, tc.path, tc.content+"\n")
			var findings []string
			checkRetainedEvidence(root, &findings)
			for _, finding := range findings {
				if strings.Contains(finding, tc.want) {
					return
				}
			}
			t.Fatalf("findings = %v, want one containing %q", findings, tc.want)
		})
	}
}

func TestCheckRetainedEvidenceAcceptsWrittenEvidence(t *testing.T) {
	root := t.TempDir()
	if err := writeRetainedEvidence(root, "run-1", []Record{retainedTestRecord()}); err != nil {
		t.Fatal(err)
	}
	var findings []string
	checkRetainedEvidence(root, &findings)
	if len(findings) != 0 {
		t.Fatalf("findings = %v, want none", findings)
	}
	if !hasStableEvidence(root, "plan-issue") {
		t.Fatal("hasStableEvidence = false, want true for a retained qualifying record")
	}
}

func TestLoadPromotionReportsCountsARetainedCopyOnce(t *testing.T) {
	root := t.TempDir()
	record := retainedTestRecord()
	if err := writeRetainedEvidence(root, record.RunID, []Record{record}); err != nil {
		t.Fatal(err)
	}
	content, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "evaluations/reports/run-1.jsonl", string(content)+"\n")
	runs, findings := loadPromotionReports(root)
	if len(findings) != 0 {
		t.Fatalf("findings = %v, want none", findings)
	}
	if len(runs["plan-issue"]) != 1 {
		t.Fatalf("records = %d, want the report and its retained copy counted once", len(runs["plan-issue"]))
	}
}

func TestLoadPromotionReportsReadsRetainedEvidence(t *testing.T) {
	root := t.TempDir()
	if err := writeRetainedEvidence(root, "run-1", []Record{retainedTestRecord()}); err != nil {
		t.Fatal(err)
	}
	runs, findings := loadPromotionReports(root)
	if len(findings) != 0 {
		t.Fatalf("findings = %v, want none", findings)
	}
	if len(runs["plan-issue"]) != 1 || runs["plan-issue"][0].InputDigest != promotionTestDigest {
		t.Fatalf("runs = %#v, want the retained plan-issue record", runs)
	}
}
