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

	specs := specFiles(root)
	expected := []string{"specs/nested/b.fsl", "specs/root-a.fsl", "skills/some-skill/specs/skill.fsl"}
	if !reflect.DeepEqual(specs, expected) {
		t.Fatalf("unexpected specs %v", specs)
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
