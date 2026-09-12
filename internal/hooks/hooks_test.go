// Package hooks holds contract tests for the retained repository Git hooks
// and local-setup shell wrappers, replacing the former tests/test_local_setup.py
// coverage. The hooks under .githooks/ and the setup scripts under
// scripts/setup/ remain shell by design; these tests lock their behavior.
package hooks

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot locates the repository root (the directory containing go.mod) from
// this package's file location.
func repoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func readRepoFile(t *testing.T, rel string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(repoRoot(), rel))
	if err != nil {
		t.Fatalf("cannot read %s: %v", rel, err)
	}
	return string(content)
}

func TestCommitMsgHookUsesPrebuiltValidator(t *testing.T) {
	hook := readRepoFile(t, ".githooks/commit-msg")
	if !strings.Contains(hook, ".mise/bin/validate-commit-message") {
		t.Fatal("commit-msg must use the prebuilt validate-commit-message binary")
	}
	if !strings.Contains(hook, "commitlint") {
		t.Fatal("commit-msg must still invoke commitlint")
	}
	if strings.Contains(hook, "python") {
		t.Fatal("commit-msg must not depend on a Python runtime")
	}
	if strings.Contains(hook, "go run") || strings.Contains(hook, "go build") {
		t.Fatal("commit-msg must not compile code or fetch modules")
	}
}

func TestPreCommitRunsLocalChecks(t *testing.T) {
	hook := readRepoFile(t, ".githooks/pre-commit")
	if !strings.Contains(hook, "mise run check:local") {
		t.Fatalf("pre-commit must run check:local: %q", hook)
	}
}

func TestPrePushRunsFullValidation(t *testing.T) {
	hook := readRepoFile(t, ".githooks/pre-push")
	if !strings.Contains(hook, "mise run validate:all") {
		t.Fatalf("pre-push must run validate: %q", hook)
	}
}

func TestPostCheckoutOnlyRefreshesSetupOnBranchCheckouts(t *testing.T) {
	hook := readRepoFile(t, ".githooks/post-checkout")
	if !strings.Contains(hook, `[ "$3" = "1" ]`) {
		t.Fatalf("post-checkout must guard on the branch-checkout flag: %q", hook)
	}
	if !strings.Contains(hook, "mise run setup:all") {
		t.Fatalf("post-checkout must run setup: %q", hook)
	}
}

func TestSetupCommitlintBuildsAndLinksMessageValidator(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/setup-commitlint.sh")
	if !strings.Contains(script, "./cmd/validate-commit-message") {
		t.Fatal("setup-commitlint must build the repository-local message validator")
	}
	if !strings.Contains(script, "validate-commit-message") && !strings.Contains(script, "commitlint") {
		t.Fatal("setup-commitlint must wire the validator beside commitlint")
	}
}

func TestRegisterLocalSkillsIsRevisionKeyed(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/register-local-skills.sh")
	if !strings.Contains(script, "worktree-snapshot") {
		t.Fatal("register-local-skills must stay keyed by the worktree snapshot")
	}
	for _, hostRoot := range []string{".agents/skills", ".claude/skills"} {
		if !strings.Contains(script, hostRoot) {
			t.Fatalf("register-local-skills must register to %s", hostRoot)
		}
	}
}

func TestRegisterLocalSkillsDiscoversNestedSkillsRecursively(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/register-local-skills.sh")
	if !strings.Contains(script, "find \"${source_root}\" -type f -name SKILL.md") {
		t.Fatal("register-local-skills must discover SKILL.md files recursively under skills/")
	}
	if !strings.Contains(script, `target="../../skills/${skill_rel}"`) {
		t.Fatal("register-local-skills must build the symlink target from the canonical repository-relative skill path")
	}
}

func TestSetupLocalSkillsEnablesHooks(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/setup-local-skills.sh")
	if !strings.Contains(script, "core.hooksPath") {
		t.Fatalf("setup-local-skills must enable .githooks via core.hooksPath: %q", script)
	}
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runTestCommand(t *testing.T, dir, name string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+filepath.Join(t.TempDir(), "gitconfig"),
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		return stdout.String(), stderr.String(), exitErr.ExitCode()
	}
	t.Fatalf("run %s %v: %v", name, args, err)
	return "", "", -1
}

func newRegistrationRepository(t *testing.T, skills map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if _, _, code := runTestCommand(t, root, "git", "init", "--quiet"); code != 0 {
		t.Fatalf("git init failed in %s", root)
	}
	if _, _, code := runTestCommand(t, root, "git", "config", "user.name", "Issue 314 test"); code != 0 {
		t.Fatal("git config user.name failed")
	}
	if _, _, code := runTestCommand(t, root, "git", "config", "user.email", "issue-314-test-email"); code != 0 {
		t.Fatal("git config user.email failed")
	}
	for skillPath, body := range skills {
		writeTestFile(t, filepath.Join(root, "skills", skillPath, "SKILL.md"), body)
	}
	if _, _, code := runTestCommand(t, root, "git", "add", "skills"); code != 0 {
		t.Fatal("git add failed")
	}
	if _, _, code := runTestCommand(t, root, "git", "commit", "--quiet", "-m", "test: create registration fixture"); code != 0 {
		t.Fatal("git commit failed")
	}
	return root
}

