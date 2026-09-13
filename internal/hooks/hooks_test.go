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
	if !strings.Contains(hook, "mise run setup:refresh") {
		t.Fatalf("post-checkout must run setup refresh: %q", hook)
	}
}

func TestSetupCommitlintPreparesOnlyTheSharedCommitlint(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/setup-commitlint.sh")
	if !strings.Contains(script, "github.com/conventionalcommit/commitlint@v0.12.0") {
		t.Fatal("setup-commitlint must install the pinned shared commitlint")
	}
	if strings.Contains(script, "validate-commit-message") {
		t.Fatal("setup-commitlint must leave revision-dependent validator preparation to refresh")
	}
}

func TestSetupRefreshOwnsTheRevisionState(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/setup-refresh.sh")
	if !strings.Contains(script, "setup-state") || !strings.Contains(script, "bootstrap_inputs") || !strings.Contains(script, "validator_inputs") {
		t.Fatal("setup-refresh must compare the setup state and both input fingerprints")
	}
	if !strings.Contains(script, "write_state \"ready\"") {
		t.Fatal("setup-refresh must write ready state only after its stages succeed")
	}
}

func TestSetupBootstrapUsesTheSharedState(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/setup-bootstrap.sh")
	if !strings.Contains(script, "write_state \"bootstrap-ready\"") {
		t.Fatal("setup-bootstrap must record an intermediate non-ready state")
	}
	if !strings.Contains(script, "core.hooksPath") {
		t.Fatal("setup-bootstrap must prepare the repository Git hook configuration")
	}
}

func TestSetupAllRunsBootstrapBeforeRefresh(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/setup-all.sh")
	bootstrap := strings.Index(script, "setup-bootstrap.sh")
	refresh := strings.Index(script, "setup-refresh.sh")
	if bootstrap < 0 || refresh < 0 || bootstrap > refresh {
		t.Fatalf("setup-all must run bootstrap before refresh: %q", script)
	}
}

func TestSetupStateProvidesAtomicStateHelpers(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/setup-state.sh")
	for _, marker := range []string{"bootstrap_inputs", "validator_inputs", "mktemp", "mv \"${temporary}\" \"${setup_state_file}\""} {
		if !strings.Contains(script, marker) {
			t.Fatalf("setup-state must contain %q: %q", marker, script)
		}
	}
}

func TestSetupValidatorBuildsAndLinksTheRevisionValidator(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/setup-validator.sh")
	if !strings.Contains(script, "./cmd/validate-commit-message") || !strings.Contains(script, "validate-commit-message") {
		t.Fatal("setup-validator must build and link the repository-local message validator")
	}
}

