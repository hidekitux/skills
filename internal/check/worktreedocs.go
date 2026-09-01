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

// removalCommands remove a worktree, whichever tool performs it. The document
// presents native Git as a supported fallback, so a guard scoped to one tool
// would leave the other free to carry the same destructive instruction.
var removalCommands = []string{"git worktree remove", "wt remove"}

// forcedRemovals are the operations that bypass the repository's
// worktree-removal rules, each listed with the flag tokens that request it.
// Matching a command and a flag token separately, rather than enumerating whole
// command spellings, keeps the guard from depending on which of the two is
// written next to the other.
var forcedRemovals = []struct {
	operation string
	tokens    []string
}{
	{"forced removal of a dirty worktree", []string{"--force", "-f"}},
	{"deletion of an unmerged branch", []string{"--force-delete", "-D"}},
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
	text := unwrapCodeSpans(string(content))

	errors := []string{}
	for _, requirement := range worktreeDocRequirements {
		if !strings.Contains(text, requirement.fragment) {
			errors = append(errors, fmt.Sprintf("%s is missing %s (%q)", worktreeDoc, requirement.reason, requirement.fragment))
		}
	}
	errors = append(errors, forcedRemovalFindings(text)...)

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

// forcedRemovalFindings reports every invocation that runs a removal command
// with a forcing flag outside a prohibition, and every forbidden operation the
// document never forbids at all.
//
// A flag counts against a command only when it falls inside that command's
// invocation span, so a paragraph may still describe a flag's behavior in
// prose. Prohibition is judged per paragraph, which is coarse enough to survive
// rewording and line wrapping, and narrow enough that a prohibition elsewhere
// in the document cannot excuse an instruction here.
func forcedRemovalFindings(text string) []string {
	findings := []string{}
	forbidden := map[string]bool{}
	for _, paragraph := range strings.Split(text, "\n\n") {
		for _, command := range removalCommands {
			for start := 0; ; {
				relative := strings.Index(paragraph[start:], command)
				if relative < 0 {
					break
				}
				index := start + relative
				span := invocationSpan(paragraph, index)
				for _, removal := range forcedRemovals {
					token := namedToken(span, removal.tokens)
					if token == "" {
						continue
					}
					if forbids(paragraph) {
						forbidden[removal.operation] = true
						continue
					}
					findings = append(findings, fmt.Sprintf("%s runs %q with %q outside a prohibition; the paragraph must forbid %s with %q or %q", worktreeDoc, command, token, removal.operation, prohibitionMarkers[0], prohibitionMarkers[1]))
				}
				start = index + len(command)
			}
		}
	}
	for _, removal := range forcedRemovals {
		if !forbidden[removal.operation] {
			findings = append(findings, fmt.Sprintf("%s must forbid %s in a paragraph that runs a removal command with %s", worktreeDoc, removal.operation, strings.Join(removal.tokens, " or ")))
		}
	}
	return findings
}

// unwrapCodeSpans joins the lines that Markdown wrapping split an inline code
// span across, so a command and its flags stay inside one invocation span.
// Prose in this repository wraps at roughly 76 characters and does split code
// spans, so without this a wrapped `wt remove` / `-f` pair would never form.
//
// Fenced blocks are left alone. There a line break separates two commands, and
// joining them would let a flag on one line pair with a command on another.
//
// Known limitation: this pattern-matches Markdown rather than parsing it, so a
// syntax error upstream of the guard weakens the guard. An opening fence with
// no closing fence leaves the document fenced to its end, and unwrapping stops
// running, so a wrapped invocation after it is not paired. The prohibition
// boundary itself still holds — `open` stays false, so no paragraph merges —
// and every well-formed document is covered. Closing this would mean validating
// the document's Markdown, which is deliberately out of scope here.
func unwrapCodeSpans(text string) string {
	var builder strings.Builder
	fenced, open := false, false
	for index, line := range strings.Split(text, "\n") {
		// A CommonMark inline code span cannot contain a blank line, so a
		// paragraph break always ends one. Without this reset, a single
		// unterminated backtick would join the rest of the document into one
		// paragraph and silently erase the boundary that keeps a prohibition
		// from excusing a later instruction. Fenced blocks may contain blank
		// lines, so `fenced` deliberately survives them.
		if strings.TrimSpace(line) == "" {
			open = false
		}
		if index > 0 {
			if open {
				builder.WriteString(" ")
				line = strings.TrimLeft(line, " \t")
			} else {
				builder.WriteString("\n")
			}
		}
		switch {
		case strings.HasPrefix(strings.TrimSpace(line), "```"):
			fenced = !fenced
			open = false
		case !fenced && strings.Count(line, "`")%2 == 1:
			open = !open
		}
		builder.WriteString(line)
	}
	return builder.String()
}

// invocationSpan returns the command invocation beginning at index, ending at
// the first backtick or line break. Commands here are written inside code spans
// or fenced blocks, so that boundary stops a flag discussed later in the same
// paragraph from being read as an argument of this command. It relies on
// unwrapCodeSpans having already rejoined any code span split across lines.
func invocationSpan(paragraph string, index int) string {
	end := strings.IndexAny(paragraph[index:], "`\n")
	if end < 0 {
		return paragraph[index:]
	}
	return paragraph[index : index+end]
}

// namedToken returns the first token the span passes as a complete flag, or the
// empty string. Boundaries on both sides keep `-f` from matching inside
// `--force` and `--force` from matching inside `--force-delete`.
func namedToken(span string, tokens []string) string {
	for _, token := range tokens {
		for start := 0; ; {
			relative := strings.Index(span[start:], token)
			if relative < 0 {
				break
			}
			begin := start + relative
			end := begin + len(token)
			leading := begin == 0 || !isFlagCharacter(span[begin-1])
			trailing := end == len(span) || !isFlagCharacter(span[end])
			if leading && trailing {
				return token
			}
			start = begin + 1
		}
	}
	return ""
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
