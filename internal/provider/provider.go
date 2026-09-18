// Package provider isolates every external operation the repository performs
// from the packages that decide what to do. An external operation is a process
// the repository starts or a request it sends over the network. Each external
// system has one named port in this package, one adapter that performs the
// operation, and one failure classification every caller reads the same way.
//
// This package belongs to the provider module. It imports the foundation
// module and nothing else. The credential pattern, the URL pattern, and the
// private host rule it applies before reporting process output come from
// internal/redact, a foundation package. They cannot come from
// internal/evidence, because the evidence module may import the policy module
// and the policy module imports this one, so that edge would make the module
// graph cyclic. See docs/architecture.md and workflow/module-ownership.yml.
//
// The filesystem is not a port. Its adapter is the operating system, and a
// test substitutes it by pointing the caller at a temporary root, the seam
// docs/architecture.md records for every module.
package provider

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/hidekitux/skills/internal/redact"
)

// Kind classifies why a provider operation did not succeed. A successful
// operation returns a nil error, so the absence of a Kind is the success
// classification. A product validation failure is an ordinary error or a
// report value and never carries a Kind, which keeps the two distinguishable.
type Kind string

const (
	// KindFailure means the operation ran and returned a non-zero result.
	KindFailure Kind = "failure"
	// KindTimeout means the operation exceeded the deadline the caller set.
	KindTimeout Kind = "timeout"
	// KindInterrupted means the operation was cancelled or signalled before
	// it produced a result.
	KindInterrupted Kind = "interrupted"
	// KindUnavailable means the capability the operation needs is absent, so
	// the operation never started.
	KindUnavailable Kind = "unavailable"
	// KindRetryExhausted means a caller that bounds its own attempts used the
	// last one. No adapter in this package retries on its own.
	KindRetryExhausted Kind = "retry_exhausted"
)

// Error reports one failed provider operation. Port names the external system,
// Operation names what the caller asked for, and Detail holds a bounded
// snippet of the process output with credentials and private addresses
// removed. Err keeps the underlying error for errors.Is and errors.As.
type Error struct {
	Port      string
	Operation string
	Kind      Kind
	ExitCode  int
	Detail    string
	Err       error
}

// Error implements error. The message names the port, the operation, and the
// classification so a diagnostic stays attributable without the caller
// inspecting the struct.
func (e *Error) Error() string {
	message := fmt.Sprintf("%s %s: %s", e.Port, e.Operation, e.Kind)
	if e.Kind == KindFailure && e.ExitCode != 0 {
		message = fmt.Sprintf("%s (exit %d)", message, e.ExitCode)
	}
	if e.Detail != "" {
		message = message + ": " + e.Detail
	}
	return message
}

// Unwrap returns the underlying error.
func (e *Error) Unwrap() error { return e.Err }

// KindOf returns the classification carried by err, or false when err is not a
// provider failure. A caller uses it to separate a provider failure from a
// product validation failure.
func KindOf(err error) (Kind, bool) {
	var providerErr *Error
	if errors.As(err, &providerErr) {
		return providerErr.Kind, true
	}
	return "", false
}

// IsUnavailable reports whether err classifies an absent capability.
func IsUnavailable(err error) bool {
	kind, ok := KindOf(err)
	return ok && kind == KindUnavailable
}

// Unavailable returns the error for a capability that is absent, so the
// caller reports an unavailable capability rather than a failed operation.
func Unavailable(port, operation, detail string) *Error {
	return &Error{Port: port, Operation: operation, Kind: KindUnavailable, Detail: Redact(detail)}
}

// RetryExhausted returns the error a caller that bounds its own attempts
// reports after the last attempt fails. The last failure stays in Err.
func RetryExhausted(port, operation string, attempts int, last error) *Error {
	return &Error{
		Port:      port,
		Operation: operation,
		Kind:      KindRetryExhausted,
		Detail:    fmt.Sprintf("%d attempts", attempts),
		Err:       last,
	}
}

// detailLimit bounds a diagnostic snippet. The limit matches the snippet
// internal/eval kept before the provider boundary existed, so a stage failure
// stays as readable as it was.
const detailLimit = 300

// Redact removes the values a provider diagnostic must never expose: a
// credential in any form redact.CredentialPattern recognizes, and a URL whose
// host redact.IsPrivateHost reports as private or local. The result is
// bounded to detailLimit characters, keeping the tail, because a process
// reports its failure last.
func Redact(detail string) string {
	cleaned := redact.CredentialPattern().ReplaceAllString(detail, "[redacted credential]")
	cleaned = redact.URLPattern().ReplaceAllStringFunc(cleaned, func(raw string) string {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Hostname() == "" || redact.IsPrivateHost(parsed.Hostname()) {
			return "[redacted private URL]"
		}
		return raw
	})
	cleaned = strings.TrimSpace(cleaned)
	if runes := []rune(cleaned); len(runes) > detailLimit {
		cleaned = "..." + string(runes[len(runes)-detailLimit:])
	}
	return cleaned
}

// classify maps a completed process onto a Kind. The context state decides
// first, because a killed process reports a signal that on its own cannot tell
// a deadline from a cancellation.
func classify(ctx context.Context, timedOut bool, err error) Kind {
	switch {
	case timedOut || errors.Is(ctx.Err(), context.DeadlineExceeded):
		return KindTimeout
	case errors.Is(ctx.Err(), context.Canceled):
		return KindInterrupted
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// A negative exit code means the process was signalled rather than
		// returning a status of its own.
		if exitErr.ExitCode() < 0 {
			return KindInterrupted
		}
		return KindFailure
	}
	// The process returned no status, so it never started: the binary is
	// missing, is not executable, or the working directory does not exist.
	return KindUnavailable
}
