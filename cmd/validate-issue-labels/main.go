package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/hidekitux/skills/internal/validate"
)

func main() {
	fs := flag.NewFlagSet("validate-issue-labels", flag.ContinueOnError)
	labels := fs.String("labels", "", "comma-separated label names from the issue event payload")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	set := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { set[f.Name] = true })
	if !set["labels"] {
		fmt.Fprintln(os.Stderr, "flag: -labels is required")
		fs.Usage()
		os.Exit(2)
	}
	var list []string
	for _, label := range strings.Split(*labels, ",") {
		trimmed := strings.TrimSpace(label)
		if trimmed != "" {
			list = append(list, trimmed)
		}
	}
	os.Exit(validate.CheckIssueLabels(list, os.Stdout, os.Stderr))
}
