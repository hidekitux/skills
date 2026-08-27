package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/fsl"
)

func main() {
	fs := flag.NewFlagSet("mutate-fsl", flag.ContinueOnError)
	changedBase := fs.String("changed-base", "", "mutate only FSL specs changed since this git revision")
	reportPath := fs.String("report", "", "write the retained mutation report to this path")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "mutate-fsl: unexpected arguments:", fs.Args())
		os.Exit(2)
	}
	os.Exit(fsl.MutateFSL(".", os.Stdout, os.Stderr, fsl.MutateOptions{
		ChangedBase: *changedBase,
		ReportPath:  *reportPath,
	}))
}
