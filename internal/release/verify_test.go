package release

import (
	"os"
	"path/filepath"
	"testing"
)

func writeSkill(t *testing.T, root, name, body string) {
	t.Helper()
	dir := filepath.Join(root, "skills", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\n---\n"+body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestRejectsReleasedSkillReferencingSkillOutsideCatalog(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "plan-issue", "Plan and hand off to `implement-issue`.\n")
	writeSkill(t, root, "implement-issue", "Implement the plan.\n")
	errors := FindCrossSkillReferences(root, map[string]bool{"plan-issue": true})
	found := false
	for _, error := range errors {
		if contains(error, "implement-issue") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected cross-reference error, got %v", errors)
	}
}

func TestAcceptsReleasedSkillReferencingCatalogSkill(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "plan-issue", "Hand off to `implement-issue`.\n")
	writeSkill(t, root, "implement-issue", "Implements.\n")
	errors := FindCrossSkillReferences(root, map[string]bool{"plan-issue": true, "implement-issue": true})
	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %v", errors)
	}
}

func TestIgnoresSelfReference(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "analyze-baseline", "See also the `analyze-baseline` notes.\n")
	errors := FindCrossSkillReferences(root, map[string]bool{"analyze-baseline": true})
	if len(errors) != 0 {
		t.Fatalf("expected no errors, got %v", errors)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
