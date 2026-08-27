package project

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// roleTitles maps a field role to its human-readable name for diagnostics.
var roleTitles = map[string]string{
	"status":   "Status",
	"priority": "Priority",
	"scope":    "Scope",
}

// IssueURL returns the GitHub URL for one Issue.
func IssueURL(owner, repo string, number int64) string {
	return fmt.Sprintf("https://github.com/%s/%s/issues/%d", owner, repo, number)
}

// splitRepo returns the owner and name for an owner/name repository string.
func splitRepo(repo string) (owner, name string, ok bool) {
	owner, name, found := strings.Cut(repo, "/")
	if !found || strings.TrimSpace(owner) == "" || strings.TrimSpace(name) == "" {
		return "", "", false
	}
	return owner, name, true
}

// ContractError reports an invalid live Project contract on an Issue
// (missing item, duplicate items, or invalid field values) as distinct from
// repository configuration drift or external access problems.
type ContractError struct {
	Message string
}

func (e *ContractError) Error() string { return e.Message }

// VerifyIssue checks that the Issue has exactly one Project item whose
// Status, Priority, and Scope values are each one of the declared options.
func (c *Client) VerifyIssue(cfg *Config, issueURL string) error {
	number, err := c.ProjectNumber(cfg)
	if err != nil {
		return err
	}
	fields, err := c.resolvedFields(cfg, number)
	if err != nil {
		return err
	}
	item, present, err := c.ItemForIssue(number, issueURL)
	if err != nil {
		return err
	}
	if !present {
		return &ContractError{Message: fmt.Sprintf("Issue %s has no item in the declared Project; expected exactly one", issueURL)}
	}
	current, err := itemFieldNames(item, fields)
	if err != nil {
		return &ContractError{Message: err.Error()}
	}
	for _, role := range RequiredFields {
		if current[role] == "" {
			return &ContractError{Message: fmt.Sprintf("Issue %s Project item has no valid %s value", issueURL, roleTitles[role])}
		}
		if !cfg.HasOption(role, current[role]) {
			return &ContractError{Message: fmt.Sprintf("Issue %s Project item has an undeclared %s value %q", issueURL, roleTitles[role], current[role])}
		}
	}
	return nil
}

// reportFailure writes an actionable diagnostic and returns a process exit
// code. Access and API failures fail safely with exit 0; configuration and
// live-contract drift fail with exit 2.
func reportFailure(err error, out, errOut io.Writer) int {
	target := &AccessError{}
	if errors.As(err, &target) {
		fmt.Fprintf(out, "Project check skipped: %s\n", target.Message)
		return 0
	}
	fmt.Fprintf(errOut, "error: %v\n", err)
	return 2
}

// CheckIssueProject verifies that the Issue has exactly one Project item with
// valid Status, Priority, and Scope values. It returns 0 when the contract
// holds or when Project access is unavailable (fail-safe skip), 1 when the
// item set or field values are invalid, and 2 for usage or configuration
// errors.
func CheckIssueProject(run Runner, cfg *Config, repo string, issueNumber int64, out, errOut io.Writer) int {
	owner, name, ok := splitRepo(repo)
	if !ok {
		fmt.Fprintf(errOut, "error: --repo must be owner/name\n")
		return 2
	}
	if issueNumber <= 0 {
		fmt.Fprintf(errOut, "error: --issue must be a positive Issue number\n")
		return 2
	}
	client := NewClient(run, owner)
	issueURL := IssueURL(owner, name, issueNumber)
	if err := client.VerifyIssue(cfg, issueURL); err != nil {
		var contract *ContractError
		if errors.As(err, &contract) {
			fmt.Fprintf(errOut, "error: %v\n", err)
			return 1
		}
		return reportFailure(err, out, errOut)
	}
	fmt.Fprintf(out, "Issue #%d has exactly one Project item with valid Status, Priority, and Scope values.\n", issueNumber)
	return 0
}