func runRegistration(t *testing.T, root string) (string, string, int) {
	t.Helper()
	return runTestCommand(t, root, "bash", filepath.Join(repoRoot(), "scripts/setup/register-local-skills.sh"))
}

func linkTarget(t *testing.T, path string) string {
	t.Helper()
	target, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("readlink %s: %v", path, err)
	}
	return target
}

func writeTestLink(t *testing.T, path, target string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
}

func assertRegistrationNames(t *testing.T, root, host string, want map[string]string) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(root, host, "skills"))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, entry := range entries {
		path := filepath.Join(root, host, "skills", entry.Name())
		if entry.Type()&os.ModeSymlink != 0 {
			got[entry.Name()] = linkTarget(t, path)
		}
	}
	if len(got) != len(want) {
		t.Fatalf("%s registration names = %#v, want %#v", host, got, want)
	}
	for name, target := range want {
		if got[name] != target {
			t.Fatalf("%s registration %s = %q, want %q", host, name, got[name], target)
		}
	}
}

func TestRegisterLocalSkillsReconcilesMigrationAndIsIdempotent(t *testing.T) {
	root := newRegistrationRepository(t, map[string]string{
		"process/current":   "---\nname: current\n---\n",
		"fix/refactor-code": "---\nname: refactor-code\n---\n",
	})
	for _, host := range []string{".agents", ".claude"} {
		writeTestLink(t, filepath.Join(root, host, "skills", "current"), "../../skills/process/current")
		writeTestLink(t, filepath.Join(root, host, "skills", "refactor-code"), "../../skills/refactor-code")
		writeTestLink(t, filepath.Join(root, host, "skills", "removed-skill"), "../../skills/removed-skill")
	}

	stdout, stderr, code := runRegistration(t, root)
	if code != 0 {
		t.Fatalf("migration registration failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	want := map[string]string{
		"current":       "../../skills/process/current",
		"refactor-code": "../../skills/fix/refactor-code",
	}
	assertRegistrationNames(t, root, ".agents", want)
	assertRegistrationNames(t, root, ".claude", want)
	if _, err := os.Lstat(filepath.Join(root, ".agents", "skills", "removed-skill")); !os.IsNotExist(err) {
		t.Fatalf("stale Codex registration still exists: %v", err)
	}
	if _, err := os.Lstat(filepath.Join(root, ".claude", "skills", "removed-skill")); !os.IsNotExist(err) {
		t.Fatalf("stale Claude Code registration still exists: %v", err)
	}
	revision, _, revisionCode := runTestCommand(t, root, "git", "rev-parse", "HEAD")
	if revisionCode != 0 {
		t.Fatal("git rev-parse failed")
	}
	stamp, err := os.ReadFile(filepath.Join(root, ".agents", "worktree-snapshot"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(stamp)) != strings.TrimSpace(revision) {
		t.Fatalf("snapshot = %q, want %q", strings.TrimSpace(string(stamp)), strings.TrimSpace(revision))
	}

	stdout, stderr, code = runRegistration(t, root)
	if code != 0 || stderr != "" || !strings.Contains(stdout, "Local skill registration is current") {
		t.Fatalf("repeated registration was not idempotent: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}

func TestRegisterLocalSkillsPreservesExternalLinkAndReportsConflict(t *testing.T) {
	root := newRegistrationRepository(t, map[string]string{
		"process/current": "---\nname: current\n---\n",
	})
	external := filepath.Join(root, "outside")
	writeTestFile(t, external, "external target")
	path := filepath.Join(root, ".agents", "skills", "external")
	writeTestLink(t, path, "../../skills/../outside")

	_, stderr, code := runRegistration(t, root)
	if code != 1 {
		t.Fatalf("external link conflict returned code %d, want 1", code)
	}
	if got := linkTarget(t, path); got != "../../skills/../outside" {
		t.Fatalf("external link target changed to %q", got)
	}
	if !strings.Contains(stderr, "Preserving non-owned symbolic link") || !strings.Contains(stderr, "remove or rename it") {
		t.Fatalf("external link diagnostic is not actionable: %q", stderr)
	}
	if _, err := os.Stat(filepath.Join(root, ".claude", "skills", "current")); err != nil {
		t.Fatalf("other host did not receive the current registration: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".agents", "worktree-snapshot")); !os.IsNotExist(err) {
		t.Fatalf("snapshot was written after a conflict: %v", err)
	}
}

func TestRegisterLocalSkillsPreservesRegularFileAndReportsConflict(t *testing.T) {
	root := newRegistrationRepository(t, map[string]string{
		"process/current": "---\nname: current\n---\n",
	})
	path := filepath.Join(root, ".claude", "skills", "current")
	writeTestFile(t, path, "user-owned entry")

	_, stderr, code := runRegistration(t, root)
	if code != 1 {
		t.Fatalf("regular-file conflict returned code %d, want 1", code)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "user-owned entry" {
		t.Fatalf("regular-file conflict changed content to %q", content)
	}
	if !strings.Contains(stderr, "Preserving existing non-symbolic-link entry") || !strings.Contains(stderr, "remove or rename it") {
		t.Fatalf("regular-file diagnostic is not actionable: %q", stderr)
	}
}
