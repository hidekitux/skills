package project

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

// fakeRunner scripts gh responses by exact argument string, recording calls.
type fakeRunner struct {
	calls     [][]string
	responses map[string]string
	failures  map[string]error
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{responses: map[string]string{}, failures: map[string]error{}}
}

func (f *fakeRunner) respond(args []string, output string) *fakeRunner {
	f.responses[strings.Join(args, " ")] = output
	return f
}

func (f *fakeRunner) fail(args []string, err error) *fakeRunner {
	f.failures[strings.Join(args, " ")] = err
	return f
}

func (f *fakeRunner) Run(args ...string) (string, error) {
	f.calls = append(f.calls, args)
	key := strings.Join(args, " ")
	if err, ok := f.failures[key]; ok {
		return "", err
	}
	if output, ok := f.responses[key]; ok {
		return output, nil
	}
	return "", fmt.Errorf("unexpected gh call: %s", key)
}

func (f *fakeRunner) called(args ...string) bool {
	key := strings.Join(args, " ")
	for _, call := range f.calls {
		if strings.Join(call, " ") == key {
			return true
		}
	}
	return false
}

func mustConfig(t *testing.T) *Config {
	t.Helper()
	return loadTestConfig(t, validConfigTOML)
}

const projectListJSON = `{"totalCount":1,"projects":[{"number":3,"title":"Skills Issues","id":"PVT_1"}]}`

const fieldListJSON = `{"fields":[
  {"id":"F_STATUS","name":"Status","dataType":"SINGLE_SELECT","options":[
    {"id":"O_BACKLOG","name":"Backlog"},{"id":"O_PLANNED","name":"Planned"},
    {"id":"O_INPROGRESS","name":"In progress"},{"id":"O_INREVIEW","name":"In review"},
    {"id":"O_DONE","name":"Done"}]},
  {"id":"F_PRIORITY","name":"Priority","dataType":"SINGLE_SELECT","options":[
    {"id":"O_HIGH","name":"High"},{"id":"O_MEDIUM","name":"Medium"},{"id":"O_LOW","name":"Low"}]},
  {"id":"F_SCOPE","name":"Scope","dataType":"SINGLE_SELECT","options":[
    {"id":"O_FEATURE","name":"Feature"},{"id":"O_BUG","name":"Bug"},{"id":"O_DOCS","name":"Docs"},
    {"id":"O_MAINT","name":"Maintenance"},{"id":"O_IMPROV","name":"Improvement"},
    {"id":"O_SEC","name":"Security"},{"id":"O_REL","name":"Release"}]}
]}`

func itemListJSON(itemID string, values ...string) string {
	if len(values) == 0 {
		values = []string{"O_BACKLOG", "O_MEDIUM", "O_IMPROV"}
	}
	return fmt.Sprintf(`{"items":[{"id":%q,"content":{"url":"https://github.com/hidekitux/skills/issues/205"},
      "fieldValues":[
        {"field":{"id":"F_STATUS","name":"Status"},"optionId":%q},
        {"field":{"id":"F_PRIORITY","name":"Priority"},"optionId":%q},
        {"field":{"id":"F_SCOPE","name":"Scope"},"optionId":%q}]}]}`,
		itemID, values[0], values[1], values[2])
}

const issueURL205 = "https://github.com/hidekitux/skills/issues/205"

func defaultScriptedRunner(t *testing.T, itemJSON string) (*fakeRunner, *Client, *Config) {
	t.Helper()
	cfg := mustConfig(t)
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "hidekitux", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "hidekitux", "--format", "json"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "hidekitux", "--limit", "100", "--format", "json"}, itemJSON)
	return runner, NewClient(runner, cfg.Project.Owner), cfg
}

func TestProjectNumberPrefersConfiguredNumber(t *testing.T) {
	cfg := mustConfig(t)
	cfg.Project.Number = 7
	runner := newFakeRunner().respond(
		[]string{"project", "view", "7", "--owner", "hidekitux", "--format", "json"},
		`{"id":"PVT_7","number":7,"title":"Skills Issues"}`)
	client := NewClient(runner, cfg.Project.Owner)
	number, err := client.ProjectNumber(cfg)
	if err != nil {
		t.Fatalf("ProjectNumber: %v", err)
	}
	if number != 7 {
		t.Fatalf("expected configured number 7, got %d", number)
	}
	if !runner.called("project", "view", "7", "--owner", "hidekitux", "--format", "json") {
		t.Fatal("configured number must still resolve the node id")
	}
}

func TestProjectNumberResolvesUniqueTitle(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().respond(
		[]string{"project", "list", "--owner", "hidekitux", "--format", "json"}, projectListJSON)
	client := NewClient(runner, cfg.Project.Owner)
	number, err := client.ProjectNumber(cfg)
	if err != nil {
		t.Fatalf("ProjectNumber: %v", err)
	}
	if number != 3 {
		t.Fatalf("expected title-resolved number 3, got %d", number)
	}
}

func TestProjectNumberFailsOnMissingOrAmbiguousTitle(t *testing.T) {
	cfg := mustConfig(t)
	missing := newFakeRunner().respond([]string{"project", "list", "--owner", "hidekitux", "--format", "json"},
		`{"totalCount":1,"projects":[{"number":3,"title":"Other","id":"PVT_1"}]}`)
	if _, err := NewClient(missing, cfg.Project.Owner).ProjectNumber(cfg); err == nil {
		t.Fatal("expected missing title to fail")
	}
	ambiguous := newFakeRunner().respond([]string{"project", "list", "--owner", "hidekitux", "--format", "json"},
		`{"totalCount":2,"projects":[{"number":1,"title":"Skills Issues","id":"PVT_1"},{"number":2,"title":"Skills Issues","id":"PVT_2"}]}`)
	if _, err := NewClient(ambiguous, cfg.Project.Owner).ProjectNumber(cfg); err == nil {
		t.Fatal("expected ambiguous title to fail")
	}
}

