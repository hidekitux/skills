package eval

import (
	"context"
	"io"
	"sync"
	"time"

	"github.com/hidekitux/skills/internal/trace"
)

// runOneDeliberation compares one bounded, read-only fan-out with the same
// scenario executed once. Candidate scenarios remove the deliberation block
// before execution, which keeps candidate prompts, sandboxes, and context
// independent.
func runOneDeliberation(ctx context.Context, sc *Scenario, host HostRunner, opts *Options, out, errOut io.Writer) Record {
	started := time.Now()
	baselineScenario := *sc
	baselineScenario.Deliberation = nil

	boundedCtx, cancel := context.WithTimeout(ctx, time.Duration(sc.Deliberation.Bounds.MaxElapsedMillis)*time.Millisecond)
	defer cancel()
	baseline := runOneSingle(boundedCtx, &baselineScenario, host, opts, out, errOut)
	if baseline.Verdict != VerdictPass && baseline.Verdict != VerdictFail {
		return baseline
	}

	candidates := make([]Record, sc.Deliberation.CandidateCount)
	var waitGroup sync.WaitGroup
	for index := range candidates {
		candidateScenario := baselineScenario
		waitGroup.Add(1)
		go func(index int, scenario Scenario) {
			defer waitGroup.Done()
			candidates[index] = runOneSingle(boundedCtx, &scenario, host, opts, out, errOut)
		}(index, candidateScenario)
	}
	waitGroup.Wait()

	selectedIndex, selected := judgeDeliberation(baseline, candidates)
	record := baseline
	if selectedIndex >= 0 {
		record.Verdict = selected.Verdict
		record.SkipReason = selected.SkipReason
		record.Failures = selected.Failures
		record.RubricScores = selected.RubricScores
		record.RubricReview = selected.RubricReview
		record.InfraError = selected.InfraError
		record.HandoffObserved = selected.HandoffObserved
	}
	record.CorrectionsUsed = baseline.CorrectionsUsed
	for _, candidate := range candidates {
		record.CorrectionsUsed += candidate.CorrectionsUsed
	}

	deliberation := deliberationTrace(sc.Deliberation, baseline, candidates, selectedIndex)
	record.Deliberation = &deliberation
	record.Comparison = deliberationComparison(sc, baseline, candidates, record, time.Since(started))
	if boundedCtx.Err() == context.DeadlineExceeded {
		record.Verdict = VerdictInfra
		record.InfraError = "deliberation exceeded max_elapsed_millis"
	}
	return record
}

// judgeDeliberation applies an evidence-required deterministic rule. A
// passing candidate is preferred in declared order; otherwise the baseline is
// retained when it passes, and the first deterministic failure is retained.
// The rule never counts a majority and never consumes candidate transcripts.
func judgeDeliberation(baseline Record, candidates []Record) (int, Record) {
	for index, candidate := range candidates {
		if candidate.Verdict == VerdictPass {
			return index, candidate
		}
	}
	if baseline.Verdict == VerdictPass {
		return -1, baseline
	}
	for index, candidate := range candidates {
		if candidate.Verdict == VerdictFail {
			return index, candidate
		}
	}
	return -1, baseline
}

