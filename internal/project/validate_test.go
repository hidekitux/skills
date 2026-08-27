package project

import (
	"bytes"
	"errors"
	"testing"
)

func TestCheckIssueProjectFailSafeSkipsOnMissingAccess(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().fail([]string{"project", "list", "--owner", "hidekitux", "--format", "json"},
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
		respond([]string{"project", "list", "--owner", "hidekitux", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "hidekitux", "--format", "json"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "hidekitux", "--limit", "100", "--format", "json"}, `{"items":[]}`)
	var out, errOut bytes.Buffer
	if code := CheckIssueProject(runner, cfg, "hidekitux/skills", 205, &out, &errOut); code != 1 {
		t.Fatalf("expected invalid contract code 1, got %d", code)
	}
}

func TestCheckIssueProjectAcceptsValidContract(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "hidekitux", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "hidekitux", "--format", "json"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "hidekitux", "--limit", "100", "--format", "json"}, itemListJSON("ITEM_1"))
	var out, errOut bytes.Buffer
	if code := CheckIssueProject(runner, cfg, "hidekitux/skills", 205, &out, &errOut); code != 0 {
		t.Fatalf("expected valid contract code 0, got %d: %s", code, errOut.String())
	}
}
