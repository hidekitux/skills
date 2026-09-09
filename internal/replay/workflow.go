package replay

import (
	"fmt"
	"strings"

	"github.com/hidekitux/skills/internal/graph"
	"github.com/hidekitux/skills/internal/trace"
)

const crossSkillSpec = "CrossSkillWorkflow"

type toolRule struct {
	allowed map[string]bool
}

// canonicalTools is the closed host-neutral map used for authority checks.
// Report-only operations do not grant write authority to the reporting skill.
var canonicalTools = map[string]toolRule{
	"read-repository":        {allowed: allSkills()},
	"read-git":               {allowed: allSkills()},
	"read-github":            {allowed: allSkills()},
	"user-approval":          {allowed: allSkills()},
	"create-issue":           {allowed: only("create-issue")},
	"post-plan-comment":      {allowed: only("plan-issue")},
	"create-issue-branch":    {allowed: only("implement-issue")},
	"write-repository":       {allowed: only("implement-issue", "fix-pr")},
	"git-commit":             {allowed: only("implement-issue", "fix-pr")},
	"push-issue-branch":      {allowed: only("create-pr", "fix-pr")},
	"rewrite-issue-branch":   {allowed: only("create-pr", "fix-pr")},
	"edit-pull-request":      {allowed: only("create-pr", "fix-pr")},
	"record-review-findings": {allowed: only("review-pr")},
	"merge-pull-request":     {allowed: only("merge-pr")},
}

var requiredTools = map[string][]string{
	"create-issue":    {"user-approval", "create-issue"},
	"plan-issue":      {"post-plan-comment"},
	"implement-issue": {"create-issue-branch", "write-repository", "git-commit"},
	"create-pr":       {"user-approval", "push-issue-branch", "edit-pull-request"},
	"review-pr":       {"record-review-findings"},
	"fix-pr":          {"write-repository", "git-commit", "push-issue-branch"},
	"merge-pr":        {"user-approval", "merge-pull-request"},
}

func allSkills() map[string]bool {
	return map[string]bool{
		"create-issue": true, "plan-issue": true, "implement-issue": true,
		"create-pr": true, "review-pr": true, "fix-pr": true, "merge-pr": true,
	}
}

func only(skills ...string) map[string]bool {
	result := map[string]bool{}
	for _, skill := range skills {
		result[skill] = true
	}
	return result
}

