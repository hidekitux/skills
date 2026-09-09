package trace

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// AdaptCodex converts the documented Codex event envelope into a semantic trace.
func AdaptCodex(data []byte) (Trace, error) {
	var input codexEnvelope
	if err := decodeStrict(data, &input); err != nil {
		return Trace{}, fmt.Errorf("decode Codex events: %w", err)
	}
	item := Trace{
		SchemaVersion: CurrentSchemaVersion, RunID: input.RunID, SkillID: input.Skill.ID,
		SkillVersion: input.Skill.Version, GraphVersion: input.GraphVersion,
		Host: input.Host.Name, HostVersion: input.Host.Version, Model: input.Model,
		RepositoryRevision: input.RepositoryRevision, StartedAt: input.StartedAt,
	}
	for _, event := range input.Events {
		converted, err := convertCodexEvent(event)
		if err != nil {
			return Trace{}, err
		}
		item.Events = append(item.Events, converted)
	}
	if len(item.Events) == 0 || item.Events[len(item.Events)-1].Terminal == nil {
		return Trace{}, fmt.Errorf("Codex events do not contain a terminal event")
	}
	item.Terminal = *item.Events[len(item.Events)-1].Terminal
	return Sanitize(item)
}

// AdaptClaudeCode converts the documented Claude Code event envelope into a semantic trace.
func AdaptClaudeCode(data []byte) (Trace, error) {
	var input claudeEnvelope
	if err := decodeStrict(data, &input); err != nil {
		return Trace{}, fmt.Errorf("decode Claude Code events: %w", err)
	}
	item := Trace{
		SchemaVersion: CurrentSchemaVersion, RunID: input.SessionID, SkillID: input.SkillName,
		SkillVersion: input.SkillVersion, GraphVersion: input.GraphSchema,
		Host: "claude-code", HostVersion: input.ProviderVersion, Model: input.Model,
		RepositoryRevision: input.RepositoryRevision, StartedAt: input.StartedAt,
	}
	for _, event := range input.Events {
		converted, err := convertClaudeEvent(event)
		if err != nil {
			return Trace{}, err
		}
		item.Events = append(item.Events, converted)
	}
	if len(item.Events) == 0 || item.Events[len(item.Events)-1].Terminal == nil {
		return Trace{}, fmt.Errorf("Claude Code events do not contain a terminal event")
	}
	item.Terminal = *item.Events[len(item.Events)-1].Terminal
	return Sanitize(item)
}

type codexEnvelope struct {
	RunID              string        `json:"run_id"`
	Skill              skillIdentity `json:"skill"`
	GraphVersion       int           `json:"graph_version"`
	Host               hostIdentity  `json:"host"`
	Model              string        `json:"model"`
	RepositoryRevision string        `json:"repository_revision"`
	StartedAt          string        `json:"started_at"`
	Events             []codexEvent  `json:"events"`
}

type skillIdentity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type hostIdentity struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type codexEvent struct {
	Sequence       int    `json:"sequence"`
	At             string `json:"at"`
	Type           string `json:"type"`
	Status         string `json:"status,omitempty"`
	ItemID         string `json:"item_id,omitempty"`
	From           string `json:"from,omitempty"`
	To             string `json:"to,omitempty"`
	EvidenceRef    string `json:"evidence_ref,omitempty"`
	Name           string `json:"name,omitempty"`
	Result         string `json:"result,omitempty"`
	Classification string `json:"classification,omitempty"`
	Retryable      bool   `json:"retryable,omitempty"`
	EvidenceKind   string `json:"evidence_kind,omitempty"`
	EvidenceResult string `json:"evidence_result,omitempty"`
	Destination    string `json:"destination,omitempty"`
	Artifact       string `json:"artifact,omitempty"`
	Outcome        string `json:"outcome,omitempty"`
	Attempt        int    `json:"attempt,omitempty"`
	MaxAttempts    int    `json:"max_attempts,omitempty"`
	Reason         string `json:"reason,omitempty"`
	TerminalStatus string `json:"terminal_status,omitempty"`
	TerminalClass  string `json:"terminal_classification,omitempty"`
}

type claudeEnvelope struct {
	SessionID          string        `json:"session_id"`
	SkillName          string        `json:"skill_name"`
	SkillVersion       string        `json:"skill_version"`
	GraphSchema        int           `json:"graph_schema"`
	ProviderVersion    string        `json:"provider_version"`
	Model              string        `json:"model"`
	RepositoryRevision string        `json:"repository_revision"`
	StartedAt          string        `json:"started_at"`
	Events             []claudeEvent `json:"events"`
}

type claudeEvent struct {
	Ordinal        int    `json:"ordinal"`
	Timestamp      string `json:"timestamp"`
	Event          string `json:"event"`
	State          string `json:"state,omitempty"`
	PreviousState  string `json:"previous_state,omitempty"`
	Item           string `json:"item,omitempty"`
	Evidence       string `json:"evidence,omitempty"`
	ToolName       string `json:"tool_name,omitempty"`
	OK             *bool  `json:"ok,omitempty"`
	FailureClass   string `json:"failure_class,omitempty"`
	Validator      string `json:"validator,omitempty"`
	Result         string `json:"result,omitempty"`
	EvidenceKind   string `json:"evidence_kind,omitempty"`
	Reference      string `json:"reference,omitempty"`
	NextSkill      string `json:"next_skill,omitempty"`
	Artifact       string `json:"artifact,omitempty"`
	Outcome        string `json:"outcome,omitempty"`
	Attempt        int    `json:"attempt,omitempty"`
	Limit          int    `json:"limit,omitempty"`
	Reason         string `json:"reason,omitempty"`
	TerminalStatus string `json:"terminal_status,omitempty"`
	Classification string `json:"classification,omitempty"`
}

