package check

import (
	"context"
	"strings"

	"github.com/hidekitux/skills/internal/provider"
)

// gitPort is the Git port every repository check reads the working tree
// through. A test substitutes it by assigning a Git built on a
// provider.Stub runner.
var gitPort = provider.NewGit(provider.OSRunner{})

// gitOutput runs git in root and returns its standard output.
func gitOutput(root string, args ...string) (string, error) {
	result, err := gitPort.Output(context.Background(), root, args...)
	return result.Stdout, err
}

// gitSucceeds reports whether a git command in root exits zero, which is how
// a check asks git a yes-or-no question such as "does this ref resolve".
func gitSucceeds(root string, args ...string) bool {
	_, err := gitOutput(root, args...)
	return err == nil
}

// gitTrimmed returns the trimmed standard output of a git command, or "" when
// the command fails.
func gitTrimmed(root string, args ...string) string {
	out, err := gitOutput(root, args...)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}
