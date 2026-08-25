package check

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hidekitux/skills/internal/support"
)

func git(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = support.GitEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func seedRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	git(t, dir, "init", "--quiet", "--initial-branch", "main")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git(t, dir, "add", "README")
	git(t, dir, "-c", "commit.gpgsign=false", "-c", "user.name=Test User", "-c", "user.email="+testEmail, "commit", "--quiet", "-m", "chore: add base")
	return dir
}

func TestCheckWhitespacePassesOnCleanCommit(t *testing.T) {
	dir := seedRepo(t)
	t.Setenv("GITHUB_SHA", "")
	t.Setenv("CHECK_BASE_SHA", "")
	var out, errOut bytes.Buffer
	if code := CheckWhitespace(dir, &out, &errOut); code != 0 {
		t.Fatalf("expected 0, got %d: %s %s", code, out.String(), errOut.String())
	}
}

func TestCheckWhitespaceRejectsTrailingWhitespaceUnstaged(t *testing.T) {
	dir := seedRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("base   \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GITHUB_SHA", "")
	t.Setenv("CHECK_BASE_SHA", "")
	var out, errOut bytes.Buffer
	if code := CheckWhitespace(dir, &out, &errOut); code != 2 {
		t.Fatalf("expected 2 (git diff --check), got %d", code)
	}
}

var testEmail = "test" + "@" + "example.invalid"
