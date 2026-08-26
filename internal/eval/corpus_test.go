package eval

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func writeTestFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fullRubric returns a scenario rubric with all seven dimensions populated.
func fullRubric() Rubric {
	return Rubric{
		TriggerSelection:    "selects the right workflow",
		TaskCompletion:      "reaches the outcome",
		EvidenceQuality:     "cites evidence",
		ScopeControl:        "stays in scope",
		Safety:              "no leaks",
		UserCorrectionCount: "no corrections",
		HandoffQuality:      "names the next owner",
	}
}

// baseScenario returns a minimal valid single-skill scenario.
func baseScenario() *Scenario {
	return &Scenario{
		ID:     "plan-issue-success",
		Skill:  "plan-issue",
		Kind:   KindPositive,
		Title:  "Plan a ready issue",
		Prompt: "Produce an ordered plan for the ready issue before coding.",
		Expectations: Expectations{
			Handoff:        "implement-issue",
			TranscriptMust: []string{"implement-issue"},
		},
		Rubric: fullRubric(),
	}
}

// scaffoldEval builds a minimal repository shape valid under CheckCorpus and
// returns its root. Fixture content defaults to one README file per fixture.
func scaffoldEval(t *testing.T, catalogSkills []map[string]string, scenarios []*Scenario, fixtureFiles map[string]map[string]string) string {
	t.Helper()
	root := t.TempDir()

	entries := make([]string, 0, len(catalogSkills))
	for _, skill := range catalogSkills {
		name := skill["name"]
		status := skill["status"]
		if status == "" {
			status = "experimental"
		}
		entries = append(entries, "  - name: "+name+"\n    owner: hidekitux\n    status: "+status+"\n    license: Apache-2.0\n    version: 0.1.0\n    layer: process\n")
	}
	writeTestFile(t, root, "CATALOG.yml", "catalog_version: 1\nlicense: Apache-2.0\nskills:\n"+strings.Join(entries, ""))

	for index, sc := range scenarios {
		content, err := yaml.Marshal(sc)
		if err != nil {
			t.Fatal(err)
		}
		// Use a unique file name per entry so duplicate-id scenarios both
		// exist as files for the corpus check to detect.
		writeTestFile(t, root, filepath.Join("evaluations", "scenarios", sc.Skill, fmt.Sprintf("%02d-%s.yaml", index, sc.ID)), string(content))
	}

	for key, files := range fixtureFiles {
		for rel, content := range files {
			writeTestFile(t, root, filepath.Join("evaluations", "fixtures", key, rel), content)
		}
	}
	return root
}

func skillEntry(name, status string) map[string]string {
	return map[string]string{"name": name, "status": status}
}

