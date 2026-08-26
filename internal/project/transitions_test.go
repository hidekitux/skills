package project

import (
	"bytes"
	"errors"
	"fmt"
	"testing"
)

func TestSetIssueStatusRejectsUsageErrors(t *testing.T) {
	cfg := mustConfig(t)
	var out, errOut bytes.Buffer
	if code := SetIssueStatus(newFakeRunner(), cfg, "not-owner-name", 205, "Backlog", false, &out, &errOut); code != 2 {
		t.Fatalf("expected usage code 2, got %d", code)
	}
	if code := SetIssueStatus(newFakeRunner(), cfg, "hidekitux/skills", 0, "Backlog", false, &out, &errOut); code != 2 {
		t.Fatalf("expected usage code 2, got %d", code)
	}
	if code := SetIssueStatus(newFakeRunner(), cfg, "hidekitux/skills", 205, "Urgent", false, &out, &errOut); code != 2 {
		t.Fatalf("expected invalid status code 2, got %d", code)
	}
}

func TestSetIssueStatusDryRunDoesNotMutate(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "hidekitux"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "hidekitux"}, fieldListJSON)
	var out, errOut bytes.Buffer
	if code := SetIssueStatus(runner, cfg, "hidekitux/skills", 205, "Backlog", true, &out, &errOut); code != 0 {
		t.Fatalf("expected success, got %d", code)
	}
	if runner.called("project", "item-add") || runner.called("project", "field-set") {
		t.Fatal("dry-run must not mutate")
	}
}

func TestSetIssueStatusSetsStatusAndAddsItemOnce(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "hidekitux"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "hidekitux"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "hidekitux"}, `{"items":[]}`).
		respond([]string{"project", "item-add", "3", "--owner", "hidekitux",
			"--url", issueURL205}, fmt.Sprintf(`{"id":"ITEM_9","content":{"url":%q}}`, issueURL205)).
		respond([]string{"project", "field-set", "3", "--owner", "hidekitux",
			"--item-id", "ITEM_9", "--field-id", "F_STATUS", "--single-select-option-id", "O_BACKLOG"}, "")
	var out, errOut bytes.Buffer
	if code := SetIssueStatus(runner, cfg, "hidekitux/skills", 205, "Backlog", false, &out, &errOut); code != 0 {
		t.Fatalf("expected success, got %d (out=%s err=%s)", code, out.String(), errOut.String())
	}
	if !runner.called("project", "field-set", "3", "--owner", "hidekitux",
		"--item-id", "ITEM_9", "--field-id", "F_STATUS", "--single-select-option-id", "O_BACKLOG") {
		t.Fatal("expected Status mutation")
	}
}

func TestSetIssueStatusFailsLoudlyOnMissingAccess(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().fail([]string{"project", "list", "--owner", "hidekitux"},
		errors.New("gh project list --owner hidekitux: your authentication token is missing required scopes [read:project]"))
	var out, errOut bytes.Buffer
	if code := SetIssueStatus(runner, cfg, "hidekitux/skills", 205, "Backlog", false, &out, &errOut); code != 1 {
		t.Fatalf("expected loud failure code 1, got %d", code)
	}
}

func TestCheckIssueProjectFailSafeSkipsOnMissingAccess(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().fail([]string{"project", "list", "--owner", "hidekitux"},
		errors.New("gh project list --owner hidekitux: your authentication token is missing required scopes [read:project]"))
	var out, errOut bytes.Buffer
	if code := CheckIssueProject(runner, cfg, "hidekitux/skills", 205, &out, &errOut); code != 0 {
		t.Fatalf("expected fail-safe skip code 0, got %d", code)
	}
	if out.String() == "" {
		t.Fatal("expected an actionable skip diagnostic")
	}
}

func TestCheckIssueProjectRejectsInvalidContract(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "hidekitux"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "hidekitux"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "hidekitux"}, `{"items":[]}`)
	var out, errOut bytes.Buffer
	if code := CheckIssueProject(runner, cfg, "hidekitux/skills", 205, &out, &errOut); code != 1 {
		t.Fatalf("expected invalid contract code 1, got %d", code)
	}
}

func TestCheckIssueProjectAcceptsValidContract(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "hidekitux"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "hidekitux"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "hidekitux"}, itemListJSON("ITEM_1"))
	var out, errOut bytes.Buffer
	if code := CheckIssueProject(runner, cfg, "hidekitux/skills", 205, &out, &errOut); code != 0 {
		t.Fatalf("expected valid contract code 0, got %d: %s", code, errOut.String())
	}
}
