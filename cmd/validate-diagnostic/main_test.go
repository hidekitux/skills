package main

import (
	"testing"

	"github.com/hidekitux/skills/internal/diagnostic"
)

func TestDiagnosticValidationReportAcceptsValidJSONL(t *testing.T) {
	item := diagnostic.Diagnostic{
		SchemaVersion: diagnostic.SchemaVersion,
		Producer:      "fixture",
		Code:          "fixture.invalid",
		Category:      diagnostic.ValidationFailure,
		SourceCommand: "fixture-validator",
		Message:       "fixture failed",
		Retryable:     false,
		Remediation:   diagnostic.InspectInput,
		Redaction:     diagnostic.RedactionSummary{Mode: "allowlist", OmittedFields: []string{}},
	}
	encoded, err := diagnostic.Marshal(item)
	if err != nil {
		t.Fatal(err)
	}
	report := diagnostic.ValidateJSONL(encoded)
	if !report.Valid || report.DiagnosticCount != 1 {
		t.Fatalf("unexpected report: %#v", report)
	}
}
