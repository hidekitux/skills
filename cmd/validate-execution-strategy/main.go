// Command validate-execution-strategy validates the repository execution
// strategy policy and its references.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/hidekitux/skills/internal/strategy"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("validate-execution-strategy", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	formatFlag := fs.String("format", "text", "output format: text or json")
	if err := fs.Parse(os.Args[1:]); err != nil {
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
	report := strategy.Validate(root)
	if *formatFlag == "json" {
		if err := json.NewEncoder(os.Stdout).Encode(report); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	} else if report.Valid {
		fmt.Printf("execution strategy policy valid: %d strategies, %d rules, schema version %d.\n", report.StrategyCount, report.RuleCount, report.SchemaVersion)
	} else {
		for _, finding := range report.Findings {
			fmt.Fprintln(os.Stderr, finding)
		}
	}
	if report.Valid {
		return
	}
	os.Exit(1)
}
