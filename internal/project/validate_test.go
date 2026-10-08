package project

import (
	"bytes"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestCheckIssueProjectFailSafeSkipsOnMissingAccess(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().fail([]string{"project", "list", "--owner", "acme", "--format", "json"},
		errors.New("gh project list --owner acme: your authentication token is missing required scopes [read:project]"))
	var out, errOut bytes.Buffer
	if code := CheckIssueProject(runner, cfg, "acme/sample", 205, &out, &errOut); code != 0 {
		t.Fatalf("expected fail-safe skip code 0, got %d", code)
	}
	if out.String() == "" {
		t.Fatal("expected an actionable skip diagnostic")
	}
}

func TestCheckIssueProjectRejectsInvalidContract(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "acme", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "acme", "--format", "json"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "acme", "--limit", itemLimit, "--format", "json"}, `{"items":[]}`)
	var out, errOut bytes.Buffer
	if code := CheckIssueProject(runner, cfg, "acme/sample", 205, &out, &errOut); code != 1 {
		t.Fatalf("expected invalid contract code 1, got %d", code)
	}
}

func TestCheckIssueProjectAcceptsValidContract(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "acme", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "acme", "--format", "json"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "acme", "--limit", itemLimit, "--format", "json"}, itemListJSON("ITEM_1"))
	var out, errOut bytes.Buffer
	if code := CheckIssueProject(runner, cfg, "acme/sample", 205, &out, &errOut); code != 0 {
		t.Fatalf("expected valid contract code 0, got %d: %s", code, errOut.String())
	}
}

var itemListArgs = []string{"project", "item-list", "3", "--owner", "acme", "--limit", itemLimit, "--format", "json"}

// waitingRunner answers the Project list and fields, and the item list with
// the given successive answers.
func waitingRunner(items ...string) *fakeRunner {
	return newFakeRunner().
		respond([]string{"project", "list", "--owner", "acme", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "acme", "--format", "json"}, fieldListJSON).
		respondInOrder(itemListArgs, items...)
}

func recordSleeps(sleeps *[]time.Duration) func(time.Duration) {
	return func(d time.Duration) { *sleeps = append(*sleeps, d) }
}

func TestCheckIssueProjectWithinPassesWhenFieldsArriveLater(t *testing.T) {
	cfg := mustConfig(t)
	runner := waitingRunner(`{"items":[]}`, itemListJSON("ITEM_1", "", "O_MEDIUM", "O_IMPROV"), itemListJSON("ITEM_1"))
	var sleeps []time.Duration
	policy := WaitPolicy{Limit: 3 * time.Minute, Interval: 30 * time.Second, Sleep: recordSleeps(&sleeps)}
	var out, errOut bytes.Buffer
	if code := CheckIssueProjectWithin(runner, cfg, "acme/sample", 205, policy, &out, &errOut); code != 0 {
		t.Fatalf("expected code 0 once the fields are set, got %d: %s", code, errOut.String())
	}
	if len(sleeps) != 2 || runner.callCount(itemListArgs...) != 3 {
		t.Fatalf("sleeps = %v, item reads = %d; want 2 waits and 3 reads", sleeps, runner.callCount(itemListArgs...))
	}
}

func TestCheckIssueProjectWithinFailsWhenFieldsStayMissing(t *testing.T) {
	cfg := mustConfig(t)
	runner := waitingRunner(itemListJSON("ITEM_1", "", "O_MEDIUM", "O_IMPROV"))
	var sleeps []time.Duration
	policy := WaitPolicy{Limit: 3 * time.Minute, Interval: 30 * time.Second, Sleep: recordSleeps(&sleeps)}
	var out, errOut bytes.Buffer
	if code := CheckIssueProjectWithin(runner, cfg, "acme/sample", 205, policy, &out, &errOut); code != 1 {
		t.Fatalf("expected code 1 for a Status that stays empty, got %d", code)
	}
	if len(sleeps) != 6 || runner.callCount(itemListArgs...) != 7 {
		t.Fatalf("sleeps = %d, item reads = %d; want 6 waits and 7 reads within 3 minutes", len(sleeps), runner.callCount(itemListArgs...))
	}
	if !strings.Contains(errOut.String(), "no valid Status value") {
		t.Fatalf("errOut = %q, want the missing Status", errOut.String())
	}
}

func TestCheckIssueProjectWithinFailsAtOnceOnAnUndeclaredValue(t *testing.T) {
	cfg := mustConfig(t)
	runner := waitingRunner(itemListJSON("ITEM_1", "O_UNKNOWN", "O_MEDIUM", "O_IMPROV"))
	var sleeps []time.Duration
	policy := WaitPolicy{Limit: 3 * time.Minute, Interval: 30 * time.Second, Sleep: recordSleeps(&sleeps)}
	var out, errOut bytes.Buffer
	if code := CheckIssueProjectWithin(runner, cfg, "acme/sample", 205, policy, &out, &errOut); code != 1 {
		t.Fatalf("expected code 1 for an undeclared value, got %d", code)
	}
	if len(sleeps) != 0 || runner.callCount(itemListArgs...) != 1 {
		t.Fatalf("sleeps = %d, item reads = %d; want no wait and one read", len(sleeps), runner.callCount(itemListArgs...))
	}
}

func TestCheckIssueProjectReadsOnceWithoutAWait(t *testing.T) {
	cfg := mustConfig(t)
	runner := waitingRunner(`{"items":[]}`, itemListJSON("ITEM_1"))
	var out, errOut bytes.Buffer
	if code := CheckIssueProject(runner, cfg, "acme/sample", 205, &out, &errOut); code != 1 {
		t.Fatalf("expected code 1 without waiting, got %d", code)
	}
	if runner.callCount(itemListArgs...) != 1 {
		t.Fatalf("item reads = %d, want 1", runner.callCount(itemListArgs...))
	}
}
