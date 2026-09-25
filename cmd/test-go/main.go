// Command test-go runs go test with its arguments while GIT_DIR points at a
// throwaway sentinel repository, and fails when a test changed that repository
// (Issue #372). The test:go and test:json mise tasks use it, so every run
// checks that the tests would leave a calling repository unchanged when a Git
// hook starts them.
//
// A terminal interrupt reaches go test through the process group, so test-go
// drops its own copy and waits: go test then removes its build directory,
// and test-go removes the sentinel. A termination signal reaches test-go
// alone, so it stops go test and removes the sentinel.
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
	// signal.Ignore would be inherited by go test, so the interrupt is
	// received and dropped instead.
	interrupts := make(chan os.Signal, 1)
	signal.Notify(interrupts, os.Interrupt)
	go func() {
		for range interrupts {
		}
	}()
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM)
	runner := provider.OSRunner{}
	code := check.RunGoTestsWithSentinel(ctx, provider.NewGit(runner), provider.NewTool(runner), ".", os.Args[1:], os.Stdout, os.Stderr)
	stop()
	os.Exit(code)
}
