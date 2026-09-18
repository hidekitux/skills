package provider

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/hidekitux/skills/internal/support"
)

// lockedWriter serializes writes to the writer it wraps. exec.Cmd copies
// standard output and standard error in separate goroutines whenever the
// writer is not an *os.File, so the combined buffer and a caller writer that
// receives both streams would otherwise be written concurrently.
type lockedWriter struct {
	mutex  *sync.Mutex
	writer io.Writer
}

func (w *lockedWriter) Write(content []byte) (int, error) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.writer.Write(content)
}

// Command describes one external process invocation. Name is the binary, by
// name for a PATH lookup or by path when the caller pins the install location.
// An empty Env means the repository's Git-free environment, which keeps a
// child process discovering its repository from Dir rather than from an outer
// hook's GIT_DIR.
type Command struct {
	Port      string
	Operation string
	Name      string
	Args      []string
	Dir       string
	Env       []string
	Stdin     string
	Stdout    io.Writer
	Stderr    io.Writer
	Timeout   time.Duration
}

// Result holds the observable outcome of one completed process. Combined
// keeps the two streams in the order the process wrote them, which is what a
// diagnostic quotes.
type Result struct {
	Stdout   string
	Stderr   string
	Combined string
	ExitCode int
}

// Runner is the process-execution port. Every external process the repository
// starts goes through one Runner, so a test substitutes every process-backed
// provider by passing a different Runner.
type Runner interface {
	// Run executes one command and returns its result. A non-zero exit, an
	// expired deadline, a cancellation, and an absent binary all return an
	// *Error carrying the classification.
	Run(ctx context.Context, command Command) (Result, error)
	// Available reports whether the named binary can be resolved.
	Available(name string) bool
}

// OSRunner runs a command as a real operating-system process. It is the
// production adapter for the Runner port.
type OSRunner struct{}

// Available resolves the binary through PATH, or reports true for a name that
// is already a path.
func (OSRunner) Available(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// Run starts the process, applies the command timeout when one is set, and
// classifies every failure before returning it.
func (OSRunner) Run(ctx context.Context, command Command) (Result, error) {
	runCtx := ctx
	var cancel context.CancelFunc
	if command.Timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, command.Timeout)
		defer cancel()
	}
	process := exec.CommandContext(runCtx, command.Name, command.Args...)
	process.Dir = command.Dir
	if command.Env == nil {
		process.Env = support.GitEnv()
	} else {
		process.Env = command.Env
	}
	if command.Stdin != "" {
		process.Stdin = strings.NewReader(command.Stdin)
	}
	var stdout, stderr, combined bytes.Buffer
	stdoutWriters := []io.Writer{&stdout, &combined}
	stderrWriters := []io.Writer{&stderr, &combined}
	if command.Stdout != nil {
		stdoutWriters = append(stdoutWriters, command.Stdout)
	}
	if command.Stderr != nil {
		stderrWriters = append(stderrWriters, command.Stderr)
	}
	// One mutex covers both streams, so the combined buffer keeps the order
	// the process wrote in and a caller writer that receives both streams
	// never sees two concurrent writes.
	var mutex sync.Mutex
	process.Stdout = &lockedWriter{mutex: &mutex, writer: io.MultiWriter(stdoutWriters...)}
	process.Stderr = &lockedWriter{mutex: &mutex, writer: io.MultiWriter(stderrWriters...)}

	err := process.Run()
	result := Result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Combined: combined.String(),
		ExitCode: support.ExitError(err),
	}
	if err == nil {
		return result, nil
	}
	timedOut := command.Timeout > 0 && runCtx.Err() == context.DeadlineExceeded
	return result, &Error{
		Port:      command.Port,
		Operation: command.Operation,
		Kind:      classify(runCtx, timedOut, err),
		ExitCode:  result.ExitCode,
		Detail:    Redact(result.Combined),
		Err:       err,
	}
}

// Stub is the deterministic adapter for the Runner port. It starts no process,
// records every command it receives, and answers from Handler. A test
// substitutes any process-backed provider with it, so the test needs no
// network access, no credentials, and no ambient repository state.
type Stub struct {
	// Handler answers one command. A nil Handler succeeds with an empty
	// result.
	Handler func(Command) (Result, error)
	// Absent names the binaries Available reports as missing.
	Absent map[string]bool
	// Calls records every command in the order it was received.
	Calls []Command
}

// Available reports a binary as present unless the test listed it in Absent.
func (s *Stub) Available(name string) bool { return !s.Absent[name] }

// Run records the command and returns the handler's answer. When the binary is
// listed in Absent, it returns an unavailable-capability error without
// consulting the handler, matching what OSRunner reports for a missing binary.
func (s *Stub) Run(_ context.Context, command Command) (Result, error) {
	s.Calls = append(s.Calls, command)
	if s.Absent[command.Name] {
		return Result{ExitCode: 1}, Unavailable(command.Port, command.Operation, command.Name+" is not installed")
	}
	if s.Handler == nil {
		return Result{}, nil
	}
	return s.Handler(command)
}

// Fail returns the result and error a stub handler uses to reproduce one
// classified provider failure.
func Fail(command Command, kind Kind, exitCode int, detail string) (Result, error) {
	return Result{Combined: detail, ExitCode: exitCode}, &Error{
		Port:      command.Port,
		Operation: command.Operation,
		Kind:      kind,
		ExitCode:  exitCode,
		Detail:    Redact(detail),
	}
}

// LookPath resolves an executable name to its absolute path. It is the one
// capability query the repository makes outside a Runner, because a caller
// that records a resolved path in a generated script needs the path itself
// rather than the result of running the command.
func LookPath(name string) (string, error) {
	path, err := exec.LookPath(name)
	if err != nil {
		return "", Unavailable(PortTool, "look-path", name+" is not installed")
	}
	return path, nil
}
