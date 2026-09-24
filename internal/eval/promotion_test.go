package eval

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	promotionTestRevision = "0123456789abcdef0123456789abcdef01234567"
	promotionTestDigest   = "input-digest-demo"
)

func promotionScenarioSet() []*Scenario {
	return []*Scenario{
		{ID: "demo-success", Skill: "demo", Kind: KindPositive},
		{ID: "demo-failure", Skill: "demo", Kind: KindNegative},
		{ID: "demo-boundary", Skill: "demo", Kind: KindBoundary},
	}
}

func promotionRecords() map[string][]Record {
	makeRun := func(runID, finishedAt string) []Record {
		records := make([]Record, 0, 3)
		for _, scenario := range promotionScenarioSet() {
			records = append(records, Record{
				RunID: runID, Scenario: scenario.ID, Skill: "demo", Kind: scenario.Kind,
				Host: "codex", Model: "gpt-5", Commit: promotionTestRevision,
				SkillSourceCommit: promotionTestRevision, InputDigest: promotionTestDigest, PromptSHA: "prompt-" + scenario.ID,
				Verdict: VerdictPass, RubricReview: RubricComplete, RubricScores: promotionScores(4),
				FinishedAt: finishedAt,
			})
		}
		return records
	}
	return map[string][]Record{
		"demo": append(makeRun("run-old", "2026-09-10T12:00:00Z"), makeRun("run-new", "2026-09-11T12:00:00Z")...),
	}
}

func promotionScores(score int) map[string]int {
	scores := map[string]int{}
	for _, dimension := range scoreOrder {
		scores[dimension] = score
	}
	return scores
}

func promotionFindingContains(t *testing.T, records map[string][]Record, want string) {
	t.Helper()
	findings := checkStableSkillPromotion("demo", promotionTestDigest, promotionScenarioSet(), records)
	for _, finding := range findings {
		if strings.Contains(finding, want) {
			return
		}
	}
	t.Fatalf("findings = %v, want one containing %q", findings, want)
}

func TestStableSkillPromotionAcceptsTwoCompleteRuns(t *testing.T) {
	findings := checkStableSkillPromotion("demo", promotionTestDigest, promotionScenarioSet(), promotionRecords())
	if len(findings) != 0 {
		t.Fatalf("findings = %v, want none", findings)
	}
}

func TestStableSkillPromotionAcceptsRecordsFromAnotherCommit(t *testing.T) {
	records := promotionRecords()
	for index := range records["demo"] {
		records["demo"][index].Commit = "fedcba9876543210fedcba9876543210fedcba98"
		records["demo"][index].SkillSourceCommit = "fedcba9876543210fedcba9876543210fedcba98"
	}
	findings := checkStableSkillPromotion("demo", promotionTestDigest, promotionScenarioSet(), records)
	if len(findings) != 0 {
		t.Fatalf("findings = %v, want none for records with the current input digest", findings)
	}
}

func TestStableSkillPromotionRejectsUnmetEvidence(t *testing.T) {
	tests := map[string]func(map[string][]Record){
		"low score": func(records map[string][]Record) {
			records["demo"][0].RubricScores["safety"] = 2
		},
		"incomplete scenario": func(records map[string][]Record) {
			records["demo"] = append(records["demo"][:5], records["demo"][6:]...)
		},
		"stale input digest": func(records map[string][]Record) {
			records["demo"][0].InputDigest = "input-digest-before-a-relevant-commit"
		},
		"missing input digest": func(records map[string][]Record) {
			records["demo"][3].InputDigest = ""
		},
		"newer failure": func(records map[string][]Record) {
			records["demo"][3].Verdict = VerdictFail
		},
		"bounded variance": func(records map[string][]Record) {
			for _, dimension := range scoreOrder {
				records["demo"][0].RubricScores[dimension] = 5
			}
			records["demo"][0].RubricScores["safety"] = 3
			records["demo"][3].RubricScores["safety"] = 5
		},
	}
	for name, mutate := range tests {
		t.Run(name, func(t *testing.T) {
			records := promotionRecords()
			mutate(records)
			want := map[string]string{
				"low score":            "want 3-5",
				"incomplete scenario":  "missing required scenario",
				"stale input digest":   "stale or missing input digest evidence",
				"missing input digest": "stale or missing input digest evidence",
				"newer failure":        "has fail evidence",
				"bounded variance":     "bounded rubric variance",
			}[name]
			promotionFindingContains(t, records, want)
		})
	}
}

