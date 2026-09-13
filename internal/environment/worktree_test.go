package environment

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type worktreeCommandCall struct {
	name string
	args []string
}

type worktreeScriptRunner struct {
	responses map[string]worktreeCommandResponse
	calls     []worktreeCommandCall
}

type worktreeCommandResponse struct {
	output string
	err    error
}

func (r *worktreeScriptRunner) Run(_ context.Context, _ string, _ []string, name string, args ...string) (string, error) {
	r.calls = append(r.calls, worktreeCommandCall{name: name, args: append([]string(nil), args...)})
	key := name + " " + strings.Join(args, " ")
	for prefix, response := range r.responses {
		if strings.HasPrefix(key, prefix) {
			return response.output, response.err
		}
	}
	return "", errors.New("unexpected command: " + key)
}

func TestParseNativeWorktreeListIncludesDetachedAndPrunableState(t *testing.T) {
	records, err := parseNativeWorktreeList("worktree /repo\nHEAD abc\nbranch refs/heads/issue/317\n\nworktree /gone\nHEAD def\ndetached HEAD\nprunable gitdir file is missing\n")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("records = %#v, want 2 records", records)
	}
	if records[0].Branch != "issue/317" || records[0].Detached || records[0].Prunable {
		t.Fatalf("branch record = %#v", records[0])
	}
	if !records[1].Detached || !records[1].Prunable || records[1].PrunableReason != "gitdir file is missing" {
		t.Fatalf("detached prunable record = %#v", records[1])
	}
}

func TestNativeGitProviderListDetectsConflictsFromStatus(t *testing.T) {
	runner := &worktreeScriptRunner{responses: map[string]worktreeCommandResponse{
		"git worktree list": {output: "worktree /repo.issue-317\nHEAD abc\nbranch refs/heads/issue/317\n"},
		"git status":        {output: "# branch.head issue/317\nu 100644 100644 100644 abc def file.txt\n"},
	}}
	records, err := (NativeGitWorktreeProvider{Runner: runner}).List(context.Background(), "/repo")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || !records[0].Conflicted {
		t.Fatalf("records = %#v", records)
	}
}

func TestParseWorktrunkListNormalizesSchemaTwoState(t *testing.T) {
	records, err := parseWorktrunkList(`{
  "schema": 2,
  "items": [{
    "branch": "issue/317",
    "worktree": {
      "path": "/repo.issue-317",
      "detached": false,
      "prunable": null,
      "operation": "rebase",
      "branch_mismatch": false,
      "duplicate_branch": false,
      "changes": {"conflicted": true}
    }
  }]
}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 {
		t.Fatalf("records = %#v, want 1 record", records)
	}
	record := records[0]
	if record.Path != "/repo.issue-317" || record.Branch != "issue/317" || record.Operation != "rebase" || !record.Conflicted {
		t.Fatalf("normalized record = %#v", record)
	}
}

func TestParseWorktrunkListNormalizesSchemaOneState(t *testing.T) {
	records, err := parseWorktrunkList(`[
  {
    "branch": "issue/317",
    "path": "/repo.issue-317",
    "operation_state": "conflicts",
    "worktree": {"state": "branch_mismatch", "reason": "configured path differs", "detached": false}
  }
]`)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 1 || !records[0].Conflicted || !records[0].BranchMismatch {
		t.Fatalf("normalized schema-one record = %#v", records)
	}
}

func TestParseWorktrunkListRejectsUnsupportedSchemaAndMissingPath(t *testing.T) {
	for name, input := range map[string]string{
		"schema":       `{"schema": 3, "items": []}`,
		"missing path": `{"schema": 2, "items": [{"branch": "issue/317", "worktree": null}]}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseWorktrunkList(input); err == nil {
				t.Fatal("expected structured state error")
			}
		})
	}
}

func TestWorktrunkProviderCreatesWithStructuredIdentityAndNoHooks(t *testing.T) {
	runner := &worktreeScriptRunner{responses: map[string]worktreeCommandResponse{
		"wt -C /repo list":   {output: `{"schema":2,"items":[]}`},
		"git show-ref":       {err: errors.New("branch not found")},
		"wt -C /repo switch": {output: `{"action":"created","branch":"issue/317","path":"/repo.issue-317","created_branch":true}`},
	}}
	provider := WorktrunkProvider{Runner: runner, Command: "wt"}
	state, err := provider.Create(context.Background(), "/repo", "/requested", "issue/317", strings.Repeat("a", 40))
	if err != nil {
		t.Fatal(err)
	}
	if state.Path != "/repo.issue-317" || state.Branch != "issue/317" {
		t.Fatalf("created state = %#v", state)
	}
	if len(runner.calls) != 3 {
		t.Fatalf("calls = %#v, want list, show-ref, switch", runner.calls)
	}
	switchCall := runner.calls[2]
	joined := strings.Join(switchCall.args, " ")
	for _, marker := range []string{"switch", "--no-cd", "--no-hooks", "--format=json", "--yes", "--create", "--base"} {
		if !strings.Contains(joined, marker) {
			t.Fatalf("switch args %q lack %q", joined, marker)
		}
	}
}

