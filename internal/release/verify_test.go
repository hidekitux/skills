package release

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeGitResult struct {
	output string
	err    error
}

type fakeGitCall struct {
	root string
	args []string
}

type fakeCommandCall struct {
	name string
	args []string
}

type fakeReleaseRunner struct {
	gitResults    map[string]fakeGitResult
	execResults   map[string]int
	streamResults map[string]int
	gitCalls      []fakeGitCall
	execCalls     []fakeGitCall
	streamCalls   []fakeCommandCall
}

func newFakeReleaseRunner() *fakeReleaseRunner {
	runner := &fakeReleaseRunner{
		gitResults:    map[string]fakeGitResult{},
		execResults:   map[string]int{},
		streamResults: map[string]int{},
	}
	runner.gitResults[commandKey("git", "diff", "--quiet")] = fakeGitResult{}
	runner.gitResults[commandKey("git", "diff", "--cached", "--quiet")] = fakeGitResult{}
	runner.gitResults[commandKey("git", "ls-files", "--others", "--exclude-standard")] = fakeGitResult{}
	runner.gitResults[commandKey("git", "rev-parse", "--verify", "--quiet", "refs/tags/v1.2.3")] = fakeGitResult{err: errors.New("tag not found")}
	runner.gitResults[commandKey("git", "remote", "get-url", "origin")] = fakeGitResult{output: "origin"}
	runner.execResults[commandKey("git", "ls-remote", "--exit-code", "--refs", "origin", "refs/tags/v1.2.3")] = 2
	return runner
}

func commandKey(name string, args ...string) string {
	values := append([]string{name}, args...)
	return strings.Join(values, "\x00")
}

func (runner *fakeReleaseRunner) gitOutput(root string, args ...string) (string, error) {
	runner.gitCalls = append(runner.gitCalls, fakeGitCall{root: root, args: append([]string{}, args...)})
	result, ok := runner.gitResults[commandKey("git", args...)]
	if !ok {
		return "", errors.New("unexpected git command: " + commandKey("git", args...))
	}
	return result.output, result.err
}

func (runner *fakeReleaseRunner) execIn(root, name string, args ...string) int {
	runner.execCalls = append(runner.execCalls, fakeGitCall{root: root, args: append([]string{name}, args...)})
	if code, ok := runner.execResults[commandKey(name, args...)]; ok {
		return code
	}
	return 99
}

func (runner *fakeReleaseRunner) stream(name string, _, _ io.Writer, args ...string) int {
	runner.streamCalls = append(runner.streamCalls, fakeCommandCall{name: name, args: append([]string{}, args...)})
	if code, ok := runner.streamResults[commandKey(name, args...)]; ok {
		return code
	}
	return 99
}

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

