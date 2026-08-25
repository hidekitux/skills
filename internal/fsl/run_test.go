package fsl

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSpecFilesCollectsRepoAndSkillSpecs(t *testing.T) {
	root := t.TempDir()
	write(t, root, "specs/root-a.fsl", "x")
	write(t, root, "specs/nested/b.fsl", "x")
	write(t, root, "skills/some-skill/specs/skill.fsl", "x")
	write(t, root, "skills/some-skill/README.md", "x")

	specs, err := specFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"specs/nested/b.fsl", "specs/root-a.fsl", "skills/some-skill/specs/skill.fsl"}
	if !reflect.DeepEqual(specs, expected) {
		t.Fatalf("unexpected specs %v", specs)
	}
}

func TestSpecFilesDedupesSymlinkedSkillSpecs(t *testing.T) {
	root := t.TempDir()
	// A skill-owned source exposed through the required repository-level symlink:
	// both paths resolve to the same physical file, so it must be collected once,
	// using the first-seen exposure path as the stable display path.
	write(t, root, "skills/some-skill/specs/skill.fsl", "x")
	writeLink(t, root, "specs/some-skill/skill.fsl", "../../skills/some-skill/specs/skill.fsl")
	write(t, root, "specs/repo-owned.fsl", "x")

	specs, err := specFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"specs/repo-owned.fsl", "specs/some-skill/skill.fsl"}
	if !reflect.DeepEqual(specs, expected) {
		t.Fatalf("unexpected specs %v", specs)
	}
}

func TestSpecFilesCollapsesDuplicateAliases(t *testing.T) {
	root := t.TempDir()
	// Two exposure symlinks aliasing the same physical source must collapse to one.
	write(t, root, "skills/some-skill/specs/skill.fsl", "x")
	writeLink(t, root, "specs/a/skill.fsl", "../../skills/some-skill/specs/skill.fsl")
	writeLink(t, root, "specs/b/skill.fsl", "../../skills/some-skill/specs/skill.fsl")

	specs, err := specFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"specs/a/skill.fsl"}
	if !reflect.DeepEqual(specs, expected) {
		t.Fatalf("unexpected specs %v", specs)
	}
}

func TestSpecFilesOrderingIsDeterministic(t *testing.T) {
	root := t.TempDir()
	write(t, root, "skills/z-skill/specs/z.fsl", "x")
	write(t, root, "specs/a.fsl", "x")
	write(t, root, "specs/m.fsl", "x")
	writeLink(t, root, "specs/z-skill/z.fsl", "../../skills/z-skill/specs/z.fsl")

	want, err := specFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		got, err := specFiles(root)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unstable ordering: got %v, want %v", got, want)
		}
	}
}

func TestSpecFilesRejectsBrokenSymlink(t *testing.T) {
	root := t.TempDir()
	writeLink(t, root, "specs/broken.fsl", "missing-target.fsl")

	if _, err := specFiles(root); err == nil {
		t.Fatal("expected error for a broken symlink")
	}
}

func TestSpecFilesRejectsSymlinkEscapingRepository(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.fsl")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	writeLink(t, root, "specs/escape.fsl", outside)

	if _, err := specFiles(root); err == nil {
		t.Fatal("expected error for a symlink resolving outside the repository")
	}
}

func TestVerifyFSLRejectsBrokenSymlinkBeforeRunningFslc(t *testing.T) {
	root := t.TempDir()
	writeLink(t, root, "specs/broken.fsl", "missing-target.fsl")
	var out, errOut bytes.Buffer
	if code := VerifyFSL(root, &out, &errOut); code == 0 {
		t.Fatalf("expected nonzero, got 0: out=%q err=%q", out.String(), errOut.String())
	}
	if !strings.Contains(errOut.String(), "broken.fsl") {
		t.Fatalf("expected error to name the broken spec, got err=%q", errOut.String())
	}
}

func TestRunFslcInvokesBinaryAtBinDir(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fslc")
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nprintf 'fake-fslc-ran\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The binary must be resolved from FSLC_BIN_DIR directly, not via the
	// inherited PATH, so verify-fsl works on CI runners with no fslc on PATH.
	t.Setenv("FSLC_BIN_DIR", dir)
	var out, errOut bytes.Buffer
	if code := runFslc(&out, &errOut, "check", "spec.fsl"); code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "fake-fslc-ran") {
		t.Fatalf("expected the binary at FSLC_BIN_DIR to run, got out=%q err=%q", out.String(), errOut.String())
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

func writeLink(t *testing.T, root, name, target string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.FromSlash(target), full); err != nil {
		t.Fatal(err)
	}
}
