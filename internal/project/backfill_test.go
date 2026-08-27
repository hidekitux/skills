package project

import (
	"errors"
	"fmt"
	"testing"
)

func triageIssue(number int64, title, state, url string, labels ...string) Issue {
	issue := Issue{Number: number, Title: title, State: state, URL: url}
	for _, label := range labels {
		issue.Labels = append(issue.Labels, issueLabel{Name: label})
	}
	return issue
}

func TestDeriveFieldValuesFromTriageLabelsAndState(t *testing.T) {
	cfg := mustConfig(t)
	cases := []struct {
		name   string
		issue  Issue
		status string
		scope  string
	}{
		{"closed reaches Done", triageIssue(1, "[Improvement]: X", "CLOSED", "u1",
			"priority:medium", "scope:improvement", "phase:in-progress"), "Done", "Improvement"},
		{"backlog", triageIssue(2, "[Feature]: X", "OPEN", "u2",
			"priority:high", "scope:feature", "phase:backlog"), "Backlog", "Feature"},
		{"planned", triageIssue(3, "[Docs]: X", "OPEN", "u3",
			"priority:low", "scope:docs", "phase:planned"), "Planned", "Docs"},
		{"in progress", triageIssue(4, "[Maintenance]: X", "OPEN", "u4",
			"priority:medium", "scope:maintenance", "phase:in-progress"), "In progress", "Maintenance"},
		{"missing phase falls back to Backlog", triageIssue(5, "[Security]: X", "OPEN", "u5",
			"priority:medium", "scope:bug"), "Backlog", "Bug"},
		{"missing priority falls back to default", triageIssue(6, "[Bug]: X", "OPEN", "u6",
			"scope:bug", "phase:backlog"), "Backlog", "Bug"},
		{"missing scope falls back to title type", triageIssue(7, "[Security]: X", "OPEN", "u7",
			"priority:medium", "phase:backlog"), "Backlog", "Security"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values, err := DeriveFieldValues(tc.issue, cfg)
			if err != nil {
				t.Fatalf("DeriveFieldValues: %v", err)
			}
			if values.Status != tc.status || values.Scope != tc.scope {
				t.Fatalf("got Status=%q Scope=%q, want Status=%q Scope=%q",
					values.Status, values.Scope, tc.status, tc.scope)
			}
			if values.Priority == "" {
				t.Fatal("priority must be derived or defaulted")
			}
		})
	}
}

func TestDeriveFieldValuesFailsOnUnknownStateOrUnusableScope(t *testing.T) {
	cfg := mustConfig(t)
	if _, err := DeriveFieldValues(triageIssue(1, "[Improvement]: X", "MERGED", "u1",
		"priority:medium", "scope:improvement", "phase:backlog"), cfg); err == nil {
		t.Fatal("expected unknown state to fail")
	}
	if _, err := DeriveFieldValues(triageIssue(2, "No type prefix", "OPEN", "u2",
		"priority:medium", "phase:backlog"), cfg); err == nil {
		t.Fatal("expected unusable scope to fail")
	}
}

func TestPlanBackfillFiltersUntouchedIssues(t *testing.T) {
	cfg := mustConfig(t)
	runner := newFakeRunner().respond(
		[]string{"issue", "list", "--repo", "acme/sample", "--state", "all", "--limit", "1000",
			"--json", "number,title,state,url,labels"},
		`[{"number":1,"title":"[Improvement]: A","state":"OPEN","url":"u1","labels":[{"name":"priority:medium"},{"name":"scope:improvement"},{"name":"phase:backlog"}]},
		  {"number":2,"title":"[Feature]: B","state":"OPEN","url":"u2","labels":[{"name":"dependencies"}]}]`)
	plans, untouched, err := PlanBackfill(runner, cfg, "acme/sample")
	if err != nil {
		t.Fatalf("PlanBackfill: %v", err)
	}
	if len(plans) != 1 || plans[0].Issue.Number != 1 || plans[0].Values.Status != "Backlog" {
		t.Fatalf("unexpected plans: %+v", plans)
	}
	if len(untouched) != 1 || untouched[0].Number != 2 {
		t.Fatalf("unexpected untouched: %+v", untouched)
	}
}

