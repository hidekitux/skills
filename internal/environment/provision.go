package environment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hidekitux/skills/internal/graph"
	"github.com/hidekitux/skills/internal/support"
)

type CommandRunner interface {
	Run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error)
}

type OSCommandRunner struct{}

func (OSCommandRunner) Run(ctx context.Context, dir string, env []string, name string, args ...string) (string, error) {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = dir
	if env != nil {
		command.Env = env
	}
	output, err := command.CombinedOutput()
	return string(output), err
}

type IssueVerifier func(context.Context, int) error

type Provisioner struct {
	Root        string
	Graph       *graph.Graph
	Runner      CommandRunner
	VerifyIssue IssueVerifier
}

type ProvisionRequest struct {
	SkillID     string
	Destination string
	Revision    string
	IssueNumber int
}

type Provisioned struct {
	Manifest      Manifest
	WorkspacePath string
	PolicyDir     string
	Environment   []string
}

func (p Provisioner) Provision(ctx context.Context, request ProvisionRequest) (Provisioned, error) {
	if p.Root == "" {
		return Provisioned{}, errors.New("provisioner root is required")
	}
	root, err := filepath.Abs(p.Root)
	if err != nil {
		return Provisioned{}, fmt.Errorf("resolve provisioner root: %w", err)
	}
	graphDocument := p.Graph
	if graphDocument == nil {
		graphDocument, err = graph.Load(root)
		if err != nil {
			return Provisioned{}, err
		}
	}
	skill, ok := graphDocument.Skill(request.SkillID)
	if !ok {
		return Provisioned{}, fmt.Errorf("skill %q is not in the graph", request.SkillID)
	}
	_, permissions, err := Derive(*skill)
	if err != nil {
		return Provisioned{}, err
	}
	if request.Destination == "" {
		return Provisioned{}, errors.New("provision destination is required")
	}
	destination, err := filepath.Abs(request.Destination)
	if err != nil {
		return Provisioned{}, fmt.Errorf("resolve provision destination: %w", err)
	}
	if sameOrWithin(root, destination) {
		return Provisioned{}, errors.New("provision destination must be outside the source repository")
	}
	revision, err := p.resolveRevision(ctx, root, request.Revision)
	if err != nil {
		return Provisioned{}, err
	}
	workspace := WorkspaceDetachedSnapshot
	branch := ""
	if RequiresIssueWorktree(permissions) {
		if request.IssueNumber <= 0 {
			return Provisioned{}, errors.New("repository or Git write authority requires an existing Issue")
		}
		if p.VerifyIssue == nil {
			return Provisioned{}, errors.New("mutating provisioning requires an Issue verifier")
		}
		if err := p.VerifyIssue(ctx, request.IssueNumber); err != nil {
			return Provisioned{}, fmt.Errorf("verify Issue #%d: %w", request.IssueNumber, err)
		}
		workspace = WorkspaceIssueWorktree
		branch = fmt.Sprintf("issue/%d", request.IssueNumber)
	}
	environmentID := makeEnvironmentID(request.SkillID, revision, request.IssueNumber, destination)
	manifest, err := NewManifest(*skill, graphDocument.SchemaVersion, environmentID, revision, request.IssueNumber, branch, workspace)
	if err != nil {
		return Provisioned{}, err
	}
	if workspace == WorkspaceDetachedSnapshot {
		if err := ensureEmptyDestination(destination); err != nil {
			return Provisioned{}, err
		}
		if _, err := p.runGit(ctx, root, "worktree", "add", "--detach", destination, revision); err != nil {
			return Provisioned{}, fmt.Errorf("create detached snapshot: %w", err)
		}
	} else {
		if err := p.provisionIssueWorktree(ctx, root, destination, branch, revision); err != nil {
			return Provisioned{}, err
		}
	}
	if err := p.runSetup(ctx, destination); err != nil {
		return Provisioned{}, err
	}
	manifest.Setup = Setup{Status: SetupSucceeded}
	policyDir := filepath.Join(filepath.Dir(destination), ".skill-environment", environmentID)
	if err := p.writePolicy(policyDir, permissions); err != nil {
		return Provisioned{}, err
	}
	if workspace == WorkspaceDetachedSnapshot {
		if err := makeReadOnly(destination); err != nil {
			return Provisioned{}, fmt.Errorf("make detached snapshot read-only: %w", err)
		}
	}
	manifest.Ownership = Ownership{Status: OwnershipVerified}
	if findings := Validate(manifest); len(findings) > 0 {
		return Provisioned{}, fmt.Errorf("provisioned manifest is invalid: %s", strings.Join(findings, "; "))
	}
	environment := []string{"PATH=" + filepath.Join(policyDir, "bin") + string(os.PathListSeparator) + os.Getenv("PATH")}
	if workspace == WorkspaceDetachedSnapshot {
		environment = append(environment, "GIT_OPTIONAL_LOCKS=0")
	}
	return Provisioned{Manifest: manifest, WorkspacePath: destination, PolicyDir: policyDir, Environment: environment}, nil
}

