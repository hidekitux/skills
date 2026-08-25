package check

import (
	"os"
	"path/filepath"
	"testing"
)

// writeNestedFile writes content to path, creating parent directories.
func writeNestedFile(path string, content []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, content, 0o644)
}

func writeSkillFile(t *testing.T, root, skill, body string) {
	t.Helper()
	if err := writeNestedFile(filepath.Join(root, "skills", skill, "SKILL.md"),
		[]byte("---\nname: "+skill+"\n---\n"+body)); err != nil {
		t.Fatal(err)
	}
}

func TestGuidedPathsAcceptsResolvableExecutableAndTemplate(t *testing.T) {
	root := t.TempDir()
	if err := writeNestedFile(filepath.Join(root, "cmd", "valid-tool", "main.go"), []byte("package main\n")); err != nil {
		t.Fatal(err)
	}
	if err := writeNestedFile(filepath.Join(root, "skills", "demo", "templates", "github", "ok.yml"), []byte("name: ok\n")); err != nil {
		t.Fatal(err)
	}
	writeSkillFile(t, root, "demo", "Run `go run ./cmd/valid-tool` and install `templates/github/ok.yml`.\n")
	if code := runCheck(t, CheckGuidedPaths, root); code != 0 {
		t.Fatalf("expected pass, got exit %d", code)
	}
}

func TestGuidedPathsRejectsMissingExecutablePath(t *testing.T) {
	root := t.TempDir()
	// No cmd/removed-tool exists; a skill still references it as current.
	writeSkillFile(t, root, "demo", "Run `go run ./cmd/removed-tool` before the API call.\n")
	if code := runCheck(t, CheckGuidedPaths, root); code != 1 {
		t.Fatalf("expected failure for a missing executable path, got exit %d", code)
	}
}

func TestGuidedPathsRejectsMissingTemplatePath(t *testing.T) {
	root := t.TempDir()
	// No templates/... exists; a skill still references a template as current.
	writeSkillFile(t, root, "demo", "Install `templates/github/removed-template.yml`.\n")
	if code := runCheck(t, CheckGuidedPaths, root); code != 1 {
		t.Fatalf("expected failure for a missing template path, got exit %d", code)
	}
}

func TestGuidedPathsAllowsGeneratedDestinationReference(t *testing.T) {
	root := t.TempDir()
	// The generated-destination validator path is allowlisted so a bootstrap
	// skill can document the generated project's layout without a source file.
	writeSkillFile(t, root, "demo", "Copy the validator to `scripts/lint/validate-commit-message.py`.\n")
	if code := runCheck(t, CheckGuidedPaths, root); code != 0 {
		t.Fatalf("expected pass for an allowlisted destination, got exit %d", code)
	}
}