func TestApplyAndVerifyBackfill(t *testing.T) {
	cfg := mustConfig(t)
	plans := []BackfillPlan{
		{Issue: triageIssue(1, "[Improvement]: A", "OPEN", "https://github.com/acme/sample/issues/205",
			"priority:medium", "scope:improvement", "phase:backlog"),
			Values: FieldValues{Status: "Backlog", Priority: "Medium", Scope: "Improvement"}},
		{Issue: triageIssue(2, "[Feature]: B", "OPEN", "https://github.com/acme/sample/issues/201",
			"priority:medium", "scope:feature", "phase:planned"),
			Values: FieldValues{Status: "Planned", Priority: "Medium", Scope: "Feature"}},
		{Issue: triageIssue(3, "[Docs]: C", "OPEN", "https://github.com/acme/sample/issues/202",
			"priority:low", "scope:docs", "phase:backlog"),
			Values: FieldValues{Status: "Backlog", Priority: "Low", Scope: "Docs"}},
	}
	itemsJSON := `{"items":[{"id":"ITEM_1","content":{"url":"https://github.com/acme/sample/issues/205"},
	  "fieldValues":[{"field":{"id":"F_STATUS","name":"Status"},"optionId":"O_BACKLOG"},
	                  {"field":{"id":"F_PRIORITY","name":"Priority"},"optionId":"O_MEDIUM"},
	                  {"field":{"id":"F_SCOPE","name":"Scope"},"optionId":"O_IMPROV"}]},
	 {"id":"ITEM_2","content":{"url":"https://github.com/acme/sample/issues/201"},"fieldValues":[]}]}`
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "acme", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "acme", "--format", "json"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "acme", "--limit", "100", "--format", "json"}, itemsJSON).
		respond([]string{"project", "item-add", "3", "--owner", "acme",
			"--url", "https://github.com/acme/sample/issues/202", "--format", "json"},
			`{"id":"ITEM_3","content":{"url":"https://github.com/acme/sample/issues/202"}}`)
	for _, edit := range [][]string{
		{"project", "item-edit", "--id", "ITEM_2", "--field-id", "F_STATUS", "--project-id", "PVT_1", "--single-select-option-id", "O_PLANNED"},
		{"project", "item-edit", "--id", "ITEM_2", "--field-id", "F_PRIORITY", "--project-id", "PVT_1", "--single-select-option-id", "O_MEDIUM"},
		{"project", "item-edit", "--id", "ITEM_2", "--field-id", "F_SCOPE", "--project-id", "PVT_1", "--single-select-option-id", "O_FEATURE"},
		{"project", "item-edit", "--id", "ITEM_3", "--field-id", "F_STATUS", "--project-id", "PVT_1", "--single-select-option-id", "O_BACKLOG"},
		{"project", "item-edit", "--id", "ITEM_3", "--field-id", "F_PRIORITY", "--project-id", "PVT_1", "--single-select-option-id", "O_LOW"},
		{"project", "item-edit", "--id", "ITEM_3", "--field-id", "F_SCOPE", "--project-id", "PVT_1", "--single-select-option-id", "O_DOCS"},
	} {
		runner.respond(edit, "")
	}

	count, err := ApplyBackfill(runner, cfg, plans)
	if err != nil {
		t.Fatalf("ApplyBackfill: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 processed, got %d", count)
	}
	// ITEM_1 already carries every desired value: no mutation may run for it.
	if runner.called("project", "item-edit", "--id", "ITEM_1") {
		t.Fatal("ITEM_1 must not be mutated when its values already match")
	}
	// A single item-list read covers the whole migration (no per-Issue reads).
	if got := len(runner.calls); got > 12 {
		t.Fatalf("expected a bounded call count, got %d (call log: %v)", got, runner.calls)
	}
}

func TestVerifyBackfillChecksEveryPlannedItem(t *testing.T) {
	cfg := mustConfig(t)
	valid := []BackfillPlan{
		{Issue: triageIssue(1, "[Improvement]: A", "OPEN", "https://github.com/acme/sample/issues/205",
			"priority:medium", "scope:improvement", "phase:backlog"),
			Values: FieldValues{Status: "Backlog", Priority: "Medium", Scope: "Improvement"}},
	}
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "acme", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "acme", "--format", "json"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "acme", "--limit", "100", "--format", "json"}, itemListJSON("ITEM_1"))
	if err := VerifyBackfill(runner, cfg, valid); err != nil {
		t.Fatalf("VerifyBackfill on valid items: %v", err)
	}
	incomplete := newFakeRunner().
		respond([]string{"project", "list", "--owner", "acme", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "acme", "--format", "json"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "acme", "--limit", "100", "--format", "json"}, `{"items":[]}`)
	if err := VerifyBackfill(incomplete, cfg, valid); err == nil {
		t.Fatal("expected missing item to fail verification")
	}
}