func (p Provisioner) resolveRevision(ctx context.Context, root, revision string) (string, error) {
	if revision == "" {
		revision = "HEAD"
	}
	output, err := p.runGit(ctx, root, "rev-parse", "--verify", revision+"^{commit}")
	if err != nil {
		return "", fmt.Errorf("resolve revision %q: %w", revision, err)
	}
	resolved := strings.TrimSpace(output)
	if len(resolved) != 40 {
		return "", fmt.Errorf("resolved revision %q is not a full commit SHA", revision)
	}
	return resolved, nil
}

func (p Provisioner) provisionIssueWorktree(ctx context.Context, root, destination, branch, revision string) error {
	worktrees, err := p.listWorktrees(ctx, root)
	if err != nil {
		return err
	}
	for _, worktree := range worktrees {
		if equivalentPath(worktree.Path, destination) && worktree.Branch == branch {
			return nil
		}
		if worktree.Branch == branch && !equivalentPath(worktree.Path, destination) {
			return fmt.Errorf("branch %s is already owned by %s", branch, worktree.Path)
		}
		if equivalentPath(worktree.Path, destination) && worktree.Branch != branch {
			return fmt.Errorf("destination is owned by branch %s, not %s", worktree.Branch, branch)
		}
	}
	if err := ensureEmptyDestination(destination); err != nil {
		return err
	}
	if _, err := p.runGit(ctx, root, "show-ref", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		if _, err := p.runGit(ctx, root, "worktree", "add", destination, branch); err != nil {
			return fmt.Errorf("attach Issue worktree: %w", err)
		}
		return nil
	}
	if _, err := p.runGit(ctx, root, "worktree", "add", "-b", branch, destination, revision); err != nil {
		return fmt.Errorf("create Issue worktree: %w", err)
	}
	return nil
}

type worktreeRecord struct {
	Path   string
	Branch string
}

func (p Provisioner) listWorktrees(ctx context.Context, root string) ([]worktreeRecord, error) {
	output, err := p.runGit(ctx, root, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("inspect worktree ownership: %w", err)
	}
	var records []worktreeRecord
	var current worktreeRecord
	flush := func() {
		if current.Path != "" {
			records = append(records, current)
		}
		current = worktreeRecord{}
	}
	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			current.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch refs/heads/"):
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		case line == "detached HEAD":
			current.Branch = ""
		}
	}
	flush()
	return records, nil
}

func (p Provisioner) runGit(ctx context.Context, dir string, args ...string) (string, error) {
	if p.Runner != nil {
		return p.Runner.Run(ctx, dir, nil, "git", args...)
	}
	return (OSCommandRunner{}).Run(ctx, dir, support.GitEnv(), "git", args...)
}

func (p Provisioner) runSetup(ctx context.Context, dir string) error {
	var output string
	var err error
	if p.Runner != nil {
		output, err = p.Runner.Run(ctx, dir, nil, "mise", "run", "setup:all")
	} else {
		output, err = (OSCommandRunner{}).Run(ctx, dir, support.GitEnv(), "mise", "run", "setup:all")
	}
	if err != nil {
		return fmt.Errorf("run setup:all before execution: %w", err)
	}
	_ = output
	return nil
}

