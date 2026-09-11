// Package environment defines the safe, machine-readable contract for one
// provisioned skill execution environment.
package environment

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hidekitux/skills/internal/graph"
)

const (
	// CurrentSchemaVersion is the version of the environment manifest.
	CurrentSchemaVersion = 1

	ProfileReadOnly         Profile = "read_only"
	ProfileRepositoryWrite  Profile = "repository_write"
	ProfileExternalMutation Profile = "external_mutation"

	WorkspaceDetachedSnapshot WorkspaceKind = "detached_snapshot"
	WorkspaceIssueWorktree    WorkspaceKind = "issue_worktree"

	SetupPending   SetupStatus = "pending"
	SetupSucceeded SetupStatus = "succeeded"
	SetupFailed    SetupStatus = "failed"
	SetupMissing   SetupStatus = "missing"

	OwnershipPending    OwnershipStatus = "pending"
	OwnershipVerified   OwnershipStatus = "verified"
	OwnershipFailed     OwnershipStatus = "failed"
	OwnershipConcurrent OwnershipStatus = "concurrent"

	CleanupNotRequested    CleanupDisposition = "not_requested"
	CleanupRetained        CleanupDisposition = "retained"
	CleanupRemovedReviewed CleanupDisposition = "removed_after_review"
	CleanupBlockedMaterial CleanupDisposition = "blocked_material"
	CleanupBlockedActive   CleanupDisposition = "blocked_active"
)

// Profile is the execution boundary derived from a skill's declared graph
// authority. It describes a boundary; it never grants authority.
type Profile string

// WorkspaceKind identifies the safe materialization used by a profile.
type WorkspaceKind string

// SetupStatus records whether repository setup completed before execution.
type SetupStatus string

// OwnershipStatus records whether the expected branch owns the workspace.
type OwnershipStatus string

// CleanupDisposition records the result of guarded cleanup inspection.
type CleanupDisposition string

// Permissions is the authority projection copied from the skill graph. The
// external mutation value is descriptive and does not grant credentials.
type Permissions struct {
	Repository       string `json:"repository"`
	Git              string `json:"git"`
	GitHub           string `json:"github"`
	ExternalMutation string `json:"external_mutation"`
}

// Setup records the setup gate result without persisting command output.
type Setup struct {
	Status     SetupStatus `json:"status"`
	Diagnostic string      `json:"diagnostic,omitempty"`
}

// Ownership records the branch ownership gate result without persisting paths.
type Ownership struct {
	Status     OwnershipStatus `json:"status"`
	Diagnostic string          `json:"diagnostic,omitempty"`
}

// Manifest is the privacy-safe identity and outcome record for one provisioned
// environment. Absolute paths and command output are intentionally absent.
type Manifest struct {
	SchemaVersion      int                `json:"schema_version"`
	EnvironmentID      string             `json:"environment_id"`
	SkillID            string             `json:"skill_id"`
	GraphVersion       int                `json:"graph_version"`
	Profile            Profile            `json:"profile"`
	WorkspaceKind      WorkspaceKind      `json:"workspace_kind"`
	RepositoryRevision string             `json:"repository_revision"`
	Branch             string             `json:"branch,omitempty"`
	IssueNumber        int                `json:"issue_number,omitempty"`
	Permissions        Permissions        `json:"permissions"`
	Setup              Setup              `json:"setup"`
	Ownership          Ownership          `json:"ownership"`
	Cleanup            CleanupDisposition `json:"cleanup"`
}

