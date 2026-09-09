package diagnostic

import (
	"encoding/json"
	"strings"
	"testing"
)

const diagnosticSHA = "0123456789abcdef0123456789abcdef01234567"

func validDiagnostic() Diagnostic {
	return Diagnostic{
		SchemaVersion: SchemaVersion,
		Producer:      "validate-issue-body",
		Code:          "validate-issue-body.missing-heading",
		Category:      ValidationFailure,
		SourceCommand: "cmd/validate-issue-body",
		Message:       "required heading is missing",
		Location:      &Location{Path: "issues/202.md", Line: 4, Column: 1},
		Rule:          "issue_body.required_headings",
		Expected:      "Context, Goal, Scope",
		Observed:      "Context, Scope",
		Evidence: []Evidence{
			{Kind: "path", Ref: "issues/202.md"},
			{Kind: "commit", Ref: diagnosticSHA},
		},
		Retryable:   false,
		Remediation: FixRepository,
		Redaction:   RedactionSummary{Mode: "allowlist", OmittedFields: []string{}},
	}
}

func TestNewAcceptsDiagnosticAndRendersAllSemanticFields(t *testing.T) {
	diagnostic, err := New(validDiagnostic())
	if err != nil {
		t.Fatal(err)
	}
	text := RenderText(diagnostic)
	for _, want := range []string{
		"validate-issue-body/validate-issue-body.missing-heading",
		"validation_failure",
		"cmd/validate-issue-body",
		"location=issues/202.md:4:1",
		"rule=issue_body.required_headings",
		"expected=Context, Goal, Scope",
		"observed=Context, Scope",
		"evidence=path:issues/202.md,commit:" + diagnosticSHA,
		"retryable=false",
		"remediation=fix_repository",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("rendering missing %q: %s", want, text)
		}
	}
	encoded, err := Marshal(diagnostic)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Diagnostic
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Code != diagnostic.Code || decoded.Observed != diagnostic.Observed {
		t.Fatalf("structured rendering changed semantic facts: %#v", decoded)
	}
}

func TestValidateDistinguishesCategories(t *testing.T) {
	categories := []Category{ValidationFailure, InfrastructureError, UnsupportedInput, IncompleteEvidence}
	for _, category := range categories {
		t.Run(string(category), func(t *testing.T) {
			diagnostic := validDiagnostic()
			diagnostic.Category = category
			if _, err := New(diagnostic); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSanitizeRedactsSecretsAndOmitsUnsafeEvidence(t *testing.T) {
	diagnostic := validDiagnostic()
	diagnostic.Message = "token=ghp_example_secret at https://private.example/run/1"
	diagnostic.Expected = "password=secret-value"
	diagnostic.Location = &Location{Path: "reports/token=secret.txt"}
	diagnostic.Evidence = append(diagnostic.Evidence, Evidence{Kind: "issue", Ref: "https://github.com/hidekitux/skills/issues/202?token=secret"})
	clean, err := Sanitize(diagnostic)
	if err != nil {
		t.Fatal(err)
	}
	joined, _ := json.Marshal(clean)
	for _, secret := range []string{"ghp_example_secret", "secret-value", "private.example", "?token=secret"} {
		if strings.Contains(string(joined), secret) {
			t.Fatalf("unsafe value survived sanitization: %q in %s", secret, joined)
		}
	}
	if len(clean.Evidence) != 2 {
		t.Fatalf("unsafe evidence was not omitted: %#v", clean.Evidence)
	}
	if clean.Redaction.RedactedCount < 4 || len(clean.Redaction.OmittedFields) != 2 || clean.Redaction.OmittedFields[0] != "evidence[2]" || clean.Redaction.OmittedFields[1] != "location" {
		t.Fatalf("unexpected redaction summary: %#v", clean.Redaction)
	}
	if _, err := New(clean); err != nil {
		t.Fatal(err)
	}
}

func TestSanitizeRejectsUnsafeIdentity(t *testing.T) {
	diagnostic := validDiagnostic()
	diagnostic.Code = "bad token=secret"
	if _, err := Sanitize(diagnostic); err == nil {
		t.Fatal("unsafe stable identity was accepted")
	}
}

func TestReadJSONLRejectsUnknownFieldsAndRequiresRecords(t *testing.T) {
	encoded, err := Marshal(validDiagnostic())
	if err != nil {
		t.Fatal(err)
	}
	if diagnostics, err := ReadJSONL(encoded); err != nil || len(diagnostics) != 1 {
		t.Fatalf("valid JSONL rejected: %d diagnostics, %v", len(diagnostics), err)
	}
	if _, err := ReadJSONL([]byte(`{"schema_version":1,"producer":"x","code":"x","category":"validation_failure","source_command":"x","message":"x","retryable":false,"remediation":"inspect_input","redaction":{"mode":"allowlist","redacted_count":0,"omitted_fields":[]},"extra":true}`)); err == nil {
		t.Fatal("unknown field was accepted")
	}
	if _, err := ReadJSONL([]byte("\n")); err == nil {
		t.Fatal("empty stream was accepted")
	}
}

func TestValidateJSONLRejectsUnsafePersistedText(t *testing.T) {
	encoded := []byte(`{"schema_version":1,"producer":"fixture","code":"fixture.invalid","category":"validation_failure","source_command":"fixture-validator","message":"private https://internal.example/run","retryable":false,"remediation":"inspect_input","redaction":{"mode":"allowlist","redacted_count":0,"omitted_fields":[]}}`)
	report := ValidateJSONL(encoded)
	if report.Valid || !strings.Contains(strings.Join(report.Findings, "\n"), "unsafe text") {
		t.Fatalf("unsafe text was accepted: %#v", report)
	}
}

func TestSelectUsesStructuredFields(t *testing.T) {
	first := validDiagnostic()
	second := validDiagnostic()
	second.Code = "fsl.verify.infrastructure"
	second.Producer = "fsl"
	second.Category = InfrastructureError
	second.Retryable = true
	second.Remediation = RetryOperation
	second.Rule = "fslc.available"
	selected := Select([]Diagnostic{first, second}, Filter{Categories: []Category{InfrastructureError}, Retryable: boolPtr(true)})
	if len(selected) != 1 || selected[0].Code != second.Code {
		t.Fatalf("structured selection returned %#v", selected)
	}
}

func TestSelectUsesStableCodeAcrossProseChanges(t *testing.T) {
	first := validDiagnostic()
	second := first
	second.Message = "the required heading is still missing"
	selected := Select([]Diagnostic{first, second}, Filter{Codes: []string{first.Code}})
	if len(selected) != 2 {
		t.Fatalf("prose change altered structured selection: %#v", selected)
	}
}

func boolPtr(value bool) *bool { return &value }
