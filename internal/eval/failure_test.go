package eval

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func validFailureRecord() failureRecord {
	return failureRecord{
		SchemaVersion:  1,
		ID:             "synthetic-failure-001",
		Classification: "repository_design",
		Status:         "promoted",
		Summary:        "A required evidence field is missing.",
		Reproduction: failureReproduction{
			Kind:      "test_fixture",
			Fixture:   "evaluations/fixtures/failure",
			Steps:     []string{"Load the fixture.", "Run the check."},
			Sanitized: true,
		},
		ExpectedOutcome: "The check rejects the missing evidence.",
		Owner:           failureOwner{Layer: "static_check", Name: "check-evaluation.failure-records"},
		Evidence: []failureEvidence{
			{Kind: "path", Ref: "internal/eval/failure_test.go"},
		},
		RegressionAsset:   "scenario:synthetic.yaml",
		InstructionAction: "replace",
		InstructionRef:    "docs/evaluation.md#What-counts-as-evidence",
		DecisionReason:    "The invariant is fixed repository metadata.",
		Observations: []failureObservation{
			{Host: "go-test", Model: "go1.26.6", Skill: "check-evaluation", RepositoryRevision: "094d89c2fee71fec5bd22141faf77773fd1e6f26", RunID: "run-1", Outcome: "deterministic_failure"},
			{Host: "go-test", Model: "go1.26.6", Skill: "check-evaluation", RepositoryRevision: "094d89c2fee71fec5bd22141faf77773fd1e6f26", RunID: "run-2", Outcome: "deterministic_failure"},
		},
		RecurrenceCount: 2,
	}
}

func scaffoldFailureRecord(t *testing.T, record failureRecord) string {
	t.Helper()
	root := t.TempDir()
	writeTestFile(t, root, "workflow/failure-record.schema.json", "{}\n")
	writeTestFile(t, root, "evaluations/fixtures/failure/README.md", "synthetic fixture\n")
	writeTestFile(t, root, "evaluations/scenarios/synthetic.yaml", "synthetic scenario\n")
	writeTestFile(t, root, "docs/evaluation.md", "evaluation guidance\n")
	writeTestFile(t, root, "internal/eval/failure_test.go", "test evidence\n")
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, root, "workflow/failure-records/records.jsonl", string(encoded)+"\n")
	return root
}

func runFailureRecordCheck(t *testing.T, root string) (int, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := CheckFailureRecords(root, &out, &errOut)
	return code, out.String() + errOut.String()
}

func TestCheckFailureRecordsAcceptsValidPromotedRecord(t *testing.T) {
	root := scaffoldFailureRecord(t, validFailureRecord())
	code, output := runFailureRecordCheck(t, root)
	if code != 0 {
		t.Fatalf("expected valid record, got %d: %s", code, output)
	}
}

func TestCheckFailureRecordsRejectsUnsanitizedPromotedRecord(t *testing.T) {
	record := validFailureRecord()
	record.Reproduction.Sanitized = false
	root := scaffoldFailureRecord(t, record)
	code, output := runFailureRecordCheck(t, root)
	if code != 1 || !strings.Contains(output, "sanitized fixture") {
		t.Fatalf("expected unsanitized record to fail, got %d: %s", code, output)
	}
}

func TestCheckFailureRecordsRejectsPromotedRecordWithoutEvidence(t *testing.T) {
	record := validFailureRecord()
	record.Evidence = nil
	root := scaffoldFailureRecord(t, record)
	code, output := runFailureRecordCheck(t, root)
	if code != 1 || !strings.Contains(output, "evidence must not be empty") {
		t.Fatalf("expected missing evidence to fail, got %d: %s", code, output)
	}
}

func TestCheckFailureRecordsRejectsInfrastructurePromotion(t *testing.T) {
	record := validFailureRecord()
	record.Classification = "infrastructure"
	root := scaffoldFailureRecord(t, record)
	code, output := runFailureRecordCheck(t, root)
	if code != 1 || !strings.Contains(output, "cannot be promoted") {
		t.Fatalf("expected infrastructure promotion to fail, got %d: %s", code, output)
	}
}

func TestCheckFailureRecordsRejectsUnmeasuredRecurrence(t *testing.T) {
	record := validFailureRecord()
	record.RecurrenceCount = 1
	root := scaffoldFailureRecord(t, record)
	code, output := runFailureRecordCheck(t, root)
	if code != 1 || !strings.Contains(output, "recurrence_count") {
		t.Fatalf("expected recurrence mismatch to fail, got %d: %s", code, output)
	}
}

func TestCheckFailureRecordsAllowsUnpromotedInfrastructure(t *testing.T) {
	record := validFailureRecord()
	record.Classification = "infrastructure"
	record.Status = "not_promoted"
	record.Owner = failureOwner{Layer: "none", Name: "evaluation-operator"}
	record.RegressionAsset = ""
	record.InstructionAction = "none"
	record.InstructionRef = ""
	record.DecisionReason = "The host is unavailable outside the repository."
	record.Observations = record.Observations[:1]
	record.RecurrenceCount = 1
	root := scaffoldFailureRecord(t, record)
	code, output := runFailureRecordCheck(t, root)
	if code != 0 {
		t.Fatalf("expected unpromoted infrastructure record, got %d: %s", code, output)
	}
}

func TestFailureRecordFixtureIsNotOutsideRoot(t *testing.T) {
	record := validFailureRecord()
	record.Reproduction.Fixture = filepath.Join("..", "outside")
	root := scaffoldFailureRecord(t, record)
	code, output := runFailureRecordCheck(t, root)
	if code != 1 || !strings.Contains(output, "does not resolve") {
		t.Fatalf("expected outside fixture to fail, got %d: %s", code, output)
	}
	if _, err := os.Stat(filepath.Join(root, record.Reproduction.Fixture)); err == nil {
		t.Fatal("test fixture unexpectedly created outside root")
	}
}
