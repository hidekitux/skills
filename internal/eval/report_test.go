package eval

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRecordSerializesFailureProvenance(t *testing.T) {
	record := Record{
		RunID:             "run-1",
		Scenario:          "scenario-1",
		Skill:             "debug-code",
		Host:              "codex",
		Model:             "gpt-5.6-luna",
		Commit:            "094d89c2fee71fec5bd22141faf77773fd1e6f26",
		Verdict:           VerdictFail,
		FailureID:         "scope-boundary-001",
		FailureCause:      "model_behavior",
		FailureRecurrence: 2,
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	text := string(encoded)
	for _, field := range []string{"failure_id", "failure_cause", "failure_recurrence_count"} {
		if !strings.Contains(text, field) {
			t.Fatalf("serialized record omitted %s: %s", field, text)
		}
	}
}

func TestMarkdownSummaryIncludesFailureRecurrenceProvenance(t *testing.T) {
	record := Record{
		RunID:             "run-1",
		Scenario:          "scenario-1",
		Skill:             "debug-code",
		Host:              "codex",
		Model:             "gpt-5.6-luna",
		Commit:            "094d89c2fee71fec5bd22141faf77773fd1e6f26",
		Verdict:           VerdictFail,
		FailureID:         "scope-boundary-001",
		FailureCause:      "model_behavior",
		FailureRecurrence: 2,
	}
	var output bytes.Buffer
	markdownSummary(&output, []Record{record}, map[string]string{"scenario-1": VerdictFail}, record.Model, record.Commit)
	text := output.String()
	for _, value := range []string{"Failure recurrence", "scope-boundary-001", "model_behavior", "gpt-5.6-luna", "run-1", "2"} {
		if !strings.Contains(text, value) {
			t.Fatalf("summary omitted %q:\n%s", value, text)
		}
	}
}

func TestAttachFailureMetadataUsesPromotedRecord(t *testing.T) {
	root := scaffoldFailureRecord(t, validFailureRecord())
	record := Record{Scenario: "synthetic", Skill: "synthetic"}
	attachFailureMetadata(root, &record)
	if record.FailureID != "synthetic-failure-001" || record.FailureCause != "repository_design" || record.FailureRecurrence != 2 {
		t.Fatalf("failure metadata = %#v", record)
	}
}
