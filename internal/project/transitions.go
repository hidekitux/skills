package project

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// PullRequestEvent is one pull_request event the Project automation reacts
// to. Only the events that change workflow state are handled; metadata edits
// (edited, labeled, review requested, and so on) never change Status.
type PullRequestEvent struct {
	Type   string // opened, reopened, synchronize, ready_for_review, converted_to_draft, closed
	Draft  bool
	Merged bool
}

// PullRequestStatus maps a pull_request event to the governing Issue Project
// Status. A merged Pull Request returns ok=false: the linked Issue closes and
// the built-in Item closed workflow reaches Done, while release Issues
// tracked with Tracks stay non-terminal until publication.
func PullRequestStatus(event PullRequestEvent) (string, bool, error) {
	switch event.Type {
	case "converted_to_draft":
		return "In progress", true, nil
	case "ready_for_review":
		return "In review", true, nil
	case "opened", "reopened", "synchronize":
		if event.Draft {
			return "In progress", true, nil
		}
		return "In review", true, nil
	case "closed":
		if event.Merged {
			return "", false, nil
		}
		return "In progress", true, nil
	default:
		return "", false, fmt.Errorf("unsupported pull_request event type %q", event.Type)
	}
}

// SetIssueStatus resolves the declared Project and sets the Issue's Status
// option, adding the item once when missing. It returns 0 on success (or a
// dry-run report), 1 on access or API failure, and 2 on usage or
// configuration errors. Every mutable ID is resolved before any mutation;
// when skipClosed is set, a closed Issue is never moved away from Done.
func SetIssueStatus(run Runner, cfg *Config, repo string, issueNumber int64, status string, dryRun, skipClosed bool, out, errOut io.Writer) int {
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
	if skipClosed {
		closed, err := issueIsClosed(run, repo, issueNumber)
		if err != nil {
			return hardFailure(err, out, errOut)
		}
		if closed {
			fmt.Fprintf(out, "Issue #%d is closed; leaving its Project Status at %q.\n", issueNumber, "Done")
			return 0
		}
	}
	client := NewClient(run, owner)
	issueURL := IssueURL(owner, name, issueNumber)
	number, projectID, err := client.ProjectTarget(cfg)
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
	var item itemDTO
	present := false
	if status == "Planned" {
		item, present, err = client.ItemForIssue(number, issueURL)
		if err != nil {
			return hardFailure(err, out, errOut)
		}
	}
	if present && status == "Planned" {
		current, err := itemFieldNames(item, fields)
		if err != nil {
			return hardFailure(err, out, errOut)
		}
		switch current["status"] {
		case "Planned":
			fmt.Fprintf(out, "Issue %s already has Project Status %q; leaving it unchanged.\n", issueURL, status)
			return 0
		case "In progress", "In review", "Done":
			fmt.Fprintf(out, "Issue %s already has Project Status %q; refusing to regress it to %q.\n", issueURL, current["status"], status)
			return 0
		}
	}
	if dryRun {
		fmt.Fprintf(out, "dry-run: Issue %s would get exactly one Project item with Status %q\n", issueURL, status)
		return 0
	}
	itemID := ""
	if present {
		itemID = item.ID
	} else {
		itemID, err = client.AddItem(number, issueURL)
		if err != nil {
			return hardFailure(err, out, errOut)
		}
	}
	if err := client.SetSingleSelect(projectID, itemID, fields["status"].ID, statusID); err != nil {
		return hardFailure(err, out, errOut)
	}
	fmt.Fprintf(out, "Issue %s Status set to %q\n", issueURL, status)
	return 0
}

// issueIsClosed reports whether the Issue is currently closed.
func issueIsClosed(run Runner, repo string, issueNumber int64) (bool, error) {
	out, err := run.Run("issue", "view", fmt.Sprint(issueNumber), "--repo", repo, "--json", "state")
	if err != nil {
		return false, err
	}
	var view struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal([]byte(out), &view); err != nil {
		return false, fmt.Errorf("issue view output cannot be parsed: %w", err)
	}
	return strings.EqualFold(view.State, "CLOSED"), nil
}

// hardFailure reports a live-operation failure loudly for automation, since
// an unavailable or misconfigured Project must stay visible.
func hardFailure(err error, out, errOut io.Writer) int {
	fmt.Fprintf(errOut, "error: %v\n", err)
	return 1
}
