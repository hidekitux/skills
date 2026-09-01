package check

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// completeWorktreeDoc is the smallest document that satisfies every
// requirement, so each test can remove exactly one fragment.
const completeWorktreeDoc = `# Worktree workflow

## Decision

| Reviewed version | v0.75.0 |
| License | MIT OR Apache-2.0 |

## Workflow

wt switch --create issue/1
wt list
wt remove issue/1

Never use wt remove --force or wt remove -D.
Native git worktree remains the fallback.
`

func writeWorktreeDocFixture(t *testing.T, doc, readme string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, worktreeDoc), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func runWorktreeDocsCheck(t *testing.T, root string) (int, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := CheckWorktreeDocs(root, &out, &errOut)
	return code, out.String() + errOut.String()
}

func TestWorktreeDocsAcceptsCompleteDocument(t *testing.T) {
	root := writeWorktreeDocFixture(t, completeWorktreeDoc, "See [worktrees](docs/worktrees.md).\n")
	if code, output := runWorktreeDocsCheck(t, root); code != 0 {
		t.Fatalf("expected pass, got exit %d: %s", code, output)
	}
}

func TestWorktreeDocsRejectsMissingDocument(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("no link\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, output := runWorktreeDocsCheck(t, root)
	if code != 1 || !strings.Contains(output, "cannot read docs/worktrees.md") {
		t.Fatalf("expected missing-document failure, got exit %d: %s", code, output)
	}
}

func TestWorktreeDocsRejectsMissingSafeRemovalRule(t *testing.T) {
	doc := strings.ReplaceAll(completeWorktreeDoc, "Never use wt remove --force or wt remove -D.\n", "")
	root := writeWorktreeDocFixture(t, doc, "See [worktrees](docs/worktrees.md).\n")
	code, output := runWorktreeDocsCheck(t, root)
	if code != 1 {
		t.Fatalf("expected failure, got exit %d: %s", code, output)
	}
	if !strings.Contains(output, "forced removal of a dirty worktree") || !strings.Contains(output, "deletion of an unmerged branch") {
		t.Fatalf("expected both forbidden operations reported, got: %s", output)
	}
}

// Naming a forcing flag is not enough: a document that drifted from forbidding
// the flag to recommending it must fail, which a presence-only check cannot do.
func TestWorktreeDocsRejectsRecommendedForcedRemoval(t *testing.T) {
	doc := strings.ReplaceAll(completeWorktreeDoc,
		"Never use wt remove --force or wt remove -D.",
		"Use wt remove --force or wt remove -D to clean up quickly.")
	root := writeWorktreeDocFixture(t, doc, "See [worktrees](docs/worktrees.md).\n")
	code, output := runWorktreeDocsCheck(t, root)
	if code != 1 {
		t.Fatalf("expected failure, got exit %d: %s", code, output)
	}
	if !strings.Contains(output, "outside a prohibition") {
		t.Fatalf("expected the prohibition failure, got: %s", output)
	}
}

// A prohibition elsewhere in the document must not excuse an instruction to run
// the flag in another paragraph.
func TestWorktreeDocsRejectsForcedRemovalDespiteDistantProhibition(t *testing.T) {
	doc := completeWorktreeDoc + "\nUse wt remove --force when a worktree is stale.\n"
	root := writeWorktreeDocFixture(t, doc, "See [worktrees](docs/worktrees.md).\n")
	code, output := runWorktreeDocsCheck(t, root)
	if code != 1 || !strings.Contains(output, "outside a prohibition") {
		t.Fatalf("expected the distant-prohibition case to fail, got exit %d: %s", code, output)
	}
}

// The rule pairs a removal command with a flag token, so it must catch every
// alias `wt remove --help` documents and the native fallback the same document
// recommends, without enumerating whole command spellings.
func TestWorktreeDocsRejectsForcedRemovalAcrossCommandsAndAliases(t *testing.T) {
	cases := []struct {
		invocation string
		command    string
		token      string
	}{
		{"wt remove -f <branch>", "wt remove", "-f"},
		{"wt remove --force <branch>", "wt remove", "--force"},
		{"wt remove --force-delete <branch>", "wt remove", "--force-delete"},
		{"wt remove -D <branch>", "wt remove", "-D"},
		{"git worktree remove --force <path>", "git worktree remove", "--force"},
		{"git worktree remove -f <path>", "git worktree remove", "-f"},
	}
	for _, testCase := range cases {
		t.Run(testCase.invocation, func(t *testing.T) {
			doc := completeWorktreeDoc + "\nFor a stale worktree, run `" + testCase.invocation + "` to clean up quickly.\n"
			root := writeWorktreeDocFixture(t, doc, "See [worktrees](docs/worktrees.md).\n")
			code, output := runWorktreeDocsCheck(t, root)
			if code != 1 || !strings.Contains(output, "outside a prohibition") {
				t.Fatalf("expected the recommendation to fail, got exit %d: %s", code, output)
			}
			want := "runs \"" + testCase.command + "\" with \"" + testCase.token + "\""
			if !strings.Contains(output, want) {
				t.Fatalf("expected the finding to report %s, got: %s", want, output)
			}
		})
	}
}

// Markdown wrapping must not hide an invocation: prose here wraps at roughly 76
// characters, and this repository has already had a wrapped command escape a
// literal-matching guard.
func TestWorktreeDocsRejectsWrappedForcedRemoval(t *testing.T) {
	doc := completeWorktreeDoc + "\nFor a stale worktree, run `wt remove\n-f <branch>` to clean it up quickly.\n"
	root := writeWorktreeDocFixture(t, doc, "See [worktrees](docs/worktrees.md).\n")
	code, output := runWorktreeDocsCheck(t, root)
	if code != 1 || !strings.Contains(output, "outside a prohibition") {
		t.Fatalf("expected the wrapped invocation to fail, got exit %d: %s", code, output)
	}
	if !strings.Contains(output, "runs \"wt remove\" with \"-f\"") {
		t.Fatalf("expected the wrapped command and flag to be paired, got: %s", output)
	}
}

// An unterminated code span must not join paragraphs. A blank line always ends
// an inline code span, and without that reset one backtick typo would let an
// unrelated prohibition excuse every later instruction in the document.
func TestWorktreeDocsRejectsForcedRemovalAfterAnUnterminatedCodeSpan(t *testing.T) {
	doc := completeWorktreeDoc +
		"\nNote that the `--reap flag is experimental and Unix only.\n" +
		"\nNever remove a worktree that still holds unpushed commits.\n" +
		"\nFor a stale worktree, run `wt remove -f <branch>` to clean it up.\n"
	root := writeWorktreeDocFixture(t, doc, "See [worktrees](docs/worktrees.md).\n")
	code, output := runWorktreeDocsCheck(t, root)
	if code != 1 || !strings.Contains(output, "runs \"wt remove\" with \"-f\"") {
		t.Fatalf("expected the instruction to stay unexcused, got exit %d: %s", code, output)
	}
}

// A fenced block may contain blank lines, so the blank-line reset must not end
// the block and start treating its content as prose.
func TestWorktreeDocsAcceptsAFencedBlockContainingABlankLine(t *testing.T) {
	doc := completeWorktreeDoc + "\n```bash\nwt remove issue/1\n\ngit worktree list -f\n```\n"
	root := writeWorktreeDocFixture(t, doc, "See [worktrees](docs/worktrees.md).\n")
	if code, output := runWorktreeDocsCheck(t, root); code != 0 {
		t.Fatalf("expected a fenced block with a blank line to pass, got exit %d: %s", code, output)
	}
}

// Inside a fenced block a line break separates two commands, so the flag on one
// line must not pair with the command on the previous one.
func TestWorktreeDocsAcceptsAdjacentLinesInAFencedBlock(t *testing.T) {
	doc := completeWorktreeDoc + "\n```bash\nwt remove issue/1\ngit worktree list -f\n```\n"
	root := writeWorktreeDocFixture(t, doc, "See [worktrees](docs/worktrees.md).\n")
	if code, output := runWorktreeDocsCheck(t, root); code != 0 {
		t.Fatalf("expected adjacent fenced lines to stay separate, got exit %d: %s", code, output)
	}
}

// A flag discussed in prose elsewhere in the paragraph is not an argument of
// the command, so describing behavior must not be reported as an instruction.
func TestWorktreeDocsAcceptsFlagDescribedOutsideAnInvocation(t *testing.T) {
	doc := completeWorktreeDoc + "\n| `wt remove issue/999` | Kept the branch and reported that `-D` would delete it |\n"
	root := writeWorktreeDocFixture(t, doc, "See [worktrees](docs/worktrees.md).\n")
	if code, output := runWorktreeDocsCheck(t, root); code != 0 {
		t.Fatalf("expected a described flag to pass, got exit %d: %s", code, output)
	}
}

// Naming an operation through either spelling satisfies the naming rule, so a
// document may forbid the short form alone.
func TestWorktreeDocsAcceptsAliasOnlyProhibition(t *testing.T) {
	doc := strings.ReplaceAll(completeWorktreeDoc,
		"Never use wt remove --force or wt remove -D.",
		"Never use wt remove -f or wt remove --force-delete.")
	root := writeWorktreeDocFixture(t, doc, "See [worktrees](docs/worktrees.md).\n")
	if code, output := runWorktreeDocsCheck(t, root); code != 0 {
		t.Fatalf("expected an alias-only prohibition to pass, got exit %d: %s", code, output)
	}
}

func TestWorktreeDocsRejectsUnlinkedDocument(t *testing.T) {
	root := writeWorktreeDocFixture(t, completeWorktreeDoc, "no link to the workflow\n")
	code, output := runWorktreeDocsCheck(t, root)
	if code != 1 || !strings.Contains(output, "does not link docs/worktrees.md") {
		t.Fatalf("expected unlinked-document failure, got exit %d: %s", code, output)
	}
}