func TestRegisterLocalSkillsDoesNotOwnRevisionState(t *testing.T) {
	script := readRepoFile(t, "scripts/setup/register-local-skills.sh")
	if strings.Contains(script, "worktree-snapshot") {
		t.Fatal("register-local-skills must leave revision state to setup-refresh")
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
	if !strings.Contains(script, "setup-refresh.sh") {
		t.Fatalf("setup-local-skills must route compatibility setup through refresh: %q", script)
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
	return runTestCommandWithEnv(t, dir, nil, name, args...)
}

func runTestCommandWithEnv(t *testing.T, dir string, extraEnv []string, name string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	env := make([]string, 0, len(os.Environ())+len(extraEnv)+2)
	for _, value := range os.Environ() {
		name, _, ok := strings.Cut(value, "=")
		if ok && strings.HasPrefix(name, "GIT_") {
			continue
		}
		env = append(env, value)
	}
	env = append(env, extraEnv...)
	cmd.Env = append(env,
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

func copyRepositoryFile(t *testing.T, sourceRoot, destinationRoot, rel string) {
	t.Helper()
	source := filepath.Join(sourceRoot, rel)
	destination := filepath.Join(destinationRoot, rel)
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read fixture %s: %v", rel, err)
	}
	info, err := os.Stat(source)
	if err != nil {
		t.Fatalf("stat fixture %s: %v", rel, err)
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination, content, info.Mode().Perm()); err != nil {
		t.Fatalf("write fixture %s: %v", rel, err)
	}
}

func newSetupRepository(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	sourceRoot := repoRoot()
	for _, rel := range []string{
		"mise.toml",
		"go.mod",
		"go.sum",
		"cmd/validate-commit-message/main.go",
		"internal/commitlint/message.go",
		".githooks/commit-msg",
		"scripts/setup/register-local-skills.sh",
		"scripts/setup/setup-all.sh",
		"scripts/setup/setup-bootstrap.sh",
		"scripts/setup/setup-commitlint.sh",
		"scripts/setup/setup-local-skills.sh",
		"scripts/setup/setup-refresh.sh",
		"scripts/setup/setup-state.sh",
		"scripts/setup/setup-validator.sh",
	} {
		copyRepositoryFile(t, sourceRoot, root, rel)
	}
	writeTestFile(t, filepath.Join(root, ".gitignore"), ".agents/\n.claude/\n.mise/\n")
	writeTestFile(t, filepath.Join(root, "skills", "process", "current", "SKILL.md"), "---\nname: current\n---\n")
	if _, _, code := runTestCommand(t, root, "git", "init", "--quiet"); code != 0 {
		t.Fatal("fixture git init failed")
	}
	if _, _, code := runTestCommand(t, root, "git", "config", "user.name", "Issue 316 test"); code != 0 {
		t.Fatal("fixture git config user.name failed")
	}
	if _, _, code := runTestCommand(t, root, "git", "config", "user.email", "issue-316-test-email"); code != 0 {
		t.Fatal("fixture git config user.email failed")
	}
	if _, _, code := runTestCommand(t, root, "git", "add", "."); code != 0 {
		t.Fatal("fixture git add failed")
	}
	if _, _, code := runTestCommand(t, root, "git", "commit", "--quiet", "-m", "test: create setup fixture"); code != 0 {
		t.Fatal("fixture git commit failed")
	}

	sharedBin := filepath.Join(root, ".git", ".mise", "bin")
	writeTestFile(t, filepath.Join(sharedBin, "commitlint"), "#!/bin/sh\nexit 0\n")
	if err := os.Chmod(filepath.Join(sharedBin, "commitlint"), 0o755); err != nil {
		t.Fatal(err)
	}
	fakeBin := filepath.Join(root, "fake-bin")
	fakeGo := "#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$SETUP_LOG\"\nif [ \"$SETUP_FAIL_BUILD\" = 1 ] && [ \"$1\" = build ]; then exit 42; fi\noutput=\"\"\nwhile [ \"$#\" -gt 0 ]; do\n  if [ \"$1\" = -o ]; then shift; output=\"$1\"; fi\n  shift\ndone\nif [ -n \"$output\" ]; then printf '#!/bin/sh\\nexit 0\\n' > \"$output\"; chmod 755 \"$output\"; fi\n"
	writeTestFile(t, filepath.Join(fakeBin, "go"), fakeGo)
	if err := os.Chmod(filepath.Join(fakeBin, "go"), 0o755); err != nil {
		t.Fatal(err)
	}
	return root, fakeBin
}

func runSetupRefresh(t *testing.T, root, fakeBin string, failBuild bool) (string, string, int) {
	t.Helper()
	fail := "0"
	if failBuild {
		fail = "1"
	}
	env := []string{
		"SETUP_ROOT=" + root,
		"SETUP_LOG=" + filepath.Join(root, "go-calls"),
		"SETUP_FAIL_BUILD=" + fail,
		"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH"),
	}
	return runTestCommandWithEnv(t, root, env, "bash", filepath.Join(root, "scripts/setup/setup-refresh.sh"))
}

func TestSetupRefreshBootstrapsIdempotentlyAndTracksRevision(t *testing.T) {
	root, fakeBin := newSetupRepository(t)
	stdout, stderr, code := runSetupRefresh(t, root, fakeBin, false)
	if code != 0 {
		t.Fatalf("first refresh failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	statePath := filepath.Join(root, ".agents", "setup-state")
	state, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(state), "status=ready") || !strings.Contains(string(state), "revision=") {
		t.Fatalf("first refresh did not write ready revision state: %q", state)
	}
	link := filepath.Join(root, ".agents", "skills", "current")
	linkInfo, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	calls, err := os.ReadFile(filepath.Join(root, "go-calls"))
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(strings.TrimSpace(string(calls)), "\n") + 1; lines != 1 {
		t.Fatalf("first refresh go calls = %d, want 1: %q", lines, calls)
	}

	stdout, stderr, code = runSetupRefresh(t, root, fakeBin, false)
	if code != 0 || stderr != "" || !strings.Contains(stdout, "Worktree setup is current") {
		t.Fatalf("repeated refresh was not a no-op: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	repeatedState, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	repeatedLink, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if string(repeatedState) != string(state) || !repeatedLink.ModTime().Equal(linkInfo.ModTime()) {
		t.Fatal("repeated refresh rewrote setup state or an unchanged link")
	}

	writeTestFile(t, filepath.Join(root, "skills", "process", "new", "SKILL.md"), "---\nname: new\n---\n")
	if _, _, code := runTestCommand(t, root, "git", "add", "skills/process/new/SKILL.md"); code != 0 {
		t.Fatal("revision fixture git add failed")
	}
	if _, _, code := runTestCommand(t, root, "git", "commit", "--quiet", "-m", "test: change setup revision"); code != 0 {
		t.Fatal("revision fixture git commit failed")
	}
	stdout, stderr, code = runSetupRefresh(t, root, fakeBin, false)
	if code != 0 || stderr != "" {
		t.Fatalf("revision refresh failed: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	updatedState, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(updatedState) == string(state) || !strings.Contains(stdout, "Worktree refresh complete") {
		t.Fatalf("revision refresh did not publish new ready state: stdout=%q state=%q", stdout, updatedState)
	}
	updatedCalls, err := os.ReadFile(filepath.Join(root, "go-calls"))
	if err != nil {
		t.Fatal(err)
	}
	if lines := strings.Count(strings.TrimSpace(string(updatedCalls)), "\n") + 1; lines != 1 {
		t.Fatalf("revision refresh rebuilt validator unnecessarily: %q", updatedCalls)
	}
}

func TestSetupRefreshFailureLeavesPreviousStateAndReportsStage(t *testing.T) {
	root, fakeBin := newSetupRepository(t)
	if _, stderr, code := runSetupRefresh(t, root, fakeBin, false); code != 0 || stderr != "" {
		t.Fatalf("initial refresh failed: code=%d stderr=%q", code, stderr)
	}
	statePath := filepath.Join(root, ".agents", "setup-state")
	before, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module fixture\n\ngo 1.26\n\n// changed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, stderr, code := runSetupRefresh(t, root, fakeBin, true)
	if code == 0 || !strings.Contains(stderr, "commit-message validator") {
		t.Fatalf("failed validator stage lacked actionable diagnostic: code=%d stderr=%q", code, stderr)
	}
	after, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatalf("failed refresh published false-ready state: before=%q after=%q", before, after)
	}
	if _, stderr, code = runSetupRefresh(t, root, fakeBin, false); code != 0 || stderr != "" {
		t.Fatalf("refresh did not recover after failed stage: code=%d stderr=%q", code, stderr)
	}
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
	currentLink, err := os.Lstat(filepath.Join(root, ".agents", "skills", "current"))
	if err != nil {
		t.Fatal(err)
	}
	stdout, stderr, code = runRegistration(t, root)
	if code != 0 || stderr != "" || stdout != "" {
		t.Fatalf("repeated registration was not idempotent: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	repeatedLink, err := os.Lstat(filepath.Join(root, ".agents", "skills", "current"))
	if err != nil {
		t.Fatal(err)
	}
	if !repeatedLink.ModTime().Equal(currentLink.ModTime()) {
		t.Fatal("repeated registration rewrote an unchanged link")
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
