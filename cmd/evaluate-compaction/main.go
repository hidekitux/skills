// Command evaluate-compaction runs the same evaluation corpus against full
// and compact skill sources, then compares their deterministic and rubric
// evidence.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hidekitux/skills/internal/eval"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("evaluate-compaction", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root containing scenarios and fixtures (default: current working directory)")
	fullFlag := fs.String("full-skill-root", "", "repository root containing the full instruction source")
	compactFlag := fs.String("compact-skill-root", "", "repository root containing the compact instruction source")
	hostFlag := fs.String("host", "codex,claude-code", "drivers to evaluate")
	smokeOnly := fs.Bool("smoke-only", false, "run only the smoke scenarios")
	skillsFlag := fs.String("skills", "", "skills to evaluate (comma-separated)")
	outputFlag := fs.String("output", "", "comparison output directory")
	reviewerFlag := fs.String("reviewer-cmd", "", "external rubric reviewer command")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(3)
	}
	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(3)
	}
	if *fullFlag == "" || *compactFlag == "" {
		fmt.Fprintln(os.Stderr, "evaluate-compaction: --full-skill-root and --compact-skill-root are required")
		os.Exit(3)
	}
	output := *outputFlag
	if output == "" {
		output = filepath.Join(root, "evaluations", "reports", "compaction", time.Now().UTC().Format("20060102T150405Z"))
	}
	hosts := splitList(*hostFlag)
	common := func(skillRoot, variant, reportDir string) int {
		opts := &eval.Options{
			Root:               root,
			SkillRoot:          skillRoot,
			InstructionVariant: variant,
			Hosts:              hosts,
			SmokeOnly:          *smokeOnly,
			Skills:             splitList(*skillsFlag),
			OutputDir:          reportDir,
		}
		if *reviewerFlag != "" {
			opts.Reviewer = &eval.CommandReviewer{Command: *reviewerFlag}
		}
		return eval.Run(context.Background(), opts, os.Stdout, os.Stderr)
	}
	fullDir := filepath.Join(output, "full")
	compactDir := filepath.Join(output, "compact")
	fullCode := common(*fullFlag, "full", fullDir)
	compactCode := common(*compactFlag, "compact", compactDir)
	fullReport, err := eval.LatestReport(fullDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "evaluate-compaction: full report: %v\n", err)
		os.Exit(2)
	}
	compactReport, err := eval.LatestReport(compactDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "evaluate-compaction: compact report: %v\n", err)
		os.Exit(2)
	}
	pair, err := eval.CompareReports(fullReport, compactReport)
	if err != nil {
		fmt.Fprintf(os.Stderr, "evaluate-compaction: compare reports: %v\n", err)
		os.Exit(2)
	}
	if err := eval.WritePairReport(output, pair, os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "evaluate-compaction: write comparison: %v\n", err)
		os.Exit(2)
	}
	for _, result := range pair.Results {
		if result.Status != "pass" {
			fmt.Fprintf(os.Stderr, "comparison %s/%s: %s (%s)\n", result.Scenario, result.Host, result.Status, strings.Join(result.Reasons, "; "))
		}
	}
	for _, code := range []int{fullCode, compactCode} {
		if code == eval.ExitAssertion {
			os.Exit(eval.ExitAssertion)
		}
	}
	for _, result := range pair.Results {
		if result.Status == "fail" {
			os.Exit(eval.ExitAssertion)
		}
		if result.Status == "inconclusive" {
			os.Exit(eval.ExitInfra)
		}
	}
}

func splitList(value string) []string {
	var result []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			result = append(result, item)
		}
	}
	return result
}