var (
	identifierPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._/:-]*$`)
	shaPattern        = regexp.MustCompile(`^[0-9a-f]{40}$`)
	issueBranch       = regexp.MustCompile(`^issue/[1-9][0-9]*$`)
)

// Derive projects graph authority into a profile and safe manifest
// permissions. It rejects invalid authority instead of guessing a boundary.
func Derive(skill graph.Skill) (Profile, Permissions, error) {
	permissions := Permissions{
		Repository:       skill.Authority.Repository,
		Git:              skill.Authority.Git,
		GitHub:           skill.Authority.GitHub,
		ExternalMutation: skill.Authority.ExternalMutation,
	}
	if findings := validatePermissions(permissions); len(findings) > 0 {
		return "", Permissions{}, fmt.Errorf("skill %q has invalid authority: %s", skill.ID, strings.Join(findings, "; "))
	}
	if permissions.ExternalMutation != "none" {
		return ProfileExternalMutation, permissions, nil
	}
	if permissions.Repository == "write" || permissions.Git == "write" {
		return ProfileRepositoryWrite, permissions, nil
	}
	return ProfileReadOnly, permissions, nil
}

// RequiresIssueWorktree reports whether the selected authority can write the
// repository or Git state and therefore requires a governed Issue branch.
func RequiresIssueWorktree(permissions Permissions) bool {
	return permissions.Repository == "write" || permissions.Git == "write"
}

// NewManifest creates a pending manifest from the graph-derived authority.
// Callers must run Validate before exposing it to a host or trace writer.
func NewManifest(skill graph.Skill, graphVersion int, environmentID, revision string, issueNumber int, branch string, workspace WorkspaceKind) (Manifest, error) {
	profile, permissions, err := Derive(skill)
	if err != nil {
		return Manifest{}, err
	}
	manifest := Manifest{
		SchemaVersion:      CurrentSchemaVersion,
		EnvironmentID:      environmentID,
		SkillID:            skill.ID,
		GraphVersion:       graphVersion,
		Profile:            profile,
		WorkspaceKind:      workspace,
		RepositoryRevision: revision,
		Branch:             branch,
		IssueNumber:        issueNumber,
		Permissions:        permissions,
		Setup:              Setup{Status: SetupPending},
		Ownership:          Ownership{Status: OwnershipPending},
		Cleanup:            CleanupNotRequested,
	}
	return manifest, nil
}

// Validate returns deterministic findings for a manifest.
func Validate(manifest Manifest) []string {
	findings := []string{}
	if manifest.SchemaVersion != CurrentSchemaVersion {
		findings = append(findings, fmt.Sprintf("schema_version %d is unsupported", manifest.SchemaVersion))
	}
	for name, value := range map[string]string{
		"environment_id": manifest.EnvironmentID,
		"skill_id":       manifest.SkillID,
	} {
		if !identifierPattern.MatchString(value) {
			findings = append(findings, name+" is missing or invalid")
		}
	}
	if manifest.GraphVersion < 1 {
		findings = append(findings, "graph_version must be positive")
	}
	if !shaPattern.MatchString(manifest.RepositoryRevision) {
		findings = append(findings, "repository_revision must be a lowercase 40-character commit SHA")
	}
	findings = append(findings, validatePermissions(manifest.Permissions)...)
	if !oneOfProfile(manifest.Profile) {
		findings = append(findings, "profile is invalid")
	}
	if !oneOfWorkspace(manifest.WorkspaceKind) {
		findings = append(findings, "workspace_kind is invalid")
	}
	if manifest.IssueNumber < 0 {
		findings = append(findings, "issue_number must not be negative")
	}
	if manifest.Branch != "" && !issueBranch.MatchString(manifest.Branch) {
		findings = append(findings, "branch must be an issue/<number> branch")
	}
	if RequiresIssueWorktree(manifest.Permissions) {
		if manifest.IssueNumber <= 0 {
			findings = append(findings, "write authority requires a positive issue_number")
		}
		if manifest.WorkspaceKind != WorkspaceIssueWorktree {
			findings = append(findings, "write authority requires an issue_worktree")
		}
		if manifest.Branch != fmt.Sprintf("issue/%d", manifest.IssueNumber) {
			findings = append(findings, "write authority requires the matching issue branch")
		}
	}
	if manifest.WorkspaceKind == WorkspaceIssueWorktree && manifest.IssueNumber <= 0 {
		findings = append(findings, "issue_worktree requires a positive issue_number")
	}
	if manifest.WorkspaceKind == WorkspaceDetachedSnapshot && manifest.Branch != "" {
		findings = append(findings, "detached_snapshot must not name a branch")
	}
	if !oneOfSetup(manifest.Setup.Status) {
		findings = append(findings, "setup.status is invalid")
	}
	if manifest.Setup.Diagnostic != "" && !identifierPattern.MatchString(manifest.Setup.Diagnostic) {
		findings = append(findings, "setup.diagnostic is invalid")
	}
	if !oneOfOwnership(manifest.Ownership.Status) {
		findings = append(findings, "ownership.status is invalid")
	}
	if manifest.Ownership.Diagnostic != "" && !identifierPattern.MatchString(manifest.Ownership.Diagnostic) {
		findings = append(findings, "ownership.diagnostic is invalid")
	}
	if !oneOfCleanup(manifest.Cleanup) {
		findings = append(findings, "cleanup is invalid")
	}
	if manifest.Profile == ProfileReadOnly && (manifest.Permissions.Repository == "write" || manifest.Permissions.Git == "write" || manifest.Permissions.GitHub == "write" || manifest.Permissions.ExternalMutation != "none") {
		findings = append(findings, "read_only profile declares mutation authority")
	}
	return findings
}

func validatePermissions(permissions Permissions) []string {
	findings := []string{}
	if !oneOf(permissions.Repository, "none", "read", "write") {
		findings = append(findings, "permissions.repository is invalid")
	}
	if !oneOf(permissions.Git, "none", "read", "write") {
		findings = append(findings, "permissions.git is invalid")
	}
	if !oneOf(permissions.GitHub, "none", "read", "write") {
		findings = append(findings, "permissions.github is invalid")
	}
	if !oneOf(permissions.ExternalMutation, "none", "issue", "pull_request", "repository_configuration") {
		findings = append(findings, "permissions.external_mutation is invalid")
	}
	return findings
}

func oneOfProfile(value Profile) bool {
	return value == ProfileReadOnly || value == ProfileRepositoryWrite || value == ProfileExternalMutation
}

func oneOfWorkspace(value WorkspaceKind) bool {
	return value == WorkspaceDetachedSnapshot || value == WorkspaceIssueWorktree
}

func oneOfSetup(value SetupStatus) bool {
	return value == SetupPending || value == SetupSucceeded || value == SetupFailed || value == SetupMissing
}

func oneOfOwnership(value OwnershipStatus) bool {
	return value == OwnershipPending || value == OwnershipVerified || value == OwnershipFailed || value == OwnershipConcurrent
}

func oneOfCleanup(value CleanupDisposition) bool {
	return value == CleanupNotRequested || value == CleanupRetained || value == CleanupRemovedReviewed || value == CleanupBlockedMaterial || value == CleanupBlockedActive
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
