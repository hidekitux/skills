package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/hidekitux/skills/internal/diagnostic"
	"github.com/hidekitux/skills/internal/fsl"
)

func main() {
	os.Exit(run())
}

func run() int {
	fs := flag.NewFlagSet("mutate-fsl", flag.ContinueOnError)
	changedBase := fs.String("changed-base", "", "mutate only FSL specs changed since this git revision")
	reportPath := fs.String("report", "", "write the retained mutation report to this path")
	diagnosticFormat := fs.String("diagnostic-format", "text", "diagnostic format: text or json")
	if err := fs.Parse(os.Args[1:]); err != nil {
		return 2
	}
	if *diagnosticFormat != "text" && *diagnosticFormat != "json" {
		fmt.Fprintln(os.Stderr, "error: -diagnostic-format must be text or json")
		return 2
	}
	if fs.NArg() > 0 {
		fmt.Fprintln(os.Stderr, "mutate-fsl: unexpected arguments:", fs.Args())
		return 2
	}

	output := io.Writer(os.Stdout)
	effectiveReportPath := *reportPath
	cleanup := func() {}
	if *diagnosticFormat == "json" {
		output = io.Discard
		if effectiveReportPath == "" {
			file, err := os.CreateTemp("", "skills-mutation-report-*.json")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 2
			}
			effectiveReportPath = file.Name()
			file.Close()
			cleanup = func() { _ = os.Remove(effectiveReportPath) }
		}
	}
	defer cleanup()

	code := fsl.MutateFSL(".", output, os.Stderr, fsl.MutateOptions{
		ChangedBase: *changedBase,
		ReportPath:  effectiveReportPath,
	})
	if *diagnosticFormat == "json" {
		data, err := os.ReadFile(effectiveReportPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		var report fsl.MutationReport
		if err := json.Unmarshal(data, &report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		diagnostics, err := fsl.DiagnosticsForReport(report)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
		if err := diagnostic.WriteJSONL(os.Stdout, diagnostics); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 2
		}
	}
	return code
}