func TestApplyBackfillFailsBeforeMutationOnUndeclaredOption(t *testing.T) {
	cfg := mustConfig(t)
	plans := []BackfillPlan{
		{Issue: triageIssue(1, "[Improvement]: A", "OPEN", "https://github.com/acme/sample/issues/205",
			"priority:medium", "scope:improvement", "phase:backlog"),
			Values: FieldValues{Status: "Backlog", Priority: "Medium", Scope: "nonsense"}},
	}
	runner := newFakeRunner().
		respond([]string{"project", "list", "--owner", "acme", "--format", "json"}, projectListJSON).
		respond([]string{"project", "field-list", "3", "--owner", "acme", "--format", "json"}, fieldListJSON).
		respond([]string{"project", "item-list", "3", "--owner", "acme", "--limit", "100", "--format", "json"}, `{"items":[]}`)
	if count, err := ApplyBackfill(runner, cfg, plans); err == nil {
		t.Fatal("expected undeclared option to fail")
	} else if count != 0 {
		t.Fatalf("expected no processed Issues, got %d", count)
	}
	if runner.called("project", "item-add") || runner.called("project", "item-edit") {
		t.Fatal("no mutation may happen before option resolution")
	}
}

func TestRemoveMigratedLabelsKeepsUnrelatedLabels(t *testing.T) {
	issue := triageIssue(1, "[Improvement]: A", "OPEN", "u1",
		"priority:medium", "scope:improvement", "phase:backlog", "dependencies", "accessibility")
	runner := newFakeRunner()
	for _, label := range []string{"priority:medium", "scope:improvement", "phase:backlog"} {
		runner.respond([]string{"issue", "edit", "1", "--repo", "acme/sample",
			"--remove-label", label}, fmt.Sprintf(`{"number":1,"title":"[Improvement]: A"}`))
	}
	removed, err := RemoveMigratedLabels(runner, "acme/sample", issue)
	if err != nil {
		t.Fatalf("RemoveMigratedLabels: %v", err)
	}
	if len(removed) != 3 {
		t.Fatalf("expected 3 removed, got %v", removed)
	}
	if runner.called("issue", "edit", "1", "--repo", "acme/sample", "--remove-label", "dependencies") ||
		runner.called("issue", "edit", "1", "--repo", "acme/sample", "--remove-label", "accessibility") {
		t.Fatal("unrelated labels must not be removed")
	}
}

func TestVerifyLabelsGone(t *testing.T) {
	clean := newFakeRunner().respond(
		[]string{"issue", "list", "--repo", "acme/sample", "--state", "all", "--limit", "1000",
			"--json", "number,title,state,url,labels"},
		`[{"number":1,"title":"[Docs]: X","state":"CLOSED","url":"u1","labels":[{"name":"documentation"}]}]`)
	offenders, err := VerifyLabelsGone(clean, "acme/sample")
	if err != nil {
		t.Fatalf("VerifyLabelsGone: %v", err)
	}
	if len(offenders) != 0 {
		t.Fatalf("expected no offenders, got %v", offenders)
	}
	dirty := newFakeRunner().respond(
		[]string{"issue", "list", "--repo", "acme/sample", "--state", "all", "--limit", "1000",
			"--json", "number,title,state,url,labels"},
		`[{"number":7,"title":"[Bug]: X","state":"OPEN","url":"u7","labels":[{"name":"priority:low"}]}]`)
	offenders, err = VerifyLabelsGone(dirty, "acme/sample")
	if err != nil {
		t.Fatalf("VerifyLabelsGone: %v", err)
	}
	if len(offenders) != 1 || offenders[0] != 7 {
		t.Fatalf("expected offender 7, got %v", offenders)
	}
}

func TestRetireLabelDefinitionsDeletesTriageLabels(t *testing.T) {
	runner := newFakeRunner()
	for _, label := range migratedLabelDefinitions() {
		runner.respond([]string{"label", "delete", label, "--repo", "acme/sample", "--yes"}, "")
	}
	retired, err := RetireLabelDefinitions(runner, "acme/sample")
	if err != nil {
		t.Fatalf("RetireLabelDefinitions: %v", err)
	}
	if len(retired) != 12 {
		t.Fatalf("expected 12 retired label definitions, got %v", retired)
	}
	for _, label := range []string{"priority:high", "scope:release", "phase:in-progress"} {
		if !containsString(retired, label) {
			t.Fatalf("expected %q in retired %v", label, retired)
		}
	}
}

func TestMigratedLabelDefinitionsAreClosed(t *testing.T) {
	for _, label := range migratedLabelDefinitions() {
		if label == "dependencies" || label == "github_actions" {
			t.Fatalf("unrelated label %q must not be retired", label)
		}
	}
}

func TestListIssuesPropagatesAccessFailure(t *testing.T) {
	runner := newFakeRunner().fail([]string{"issue", "list", "--repo", "acme/sample",
		"--state", "all", "--limit", "1000", "--json", "number,title,state,url,labels"},
		errors.New("gh issue list: boom"))
	if _, _, err := PlanBackfill(runner, mustConfig(t), "acme/sample"); err == nil {
		t.Fatal("expected list failure to propagate")
	}
}
