package project

import (
	"bytes"
	"errors"
	"testing"
)

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