func deliberationTrace(spec *DeliberationSpec, baseline Record, candidates []Record, selectedIndex int) trace.Deliberation {
	output := trace.Deliberation{
		Pattern:      spec.Pattern,
		Signals:      append([]string(nil), spec.Signals...),
		Reason:       spec.Reason,
		Independence: spec.Independence,
		Authority:    spec.Authority,
		Concurrency:  spec.Concurrency,
		Bounds: trace.DeliberationBounds{
			MaxAgents:        spec.Bounds.MaxAgents,
			MaxRetries:       spec.Bounds.MaxRetries,
			MaxElapsedMillis: spec.Bounds.MaxElapsedMillis,
			MaxInputTokens:   spec.Bounds.MaxInputTokens,
			MaxOutputTokens:  spec.Bounds.MaxOutputTokens,
			MaxCostMicros:    spec.Bounds.MaxCostMicros,
		},
		Candidates: make([]trace.Candidate, 0, len(candidates)),
	}
	for index, candidate := range candidates {
		output.Candidates = append(output.Candidates, trace.Candidate{
			ID:       candidateID(index),
			Result:   candidateResult(candidate),
			Evidence: []trace.Evidence{{Kind: "validation", Ref: "check-evaluation", Result: candidateResult(candidate)}},
		})
	}
	decision := "baseline"
	if selectedIndex >= 0 {
		decision = candidateID(selectedIndex)
	}
	selected := baseline
	if selectedIndex >= 0 {
		selected = candidates[selectedIndex]
	}
	output.Judge = &trace.Judge{
		Decision: decision,
		Evidence: []trace.Evidence{{Kind: "validation", Ref: "check-evaluation", Result: candidateResult(selected)}},
	}
	output.MarginalCost = &trace.MarginalCost{
		Available:     false,
		ElapsedMillis: deliberationElapsed(baseline, candidates),
		Retries:       marginalRetries(candidates),
	}
	if baseline.Context != nil {
		contextTokens := int64(len(candidates) * baseline.Context.TotalTokens)
		output.MarginalCost.ContextTokens = &contextTokens
	}
	return output
}

func deliberationComparison(sc *Scenario, baseline Record, candidates []Record, selected Record, elapsed time.Duration) *Comparison {
	comparison := &Comparison{
		Pattern:                   sc.Deliberation.Pattern,
		BaselineVerdict:           baseline.Verdict,
		DeliberatedVerdict:        selected.Verdict,
		CandidateCount:            len(candidates),
		BaselineElapsedMillis:     baseline.ElapsedMillis,
		DeliberationElapsedMillis: elapsed.Milliseconds(),
		MarginalElapsedMillis:     elapsed.Milliseconds() - baseline.ElapsedMillis,
		MarginalRetries:           marginalRetries(candidates),
	}
	if comparison.MarginalElapsedMillis < 0 {
		comparison.MarginalElapsedMillis = 0
	}
	for _, candidate := range candidates {
		switch candidate.Verdict {
		case VerdictPass:
			comparison.CandidatePassCount++
		case VerdictFail:
			comparison.CandidateFailureCount++
			if sc.Kind == KindPositive && baseline.Verdict == VerdictPass {
				comparison.FalsePositiveCount++
			}
		case VerdictInfra:
			comparison.CandidateInfrastructureCount++
		}
		if candidate.Verdict != baseline.Verdict {
			comparison.DisagreementCount++
		}
	}
	comparison.QualityDelta = verdictScore(selected.Verdict) - verdictScore(baseline.Verdict)
	if baseline.Context != nil {
		comparison.ContextTokensAvailable = true
		baselineTokens := int64(baseline.Context.TotalTokens)
		deliberationTokens := int64(len(candidates) * baseline.Context.TotalTokens)
		comparison.BaselineContextTokens = &baselineTokens
		comparison.DeliberationContextTokens = &deliberationTokens
	}
	return comparison
}

func deliberationElapsed(baseline Record, candidates []Record) int64 {
	var total int64 = baseline.ElapsedMillis
	for _, candidate := range candidates {
		total += candidate.ElapsedMillis
	}
	return total - baseline.ElapsedMillis
}

func marginalRetries(candidates []Record) int {
	total := 0
	for _, candidate := range candidates {
		total += candidate.CorrectionsUsed
	}
	return total
}

func candidateID(index int) string {
	return "candidate-" + string(rune('a'+index))
}

func candidateResult(record Record) string {
	switch record.Verdict {
	case VerdictPass:
		return "pass"
	case VerdictFail:
		return "deterministic-failure"
	case VerdictInterrupted:
		return "interrupted"
	case VerdictSkipped:
		return "skipped"
	default:
		return "infrastructure-error"
	}
}

func verdictScore(verdict string) int {
	if verdict == VerdictPass {
		return 1
	}
	return 0
}
