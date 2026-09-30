package check

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
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

// guidedPathsOutput runs CheckGuidedPaths on root and returns its exit status
// and error output.
func guidedPathsOutput(t *testing.T, root string) (int, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := CheckGuidedPaths(root, &out, &errOut)
	return code, errOut.String()
}

// writeCategorySkill writes a skill at skills/<category>/<name> with body in
// SKILL.md and each extra file below the skill root.
func writeCategorySkill(t *testing.T, root, category, name, body string, extra map[string]string) {
	t.Helper()
	dir := filepath.Join(root, "skills", category, name)
	if err := writeNestedFile(filepath.Join(dir, "SKILL.md"), []byte("---\nname: "+name+"\n---\n"+body)); err != nil {
		t.Fatal(err)
	}
	for rel, content := range extra {
		if err := writeNestedFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(content)); err != nil {
			t.Fatal(err)
		}
	}
}

func TestGuidedPathsAcceptsSkillLinksInsideTheSkillRoot(t *testing.T) {
	root := t.TempDir()
	writeCategorySkill(t, root, "govern", "demo", "Read [the notes](references/notes.md#usage) and ![diagram](references/flow.png \"Flow\").\n", map[string]string{
		"references/notes.md":       "See [the skill](../SKILL.md) and [hosts](hosts/codex.md).\n",
		"references/flow.png":       "",
		"references/hosts/codex.md": "Back to [notes](../notes.md).\n",
	})
	if code, stderr := guidedPathsOutput(t, root); code != 0 {
		t.Fatalf("expected pass, got exit %d: %s", code, stderr)
	}
}

func TestGuidedPathsRejectsSkillLinkToMissingFile(t *testing.T) {
	root := t.TempDir()
	writeCategorySkill(t, root, "govern", "demo", "Read [the notes](references/missing.md).\n", nil)
	code, stderr := guidedPathsOutput(t, root)
	if code != 1 || !strings.Contains(stderr, `skills/govern/demo/SKILL.md: link "references/missing.md" does not resolve to a file`) {
		t.Fatalf("expected a missing-link failure, got exit %d: %s", code, stderr)
	}
}

func TestGuidedPathsRejectsSkillLinkOutsideTheSkillRoot(t *testing.T) {
	root := t.TempDir()
	// The target exists in the repository, as docs/model-selection.md did
	// before #407, but an installed copy holds only the skill directory.
	if err := writeNestedFile(filepath.Join(root, "docs", "model-selection.md"), []byte("# Model selection\n")); err != nil {
		t.Fatal(err)
	}
	writeCategorySkill(t, root, "govern", "demo", "", map[string]string{
		"references/hosts/claude-code.md": "See [docs/model-selection.md](../../../../../docs/model-selection.md).\n",
	})
	code, stderr := guidedPathsOutput(t, root)
	want := `skills/govern/demo/references/hosts/claude-code.md: link "../../../../../docs/model-selection.md" points outside its skill root skills/govern/demo`
	if code != 1 || !strings.Contains(stderr, want) {
		t.Fatalf("expected an outside-root failure, got exit %d: %s", code, stderr)
	}
}

func TestGuidedPathsRejectsTheBrokenLinkFixedInIssue407(t *testing.T) {
	root := t.TempDir()
	writeCategorySkill(t, root, "govern", "demo", "", map[string]string{
		"references/hosts/codex.md": "(see\n[docs/model-selection.md](../../../../docs/model-selection.md)); the selector\n",
	})
	if code, stderr := guidedPathsOutput(t, root); code != 1 || !strings.Contains(stderr, `link "../../../../docs/model-selection.md"`) {
		t.Fatalf("expected the pre-#407 link to fail, got exit %d: %s", code, stderr)
	}
}

func TestGuidedPathsIgnoresCodeURLsAndAnchorsInSkillLinks(t *testing.T) {
	root := t.TempDir()
	body := "Use `[inline](missing-inline.md)` literally.\n" +
		"```markdown\n[fenced](missing-fenced.md)\n```\n" +
		"~~~\n[tilde](missing-tilde.md)\n~~~\n" +
		"Visit [the site](https://example.com/missing.md), [mail](mailto:maintainers), " +
		"[this section](#usage), and [root](/absolute/missing.md).\n"
	writeCategorySkill(t, root, "govern", "demo", body, nil)
	if code, stderr := guidedPathsOutput(t, root); code != 0 {
		t.Fatalf("expected pass for ignored link forms, got exit %d: %s", code, stderr)
	}
}