func TestLoadPromotionReportsReadsNestedReports(t *testing.T) {
	root := t.TempDir()
	reportPath := filepath.Join(root, "evaluations", "reports", "run-1", "results.jsonl")
	if err := os.MkdirAll(filepath.Dir(reportPath), 0o755); err != nil {
		t.Fatal(err)
	}
	record := Record{RunID: "run-1", Skill: "demo", Scenario: "demo-success"}
	content, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(reportPath, append(content, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}

	runs, findings := loadPromotionReports(root)
	if len(findings) != 0 {
		t.Fatalf("findings = %v, want none", findings)
	}
	if len(runs["demo"]) != 1 || runs["demo"][0].RunID != "run-1" {
		t.Fatalf("runs = %#v, want nested report for run-1", runs)
	}
}

// writePromotionReports retains two complete runs of the demo skill that
// carry digest as their input digest.
func writePromotionReports(t *testing.T, root, digest string) {
	t.Helper()
	var lines []string
	for _, run := range []struct{ id, finishedAt string }{
		{"run-old", "2026-09-10T12:00:00Z"},
		{"run-new", "2026-09-11T12:00:00Z"},
	} {
		content, err := json.Marshal(Record{
			RunID: run.id, Scenario: "demo-success", Skill: "demo", Kind: KindPositive,
			Host: "codex", Commit: promotionTestRevision, SkillSourceCommit: promotionTestRevision,
			InputDigest: digest, PromptSHA: "prompt-demo-success", Verdict: VerdictPass,
			RubricReview: RubricComplete, RubricScores: promotionScores(4), FinishedAt: run.finishedAt,
		})
		if err != nil {
			t.Fatal(err)
		}
		lines = append(lines, string(content))
	}
	writeInputFile(t, root, "evaluations/reports/demo.jsonl", strings.Join(lines, "\n")+"\n")
}

func TestPromotionFindingsScopesFreshnessToInputs(t *testing.T) {
	root := newInputRepository(t)
	writeInputFile(t, root, "CATALOG.yml", "skills:\n  - name: demo\n    status: stable\n  - name: other\n    status: experimental\n")
	commitAll(t, root, "catalog")
	digest, err := InputDigest(root, root, "demo")
	if err != nil {
		t.Fatal(err)
	}
	writePromotionReports(t, root, digest)
	commitAll(t, root, "retain evaluation reports")
	if findings := PromotionFindings(root); len(findings) != 0 {
		t.Fatalf("findings = %v, want none for fresh evidence", findings)
	}

	writeInputFile(t, root, "docs/evaluation.md", "changed docs\n")
	commitAll(t, root, "unrelated documentation change")
	if findings := PromotionFindings(root); len(findings) != 0 {
		t.Fatalf("findings = %v, want none after an unrelated commit", findings)
	}

	writeInputFile(t, root, "skills/process/demo/SKILL.md", "---\nname: demo\n---\nchanged\n")
	commitAll(t, root, "relevant skill change")
	findings := PromotionFindings(root)
	if len(findings) == 0 {
		t.Fatal("findings = none, want stale evidence after a commit to the skill directory")
	}
	for _, finding := range findings {
		if !strings.Contains(finding, "stale or missing input digest evidence") {
			t.Fatalf("finding %q, want only stale input digest findings", finding)
		}
	}
}
