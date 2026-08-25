package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/validate"
)

func main() {
	fs := flag.NewFlagSet("validate-work-item-title", flag.ContinueOnError)
	title := fs.String("title", "", "work item title to validate")
	author := fs.String("author", os.Getenv("PR_AUTHOR"), "pull request author")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if !set["title"] {
		fmt.Fprintln(os.Stderr, "flag: -title is required")
		fs.Usage()
		os.Exit(2)
	}
	os.Exit(validate.CheckWorkItemTitle(*title, *author, os.Stdout, os.Stderr))
}
