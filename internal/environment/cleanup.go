package environment

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CleanupRequest describes a guarded cleanup decision. Active is supplied by
// the host because only the host can know whether a running skill still owns
// the environment.
type CleanupRequest struct {
	Provisioned    Provisioned
	Active         bool
	ReviewApproved bool
}

// Cleanup inspects an environment before removal. It never uses a force
// removal and returns the updated privacy-safe manifest for every guarded
// disposition.
func (p Provisioner) Cleanup(ctx context.Context, request CleanupRequest) (Manifest, error) {
	manifest := request.Provisioned.Manifest
	if manifest.EnvironmentID == "" || request.Provisioned.WorkspacePath == "" {
		return manifest, errors.New("cleanup requires a provisioned environment")
	}
	if request.Active {
		manifest.Cleanup = CleanupBlockedActive
		return manifest, nil
	}
	root, err := filepath.Abs(p.Root)
	if err != nil {
		return manifest, fmt.Errorf("resolve cleanup root: %w", err)
	}
	workspace, err := filepath.Abs(request.Provisioned.WorkspacePath)
	if err != nil {
		return manifest, fmt.Errorf("resolve cleanup workspace: %w", err)
	}
	worktrees, err := p.listWorktrees(ctx, root)
	if err != nil {
		return manifest, err
	}
	owned := false
	for _, worktree := range worktrees {
		if !equivalentPath(worktree.Path, workspace) {
			continue
		}
		owned = true
		if worktree.Branch != manifest.Branch {
			manifest.Ownership = Ownership{Status: OwnershipConcurrent}
			manifest.Cleanup = CleanupBlockedActive
			return manifest, nil
		}
		break
	}
	if !owned {
		manifest.Ownership = Ownership{Status: OwnershipFailed, Diagnostic: "worktree-not-registered"}
		manifest.Cleanup = CleanupBlockedActive
		return manifest, nil
	}
	if material, err := p.hasMaterialChanges(ctx, workspace, manifest.RepositoryRevision); err != nil {
		return manifest, err
	} else if material {
		manifest.Cleanup = CleanupBlockedMaterial
		return manifest, nil
	}
	if !request.ReviewApproved {
		manifest.Cleanup = CleanupRetained
		return manifest, nil
	}
	if _, err := p.runGit(ctx, root, "worktree", "remove", workspace); err != nil {
		manifest.Cleanup = CleanupBlockedMaterial
		return manifest, fmt.Errorf("remove clean worktree without force: %w", err)
	}
	if err := removePolicyDirectory(request.Provisioned.PolicyDir, manifest.EnvironmentID, workspace); err != nil {
		manifest.Cleanup = CleanupBlockedMaterial
		return manifest, err
	}
	manifest.Cleanup = CleanupRemovedReviewed
	return manifest, nil
}

func equivalentPath(first, second string) bool {
	return canonicalPath(first) == canonicalPath(second)
}

func canonicalPath(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		return filepath.Clean(path)
	}
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		return filepath.Clean(resolved)
	}
	return filepath.Clean(abs)
}

func (p Provisioner) hasMaterialChanges(ctx context.Context, workspace, baseline string) (bool, error) {
	status, err := p.runGit(ctx, workspace, "status", "--porcelain=v1", "--untracked-files=all")
	if err != nil {
		return false, fmt.Errorf("inspect worktree changes: %w", err)
	}
	if strings.TrimSpace(status) != "" {
		return true, nil
	}
	upstream, err := p.runGit(ctx, workspace, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		head, headErr := p.runGit(ctx, workspace, "rev-parse", "HEAD")
		if headErr != nil {
			return false, fmt.Errorf("inspect worktree head: %w", headErr)
		}
		return strings.TrimSpace(head) != baseline, nil
	}
	counts, err := p.runGit(ctx, workspace, "rev-list", "--left-right", "--count", "HEAD..."+strings.TrimSpace(upstream))
	if err != nil {
		return false, fmt.Errorf("inspect worktree divergence: %w", err)
	}
	parts := strings.Fields(counts)
	if len(parts) != 2 {
		return false, fmt.Errorf("inspect worktree divergence: unexpected count %q", strings.TrimSpace(counts))
	}
	ahead, err := strconv.Atoi(parts[0])
	if err != nil {
		return false, fmt.Errorf("inspect worktree divergence: %w", err)
	}
	behind, err := strconv.Atoi(parts[1])
	if err != nil {
		return false, fmt.Errorf("inspect worktree divergence: %w", err)
	}
	return ahead != 0 || behind != 0, nil
}

func removePolicyDirectory(policyDir, environmentID, workspace string) error {
	if policyDir == "" {
		return nil
	}
	expectedParent := filepath.Join(filepath.Dir(workspace), ".skill-environment")
	cleanPolicy := filepath.Clean(policyDir)
	if filepath.Dir(cleanPolicy) != expectedParent || filepath.Base(cleanPolicy) != environmentID {
		return errors.New("cleanup policy directory is outside the provisioned environment")
	}
	if err := os.RemoveAll(cleanPolicy); err != nil {
		return fmt.Errorf("remove command policy: %w", err)
	}
	return nil
}
