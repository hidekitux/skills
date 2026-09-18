package check

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixturePreserved is a complete preserved entry. A test that exercises one
// finding edits a single field of it.
const fixturePreserved = `schema_version: 1
contracts:
  - id: mise-task-names
    surface: The task names declared in mise.toml.
    classification: preserved
    owner: composition
    evidence:
      - mise.toml
    reason: No task was renamed, added, or removed.
    check: check:tasks validates the names and references.
`

// writeDecisionTree builds a repository root that holds the decision file, the
// architecture record, and the evidence path the fixture entries cite.
func writeDecisionTree(t *testing.T, decisions, record string) string {
	t.Helper()
	root := t.TempDir()
	for _, dir := range []string{"workflow", "docs"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "workflow", "contract-decisions.yml"), []byte(decisions), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "architecture.md"), []byte(record), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "mise.toml"), []byte("# fixture\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// runDecisions runs the check against a written tree and returns its status
// and the writer the status selects.
func runDecisions(t *testing.T, decisions, record string) (int, string) {
	t.Helper()
	root := writeDecisionTree(t, decisions, record)
	var out, errOut bytes.Buffer
	status := CheckContractDecisions(root, &out, &errOut)
	if status == 0 {
		return status, out.String()
	}
	return status, errOut.String()
}

func TestContractDecisionsAcceptsACompletePreservedEntry(t *testing.T) {
	status, output := runDecisions(t, fixturePreserved, "| mise-task-names | preserved |\n")
	if status != 0 {
		t.Fatalf("status = %d, want 0: %s", status, output)
	}
	if !strings.Contains(output, "1 contract(s), 1 preserved, 0 changed") {
		t.Fatalf("output does not report the counted classifications: %s", output)
	}
}

func TestContractDecisionsRejectsAnIncompleteChangeRecord(t *testing.T) {
	decisions := `schema_version: 1
contracts:
  - id: repository-check-list
    surface: The named checks cmd/check-repository prints.
    classification: changed
    owner: composition
    evidence:
      - mise.toml
    old: The command ran 21 checks.
    new: The command runs 23 checks.
    reason: The redesign enforces the recorded model in continuous integration.
`
	status, output := runDecisions(t, decisions, "| repository-check-list | changed |\n")
	if status != 1 {
		t.Fatalf("status = %d, want 1: %s", status, output)
	}
	for _, field := range []string{"confirmation", "migration", "compatibility", "validation"} {
		if !strings.Contains(output, "records no "+field) {
			t.Fatalf("output does not name the missing %s: %s", field, output)
		}
	}
}

func TestContractDecisionsRejectsAChangeFieldOnAPreservedEntry(t *testing.T) {
	decisions := strings.Replace(fixturePreserved,
		"    check: check:tasks validates the names and references.\n",
		"    check: check:tasks validates the names and references.\n    migration: A consumer updates the task name once.\n", 1)
	status, output := runDecisions(t, decisions, "| mise-task-names | preserved |\n")
	if status != 1 {
		t.Fatalf("status = %d, want 1: %s", status, output)
	}
	if !strings.Contains(output, "carries the change field migration") {
		t.Fatalf("output does not name the misplaced field: %s", output)
	}
}

func TestContractDecisionsRejectsAnEvidencePathThatDoesNotExist(t *testing.T) {
	decisions := strings.Replace(fixturePreserved, "      - mise.toml\n", "      - workflow/absent.yml\n", 1)
	status, output := runDecisions(t, decisions, "| mise-task-names | preserved |\n")
	if status != 1 {
		t.Fatalf("status = %d, want 1: %s", status, output)
	}
	if !strings.Contains(output, "cites workflow/absent.yml, which does not exist") {
		t.Fatalf("output does not name the absent evidence path: %s", output)
	}
}

func TestContractDecisionsRejectsAnEntryTheRecordDoesNotDescribe(t *testing.T) {
	status, output := runDecisions(t, fixturePreserved, "The record describes no contract.\n")
	if status != 1 {
		t.Fatalf("status = %d, want 1: %s", status, output)
	}
	if !strings.Contains(output, "does not appear in docs/architecture.md") {
		t.Fatalf("output does not report the drift: %s", output)
	}
}

func TestContractDecisionsRejectsAnUnapprovedVersionIdentifier(t *testing.T) {
	decisions := strings.Replace(fixturePreserved,
		"    reason: No task was renamed, added, or removed.\n",
		"    reason: The task names move to the v2 naming scheme.\n", 1)
	status, output := runDecisions(t, decisions, "| mise-task-names | preserved |\n")
	if status != 1 {
		t.Fatalf("status = %d, want 1: %s", status, output)
	}
	if !strings.Contains(output, "names the version identifier v2 without a version_decision") {
		t.Fatalf("output does not report the unapproved identifier: %s", output)
	}
}

func TestContractDecisionsAcceptsAPinnedToolVersion(t *testing.T) {
	decisions := strings.Replace(fixturePreserved,
		"    reason: No task was renamed, added, or removed.\n",
		"    reason: The pinned govulncheck v1.7.0 still resolves behind the task.\n", 1)
	status, output := runDecisions(t, decisions, "| mise-task-names | preserved |\n")
	if status != 0 {
		t.Fatalf("status = %d, want 0: %s", status, output)
	}
}

func TestContractDecisionsRejectsAnUnsupportedSchemaVersion(t *testing.T) {
	decisions := strings.Replace(fixturePreserved, "schema_version: 1\n", "schema_version: 2\n", 1)
	status, output := runDecisions(t, decisions, "| mise-task-names | preserved |\n")
	if status != 1 {
		t.Fatalf("status = %d, want 1: %s", status, output)
	}
	if !strings.Contains(output, "unsupported schema_version 2") {
		t.Fatalf("output does not name the unsupported version: %s", output)
	}
}

func TestContractDecisionsRejectsADuplicateIdentifier(t *testing.T) {
	decisions := fixturePreserved + strings.TrimPrefix(fixturePreserved, "schema_version: 1\ncontracts:\n")
	status, output := runDecisions(t, decisions, "| mise-task-names | preserved |\n")
	if status != 1 {
		t.Fatalf("status = %d, want 1: %s", status, output)
	}
	if !strings.Contains(output, "is listed more than once") {
		t.Fatalf("output does not report the duplicate: %s", output)
	}
}

func TestContractDecisionsValidatesTheCommittedFile(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	var out, errOut bytes.Buffer
	if status := CheckContractDecisions(root, &out, &errOut); status != 0 {
		t.Fatalf("status = %d, want 0: %s", status, errOut.String())
	}
	if !strings.Contains(out.String(), "contract decisions valid") {
		t.Fatalf("output does not report a valid result: %s", out.String())
	}
}

func TestContractDecisionsRejectsAMisspelledField(t *testing.T) {
	decisions := strings.Replace(fixturePreserved,
		"    check: check:tasks validates the names and references.\n",
		"    chek: check:tasks validates the names and references.\n", 1)
	status, output := runDecisions(t, decisions, "| mise-task-names | preserved |\n")
	if status != 1 {
		t.Fatalf("status = %d, want 1: %s", status, output)
	}
	if !strings.Contains(output, "cannot parse contract-decisions.yml") {
		t.Fatalf("output does not reject the unknown field: %s", output)
	}
}
