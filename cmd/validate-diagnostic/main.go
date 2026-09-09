// Command validate-diagnostic validates one or more versioned validator
// diagnostics from a JSONL file.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/hidekitux/skills/internal/diagnostic"
)

func main() {
	fs := flag.NewFlagSet("validate-diagnostic", flag.ContinueOnError)
	input := fs.String("input", "", "JSONL diagnostic file, or - for standard input")
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *input == "" {
		fmt.Fprintln(os.Stderr, "error: -input is required")
		os.Exit(2)
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintln(os.Stderr, "error: -format must be text or json")
		os.Exit(2)
	}
	var data []byte
	var err error
	if *input == "-" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(*input)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	report := diagnostic.ValidateJSONL(data)
	if *format == "json" {
		if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	} else if report.Valid {
		fmt.Printf("validator diagnostics valid: %d diagnostic(s).\n", report.DiagnosticCount)
	} else {
		for _, finding := range report.Findings {
			fmt.Fprintln(os.Stderr, finding)
		}
	}
	if !report.Valid {
		os.Exit(1)
	}
}
