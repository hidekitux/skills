// Command backfill-issue-project migrates triage-labelled Issues into the
// repository-declared GitHub Project. It derives Project field values from
// the current triage labels and Issue state, reconciles exactly one Project
// item per Issue, verifies the result, and only then removes or retires the
// migrated labels. Every mode is gated: label removal requires a passing
// backfill verification, and label retirement requires proof that no Issue
// retains a migrated label.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/project"
	"github.com/hidekitux/skills/internal/support"
)

var modes = map[string]bool{
	"dry-run":       true,
	"apply":         true,
	"verify":        true,
	"remove-labels": true,
	"verify-labels": true,
	"retire-labels": true,
}

func main() {
	fs := flag.NewFlagSet("backfill-issue-project", flag.ContinueOnError)
	root := fs.String("root", "", "repository root (default: current working directory)")
	repo := fs.String("repo", "", "repository owner/name")
	mode := fs.String("mode", "dry-run", "migration mode: dry-run, apply, verify, remove-labels, verify-labels, retire-labels")
	config := fs.String("config", "", "Project configuration path (default: <root>/.github/issue-project.toml)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if !set["repo"] {
		fmt.Fprintln(os.Stderr, "flag: --repo is required")
		fs.Usage()
		os.Exit(2)
	}
	if !modes[*mode] {
		fmt.Fprintf(os.Stderr, "error: unknown --mode %q\n", *mode)
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
	run := project.GH{}

	switch *mode {
	case "dry-run":
		plans, _, err := project.PlanBackfill(run, cfg, *repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		printPlans(plans)
	case "apply":
		plans, _, err := project.PlanBackfill(run, cfg, *repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		count, err := project.ApplyBackfill(run, cfg, plans)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Backfilled %d Issue(s) into the declared Project.\n", count)
	case "verify":
		plans, _, err := project.PlanBackfill(run, cfg, *repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := project.VerifyBackfill(run, cfg, plans); err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Verified %d Issue(s) have exactly one Project item with valid field values.\n", len(plans))
	case "remove-labels":
		plans, _, err := project.PlanBackfill(run, cfg, *repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if err := project.VerifyBackfill(run, cfg, plans); err != nil {
			fmt.Fprintf(os.Stderr, "error: backfill verification failed; labels were not removed: %v\n", err)
			os.Exit(1)
		}
		removed := 0
		for _, plan := range plans {
			labels, err := project.RemoveMigratedLabels(run, *repo, plan.Issue)
			if err != nil {
				fmt.Fprintf(os.Stderr, "error: %v\n", err)
				os.Exit(1)
			}
			removed += len(labels)
		}
		fmt.Printf("Removed %d migrated triage label(s) after verified backfill.\n", removed)
	case "verify-labels":
		offenders, err := project.VerifyLabelsGone(run, *repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if len(offenders) > 0 {
			fmt.Fprintf(os.Stderr, "error: %d Issue(s) still retain a migrated triage label: %v\n", len(offenders), offenders)
			os.Exit(1)
		}
		fmt.Println("No Issue retains a migrated triage label.")
	case "retire-labels":
		offenders, err := project.VerifyLabelsGone(run, *repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		if len(offenders) > 0 {
			fmt.Fprintf(os.Stderr, "error: %d Issue(s) still retain a migrated triage label; definitions were not retired: %v\n", len(offenders), offenders)
			os.Exit(1)
		}
		retired, err := project.RetireLabelDefinitions(run, *repo)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Retired %d label definition(s).\n", len(retired))
	}
}

func printPlans(plans []project.BackfillPlan) {
	for _, plan := range plans {
		fmt.Printf("#%d [%s] -> Status=%s, Priority=%s, Scope=%s\n",
			plan.Issue.Number, plan.Issue.State, plan.Values.Status, plan.Values.Priority, plan.Values.Scope)
	}
	fmt.Printf("%d Issue(s) carry a migrated triage label and need a Project item; "+
		"run with --mode apply after review.\n", len(plans))
}
