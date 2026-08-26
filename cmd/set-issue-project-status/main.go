// Command set-issue-project-status sets the governing Issue's Project Status
// in the repository-declared Project, adding the item exactly once when
// missing. It is used by the reopen automation and by manual transition
// verification; identifiers are resolved from declared names before mutation.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/project"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("set-issue-project-status", flag.ContinueOnError)
	root := fs.String("root", "", "repository root (default: current working directory)")
	repo := fs.String("repo", "", "repository owner/name")
	issue := fs.Int64("issue", 0, "Issue number")
	status := fs.String("status", "", "Status option to apply")
	dryRun := fs.Bool("dry-run", false, "resolve and report without mutating")
	config := fs.String("config", "", "Project configuration path (default: <root>/.github/issue-project.toml)")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if !set["repo"] || !set["issue"] || !set["status"] {
		fmt.Fprintln(os.Stderr, "flag: --repo, --issue, and --status are required")
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
	os.Exit(project.SetIssueStatus(project.GH{}, cfg, *repo, *issue, *status, *dryRun, os.Stdout, os.Stderr))
}
