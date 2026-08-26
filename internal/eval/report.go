package eval

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

// Record is one scenario result. It distinguishes deterministic failures,
// rubric scores, skipped cases, and infrastructure errors (Acceptance
// criterion 3) and records the provenance needed to reproduce the run
// (Acceptance criterion 4): host, model, prompt SHA-256, repository commit,
// and fixture IDs.
type Record struct {
	RunID           string         `json:"run_id"`
	Scenario        string         `json:"scenario"`
	Skill           string         `json:"skill"`
	Kind            string         `json:"kind"`
	Host            string         `json:"host"`
	Model           string         `json:"model,omitempty"`
	Commit          string         `json:"repo_commit"`
	PromptSHA       string         `json:"prompt_sha256"`
	Fixtures        []string       `json:"fixtures,omitempty"`
	Verdict         string         `json:"verdict"`
	SkipReason      string         `json:"skip_reason,omitempty"`
	Failures        []string       `json:"failures,omitempty"`
	RubricScores    map[string]int `json:"rubric_scores,omitempty"`
	RubricReview    string         `json:"rubric_review"`
	CorrectionsUsed int            `json:"corrections_used"`
	InfraError      string         `json:"infra_error,omitempty"`
}

// writeJSONL appends one JSON record per scenario result.
func writeJSONL(path string, records []Record) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	for _, record := range records {
		if err := encoder.Encode(record); err != nil {
			return err
		}
	}
	return nil
}

// markdownSummary writes a human-readable report for a run. Per-driver
// records are listed first; the aggregate section reports the either-pass
// gate verdict per scenario (Issue 173 decision record).
func markdownSummary(w io.Writer, records []Record, gates map[string]string, model, commit string) {
	fmt.Fprintf(w, "# Behavioral evaluation report\n\n")
	fmt.Fprintf(w, "Run %s on commit %s (model %s).\n\n", records[0].RunID, commit, model)
	fmt.Fprintf(w, "| scenario | skill | kind | driver | verdict | result |\n")
	fmt.Fprintf(w, "| --- | --- | --- | --- | --- | --- |\n")
	for _, record := range records {
		result := describeResult(record)
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s |\n",
			record.Scenario, record.Skill, record.Kind, record.Host, record.Verdict, result)
	}

	fmt.Fprintln(w, "\n## Aggregate per scenario (either-pass policy)")
	fmt.Fprintln(w, "| scenario | gate |")
	fmt.Fprintln(w, "| --- | --- |")
	seen := map[string]bool{}
	for _, record := range records {
		if seen[record.Scenario] {
			continue
		}
		seen[record.Scenario] = true
		fmt.Fprintf(w, "| %s | %s |\n", record.Scenario, gates[record.Scenario])
	}

	counts := map[string]int{}
	for _, verdict := range gates {
		counts[verdict]++
	}
	fmt.Fprintln(w, "\n## Summary")
	for _, verdict := range []string{VerdictPass, VerdictFail, VerdictSkipped, VerdictInfra} {
		fmt.Fprintf(w, "- %s: %d\n", verdict, counts[verdict])
	}

	var failed []Record
	for _, record := range records {
		if record.Verdict == VerdictFail || record.Verdict == VerdictInfra {
			failed = append(failed, record)
		}
	}
	if len(failed) > 0 {
		fmt.Fprintln(w, "\n## Failures and infrastructure errors")
		sort.Slice(failed, func(i, j int) bool { return failed[i].Scenario < failed[j].Scenario })
		for _, record := range failed {
			fmt.Fprintf(w, "\n### %s (%s) on %s\n", record.Scenario, record.Skill, record.Host)
			if record.InfraError != "" {
				fmt.Fprintf(w, "Infrastructure error: %s\n", record.InfraError)
			}
			if len(record.Failures) > 0 {
				for _, failure := range record.Failures {
					fmt.Fprintf(w, "- %s\n", failure)
				}
			}
		}
	}
}

// describeResult returns the compact human-readable result cell for a record.
func describeResult(record Record) string {
	switch record.Verdict {
	case VerdictPass:
		if len(record.RubricScores) > 0 {
			return scoreSummary(record.RubricScores)
		}
		return "deterministic assertions passed"
	case VerdictFail:
		first := "deterministic failure"
		if len(record.Failures) > 0 {
			first = record.Failures[0]
		}
		return first
	case VerdictSkipped:
		return "skipped: " + record.SkipReason
	case VerdictInfra:
		return "infrastructure error"
	default:
		return record.Verdict
	}
}

// scoreSummary renders the rubric scores in deterministic dimension order.
var scoreOrder = []string{
	"trigger_selection",
	"task_completion",
	"evidence_quality",
	"scope_control",
	"safety",
	"user_correction_count",
	"handoff_quality",
}

func scoreSummary(scores map[string]int) string {
	parts := make([]string, 0, len(scores))
	for _, dimension := range scoreOrder {
		if score, ok := scores[dimension]; ok {
			parts = append(parts, fmt.Sprintf("%s=%d", dimension, score))
		}
	}
	if len(parts) == 0 {
		return "rubric pending"
	}
	return strings.Join(parts, " ")
}
