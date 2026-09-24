package environment

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/hidekitux/skills/internal/support"
)

const worktrunkCommand = "wt"

// WorktreeState is the privacy-safe, normalized state used by the environment
// ownership and cleanup gates. Path is used internally and never enters the
// environment manifest.
type WorktreeState struct {
	Path            string
	Branch          string
	Detached        bool
	Prunable        bool
	PrunableReason  string
	Locked          bool
	LockedReason    string
	Conflicted      bool
	Operation       string
	BranchMismatch  bool
	DuplicateBranch bool
}

// WorktreeProvider owns the lifecycle operations that differ between native
// Git and the local worktrunk interface.
type WorktreeProvider interface {
	Create(ctx context.Context, root, destination, branch, revision string) (WorktreeState, error)
	List(ctx context.Context, root string) ([]WorktreeState, error)
	Remove(ctx context.Context, root string, worktree WorktreeState) error
}

// NewLocalWorktreeProvider selects worktrunk when it is installed and falls
// back to native Git when the local convenience tool is unavailable. CI and
// other non-local callers should leave Provisioner.Worktree nil, which keeps
// the native provider as the deterministic default.
func NewLocalWorktreeProvider(runner CommandRunner) WorktreeProvider {
	if command, err := exec.LookPath(worktrunkCommand); err == nil {
		return WorktrunkProvider{Runner: runner, Command: command}
	}
	return NativeGitWorktreeProvider{Runner: runner}
}

// NativeGitWorktreeProvider uses Git's machine-readable worktree porcelain.
type NativeGitWorktreeProvider struct {
	Runner CommandRunner
}

func (p NativeGitWorktreeProvider) Create(ctx context.Context, root, destination, branch, revision string) (WorktreeState, error) {
	worktrees, err := p.List(ctx, root)
	if err != nil {
		return WorktreeState{}, err
	}
	if existing, found, err := resolveWorktree(worktrees, destination, branch, false); err != nil {
		return WorktreeState{}, err
	} else if found {
		return existing, nil
	}
	if err := ensureEmptyDestination(destination); err != nil {
		return WorktreeState{}, err
	}
	if _, err := p.runGit(ctx, root, "show-ref", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		if _, err := p.runGit(ctx, root, "-c", "core.hooksPath="+os.DevNull, "worktree", "add", destination, branch); err != nil {
			return WorktreeState{}, fmt.Errorf("attach Issue worktree: %w", err)
		}
	} else if _, err := p.runGit(ctx, root, "-c", "core.hooksPath="+os.DevNull, "worktree", "add", "-b", branch, destination, revision); err != nil {
		return WorktreeState{}, fmt.Errorf("create Issue worktree: %w", err)
	}
	return WorktreeState{Path: destination, Branch: branch}, nil
}

func (p NativeGitWorktreeProvider) List(ctx context.Context, root string) ([]WorktreeState, error) {
	output, err := p.runGit(ctx, root, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, fmt.Errorf("inspect worktree ownership: %w", err)
	}
	records, err := parseNativeWorktreeList(output)
	if err != nil {
		return nil, err
	}
	for index := range records {
		if records[index].Prunable {
			continue
		}
		status, err := p.runGit(ctx, records[index].Path, "status", "--porcelain=v2", "--branch")
		if err != nil {
			return nil, fmt.Errorf("inspect native worktree state: %w", err)
		}
		for _, line := range strings.Split(status, "\n") {
			if strings.HasPrefix(line, "u ") {
				records[index].Conflicted = true
				break
			}
		}
	}
	return records, nil
}

func (p NativeGitWorktreeProvider) Remove(ctx context.Context, root string, worktree WorktreeState) error {
	if worktree.Path == "" {
		return errors.New("native Git removal requires a worktree path")
	}
	if _, err := p.runGit(ctx, root, "-c", "core.hooksPath="+os.DevNull, "worktree", "remove", worktree.Path); err != nil {
		return fmt.Errorf("remove clean worktree without force: %w", err)
	}
	return nil
}

func (p NativeGitWorktreeProvider) runGit(ctx context.Context, dir string, args ...string) (string, error) {
	if p.Runner != nil {
		return p.Runner.Run(ctx, dir, nil, "git", args...)
	}
	return (OSCommandRunner{}).Run(ctx, dir, support.GitEnv(), "git", args...)
}

// WorktrunkProvider invokes worktrunk's structured automation interface. It
// never asks worktrunk to change the caller's directory or run hooks during a
// controlled Provisioner operation.
type WorktrunkProvider struct {
	Runner  CommandRunner
	Command string
}

func (p WorktrunkProvider) command() string {
	if p.Command != "" {
		return p.Command
	}
	return worktrunkCommand
}

