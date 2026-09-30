package check

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// bootstrapTemplateScripts is the directory of the validator scripts that
// bootstrap-project copies into the projects it generates.
const bootstrapTemplateScripts = "../../skills/govern/bootstrap-project/templates/github/scripts"

// runPythonScript runs script with python3 and returns its stdout, stderr,
// and exit status. extraEnv entries override the inherited environment.
func runPythonScript(t *testing.T, extraEnv []string, script string, args ...string) (string, string, int) {
	t.Helper()
	python, err := exec.LookPath("python3")
	if err != nil {
		t.Fatalf("python3 is required to test the scripts bundled with skills: %v", err)
	}
	path, err := filepath.Abs(script)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(python, append([]string{path}, args...)...)
	cmd.Env = append(os.Environ(), extraEnv...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err = cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		return stdout.String(), stderr.String(), 0
	case errors.As(err, &exitErr):
		return stdout.String(), stderr.String(), exitErr.ExitCode()
	default:
		t.Fatalf("run %s: %v", script, err)
		return "", "", -1
	}
}

func TestWorkItemTitleScriptAcceptsStandardAndReleaseTitles(t *testing.T) {
	script := filepath.Join(bootstrapTemplateScripts, "validate-work-item-title.py")
	for _, title := range []string{
		"[Feature]: Add sentence-case title checks",
		"[Maintenance]: Pin the GitHub Actions runner for iOS builds",
		"[Release]: v1.2.3",
		"[Release]: v1.2.3+4",
	} {
		stdout, stderr, code := runPythonScript(t, nil, script, "--title", title)
		if code != 0 || strings.TrimSpace(stdout) != "Work item title is valid." {
			t.Errorf("title %q: code=%d stdout=%q stderr=%q", title, code, stdout, stderr)
		}
	}
}

func TestWorkItemTitleScriptRejectsEachTitleClass(t *testing.T) {
	script := filepath.Join(bootstrapTemplateScripts, "validate-work-item-title.py")
	for _, tc := range []struct {
		title string
		want  string
	}{
		{"Add sentence-case title checks", "title must be [Type]: Summary"},
		{"[Chore]: Add sentence-case title checks", "title must be [Type]: Summary"},
		{"[Release]: 1.2.3", "title must be [Type]: Summary"},
		{"[Bug]: ", "summary must contain at least one non-empty word"},
		{"[Bug]: fix the parser", "summary must begin with a capital letter"},
		{"[Feature]: Add", "the bare template suffix \"Add\" is a placeholder"},
		{"[Feature]: Add New Title Case Words", "title-case run"},
	} {
		stdout, stderr, code := runPythonScript(t, nil, script, "--title", tc.title)
		if code != 1 || stdout != "" || !strings.Contains(stderr, tc.want) {
			t.Errorf("title %q: code=%d stdout=%q stderr=%q, want exit 1 with %q", tc.title, code, stdout, stderr, tc.want)
		}
	}
}

func TestCommitMessageScriptAcceptsSingleHeaderWithIssue(t *testing.T) {
	script := filepath.Join(bootstrapTemplateScripts, "validate-commit-message.py")
	for _, message := range []string{
		"fix: handle an empty input #12",
		"feat(parser): accept a trailing comma #345",
	} {
		stdout, stderr, code := runPythonScript(t, nil, script, "--message", message)
		if code != 0 || strings.TrimSpace(stdout) != "Commit message shape is valid." {
			t.Errorf("message %q: code=%d stdout=%q stderr=%q", message, code, stdout, stderr)
		}
	}
}

func TestCommitMessageScriptRejectsEachMessageClass(t *testing.T) {
	script := filepath.Join(bootstrapTemplateScripts, "validate-commit-message.py")
	for _, tc := range []struct {
		message string
		want    string
	}{
		{"", "commit message must not be empty"},
		{"fix: handle an empty input #12\n\nExplain the change.", "commit message must be exactly one line"},
		{"fix: handle an empty input", "header must be `type(scope): summary #NNN`"},
		{"fix: handle an empty input #abc", "header must be `type(scope): summary #NNN`"},
		{"fix: handle an empty input. #12", "summary must be a single sentence without terminal punctuation"},
	} {
		stdout, stderr, code := runPythonScript(t, nil, script, "--message", tc.message)
		if code != 1 || stdout != "" || !strings.Contains(stderr, tc.want) {
			t.Errorf("message %q: code=%d stdout=%q stderr=%q, want exit 1 with %q", tc.message, code, stdout, stderr, tc.want)
		}
	}
}
