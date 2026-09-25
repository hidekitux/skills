// Command test-go runs go test with its arguments while GIT_DIR points at a
// throwaway sentinel repository, and fails when a test changed that repository (Issue #372). The test:go and
// test:json mise tasks use it, so every run checks that the tests would leave
// a calling repository unchanged when a Git hook starts them.
package main

import (
	"os"

	"github.com/hidekitux/skills/internal/check"
	"github.com/hidekitux/skills/internal/provider"
)

func main() {
	runner := provider.OSRunner{}
	os.Exit(check.RunGoTestsWithSentinel(provider.NewGit(runner), provider.NewTool(runner), ".", os.Args[1:], os.Stdout, os.Stderr))
}
