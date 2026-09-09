// Command validate-skill-graph validates the repository-owned skill graph.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/graph"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("validate-skill-graph", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	format := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(2)
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintln(os.Stderr, "error: --format must be text or json")
		os.Exit(2)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	report := graph.Validate(root)
	if *format == "json" {
		if err := graph.WriteJSON(os.Stdout, report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	} else if report.Valid {
		fmt.Fprintf(os.Stdout, "skill graph valid: %d skills, schema version %d\n", report.SkillCount, report.SchemaVersion)
	} else {
		for _, finding := range report.Findings {
			fmt.Fprintln(os.Stderr, finding)
		}
	}
	if !report.Valid {
		os.Exit(1)
	}
}
