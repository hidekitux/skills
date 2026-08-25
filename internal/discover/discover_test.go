package discover

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func writeSkill(t *testing.T, root, dir, name string) {
	t.Helper()
	full := filepath.Join(root, "skills", filepath.FromSlash(dir), "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte("---\nname: "+name+"\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAllDiscoversFlatAndNestedSkills(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "plan-issue", "plan-issue")
	writeSkill(t, root, "skills/refactor-code", "refactor-code")

	skills := All(root)
	if len(skills) != 2 {
		t.Fatalf("expected 2 skills, got %d", len(skills))
	}
	got := []string{}
	for _, skill := range skills {
		got = append(got, skill.Dir)
	}
	want := []string{"skills/plan-issue", "skills/skills/refactor-code"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestByNameGroupsDuplicateBareNamesByCanonicalPath(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "a/dup", "dup")
	writeSkill(t, root, "b/dup", "dup")

	matches := ByName(root)["dup"]
	if len(matches) != 2 {
		t.Fatalf("expected 2 matches for duplicate name, got %d", len(matches))
	}
	if matches[0].Dir != "skills/a/dup" || matches[1].Dir != "skills/b/dup" {
		t.Fatalf("unexpected canonical paths: %v", matches)
	}
}

func TestNamesReturnsSortedUniqueNames(t *testing.T) {
	root := t.TempDir()
	writeSkill(t, root, "a/dup", "dup")
	writeSkill(t, root, "b/dup", "dup")
	writeSkill(t, root, "plan-issue", "plan-issue")

	want := []string{"dup", "plan-issue"}
	if got := Names(root); !reflect.DeepEqual(got, want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
}
