package main

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/eval"
)

func TestRunCompactionSourceRequiresCurrentReport(t *testing.T) {
	reportDir := t.TempDir()
	stalePath := filepath.Join(reportDir, "99999999999999-full.jsonl")
	if err := os.WriteFile(stalePath, []byte("{\"run_id\":\"stale-full\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := &eval.Options{
		OutputDir:          reportDir,
		RunID:              "current-full",
		InstructionVariant: "full",
	}
	runner := func(context.Context, *eval.Options, io.Writer, io.Writer) int {
		return eval.ExitInfra
	}

	code, reportPath, err := runCompactionSource(context.Background(), opts, runner)
	if code != eval.ExitInfra {
		t.Fatalf("exit = %d, want %d", code, eval.ExitInfra)
	}
	if reportPath != "" {
		t.Fatalf("report path = %q, want empty", reportPath)
	}
	if err == nil || !strings.Contains(err.Error(), "current report") {
		t.Fatalf("error = %v, want current report error", err)
	}
}

func TestRunCompactionSourceSelectsExactCurrentReport(t *testing.T) {
	reportDir := t.TempDir()
	stalePath := filepath.Join(reportDir, "99999999999999-full.jsonl")
	currentPath := filepath.Join(reportDir, "current-full.jsonl")
	for path, runID := range map[string]string{stalePath: "stale-full", currentPath: "current-full"} {
		if err := os.WriteFile(path, []byte("{\"run_id\":\""+runID+"\"}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	opts := &eval.Options{
		OutputDir:          reportDir,
		RunID:              "current-full",
		InstructionVariant: "full",
	}
	runner := func(context.Context, *eval.Options, io.Writer, io.Writer) int {
		return eval.ExitOK
	}

	code, reportPath, err := runCompactionSource(context.Background(), opts, runner)
	if err != nil {
		t.Fatal(err)
	}
	if code != eval.ExitOK {
		t.Fatalf("exit = %d, want %d", code, eval.ExitOK)
	}
	if reportPath != currentPath {
		t.Fatalf("report path = %q, want %q", reportPath, currentPath)
	}
}

func TestRunCompactionSourceRejectsReportFromAnotherRun(t *testing.T) {
	reportDir := t.TempDir()
	reportPath := filepath.Join(reportDir, "current-full.jsonl")
	if err := os.WriteFile(reportPath, []byte("{\"run_id\":\"stale-full\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	opts := &eval.Options{
		OutputDir:          reportDir,
		RunID:              "current-full",
		InstructionVariant: "full",
	}
	runner := func(context.Context, *eval.Options, io.Writer, io.Writer) int {
		return eval.ExitOK
	}

	code, report, err := runCompactionSource(context.Background(), opts, runner)
	if code != eval.ExitOK {
		t.Fatalf("exit = %d, want %d", code, eval.ExitOK)
	}
	if report != "" {
		t.Fatalf("report path = %q, want empty", report)
	}
	if err == nil || !strings.Contains(err.Error(), "has run ID") {
		t.Fatalf("error = %v, want run ID error", err)
	}
}

func TestCompactionCommandUsesPairedReportContract(t *testing.T) {
	report := eval.PairReport{Encoding: "cl100k_base", Results: []eval.PairResult{{Status: "inconclusive"}}}
	if report.Encoding != "cl100k_base" || len(report.Results) != 1 {
		t.Fatalf("paired report contract = %#v", report)
	}
}

func TestCompactionExitCodeUsesPairedResults(t *testing.T) {
	tests := []struct {
		name                  string
		fullCode, compactCode int
		results               []eval.PairResult
		want                  int
	}{
		{
			name:     "common baseline assertion does not block",
			fullCode: eval.ExitAssertion, compactCode: eval.ExitAssertion,
			results: []eval.PairResult{{Status: "pass"}}, want: eval.ExitOK,
		},
		{
			name:     "paired regression blocks",
			fullCode: eval.ExitAssertion, compactCode: eval.ExitAssertion,
			results: []eval.PairResult{{Status: "fail"}}, want: eval.ExitAssertion,
		},
		{
			name:    "incomplete comparison is infrastructure",
			results: []eval.PairResult{{Status: "inconclusive"}}, want: eval.ExitInfra,
		},
		{
			name:        "full source interruption remains infrastructure",
			fullCode:    eval.ExitInfra,
			compactCode: eval.ExitOK,
			results:     []eval.PairResult{{Status: "pass"}},
			want:        eval.ExitInfra,
		},
		{
			name:        "compact source interruption remains infrastructure",
			fullCode:    eval.ExitOK,
			compactCode: eval.ExitInfra,
			results:     []eval.PairResult{{Status: "pass"}},
			want:        eval.ExitInfra,
		},
		{
			name:     "usage remains usage",
			fullCode: eval.ExitUsage, results: []eval.PairResult{{Status: "pass"}}, want: eval.ExitUsage,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := compactionExitCode(test.fullCode, test.compactCode, test.results); got != test.want {
				t.Fatalf("compactionExitCode() = %d, want %d", got, test.want)
			}
		})
	}
}
