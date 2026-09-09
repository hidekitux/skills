package eval

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func TestPromotedScopeFailureUsesIssue173DeterministicAsset(t *testing.T) {
	root := repositoryRoot(t)
	scenarioPath := filepath.Join(root, "evaluations", "scenarios", "implement-issue", "implement-issue-negative.yaml")
	sc, err := LoadScenario(scenarioPath)
	if err != nil {
		t.Fatal(err)
	}

	brokenSandbox := t.TempDir()
	if err := stageFixture(root, sc.Fixture, brokenSandbox); err != nil {
		t.Fatal(err)
	}
	prepareIssue173SuccessFixture(t, brokenSandbox)
	before := snapshotHashes(brokenSandbox, sc.Expectations.UnchangedFiles)
	if err := os.WriteFile(filepath.Join(brokenSandbox, "docs", "roadmap.md"), []byte("unauthorized change\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	broken := evaluateAssertions(context.Background(), sc, "outside the plan; handing the branch to create-pr", brokenSandbox, before, map[string]bool{"create-pr": true})
	if len(broken) == 0 || !strings.Contains(strings.Join(broken, "\n"), "out-of-scope file docs/roadmap.md was modified") {
		t.Fatalf("expected the promoted scope failure to be detected, got %v", broken)
	}

	cleanSandbox := t.TempDir()
	if err := stageFixture(root, sc.Fixture, cleanSandbox); err != nil {
		t.Fatal(err)
	}
	prepareIssue173SuccessFixture(t, cleanSandbox)
	cleanBefore := snapshotHashes(cleanSandbox, sc.Expectations.UnchangedFiles)
	clean := evaluateAssertions(context.Background(), sc, "outside the plan; handing the branch to create-pr", cleanSandbox, cleanBefore, map[string]bool{"create-pr": true})
	if len(clean) != 0 {
		t.Fatalf("expected the corrected scenario to pass, got %v", clean)
	}
}

func prepareIssue173SuccessFixture(t *testing.T, sandbox string) {
	t.Helper()
	path := filepath.Join(sandbox, "src", "currency.go")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString("\nfunc applyTax(total, rate float64) float64 { return total * (1 + rate) }\n"); err != nil {
		t.Fatal(err)
	}
}

func TestCompactionPairPreservesDeterministicContract(t *testing.T) {
	root := repositoryRoot(t)
	fullPath := filepath.Join(root, "evaluations", "fixtures", "failure-promotion", "compaction", "full", "instructions.md")
	compactPath := filepath.Join(root, "evaluations", "fixtures", "failure-promotion", "compaction", "compact", "instructions.md")
	full, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatal(err)
	}
	compact, err := os.ReadFile(compactPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(compact) >= len(full) {
		t.Fatalf("compact fixture is not smaller: full=%d compact=%d", len(full), len(compact))
	}

	sc := &Scenario{Expectations: Expectations{TranscriptMust: []string{"fixture unchanged", "outcome", "evidence", "owner"}}}
	for name, transcript := range map[string]string{"full": string(full), "compact": string(compact)} {
		if failures := evaluateAssertions(context.Background(), sc, transcript, t.TempDir(), nil, nil); len(failures) != 0 {
			t.Fatalf("%s instruction lost deterministic contract: %v", name, failures)
		}
	}
}
