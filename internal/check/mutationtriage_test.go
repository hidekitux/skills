package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const validTriageDoc = `# Mutation survivor triage register

## Register

<!-- mutation-triage:start -->
` + "```json" + `
{"specs": [
  {"spec": "specs/branch-flow.fsl", "survivors": [
    {"op": "enum_constant_swap", "target": "init assignment Draft->IssueCreated", "line": 5, "column": 10, "disposition": "accepted", "reason": "equivalent: swapped target states satisfy the invariants"}
  ]},
  {"spec": "specs/release-gate.fsl", "survivors": []}
]}
` + "```" + `
<!-- mutation-triage:end -->
`

func writeTriageDoc(t *testing.T, root, content string) {
	t.Helper()
	dir := filepath.Join(root, "docs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "mutation-triage.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestMutationTriageAcceptsCoveredRegister(t *testing.T) {
	root := t.TempDir()
	writeTriageDoc(t, root, validTriageDoc)
	if code := runCheck(t, CheckMutationTriage, root); code != 0 {
		t.Fatalf("expected pass, got exit %d", code)
	}
}

func TestMutationTriageRejectsMissingFile(t *testing.T) {
	root := t.TempDir()
	if code := runCheck(t, CheckMutationTriage, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func TestMutationTriageRejectsMissingBlock(t *testing.T) {
	root := t.TempDir()
	writeTriageDoc(t, root, "# No register block here\n")
	if code := runCheck(t, CheckMutationTriage, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func TestMutationTriageRejectsUntriagedSurvivor(t *testing.T) {
	root := t.TempDir()
	doc := strings.Replace(validTriageDoc, `"disposition": "accepted"`, `"disposition": "needs-review"`, 1)
	writeTriageDoc(t, root, doc)
	if code := runCheck(t, CheckMutationTriage, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func TestMutationTriageRejectsUnknownDisposition(t *testing.T) {
	root := t.TempDir()
	doc := strings.Replace(validTriageDoc, `"disposition": "accepted"`, `"disposition": "maybe"`, 1)
	writeTriageDoc(t, root, doc)
	if code := runCheck(t, CheckMutationTriage, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func TestMutationTriageRejectsMissingReason(t *testing.T) {
	root := t.TempDir()
	doc := strings.Replace(validTriageDoc, `"reason": "equivalent: swapped target states satisfy the invariants"`, `"reason": ""`, 1)
	writeTriageDoc(t, root, doc)
	if code := runCheck(t, CheckMutationTriage, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func TestMutationTriageRejectsInvalidJSON(t *testing.T) {
	root := t.TempDir()
	doc := strings.Replace(validTriageDoc, `"specs"`, `"specs" : {`, 1)
	writeTriageDoc(t, root, doc)
	if code := runCheck(t, CheckMutationTriage, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func TestMutationTriageAcceptsFixPlanned(t *testing.T) {
	root := t.TempDir()
	doc := strings.Replace(validTriageDoc, `"disposition": "accepted"`, `"disposition": "fix-planned"`, 1)
	writeTriageDoc(t, root, doc)
	if code := runCheck(t, CheckMutationTriage, root); code != 0 {
		t.Fatalf("expected pass, got exit %d", code)
	}
}
