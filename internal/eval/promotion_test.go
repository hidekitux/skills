package eval

import (
	"strings"
	"testing"
)

const promotionTestRevision = "0123456789abcdef0123456789abcdef01234567"

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
				SkillSourceCommit: promotionTestRevision, PromptSHA: "prompt-" + scenario.ID,
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
	findings := checkStableSkillPromotion("demo", promotionTestRevision, promotionScenarioSet(), records)
	for _, finding := range findings {
		if strings.Contains(finding, want) {
			return
		}
	}
	t.Fatalf("findings = %v, want one containing %q", findings, want)
}

func TestStableSkillPromotionAcceptsTwoCompleteRuns(t *testing.T) {
	findings := checkStableSkillPromotion("demo", promotionTestRevision, promotionScenarioSet(), promotionRecords())
	if len(findings) != 0 {
		t.Fatalf("findings = %v, want none", findings)
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
		"stale revision": func(records map[string][]Record) {
			records["demo"][0].Commit = "fedcba9876543210fedcba9876543210fedcba9"
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
				"low score":           "want 3-5",
				"incomplete scenario": "missing required scenario",
				"stale revision":      "stale or incomplete revision evidence",
				"newer failure":       "has fail evidence",
				"bounded variance":    "bounded rubric variance",
			}[name]
			promotionFindingContains(t, records, want)
		})
	}
}
