package main

import (
	"flag"
	"os"

	"github.com/hidekitux/skills/internal/diagnose"
)

func main() {
	fs := flag.NewFlagSet("diagnose-worktree", flag.ContinueOnError)
	branch := fs.String("branch", "", "check the worktree that owns this branch")
	base := fs.String("base", "origin/main", "merged-state base branch")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	os.Exit(diagnose.DiagnoseWorktree(*branch, *base, os.Stdout, os.Stderr))
}
