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

func TestVerificationDiagnosticForResultPreservesPhaseAndCategory(t *testing.T) {
	item, err := VerificationDiagnosticForResult(VerificationResult{ExitCode: 1, Spec: "specs/branch-flow.fsl", Phase: "check"})
	if err != nil {
		t.Fatal(err)
	}
	if item.Category != diagnostic.ValidationFailure || item.Code != VerificationValidationDiagnosticCode || item.Retryable {
		t.Fatalf("unexpected validation diagnostic: %#v", item)
	}
	if len(item.Evidence) != 1 || item.Evidence[0].Ref != "specs/branch-flow.fsl" {
		t.Fatalf("missing specification evidence: %#v", item.Evidence)
	}
}
