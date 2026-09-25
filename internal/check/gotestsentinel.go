package check

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hidekitux/skills/internal/provider"
	"github.com/hidekitux/skills/internal/support"
)

// sentinelElements names the parts of the sentinel repository compared before
// and after the test run, in report order.
var sentinelElements = []string{"config", "HEAD", "refs", "index"}

// RunGoTestsWithSentinel runs go test with args in dir while GIT_DIR points at
// a throwaway sentinel repository, the environment a Git hook in a linked
// worktree passes to mise run validate:all. A test that starts git without
// support.GitEnv() then writes to the sentinel instead of its own temporary
// repository (Issue #372). GIT_WORK_TREE stays unset on purpose: with it, a
// leaked git add or git commit sees the sentinel's empty work tree and writes
// nothing, so the leak would go unreported. The function returns go test's
// exit status when the sentinel's configuration, HEAD, refs, and index are
// unchanged, 1 when any of them changed, and 2 when the sentinel cannot be
// prepared or read. A leak that only reads from the sentinel leaves no trace;
// CheckTestGitIsolation covers the literal git calls such a leak comes from.
func RunGoTestsWithSentinel(git provider.Git, tool provider.Tool, dir string, args []string, out, errOut io.Writer) int {
	parent, err := os.MkdirTemp("", "skills-git-sentinel-")
	if err != nil {
		fmt.Fprintf(errOut, "error: create the Git sentinel repository: %v\n", err)
		return 2
	}
	defer os.RemoveAll(parent)
	sentinel, err := prepareSentinel(git, parent)
	if err != nil {
		fmt.Fprintf(errOut, "error: create the Git sentinel repository: %v\n", err)
		return 2
	}
	gitDir := filepath.Join(sentinel, ".git")
	before, err := snapshotSentinel(git, sentinel)
	if err != nil {
		fmt.Fprintf(errOut, "error: read the Git sentinel repository: %v\n", err)
		return 2
	}
	env := append(support.GitEnv(), "GIT_DIR="+gitDir)
	result, runErr := tool.Invoke(context.Background(), provider.Command{
		Name:   "go",
		Args:   append([]string{"test"}, args...),
		Dir:    dir,
		Env:    env,
		Stdout: out,
		Stderr: errOut,
	})
	var providerErr *provider.Error
	if runErr != nil && (!errors.As(runErr, &providerErr) || providerErr.Kind != provider.KindFailure) {
		fmt.Fprintf(errOut, "error: run go test: %v\n", runErr)
		return 2
	}
	after, err := snapshotSentinel(git, sentinel)
	if err != nil {
		fmt.Fprintf(errOut, "error: read the Git sentinel repository: %v\n", err)
		return 2
	}
	changed := []string{}
	for _, element := range sentinelElements {
		if before[element] != after[element] {
			changed = append(changed, element)
		}
	}
	if len(changed) > 0 {
		fmt.Fprintf(errOut, "error: go test changed the sentinel repository that GIT_DIR pointed at (%v). A test started git with the inherited Git environment; set its Env from support.GitEnv().\n", changed)
		return 1
	}
	return result.ExitCode
}

// prepareSentinel creates a repository with one commit below parent and
// returns its work tree. The path is resolved through symbolic links so the
// GIT_DIR a test inherits names the same directory git reports.
func prepareSentinel(git provider.Git, parent string) (string, error) {
	resolved, err := filepath.EvalSymlinks(parent)
	if err != nil {
		return "", err
	}
	sentinel := filepath.Join(resolved, "sentinel")
	if err := os.Mkdir(sentinel, 0o755); err != nil {
		return "", err
	}
	for _, args := range [][]string{
		{"init", "--quiet"},
		{"-c", "user.name=Sentinel", "-c", "user.email=sentinel" + "@" + "example.invalid", "-c", "commit.gpgsign=false", "commit", "--quiet", "--allow-empty", "--message", "sentinel"},
	} {
		if _, err := git.Output(context.Background(), sentinel, args...); err != nil {
			return "", err
		}
	}
	return sentinel, nil
}

// snapshotSentinel reads each element of sentinelElements from the sentinel
// repository at sentinel.
func snapshotSentinel(git provider.Git, sentinel string) (map[string]string, error) {
	gitDir := filepath.Join(sentinel, ".git")
	snapshot := map[string]string{}
	for _, name := range []string{"config", "HEAD", "index"} {
		content, err := os.ReadFile(filepath.Join(gitDir, name))
		if errors.Is(err, os.ErrNotExist) {
			snapshot[name] = "<absent>"
			continue
		}
		if err != nil {
			return nil, err
		}
		snapshot[name] = string(content)
	}
	refs, err := git.Output(context.Background(), sentinel, "for-each-ref", "--format=%(refname) %(objectname)")
	if err != nil {
		return nil, err
	}
	snapshot["refs"] = refs.Stdout
	return snapshot, nil
}
