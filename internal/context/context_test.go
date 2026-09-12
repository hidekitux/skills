package context

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type testCounter func(string) (int, error)

func (c testCounter) Count(value string) (int, error) { return c(value) }

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func TestCompileSelectsOnlyActivatedModules(t *testing.T) {
	root := repositoryRoot(t)
	compiler := Compiler{Root: root, Counter: testCounter(func(string) (int, error) { return 1, nil })}
	compiled, err := compiler.Compile("review-pr", Signals{TaskKind: "security", Paths: []string{"internal/trace/trace.go"}})
	if err != nil {
		t.Fatal(err)
	}
	if compiled.Manifest.Overflow != nil {
		t.Fatalf("unexpected overflow: %#v", compiled.Manifest.Overflow)
	}
	if !hasModule(compiled.Modules, "reference.review.security") {
		t.Fatalf("security reference was not selected: %#v", compiled.Manifest.Decisions)
	}
	if hasModule(compiled.Modules, "reference.review.performance") {
		t.Fatal("performance reference was selected for a security task")
	}
	for _, module := range compiled.Modules {
		if strings.Contains(module.ID, "feedback") {
			t.Fatal("feedback module was selected without feedback signals")
		}
	}
}

func TestCompileStopsOnRequiredOverflow(t *testing.T) {
	root := repositoryRoot(t)
	compiler := Compiler{Root: root, Counter: testCounter(func(string) (int, error) { return 100000, nil })}
	compiled, err := compiler.Compile("plan-issue", Signals{})
	if !errors.Is(err, ErrRequiredOverflow) {
		t.Fatalf("error = %v, want required overflow", err)
	}
	if compiled.Manifest.Overflow == nil || compiled.Manifest.Overflow.Action != "stop-and-escalate" {
		t.Fatalf("overflow manifest = %#v", compiled.Manifest.Overflow)
	}
	if compiled.Modules != nil {
		t.Fatalf("overflow returned content modules: %#v", compiled.Modules)
	}
}

func TestManifestDoesNotContainModuleContent(t *testing.T) {
	root := repositoryRoot(t)
	compiled, err := (Compiler{Root: root, Counter: testCounter(func(string) (int, error) { return 1, nil })}).Compile("plan-issue", Signals{})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(compiled.Manifest)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "# Plan Issue") || strings.Contains(string(encoded), "content") {
		t.Fatalf("manifest contains module content: %s", encoded)
	}
}

func TestDecodeSignalsRejectsTrailingJSON(t *testing.T) {
	if _, err := DecodeSignals([]byte(`{"task_kind":"review"} {"private":"value"}`)); err == nil {
		t.Fatal("trailing JSON was accepted")
	}
}

func TestAllProfilesCompileWithRepresentativeCounter(t *testing.T) {
	root := repositoryRoot(t)
	// This deterministic approximation catches profile wiring and budget
	// regressions without making unit tests depend on tokenizer downloads.
	counter := testCounter(func(value string) (int, error) {
		return len([]byte(value))/5 + 1, nil
	})
	skills := []string{
		"analyze-codebase", "analyze-project", "audit-workflow-enforcement", "bootstrap-project", "create-issue",
		"create-pr", "debug-code", "deliver-change", "fix-pr", "implement-issue",
		"improve-project", "merge-pr", "plan-issue", "propose-improvements", "refactor-code", "resolve-defect",
		"review-pr", "write-tests",
	}
	for _, skill := range skills {
		t.Run(skill, func(t *testing.T) {
			if compiled, err := (Compiler{Root: root, Counter: counter}).Compile(skill, Signals{}); err != nil {
				t.Fatalf("compile failed: %v; manifest=%#v", err, compiled.Manifest)
			}
		})
	}
}

func hasModule(modules []Module, id string) bool {
	for _, module := range modules {
		if module.ID == id {
			return true
		}
	}
	return false
}