// Replay validates one ordered trace set against the graph and returns the
// normalized actions that can be passed to fslc replay.
func Replay(root string, set TraceSet) Report {
	report := NewReport(crossSkillSpec)
	loaded, err := graph.Load(root)
	if err != nil {
		report.Findings = []Finding{{Invariant: "GraphAvailable", Category: "input", Message: "skill graph could not be loaded"}}
		return report
	}
	if len(set.Records) == 0 {
		report.Findings = []Finding{{Invariant: "CompleteTraceRequired", Category: "input", Message: "trace set contains no records"}}
		return report
	}

	expectedSkill := "create-issue"
	pendingReview := false
	corrections := 0
	for index, record := range set.Records {
		report.StepsChecked++
		if !record.Validation.Valid {
			if incompleteTrace(record.Trace, record.Validation) {
				return finishIncomplete(report, record, index, "trace is missing a terminal event")
			}
			return finishInvalidInput(report, record, index, "trace does not satisfy the structured trace contract")
		}
		if record.Trace.SkillID != expectedSkill {
			return finishViolation(report, Finding{
				Invariant:     "HandoffMatchesGraph",
				Category:      "ordering",
				Message:       fmt.Sprintf("expected next skill %q", expectedSkill),
				TraceIndex:    index,
				Line:          record.Line,
				EventSequence: firstEventSequence(record.Trace),
			})
		}
		if record.Trace.Terminal.Status == trace.StatusInterrupted {
			return finishInterrupted(report, record, index)
		}
		if record.Trace.Terminal.Status != trace.StatusSuccess {
			return finishViolation(report, Finding{
				Invariant:     "TerminalOutcomeIsNotSuccess",
				Category:      "terminal",
				Message:       "non-success terminal cannot establish a successful handoff",
				TraceIndex:    index,
				Line:          record.Line,
				EventSequence: lastEventSequence(record.Trace),
			})
		}
		if finding, outcome, ok := checkTools(record, index); ok {
			if outcome == OutcomeIncomplete {
				return finishIncompleteFinding(report, finding)
			}
			return finishViolation(report, finding)
		}

		handoff, handoffSeq, handoffCount := handoffEvent(record.Trace)
		if handoffCount > 1 {
			return finishViolation(report, Finding{
				Invariant:     "OneHandoffPerRun",
				Category:      "ordering",
				Message:       "one skill run contains more than one handoff",
				TraceIndex:    index,
				Line:          record.Line,
				EventSequence: handoffSeq,
			})
		}

		if !hasCompletedTodo(record.Trace) || !hasSafeEvidence(record.Trace) || !hasSuccessfulValidation(record.Trace) {
			return finishIncomplete(report, record, index, "successful transition lacks Todo, evidence, or validation")
		}
		if finding, ok := missingRequiredTool(record, index); ok {
			return finishIncompleteFinding(report, finding)
		}

		if handoff == nil {
			if record.Trace.SkillID == "review-pr" && record.Trace.Terminal.Status == trace.StatusSuccess {
				if index != len(set.Records)-1 {
					return finishTrailing(report, record, index)
				}
				if pendingReview {
					return finishViolation(report, Finding{
						Invariant:     "HandoffMatchesGraph",
						Category:      "ordering",
						Message:       "review completed while a correction handoff remained pending",
						TraceIndex:    index,
						Line:          record.Line,
						EventSequence: lastEventSequence(record.Trace),
					})
				}
				report.Observations = append(report.Observations, observation("complete_review", record, index, lastEventSequence(record.Trace)))
				return finishSuccess(report)
			}
			if record.Trace.SkillID == "merge-pr" && record.Trace.Terminal.Status == trace.StatusSuccess {
				report.Observations = append(report.Observations, observation("complete_merge", record, index, lastEventSequence(record.Trace)))
				return finishSuccess(report)
			}
			return finishIncomplete(report, record, index, "successful skill run has no graph handoff")
		}

		outcome := handoff.Outcome
		if outcome == "" {
			outcome = trace.StatusSuccess
		}
		transition, ok := matchingTransition(loaded, record.Trace.SkillID, outcome, handoff.Artifact, handoff.Destination)
		if !ok {
			return finishViolation(report, Finding{
				Invariant:     "HandoffMatchesGraph",
				Category:      "ordering",
				Message:       "handoff does not match an authoritative graph transition",
				TraceIndex:    index,
				Line:          record.Line,
				EventSequence: handoffSeq,
			})
		}

		switch {
		case outcome == "blocked" || handoff.Destination == "blocked":
			if index != len(set.Records)-1 {
				return finishTrailing(report, record, index)
			}
			report.Observations = append(report.Observations, observation("block", record, index, handoffSeq))
			return finishTerminal(report, OutcomeBlocked, true)
		case outcome == "retry_exhausted" || handoff.Destination == "retry_exhausted":
			if index != len(set.Records)-1 {
				return finishTrailing(report, record, index)
			}
			report.Observations = append(report.Observations, observation("exhaust_review_loop", record, index, handoffSeq))
			return finishTerminal(report, OutcomeRetryExhausted, true)
		case record.Trace.SkillID == "review-pr" && outcome == "findings":
			if corrections >= 2 {
				report.Findings = append(report.Findings, Finding{
					Invariant:     "ReviewLoopBounded",
					Category:      "retry",
					Message:       "review findings continued after two correction passes",
					TraceIndex:    index,
					Line:          record.Line,
					EventSequence: handoffSeq,
				})
				report.Observations = append(report.Observations, observation("exhaust_review_loop", record, index, handoffSeq))
				return finishTerminal(report, OutcomeRetryExhausted, false)
			}
			pendingReview = true
		case record.Trace.SkillID == "review-pr" && outcome == "success" && handoff.Destination == "merge-pr":
			report.Observations = append(report.Observations, observation("handoff_merge", record, index, handoffSeq))
		case record.Trace.SkillID == "fix-pr" && outcome == "success":
			if !pendingReview {
				return finishViolation(report, Finding{
					Invariant:     "HandoffMatchesGraph",
					Category:      "ordering",
					Message:       "fix-pr ran without a preceding review finding",
					TraceIndex:    index,
					Line:          record.Line,
					EventSequence: handoffSeq,
				})
			}
			attempt, maxAttempts, found := retryAttempt(record.Trace)
			if !found || maxAttempts != 2 || attempt < 1 || attempt > 2 {
				return finishViolation(report, Finding{
					Invariant:     "ReviewLoopBounded",
					Category:      "retry",
					Message:       "fix-pr retry is missing or exceeds the graph bound of two",
					TraceIndex:    index,
					Line:          record.Line,
					EventSequence: handoffSeq,
				})
			}
			corrections = attempt
			if attempt == 1 {
				report.Observations = append(report.Observations, observation("publish_first_correction", record, index, handoffSeq))
			} else {
				report.Observations = append(report.Observations, observation("publish_second_correction", record, index, handoffSeq))
			}
			pendingReview = false
		default:
			for _, action := range actionsForSkill(record.Trace.SkillID) {
				report.Observations = append(report.Observations, observation(action, record, index, handoffSeq))
			}
		}

		if transition.Destination.Skill != "" {
			expectedSkill = transition.Destination.Skill
			continue
		}
		if transition.Destination.TerminalOutcome != "" {
			if index != len(set.Records)-1 {
				return finishTrailing(report, record, index)
			}
			return finishTerminal(report, terminalOutcome(transition.Destination.TerminalOutcome), true)
		}
		return finishViolation(report, Finding{
			Invariant:     "HandoffMatchesGraph",
			Category:      "ordering",
			Message:       "graph transition has no destination",
			TraceIndex:    index,
			Line:          record.Line,
			EventSequence: handoffSeq,
		})
	}
	return finishIncomplete(report, set.Records[len(set.Records)-1], len(set.Records)-1, "trace set ended before a terminal outcome")
}