func (p WorktrunkProvider) Create(ctx context.Context, root, destination, branch, revision string) (WorktreeState, error) {
	worktrees, err := p.List(ctx, root)
	if err != nil {
		return WorktreeState{}, err
	}
	if existing, found, err := resolveWorktree(worktrees, destination, branch, true); err != nil {
		return WorktreeState{}, err
	} else if found {
		return existing, nil
	}

	args := []string{"-C", root, "switch", "--no-cd", "--no-hooks", "--format=json", "--yes"}
	if _, err := p.runGit(ctx, root, "show-ref", "--verify", "--quiet", "refs/heads/"+branch); err != nil {
		args = append(args, "--create", branch, "--base", revision)
	} else {
		args = append(args, branch)
	}
	output, err := p.run(ctx, root, args...)
	if err != nil {
		return WorktreeState{}, fmt.Errorf("create or switch worktrunk worktree: %w", err)
	}
	created, err := parseWorktrunkSwitch(output)
	if err != nil {
		return WorktreeState{}, err
	}
	if created.Branch != branch || created.Path == "" {
		return WorktreeState{}, fmt.Errorf("worktrunk returned unexpected worktree identity for %s", branch)
	}
	return created, nil
}

func (p WorktrunkProvider) List(ctx context.Context, root string) ([]WorktreeState, error) {
	output, err := p.run(ctx, root, "-C", root, "list", "--format=json")
	if err != nil {
		return nil, fmt.Errorf("inspect worktrunk worktrees: %w", err)
	}
	return parseWorktrunkList(output)
}

func (p WorktrunkProvider) Remove(ctx context.Context, root string, worktree WorktreeState) error {
	target := worktree.Branch
	if target == "" {
		target = worktree.Path
	}
	if target == "" {
		return errors.New("worktrunk removal requires a branch or worktree path")
	}
	output, err := p.run(ctx, root, "-C", root, "remove", "--foreground", "--no-hooks", "--no-delete-branch", "--format=json", "--yes", target)
	if err != nil {
		return fmt.Errorf("remove worktrunk worktree without force: %w", err)
	}
	result, err := parseWorktrunkRemoval(output)
	if err != nil {
		return err
	}
	if result.BranchOutcome != "" && result.BranchOutcome != "not_attempted" && result.BranchOutcome != "deleted" {
		return fmt.Errorf("worktrunk retained worktree branch with outcome %q", result.BranchOutcome)
	}
	return nil
}

func (p WorktrunkProvider) run(ctx context.Context, dir string, args ...string) (string, error) {
	if p.Runner != nil {
		return p.Runner.Run(ctx, dir, nil, p.command(), args...)
	}
	return (OSCommandRunner{}).Run(ctx, dir, support.GitEnv(), p.command(), args...)
}

func (p WorktrunkProvider) runGit(ctx context.Context, dir string, args ...string) (string, error) {
	if p.Runner != nil {
		return p.Runner.Run(ctx, dir, nil, "git", args...)
	}
	return (OSCommandRunner{}).Run(ctx, dir, support.GitEnv(), "git", args...)
}

func resolveWorktree(worktrees []WorktreeState, destination, branch string, adoptBranchPath bool) (WorktreeState, bool, error) {
	for _, worktree := range worktrees {
		if worktree.Branch == branch {
			if adoptBranchPath || destination == "" || equivalentPath(worktree.Path, destination) {
				return worktree, true, nil
			}
			return WorktreeState{}, false, fmt.Errorf("branch %s is already owned by another worktree", branch)
		}
		if destination != "" && equivalentPath(worktree.Path, destination) {
			if worktree.Branch != branch {
				return WorktreeState{}, false, fmt.Errorf("destination is owned by branch %s, not %s", worktree.Branch, branch)
			}
			return worktree, true, nil
		}
	}
	return WorktreeState{}, false, nil
}

func parseNativeWorktreeList(output string) ([]WorktreeState, error) {
	var records []WorktreeState
	var current *WorktreeState
	flush := func() {
		if current != nil && current.Path != "" {
			records = append(records, *current)
		}
		current = nil
	}
	for _, line := range strings.Split(output, "\n") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			current = &WorktreeState{Path: strings.TrimPrefix(line, "worktree ")}
		case current == nil:
			if strings.TrimSpace(line) != "" {
				return nil, fmt.Errorf("parse native worktree state: record starts before worktree path")
			}
		case strings.HasPrefix(line, "branch refs/heads/"):
			current.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		case line == "detached HEAD":
			current.Detached = true
		case strings.HasPrefix(line, "prunable"):
			current.Prunable = true
			current.PrunableReason = strings.TrimSpace(strings.TrimPrefix(line, "prunable"))
		}
	}
	flush()
	return records, nil
}

