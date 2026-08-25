package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func passCheck(name, message string) repoCheck {
	return repoCheck{
		name: name,
		fn: func(root string, out, errOut io.Writer) int {
			io.WriteString(out, message+"\n")
			return 0
		},
	}
}

func failCheck(name, message string) repoCheck {
	return repoCheck{
		name: name,
		fn: func(root string, out, errOut io.Writer) int {
			io.WriteString(errOut, message+"\n")
			return 1
		},
	}
}

func TestRunReportsEveryCheckAndPasses(t *testing.T) {
	checks := []repoCheck{
		passCheck("first", "first ok"),
		passCheck("second", "second ok"),
	}
	var out, errOut bytes.Buffer
	if code := run(".", &out, &errOut, checks); code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
	for _, want := range []string{"[first] first ok", "[second] second ok", "all 2 repository checks passed"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
	if errOut.Len() != 0 {
		t.Fatalf("unexpected error output: %q", errOut.String())
	}
}

func TestRunFailsAggregateAndNamesFailingCheck(t *testing.T) {
	checks := []repoCheck{
		passCheck("first", "first ok"),
		failCheck("second", "second broke"),
		passCheck("third", "third ok"),
	}
	var out, errOut bytes.Buffer
	if code := run(".", &out, &errOut, checks); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	// Every check still ran even though one failed.
	for _, want := range []string{"[first] first ok", "[third] third ok"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("output missing %q:\n%s", want, out.String())
		}
	}
	for _, want := range []string{"[second] second broke", "FAIL: second", "1 of 3 repository checks failed: second"} {
		if !strings.Contains(errOut.String(), want) {
			t.Fatalf("error output missing %q:\n%s", want, errOut.String())
		}
	}
}

func TestRunNamesMultipleFailures(t *testing.T) {
	checks := []repoCheck{
		failCheck("first", "first broke"),
		passCheck("second", "second ok"),
		failCheck("third", "third broke"),
	}
	var out, errOut bytes.Buffer
	if code := run(".", &out, &errOut, checks); code != 1 {
		t.Fatalf("expected 1, got %d", code)
	}
	want := "2 of 3 repository checks failed: first, third"
	if !strings.Contains(errOut.String(), want) {
		t.Fatalf("error output missing %q:\n%s", want, errOut.String())
	}
}

// TestRunStartsEveryCheckBeforeAnyCompletes proves checks execute
// concurrently: if the runner serialized them, the first check would block
// until the test times out instead of both starting.
func TestRunStartsEveryCheckBeforeAnyCompletes(t *testing.T) {
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	checks := []repoCheck{
		{name: "a", fn: func(root string, out, errOut io.Writer) int { started <- struct{}{}; <-release; return 0 }},
		{name: "b", fn: func(root string, out, errOut io.Writer) int { started <- struct{}{}; <-release; return 0 }},
	}
	done := make(chan int, 1)
	go func() {
		var out, errOut bytes.Buffer
		done <- run(".", &out, &errOut, checks)
	}()
	for i := 0; i < 2; i++ {
		select {
		case <-started:
		case <-time.After(5 * time.Second):
			t.Fatal("runner did not start every check concurrently")
		}
	}
	close(release)
	if code := <-done; code != 0 {
		t.Fatalf("expected 0, got %d", code)
	}
}

// TestRunRepositoryChecksAgainstRepositoryRoot exercises the real check set
// against the repository tree as an end-to-end guard: the aggregate command
// must pass exactly when every repository check passes.
func TestRunRepositoryChecksAgainstRepositoryRoot(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := filepath.Dir(filepath.Dir(cwd))
	var out, errOut bytes.Buffer
	if code := run(root, &out, &errOut, repoChecks); code != 0 {
		t.Fatalf("repository checks failed (%d):\n%s%s", code, out.String(), errOut.String())
	}
	want := "all 6 repository checks passed"
	if !strings.Contains(out.String(), want) {
		t.Fatalf("output missing %q:\n%s", want, out.String())
	}
	for _, label := range []string{"[validate-repository]", "[check-tool-licenses]", "[validate-script-tests]", "[check-sensitive-content]", "[check-mutation-badges]", "[check-analyze-readonly]"} {
		if !strings.Contains(out.String(), label) {
			t.Fatalf("output missing check label %q:\n%s", label, out.String())
		}
	}
}
