// Command test-go runs go test with its arguments while GIT_DIR points at a
// throwaway sentinel repository, and fails when a test changed that repository
// (Issue #372). The test:go and test:json mise tasks use it, so every run
// checks that the tests would leave a calling repository unchanged when a Git
// hook starts them. An interrupt or termination signal stops go test and still
// removes the sentinel.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/hidekitux/skills/internal/check"
	"github.com/hidekitux/skills/internal/provider"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	runner := provider.OSRunner{}
	code := check.RunGoTestsWithSentinel(ctx, provider.NewGit(runner), provider.NewTool(runner), ".", os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
