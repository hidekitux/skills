package project

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestPullRequestStatusCoversAllEventPatterns(t *testing.T) {
	cases := []struct {
		name   string
		event  PullRequestEvent
		status string
		change bool
	}{
		{"opened ready", PullRequestEvent{Type: "opened", Draft: false}, "In review", true},
		{"opened draft", PullRequestEvent{Type: "opened", Draft: true}, "In progress", true},
		{"reopened ready", PullRequestEvent{Type: "reopened", Draft: false}, "In review", true},
		{"reopened draft", PullRequestEvent{Type: "reopened", Draft: true}, "In progress", true},
		{"synchronize ready", PullRequestEvent{Type: "synchronize", Draft: false}, "In review", true},
		{"synchronize draft", PullRequestEvent{Type: "synchronize", Draft: true}, "In progress", true},
		{"ready for review", PullRequestEvent{Type: "ready_for_review"}, "In review", true},
		{"converted to draft", PullRequestEvent{Type: "converted_to_draft"}, "In progress", true},
		{"closed unmerged", PullRequestEvent{Type: "closed", Merged: false}, "In progress", true},
		{"closed merged", PullRequestEvent{Type: "closed", Merged: true}, "", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			status, change, err := PullRequestStatus(tc.event)
			if err != nil {
				t.Fatalf("PullRequestStatus: %v", err)
			}
			if status != tc.status || change != tc.change {
				t.Fatalf("got status=%q change=%v, want status=%q change=%v",
					status, change, tc.status, tc.change)
			}
		})
	}
}

func TestPullRequestStatusRejectsUnknownEventType(t *testing.T) {
	if _, _, err := PullRequestStatus(PullRequestEvent{Type: "edited"}); err == nil {
		t.Fatal("expected unsupported event type to fail")
	}
}

func TestSetIssueStatusRejectsUsageErrors(t *testing.T) {
	cfg := mustConfig(t)
	var out, errOut bytes.Buffer
	if code := SetIssueStatus(newFakeRunner(), cfg, "owner", 205, "Backlog", false, false, &out, &errOut); code != 2 {
		t.Fatalf("expected usage code 2 for malformed repo, got %d", code)
	}
	if code := SetIssueStatus(newFakeRunner(), cfg, "acme/sample", 0, "Backlog", false, false, &out, &errOut); code != 2 {
		t.Fatalf("expected usage code 2 for zero issue, got %d", code)
	}
	if code := SetIssueStatus(newFakeRunner(), cfg, "acme/sample", 205, "Urgent", false, false, &out, &errOut); code != 2 {
		t.Fatalf("expected usage code 2 for invalid status, got %d", code)
	}
}

func TestSetIssueStatusDryRunDoesNotMutate(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "acme", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "acme", "--format", "json"}, fieldListJSON)
	var out, errOut bytes.Buffer
	if code := SetIssueStatus(runner, cfg, "acme/sample", 205, "Backlog", true, false, &out, &errOut); code != 0 {
		t.Fatalf("expected success, got %d", code)
	}
	if runner.called("project", "item-add") || runner.called("project", "item-edit") {
		t.Fatal("dry-run must not mutate")
	}
}

func TestSetIssueStatusSetsStatusAndAddsItemOnce(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "acme", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "acme", "--format", "json"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "acme", "--limit", "100", "--format", "json"}, `{"items":[]}`).
		respond([]string{"project", "item-add", "3", "--owner", "acme",
			"--url", issueURL205, "--format", "json"}, fmt.Sprintf(`{"id":"ITEM_9","content":{"url":%q}}`, issueURL205)).
		respond([]string{"project", "item-edit", "--id", "ITEM_9", "--field-id", "F_STATUS",
			"--project-id", "PVT_1", "--single-select-option-id", "O_BACKLOG"}, "")
	var out, errOut bytes.Buffer
	if code := SetIssueStatus(runner, cfg, "acme/sample", 205, "Backlog", false, false, &out, &errOut); code != 0 {
		t.Fatalf("expected success, got %d (out=%s err=%s)", code, out.String(), errOut.String())
	}
	if !runner.called("project", "item-edit", "--id", "ITEM_9", "--field-id", "F_STATUS",
		"--project-id", "PVT_1", "--single-select-option-id", "O_BACKLOG") {
		t.Fatal("expected Status mutation")
	}
}

func TestSetIssueStatusDoesNotRegressPlannedFromLaterLifecycleState(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "acme", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "acme", "--format", "json"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "acme", "--limit", "100", "--format", "json"}, itemListJSON("ITEM_1", "O_INPROGRESS", "O_MEDIUM", "O_IMPROV"))
	var out, errOut bytes.Buffer
	if code := SetIssueStatus(runner, cfg, "acme/sample", 205, "Planned", false, false, &out, &errOut); code != 0 {
		t.Fatalf("expected success, got %d (out=%s err=%s)", code, out.String(), errOut.String())
	}
	if runner.called("project", "item-edit") {
		t.Fatal("Planned must not regress an In progress item")
	}
	if !strings.Contains(out.String(), "refusing to regress") {
		t.Fatalf("expected regression diagnostic, got %q", out.String())
	}
}

func TestSetIssueStatusDoesNotMutateAlreadyPlannedItem(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "acme", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "acme", "--format", "json"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "acme", "--limit", "100", "--format", "json"}, itemListJSON("ITEM_1", "O_PLANNED", "O_MEDIUM", "O_IMPROV"))
	var out, errOut bytes.Buffer
	if code := SetIssueStatus(runner, cfg, "acme/sample", 205, "Planned", false, false, &out, &errOut); code != 0 {
		t.Fatalf("expected success, got %d (out=%s err=%s)", code, out.String(), errOut.String())
	}
	if runner.called("project", "item-edit") {
		t.Fatal("already Planned item must not be mutated")
	}
}

func TestSetIssueStatusSkipsClosedIssue(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().respond(
		[]string{"issue", "view", "205", "--repo", "acme/sample", "--json", "state"},
		`{"state":"CLOSED"}`)
	var out, errOut bytes.Buffer
	if code := SetIssueStatus(runner, cfg, "acme/sample", 205, "Backlog", false, true, &out, &errOut); code != 0 {
		t.Fatalf("expected success, got %d", code)
	}
	if runner.called("project", "list") || runner.called("project", "item-edit") {
		t.Fatal("closed Issue must not be mutated")
	}
}

func TestSetIssueStatusSurfacesIssueViewFailure(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().fail(
		[]string{"issue", "view", "205", "--repo", "acme/sample", "--json", "state"},
		errors.New("gh issue view: boom"))
	var out, errOut bytes.Buffer
	if code := SetIssueStatus(runner, cfg, "acme/sample", 205, "Backlog", false, true, &out, &errOut); code != 1 {
		t.Fatalf("expected loud failure code 1, got %d", code)
	}
}

func TestSetIssueStatusFailsLoudlyOnMissingAccess(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().fail([]string{"project", "list", "--owner", "acme", "--format", "json"},
		errors.New("gh project list --owner acme: your authentication token is missing required scopes [read:project]"))
	var out, errOut bytes.Buffer
	if code := SetIssueStatus(runner, cfg, "acme/sample", 205, "Backlog", false, false, &out, &errOut); code != 1 {
		t.Fatalf("expected loud failure code 1, got %d", code)
	}
}
