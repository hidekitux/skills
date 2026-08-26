package project

import (
	"fmt"
	"io"
)

// SetIssueStatus resolves the declared Project and sets the Issue's Status
// option, adding the item once when missing. It returns 0 on success (or a
// dry-run report), 1 on access or API failure, and 2 on usage or
// configuration errors. Every mutable ID is resolved before any mutation.
func SetIssueStatus(run Runner, cfg *Config, repo string, issueNumber int64, status string, dryRun bool, out, errOut io.Writer) int {
	owner, name, ok := splitRepo(repo)
	if !ok {
		fmt.Fprintf(errOut, "error: --repo must be owner/name\n")
		return 2
	}
	if issueNumber <= 0 {
		fmt.Fprintf(errOut, "error: --issue must be a positive Issue number\n")
		return 2
	}
	statusField, _ := cfg.Field("status")
	if !cfg.HasOption("status", status) {
		fmt.Fprintf(errOut, "error: --status %q must be one of fields.status.options %v\n",
			status, statusField.Options)
		return 2
	}
	client := NewClient(run, owner)
	issueURL := IssueURL(owner, name, issueNumber)
	number, err := client.ProjectNumber(cfg)
	if err != nil {
		return hardFailure(err, out, errOut)
	}
	fields, err := client.resolvedFields(cfg, number)
	if err != nil {
		return hardFailure(err, out, errOut)
	}
	statusID, err := optionID(fields, "status", status)
	if err != nil {
		return hardFailure(err, out, errOut)
	}
	if dryRun {
		fmt.Fprintf(out, "dry-run: Issue %s would get exactly one Project item with Status %q\n", issueURL, status)
		return 0
	}
	itemID, err := client.AddItem(number, issueURL)
	if err != nil {
		return hardFailure(err, out, errOut)
	}
	if err := client.SetSingleSelect(number, itemID, fields["status"].ID, statusID); err != nil {
		return hardFailure(err, out, errOut)
	}
	fmt.Fprintf(out, "Issue %s Status set to %q\n", issueURL, status)
	return 0
}

// hardFailure reports a live-operation failure loudly for automation, since
// an unavailable or misconfigured Project must stay visible.
func hardFailure(err error, out, errOut io.Writer) int {
	fmt.Fprintf(errOut, "error: %v\n", err)
	return 1
}
