package main

import (
	"flag"
	"os"

	"github.com/hidekitux/skills/internal/validate"
)

func main() {
	fs := flag.NewFlagSet("validate-pr-commit-signatures", flag.ContinueOnError)
	repo := fs.String("repo", "", "repository in OWNER/REPO form")
	pullRequest := fs.Int("pull-request", 0, "positive pull request number")
	commitsJSON := fs.String("commits-json", "", "local API response fixture")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	os.Exit(validate.CheckPrCommitSignatures(*repo, *pullRequest, *commitsJSON, os.Stdout, os.Stderr))
}
