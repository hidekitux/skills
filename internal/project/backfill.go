package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

// MigratedLabelPrefixes are the triage label prefixes retired by the
// migration to the GitHub Project.
var MigratedLabelPrefixes = []string{"priority:", "scope:", "phase:"}

// Issue is one triage input record from `gh issue list`.
type Issue struct {
	Number int64        `json:"number"`
	Title  string       `json:"title"`
	State  string       `json:"state"`
	URL    string       `json:"url"`
	Labels []issueLabel `json:"labels"`
}

type issueLabel struct {
	Name string `json:"name"`
}

// typeScope maps Issue title types to the declared Scope option names.
var typeScope = map[string]string{
	"Feature":       "Feature",
	"Bug":           "Bug",
	"Documentation": "Docs",
	"Maintenance":   "Maintenance",
	"Improvement":   "Improvement",
	"Security":      "Security",
	"Release":       "Release",
}

// labelStatus maps phase label values to the declared Status option names.
var labelStatus = map[string]string{
	"backlog":     "Backlog",
	"planned":     "Planned",
	"in-progress": "In progress",
}

// labelPriority maps priority label values to the declared Priority option
// names.
var labelPriority = map[string]string{
	"high":   "High",
	"medium": "Medium",
	"low":    "Low",
}

// labelScope maps scope label values to the declared Scope option names.
var labelScope = map[string]string{
	"feature":     "Feature",
	"bug":         "Bug",
	"docs":        "Docs",
	"maintenance": "Maintenance",
	"improvement": "Improvement",
	"release":     "Release",
}

func labelNames(labels []issueLabel) []string {
	names := make([]string, 0, len(labels))
	for _, label := range labels {
		names = append(names, label.Name)
	}
	return names
}

func labelValue(labels []string, prefix string) string {
	for _, label := range labels {
		if strings.HasPrefix(label, prefix) {
			return strings.TrimPrefix(label, prefix)
		}
	}
	return ""
}

func hasMigratedLabel(labels []string) bool {
	for _, prefix := range MigratedLabelPrefixes {
		if labelValue(labels, prefix) != "" {
			return true
		}
	}
	return false
}

func hasAnyLabel(labels []string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if labelValue(labels, prefix) != "" {
			return true
		}
	}
	return false
}

// scopeFromTitle derives the Scope option from an Issue title such as
// "[Improvement]: Summary", returning "" when the type does not map.
func scopeFromTitle(title string) string {
	title = strings.TrimSpace(title)
	if !strings.HasPrefix(title, "[") {
		return ""
	}
	end := strings.Index(title, "]")
	if end <= 1 {
		return ""
	}
	scope, ok := typeScope[title[1:end]]
	if !ok {
		return ""
	}
	return scope
}

// DeriveFieldValues maps triage labels and Issue state to the Project field
// values for the migration. Closed Issues reach Done; open Issues keep their
// phase-derived Status with Backlog as the fallback. Priority falls back to
// the configured default and Scope falls back to the Issue type from the
// title when the label is absent. An unusable derivation is an error so the
// migration never mutates with invalid values.
func DeriveFieldValues(issue Issue, cfg *Config) (FieldValues, error) {
	labels := labelNames(issue.Labels)
	values := FieldValues{}
	switch issue.State {
	case "CLOSED":
		values.Status = "Done"
	case "OPEN":
		status, ok := labelStatus[labelValue(labels, "phase:")]
		if !ok {
			status = "Backlog"
		}
		values.Status = status
	default:
		return FieldValues{}, fmt.Errorf("issue #%d has unknown state %q", issue.Number, issue.State)
	}
	priority, ok := labelPriority[labelValue(labels, "priority:")]
	if !ok {
		priority = cfg.Project.DefaultPriority
	}
	values.Priority = priority
	scope, ok := labelScope[labelValue(labels, "scope:")]
	if !ok {
		scope = scopeFromTitle(issue.Title)
	}
	values.Scope = scope
	for role, option := range map[string]string{"status": values.Status, "priority": values.Priority, "scope": values.Scope} {
		if !cfg.HasOption(role, option) {
			return FieldValues{}, fmt.Errorf("issue #%d derives %s %q, which is not a declared option", issue.Number, role, option)
		}
	}
	return values, nil
}