func TestWorktrunkProviderAdoptsExistingBranchPath(t *testing.T) {
	runner := &worktreeScriptRunner{responses: map[string]worktreeCommandResponse{
		"wt -C /repo list": {output: `{"schema":2,"items":[{"branch":"issue/317","worktree":{"path":"/repo.issue-317","detached":false,"changes":{"conflicted":false}}}]}`},
	}}
	provider := WorktrunkProvider{Runner: runner, Command: "wt"}
	state, err := provider.Create(context.Background(), "/repo", "/requested", "issue/317", strings.Repeat("a", 40))
	if err != nil {
		t.Fatal(err)
	}
	if state.Path != "/repo.issue-317" || len(runner.calls) != 1 {
		t.Fatalf("state/calls = %#v/%#v", state, runner.calls)
	}
}

func TestWorktrunkProviderRemovalUsesForegroundAndPreservesBranch(t *testing.T) {
	runner := &worktreeScriptRunner{responses: map[string]worktreeCommandResponse{
		"wt -C /repo remove": {output: `{"kind":"worktree","branch":"issue/317","path":"/repo.issue-317","branch_outcome":"not_attempted"}`},
	}}
	provider := WorktrunkProvider{Runner: runner, Command: "wt"}
	err := provider.Remove(context.Background(), "/repo", WorktreeState{Path: "/repo.issue-317", Branch: "issue/317"})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(runner.calls[0].args, " ")
	for _, marker := range []string{"remove", "--foreground", "--no-hooks", "--no-delete-branch", "--format=json", "--yes", "issue/317"} {
		if !strings.Contains(joined, marker) {
			t.Fatalf("remove args %q lack %q", joined, marker)
		}
	}
}

func TestNewLocalWorktreeProviderFallsBackWhenWorktrunkIsUnavailable(t *testing.T) {
	path := t.TempDir()
	t.Setenv("PATH", path)
	provider := NewLocalWorktreeProvider(nil)
	if _, ok := provider.(NativeGitWorktreeProvider); !ok {
		t.Fatalf("provider = %T, want NativeGitWorktreeProvider", provider)
	}
}

func TestNativeGitProviderUsesHooklessLifecycleCommands(t *testing.T) {
	root := t.TempDir()
	destination := filepath.Join(t.TempDir(), "issue-317")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	// The command stream is enough to verify the provider's lifecycle contract;
	// no repository mutation is needed for this focused argument test.
	runner := &worktreeScriptRunner{responses: map[string]worktreeCommandResponse{
		"git worktree list": {output: ""},
		"git status":        {output: ""},
		"git show-ref":      {err: errors.New("branch not found")},
		"git -c":            {err: errors.New("stop after argument capture")},
	}}
	provider := NativeGitWorktreeProvider{Runner: runner}
	_, err := provider.Create(context.Background(), root, destination, "issue/317", strings.Repeat("a", 40))
	if err == nil {
		t.Fatal("expected fixture command to stop after hookless command")
	}
	found := false
	for _, call := range runner.calls {
		if call.name == "git" && strings.Contains(strings.Join(call.args, " "), "core.hooksPath="+os.DevNull) {
			found = true
		}
	}
	if !found {
		t.Fatalf("calls = %#v, want a hookless git worktree command", runner.calls)
	}
}

type staticWorktreeProvider struct {
	worktrees []WorktreeState
	removed   bool
}

func (p *staticWorktreeProvider) Create(context.Context, string, string, string, string) (WorktreeState, error) {
	return WorktreeState{}, errors.New("not used")
}

func (p *staticWorktreeProvider) List(context.Context, string) ([]WorktreeState, error) {
	return p.worktrees, nil
}

func (p *staticWorktreeProvider) Remove(context.Context, string, WorktreeState) error {
	p.removed = true
	return nil
}

func TestCleanupRetainsStructuredUnsafeWorktreeWithReason(t *testing.T) {
	workspace := filepath.Join(t.TempDir(), "worktree")
	provider := &staticWorktreeProvider{worktrees: []WorktreeState{{
		Path: workspace, Branch: "issue/317", Prunable: true,
	}}}
	provisioner := Provisioner{Root: t.TempDir(), Worktree: provider}
	manifest := Manifest{
		EnvironmentID:      "environment-317",
		WorkspaceKind:      WorkspaceIssueWorktree,
		RepositoryRevision: strings.Repeat("a", 40),
		Branch:             "issue/317",
	}
	result, err := provisioner.Cleanup(context.Background(), CleanupRequest{
		Provisioned:    Provisioned{Manifest: manifest, WorkspacePath: workspace},
		ReviewApproved: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Cleanup != CleanupBlockedMaterial || result.Ownership.Diagnostic != "worktree-prunable" {
		t.Fatalf("cleanup result = %#v", result)
	}
	if provider.removed {
		t.Fatal("unsafe worktree was removed")
	}
}
