package main

import (
	"testing"

	"github.com/hidekitux/skills/internal/eval"
)

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
