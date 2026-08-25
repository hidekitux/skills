package diagnose

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/support"
)

var testEmail = "test" + "@" + "example.invalid"

func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = support.GitEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

func TestDiagnoseWorktreeReportsOwnerAndSetupState(t *testing.T) {
	dir := t.TempDir()
	gitIn(t, dir, "init", "--quiet", "--initial-branch", "main")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "README")
	gitIn(t, dir, "-c", "commit.gpgsign=false", "-c", "user.name=Test User", "-c", "user.email="+testEmail, "commit", "--quiet", "-m", "chore: add base")
	t.Chdir(dir)

	var out, errOut bytes.Buffer
	code := DiagnoseWorktree("main", "main", &out, &errOut)
	if code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), dir) {
		t.Fatalf("expected output to mention worktree path: %q", out.String())
	}
	if !strings.Contains(out.String(), "setup") {
		t.Fatalf("expected setup state: %q", out.String())
	}
	if !strings.Contains(out.String(), "git worktree add --detach") {
		t.Fatalf("expected remediation guidance: %q", out.String())
	}
	if !strings.Contains(out.String(), "git worktree add -b issue/<number>") {
		t.Fatalf("expected issue-branch guidance: %q", out.String())
	}
}

func TestDiagnoseWorktreeNotesMissingSetup(t *testing.T) {
	dir := t.TempDir()
	gitIn(t, dir, "init", "--quiet", "--initial-branch", "main")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	gitIn(t, dir, "add", "README")
	gitIn(t, dir, "-c", "commit.gpgsign=false", "-c", "user.name=Test User", "-c", "user.email="+testEmail, "commit", "--quiet", "-m", "chore: add base")
	gitIn(t, dir, "worktree", "add", "-b", "issue/9", t.TempDir())
	t.Chdir(dir)

	var out, errOut bytes.Buffer
	code := DiagnoseWorktree("issue/9", "main", &out, &errOut)
	if code != 0 {
		t.Fatalf("expected 0, got %d: %s", code, errOut.String())
	}
	if !strings.Contains(out.String(), "Run 'mise run setup' there before continuing") {
		t.Fatalf("expected missing-setup guidance: %q", out.String())
	}
}

func TestSetupStateMissingStampIsNotRun(t *testing.T) {
	dir := t.TempDir()
	if state := SetupState(Worktree{Path: dir}); state != "not run" {
		t.Fatalf("expected not run, got %q", state)
	}
}

func TestSetupStateMissingPath(t *testing.T) {
	if state := SetupState(Worktree{Path: filepath.Join(t.TempDir(), "missing")}); state != "missing" {
		t.Fatalf("expected missing, got %q", state)
	}
}