type worktrunkItem struct {
	Branch         *string `json:"branch"`
	Path           string  `json:"path"`
	OperationState string  `json:"operation_state"`
	Worktree       *struct {
		Path     string `json:"path"`
		Detached bool   `json:"detached"`
		State    string `json:"state"`
		Reason   string `json:"reason"`
		Prunable *struct {
			Reason string `json:"reason"`
		} `json:"prunable"`
		Locked *struct {
			Reason string `json:"reason"`
		} `json:"locked"`
		Operation       *string `json:"operation"`
		BranchMismatch  bool    `json:"branch_mismatch"`
		DuplicateBranch bool    `json:"duplicate_branch"`
		Changes         *struct {
			Conflicted bool `json:"conflicted"`
		} `json:"changes"`
	} `json:"worktree"`
}

type worktrunkEnvelope struct {
	Schema int             `json:"schema"`
	Items  []worktrunkItem `json:"items"`
}

func parseWorktrunkList(output string) ([]WorktreeState, error) {
	var items []worktrunkItem
	trimmed := strings.TrimSpace(output)
	if trimmed == "" {
		return nil, errors.New("worktrunk returned empty structured output")
	}
	if strings.HasPrefix(trimmed, "[") {
		if err := json.Unmarshal([]byte(trimmed), &items); err != nil {
			return nil, fmt.Errorf("parse worktrunk list: %w", err)
		}
	} else {
		var envelope worktrunkEnvelope
		if err := json.Unmarshal([]byte(trimmed), &envelope); err != nil {
			return nil, fmt.Errorf("parse worktrunk list: %w", err)
		}
		if envelope.Schema != 2 {
			return nil, fmt.Errorf("unsupported worktrunk list schema %d", envelope.Schema)
		}
		items = envelope.Items
	}
	return normalizeWorktrunkItems(items)
}

func normalizeWorktrunkItems(items []worktrunkItem) ([]WorktreeState, error) {
	records := make([]WorktreeState, 0, len(items))
	for index, item := range items {
		path := item.Path
		if item.Worktree != nil && item.Worktree.Path != "" {
			path = item.Worktree.Path
		}
		if path == "" {
			return nil, fmt.Errorf("worktrunk item %d has no worktree path", index)
		}
		record := WorktreeState{Path: path}
		if item.Branch != nil {
			record.Branch = *item.Branch
		}
		if item.Worktree != nil {
			record.Detached = item.Worktree.Detached
			record.BranchMismatch = item.Worktree.BranchMismatch
			record.DuplicateBranch = item.Worktree.DuplicateBranch
		}
		if item.OperationState != "" && item.OperationState != "conflicts" {
			record.Operation = item.OperationState
		}
		if item.OperationState == "conflicts" {
			record.Conflicted = true
		}
		if item.Worktree != nil {
			switch item.Worktree.State {
			case "prunable":
				record.Prunable = true
				record.PrunableReason = item.Worktree.Reason
			case "locked":
				record.Locked = true
				record.LockedReason = item.Worktree.Reason
			case "duplicate_branch":
				record.DuplicateBranch = true
			case "branch_mismatch":
				record.BranchMismatch = true
			case "conflicts":
				record.Conflicted = true
			}
		}
		if item.Worktree.Prunable != nil {
			record.Prunable = true
			record.PrunableReason = item.Worktree.Prunable.Reason
		}
		if item.Worktree.Locked != nil {
			record.Locked = true
			record.LockedReason = item.Worktree.Locked.Reason
		}
		if item.Worktree.Operation != nil {
			record.Operation = *item.Worktree.Operation
		}
		if item.Worktree.Changes != nil {
			record.Conflicted = item.Worktree.Changes.Conflicted
		}
		records = append(records, record)
	}
	return records, nil
}

type worktrunkSwitchResult struct {
	Action        string `json:"action"`
	Branch        string `json:"branch"`
	Path          string `json:"path"`
	CreatedBranch bool   `json:"created_branch"`
}

func parseWorktrunkSwitch(output string) (WorktreeState, error) {
	var result worktrunkSwitchResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		return WorktreeState{}, fmt.Errorf("parse worktrunk switch result: %w", err)
	}
	return WorktreeState{Path: result.Path, Branch: result.Branch}, nil
}

type worktrunkRemovalResult struct {
	BranchOutcome string `json:"branch_outcome"`
}

func parseWorktrunkRemoval(output string) (worktrunkRemovalResult, error) {
	var result worktrunkRemovalResult
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &result); err != nil {
		return result, fmt.Errorf("parse worktrunk removal result: %w", err)
	}
	return result, nil
}
