// Command evaluate runs the outcome-based behavioral evaluation suite
// locally: it drives one or more host CLIs headlessly in isolated sandboxes,
// applies deterministic assertions, and writes machine-readable JSONL plus
// human-readable Markdown reports. It is the mise task entry point for
// `mise run evaluate` and `mise run evaluate:smoke`.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hidekitux/skills/internal/eval"
	"github.com/hidekitux/skills/internal/support"
)

func main() {
	fs := flag.NewFlagSet("evaluate", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	hostFlag := fs.String("host", "all", "drivers to evaluate: codex, claude-code, opencode, antigravity, or a comma-separated list; all runs every driver")
	smokeOnly := fs.Bool("smoke-only", false, "run only the minimum smoke subset")
	scenarioFlag := fs.String("scenario", "", "run a single scenario by id")
	skillsFlag := fs.String("skills", "", "run scenarios for these skills only (comma-separated)")
	outputFlag := fs.String("output", "", "report output directory (default: evaluations/reports)")
	dryRun := fs.Bool("dry-run", false, "record skipped runs without executing host CLIs")
	modelFlag := fs.String("model", "", "model provenance override (default: agent.low.model from opencode.json)")
	reviewerCmd := fs.String("reviewer-cmd", "", "external rubric reviewer command; receives scenario JSON on stdin, returns scores JSON")
	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(eval.ExitUsage)
	}

	root, err := support.ResolveRoot(*rootFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(eval.ExitUsage)
	}
	hosts, err := eval.ResolveHosts(*hostFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(eval.ExitUsage)
	}

	outputDir := *outputFlag
	if outputDir == "" {
		outputDir = filepath.Join(root, "evaluations", "reports")
	}

	opts := &eval.Options{
		Root:       root,
		Hosts:      hosts,
		SmokeOnly:  *smokeOnly,
		ScenarioID: *scenarioFlag,
		Skills:     splitList(*skillsFlag),
		OutputDir:  outputDir,
		DryRun:     *dryRun,
		Model:      *modelFlag,
	}
	if *reviewerCmd != "" {
		opts.Reviewer = &eval.CommandReviewer{Command: *reviewerCmd}
	}

	os.Exit(eval.Run(context.Background(), opts, os.Stdout, os.Stderr))
}

// splitList splits a comma-separated flag value into trimmed non-empty parts.
func splitList(value string) []string {
	if value == "" {
		return nil
	}
	var parts []string
	for _, part := range strings.Split(value, ",") {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}
