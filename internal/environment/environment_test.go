package environment

import (
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/graph"
)

func TestDeriveProfilesFromGraphAuthority(t *testing.T) {
	tests := []struct {
		name      string
		authority graph.Authority
		want      Profile
		wantIssue bool
	}{
		{name: "read only", authority: graph.Authority{Repository: "read", Git: "read", GitHub: "read", ExternalMutation: "none"}, want: ProfileReadOnly},
		{name: "repository write", authority: graph.Authority{Repository: "write", Git: "write", GitHub: "read", ExternalMutation: "none"}, want: ProfileRepositoryWrite, wantIssue: true},
		{name: "external issue", authority: graph.Authority{Repository: "read", Git: "none", GitHub: "write", ExternalMutation: "issue"}, want: ProfileExternalMutation},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile, permissions, err := Derive(graph.Skill{ID: "test-skill", Authority: tt.authority})
			if err != nil {
				t.Fatal(err)
			}
			if profile != tt.want {
				t.Fatalf("profile = %q, want %q", profile, tt.want)
			}
			if RequiresIssueWorktree(permissions) != tt.wantIssue {
				t.Fatalf("RequiresIssueWorktree = %v, want %v", RequiresIssueWorktree(permissions), tt.wantIssue)
			}
		})
	}
}

func TestNewManifestRequiresMatchingIssueWorktreeForWrites(t *testing.T) {
	skill := graph.Skill{ID: "implement-issue", Authority: graph.Authority{Repository: "write", Git: "write", GitHub: "read", ExternalMutation: "none"}}
	manifest, err := NewManifest(skill, 1, "env-1", strings.Repeat("a", 40), 199, "issue/199", WorkspaceIssueWorktree)
	if err != nil {
		t.Fatal(err)
	}
	if findings := Validate(manifest); len(findings) != 0 {
		t.Fatalf("unexpected findings: %v", findings)
	}
	manifest.Branch = "feature/unrelated"
	if findings := Validate(manifest); !contains(findings, "branch must be an issue/<number> branch") {
		t.Fatalf("expected branch finding, got %v", findings)
	}
}

func TestReadOnlyManifestRejectsMutationAuthorityAndBranch(t *testing.T) {
	manifest := Manifest{
		SchemaVersion:      CurrentSchemaVersion,
		EnvironmentID:      "env-1",
		SkillID:            "plan-issue",
		GraphVersion:       1,
		Profile:            ProfileReadOnly,
		WorkspaceKind:      WorkspaceDetachedSnapshot,
		RepositoryRevision: strings.Repeat("b", 40),
		Branch:             "issue/199",
		Permissions:        Permissions{Repository: "read", Git: "read", GitHub: "read", ExternalMutation: "issue"},
		Setup:              Setup{Status: SetupPending},
		Ownership:          Ownership{Status: OwnershipPending},
		Cleanup:            CleanupNotRequested,
	}
	findings := Validate(manifest)
	if !contains(findings, "detached_snapshot must not name a branch") || !contains(findings, "read_only profile declares mutation authority") {
		t.Fatalf("expected read-only boundary findings, got %v", findings)
	}
}

func TestInvalidAuthorityFailsClosed(t *testing.T) {
	_, _, err := Derive(graph.Skill{ID: "unknown", Authority: graph.Authority{Repository: "write", Git: "write", GitHub: "read", ExternalMutation: "remote-admin"}})
	if err == nil || !strings.Contains(err.Error(), "invalid authority") {
		t.Fatalf("expected invalid authority error, got %v", err)
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
