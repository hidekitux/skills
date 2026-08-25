// Package diagnose implements worktree ownership and setup-state reporting,
// ported from scripts/diagnose/diagnose-worktree.py.
package diagnose

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hidekitux/skills/internal/support"
)

// Worktree describes one entry from `git worktree list --porcelain`.
type Worktree struct {
	Path   string
	Branch string
}

// Worktrees parses the porcelain worktree listing into entries.
func Worktrees() ([]Worktree, error) {
	stdout, err := support.GitOutput("worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var entries []Worktree
	var current *Worktree
	for _, line := range append(strings.Split(stdout, "\n"), "") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			if current != nil {
				entries = append(entries, *current)
			}
			current = &Worktree{Path: strings.TrimPrefix(line, "worktree ")}
		case strings.HasPrefix(line, "branch "):
			if current != nil {
				current.Branch = strings.TrimPrefix(line, "branch ")
			}
		case line == "" && current != nil:
			entries = append(entries, *current)
			current = nil
		}
	}
	return entries, nil
}

// SetupState reports whether a worktree's local skill registration stamp
// matches its checked-out revision.
func SetupState(entry Worktree) string {
	if info, err := os.Stat(entry.Path); err != nil || !info.IsDir() {
		return "missing"
	}
	stamp := filepath.Join(entry.Path, ".agents", "worktree-snapshot")
	content, err := os.ReadFile(stamp)
	if os.IsNotExist(err) {
		return "not run"
	}
	if err != nil {
		return "unreadable"
	}
	revision := strings.TrimSpace(string(content))
	head, err := support.GitOutput("--git-dir", entry.Path, "rev-parse", "HEAD")
	if err != nil {
		return "unreadable"
	}
	if revision == strings.TrimSpace(head) {
		return "current"
	}
	return "stale"
}

// IsMerged reports whether branch is an ancestor of base.
func IsMerged(branch, base string) bool {
	_, err := support.GitOutput("merge-base", "--is-ancestor", branch, base)
	return err == nil
}

// DiagnoseWorktree reports worktree branch ownership and setup state,
// returning 0 on success or 2 when Git inspection fails.
func DiagnoseWorktree(branch, base string, out, errOut io.Writer) int {
	bareOutput, err := support.GitOutput("rev-parse", "--is-bare-repository")
	if err != nil {
		fmt.Fprintf(errOut, "%s\n", gitFailureMessage(err))
		return 2
	}
	entries, err := Worktrees()
	if err != nil {
		fmt.Fprintf(errOut, "%s\n", gitFailureMessage(err))
		return 2
	}

	if branch == "" {
		for _, entry := range entries {
			name := entry.Branch
			if name == "" {
				name = "(detached HEAD)"
			}
			fmt.Fprintf(out, "%s: %s (setup %s)\n", entry.Path, name, SetupState(entry))
		}
		if strings.TrimSpace(bareOutput) == "true" {
			fmt.Fprintln(out, "warning: this directory is a bare Git repository; run development commands from a listed worktree.")
		}
		return 0
	}

	reference := "refs/heads/" + branch
	var owners []Worktree
	for _, entry := range entries {
		if entry.Branch == reference {
			owners = append(owners, entry)
		}
	}
	if len(owners) == 0 {
		fmt.Fprintf(out, "Branch %s is not checked out by a registered worktree.\n", branch)
		return 0
	}

	owner := owners[0]
	mergedState := "not known to be merged"
	if IsMerged(branch, base) {
		mergedState = "merged"
	}
	fmt.Fprintf(out, "Branch %s is checked out at %s (setup %s, %s into %s).\n",
		branch, owner.Path, SetupState(owner), mergedState, base)
	if branch == "main" {
		fmt.Fprintln(out, "The primary main worktree must remain in place; do not use git worktree remove or --force as a remediation.")
		fmt.Fprintln(out, "For an additional read-only main snapshot, use: git worktree add --detach <path> origin/main")
		fmt.Fprintln(out, "For changes, create an Issue branch instead: git worktree add -b issue/<number> <path> origin/main")
		return 0
	}
	fmt.Fprintf(out, "Do not remove the worktree automatically. Inspect its status, then run `git worktree remove %q` only when it is no longer active.\n", owner.Path)
	if SetupState(owner) != "current" {
		fmt.Fprintf(out, "Setup for %s did not finish. Run 'mise run setup' there before continuing.\n", owner.Path)
	}
	return 0
}

func gitFailureMessage(err error) string {
	if err != nil {
		if exitErr, ok := err.(interface{ Stderr() []byte }); ok && len(exitErr.Stderr()) > 0 {
			return strings.TrimSpace(string(exitErr.Stderr()))
		}
	}
	return "error: unable to inspect Git worktrees"
}
