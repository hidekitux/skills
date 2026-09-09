package fsl

import (
	"testing"

	"github.com/hidekitux/skills/internal/diagnostic"
)

func TestDiagnosticsForReportDistinguishesSurvivorAndInfrastructure(t *testing.T) {
	report := MutationReport{Specs: []SpecReport{
		{Spec: "specs/review-flow.fsl", Status: "mutated", Survived: 2},
		{Spec: "specs/branch-flow.fsl", Status: "error"},
	}}
	diagnostics, err := DiagnosticsForReport(report)
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) != 2 {
		t.Fatalf("got %d diagnostics: %#v", len(diagnostics), diagnostics)
	}
	if diagnostics[0].Category != diagnostic.ValidationFailure || diagnostics[1].Category != diagnostic.InfrastructureError {
		t.Fatalf("categories=%s,%s", diagnostics[0].Category, diagnostics[1].Category)
	}
}
