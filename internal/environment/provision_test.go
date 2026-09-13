package environment

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/graph"
	"github.com/hidekitux/skills/internal/support"
)

type provisioningRunner struct {
	setupCalls int
	setupErr   error
}

func (r *provisioningRunner) Run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	if name == "mise" {
		r.setupCalls++
		return "", r.setupErr
	}
	return (OSCommandRunner{}).Run(ctx, dir, env, name, args...)
}

func TestProvisionReadOnlySnapshotRunsSetupAndDeniesMutation(t *testing.T) {
	root, revision := testRepository(t)
	destination := filepath.Join(t.TempDir(), "snapshot")
	runner := &provisioningRunner{}
	provisioner := Provisioner{
		Root: root,
		Graph: &graph.Graph{
			SchemaVersion: 1,
			Skills: []graph.Skill{{
				ID:        "plan-issue",
				Authority: graph.Authority{Repository: "read", Git: "read", GitHub: "read", ExternalMutation: "none"},
			}},
		},
		Runner: runner,
	}

	result, err := provisioner.Provision(context.Background(), ProvisionRequest{
		SkillID: "plan-issue", Destination: destination, Revision: revision,
	})
	defer restoreWritePermissions(destination)
	if err != nil {
		t.Fatal(err)
	}
	if runner.setupCalls != 1 {
		t.Fatalf("setup calls = %d, want 1", runner.setupCalls)
	}
	if result.Manifest.Profile != ProfileReadOnly || result.Manifest.WorkspaceKind != WorkspaceDetachedSnapshot {
		t.Fatalf("unexpected manifest: %#v", result.Manifest)
	}
	if result.Manifest.Setup.Status != SetupSucceeded || result.Manifest.Ownership.Status != OwnershipVerified {
		t.Fatalf("setup or ownership gate did not succeed: %#v", result.Manifest)
	}
	if mode := fileMode(t, filepath.Join(destination, "README.md")); mode&0o222 != 0 {
		t.Fatalf("snapshot file remains writable: %o", mode)
	}

	policyGit := filepath.Join(result.PolicyDir, "bin", "git")
	cmd := exec.Command(policyGit, "branch", "-d", "issue/199")
	cmd.Dir = destination
	output, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(output), "denied") {
		t.Fatalf("read-only branch deletion was not denied: err=%v output=%q", err, output)
	}
	policyGH := filepath.Join(result.PolicyDir, "bin", "gh")
	remote := exec.Command(policyGH, "issue", "close", "199")
	remote.Dir = destination
	remoteOutput, remoteErr := remote.CombinedOutput()
	if remoteErr == nil || !strings.Contains(string(remoteOutput), "skill-environment") {
		t.Fatalf("read-only GitHub mutation was not denied: err=%v output=%q", remoteErr, remoteOutput)
	}
}

