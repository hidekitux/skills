package eval

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/provider"
)

// stubSandboxIssues substitutes githubPort with a stub that reports the open
// issue numbers listed by each successive `gh issue list` call and records
// every `gh issue close` call.
func stubSandboxIssues(t *testing.T, listings ...string) *provider.Stub {
	t.Helper()
	return stubSandboxIssuesWith(t, false, listings...)
}

// stubSandboxIssuesWith is stubSandboxIssues with every `gh issue close`
// failing when failClose is true.
func stubSandboxIssuesWith(t *testing.T, failClose bool, listings ...string) *provider.Stub {
	t.Helper()
	calls := 0
	stub := &provider.Stub{Handler: func(command provider.Command) (provider.Result, error) {
		if len(command.Args) >= 2 && command.Args[0] == "issue" && command.Args[1] == "list" {
			if calls >= len(listings) {
				return provider.Fail(command, provider.KindFailure, 1, "listing unavailable")
			}
			listing := listings[calls]
			calls++
			return provider.Result{Combined: listing}, nil
		}
		if failClose && len(command.Args) >= 2 && command.Args[0] == "issue" && command.Args[1] == "close" {
			return provider.Fail(command, provider.KindFailure, 1, "close refused")
		}
		return provider.Result{}, nil
	}}
	previous := githubPort
	githubPort = provider.NewGitHub(stub)
	t.Cleanup(func() { githubPort = previous })
	return stub
}

func closedIssues(stub *provider.Stub) []string {
	var closed []string
	for _, call := range stub.Calls {
		if len(call.Args) >= 3 && call.Args[0] == "issue" && call.Args[1] == "close" {
			closed = append(closed, call.Args[2])
		}
	}
	return closed
}

func TestCloseNewSandboxIssuesClosesOnlyNewIssues(t *testing.T) {
	stub := stubSandboxIssues(t, "4\n", "4\n7\n9\n")
	before, err := openSandboxIssues(context.Background(), "owner/sandbox")
	if err != nil {
		t.Fatal(err)
	}
	if err := closeNewSandboxIssues(context.Background(), "owner/sandbox", "create-issue-success", before); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(closedIssues(stub), ","); got != "7,9" {
		t.Fatalf("closed issues = %q, want 7,9; pre-existing #4 must stay open", got)
	}
	for _, call := range stub.Calls {
		if len(call.Args) >= 2 && call.Args[1] == "close" && !strings.Contains(strings.Join(call.Args, " "), "create-issue-success") {
			t.Fatalf("close comment does not name the scenario: %v", call.Args)
		}
	}
}

func TestOpenSandboxIssuesReportsListingFailure(t *testing.T) {
	stubSandboxIssues(t)
	if _, err := openSandboxIssues(context.Background(), "owner/sandbox"); err == nil {
		t.Fatal("openSandboxIssues error = nil, want the listing failure")
	}
}

func TestRunOneClosesIssuesTheScenarioCreated(t *testing.T) {
	t.Setenv("EVAL_GITHUB_REPO", "owner/sandbox")
	stub := stubSandboxIssues(t, "1\n", "1\n2\n")
	sc := &Scenario{ID: "create-issue-success", Skill: "create-issue", Kind: KindPositive, GithubSandbox: true, Prompt: "p"}
	runOneForTest(t, sc, passingFake("codex"), &Options{})
	if got := strings.Join(closedIssues(stub), ","); got != "2" {
		t.Fatalf("closed issues = %q, want 2", got)
	}
}

func TestRunOneReportsSandboxListingFailure(t *testing.T) {
	t.Setenv("EVAL_GITHUB_REPO", "owner/sandbox")
	stubSandboxIssues(t)
	sc := &Scenario{ID: "create-issue-success", Skill: "create-issue", Kind: KindPositive, GithubSandbox: true, Prompt: "p"}
	host := passingFake("codex")
	record := runOneForTest(t, sc, host, &Options{})
	if record.Verdict != VerdictInfra || !strings.Contains(record.InfraError, "sandbox issue listing") {
		t.Fatalf("verdict = %s (%s), want an infrastructure error for the listing", record.Verdict, record.InfraError)
	}
	if host.CallCount() != 0 {
		t.Fatalf("host ran %d times, want none after a listing failure", host.CallCount())
	}
}

func TestRunOneReportsSandboxCloseFailure(t *testing.T) {
	t.Setenv("EVAL_GITHUB_REPO", "owner/sandbox")
	stubSandboxIssuesWith(t, true, "1\n", "1\n2\n")
	sc := &Scenario{ID: "create-issue-success", Skill: "create-issue", Kind: KindPositive, GithubSandbox: true, Prompt: "p"}
	var errOut bytes.Buffer
	runOne(context.Background(), sc, passingFake("codex"), &Options{}, io.Discard, &errOut)
	if !strings.Contains(errOut.String(), "cannot close sandbox issues") || !strings.Contains(errOut.String(), "#2") {
		t.Fatalf("errOut = %q, want the close failure for #2", errOut.String())
	}
}
