package eval

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestEvaluateAssertions(t *testing.T) {
	sandbox := t.TempDir()
	writeTestFile(t, sandbox, "src/app.go", "package app\nfunc Apply() {}\n")
	writeTestFile(t, sandbox, "out.txt", "done\n")

	before := snapshotHashes(sandbox, []string{"src/app.go", "out.txt"})

	t.Run("passes", func(t *testing.T) {
		sc := &Scenario{Expectations: Expectations{
			Handoff:           "implement-issue",
			TranscriptMust:    []string{"implement-issue"},
			TranscriptMustNot: []string{"ghp_"},
			FilesMustExist:    []string{"out.txt"},
			FilesMustNotExist: []string{"secret.txt"},
			FileContains:      []FileContains{{Path: "src/app.go", Pattern: "Apply"}},
			UnchangedFiles:    []string{"src/app.go"},
			CommandRun:        []CommandRun{{Run: "true", Dir: "", Exit: 0}},
		}}
		failures := evaluateAssertions(context.Background(), sc, "handing to implement-issue", sandbox, before, map[string]bool{"implement-issue": true})
		if len(failures) != 0 {
			t.Fatalf("expected no failures, got %v", failures)
		}
	})

	t.Run("reports every failing assertion", func(t *testing.T) {
		sc := &Scenario{Expectations: Expectations{
			Handoff:           "implement-issue",
			TranscriptMust:    []string{"implement-issue", "missing-pattern"},
			TranscriptMustNot: []string{"ghp_"},
			FilesMustExist:    []string{"nope.txt"},
			FilesMustNotExist: []string{"out.txt"},
			FileContains:      []FileContains{{Path: "src/app.go", Pattern: "Absent"}},
			UnchangedFiles:    []string{"src/app.go"},
		}}
		if err := os.WriteFile(filepath.Join(sandbox, "src/app.go"), []byte("changed"), 0o644); err != nil {
			t.Fatal(err)
		}
		defer writeTestFile(t, sandbox, "src/app.go", "package app\nfunc Apply() {}\n")
		failures := evaluateAssertions(context.Background(), sc, "no handoff here ghp_123", sandbox, before, map[string]bool{"implement-issue": true})
		want := []string{
			`transcript does not name the expected handoff "implement-issue"`,
			`transcript does not contain "implement-issue"`,
			`transcript does not contain "missing-pattern"`,
			`transcript contains forbidden "ghp_"`,
			"expected file nope.txt does not exist",
			"forbidden file out.txt exists",
			`file src/app.go does not contain "Absent"`,
			"out-of-scope file src/app.go was modified",
		}
		if !reflect.DeepEqual(failures, want) {
			t.Fatalf("failures = %v, want %v", failures, want)
		}
	})

	t.Run("command exit mismatch fails and reports output", func(t *testing.T) {
		sc := &Scenario{Expectations: Expectations{
			CommandRun: []CommandRun{{Run: "exit 3", Dir: "", Exit: 0}},
		}}
		failures := evaluateAssertions(context.Background(), sc, "", sandbox, nil, nil)
		if len(failures) == 0 {
			t.Fatal("expected command exit mismatch to fail")
		}
	})

	t.Run("transcript must-any alternatives", func(t *testing.T) {
		for _, transcript := range []string{"found zero mandatory rules", "no enforcement gaps found"} {
			sc := &Scenario{Expectations: Expectations{
				TranscriptMustAny: []string{"no mandatory", "zero mandatory", "no enforcement gaps"},
			}}
			if failures := evaluateAssertions(context.Background(), sc, transcript, sandbox, nil, nil); len(failures) != 0 {
				t.Fatalf("transcript %q should match an alternative, got %v", transcript, failures)
			}
		}
		sc := &Scenario{Expectations: Expectations{
			TranscriptMustAny: []string{"no mandatory", "zero mandatory", "no enforcement gaps"},
		}}
		failures := evaluateAssertions(context.Background(), sc, "all rules are documented only", sandbox, nil, nil)
		if len(failures) == 0 || !strings.Contains(failures[0], "none of the alternatives") {
			t.Fatalf("expected unmatched-alternatives failure, got %v", failures)
		}
	})

	t.Run("unchanged file absent before is vacuous", func(t *testing.T) {
		sc := &Scenario{Expectations: Expectations{UnchangedFiles: []string{"never-existed.go"}}}
		failures := evaluateAssertions(context.Background(), sc, "", sandbox, nil, nil)
		if len(failures) != 0 {
			t.Fatalf("expected vacuous pass, got %v", failures)
		}
	})

	t.Run("handoff names a cataloged skill as transcript assertion", func(t *testing.T) {
		names := map[string]bool{"create-pr": true}
		sc := &Scenario{Expectations: Expectations{Handoff: "create-pr"}}
		failures := evaluateAssertions(context.Background(), sc, "implemented, stopping before the pull request", sandbox, nil, names)
		if len(failures) == 0 || !strings.Contains(failures[0], `does not name the expected handoff "create-pr"`) {
			t.Fatalf("expected handoff failure, got %v", failures)
		}
		failures = evaluateAssertions(context.Background(), sc, "handing the branch to create-pr", sandbox, nil, names)
		if len(failures) != 0 {
			t.Fatalf("expected pass when the next owner is named, got %v", failures)
		}
	})

	t.Run("outcome marker handoffs stay expectation-driven", func(t *testing.T) {
		sc := &Scenario{Expectations: Expectations{Handoff: "blocked-ask"}}
		names := map[string]bool{"blocked-ask": false}
		if failures := evaluateAssertions(context.Background(), sc, "stopping and asking the requester", sandbox, nil, names); len(failures) != 0 {
			t.Fatalf("marker handoff must not be asserted, got %v", failures)
		}
	})
}

func TestSnapshotHashes(t *testing.T) {
	sandbox := t.TempDir()
	writeTestFile(t, sandbox, "a.txt", "aaa")
	writeTestFile(t, sandbox, "b.txt", "bbb")
	hashes := snapshotHashes(sandbox, []string{"a.txt", "missing.txt"})
	if len(hashes) != 1 {
		t.Fatalf("expected only existing file hashed, got %v", hashes)
	}
	if _, ok := hashes["a.txt"]; !ok {
		t.Fatalf("missing expected hash: %v", hashes)
	}
}
