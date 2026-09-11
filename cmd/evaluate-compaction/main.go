// Command evaluate-compaction runs the same evaluation corpus against full
// and compact skill sources, then compares their deterministic and rubric
// evidence.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hidekitux/skills/internal/eval"
	"github.com/hidekitux/skills/internal/support"
)

type compactionRunner func(context.Context, *eval.Options, io.Writer, io.Writer) int

func runCompactionSource(ctx context.Context, opts *eval.Options, runner compactionRunner) (int, string, error) {
	code := runner(ctx, opts, os.Stdout, os.Stderr)
	reportPath := filepath.Join(opts.OutputDir, opts.RunID+".jsonl")
	info, err := os.Stat(reportPath)
	if err != nil {
		return code, "", fmt.Errorf("%s current report %q is unavailable: %w", opts.InstructionVariant, reportPath, err)
	}
	if !info.Mode().IsRegular() {
		return code, "", fmt.Errorf("%s current report %q is not a regular file", opts.InstructionVariant, reportPath)
	}
	records, err := eval.LoadRecords(reportPath)
	if err != nil {
		return code, "", fmt.Errorf("%s current report %q is unreadable: %w", opts.InstructionVariant, reportPath, err)
	}
	for _, record := range records {
		if record.RunID != opts.RunID {
			return code, "", fmt.Errorf("%s current report %q has run ID %q, want %q", opts.InstructionVariant, reportPath, record.RunID, opts.RunID)
		}
	}
	return code, reportPath, nil
}

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
	runID := time.Now().UTC().Format("20060102T150405.000000000Z")
	runSource := func(skillRoot, variant, reportDir string) (int, string, error) {
		opts := &eval.Options{
			Root:               root,
			SkillRoot:          skillRoot,
			InstructionVariant: variant,
			Hosts:              hosts,
			SmokeOnly:          *smokeOnly,
			Skills:             splitList(*skillsFlag),
			OutputDir:          reportDir,
			RunID:              runID + "-" + variant,
			ContextMode:        "compiled",
		}
		if *reviewerFlag != "" {
			opts.Reviewer = &eval.CommandReviewer{Command: *reviewerFlag}
		}
		return runCompactionSource(context.Background(), opts, eval.Run)
	}
	fullDir := filepath.Join(output, "full")
	compactDir := filepath.Join(output, "compact")
	fullCode, fullReport, fullErr := runSource(*fullFlag, "full", fullDir)
	compactCode, compactReport, compactErr := runSource(*compactFlag, "compact", compactDir)
	if fullErr != nil || compactErr != nil {
		if fullErr != nil {
			fmt.Fprintf(os.Stderr, "evaluate-compaction: full report: %v\n", fullErr)
		}
		if compactErr != nil {
			fmt.Fprintf(os.Stderr, "evaluate-compaction: compact report: %v\n", compactErr)
		}
		if fullCode == eval.ExitUsage || compactCode == eval.ExitUsage {
			os.Exit(eval.ExitUsage)
		}
		os.Exit(eval.ExitInfra)
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
	os.Exit(compactionExitCode(fullCode, compactCode, pair.Results))
}

// compactionExitCode lets the paired comparison distinguish a common
// baseline failure from a compact-source regression. Source assertion codes
// are evidence for the paired report, not a comparison verdict.
func compactionExitCode(fullCode, compactCode int, results []eval.PairResult) int {
	for _, code := range []int{fullCode, compactCode} {
		if code == eval.ExitUsage {
			return eval.ExitUsage
		}
	}
	for _, code := range []int{fullCode, compactCode} {
		if code == eval.ExitInfra {
			return eval.ExitInfra
		}
	}
	for _, result := range results {
		switch result.Status {
		case "fail":
			return eval.ExitAssertion
		case "inconclusive":
			return eval.ExitInfra
		}
	}
	return eval.ExitOK
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
