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
	{"git worktree", "the native fallback for when the tool is unavailable"},
}

// forcingFlags are the operations that bypass the repository's worktree-removal
// rules, each listed with every spelling the CLI accepts. `wt remove --help`
// documents `-f, --force` and `-D, --force-delete`, so covering one spelling per
// operation would leave the other free to appear in the guidance this check
// exists to reject. The document must name each operation so a reader
// recognizes it, and every paragraph naming one must also forbid it.
var forcingFlags = []struct {
	operation string
	spellings []string
}{
	{"forced removal of a dirty worktree", []string{"wt remove --force", "wt remove -f"}},
	{"deletion of an unmerged branch", []string{"wt remove --force-delete", "wt remove -D"}},
}

// prohibitionMarkers distinguish a paragraph that forbids an operation from one
// that instructs the reader to perform it.
var prohibitionMarkers = []string{"do not", "never"}

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
	errors = append(errors, forcingFlagFindings(text)...)

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

// forcingFlagFindings reports every forcing flag the document fails to name and
// every paragraph that names one without forbidding it. Paragraph granularity
// is deliberate: it is coarse enough to survive rewording and line wrapping,
// and narrow enough that a prohibition elsewhere in the document cannot excuse
// an instruction to run the flag here.
func forcingFlagFindings(text string) []string {
	paragraphs := strings.Split(text, "\n\n")
	findings := []string{}
	for _, flag := range forcingFlags {
		named := false
		for _, paragraph := range paragraphs {
			for _, spelling := range flag.spellings {
				if !namesFlag(paragraph, spelling) {
					continue
				}
				named = true
				if !forbids(paragraph) {
					findings = append(findings, fmt.Sprintf("%s names %q outside a prohibition; the paragraph must forbid %s with %q or %q", worktreeDoc, spelling, flag.operation, prohibitionMarkers[0], prohibitionMarkers[1]))
				}
			}
		}
		if !named {
			findings = append(findings, fmt.Sprintf("%s must name %s (%s) so the forbidden operation stays recognizable", worktreeDoc, flag.operation, strings.Join(flag.spellings, " or ")))
		}
	}
	return findings
}

// namesFlag reports whether the paragraph contains the spelling as a complete
// flag rather than as the prefix of a longer one, so `wt remove --force` is not
// attributed to a paragraph that only says `wt remove --force-delete`.
func namesFlag(paragraph, spelling string) bool {
	for start := 0; ; {
		relative := strings.Index(paragraph[start:], spelling)
		if relative < 0 {
			return false
		}
		index := start + relative + len(spelling)
		if index == len(paragraph) || !isFlagCharacter(paragraph[index]) {
			return true
		}
		start = index
	}
}

func isFlagCharacter(value byte) bool {
	return value == '-' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}

func forbids(paragraph string) bool {
	lowered := strings.ToLower(paragraph)
	for _, marker := range prohibitionMarkers {
		if strings.Contains(lowered, marker) {
			return true
		}
	}
	return false
}
