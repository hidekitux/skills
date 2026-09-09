// Command validate-skill-trace validates redacted, structured skill traces.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/support"
	"github.com/hidekitux/skills/internal/trace"
)

func main() {
	fs := flag.NewFlagSet("validate-skill-trace", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	inputFlag := fs.String("input", "", "JSONL trace file")
	formatFlag := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *inputFlag == "" {
		fmt.Fprintln(os.Stderr, "error: --input is required")
		os.Exit(2)
	}
	if *formatFlag != "text" && *formatFlag != "json" {
		fmt.Fprintln(os.Stderr, "error: --format must be text or json")
		os.Exit(2)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	report := trace.ValidateJSONL(*inputFlag, root)
	if *formatFlag == "json" {
		if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	} else if report.Valid {
		fmt.Printf("skill trace valid: %d trace(s).\n", report.TraceCount)
	} else {
		for _, finding := range report.Findings {
			fmt.Fprintln(os.Stderr, finding)
		}
	}
	if !report.Valid {
		os.Exit(1)
	}
}
