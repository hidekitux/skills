package instructions

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fixedCounter int

func (f fixedCounter) Count(text string) (int, error) { return int(f) + len(text), nil }

func TestHeadingNamesIgnoresFencedCode(t *testing.T) {
	content := "# Title\n\n## Visible\n```md\n## Hidden\n```\n### Detail\n"
	want := []string{"Visible", "Detail"}
	got := HeadingNames(content)
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("headings = %v, want %v", got, want)
	}
}

func TestLoadRejectsConditionalSectionWithoutReference(t *testing.T) {
	path := filepath.Join(t.TempDir(), "inventory.yml")
	content := "version: 1\ntokenizer:\n  encoding: cl100k_base\n  module: github.com/pkoukk/tiktoken-go\n  version: v0.1.6\nskills:\n  - name: demo\n    path: skills/demo/SKILL.md\n    sections:\n      - heading: Workflow\n        classification: conditional-reference\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected missing conditional reference to fail")
	}
}

func TestValidateCoverageAndMeasure(t *testing.T) {
	root := t.TempDir()
	skillPath := filepath.Join(root, "skills", "demo", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(skillPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(skillPath, []byte("# Demo\n\n## Workflow\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	inventory := Inventory{Version: 1}
	inventory.Tokenizer.Encoding = EncodingName
	inventory.Tokenizer.Module = TokenizerModule
	inventory.Tokenizer.Version = TokenizerVersion
	inventory.Skills = []Skill{{
		Name:     "demo",
		Path:     "skills/demo/SKILL.md",
		Sections: []Section{{Heading: "Workflow", Classification: TaskProcedure}},
	}}
	if err := ValidateCoverage(root, inventory); err != nil {
		t.Fatal(err)
	}
	counts, err := Measure(root, inventory, fixedCounter(7))
	if err != nil {
		t.Fatal(err)
	}
	if counts["demo"] != len("# Demo\n\n## Workflow\n")+7 {
		t.Fatalf("count = %d, want %d", counts["demo"], len("# Demo\n\n## Workflow\n")+7)
	}
}
