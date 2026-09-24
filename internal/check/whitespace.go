package check

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/hidekitux/skills/internal/support"
)

// gitDiffCheck runs `git diff --check` (plus any arguments) in root and
// streams its output, returning the process exit code.
func gitDiffCheck(root string, out io.Writer, args ...string) int {
	full := append([]string{"diff", "--check"}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = root
	cmd.Env = support.GitEnv()
	cmd.Stdout = out
	cmd.Stderr = out
	err := cmd.Run()
	return support.ExitError(err)
}

// gitVerify reports whether a Git ref resolves to a commit.
func gitVerify(root, ref string) bool {
	_, err := support.GitOutputIn(root, "rev-parse", "--verify", "--quiet", ref)
	return err == nil
}

// emptyTreeSHA returns the SHA-1 of the empty tree object.
func emptyTreeSHA(root string) string {
	stdout, err := support.GitOutputIn(root, "hash-object", "-t", "tree", os.DevNull)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(stdout)
}

func isAncestor(root, base, head string) bool {
	_, err := support.GitOutputIn(root, "merge-base", "--is-ancestor", base, head)
	return err == nil
}

// CheckWhitespace rejects whitespace errors in local changes and the committed
// CI diff, mirroring check-whitespace.sh and its environment contract
// (GITHUB_SHA, CHECK_BASE_SHA).
func CheckWhitespace(root string, out, errOut io.Writer) int {
	if code := gitDiffCheck(root, out); code != 0 {
		return code
	}
	if code := gitDiffCheck(root, out, "--cached"); code != 0 {
		return code
	}

	head := support.EnvOr("GITHUB_SHA", "HEAD")
	if !gitVerify(root, head+"^{commit}") {
		fmt.Fprintln(out, "No committed tree is available; checked staged and unstaged changes only.")
		return 0
	}

	base := os.Getenv("CHECK_BASE_SHA")
	if base == "" || support.IsZeroSHA(base) {
		fmt.Fprintf(out, "Checking whitespace in the committed tree at %s.\n", head)
		return gitDiffCheck(root, out, emptyTreeSHA(root), head)
	}

	if !gitVerify(root, base+"^{commit}") {
		fmt.Fprintf(errOut, "Whitespace check base commit is unavailable: %s\n", base)
		return 2
	}
	fmt.Fprintf(out, "Checking whitespace in %s...%s.\n", base, head)
	if isAncestor(root, base, head) {
		return gitDiffCheck(root, out, base+"..."+head)
	}
	fmt.Fprintln(out, "Base is not an ancestor of "+head+"; checking the committed tree instead.")
	return gitDiffCheck(root, out, emptyTreeSHA(root), head)
}
