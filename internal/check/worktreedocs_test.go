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
	if !strings.Contains(output, "wt remove --force") || !strings.Contains(output, "wt remove -D") {
		t.Fatalf("expected both forced-removal prohibitions reported, got: %s", output)
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

func TestWorktreeDocsRejectsUnlinkedDocument(t *testing.T) {
	root := writeWorktreeDocFixture(t, completeWorktreeDoc, "no link to the workflow\n")
	code, output := runWorktreeDocsCheck(t, root)
	if code != 1 || !strings.Contains(output, "does not link docs/worktrees.md") {
		t.Fatalf("expected unlinked-document failure, got exit %d: %s", code, output)
	}
}
