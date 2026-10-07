package eval

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/provider"
)

// refsStub describes the sandbox state a stubbed githubPort reports: each
// successive `gh pr list` and branch listing returns the next entry, and the
// commands named in fail are refused.
type refsStub struct {
	pulls    []string
	branches []string
	fail     map[string]bool
}

func stubSandboxRefs(t *testing.T, state refsStub) *provider.Stub {
	t.Helper()
	pullCalls, branchCalls := 0, 0
	stub := &provider.Stub{Handler: func(command provider.Command) (provider.Result, error) {
		args := strings.Join(command.Args, " ")
		switch {
		case strings.HasPrefix(args, "pr list"):
			if pullCalls >= len(state.pulls) {
				return provider.Fail(command, provider.KindFailure, 1, "listing unavailable")
			}
			pullCalls++
			return provider.Result{Combined: state.pulls[pullCalls-1]}, nil
		case strings.Contains(args, "/branches"):
			if branchCalls >= len(state.branches) {
				return provider.Fail(command, provider.KindFailure, 1, "listing unavailable")
			}
			branchCalls++
			return provider.Result{Combined: state.branches[branchCalls-1]}, nil
		case strings.Contains(args, ".default_branch"):
			return provider.Result{Combined: "main\n"}, nil
		case state.fail["pr close"] && strings.HasPrefix(args, "pr close"),
			state.fail["delete"] && strings.Contains(args, "DELETE"):
			return provider.Fail(command, provider.KindFailure, 1, "refused")
		}
		return provider.Result{}, nil
	}}
	previous := githubPort
	githubPort = provider.NewGitHub(stub)
	t.Cleanup(func() { githubPort = previous })
	return stub
}

func closedPulls(stub *provider.Stub) []string {
	var closed []string
	for _, call := range stub.Calls {
		if len(call.Args) >= 3 && call.Args[0] == "pr" && call.Args[1] == "close" {
			closed = append(closed, call.Args[2])
		}
	}
	return closed
}

func deletedBranches(stub *provider.Stub) []string {
	var deleted []string
	for _, call := range stub.Calls {
		args := strings.Join(call.Args, " ")
		if i := strings.Index(args, "/git/refs/heads/"); i >= 0 && strings.Contains(args, "DELETE") {
			deleted = append(deleted, args[i+len("/git/refs/heads/"):])
		}
	}
	return deleted
}

func TestCleanNewSandboxRefsClosesOnlyNewPullRequests(t *testing.T) {
	stub := stubSandboxRefs(t, refsStub{pulls: []string{"70\n", "70\n74\n75\n"}, branches: []string{"main\n", "main\n"}})
	before, err := snapshotSandboxRefs(context.Background(), "owner/sandbox")
	if err != nil {
		t.Fatal(err)
	}
	if err := cleanNewSandboxRefs(context.Background(), "owner/sandbox", "e2e-review-mechanism-flow", before); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(closedPulls(stub), ","); got != "74,75" {
		t.Fatalf("closed pull requests = %q, want 74,75; pre-existing #70 must stay open", got)
	}
	for _, call := range stub.Calls {
		if len(call.Args) >= 2 && call.Args[0] == "pr" && call.Args[1] == "close" && !strings.Contains(strings.Join(call.Args, " "), "e2e-review-mechanism-flow") {
			t.Fatalf("close comment does not name the scenario: %v", call.Args)
		}
	}
}

func TestCleanNewSandboxRefsDeletesOnlyNewBranches(t *testing.T) {
	stub := stubSandboxRefs(t, refsStub{
		pulls:    []string{"", ""},
		branches: []string{"main\nissue/68\n", "main\nissue/68\nissue/73\nissue/74\n"},
	})
	before, err := snapshotSandboxRefs(context.Background(), "owner/sandbox")
	if err != nil {
		t.Fatal(err)
	}
	if err := cleanNewSandboxRefs(context.Background(), "owner/sandbox", "s", before); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(deletedBranches(stub), ","); got != "issue/73,issue/74" {
		t.Fatalf("deleted branches = %q, want issue/73,issue/74; pre-existing issue/68 must stay", got)
	}
}

