package eval

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	skillcontext "github.com/hidekitux/skills/internal/context"
	executionstrategy "github.com/hidekitux/skills/internal/strategy"
	"github.com/hidekitux/skills/internal/trace"
)

// Record is one scenario result. It distinguishes deterministic failures,
// rubric scores, skipped cases, and infrastructure errors (Acceptance
// criterion 3) and records the provenance needed to reproduce the run
// (Acceptance criterion 4): host, model, prompt SHA-256, repository commit,
// and fixture IDs.
type Record struct {
	RunID              string                      `json:"run_id"`
	Scenario           string                      `json:"scenario"`
	Skill              string                      `json:"skill"`
	Kind               string                      `json:"kind"`
	Host               string                      `json:"host"`
	Model              string                      `json:"model,omitempty"`
	Commit             string                      `json:"repo_commit"`
	SkillSourceCommit  string                      `json:"skill_source_commit,omitempty"`
	InstructionVariant string                      `json:"instruction_variant,omitempty"`
	ContextMode        string                      `json:"context_mode,omitempty"`
	Context            *skillcontext.Manifest      `json:"context,omitempty"`
	Deliberation       *trace.Deliberation         `json:"deliberation,omitempty"`
	Comparison         *Comparison                 `json:"comparison,omitempty"`
	Strategy           *executionstrategy.Decision `json:"strategy,omitempty"`
	StrategyComparison *StrategyComparison         `json:"strategy_comparison,omitempty"`
	PromptSHA          string                      `json:"prompt_sha256"`
	Fixtures           []string                    `json:"fixtures,omitempty"`
	Verdict            string                      `json:"verdict"`
	SkipReason         string                      `json:"skip_reason,omitempty"`
	Failures           []string                    `json:"failures,omitempty"`
	RubricScores       map[string]int              `json:"rubric_scores,omitempty"`
	RubricReview       string                      `json:"rubric_review"`
	CorrectionsUsed    int                         `json:"corrections_used"`
	HandoffObserved    bool                        `json:"handoff_observed,omitempty"`
	InfraError         string                      `json:"infra_error,omitempty"`
	StartedAt          string                      `json:"started_at,omitempty"`
	FinishedAt         string                      `json:"finished_at,omitempty"`
	ElapsedMillis      int64                       `json:"elapsed_millis,omitempty"`
	FailureID          string                      `json:"failure_id,omitempty"`
	FailureCause       string                      `json:"failure_cause,omitempty"`
	FailureRecurrence  int                         `json:"failure_recurrence_count,omitempty"`
}

// StrategyComparison records the deterministic policy difference between an
// adaptive decision and a fixed baseline. Provider usage remains separate and
// is reported only when the host exposes it.
type StrategyComparison struct {
	FixedStrategy       string `json:"fixed_strategy"`
	AdaptiveStrategy    string `json:"adaptive_strategy"`
	AdaptiveOutcome     string `json:"adaptive_outcome"`
	Changed             bool   `json:"changed"`
	SafetyPreserved     bool   `json:"safety_preserved"`
	QualityDelta        int    `json:"quality_delta"`
	ValidationTierDelta int    `json:"validation_tier_delta"`
	RetryBoundDelta     int    `json:"retry_bound_delta"`
	ElapsedBoundDelta   int    `json:"elapsed_bound_delta_millis"`
	ParallelismChanged  bool   `json:"parallelism_changed"`
}