// BackfillPlan pairs one migrated Issue with its derived Project values.
type BackfillPlan struct {
	Issue  Issue
	Values FieldValues
}

// listIssues returns every Issue for the repository state via gh.
func listIssues(run Runner, repo, state string) ([]Issue, error) {
	out, err := run.Run("issue", "list", "--repo", repo, "--state", state, "--limit", "1000",
		"--json", "number,title,state,url,labels")
	if err != nil {
		return nil, err
	}
	var issues []Issue
	if err := json.Unmarshal([]byte(out), &issues); err != nil {
		return nil, fmt.Errorf("issue list output cannot be parsed: %w", err)
	}
	return issues, nil
}

// PlanBackfill derives Project values for every Issue carrying a migrated
// triage label. Issues without migrated labels are left untouched.
func PlanBackfill(run Runner, cfg *Config, repo string) ([]BackfillPlan, []Issue, error) {
	issues, err := listIssues(run, repo, "all")
	if err != nil {
		return nil, nil, err
	}
	var plans []BackfillPlan
	var untouched []Issue
	for _, issue := range issues {
		if !hasMigratedLabel(labelNames(issue.Labels)) {
			untouched = append(untouched, issue)
			continue
		}
		values, err := DeriveFieldValues(issue, cfg)
		if err != nil {
			return nil, nil, err
		}
		plans = append(plans, BackfillPlan{Issue: issue, Values: values})
	}
	return plans, untouched, nil
}

// ApplyBackfill reconciles every planned Issue into the Project using one
// item-list read, returning the number of Issues whose item was added or
// updated. Items that already carry the desired field values are left
// untouched, so repeated runs are idempotent and cheap.
func ApplyBackfill(run Runner, cfg *Config, plans []BackfillPlan) (int, error) {
	client := NewClient(run, cfg.Project.Owner)
	number, projectID, err := client.ProjectTarget(cfg)
	if err != nil {
		return 0, err
	}
	fields, err := client.resolvedFields(cfg, number)
	if err != nil {
		return 0, err
	}
	items, err := client.Items(number)
	if err != nil {
		return 0, err
	}
	byURL := map[string]itemDTO{}
	for _, item := range items {
		byURL[item.Content.URL] = item
	}

	processed := 0
	for _, plan := range plans {
		// Resolve every desired option before any mutation so an unknown or
		// missing option never leaves a partially mutated item.
		desiredIDs := map[string]string{}
		for _, role := range RequiredFields {
			optionID, err := optionID(fields, role, planValue(plan, role))
			if err != nil {
				return processed, err
			}
			desiredIDs[role] = optionID
		}

		item, exists := byURL[plan.Issue.URL]
		current := map[string]string{}
		itemID := ""
		if exists {
			itemID = item.ID
			current, err = itemFieldNames(item, fields)
			if err != nil {
				return processed, err
			}
		} else {
			itemID, err = client.addItemUnchecked(number, plan.Issue.URL)
			if err != nil {
				return processed, err
			}
			byURL[plan.Issue.URL] = itemDTO{ID: itemID}
		}

		for _, role := range RequiredFields {
			if exists && current[role] == planValue(plan, role) {
				continue
			}
			if err := client.SetSingleSelect(projectID, itemID, fields[role].ID, desiredIDs[role]); err != nil {
				return processed, err
			}
		}
		processed++
	}
	return processed, nil
}

// planValue returns the desired field value for one plan role.
func planValue(plan BackfillPlan, role string) string {
	switch role {
	case "status":
		return plan.Values.Status
	case "priority":
		return plan.Values.Priority
	default:
		return plan.Values.Scope
	}
}

