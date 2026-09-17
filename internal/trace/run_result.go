package trace

import (
	"strings"
	"time"

	skillcontext "github.com/hidekitux/skills/internal/context"
	executionstrategy "github.com/hidekitux/skills/internal/strategy"
)

// RunOutcome is the typed outcome an execution module reports for one observed
// run. The evidence module maps it to the persisted terminal status and
// classification; an execution module never writes those strings itself.
type RunOutcome string

const (
	RunPassed              RunOutcome = "pass"
	RunFailed              RunOutcome = "fail"
	RunSkipped             RunOutcome = "skipped"
	RunInterrupted         RunOutcome = "interrupted"
	RunInfrastructureError RunOutcome = "infrastructure_error"
)

// Evaluation names the evidence produces for a run an evaluation observed.
const (
	evaluationToolName       = "host-stage"
	evaluationValidationName = "check-evaluation"
	evaluationProducer       = "evaluate"
	evaluationTodoItem       = "evaluation"
	evaluationArtifact       = "evaluation-result"
	evaluationRetryReason    = "user-correction"
)

// RunResult is the typed domain result for one observed run. It carries only
// what an execution module observes. The schema version, the event order, the
// terminal classification, and the redaction summary are decided here, so an
// execution module never assembles a persisted trace.
type RunResult struct {
	RunID              string
	ScenarioID         string
	SkillID            string
	SkillVersion       string
	GraphVersion       int
	Host               string
	Model              string
	RepositoryRevision string
	StartedAt          string
	FinishedAt         string
	ElapsedMillis      int64
	Context            *skillcontext.Manifest
	Deliberation       *Deliberation
	Strategy           *executionstrategy.Decision
	Outcome            RunOutcome
	CorrectionAttempts int
	HandoffDestination string
}

// FromEvaluationRun builds the persisted trace for one evaluated run.
func FromEvaluationRun(result RunResult) Trace {
	started := result.StartedAt
	if started == "" {
		started = time.Unix(0, 0).UTC().Format(time.RFC3339Nano)
	}
	finished := result.FinishedAt
	if finished == "" {
		finished = started
	}
	terminalStatus, classification := result.Outcome.terminal()
	terminal := Terminal{Status: terminalStatus, Classification: classification, At: finished}
	item := Trace{
		SchemaVersion:      CurrentSchemaVersion,
		RunID:              result.RunID,
		ScenarioID:         result.ScenarioID,
		SkillID:            result.SkillID,
		SkillVersion:       result.SkillVersion,
		GraphVersion:       result.GraphVersion,
		Host:               result.Host,
		Model:              result.Model,
		RepositoryRevision: normalizedRevision(result.RepositoryRevision),
		StartedAt:          started,
		Context:            result.Context,
		Deliberation:       result.Deliberation,
		Strategy:           result.Strategy,
		Usage:              &Usage{Available: false, ElapsedMillis: result.ElapsedMillis},
		Terminal:           terminal,
		Redaction:          RedactionSummary{Mode: "allowlist", OmittedFields: []string{}, RetentionDays: DefaultRetentionDays},
	}
	item.Events = append(item.Events,
		Event{Sequence: 1, Kind: KindRunStarted, Status: StatusStarted, At: started},
		Event{Sequence: 2, Kind: KindTodoTransition, Status: StatusSuccess, At: started, Todo: &TodoTransition{ItemID: evaluationTodoItem, From: "pending", To: "in_progress"}},
	)
	eventStatus, eventClass := result.Outcome.event()
	identifier := result.Outcome.identifier()
	item.Events = append(item.Events, Event{Sequence: len(item.Events) + 1, Kind: KindToolOutcome, Status: eventStatus, At: finished, Tool: &ToolOutcome{Name: evaluationToolName, Result: eventStatus, Classification: eventClass}})
	item.Events = append(item.Events, Event{Sequence: len(item.Events) + 1, Kind: KindValidationOutcome, Status: eventStatus, At: finished, Validation: &Validation{
		Name: evaluationValidationName, Result: identifier, Classification: eventClass,
		Diagnostics: []DiagnosticRef{{Producer: evaluationProducer, Code: evaluationProducer + ".scenario." + identifier}},
	}})
	item.Events = append(item.Events, Event{Sequence: len(item.Events) + 1, Kind: KindEvidence, Status: eventStatus, At: finished, Evidence: &Evidence{Kind: "validation", Ref: evaluationValidationName, Result: identifier}})
	if result.CorrectionAttempts > 0 {
		item.Events = append(item.Events, Event{Sequence: len(item.Events) + 1, Kind: KindRetry, Status: StatusFailed, At: finished, Retry: &Retry{Attempt: result.CorrectionAttempts, MaxAttempts: result.CorrectionAttempts, Reason: evaluationRetryReason}})
	}
	if result.HandoffDestination != "" {
		item.Events = append(item.Events, Event{Sequence: len(item.Events) + 1, Kind: KindHandoff, Status: StatusSuccess, At: finished, Handoff: &Handoff{Destination: result.HandoffDestination, Artifact: evaluationArtifact, Outcome: StatusSuccess}})
	}
	if result.Outcome == RunPassed {
		item.Events = append(item.Events, Event{Sequence: len(item.Events) + 1, Kind: KindTodoTransition, Status: StatusSuccess, At: finished, Todo: &TodoTransition{ItemID: evaluationTodoItem, From: "in_progress", To: "completed", EvidenceRef: evaluationValidationName}})
	}
	item.Events = append(item.Events, Event{Sequence: len(item.Events) + 1, Kind: KindRunFinished, Status: terminalEventStatus(terminalStatus), At: finished, Terminal: &terminal})
	return item
}

// terminal maps the outcome to the persisted terminal status and
// classification.
func (o RunOutcome) terminal() (string, string) {
	switch o {
	case RunPassed:
		return "success", ClassificationNone
	case RunFailed:
		return "failed", ClassificationDeterministic
	case RunSkipped:
		return "skipped", ClassificationSkip
	case RunInterrupted:
		return "interrupted", ClassificationInterruption
	default:
		return "infrastructure_error", ClassificationInfrastructure
	}
}

// event maps the outcome to the status and classification of the tool and
// validation events. It differs from terminal: a passing run reports the
// success status rather than the success terminal status.
func (o RunOutcome) event() (string, string) {
	switch o {
	case RunPassed:
		return StatusSuccess, ClassificationNone
	case RunFailed:
		return StatusFailed, ClassificationDeterministic
	case RunSkipped:
		return StatusSkipped, ClassificationSkip
	case RunInterrupted:
		return StatusInterrupted, ClassificationInterruption
	default:
		return StatusError, ClassificationInfrastructure
	}
}

// identifier renders the outcome as a safe identifier for a validation result
// and a diagnostic code.
func (o RunOutcome) identifier() string {
	return strings.ReplaceAll(string(o), "_", "-")
}

func normalizedRevision(value string) string {
	if len(value) == 40 {
		return value
	}
	return RepositoryRevision(value)
}