func actionsForSkill(skill string) []string {
	switch skill {
	case "create-issue":
		return []string{"create_issue"}
	case "plan-issue":
		return []string{"post_plan"}
	case "implement-issue":
		return []string{"record_implementation", "pass_validation"}
	case "create-pr":
		return []string{"open_pull_request"}
	default:
		return nil
	}
}

func matchingTransition(g *graph.Graph, skillID, outcome, artifact, destination string) (graph.Transition, bool) {
	skill, ok := g.Skill(skillID)
	if !ok {
		return graph.Transition{}, false
	}
	for _, transition := range skill.Transitions {
		if transition.Outcome != outcome || transition.Artifact != artifact {
			continue
		}
		if transition.Destination.Skill == destination || transition.Destination.TerminalOutcome == destination {
			return transition, true
		}
	}
	return graph.Transition{}, false
}

func checkTools(record TraceRecord, index int) (Finding, Outcome, bool) {
	for _, event := range record.Trace.Events {
		if event.Tool == nil {
			continue
		}
		rule, ok := canonicalTools[event.Tool.Name]
		if !ok {
			return Finding{Invariant: "CanonicalToolObservation", Category: "incomplete", Message: "tool name is not in the canonical replay map", TraceIndex: index, Line: record.Line, EventSequence: event.Sequence}, OutcomeIncomplete, true
		}
		if !rule.allowed[record.Trace.SkillID] {
			return Finding{Invariant: "ReadOnlyPhaseHasNoMutation", Category: "authority", Message: "tool operation is not owned by the running skill", TraceIndex: index, Line: record.Line, EventSequence: event.Sequence}, OutcomeViolation, true
		}
		if event.Status != trace.StatusSuccess || event.Tool.Result != trace.StatusSuccess {
			return Finding{Invariant: "RequiredToolObservation", Category: "incomplete", Message: "canonical tool operation did not succeed", TraceIndex: index, Line: record.Line, EventSequence: event.Sequence}, OutcomeIncomplete, true
		}
	}
	return Finding{}, "", false
}

func missingRequiredTool(record TraceRecord, index int) (Finding, bool) {
	required, ok := requiredTools[record.Trace.SkillID]
	if !ok {
		return Finding{}, false
	}
	observed := map[string]bool{}
	for _, event := range record.Trace.Events {
		if event.Tool != nil && event.Status == trace.StatusSuccess && event.Tool.Result == trace.StatusSuccess {
			observed[event.Tool.Name] = true
		}
	}
	for _, name := range required {
		if !observed[name] {
			return Finding{Invariant: "RequiredToolObservation", Category: "incomplete", Message: "required canonical tool operation is missing", TraceIndex: index, Line: record.Line, EventSequence: firstEventSequence(record.Trace)}, true
		}
	}
	return Finding{}, false
}

func incompleteTrace(item trace.Trace, validation trace.ValidationReport) bool {
	if len(item.Events) == 0 {
		return true
	}
	for _, finding := range validation.Findings {
		if strings.Contains(finding, "run_started and run_finished") || strings.Contains(finding, "last event must be run_finished") || strings.Contains(finding, "exactly one run_finished") {
			return true
		}
	}
	return false
}

func handoffEvent(item trace.Trace) (*trace.Handoff, int, int) {
	var result *trace.Handoff
	sequence := 0
	count := 0
	for _, event := range item.Events {
		if event.Handoff != nil {
			copy := *event.Handoff
			result = &copy
			sequence = event.Sequence
			count++
		}
	}
	return result, sequence, count
}

