package provider

import (
	"context"
	"io"
	"time"
)

// Port identifiers. Each value names one external system and appears in every
// Error the corresponding adapter returns, so a diagnostic states which
// provider failed.
const (
	PortGit    = "git"
	PortGitHub = "github"
	PortMise   = "mise"
	PortFSL    = "fsl"
	PortTool   = "tool"
	PortShell  = "shell"
	PortHost   = "host-cli"
	PortAPI    = "github-api"
)

// Git runs the git command line. Authority: the local repository and the
// credentials git itself resolves for a remote operation. The adapter adds no
// credential of its own.
type Git interface {
	// Output runs git in dir and returns its standard output.
	Output(ctx context.Context, dir string, args ...string) (Result, error)
	// Stream runs git in dir and writes both streams to out and errOut.
	Stream(ctx context.Context, dir string, out, errOut io.Writer, args ...string) (Result, error)
	// Available reports whether the git binary can be resolved.
	Available() bool
}

// GitHub runs the gh command line, including the gh skill subcommands.
// Authority: the gh authentication already present in the environment. The
// adapter never reads or writes a token.
type GitHub interface {
	// Combined runs gh in dir and returns both streams in write order.
	Combined(ctx context.Context, dir string, args ...string) (Result, error)
	// Stream runs gh in dir and writes both streams to out and errOut.
	Stream(ctx context.Context, dir string, out, errOut io.Writer, args ...string) (Result, error)
	// Available reports whether the gh binary can be resolved.
	Available() bool
}

// Mise runs a mise task. Authority: none beyond the local toolchain mise
// manages.
type Mise interface {
	// Task runs mise in dir and writes both streams to out and errOut.
	Task(ctx context.Context, dir string, out, errOut io.Writer, args ...string) (Result, error)
}

// FSL runs the pinned fslc verifier. The caller passes the binary path rather
// than a name, because resolution must not depend on the inherited PATH.
// Authority: none; fslc reads local specification files only.
type FSL interface {
	// Verify runs fslc at binary and writes both streams to out and errOut.
	Verify(ctx context.Context, binary string, out, errOut io.Writer, args ...string) (Result, error)
	// Available reports whether the binary exists at the given path.
	Available(binary string) bool
}

// Tool runs a pinned development tool: commitlint, govulncheck, uv, or the
// worktree command. Authority: none beyond the local toolchain.
type Tool interface {
	// Invoke runs one tool command. The caller sets Name, Args, and any
	// stream, stdin, or environment the tool needs.
	Invoke(ctx context.Context, command Command) (Result, error)
	// Available reports whether the named tool can be resolved.
	Available(name string) bool
}

// Shell runs one bounded command through sh -c. The repository uses it for
// evaluation assertions and for an external rubric reviewer, where the command
// text comes from a scenario file rather than from user input. Authority: the
// environment the caller passes, which excludes credential-like variables when
// the caller filters them.
type Shell interface {
	// Script runs one sh -c command in dir with an optional stdin payload and
	// an optional timeout.
	Script(ctx context.Context, dir, script, stdin string, env []string, timeout time.Duration) (Result, error)
}

// NewGit returns the git adapter over a Runner.
func NewGit(runner Runner) Git { return &gitAdapter{runner: runner} }

type gitAdapter struct{ runner Runner }

func (a *gitAdapter) Available() bool { return a.runner.Available("git") }

func (a *gitAdapter) Output(ctx context.Context, dir string, args ...string) (Result, error) {
	return a.runner.Run(ctx, Command{
		Port: PortGit, Operation: operationOf(args), Name: "git", Args: args, Dir: dir,
	})
}

func (a *gitAdapter) Stream(ctx context.Context, dir string, out, errOut io.Writer, args ...string) (Result, error) {
	return a.runner.Run(ctx, Command{
		Port: PortGit, Operation: operationOf(args), Name: "git", Args: args, Dir: dir,
		Stdout: out, Stderr: errOut,
	})
}

// NewGitHub returns the gh adapter over a Runner.
func NewGitHub(runner Runner) GitHub { return &githubAdapter{runner: runner} }

type githubAdapter struct{ runner Runner }

func (a *githubAdapter) Available() bool { return a.runner.Available("gh") }

func (a *githubAdapter) Combined(ctx context.Context, dir string, args ...string) (Result, error) {
	return a.runner.Run(ctx, Command{
		Port: PortGitHub, Operation: operationOf(args), Name: "gh", Args: args, Dir: dir,
	})
}

func (a *githubAdapter) Stream(ctx context.Context, dir string, out, errOut io.Writer, args ...string) (Result, error) {
	return a.runner.Run(ctx, Command{
		Port: PortGitHub, Operation: operationOf(args), Name: "gh", Args: args, Dir: dir,
		Stdout: out, Stderr: errOut,
	})
}

// NewMise returns the mise adapter over a Runner.
func NewMise(runner Runner) Mise { return &miseAdapter{runner: runner} }

type miseAdapter struct{ runner Runner }

func (a *miseAdapter) Task(ctx context.Context, dir string, out, errOut io.Writer, args ...string) (Result, error) {
	return a.runner.Run(ctx, Command{
		Port: PortMise, Operation: operationOf(args), Name: "mise", Args: args, Dir: dir,
		Stdout: out, Stderr: errOut,
	})
}

// NewFSL returns the fslc adapter over a Runner.
func NewFSL(runner Runner) FSL { return &fslAdapter{runner: runner} }

type fslAdapter struct{ runner Runner }

func (a *fslAdapter) Available(binary string) bool { return a.runner.Available(binary) }

func (a *fslAdapter) Verify(ctx context.Context, binary string, out, errOut io.Writer, args ...string) (Result, error) {
	return a.runner.Run(ctx, Command{
		Port: PortFSL, Operation: operationOf(args), Name: binary, Args: args,
		Stdout: out, Stderr: errOut,
	})
}

// NewTool returns the pinned-tool adapter over a Runner.
func NewTool(runner Runner) Tool { return &toolAdapter{runner: runner} }

type toolAdapter struct{ runner Runner }

func (a *toolAdapter) Available(name string) bool { return a.runner.Available(name) }

func (a *toolAdapter) Invoke(ctx context.Context, command Command) (Result, error) {
	if command.Port == "" {
		command.Port = PortTool
	}
	if command.Operation == "" {
		command.Operation = operationOf(command.Args)
	}
	return a.runner.Run(ctx, command)
}

// NewShell returns the sh adapter over a Runner.
func NewShell(runner Runner) Shell { return &shellAdapter{runner: runner} }

type shellAdapter struct{ runner Runner }

func (a *shellAdapter) Script(ctx context.Context, dir, script, stdin string, env []string, timeout time.Duration) (Result, error) {
	return a.runner.Run(ctx, Command{
		Port: PortShell, Operation: "script", Name: "sh", Args: []string{"-c", script},
		Dir: dir, Env: env, Stdin: stdin, Timeout: timeout,
	})
}

// operationOf names the operation from the leading subcommand, so an Error
// stays attributable without carrying the full argument list, which can hold a
// remote URL or a prompt.
func operationOf(args []string) string {
	if len(args) == 0 {
		return "run"
	}
	return args[0]
}
