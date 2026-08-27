// Command set-issue-project-status sets the governing Issue's Project Status
// in the repository-declared Project, adding the item exactly once when
// missing. It is used by the Pull Request Project Status automation: pass
// --pr-type (plus --pr-draft and --pr-merged) to derive the Status from a
// pull_request event, or pass --status directly. Identifiers are resolved
// from declared names before any mutation, and a closed Issue is never moved
// away from Done.
package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"

	"github.com/hidekitux/skills/internal/project"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("set-issue-project-status", flag.ContinueOnError)
	root := fs.String("root", "", "repository root (default: current working directory)")
	repo := fs.String("repo", "", "repository owner/name")
	issue := fs.Int64("issue", 0, "Issue number")
	status := fs.String("status", "", "Status option to apply (when not deriving from an event)")
	prType := fs.String("pr-type", "", "pull_request event type: opened, reopened, synchronize, ready_for_review, converted_to_draft, closed")
	prDraft := fs.String("pr-draft", "false", "whether the Pull Request is a draft")
	prMerged := fs.String("pr-merged", "false", "whether a closed Pull Request was merged")
	dryRun := fs.Bool("dry-run", false, "resolve and report without mutating")
	config := fs.String("config", "", "Project configuration path (default: <root>/.github/issue-project.toml)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if !set["repo"] || !set["issue"] {
		fmt.Fprintln(os.Stderr, "flag: --repo and --issue are required")
		fs.Usage()
		os.Exit(2)
	}
	if !set["status"] && !set["pr-type"] {
		fmt.Fprintln(os.Stderr, "flag: --status or --pr-type is required")
		fs.Usage()
		os.Exit(2)
	}
	repoRoot, err := support.ResolveRoot(*root)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	configPath := *config
	if configPath == "" {
		configPath = project.ConfigPath(repoRoot)
	}
	cfg, err := project.LoadConfig(configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(2)
	}

	target := *status
	skipClosed := false
	if set["pr-type"] {
		draft, err := strconv.ParseBool(*prDraft)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: --pr-draft must be true or false\n")
			os.Exit(2)
		}
		merged, err := strconv.ParseBool(*prMerged)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: --pr-merged must be true or false\n")
			os.Exit(2)
		}
		derived, change, err := project.PullRequestStatus(project.PullRequestEvent{
			Type: *prType, Draft: draft, Merged: merged,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(2)
		}
		if !change {
			fmt.Fprintln(os.Stdout, "Merged Pull Request: no Project Status change; "+
				"the linked Issue reaches Done through the built-in Item closed workflow.")
			return
		}
		target = derived
		skipClosed = true
	}

	os.Exit(project.SetIssueStatus(project.GH{}, cfg, *repo, *issue, target, *dryRun, skipClosed, os.Stdout, os.Stderr))
}