// Comparison records the measurable difference between the single-agent
// baseline and the bounded deliberation result. Token and monetary meters are
// explicit about availability because the local HostRunner contract does not
// expose provider usage data.
type Comparison struct {
	Pattern                      string `json:"pattern"`
	BaselineVerdict              string `json:"baseline_verdict"`
	DeliberatedVerdict           string `json:"deliberated_verdict"`
	CandidateCount               int    `json:"candidate_count"`
	CandidatePassCount           int    `json:"candidate_pass_count"`
	CandidateFailureCount        int    `json:"candidate_failure_count"`
	CandidateInfrastructureCount int    `json:"candidate_infrastructure_error_count"`
	DisagreementCount            int    `json:"disagreement_count"`
	FalsePositiveCount           int    `json:"false_positive_count"`
	QualityDelta                 int    `json:"quality_delta"`
	BaselineElapsedMillis        int64  `json:"baseline_elapsed_millis"`
	DeliberationElapsedMillis    int64  `json:"deliberation_elapsed_millis"`
	MarginalElapsedMillis        int64  `json:"marginal_elapsed_millis"`
	MarginalRetries              int    `json:"marginal_retries"`
	InputTokensAvailable         bool   `json:"input_tokens_available"`
	OutputTokensAvailable        bool   `json:"output_tokens_available"`
	ContextTokensAvailable       bool   `json:"context_tokens_available"`
	BaselineContextTokens        *int64 `json:"baseline_context_tokens,omitempty"`
	DeliberationContextTokens    *int64 `json:"deliberation_context_tokens,omitempty"`
	CostAvailable                bool   `json:"cost_available"`
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

	comparisons := make([]Record, 0)
	for _, record := range records {
		if record.Comparison != nil {
			comparisons = append(comparisons, record)
		}
	}
	if len(comparisons) > 0 {
		fmt.Fprintln(w, "\n## Deliberation comparison (Issue 173)")
		fmt.Fprintln(w, "| scenario | pattern | baseline | deliberated | quality_delta | disagreements | false_positives | marginal_elapsed_ms | marginal_retries |")
		fmt.Fprintln(w, "| --- | --- | --- | --- | --- | --- | --- | --- | --- |")
		for _, record := range comparisons {
			comparison := record.Comparison
			fmt.Fprintf(w, "| %s | %s | %s | %s | %d | %d | %d | %d | %d |\n",
				record.Scenario, comparison.Pattern, comparison.BaselineVerdict,
				comparison.DeliberatedVerdict, comparison.QualityDelta,
				comparison.DisagreementCount, comparison.FalsePositiveCount,
				comparison.MarginalElapsedMillis, comparison.MarginalRetries)
		}
	}

	strategyComparisons := make([]Record, 0)
	for _, record := range records {
		if record.StrategyComparison != nil {
			strategyComparisons = append(strategyComparisons, record)
		}
	}
	if len(strategyComparisons) > 0 {
		fmt.Fprintln(w, "\n## Execution strategy comparison (Issue 204)")
		fmt.Fprintln(w, "The quality delta is zero because both paths use the same deterministic scenario assertions. Host usage remains unavailable unless the driver exposes it.")
		fmt.Fprintln(w, "| scenario | fixed | adaptive | outcome | changed | safety_preserved | quality_delta | validation_tier_delta | retry_bound_delta | elapsed_bound_delta_ms | parallelism_changed |")
		fmt.Fprintln(w, "| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |")
		for _, record := range strategyComparisons {
			comparison := record.StrategyComparison
			fmt.Fprintf(w, "| %s | %s | %s | %s | %t | %t | %d | %d | %d | %d | %t |\n",
				record.Scenario, comparison.FixedStrategy, comparison.AdaptiveStrategy,
				comparison.AdaptiveOutcome, comparison.Changed, comparison.SafetyPreserved,
				comparison.QualityDelta, comparison.ValidationTierDelta, comparison.RetryBoundDelta,
				comparison.ElapsedBoundDelta, comparison.ParallelismChanged)
		}
	}

	fmt.Fprintln(w, "\n## Failure recurrence")
	fmt.Fprintln(w, "| failure_id | cause | scenario | host | model | repository_revision | run_id | outcome | recurrence_count |")
	fmt.Fprintln(w, "| --- | --- | --- | --- | --- | --- | --- | --- | --- |")
	for _, record := range records {
		if record.Verdict == VerdictPass && record.FailureID == "" {
			continue
		}
		failureID := record.FailureID
		if failureID == "" {
			failureID = "unregistered"
		}
		cause := record.FailureCause
		if cause == "" {
			cause = "unclassified"
		}
		recurrence := record.FailureRecurrence
		if recurrence == 0 {
			recurrence = 1
		}
		fmt.Fprintf(w, "| %s | %s | %s | %s | %s | %s | %s | %s | %d |\n",
			failureID, cause, record.Scenario, record.Host, record.Model,
			record.Commit, record.RunID, recordOutcome(record), recurrence)
	}

	counts := map[string]int{}
	for _, verdict := range gates {
		counts[verdict]++
	}
	fmt.Fprintln(w, "\n## Summary")
	for _, verdict := range []string{VerdictPass, VerdictFail, VerdictSkipped, VerdictInterrupted, VerdictInfra} {
		fmt.Fprintf(w, "- %s: %d\n", verdict, counts[verdict])
	}

	var failed []Record
	for _, record := range records {
		if record.Verdict == VerdictFail || record.Verdict == VerdictInterrupted || record.Verdict == VerdictInfra {
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

func recordOutcome(record Record) string {
	switch record.Verdict {
	case VerdictFail:
		return "deterministic_failure"
	case VerdictSkipped:
		return "skipped"
	case VerdictInfra:
		return "infrastructure_error"
	case VerdictInterrupted:
		return "behavioral_failure"
	default:
		return "resolved"
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
	case VerdictInterrupted:
		return "execution interrupted"
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
