package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/validate"
)

func main() {
	fs := flag.NewFlagSet("validate-issue-body", flag.ContinueOnError)
	title := fs.String("title", "", "issue title")
	body := fs.String("body", "", "issue body")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if !set["title"] || !set["body"] {
		fmt.Fprintln(os.Stderr, "flag: -title and -body are required")
		fs.Usage()
		os.Exit(2)
	}
	errors := validate.IssueBodyValidationErrors(*title, *body)
	if len(errors) > 0 {
		for _, error := range errors {
			fmt.Fprintf(os.Stderr, "error: %s\n", error)
		}
		os.Exit(1)
	}
	fmt.Println("Issue body is valid.")
}
