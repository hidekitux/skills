package fsl

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSpecFilesCollectsRepoAndSkillSpecs(t *testing.T) {
	root := t.TempDir()
	write(t, root, "specs/root-a.fsl", "x")
	write(t, root, "specs/nested/b.fsl", "x")
	write(t, root, "skills/some-skill/specs/skill.fsl", "x")
	write(t, root, "skills/some-skill/README.md", "x")

	specs := specFiles(root)
	expected := []string{"specs/nested/b.fsl", "specs/root-a.fsl", "skills/some-skill/specs/skill.fsl"}
	if !reflect.DeepEqual(specs, expected) {
		t.Fatalf("unexpected specs %v", specs)
	}
}

func TestCacheRootDefaultsToTemp(t *testing.T) {
	t.Setenv("RUNNER_TEMP", "")
	t.Setenv("TMPDIR", "/tmp/custom")
	if got := cacheRoot(); got != "/tmp/custom/skills-fslc" {
		t.Fatalf("unexpected cache root %q", got)
	}
}

func write(t *testing.T, root, name, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
