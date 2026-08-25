// Package support provides shared helpers for the repository Go commands:
// repository-root resolution, subprocess execution, and environment access.
package support

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ResolveRoot returns the absolute repository root for a working directory.
// Commands default their --root flag to the current working directory because
// mise tasks and workflow steps run from the repository root.
func ResolveRoot(cwd string) (string, error) {
	if cwd == "" {
		abs, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("cannot resolve working directory: %w", err)
		}
		return filepath.Clean(abs), nil
	}
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return "", fmt.Errorf("cannot resolve root %q: %w", cwd, err)
	}
	return filepath.Clean(abs), nil
}

// Output runs a command with the given arguments, returning its combined
// stdout and stderr. It returns an error when the command fails to start
// or exits non-zero.
func Output(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// OutputIn runs a command in dir, returning combined output. It returns an
// error when the command fails to start or exits non-zero.
func OutputIn(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// ExitError returns the process exit code carried by err, or 1 when err is
// non-nil but not an *exec.ExitError. A nil err returns 0.
func ExitError(err error) int {
	if err == nil {
		return 0
	}
	var exitErr *exec.ExitError
	if asExitError(err, &exitErr) {
		return exitErr.ExitCode()
	}
	return 1
}

func asExitError(err error, target **exec.ExitError) bool {
	for err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			*target = exitErr
			return true
		}
		type unwrapper interface{ Unwrap() error }
		u, ok := err.(unwrapper)
		if !ok {
			return false
		}
		err = u.Unwrap()
	}
	return false
}

// IsZeroSHA reports whether a SHA is empty or consists entirely of zeros,
// matching the "no previous value" convention used by Git push and CI events.
func IsZeroSHA(sha string) bool {
	if sha == "" {
		return true
	}
	for _, r := range sha {
		if r != '0' {
			return false
		}
	}
	return true
}

// EnvOr returns the value of the named environment variable, or fallback when
// it is unset or empty.
func EnvOr(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}