func hasCompletedTodo(item trace.Trace) bool {
	for _, event := range item.Events {
		if event.Todo != nil && event.Todo.To == "completed" && event.Todo.EvidenceRef != "" {
			return true
		}
	}
	return false
}

func hasSafeEvidence(item trace.Trace) bool {
	for _, event := range item.Events {
		if event.Evidence != nil && event.Status == trace.StatusSuccess && event.Evidence.Ref != "" {
			return true
		}
	}
	return false
}

func hasSuccessfulValidation(item trace.Trace) bool {
	for _, event := range item.Events {
		if event.Validation == nil || event.Status != trace.StatusSuccess {
			continue
		}
		switch event.Validation.Result {
		case trace.StatusSuccess, "passed", "pass", "valid":
			return true
		}
	}
	return false
}

func retryAttempt(item trace.Trace) (int, int, bool) {
	for _, event := range item.Events {
		if event.Retry != nil {
			return event.Retry.Attempt, event.Retry.MaxAttempts, true
		}
	}
	return 0, 0, false
}

func firstEventSequence(item trace.Trace) int {
	if len(item.Events) == 0 {
		return 0
	}
	return item.Events[0].Sequence
}

func lastEventSequence(item trace.Trace) int {
	if len(item.Events) == 0 {
		return 0
	}
	return item.Events[len(item.Events)-1].Sequence
}

func observation(action string, record TraceRecord, index, sequence int) Observation {
	return Observation{Action: action, SkillID: record.Trace.SkillID, TraceIndex: index, Line: record.Line, EventSequence: sequence}
}

func terminalOutcome(value string) Outcome {
	switch value {
	case "retry_exhausted":
		return OutcomeRetryExhausted
	case "blocked":
		return OutcomeBlocked
	default:
		return OutcomeValid
	}
}

func finishSuccess(report Report) Report {
	return finishTerminal(report, OutcomeValid, true)
}

func finishTerminal(report Report, outcome Outcome, valid bool) Report {
	report.Valid = valid
	report.Outcome = outcome
	return report
}

func finishViolation(report Report, finding Finding) Report {
	report.Findings = append(report.Findings, finding)
	report.Valid = false
	report.Outcome = OutcomeViolation
	return report
}

func finishIncomplete(report Report, record TraceRecord, index int, message string) Report {
	report.Findings = append(report.Findings, Finding{Invariant: "IncompleteTraceIsNotSuccess", Category: "incomplete", Message: message, TraceIndex: index, Line: record.Line, EventSequence: lastEventSequence(record.Trace)})
	report.Observations = append(report.Observations, observation("mark_incomplete", record, index, lastEventSequence(record.Trace)))
	report.Valid = false
	report.Outcome = OutcomeIncomplete
	return report
}

func finishIncompleteFinding(report Report, finding Finding) Report {
	report.Findings = append(report.Findings, finding)
	report.Observations = append(report.Observations, Observation{Action: "mark_incomplete", TraceIndex: finding.TraceIndex, Line: finding.Line, EventSequence: finding.EventSequence})
	report.Valid = false
	report.Outcome = OutcomeIncomplete
	return report
}

func finishInvalidInput(report Report, record TraceRecord, index int, message string) Report {
	report.Findings = append(report.Findings, Finding{Invariant: "StructuredTraceValid", Category: "input", Message: message, TraceIndex: index, Line: record.Line, EventSequence: firstEventSequence(record.Trace)})
	report.Valid = false
	report.Outcome = OutcomeInvalidInput
	return report
}

func finishInterrupted(report Report, record TraceRecord, index int) Report {
	report.Findings = append(report.Findings, Finding{Invariant: "InterruptedIsNotSuccess", Category: "terminal", Message: "trace ended with explicit interruption", TraceIndex: index, Line: record.Line, EventSequence: lastEventSequence(record.Trace)})
	report.Observations = append(report.Observations, observation("mark_interrupted", record, index, lastEventSequence(record.Trace)))
	report.Valid = false
	report.Outcome = OutcomeInterrupted
	return report
}

func finishTrailing(report Report, record TraceRecord, index int) Report {
	report.Findings = append(report.Findings, Finding{
		Invariant:     "TrailingTraceAfterTerminal",
		Category:      "ordering",
		Message:       "trace records remain after a terminal lifecycle outcome",
		TraceIndex:    index,
		Line:          record.Line,
		EventSequence: lastEventSequence(record.Trace),
	})
	report.Valid = false
	report.Outcome = OutcomeViolation
	return report
}
