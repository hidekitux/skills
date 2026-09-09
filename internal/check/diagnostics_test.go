package check

import (
	"testing"

	"github.com/hidekitux/skills/internal/diagnostic"
)

func TestDiagnosticForResultUsesStableCheckIdentity(t *testing.T) {
	item, err := DiagnosticForResult("check-writing-quality", 1)
	if err != nil {
		t.Fatal(err)
	}
	if item.Producer != "check-repository" || item.Code != "check-repository.check-writing-quality.failed" {
		t.Fatalf("unexpected identity: %#v", item)
	}
	if item.Category != diagnostic.ValidationFailure || item.Observed != "check-writing-quality exit_code=1" {
		t.Fatalf("unexpected semantics: %#v", item)
	}
}
