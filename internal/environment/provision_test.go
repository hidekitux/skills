package environment

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/graph"
)

type provisioningRunner struct {
	setupCalls int
}

func (r *provisioningRunner) Run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	if name == "mise" {
		r.setupCalls++
		return "", nil
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

func testRepository(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	run := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null")
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
