package fsl

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const sampleMutationOutput = `Mutating specs/branch-flow.fsl at depth 8
{
  "fsl": "1.0",
  "result": "mutated",
  "spec": "BranchFlow",
  "depth": 8,
  "baseline": "verified",
  "mutants": [
    {"op": "assignment_remove", "loc": {"line": 5, "column": 10}, "target": "init assignment", "status": "killed", "killed_by": "_bounds_status"},
    {"op": "enum_constant_swap", "loc": {"line": 5, "column": 10}, "target": "init assignment Draft->IssueCreated", "status": "survived"},
    {"op": "requires_remove", "loc": {"line": 6, "column": 27}, "target": "create_issue requires #1", "status": "killed", "killed_by": "MergedOnlyByIssuePullRequest"},
    {"op": "assignment_remove", "loc": {"line": 6, "column": 40}, "target": "transition guard", "status": "survived"}
  ],
  "summary": {"total": 4, "killed": 2, "survived": 2, "invalid": 0, "kill_rate": 0.5}
}
Mutated 1 FSL spec(s).
`

func TestParseMutationDocumentsCollectsSurvivors(t *testing.T) {
	specs, err := parseMutationDocuments(sampleMutationOutput)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 1 {
		t.Fatalf("expected 1 spec report, got %d", len(specs))
	}
	report := specs[0]
	if report.Total != 4 || report.Killed != 2 || report.Survived != 2 || report.Invalid != 0 {
		t.Fatalf("unexpected totals: %+v", report)
	}
	want := []Survivor{
		{Op: "enum_constant_swap", Target: "init assignment Draft->IssueCreated", Line: 5, Column: 10},
		{Op: "assignment_remove", Target: "transition guard", Line: 6, Column: 40},
	}
	if !reflect.DeepEqual(report.Survivors, want) {
		t.Fatalf("unexpected survivors %+v, want %+v", report.Survivors, want)
	}
}

func TestParseMutationDocumentsMultipleSpecs(t *testing.T) {
	text := sampleMutationOutput + "\nMutating specs/review-flow.fsl at depth 8\n" +
		"{\"result\":\"mutated\",\"spec\":\"ReviewFlow\",\"mutants\":[{\"target\":\"x\",\"status\":\"killed\"}]," +
		"\"summary\":{\"total\":1,\"killed\":1,\"survived\":0,\"invalid\":0}}\n"
	specs, err := parseMutationDocuments(text)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 2 {
		t.Fatalf("expected 2 spec reports, got %d", len(specs))
	}
}

func TestParseMutationDocumentsFailsOnMissingDocument(t *testing.T) {
	text := "Mutating specs/branch-flow.fsl at depth 8\nSome output without JSON\n"
	if _, err := parseMutationDocuments(text); err == nil {
		t.Fatal("expected an error when a mutation header has no document")
	}
}

func TestWriteMutationReportWritesJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "report.json")
	report := MutationReport{
		Depth: "8",
		Specs: []SpecReport{{
			Spec: "specs/branch-flow.fsl", Status: "mutated",
			Total: 4, Killed: 2, Survived: 2,
			Survivors: []Survivor{{Op: "enum_constant_swap", Target: "x", Line: 5, Column: 10}},
		}},
	}
	report.Totals.Specs = 1
	report.Totals.Killed = 2
	report.Totals.Survived = 2
	if err := WriteMutationReport(path, report); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	for _, want := range []string{`"specs/branch-flow.fsl"`, `"killed": 2`, `"survived": 2`, `"infrastructure_errors": 0`} {
		if !strings.Contains(text, want) {
			t.Fatalf("report %q missing %q", text, want)
		}
	}
}

func TestIntFieldHandlesJSONVariants(t *testing.T) {
	for _, tc := range []struct {
		value any
		want  int
	}{
		{value: 3, want: 3},
		{value: float64(3), want: 3},
		{value: nil, want: 0},
	} {
		got, err := intField(map[string]any{"k": tc.value}, "k")
		if err != nil {
			t.Fatalf("intField(%v): %v", tc.value, err)
		}
		if got != tc.want {
			t.Fatalf("intField(%v) = %d, want %d", tc.value, got, tc.want)
		}
	}
	if _, err := intField(map[string]any{"k": "nope"}, "k"); err == nil {
		t.Fatal("expected an error for a non-integer value")
	}
}