func TestOSCommandRunnerScrubsExplicitGitEnvironment(t *testing.T) {
	root, _ := testRepository(t)
	other, _ := testRepository(t)
	env := append(support.GitEnv(),
		"GIT_DIR="+filepath.Join(other, ".git"),
		"GIT_WORK_TREE="+other,
	)

	output, err := (OSCommandRunner{}).Run(context.Background(), root, env, "git", "rev-parse", "--show-toplevel")
	if err != nil {
		t.Fatalf("git command failed: %v", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(output) != resolvedRoot {
		t.Fatalf("git command used ambient repository: got %q, want %q", strings.TrimSpace(output), resolvedRoot)
	}
}

func TestGHPolicyDeniesMutationForms(t *testing.T) {
	directory := t.TempDir()
	realGH := filepath.Join(directory, "real-gh")
	policyGH := filepath.Join(directory, "gh")
	if err := os.WriteFile(realGH, []byte("#!/bin/sh\nprintf '%s\\n' \"$*\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(policyGH, []byte(ghPolicyScript(realGH)), 0o755); err != nil {
		t.Fatal(err)
	}

	denied := [][]string{
		{"api", "-X", "POST", "repos/hidekitux/skills/issues/199"},
		{"api", "--method", "PATCH", "repos/hidekitux/skills/issues/199"},
		{"api", "--method=PUT", "repos/hidekitux/skills/issues/199"},
		{"api", "-XDELETE", "repos/hidekitux/skills/issues/199"},
		{"api", "-f", "body=closed", "repos/hidekitux/skills/issues/199"},
		{"api", "--field=body=closed", "repos/hidekitux/skills/issues/199"},
		{"api", "--input", "payload.json", "repos/hidekitux/skills/issues/199"},
		{"api", "-fbody=closed", "repos/hidekitux/skills/issues/199"},
		{"api", "-Fbody=closed", "repos/hidekitux/skills/issues/199"},
	}
	for _, args := range denied {
		name := strings.Join(args, "_")
		t.Run(name, func(t *testing.T) {
			command := exec.Command(policyGH, args...)
			output, err := command.CombinedOutput()
			if err == nil || !strings.Contains(string(output), "denied GitHub mutation") {
				t.Fatalf("mutation was not denied: err=%v output=%q", err, output)
			}
		})
	}

	allowed := [][]string{
		{"api", "repos/hidekitux/skills/issues/199"},
		{"api", "-X", "GET", "repos/hidekitux/skills/issues", "-f", "state=open"},
		{"api", "--method=HEAD", "repos/hidekitux/skills/issues/199"},
	}
	for _, args := range allowed {
		name := "allow_" + strings.Join(args, "_")
		t.Run(name, func(t *testing.T) {
			command := exec.Command(policyGH, args...)
			output, err := command.CombinedOutput()
			if err != nil || !strings.Contains(string(output), "api") {
				t.Fatalf("safe API request was not allowed: err=%v output=%q", err, output)
			}
		})
	}
}

func TestProvisionWriteProfileRequiresAndOwnsIssueBranch(t *testing.T) {
	root, revision := testRepository(t)
	destination := filepath.Join(t.TempDir(), "issue-worktree")
	runner := &provisioningRunner{}
	var verified int
	provisioner := Provisioner{
		Root: root,
		Graph: &graph.Graph{
			SchemaVersion: 1,
			Skills: []graph.Skill{{
				ID:        "implement-issue",
				Authority: graph.Authority{Repository: "write", Git: "write", GitHub: "read", ExternalMutation: "none"},
			}},
		},
		Runner: runner,
		VerifyIssue: func(_ context.Context, issue int) error {
			if issue != 199 {
				t.Fatalf("verified issue = %d, want 199", issue)
			}
			verified++
			return nil
		},
	}

	result, err := provisioner.Provision(context.Background(), ProvisionRequest{
		SkillID: "implement-issue", Destination: destination, Revision: revision, IssueNumber: 199,
	})
	if err != nil {
		t.Fatal(err)
	}
	if verified != 1 || runner.setupCalls != 1 {
		t.Fatalf("issue verification/setup calls = %d/%d, want 1/1", verified, runner.setupCalls)
	}
	if result.Manifest.Profile != ProfileRepositoryWrite ||
		result.Manifest.WorkspaceKind != WorkspaceIssueWorktree ||
		result.Manifest.Branch != "issue/199" ||
		result.Manifest.IssueNumber != 199 {
		t.Fatalf("unexpected issue manifest: %#v", result.Manifest)
	}
}

func TestProvisionWriteProfileFailsWithoutIssueVerification(t *testing.T) {
	root, revision := testRepository(t)
	_, err := (Provisioner{
		Root: root,
		Graph: &graph.Graph{
			SchemaVersion: 1,
			Skills: []graph.Skill{{
				ID:        "implement-issue",
				Authority: graph.Authority{Repository: "write", Git: "write", GitHub: "read", ExternalMutation: "none"},
			}},
		},
		Runner: &provisioningRunner{},
	}).Provision(context.Background(), ProvisionRequest{
		SkillID: "implement-issue", Destination: filepath.Join(t.TempDir(), "worktree"), Revision: revision,
	})
	if err == nil || !strings.Contains(err.Error(), "requires an existing Issue") {
		t.Fatalf("expected Issue requirement, got %v", err)
	}
}

func TestProvisionRejectsWrongAndConcurrentIssueWorktrees(t *testing.T) {
	root, revision := testRepository(t)
	makeProvisioner := func() Provisioner {
		return Provisioner{
			Root: root,
			Graph: &graph.Graph{
				SchemaVersion: 1,
				Skills: []graph.Skill{{
					ID:        "implement-issue",
					Authority: graph.Authority{Repository: "write", Git: "write", GitHub: "read", ExternalMutation: "none"},
				}},
			},
			Runner: &provisioningRunner{},
			VerifyIssue: func(_ context.Context, issue int) error {
				if issue != 199 {
					t.Fatalf("verified issue = %d, want 199", issue)
				}
				return nil
			},
		}
	}

	wrongPath := filepath.Join(t.TempDir(), "wrong-branch")
	addWorktree(t, root, wrongPath, "-b", "unrelated", revision)
	_, err := makeProvisioner().Provision(context.Background(), ProvisionRequest{
		SkillID: "implement-issue", Destination: wrongPath, Revision: revision, IssueNumber: 199,
	})
	if err == nil || !strings.Contains(err.Error(), "destination is owned by branch unrelated") {
		t.Fatalf("wrong branch was not rejected: %v", err)
	}

	ownedPath := filepath.Join(t.TempDir(), "owned-branch")
	addWorktree(t, root, ownedPath, "-b", "issue/199", revision)
	_, err = makeProvisioner().Provision(context.Background(), ProvisionRequest{
		SkillID: "implement-issue", Destination: filepath.Join(t.TempDir(), "concurrent"), Revision: revision, IssueNumber: 199,
	})
	if err == nil || !strings.Contains(err.Error(), "already owned") {
		t.Fatalf("concurrent branch was not rejected: %v", err)
	}
}

func TestProvisionStopsBeforeExecutionWhenSetupFails(t *testing.T) {
	root, revision := testRepository(t)
	runner := &provisioningRunner{setupErr: errors.New("mise unavailable")}
	provisioner := Provisioner{
		Root: root,
		Graph: &graph.Graph{
			SchemaVersion: 1,
			Skills: []graph.Skill{{
				ID:        "plan-issue",
				Authority: graph.Authority{Repository: "read", Git: "read", GitHub: "read", ExternalMutation: "none"},
			}},
		},
		Runner: runner,
	}
	_, err := provisioner.Provision(context.Background(), ProvisionRequest{
		SkillID: "plan-issue", Destination: filepath.Join(t.TempDir(), "snapshot"), Revision: revision,
	})
	if err == nil || !strings.Contains(err.Error(), "setup:all") {
		t.Fatalf("setup failure was not fatal: %v", err)
	}
	if runner.setupCalls != 1 {
		t.Fatalf("setup calls = %d, want 1", runner.setupCalls)
	}
}

func TestCleanupRetainsActiveOrMaterialWorktree(t *testing.T) {
	provisioner, result := provisionIssueEnvironment(t)

	active, err := provisioner.Cleanup(context.Background(), CleanupRequest{
		Provisioned: result, Active: true, ReviewApproved: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if active.Cleanup != CleanupBlockedActive {
		t.Fatalf("active cleanup = %q, want %q", active.Cleanup, CleanupBlockedActive)
	}

	if err := os.WriteFile(filepath.Join(result.WorkspacePath, "README.md"), []byte("material\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	material, err := provisioner.Cleanup(context.Background(), CleanupRequest{
		Provisioned: result, ReviewApproved: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if material.Cleanup != CleanupBlockedMaterial {
		t.Fatalf("material cleanup = %q, want %q", material.Cleanup, CleanupBlockedMaterial)
	}
	if err := os.WriteFile(filepath.Join(result.WorkspacePath, "README.md"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(result.WorkspacePath, "README.md"), []byte("unpushed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runFixtureGit(t, result.WorkspacePath, "add", "README.md")
	runFixtureGit(t, result.WorkspacePath, "commit", "-q", "-m", "unpushed fixture")
	unpushed, err := provisioner.Cleanup(context.Background(), CleanupRequest{
		Provisioned: result, ReviewApproved: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if unpushed.Cleanup != CleanupBlockedMaterial {
		t.Fatalf("unpushed cleanup = %q, want %q", unpushed.Cleanup, CleanupBlockedMaterial)
	}
	runFixtureGit(t, result.WorkspacePath, "reset", "--hard", result.Manifest.RepositoryRevision)
	removed, err := provisioner.Cleanup(context.Background(), CleanupRequest{
		Provisioned: result, ReviewApproved: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if removed.Cleanup != CleanupRemovedReviewed {
		t.Fatalf("clean cleanup = %q, want %q", removed.Cleanup, CleanupRemovedReviewed)
	}
	if _, err := os.Stat(result.WorkspacePath); !os.IsNotExist(err) {
		t.Fatalf("worktree still exists after guarded removal: %v", err)
	}
}

func provisionIssueEnvironment(t *testing.T) (Provisioner, Provisioned) {
	t.Helper()
	root, revision := testRepository(t)
	provisioner := Provisioner{
		Root: root,
		Graph: &graph.Graph{
			SchemaVersion: 1,
			Skills: []graph.Skill{{
				ID:        "implement-issue",
				Authority: graph.Authority{Repository: "write", Git: "write", GitHub: "read", ExternalMutation: "none"},
			}},
		},
		Runner: &provisioningRunner{},
		VerifyIssue: func(_ context.Context, issue int) error {
			if issue != 199 {
				t.Fatalf("verified issue = %d, want 199", issue)
			}
			return nil
		},
	}
	result, err := provisioner.Provision(context.Background(), ProvisionRequest{
		SkillID: "implement-issue", Destination: filepath.Join(t.TempDir(), "issue-worktree"),
		Revision: revision, IssueNumber: 199,
	})
	if err != nil {
		t.Fatal(err)
	}
	return provisioner, result
}

func testRepository(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = fixtureGitEnvironment()
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, output)
		}
		return strings.TrimSpace(string(output))
	}
	run("init", "-q")
	run("config", "user.name", "Test User")
	run("config", "user.email", "test-user")
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README.md")
	run("commit", "-q", "-m", "fixture")
	return root, run("rev-parse", "HEAD")
}

func TestTestRepositoryIgnoresAmbientGitContext(t *testing.T) {
	outer := t.TempDir()
	runFixtureGit(t, outer, "init", "-q")
	runFixtureGit(t, outer, "config", "user.name", "Outer User")
	runFixtureGit(t, outer, "config", "user.email", "outer@example.invalid")
	t.Setenv("GIT_DIR", filepath.Join(outer, ".git"))
	t.Setenv("GIT_WORK_TREE", outer)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(outer, "outer-index"))

	root, _ := testRepository(t)
	if root == outer {
		t.Fatal("test repository reused the ambient work tree")
	}
	if got := runFixtureGitOutput(t, outer, "config", "--get", "user.name"); got != "Outer User" {
		t.Fatalf("ambient user.name changed to %q", got)
	}
	if _, err := exec.Command("git", "-C", outer, "rev-parse", "--verify", "HEAD").Output(); err == nil {
		t.Fatal("fixture commit was created in the ambient repository")
	}
}

func addWorktree(t *testing.T, root, destination string, branchArgs ...string) {
	t.Helper()
	args := append([]string{"worktree", "add"}, append(branchArgs[:len(branchArgs)-1], destination, branchArgs[len(branchArgs)-1])...)
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Env = fixtureGitEnvironment()
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}

func runFixtureGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	_ = runFixtureGitOutput(t, dir, args...)
}

func runFixtureGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = fixtureGitEnvironment()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func fixtureGitEnvironment() []string {
	return append(support.GitEnv(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
}

func fileMode(t *testing.T, path string) os.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return info.Mode().Perm()
}

func restoreWritePermissions(root string) {
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return os.Chmod(path, 0o755)
		}
		return os.Chmod(path, 0o644)
	})
}