func convertCodexEvent(input codexEvent) (Event, error) {
	event := Event{Sequence: input.Sequence, At: input.At}
	switch input.Type {
	case "session_start":
		event.Kind, event.Status = KindRunStarted, StatusStarted
	case "todo":
		event.Kind, event.Status = KindTodoTransition, StatusSuccess
		event.Todo = &TodoTransition{ItemID: input.ItemID, From: input.From, To: input.To, EvidenceRef: input.EvidenceRef}
	case "tool":
		event.Kind, event.Status = KindToolOutcome, outcomeStatus(input.Status)
		event.Tool = &ToolOutcome{Name: input.Name, Result: input.Result, Classification: input.Classification, Retryable: input.Retryable}
	case "validation":
		event.Kind, event.Status = KindValidationOutcome, outcomeStatus(input.Status)
		event.Validation = &Validation{Name: input.Name, Result: input.Result, Classification: input.Classification}
	case "evidence":
		event.Kind, event.Status = KindEvidence, StatusSuccess
		event.Evidence = &Evidence{Kind: input.EvidenceKind, Ref: input.Name, Result: input.EvidenceResult}
	case "handoff":
		event.Kind, event.Status = KindHandoff, StatusSuccess
		event.Handoff = &Handoff{Destination: input.Destination, Artifact: input.Artifact, Outcome: input.Outcome}
	case "retry":
		event.Kind, event.Status = KindRetry, StatusFailed
		event.Retry = &Retry{Attempt: input.Attempt, MaxAttempts: input.MaxAttempts, Reason: input.Reason}
	case "session_end":
		event.Kind = KindRunFinished
		event.Terminal = &Terminal{Status: input.TerminalStatus, Classification: input.TerminalClass, At: input.At}
		event.Status = terminalEventStatus(input.TerminalStatus)
	default:
		return Event{}, fmt.Errorf("unsupported Codex event type %q", input.Type)
	}
	return event, nil
}

func convertClaudeEvent(input claudeEvent) (Event, error) {
	event := Event{Sequence: input.Ordinal, At: input.Timestamp}
	switch input.Event {
	case "session_start":
		event.Kind, event.Status = KindRunStarted, StatusStarted
	case "todo_updated":
		event.Kind, event.Status = KindTodoTransition, StatusSuccess
		event.Todo = &TodoTransition{ItemID: input.Item, From: input.PreviousState, To: input.State, EvidenceRef: input.Evidence}
	case "tool_result":
		event.Kind, event.Status = KindToolOutcome, boolStatus(input.OK)
		event.Tool = &ToolOutcome{Name: input.ToolName, Result: input.Result, Classification: input.FailureClass}
	case "validation_result":
		event.Kind, event.Status = KindValidationOutcome, resultStatus(input.Result)
		event.Validation = &Validation{Name: input.Validator, Result: input.Result, Classification: input.FailureClass}
	case "evidence":
		event.Kind, event.Status = KindEvidence, StatusSuccess
		event.Evidence = &Evidence{Kind: input.EvidenceKind, Ref: input.Reference, Result: input.Result}
	case "handoff_emitted":
		event.Kind, event.Status = KindHandoff, StatusSuccess
		event.Handoff = &Handoff{Destination: input.NextSkill, Artifact: input.Artifact, Outcome: input.Outcome}
	case "retry":
		event.Kind, event.Status = KindRetry, StatusFailed
		event.Retry = &Retry{Attempt: input.Attempt, MaxAttempts: input.Limit, Reason: input.Reason}
	case "session_end":
		event.Kind = KindRunFinished
		event.Terminal = &Terminal{Status: input.TerminalStatus, Classification: input.Classification, At: input.Timestamp}
		event.Status = terminalEventStatus(input.TerminalStatus)
	default:
		return Event{}, fmt.Errorf("unsupported Claude Code event type %q", input.Event)
	}
	return event, nil
}

func decodeStrict(data []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func outcomeStatus(value string) string {
	switch value {
	case StatusSuccess:
		return StatusSuccess
	case StatusSkipped:
		return StatusSkipped
	case StatusInterrupted:
		return StatusInterrupted
	case StatusFailed:
		return StatusFailed
	default:
		return StatusError
	}
}

func boolStatus(value *bool) string {
	if value != nil && *value {
		return StatusSuccess
	}
	return StatusFailed
}

func resultStatus(value string) string {
	if value == StatusSuccess || value == "passed" {
		return StatusSuccess
	}
	if value == StatusSkipped {
		return StatusSkipped
	}
	return StatusFailed
}

func terminalEventStatus(value string) string {
	if value == "infrastructure_error" {
		return StatusError
	}
	return value
}