func runCheckCorpus(t *testing.T, root string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := CheckCorpus(root, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestCheckCorpusPassesCompleteCorpus(t *testing.T) {
	positive := baseScenario()
	positive.Fixture = "plan-fixture"
	root := scaffoldEval(t,
		[]map[string]string{skillEntry("plan-issue", "experimental")},
		[]*Scenario{
			positive,
			{ID: "plan-issue-boundary", Skill: "plan-issue", Kind: KindBoundary, Title: "Stop without Scope", Prompt: "Plan the draft issue.", Expectations: Expectations{Handoff: "blocked-ask"}, Rubric: fullRubric()},
		},
		map[string]map[string]string{"plan-fixture": {"README.md": "# fixture"}},
	)

	code, _, errOut := runCheckCorpus(t, root)
	if code != 0 {
		t.Fatalf("expected pass, got %d: %s", code, errOut)
	}
}

func TestCheckCorpusRejectsMissingFailureScenario(t *testing.T) {
	root := scaffoldEval(t,
		[]map[string]string{skillEntry("plan-issue", "experimental")},
		[]*Scenario{baseScenario()},
		nil,
	)
	code, _, errOut := runCheckCorpus(t, root)
	if code != 1 {
		t.Fatalf("expected failure, got %d", code)
	}
	if !strings.Contains(errOut, "has no failure or boundary scenario") {
		t.Fatalf("missing coverage finding:\n%s", errOut)
	}
}

func TestCheckCorpusRejectsMissingPositiveScenario(t *testing.T) {
	root := scaffoldEval(t,
		[]map[string]string{skillEntry("plan-issue", "experimental")},
		[]*Scenario{
			{ID: "plan-issue-boundary", Skill: "plan-issue", Kind: KindBoundary, Title: "Boundary", Prompt: "Plan the draft issue.", Expectations: Expectations{Handoff: "blocked-ask"}, Rubric: fullRubric()},
		},
		nil,
	)
	code, _, errOut := runCheckCorpus(t, root)
	if code != 1 {
		t.Fatalf("expected failure, got %d", code)
	}
	if !strings.Contains(errOut, "has no positive success scenario") {
		t.Fatalf("missing coverage finding:\n%s", errOut)
	}
}

func TestCheckCorpusRejectsPromptRevealingSkill(t *testing.T) {
	leaky := baseScenario()
	leaky.Prompt = "Use plan-issue to plan this change."
	root := scaffoldEval(t,
		[]map[string]string{skillEntry("plan-issue", "experimental")},
		[]*Scenario{leaky},
		nil,
	)
	code, _, errOut := runCheckCorpus(t, root)
	if code != 1 {
		t.Fatalf("expected failure, got %d", code)
	}
	if !strings.Contains(errOut, "revealing trigger selection") {
		t.Fatalf("missing leak finding:\n%s", errOut)
	}
}

func TestCheckCorpusRejectsPromptRevealingHandoff(t *testing.T) {
	leaky := baseScenario()
	leaky.Prompt = "Hand the plan to implement-issue afterwards."
	root := scaffoldEval(t,
		[]map[string]string{skillEntry("plan-issue", "experimental")},
		[]*Scenario{leaky},
		nil,
	)
	code, _, errOut := runCheckCorpus(t, root)
	if code != 1 {
		t.Fatalf("expected failure, got %d", code)
	}
	if !strings.Contains(errOut, "reveals the expected handoff") {
		t.Fatalf("missing leak finding:\n%s", errOut)
	}
}

func TestCheckCorpusRejectsDuplicateIDs(t *testing.T) {
	positive := baseScenario()
	duplicate := baseScenario()
	duplicate.Title = "Duplicate"
	duplicate.Prompt = "Another prompt."
	root := scaffoldEval(t,
		[]map[string]string{skillEntry("plan-issue", "experimental")},
		[]*Scenario{
			positive,
			duplicate,
			{ID: "plan-issue-boundary", Skill: "plan-issue", Kind: KindBoundary, Title: "Boundary", Prompt: "Plan the draft issue.", Expectations: Expectations{Handoff: "blocked-ask"}, Rubric: fullRubric()},
		},
		nil,
	)
	code, _, errOut := runCheckCorpus(t, root)
	if code != 1 {
		t.Fatalf("expected failure, got %d", code)
	}
	if !strings.Contains(errOut, "is duplicated") {
		t.Fatalf("missing duplicate finding:\n%s", errOut)
	}
}

func TestCheckCorpusRejectsMissingFixture(t *testing.T) {
	sc := baseScenario()
	sc.Fixture = "does-not-exist"
	root := scaffoldEval(t,
		[]map[string]string{skillEntry("plan-issue", "experimental")},
		[]*Scenario{sc},
		nil,
	)
	code, _, errOut := runCheckCorpus(t, root)
	if code != 1 {
		t.Fatalf("expected failure, got %d", code)
	}
	if !strings.Contains(errOut, `fixture "does-not-exist" does not resolve`) {
		t.Fatalf("missing fixture finding:\n%s", errOut)
	}
}

func TestCheckCorpusRejectsStagesWithoutE2E(t *testing.T) {
	sc := baseScenario()
	sc.Stages = []Stage{{Skill: "plan-issue", Prompt: "stage prompt"}}
	sc.Prompt = ""
	root := scaffoldEval(t,
		[]map[string]string{skillEntry("plan-issue", "experimental")},
		[]*Scenario{sc},
		nil,
	)
	code, _, errOut := runCheckCorpus(t, root)
	if code != 1 {
		t.Fatalf("expected failure, got %d", code)
	}
	if !strings.Contains(errOut, "stages are only allowed for skill: e2e") {
		t.Fatalf("missing stages finding:\n%s", errOut)
	}
}

func TestCheckCorpusRejectsIncompleteRubric(t *testing.T) {
	sc := baseScenario()
	sc.Rubric.Safety = ""
	root := scaffoldEval(t,
		[]map[string]string{skillEntry("plan-issue", "experimental")},
		[]*Scenario{sc},
		nil,
	)
	code, _, errOut := runCheckCorpus(t, root)
	if code != 1 {
		t.Fatalf("expected failure, got %d", code)
	}
	if !strings.Contains(errOut, "rubric must define all seven dimensions") {
		t.Fatalf("missing rubric finding:\n%s", errOut)
	}
}

func TestCheckCorpusStableStatusRequiresEvidence(t *testing.T) {
	stable := skillEntry("plan-issue", "stable")
	root := scaffoldEval(t,
		[]map[string]string{stable},
		[]*Scenario{
			baseScenario(),
			{ID: "plan-issue-boundary", Skill: "plan-issue", Kind: KindBoundary, Title: "Boundary", Prompt: "Plan the draft issue.", Expectations: Expectations{Handoff: "blocked-ask"}, Rubric: fullRubric()},
		},
		map[string]map[string]string{},
	)
	code, _, errOut := runCheckCorpus(t, root)
	if code != 1 {
		t.Fatalf("expected failure for stable without evidence, got %d", code)
	}
	if !strings.Contains(errOut, "stable but has no passing evaluation evidence") {
		t.Fatalf("missing promotion-gate finding:\n%s", errOut)
	}

	writeTestFile(t, root, "evaluations/reports/run-1.jsonl",
		`{"scenario":"plan-issue-success","skill":"plan-issue","host":"codex","verdict":"pass"}`+"\n")
	code, _, errOut = runCheckCorpus(t, root)
	if code != 0 {
		t.Fatalf("expected pass with evidence, got %d: %s", code, errOut)
	}
}

func TestCheckCorpusRequiresAtLeastOneScenario(t *testing.T) {
	root := scaffoldEval(t,
		[]map[string]string{skillEntry("plan-issue", "experimental")},
		nil,
		nil,
	)
	code, _, errOut := runCheckCorpus(t, root)
	if code != 1 {
		t.Fatalf("expected failure, got %d", code)
	}
	if !strings.Contains(errOut, "no scenarios found") {
		t.Fatalf("missing empty-corpus finding:\n%s", errOut)
	}
}