func TestCleanNewSandboxRefsNeverDeletesTheDefaultBranch(t *testing.T) {
	// The default branch is missing from the first listing, as in a sandbox
	// whose main branch a scenario creates.
	stub := stubSandboxRefs(t, refsStub{pulls: []string{"", ""}, branches: []string{"", "main\nissue/9\n"}})
	before, err := snapshotSandboxRefs(context.Background(), "owner/sandbox")
	if err != nil {
		t.Fatal(err)
	}
	if err := cleanNewSandboxRefs(context.Background(), "owner/sandbox", "s", before); err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(deletedBranches(stub), ","); got != "issue/9" {
		t.Fatalf("deleted branches = %q, want issue/9 only", got)
	}
}

func TestRunOneCleansPullRequestsAndBranchesTheScenarioCreated(t *testing.T) {
	t.Setenv("EVAL_GITHUB_REPO", "owner/sandbox")
	stub := stubSandboxRefs(t, refsStub{pulls: []string{"", "5\n"}, branches: []string{"main\n", "main\nissue/4\n"}})
	sc := &Scenario{ID: "create-pr-success", Skill: "create-pr", Kind: KindPositive, GithubSandbox: true, Prompt: "p"}
	runOneForTest(t, sc, passingFake("codex"), &Options{})
	if got := strings.Join(closedPulls(stub), ","); got != "5" {
		t.Fatalf("closed pull requests = %q, want 5", got)
	}
	if got := strings.Join(deletedBranches(stub), ","); got != "issue/4" {
		t.Fatalf("deleted branches = %q, want issue/4", got)
	}
}

func TestRunOneReportsSandboxRefCleanupFailureWithoutChangingTheVerdict(t *testing.T) {
	t.Setenv("EVAL_GITHUB_REPO", "owner/sandbox")
	stubSandboxRefs(t, refsStub{
		pulls:    []string{"", "5\n"},
		branches: []string{"main\n", "main\nissue/4\n"},
		fail:     map[string]bool{"pr close": true, "delete": true},
	})
	sc := &Scenario{ID: "create-pr-success", Skill: "create-pr", Kind: KindPositive, GithubSandbox: true, Prompt: "p"}
	var errOut bytes.Buffer
	record := runOne(context.Background(), sc, passingFake("codex"), &Options{}, io.Discard, &errOut)
	if record.Verdict == VerdictInfra {
		t.Fatalf("verdict = %s (%s), want the scenario verdict unchanged by a cleanup failure", record.Verdict, record.InfraError)
	}
	for _, want := range []string{"cannot clean sandbox pull requests and branches", "pull request #5", "branch issue/4"} {
		if !strings.Contains(errOut.String(), want) {
			t.Fatalf("errOut = %q, want %q", errOut.String(), want)
		}
	}
}

func TestRunOneReportsSandboxRefListingFailure(t *testing.T) {
	t.Setenv("EVAL_GITHUB_REPO", "owner/sandbox")
	stubSandboxRefs(t, refsStub{})
	sc := &Scenario{ID: "create-pr-success", Skill: "create-pr", Kind: KindPositive, GithubSandbox: true, Prompt: "p"}
	host := passingFake("codex")
	record := runOneForTest(t, sc, host, &Options{})
	if record.Verdict != VerdictInfra || !strings.Contains(record.InfraError, "sandbox pull request and branch listing") {
		t.Fatalf("verdict = %s (%s), want an infrastructure error for the listing", record.Verdict, record.InfraError)
	}
	if host.CallCount() != 0 {
		t.Fatalf("host ran %d times, want none after a listing failure", host.CallCount())
	}
}
