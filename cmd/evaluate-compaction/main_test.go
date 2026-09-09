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
