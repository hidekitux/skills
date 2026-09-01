package check

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const worktreeDoc = "docs/worktrees.md"

// worktreeDocRequirements are the fragments that make the documented worktree
// workflow usable and safe. The retired diagnose:worktree task was removed in
// favor of an external tool, so the replacement document has to keep naming the
// reviewed version, the supported commands, the forced-removal prohibition, and
// the native fallback rather than degrading into a bare tool mention.
var worktreeDocRequirements = []struct {
	fragment string
	reason   string
}{
	{"## Decision", "a decision section recording the selected tool"},
	{"## Workflow", "a workflow section with the supported commands"},
	{"Reviewed version", "the reviewed version the workflow was verified against"},
	{"License", "the license reviewed for the selected tool"},
	{"wt switch", "the command that creates or locates an Issue worktree"},
	{"wt list", "the command that reports which worktree owns a branch"},
	{"wt remove", "the command that removes an inspected, inactive worktree"},
	{"wt remove --force", "the prohibition on forced removal of a dirty worktree"},
	{"wt remove -D", "the prohibition on deleting an unmerged branch"},
	{"git worktree", "the native fallback for when the tool is unavailable"},
}

// CheckWorktreeDocs requires the replacement worktree workflow document to
// exist, to be referenced from README.md, and to keep the content that makes
// the workflow reproducible and safe, returning 0 on success or 1 otherwise.
func CheckWorktreeDocs(root string, out, errOut io.Writer) int {
	content, err := os.ReadFile(filepath.Join(root, worktreeDoc))
	if err != nil {
		fmt.Fprintf(errOut, "Worktree-docs check failed: cannot read %s: %v\n", worktreeDoc, err)
		return 1
	}
	text := string(content)

	errors := []string{}
	for _, requirement := range worktreeDocRequirements {
		if !strings.Contains(text, requirement.fragment) {
			errors = append(errors, fmt.Sprintf("%s is missing %s (%q)", worktreeDoc, requirement.reason, requirement.fragment))
		}
	}

	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		errors = append(errors, fmt.Sprintf("cannot read README.md: %v", err))
	} else if !strings.Contains(string(readme), "("+worktreeDoc+")") {
		errors = append(errors, fmt.Sprintf("README.md does not link %s", worktreeDoc))
	}

	if len(errors) > 0 {
		fmt.Fprintln(errOut, "Worktree-docs check failed:")
		for _, error := range errors {
			fmt.Fprintf(errOut, "- %s\n", error)
		}
		return 1
	}
	fmt.Fprintf(out, "Worktree-docs check passed: %s documents the replacement workflow.\n", worktreeDoc)
	return 0
}
