package eval

import (
	"fmt"
	"strings"
	"time"

	"github.com/hidekitux/skills/internal/trace"
)

// traceForRecord converts one structured evaluation result into the shared
// semantic trace without persisting the host transcript.
func traceForRecord(sc *Scenario, record Record, graphVersion int, skillVersion string) (trace.Trace, error) {
	started := record.StartedAt
	if started == "" {
		started = time.Unix(0, 0).UTC().Format(time.RFC3339Nano)
	}
	finished := record.FinishedAt
	if finished == "" {
		finished = started
	}
	terminalStatus, classification := traceTerminal(record)
	terminal := trace.Terminal{Status: terminalStatus, Classification: classification, At: finished}
	item := trace.Trace{
		SchemaVersion:      trace.CurrentSchemaVersion,
		RunID:              record.RunID,
		ScenarioID:         record.Scenario,
		SkillID:            traceSkill(sc, record),
		SkillVersion:       skillVersion,
		GraphVersion:       graphVersion,
		Host:               record.Host,
		Model:              record.Model,
		RepositoryRevision: normalizedRevision(record.Commit),
		StartedAt:          started,
		Usage:              &trace.Usage{Available: false, ElapsedMillis: record.ElapsedMillis},
		Terminal:           terminal,
		Redaction:          trace.RedactionSummary{Mode: "allowlist", OmittedFields: []string{}, RetentionDays: trace.DefaultRetentionDays},
	}
	item.Events = append(item.Events,
		trace.Event{Sequence: 1, Kind: trace.KindRunStarted, Status: trace.StatusStarted, At: started},
		trace.Event{Sequence: 2, Kind: trace.KindTodoTransition, Status: trace.StatusSuccess, At: started, Todo: &trace.TodoTransition{ItemID: "evaluation", From: "pending", To: "in_progress"}},
	)
	toolStatus, toolClass := traceOutcome(record)
	item.Events = append(item.Events, trace.Event{Sequence: len(item.Events) + 1, Kind: trace.KindToolOutcome, Status: toolStatus, At: finished, Tool: &trace.ToolOutcome{Name: "host-stage", Result: toolStatus, Classification: toolClass}})
	validationStatus, validationClass := traceOutcome(record)
	item.Events = append(item.Events, trace.Event{Sequence: len(item.Events) + 1, Kind: trace.KindValidationOutcome, Status: validationStatus, At: finished, Validation: &trace.Validation{
		Name: "check-evaluation", Result: traceIdentifier(record.Verdict), Classification: validationClass,
		Diagnostics: []trace.DiagnosticRef{{Producer: "evaluate", Code: "evaluate.scenario." + traceIdentifier(record.Verdict)}},
	}})
	item.Events = append(item.Events, trace.Event{Sequence: len(item.Events) + 1, Kind: trace.KindEvidence, Status: validationStatus, At: finished, Evidence: &trace.Evidence{Kind: "validation", Ref: "check-evaluation", Result: traceIdentifier(record.Verdict)}})
	if record.CorrectionsUsed > 0 {
		item.Events = append(item.Events, trace.Event{Sequence: len(item.Events) + 1, Kind: trace.KindRetry, Status: trace.StatusFailed, At: finished, Retry: &trace.Retry{Attempt: record.CorrectionsUsed, MaxAttempts: record.CorrectionsUsed, Reason: "user-correction"}})
	}
	if record.HandoffObserved && sc.Expectations.Handoff != "" {
		item.Events = append(item.Events, trace.Event{Sequence: len(item.Events) + 1, Kind: trace.KindHandoff, Status: trace.StatusSuccess, At: finished, Handoff: &trace.Handoff{Destination: sc.Expectations.Handoff, Artifact: "evaluation-result", Outcome: trace.StatusSuccess}})
	}
	if record.Verdict == VerdictPass {
		item.Events = append(item.Events, trace.Event{Sequence: len(item.Events) + 1, Kind: trace.KindTodoTransition, Status: trace.StatusSuccess, At: finished, Todo: &trace.TodoTransition{ItemID: "evaluation", From: "in_progress", To: "completed", EvidenceRef: "check-evaluation"}})
	}
	item.Events = append(item.Events, trace.Event{Sequence: len(item.Events) + 1, Kind: trace.KindRunFinished, Status: terminalEventStatus(terminalStatus), At: finished, Terminal: &terminal})
	return item, nil
}

func traceTerminal(record Record) (string, string) {
	switch record.Verdict {
	case VerdictPass:
		return "success", trace.ClassificationNone
	case VerdictFail:
		return "failed", trace.ClassificationDeterministic
	case VerdictSkipped:
		return "skipped", trace.ClassificationSkip
	case VerdictInterrupted:
		return "interrupted", trace.ClassificationInterruption
	default:
		return "infrastructure_error", trace.ClassificationInfrastructure
	}
}

func traceOutcome(record Record) (string, string) {
	switch record.Verdict {
	case VerdictPass:
		return trace.StatusSuccess, trace.ClassificationNone
	case VerdictFail:
		return trace.StatusFailed, trace.ClassificationDeterministic
	case VerdictSkipped:
		return trace.StatusSkipped, trace.ClassificationSkip
	case VerdictInterrupted:
		return trace.StatusInterrupted, trace.ClassificationInterruption
	default:
		return trace.StatusError, trace.ClassificationInfrastructure
	}
}

func terminalEventStatus(status string) string {
	if status == "infrastructure_error" {
		return trace.StatusError
	}
	return status
}

func normalizedRevision(value string) string {
	if len(value) == 40 {
		return value
	}
	return trace.RepositoryRevision(value)
}

func traceIdentifier(value string) string {
	return strings.ReplaceAll(value, "_", "-")
}

func traceSkill(sc *Scenario, record Record) string {
	if record.Skill != E2ESkill || len(sc.Stages) == 0 {
		return record.Skill
	}
	return sc.Stages[0].Skill
}

func traceSkillVersion(root string, sc *Scenario, record Record) (string, error) {
	version, err := trace.SkillVersion(root, traceSkill(sc, record))
	if err != nil {
		return "", fmt.Errorf("skill version for %q: %w", traceSkill(sc, record), err)
	}
	return version, nil
}
