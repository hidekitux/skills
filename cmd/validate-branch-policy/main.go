package main

import (
	"flag"
	"os"

	"github.com/hidekitux/skills/internal/validate"
)

func main() {
	fs := flag.NewFlagSet("validate-branch-policy", flag.ContinueOnError)
	config := fs.String("config", ".github/branch-policy.toml", "branch policy configuration path")
	base := fs.String("base", os.Getenv("PR_BASE_REF"), "pull request base branch")
	head := fs.String("head", os.Getenv("PR_HEAD_REF"), "pull request head branch")
	body := fs.String("body", os.Getenv("PR_BODY"), "pull request body")
	validateConfig := fs.Bool("validate-config", false, "only validate the configuration file")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	os.Exit(validate.CheckBranchPolicy(*config, *base, *head, *body, *validateConfig, os.Stdout, os.Stderr))
}