// VerifyBackfill checks that every planned Issue has exactly one Project item
// with valid Status, Priority, and Scope values using one Project, field, and
// item read, so repository-wide reconciliation stays cheap under the
// Projects API rate budget.
func VerifyBackfill(run Runner, cfg *Config, plans []BackfillPlan) error {
	client := NewClient(run, cfg.Project.Owner)
	number, err := client.ProjectNumber(cfg)
	if err != nil {
		return err
	}
	fields, err := client.resolvedFields(cfg, number)
	if err != nil {
		return err
	}
	items, err := client.Items(number)
	if err != nil {
		return err
	}
	byURL := map[string]itemDTO{}
	for _, item := range items {
		byURL[item.Content.URL] = item
	}
	for _, plan := range plans {
		item, exists := byURL[plan.Issue.URL]
		if !exists {
			return fmt.Errorf("Issue %s has no item in the declared Project; expected exactly one", plan.Issue.URL)
		}
		current, err := itemFieldNames(item, fields)
		if err != nil {
			return err
		}
		for _, role := range RequiredFields {
			if current[role] == "" {
				return fmt.Errorf("Issue %s Project item has no valid %s value", plan.Issue.URL, roleTitles[role])
			}
			if !cfg.HasOption(role, current[role]) {
				return fmt.Errorf("Issue %s Project item has an undeclared %s value %q", plan.Issue.URL, roleTitles[role], current[role])
			}
		}
	}
	return nil
}

// RemoveMigratedLabels removes every migrated triage label from the Issue,
// returning the labels removed.
func RemoveMigratedLabels(run Runner, repo string, issue Issue) ([]string, error) {
	var removed []string
	for _, label := range labelNames(issue.Labels) {
		if !hasAnyLabel([]string{label}, MigratedLabelPrefixes) {
			continue
		}
		if _, err := run.Run("issue", "edit", fmt.Sprint(issue.Number), "--repo", repo, "--remove-label", label); err != nil {
			return removed, err
		}
		removed = append(removed, label)
	}
	sort.Strings(removed)
	return removed, nil
}

// VerifyLabelsGone confirms that no Issue in the repository retains a
// migrated triage label, returning the offending Issue numbers.
func VerifyLabelsGone(run Runner, repo string) ([]int64, error) {
	issues, err := listIssues(run, repo, "all")
	if err != nil {
		return nil, err
	}
	var offenders []int64
	for _, issue := range issues {
		if hasMigratedLabel(labelNames(issue.Labels)) {
			offenders = append(offenders, issue.Number)
		}
	}
	sort.Slice(offenders, func(i, j int) bool { return offenders[i] < offenders[j] })
	return offenders, nil
}

// RetireLabelDefinitions deletes the migrated label definitions from the
// repository, returning the retired label names.
func RetireLabelDefinitions(run Runner, repo string) ([]string, error) {
	definitions := []string{}
	for _, label := range migratedLabelDefinitions() {
		if _, err := run.Run("label", "delete", label, "--repo", repo, "--yes"); err != nil {
			return definitions, err
		}
		definitions = append(definitions, label)
	}
	return definitions, nil
}

// migratedLabelDefinitions returns the retired triage label names.
func migratedLabelDefinitions() []string {
	definitions := []string{}
	for _, prefix := range MigratedLabelPrefixes {
		fixed := strings.TrimSuffix(prefix, ":")
		switch fixed {
		case "priority":
			for _, option := range []string{"high", "medium", "low"} {
				definitions = append(definitions, "priority:"+option)
			}
		case "scope":
			for _, option := range []string{"feature", "bug", "docs", "maintenance", "improvement", "release"} {
				definitions = append(definitions, "scope:"+option)
			}
		case "phase":
			for _, option := range []string{"backlog", "planned", "in-progress"} {
				definitions = append(definitions, "phase:"+option)
			}
		}
	}
	sort.Strings(definitions)
	return definitions
}

// reportBackfillError returns the operator exit code for a migration
// failure: 1 for live-operation failures and 2 for configuration errors.
func reportBackfillError(err error) int {
	target := &AccessError{}
	if errors.As(err, &target) {
		return 1
	}
	return 2
}
