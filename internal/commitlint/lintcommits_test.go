package commitlint

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type fakeRunner struct {
	rootOut           string
	revListStdout     string
	showOutputs       []string
	authorOutputs     []string
	commitlintReturn  int
	missingCommitlint bool
	missingRevParse   bool
	calls             []string
}

func (f *fakeRunner) runValue(input, name string, args ...string) (string, int) {
	f.calls = append(f.calls, name+" "+strings.Join(args, " "))
	if name == "git" {
		switch args[0] {
		case "rev-parse":
			if f.missingRevParse {
				return "", 1
			}
			return f.rootOut + "\n", 0
		case "rev-list":
			return f.revListStdout, 0
		case "show":
			if containsRune(args, "--format=%an") {
				next := f.authorOutputs[0]
				f.authorOutputs = f.authorOutputs[1:]
				return next + "\n", 0
			}
			next := f.showOutputs[0]
			f.showOutputs = f.showOutputs[1:]
			return next, 0
		}
	}
	if len(args) == 1 && args[0] == "lint" {
		if f.missingCommitlint {
			return "", 127
		}
		return "", f.commitlintReturn
	}
	return "", 1
}

func containsRune(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func defaultFake() *fakeRunner {
	return &fakeRunner{
		rootOut:          "repo-root",
		authorOutputs:    []string{"Jane Doe"},
		showOutputs:      []string{"feat: add feature #1\n\n"},
		commitlintReturn: 0,
	}
}

func hasCall(runner *fakeRunner, fragment string) bool {
	for _, call := range runner.calls {
		if strings.Contains(call, fragment) {
			return true
		}
	}
	return false
}

func TestResolveCommitlintUsesEnvironmentOverride(t *testing.T) {
	t.Setenv("COMMITLINT_BIN", "/custom/bin/commitlint")
	got, code := resolveCommitlint("repo-root", defaultFake())
	if code != 0 || got != "/custom/bin/commitlint" {
		t.Fatalf("got %q code %d", got, code)
	}
}

func TestResolveCommitlintUsesProjectLocalBinary(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, ".mise", "bin", "commitlint")
	if err := os.MkdirAll(filepath.Dir(bin), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(bin, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, code := resolveCommitlint(dir, defaultFake())
	if code != 0 || got != bin {
		t.Fatalf("got %q code %d", got, code)
	}
}

func TestResolveCommitlintExits2WhenMissing(t *testing.T) {
	dir := t.TempDir()
	_, code := resolveCommitlint(dir, defaultFake())
	if code != 2 {
		t.Fatalf("expected exit 2, got %d", code)
	}
}

func TestCollectCommitsRunsRevListWithoutMergesRange(t *testing.T) {
	runner := defaultFake()
	runner.revListStdout = "abc123\ndef456\n"
	got := collectCommits("base", "head", runner)
	if len(got) != 2 || got[0] != "abc123" || got[1] != "def456" {
		t.Fatalf("unexpected commits %v", got)
	}
	if !hasCall(runner, "rev-list --no-merges base..head") {
		t.Fatalf("missing rev-list range call: %v", runner.calls)
	}
}

func TestCollectCommitsSupportsSingleSide(t *testing.T) {
	runner := defaultFake()
	runner.revListStdout = "abc123\n"
	got := collectCommits("", "HEAD", runner)
	if len(got) != 1 || got[0] != "abc123" {
		t.Fatalf("unexpected commits %v", got)
	}
	if !hasCall(runner, "rev-list --no-merges HEAD") {
		t.Fatalf("missing rev-list call: %v", runner.calls)
	}
}

func TestLintMessageReportsCommitlintFailure(t *testing.T) {
	runner := defaultFake()
	runner.commitlintReturn = 1
	var out, errOut bytes.Buffer
	if code := lintMessage("feat: bad header #1\n", "/bin/commitlint", runner, &out, &errOut); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	if !hasCall(runner, "commitlint lint") {
		t.Fatalf("missing commitlint call: %v", runner.calls)
	}
}

func TestLintMessageReportsValidatorFailure(t *testing.T) {
	runner := defaultFake()
	var out, errOut bytes.Buffer
	if code := lintMessage("feat: no issue number\n", "/bin/commitlint", runner, &out, &errOut); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "numeric issue number") {
		t.Fatalf("missing validator guidance: %q", errOut.String())
	}
}

func TestLintMessagePassesValidMessage(t *testing.T) {
	runner := defaultFake()
	var out, errOut bytes.Buffer
	if code := lintMessage("feat: add feature #1\n", "/bin/commitlint", runner, &out, &errOut); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

func TestLintCommitsLintsPRRangeAndExitsZero(t *testing.T) {
	t.Setenv("COMMITLINT_BIN", "/bin/commitlint")
	t.Setenv("PR_BASE_SHA", "base")
	t.Setenv("PR_HEAD_SHA", "head")
	runner := defaultFake()
	runner.revListStdout = "abc123\n"
	var out, errOut bytes.Buffer
	if code := LintCommits(runner, &out, &errOut); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !hasCall(runner, "rev-list --no-merges base..head") {
		t.Fatalf("missing PR range: %v", runner.calls)
	}
	if !hasCall(runner, "show -s --format=%B abc123") {
		t.Fatalf("missing message fetch: %v", runner.calls)
	}
}

func TestLintCommitsExitsOneWhenIssueNumberMissing(t *testing.T) {
	t.Setenv("COMMITLINT_BIN", "/bin/commitlint")
	t.Setenv("PR_BASE_SHA", "base")
	t.Setenv("PR_HEAD_SHA", "head")
	runner := defaultFake()
	runner.revListStdout = "abc123\n"
	runner.showOutputs = []string{"feat: change readme\n"}
	var out, errOut bytes.Buffer
	if code := LintCommits(runner, &out, &errOut); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	if !strings.Contains(errOut.String(), "numeric issue number") {
		t.Fatalf("missing guidance: %q", errOut.String())
	}
}

func TestLintCommitsReportsNoCommits(t *testing.T) {
	t.Setenv("COMMITLINT_BIN", "/bin/commitlint")
	runner := defaultFake()
	runner.revListStdout = ""
	var out, errOut bytes.Buffer
	if code := LintCommits(runner, &out, &errOut); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(out.String(), "No commits to lint.") {
		t.Fatalf("missing message: %q", out.String())
	}
	if !hasCall(runner, "rev-list --no-merges HEAD") {
		t.Fatalf("missing single-side call: %v", runner.calls)
	}
}

func TestLintCommitsPrefersPushBeforeSHA(t *testing.T) {
	t.Setenv("COMMITLINT_BIN", "/bin/commitlint")
	t.Setenv("PUSH_BEFORE_SHA", "beef")
	t.Setenv("GITHUB_SHA", "cafe")
	runner := defaultFake()
	runner.revListStdout = "abc123\n"
	var out, errOut bytes.Buffer
	if code := LintCommits(runner, &out, &errOut); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !hasCall(runner, "rev-list --no-merges beef..cafe") {
		t.Fatalf("missing push range: %v", runner.calls)
	}
}

func TestLintCommitsTreatsAllZeroPushBeforeAsSingleSide(t *testing.T) {
	t.Setenv("COMMITLINT_BIN", "/bin/commitlint")
	t.Setenv("PUSH_BEFORE_SHA", "0000")
	t.Setenv("GITHUB_SHA", "cafe")
	runner := defaultFake()
	runner.revListStdout = ""
	var out, errOut bytes.Buffer
	if code := LintCommits(runner, &out, &errOut); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !hasCall(runner, "rev-list --no-merges cafe") {
		t.Fatalf("missing single-side call: %v", runner.calls)
	}
}

func TestLintCommitsExits2WhenCommitlintMissing(t *testing.T) {
	runner := defaultFake()
	runner.missingCommitlint = true
	var out, errOut bytes.Buffer
	if code := LintCommits(runner, &out, &errOut); code != 2 {
		t.Fatalf("expected 2, got %d", code)
	}
	if !strings.Contains(errOut.String(), "commitlint is not installed") {
		t.Fatalf("missing guidance: %q", errOut.String())
	}
}

func TestIsDependabotPullRequestMatchesAuthor(t *testing.T) {
	t.Setenv("PR_AUTHOR", "dependabot[bot]")
	if !isDependabotPullRequest() {
		t.Fatal("expected dependabot author match")
	}
}

func TestIsDependabotPullRequestIgnoresHeadRefWithoutAuthor(t *testing.T) {
	t.Setenv("PR_HEAD_REF", "dependabot/github_actions/actions-checkout")
	if isDependabotPullRequest() {
		t.Fatal("PR_AUTHOR must be the deciding signal")
	}
}

func TestLintCommitsSkipsDependabotPullRequest(t *testing.T) {
	t.Setenv("PR_AUTHOR", "dependabot[bot]")
	runner := defaultFake()
	var out, errOut bytes.Buffer
	if code := LintCommits(runner, &out, &errOut); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Skipping commit lint for a Dependabot pull request.") {
		t.Fatalf("missing skip message: %q", out.String())
	}
}

func TestLintCommitsSkipsDependabotAuthoredCommitsInPushRange(t *testing.T) {
	t.Setenv("COMMITLINT_BIN", "/bin/commitlint")
	t.Setenv("PUSH_BEFORE_SHA", "beef")
	t.Setenv("GITHUB_SHA", "cafe")
	runner := defaultFake()
	runner.revListStdout = "deps123\nhuman456\n"
	runner.authorOutputs = []string{"dependabot[bot]", "Jane Doe"}
	runner.showOutputs = []string{"feat: add feature #1\n\n"}
	var out, errOut bytes.Buffer
	if code := LintCommits(runner, &out, &errOut); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !strings.Contains(out.String(), "Checking commit deps123") {
		t.Fatalf("missing first commit: %q", out.String())
	}
	if !strings.Contains(out.String(), "Skipping Dependabot-authored commit deps123") {
		t.Fatalf("missing skip message: %q", out.String())
	}
	if !strings.Contains(out.String(), "Checking commit human456") {
		t.Fatalf("missing second commit: %q", out.String())
	}
}

func TestLintCommitsLintsHumanPRWithDependabotHeadRef(t *testing.T) {
	t.Setenv("COMMITLINT_BIN", "/bin/commitlint")
	t.Setenv("PR_BASE_SHA", "base")
	t.Setenv("PR_HEAD_SHA", "head")
	t.Setenv("PR_HEAD_REF", "dependabot/example")
	t.Setenv("PR_AUTHOR", "octocat")
	runner := defaultFake()
	runner.revListStdout = "abc123\n"
	var out, errOut bytes.Buffer
	if code := LintCommits(runner, &out, &errOut); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	if !hasCall(runner, "rev-list --no-merges base..head") {
		t.Fatalf("missing PR range: %v", runner.calls)
	}
}

func TestLintCommitsRejectsDependabotAuthoredCommitInHumanPR(t *testing.T) {
	t.Setenv("COMMITLINT_BIN", "/bin/commitlint")
	t.Setenv("PR_BASE_SHA", "base")
	t.Setenv("PR_HEAD_SHA", "head")
	t.Setenv("PR_HEAD_REF", "issue/123")
	t.Setenv("PR_AUTHOR", "octocat")
	runner := defaultFake()
	runner.revListStdout = "abc123\n"
	runner.authorOutputs = []string{"dependabot[bot]"}
	runner.showOutputs = []string{"Bump actions/checkout from 4 to 5\n"}
	runner.commitlintReturn = 1
	var out, errOut bytes.Buffer
	if code := LintCommits(runner, &out, &errOut); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	if !hasCall(runner, "commitlint lint") {
		t.Fatalf("missing commitlint call: %v", runner.calls)
	}
}
