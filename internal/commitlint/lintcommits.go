package commitlint

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

const dependabotAuthor = "dependabot[bot]"

// commandRunner abstracts subprocess execution so the linter is testable
// without a real Git repository. runValue writes input to stdin (empty for
// none) and returns combined output and the process exit code.
type commandRunner interface {
	runValue(input string, name string, args ...string) (string, int)
}

type execRunner struct{}

// ExecRunner returns the real subprocess-backed command runner used in
// production command entrypoints.
func ExecRunner() commandRunner { return execRunner{} }

func (execRunner) runValue(input, name string, args ...string) (string, int) {
	cmd := exec.Command(name, args...)
	var buffer bytes.Buffer
	cmd.Stdout = &buffer
	cmd.Stderr = &buffer
	if input != "" {
		cmd.Stdin = strings.NewReader(input)
	}
	err := cmd.Run()
	code := 0
	if err != nil {
		var exitErr *exec.ExitError
		if ok := asExitError(err, &exitErr); ok {
			code = exitErr.ExitCode()
		} else {
			code = 1
		}
	}
	return buffer.String(), code
}

func asExitError(err error, target **exec.ExitError) bool {
	for err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			*target = exitErr
			return true
		}
		type unwrapper interface{ Unwrap() error }
		if u, ok := err.(unwrapper); ok {
			err = u.Unwrap()
			continue
		}
		return false
	}
	return false
}

func isDependabotPullRequest() bool {
	return os.Getenv("PR_AUTHOR") == dependabotAuthor
}

// resolveCommitlint returns the commitlint binary path, or an error carrying
// exit code 2 when none is installed.
func resolveCommitlint(root string, runner commandRunner) (string, int) {
	if override := os.Getenv("COMMITLINT_BIN"); override != "" {
		return override, 0
	}
	candidate := root + "/.mise/bin/commitlint"
	if info, err := os.Stat(candidate); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
		return candidate, 0
	}
	return "", 2
}

func collectCommits(base, head string, runner commandRunner) []string {
	argv := []string{"rev-list", "--no-merges"}
	if base != "" {
		argv = append(argv, base+".."+head)
	} else {
		argv = append(argv, head)
	}
	stdout, _ := runner.runValue("", "git", argv...)
	var commits []string
	for _, line := range strings.Split(stdout, "\n") {
		if line != "" {
			commits = append(commits, line)
		}
	}
	return commits
}

func commitAuthor(commit string, runner commandRunner) string {
	stdout, _ := runner.runValue("", "git", "show", "-s", "--format=%an", commit)
	return strings.TrimSpace(stdout)
}

func isDependabotCommit(commit string, runner commandRunner) bool {
	return commitAuthor(commit, runner) == dependabotAuthor
}

func messageFor(commit string, runner commandRunner) string {
	stdout, _ := runner.runValue("", "git", "show", "-s", "--format=%B", commit)
	return strings.TrimSuffix(stdout, "\n")
}

func lintMessage(message, commitlintBin string, runner commandRunner, out, errOut io.Writer) int {
	stdout, code := runner.runValue(message, commitlintBin, "lint")
	if code != 0 {
		fmt.Fprint(errOut, stdout)
		return code
	}
	errors := ValidateMessage(message)
	if len(errors) > 0 {
		fmt.Fprintf(errOut, "error: %s\n", strings.Join(errors, "; "))
		return 1
	}
	fmt.Fprintln(out, "Commit message shape is valid.")
	return 0
}

// LintCommits drives commitlint over the commits in the current push or pull
// request range, returning the process exit code.
func LintCommits(runner commandRunner, out, errOut io.Writer) int {
	root, _ := runner.runValue("", "git", "rev-parse", "--show-toplevel")
	root = strings.TrimSpace(root)
	if isDependabotPullRequest() {
		fmt.Fprintln(out, "Skipping commit lint for a Dependabot pull request.")
		return 0
	}
	commitlintBin, code := resolveCommitlint(root, runner)
	if code != 0 {
		fmt.Fprintln(errOut, "commitlint is not installed for this project. Run: mise run setup:commitlint")
		return 2
	}
	prContext := os.Getenv("PR_BASE_SHA") != "" && os.Getenv("PR_HEAD_SHA") != ""
	var commits []string
	if prContext {
		commits = collectCommits(os.Getenv("PR_BASE_SHA"), os.Getenv("PR_HEAD_SHA"), runner)
	} else {
		before := os.Getenv("PUSH_BEFORE_SHA")
		after := os.Getenv("GITHUB_SHA")
		if after == "" {
			after = "HEAD"
		}
		if before != "" && !isAllZeros(before) {
			commits = collectCommits(before, after, runner)
		} else {
			commits = collectCommits("", after, runner)
		}
	}
	if len(commits) == 0 {
		fmt.Fprintln(out, "No commits to lint.")
		return 0
	}
	for _, commit := range commits {
		fmt.Fprintf(out, "Checking commit %s\n", commit)
		if !prContext && isDependabotCommit(commit, runner) {
			fmt.Fprintf(out, "Skipping Dependabot-authored commit %s\n", commit)
			continue
		}
		message := messageFor(commit, runner)
		if status := lintMessage(message, commitlintBin, runner, out, errOut); status != 0 {
			return status
		}
	}
	return 0
}

func isAllZeros(sha string) bool {
	for _, r := range sha {
		if r != '0' {
			return false
		}
	}
	return true
}