func TestResolvedFieldsFailsBeforeMutationOnMissingFieldOrOption(t *testing.T) {
	cfg := mustConfig(t)
	missingField := newFakeRunner().respond(
		[]string{"project", "field-list", "3", "--owner", "hidekitux", "--format", "json"},
		`{"fields":[{"id":"F_STATUS","name":"Phase","dataType":"SINGLE_SELECT","options":[]}]}`)
	if _, err := NewClient(missingField, cfg.Project.Owner).resolvedFields(cfg, 3); err == nil {
		t.Fatal("expected missing declared field to fail")
	}
	missingOption := newFakeRunner().respond(
		[]string{"project", "field-list", "3", "--owner", "hidekitux", "--format", "json"},
		`{"fields":[{"id":"F_STATUS","name":"Status","dataType":"SINGLE_SELECT","options":[
		   {"id":"O_BACKLOG","name":"Backlog"}]},
		  {"id":"F_PRIORITY","name":"Priority","dataType":"SINGLE_SELECT","options":[
		   {"id":"O_MEDIUM","name":"Medium"}]},
		  {"id":"F_SCOPE","name":"Scope","dataType":"SINGLE_SELECT","options":[
		   {"id":"O_IMPROV","name":"Improvement"}]}]}`)
	if _, err := NewClient(missingOption, cfg.Project.Owner).resolvedFields(cfg, 3); err == nil {
		t.Fatal("expected missing declared option to fail")
	}
}

func TestAddItemReusesExistingItemWithoutDuplicate(t *testing.T) {
	runner, client, _ := defaultScriptedRunner(t, itemListJSON("ITEM_1"))
	itemID, err := client.AddItem(3, issueURL205)
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if itemID != "ITEM_1" {
		t.Fatalf("expected existing item ITEM_1, got %q", itemID)
	}
	if runner.called("project", "item-add", "3", "--owner", "hidekitux", "--url", issueURL205, "--format", "json") {
		t.Fatal("existing item must not be added again")
	}
}

func TestAddItemCreatesMissingItemOnce(t *testing.T) {
	runner, client, _ := defaultScriptedRunner(t, `{"items":[]}`)
	runner.respond([]string{"project", "item-add", "3", "--owner", "hidekitux",
		"--url", issueURL205, "--format", "json"}, fmt.Sprintf(`{"id":"ITEM_2","content":{"url":%q}}`, issueURL205))
	itemID, err := client.AddItem(3, issueURL205)
	if err != nil {
		t.Fatalf("AddItem: %v", err)
	}
	if itemID != "ITEM_2" {
		t.Fatalf("expected new item ITEM_2, got %q", itemID)
	}
	if !runner.called("project", "item-add", "3", "--owner", "hidekitux", "--url", issueURL205, "--format", "json") {
		t.Fatal("missing item must be added once")
	}
}

func TestItemForIssueFailsOnAmbiguity(t *testing.T) {
	runner := newFakeRunner().respond([]string{"project", "item-list", "3", "--owner", "hidekitux", "--limit", "100", "--format", "json"},
		`{"items":[{"id":"A","content":{"url":"`+issueURL205+`"}},{"id":"B","content":{"url":"`+issueURL205+`"}}]}`)
	client := NewClient(runner, "hidekitux")
	if _, _, err := client.ItemForIssue(3, issueURL205); err == nil {
		t.Fatal("expected duplicate item to fail")
	}
}

func TestAccessErrorClassifiesMissingScopes(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().fail([]string{"project", "list", "--owner", "hidekitux", "--format", "json"},
		errors.New("gh project list --owner hidekitux: your authentication token is missing required scopes [read:project]"))
	_, err := NewClient(runner, cfg.Project.Owner).ProjectNumber(cfg)
	target := &AccessError{}
	if !errors.As(err, &target) {
		t.Fatalf("expected AccessError, got %T: %v", err, err)
	}
}

func TestVerifyIssueAcceptsValidContract(t *testing.T) {
	_, client, cfg := defaultScriptedRunner(t, itemListJSON("ITEM_1", "O_BACKLOG", "O_MEDIUM", "O_IMPROV"))
	if err := client.VerifyIssue(cfg, issueURL205); err != nil {
		t.Fatalf("VerifyIssue: %v", err)
	}
}

func TestVerifyIssueAcceptsFlattenedFieldShape(t *testing.T) {
	// Newer gh versions flatten single-select values into top-level item
	// fields instead of the fieldValues array; both shapes must verify.
	cfg := mustConfig(t)
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "hidekitux", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "hidekitux", "--format", "json"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "hidekitux", "--limit", "100", "--format", "json"},
			`{"items":[{"id":"ITEM_1","status":"Backlog","priority":"Medium","scope":"Improvement","content":{"url":"`+issueURL205+`"}}]}`)
	client := NewClient(runner, cfg.Project.Owner)
	if err := client.VerifyIssue(cfg, issueURL205); err != nil {
		t.Fatalf("VerifyIssue with flattened fields: %v", err)
	}
}

func TestVerifyIssueRejectsMissingItemOrValue(t *testing.T) {
	_, client, cfg := defaultScriptedRunner(t, `{"items":[]}`)
	if err := client.VerifyIssue(cfg, issueURL205); err == nil {
		t.Fatal("expected missing item to fail")
	}
	_, client, cfg = defaultScriptedRunner(t, itemListJSON("ITEM_1", "", "O_MEDIUM", "O_IMPROV"))
	if err := client.VerifyIssue(cfg, issueURL205); err == nil {
		t.Fatal("expected missing Status value to fail")
	}
}
