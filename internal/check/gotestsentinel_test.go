package check

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/provider"
)

// sentinelEnv returns the value of name in env.
func sentinelEnv(env []string, name string) string {
	for _, entry := range env {
		if value, ok := strings.CutPrefix(entry, name+"="); ok {
			return value
		}
	}
	return ""
}

// realGit is the Git port over real processes; the sentinel must be a real
// repository for the snapshot to mean anything.
var realGit = provider.NewGit(provider.OSRunner{})

func TestRunGoTestsWithSentinel(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(t *testing.T, command provider.Command)
		exit    int
		want    int
		element string
	}{
		{name: "untouched sentinel passes", want: 0},
		{name: "test failure passes through", exit: 3, want: 3},
		{
			name: "config change",
			mutate: func(t *testing.T, command provider.Command) {
				runSentinelGit(t, command, "config", "core.bare", "true")
			},
			want:    1,
			element: "config",
		},
		{
			name: "new ref",
			mutate: func(t *testing.T, command provider.Command) {
				runSentinelGit(t, command, "branch", "leaked")
			},
			want:    1,
			element: "refs",
		},
		{
			name: "HEAD change",
			mutate: func(t *testing.T, command provider.Command) {
				runSentinelGit(t, command, "symbolic-ref", "HEAD", "refs/heads/other")
			},
			want:    1,
			element: "HEAD",
		},
		{
			name: "index change",
			mutate: func(t *testing.T, command provider.Command) {
				runSentinelGit(t, command, "add", "leaked.txt")
			},
			want:    1,
			element: "index",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			stub := &provider.Stub{Handler: func(command provider.Command) (provider.Result, error) {
				if tc.mutate != nil {
					tc.mutate(t, command)
				}
				if tc.exit != 0 {
					return provider.Fail(command, provider.KindFailure, tc.exit, "FAIL")
				}
				return provider.Result{}, nil
			}}
			var out, errOut bytes.Buffer
			got := RunGoTestsWithSentinel(realGit, provider.NewTool(stub), "/module", []string{"-json", "./..."}, &out, &errOut)
			if got != tc.want {
				t.Fatalf("RunGoTestsWithSentinel() = %d, want %d; err=%q", got, tc.want, errOut.String())
			}
			if len(stub.Calls) != 1 {
				t.Fatalf("expected one go test call, got %d", len(stub.Calls))
			}
			call := stub.Calls[0]
			if call.Name != "go" || !slices.Equal(call.Args, []string{"test", "-json", "./..."}) || call.Dir != "/module" {
				t.Fatalf("unexpected go test call %#v", call)
			}
			gitDir := sentinelEnv(call.Env, "GIT_DIR")
			if filepath.Base(gitDir) != ".git" || sentinelEnv(call.Env, "GIT_WORK_TREE") != "" {
				t.Fatalf("go test did not get the sentinel Git environment: %q", call.Env)
			}
			if _, err := os.Stat(gitDir); !os.IsNotExist(err) {
				t.Fatalf("sentinel repository %s was not removed: %v", gitDir, err)
			}
			reported := strings.Contains(errOut.String(), "changed the sentinel repository")
			if tc.element == "" && reported {
				t.Fatalf("unexpected sentinel report %q", errOut.String())
			}
			if tc.element != "" && (!reported || !strings.Contains(errOut.String(), tc.element)) {
				t.Fatalf("expected a report naming %q, got %q", tc.element, errOut.String())
			}
		})
	}
}

// TestRunGoTestsWithSentinelReportsAnUnrunTest keeps a go binary that could
// not start from reading as a test result.
func TestRunGoTestsWithSentinelReportsAnUnrunTest(t *testing.T) {
	stub := &provider.Stub{Absent: map[string]bool{"go": true}}
	var out, errOut bytes.Buffer
	if got := RunGoTestsWithSentinel(realGit, provider.NewTool(stub), ".", nil, &out, &errOut); got != 2 {
		t.Fatalf("RunGoTestsWithSentinel() = %d, want 2; err=%q", got, errOut.String())
	}
	if !strings.Contains(errOut.String(), "error: run go test:") {
		t.Fatalf("expected the start failure in err=%q", errOut.String())
	}
}

// TestRunGoTestsWithSentinelReportsAnUnpreparedSentinel keeps a git failure
// while creating the sentinel from running the tests unguarded.
func TestRunGoTestsWithSentinelReportsAnUnpreparedSentinel(t *testing.T) {
	git := provider.NewGit(&provider.Stub{Handler: func(command provider.Command) (provider.Result, error) {
		return provider.Fail(command, provider.KindFailure, 128, "fatal")
	}})
	tool := &provider.Stub{}
	var out, errOut bytes.Buffer
	if got := RunGoTestsWithSentinel(git, provider.NewTool(tool), ".", nil, &out, &errOut); got != 2 {
		t.Fatalf("RunGoTestsWithSentinel() = %d, want 2; err=%q", got, errOut.String())
	}
	if len(tool.Calls) != 0 {
		t.Fatalf("go test ran without a sentinel: %#v", tool.Calls)
	}
}

// runSentinelGit runs git in a fresh temporary directory holding leaked.txt
// with the environment the stubbed go test received, which is what an
// unisolated test does with its own temporary repository.
func runSentinelGit(t *testing.T, command provider.Command, args ...string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "leaked.txt"), []byte("leaked\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runner := provider.OSRunner{}
	if _, err := runner.Run(context.Background(), provider.Command{Name: "git", Args: args, Dir: dir, Env: command.Env}); err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
}
