// Command replay-skill-trace validates an ordered structured trace set and
// replays its normalized observations through the repository FSL model.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hidekitux/skills/internal/fsl"
	"github.com/hidekitux/skills/internal/replay"
	"github.com/hidekitux/skills/internal/support"
)

type commandReport struct {
	replay.Report
	FSLConformant bool `json:"fsl_conformant"`
}

func main() {
	fs := flag.NewFlagSet("replay-skill-trace", flag.ContinueOnError)
	rootFlag := fs.String("root", "", "repository root (default: current working directory)")
	inputFlag := fs.String("input", "", "ordered JSONL structured trace file")
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
	set, err := replay.ReadJSONL(*inputFlag)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	report := replay.Replay(root, set)
	result := commandReport{Report: report}
	if report.Outcome != replay.OutcomeViolation && report.Outcome != replay.OutcomeInvalidInput {
		publicTrace, traceErr := replay.PublicTraceForReport(report)
		if traceErr != nil {
			result.Findings = append(result.Findings, replay.Finding{Invariant: "FSLObservationReplay", Category: "incomplete", Message: "normalized observations could not be prepared"})
			result.Valid = false
			result.Outcome = replay.OutcomeIncomplete
		} else {
			traceFile, fileErr := os.CreateTemp("", "skills-replay-*.json")
			if fileErr != nil {
				result.Findings = append(result.Findings, replay.Finding{Invariant: "FSLObservationReplay", Category: "infrastructure", Message: "temporary FSL trace could not be created"})
				result.Valid = false
				result.Outcome = replay.OutcomeFailed
			} else {
				tracePath := traceFile.Name()
				defer os.Remove(tracePath)
				encodeErr := json.NewEncoder(traceFile).Encode(publicTrace)
				closeErr := traceFile.Close()
				if encodeErr != nil || closeErr != nil {
					result.Findings = append(result.Findings, replay.Finding{Invariant: "FSLObservationReplay", Category: "infrastructure", Message: "temporary FSL trace could not be written"})
					result.Valid = false
					result.Outcome = replay.OutcomeFailed
				} else {
					spec := filepath.Join(root, "specs", "cross-skill-workflow.fsl")
					if code := fsl.ReplayTrace(spec, tracePath, os.Stderr, os.Stderr); code != 0 {
						result.Findings = append(result.Findings, replay.Finding{Invariant: "FSLObservationReplay", Category: "violation", Message: "normalized observations did not conform to the FSL model"})
						result.Valid = false
						result.Outcome = replay.OutcomeViolation
					} else {
						result.FSLConformant = true
					}
				}
			}
		}
	}
	if *formatFlag == "json" {
		if err := json.NewEncoder(os.Stdout).Encode(result); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	} else {
		if result.Valid && result.Outcome == replay.OutcomeValid {
			fmt.Printf("skill trace replay valid: %d step(s); FSL observations conformant.\n", result.StepsChecked)
		} else {
			fmt.Printf("skill trace replay outcome: %s\n", result.Outcome)
			for _, finding := range result.Findings {
				fmt.Fprintf(os.Stderr, "%s: %s\n", finding.Invariant, finding.Message)
			}
		}
	}
	if result.Valid && result.Outcome == replay.OutcomeValid {
		os.Exit(0)
	}
	os.Exit(1)
}
