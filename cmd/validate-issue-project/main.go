// Command validate-issue-project checks that a GitHub Issue has exactly one
// item in the repository-declared Project with valid Status, Priority, and
// Scope values. It fails safely when Project access is unavailable.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/project"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("validate-issue-project", flag.ContinueOnError)
	root := fs.String("root", "", "repository root (default: current working directory)")
	repo := fs.String("repo", "", "repository owner/name")
	issue := fs.Int64("issue", 0, "Issue number")
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
	os.Exit(project.CheckIssueProject(project.GH{}, cfg, *repo, *issue, os.Stdout, os.Stderr))
}
