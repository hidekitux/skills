package validate

import (
	"strings"
	"testing"

	"github.com/hidekitux/skills/internal/diagnostic"
)

func TestIssueBodyDiagnosticsExposeStableStructuredFacts(t *testing.T) {
	diagnostics, err := IssueBodyDiagnostics("[Improvement]: invalid", "## Goal\nmissing other sections")
	if err != nil {
		t.Fatal(err)
	}
	if len(diagnostics) == 0 {
		t.Fatal("expected diagnostics")
	}
	item := diagnostics[0]
	if item.Producer != "validate-issue-body" || item.Code != IssueBodyDiagnosticCode || item.Category != diagnostic.ValidationFailure {
		t.Fatalf("unexpected diagnostic: %#v", item)
	}
	if strings.Contains(item.Message, "[Improvement]") {
		t.Fatalf("diagnostic leaked unrelated input: %#v", item)
	}
}

func TestBranchPolicyDiagnosticsClassifyUnsupportedInput(t *testing.T) {
	diagnostics, code, _, _ := BranchPolicyDiagnostics("missing.toml", "", "", "", false)
	if code != 2 || len(diagnostics) != 1 {
		t.Fatalf("unexpected result: code=%d diagnostics=%#v", code, diagnostics)
	}
	if diagnostics[0].Category != diagnostic.UnsupportedInput {
		t.Fatalf("category=%s", diagnostics[0].Category)
	}
}

func TestHostsDiagnosticsPreserveDetailedOutput(t *testing.T) {
	diagnostics, code, out, errOut := HostsDiagnostics("/path/that/does/not/exist")
	if code == 0 || len(diagnostics) != 1 {
		t.Fatalf("unexpected result: code=%d diagnostics=%#v", code, diagnostics)
	}
	if diagnostics[0].Category != diagnostic.InfrastructureError || !diagnostics[0].Retryable {
		t.Fatalf("unexpected diagnostic: %#v", diagnostics[0])
	}
	if out == "" && errOut == "" {
		t.Fatal("detailed host output was not preserved")
	}
}