func writeReleaseFixture(t *testing.T, catalogVersion string) string {
	t.Helper()
	root := t.TempDir()
	writeSkill(t, root, "demo", "Release test fixture.\n")
	catalog := "skills:\n  - name: demo\n    version: " + catalogVersion + "\n"
	if err := os.WriteFile(filepath.Join(root, "CATALOG.yml"), []byte(catalog), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func writeExecutable(t *testing.T, dir, name, body string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestVerifyReleaseUsesCommandAdapter(t *testing.T) {
	root := writeReleaseFixture(t, "1.2.3")
	bin := t.TempDir()
	gitLog := filepath.Join(t.TempDir(), "git.log")
	writeExecutable(t, bin, "git", `#!/bin/sh
if [ -n "$GIT_TEST_SENTINEL" ]; then exit 97; fi
printf '%s|%s\n' "$(pwd -P)" "$*" >> "$RELEASE_GIT_LOG"
case "$*" in
  "diff --quiet"|"diff --cached --quiet"|"ls-files --others --exclude-standard") exit 0 ;;
  "rev-parse --verify --quiet refs/tags/v1.2.3") exit 1 ;;
  "remote get-url origin") printf 'origin\n'; exit 0 ;;
  "ls-remote --exit-code --refs origin refs/tags/v1.2.3") exit 2 ;;
  *) exit 98 ;;
esac
`)
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("RELEASE_GIT_LOG", gitLog)
	t.Setenv("GIT_TEST_SENTINEL", "must-not-reach-child")

	var out, errOut bytes.Buffer
	if code := VerifyRelease("v1.2.3", root, &out, &errOut); code != 0 {
		t.Fatalf("expected command adapter to pass, got %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Release contract is valid for v1.2.3") {
		t.Fatalf("expected success output, got %q", out.String())
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(gitLog)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"diff --quiet",
		"diff --cached --quiet",
		"ls-files --others --exclude-standard",
		"rev-parse --verify --quiet refs/tags/v1.2.3",
		"remote get-url origin",
		"ls-remote --exit-code --refs origin refs/tags/v1.2.3",
	}
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	if len(lines) != len(want) {
		t.Fatalf("Git call log = %q, want %d calls", content, len(want))
	}
	for index, line := range lines {
		expected := resolvedRoot + "|" + want[index]
		if line != expected {
			t.Errorf("Git call %d = %q, want %q", index+1, line, expected)
		}
	}
}

func TestVerifyReleaseAcceptsCleanCatalogConsistentTag(t *testing.T) {
	root := writeReleaseFixture(t, "1.2.3")
	runner := newFakeReleaseRunner()
	var out, errOut bytes.Buffer

	if code := verifyRelease("v1.2.3", root, &out, &errOut, runner); code != 0 {
		t.Fatalf("expected clean release to pass, got %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Release contract is valid for v1.2.3") {
		t.Fatalf("expected success output, got %q", out.String())
	}
	for _, call := range runner.gitCalls {
		if call.root != root {
			t.Fatalf("Git root = %q, want %q", call.root, root)
		}
	}
	if len(runner.execCalls) != 1 || runner.execCalls[0].root != root {
		t.Fatalf("unexpected remote calls: %+v", runner.execCalls)
	}
}

func TestVerifyReleaseRejectsDirtyWorkingTree(t *testing.T) {
	root := writeReleaseFixture(t, "1.2.3")
	runner := newFakeReleaseRunner()
	runner.gitResults[commandKey("git", "diff", "--quiet")] = fakeGitResult{err: errors.New("unstaged changes")}
	var out, errOut bytes.Buffer

	if code := verifyRelease("v1.2.3", root, &out, &errOut, runner); code != 1 {
		t.Fatalf("expected dirty tree to fail, got %d", code)
	}
	if !strings.Contains(errOut.String(), "working tree has unstaged changes") {
		t.Fatalf("expected dirty-tree diagnostic, got %q", errOut.String())
	}
}

func TestVerifyReleaseRejectsCatalogTagMismatch(t *testing.T) {
	root := writeReleaseFixture(t, "1.2.2")
	runner := newFakeReleaseRunner()
	var out, errOut bytes.Buffer

	if code := verifyRelease("v1.2.3", root, &out, &errOut, runner); code != 1 {
		t.Fatalf("expected catalog mismatch to fail, got %d", code)
	}
	if !strings.Contains(errOut.String(), `version "1.2.2" does not match "1.2.3"`) {
		t.Fatalf("expected version diagnostic, got %q", errOut.String())
	}
}

func TestVerifyReleaseRejectsStableCatalogWithoutPromotionEvidence(t *testing.T) {
	root := writeReleaseFixture(t, "1.2.3")
	catalogPath := filepath.Join(root, "CATALOG.yml")
	catalog, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	catalog = []byte(strings.Replace(string(catalog), "version: 1.2.3", "status: stable\n    version: 1.2.3", 1))
	if err := os.WriteFile(catalogPath, catalog, 0o644); err != nil {
		t.Fatal(err)
	}
	runner := newFakeReleaseRunner()
	var out, errOut bytes.Buffer
	if code := verifyRelease("v1.2.3", root, &out, &errOut, runner); code != 1 {
		t.Fatalf("expected stable promotion evidence to block release, got %d", code)
	}
	if !strings.Contains(errOut.String(), "stable promotion requires") {
		t.Fatalf("expected promotion finding, got %q", errOut.String())
	}
}

func TestVerifyReleaseRejectsExistingLocalTag(t *testing.T) {
	root := writeReleaseFixture(t, "1.2.3")
	runner := newFakeReleaseRunner()
	runner.gitResults[commandKey("git", "rev-parse", "--verify", "--quiet", "refs/tags/v1.2.3")] = fakeGitResult{}
	var out, errOut bytes.Buffer

	if code := verifyRelease("v1.2.3", root, &out, &errOut, runner); code != 1 {
		t.Fatalf("expected existing local tag to fail, got %d", code)
	}
	if !strings.Contains(errOut.String(), "tag v1.2.3 already exists locally") {
		t.Fatalf("expected local-tag diagnostic, got %q", errOut.String())
	}
}

func TestVerifyReleaseRejectsExistingOriginTag(t *testing.T) {
	root := writeReleaseFixture(t, "1.2.3")
	runner := newFakeReleaseRunner()
	runner.execResults[commandKey("git", "ls-remote", "--exit-code", "--refs", "origin", "refs/tags/v1.2.3")] = 0
	var out, errOut bytes.Buffer

	if code := verifyRelease("v1.2.3", root, &out, &errOut, runner); code != 1 {
		t.Fatalf("expected existing origin tag to fail, got %d", code)
	}
	if !strings.Contains(errOut.String(), "tag v1.2.3 already exists on the origin remote") {
		t.Fatalf("expected origin-tag diagnostic, got %q", errOut.String())
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