func (p Provisioner) writePolicy(policyDir string, permissions Permissions) error {
	if err := os.MkdirAll(filepath.Join(policyDir, "bin"), 0o755); err != nil {
		return fmt.Errorf("create command policy: %w", err)
	}
	gitPath, err := exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("resolve git for command policy: %w", err)
	}
	ghPath, err := exec.LookPath("gh")
	if err != nil {
		ghPath = ""
	}
	if err := os.WriteFile(filepath.Join(policyDir, "bin", "git"), []byte(gitPolicyScript(gitPath, permissions)), 0o755); err != nil {
		return fmt.Errorf("write git command policy: %w", err)
	}
	if err := os.WriteFile(filepath.Join(policyDir, "bin", "gh"), []byte(ghPolicyScript(ghPath)), 0o755); err != nil {
		return fmt.Errorf("write gh command policy: %w", err)
	}
	return nil
}

func gitPolicyScript(realGit string, permissions Permissions) string {
	localWrite := permissions.Repository == "write" || permissions.Git == "write"
	localCase := "status|diff|log|show|rev-parse|ls-files|cat-file|grep"
	if localWrite {
		localCase += "|add|commit|restore|rm|mv|config|update-index|checkout|switch|merge|rebase|reset|tag"
	}
	return fmt.Sprintf(`#!/bin/sh
set -eu
case "${1-}" in
  %s) export GIT_OPTIONAL_LOCKS="${GIT_OPTIONAL_LOCKS:-0}"; exec %s "$@" ;;
  branch)
    case "${2-}" in
      --show-current|--list|-vv) exec %s "$@" ;;
    esac
    ;;
  worktree)
    [ "${2-}" = "list" ] && exec %s "$@"
    ;;
esac
echo "skill-environment: denied git mutation or unknown operation" >&2
exit 126
`, localCase, shellQuote(realGit), shellQuote(realGit), shellQuote(realGit))
}

func ghPolicyScript(realGH string) string {
	return fmt.Sprintf(`#!/bin/sh
set -eu
if [ -z %s ]; then
	echo "skill-environment: GitHub CLI is unavailable" >&2
	exit 126
fi
deny() {
	echo "skill-environment: denied GitHub mutation or unknown operation" >&2
	exit 126
}
checkAPI() {
	shift
	method=GET
	explicitMethod=0
	hasPayload=0
	while [ "$#" -gt 0 ]; do
		case "$1" in
		-X|--method)
			[ "$#" -ge 2 ] || deny
			method="$2"
			explicitMethod=1
			shift 2
			;;
		--method=*)
			method="${1#--method=}"
			explicitMethod=1
			shift
			;;
		-X?*)
			method="${1#-X}"
			explicitMethod=1
			shift
			;;
		-f|-F|--raw-field|--field|--input)
			hasPayload=1
			[ "$#" -ge 2 ] || deny
			shift 2
			;;
		-f=*|-F=*|-f?*|-F?*|--raw-field=*|--field=*|--input=*)
			hasPayload=1
			shift
			;;
		*)
			shift
			;;
		esac
	done
	case "$method" in
	GET|HEAD) ;;
	*) deny ;;
	esac
	[ "$hasPayload" -eq 0 ] || [ "$explicitMethod" -eq 1 ] || deny
}
case "${1-}" in
	issue|pr|repo)
		case "${2-}" in
		view|list|status|checks) exec %s "$@" ;;
		esac
		;;
	api)
		checkAPI "$@"
		exec %s "$@"
		;;
esac
deny
`, shellQuote(realGH), shellQuote(realGH), shellQuote(realGH))
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func makeReadOnly(root string) error {
	return filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		if info.IsDir() {
			mode = 0o555
		} else {
			mode = (mode & 0o111) | 0o444
		}
		return os.Chmod(path, mode)
	})
}

func ensureEmptyDestination(path string) error {
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(filepath.Dir(path), 0o755)
	}
	if err != nil {
		return fmt.Errorf("inspect provision destination: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("provision destination %q is not a directory", path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("inspect provision destination: %w", err)
	}
	if len(entries) != 0 {
		return fmt.Errorf("provision destination %q is not empty", path)
	}
	return nil
}

func sameOrWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return true
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func makeEnvironmentID(skillID, revision string, issueNumber int, destination string) string {
	hash := sha256.Sum256([]byte(skillID + "\x00" + revision + "\x00" + strconv.Itoa(issueNumber) + "\x00" + destination))
	return "env-" + hex.EncodeToString(hash[:8])
}
