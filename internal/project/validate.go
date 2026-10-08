package project

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
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
	// Pending marks a contract that create-issue may still complete: the
	// Issue has no item yet, or a required field is still empty.
	Pending bool
}

func (e *ContractError) Error() string { return e.Message }

// VerifyIssue checks that the Issue has exactly one Project item whose
// Status, Priority, and Scope values are each one of the declared options.
func (c *Client) VerifyIssue(cfg *Config, issueURL string) error {
	snapshot, err := c.projectSnapshot(cfg, true)
	if err != nil {
		return err
	}
	item, present, err := snapshot.itemForIssue(issueURL)
	if err != nil {
		return err
	}
	if !present {
		return &ContractError{Message: fmt.Sprintf("Issue %s has no item in the declared Project; expected exactly one", issueURL), Pending: true}
	}
	current, err := itemFieldNames(item, snapshot.fields)
	if err != nil {
		return &ContractError{Message: err.Error()}
	}
	for _, role := range RequiredFields {
		if current[role] == "" {
			return &ContractError{Message: fmt.Sprintf("Issue %s Project item has no valid %s value", issueURL, roleTitles[role]), Pending: true}
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
	return CheckIssueProjectWithin(run, cfg, repo, issueNumber, WaitPolicy{}, out, errOut)
}

// WaitPolicy bounds how long CheckIssueProjectWithin reads the Project again
// while the contract is pending. A zero Limit reads once. Sleep defaults to
// time.Sleep and is replaceable in tests.
type WaitPolicy struct {
	Limit    time.Duration
	Interval time.Duration
	Sleep    func(time.Duration)
}

// CheckIssueProjectWithin is CheckIssueProject with waiting: while the Issue
// has no Project item or a required field is still empty, it reads the
// Project again every Interval until Limit has passed, because create-issue
// sets the fields only after the Issue opens. Any other result returns at
// once.
func CheckIssueProjectWithin(run Runner, cfg *Config, repo string, issueNumber int64, wait WaitPolicy, out, errOut io.Writer) int {
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
	sleep := wait.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	err := client.VerifyIssue(cfg, issueURL)
	for waited := time.Duration(0); wait.Interval > 0 && waited < wait.Limit && isPending(err); waited += wait.Interval {
		fmt.Fprintf(out, "Project contract is pending (%v); reading again in %s.\n", err, wait.Interval)
		sleep(wait.Interval)
		err = client.VerifyIssue(cfg, issueURL)
	}
	if err != nil {
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

// isPending reports whether err is a contract that create-issue may still
// complete.
func isPending(err error) bool {
	var contract *ContractError
	return errors.As(err, &contract) && contract.Pending
}
