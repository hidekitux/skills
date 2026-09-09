// Package trace validates and persists redacted skill execution traces.
package trace

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/hidekitux/skills/internal/graph"
	"gopkg.in/yaml.v3"
)

const (
	// CurrentSchemaVersion is the version implemented by this package.
	CurrentSchemaVersion = 1
	// DefaultRetentionDays bounds the local retention period for persisted traces.
	DefaultRetentionDays = 30
)

const (
	KindRunStarted               = "run_started"
	KindTodoTransition           = "todo_transition"
	KindToolOutcome              = "tool_outcome"
	KindValidationOutcome        = "validation_outcome"
	KindEvidence                 = "evidence"
	KindHandoff                  = "handoff"
	KindRetry                    = "retry"
	KindRunFinished              = "run_finished"
	StatusStarted                = "started"
	StatusSuccess                = "success"
	StatusFailed                 = "failed"
	StatusSkipped                = "skipped"
	StatusInterrupted            = "interrupted"
	StatusError                  = "error"
	ClassificationNone           = "none"
	ClassificationDeterministic  = "deterministic_failure"
	ClassificationBehavioral     = "behavioral_failure"
	ClassificationInfrastructure = "infrastructure_error"
	ClassificationInterruption   = "user_interruption"
	ClassificationSkip           = "intentional_skip"
)

// Trace is one versioned record of one skill run.
type Trace struct {
	SchemaVersion      int              `json:"schema_version"`
	RunID              string           `json:"run_id"`
	ScenarioID         string           `json:"scenario_id,omitempty"`
	SkillID            string           `json:"skill_id"`
	SkillVersion       string           `json:"skill_version"`
	GraphVersion       int              `json:"graph_version"`
	Host               string           `json:"host"`
	HostVersion        string           `json:"host_version,omitempty"`
	Model              string           `json:"model"`
	ModelTier          string           `json:"model_tier,omitempty"`
	RepositoryRevision string           `json:"repository_revision"`
	StartedAt          string           `json:"started_at"`
	Usage              *Usage           `json:"usage,omitempty"`
	Events             []Event          `json:"events"`
	Terminal           Terminal         `json:"terminal"`
	Redaction          RedactionSummary `json:"redaction"`
}

// Event is one ordered semantic execution event.
type Event struct {
	Sequence   int             `json:"sequence"`
	Kind       string          `json:"kind"`
	Status     string          `json:"status"`
	At         string          `json:"at"`
	Todo       *TodoTransition `json:"todo,omitempty"`
	Tool       *ToolOutcome    `json:"tool,omitempty"`
	Validation *Validation     `json:"validation,omitempty"`
	Evidence   *Evidence       `json:"evidence,omitempty"`
	Handoff    *Handoff        `json:"handoff,omitempty"`
	Retry      *Retry          `json:"retry,omitempty"`
	Terminal   *Terminal       `json:"terminal,omitempty"`
}

// TodoTransition records one Todo List state change.
type TodoTransition struct {
	ItemID      string `json:"item_id"`
	From        string `json:"from"`
	To          string `json:"to"`
	EvidenceRef string `json:"evidence_ref,omitempty"`
}

// ToolOutcome records a tool result without arguments or output.
type ToolOutcome struct {
	Name           string `json:"name"`
	Result         string `json:"result"`
	Classification string `json:"classification,omitempty"`
	Retryable      bool   `json:"retryable,omitempty"`
}

// Validation records a validation result without command output.
type Validation struct {
	Name           string `json:"name"`
	Result         string `json:"result"`
	Classification string `json:"classification,omitempty"`
}

// Evidence is a safe pointer to an externally stored result.
type Evidence struct {
	Kind   string `json:"kind"`
	Ref    string `json:"ref"`
	Result string `json:"result,omitempty"`
}

// Handoff records the next skill and named artifact.
type Handoff struct {
	Destination string `json:"destination"`
	Artifact    string `json:"artifact"`
	Outcome     string `json:"outcome,omitempty"`
}

// Retry records one bounded retry attempt.
type Retry struct {
	Attempt     int    `json:"attempt"`
	MaxAttempts int    `json:"max_attempts"`
	Reason      string `json:"reason"`
}

// Terminal records the final run state.
type Terminal struct {
	Status         string `json:"status"`
	Classification string `json:"classification"`
	At             string `json:"at"`
}

// Usage records numeric context, cost, and elapsed-time data.
type Usage struct {
	Available     bool   `json:"available"`
	InputTokens   *int64 `json:"input_tokens,omitempty"`
	OutputTokens  *int64 `json:"output_tokens,omitempty"`
	ContextTokens *int64 `json:"context_tokens,omitempty"`
	CostMicros    *int64 `json:"cost_micros,omitempty"`
	ElapsedMillis int64  `json:"elapsed_millis"`
}

// RedactionSummary records the privacy policy without recording rejected data.
type RedactionSummary struct {
	Mode          string   `json:"mode"`
	RedactedCount int      `json:"redacted_count"`
	OmittedFields []string `json:"omitted_fields"`
	RetentionDays int      `json:"retention_days"`
}

// ValidationReport is the stable machine-readable result of trace validation.
type ValidationReport struct {
	Valid         bool     `json:"valid"`
	SchemaVersion int      `json:"schema_version"`
	RunID         string   `json:"run_id,omitempty"`
	EventCount    int      `json:"event_count"`
	Findings      []string `json:"findings,omitempty"`
}

// FileValidationReport is the stable result for a JSONL trace file.
type FileValidationReport struct {
	Valid      bool     `json:"valid"`
	TraceCount int      `json:"trace_count"`
	Findings   []string `json:"findings,omitempty"`
}

var (
	identifierPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9._/:-]*$`)
	runIDPattern      = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/:-]*$`)
	metadataPattern   = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/:@+-]*$`)
	versionPattern    = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)
	shaPattern        = regexp.MustCompile(`^[0-9a-f]{40}$`)
	credentialPattern = regexp.MustCompile(`(?i)(bearer\s+|password\s*=\s*|token\s*=\s*|secret\s*=\s*|api[_-]?key\s*=\s*)([^\s,;]+)|(?:gh[pousr]_[A-Za-z0-9_]+|github_pat_[A-Za-z0-9_]+|sk-[A-Za-z0-9_-]+|AKIA[0-9A-Z]{16})`)
	urlPattern        = regexp.MustCompile(`https?://[^\s"']+`)
)

// Validate checks a trace against the semantic contract.
func Validate(t Trace) ValidationReport {
	report := ValidationReport{Valid: true, SchemaVersion: t.SchemaVersion, RunID: t.RunID, EventCount: len(t.Events)}
	findings := []string{}
	if t.SchemaVersion != CurrentSchemaVersion {
		findings = append(findings, fmt.Sprintf("schema_version %d is unsupported", t.SchemaVersion))
	}
	if !validRunID(t.RunID) {
		findings = append(findings, "run_id is missing or invalid")
	}
	if t.ScenarioID != "" && !validIdentifier(t.ScenarioID) {
		findings = append(findings, "scenario_id is invalid")
	}
	if !validIdentifier(t.SkillID) {
		findings = append(findings, "skill_id is missing or invalid")
	}
	if !versionPattern.MatchString(t.SkillVersion) {
		findings = append(findings, "skill_version is missing or invalid")
	}
	if t.GraphVersion < 1 {
		findings = append(findings, "graph_version must be positive")
	}
	if !validIdentifier(t.Host) || !validShortString(t.Model) {
		findings = append(findings, "host or model is missing or invalid")
	}
	if t.HostVersion != "" && !validShortString(t.HostVersion) {
		findings = append(findings, "host_version is invalid")
	}
	if t.ModelTier != "" && !validIdentifier(t.ModelTier) {
		findings = append(findings, "model_tier is invalid")
	}
	if !shaPattern.MatchString(t.RepositoryRevision) {
		findings = append(findings, "repository_revision must be a lowercase 40-character commit SHA")
	}
	if !validTime(t.StartedAt) {
		findings = append(findings, "started_at must be an RFC3339 timestamp")
	}
	findings = append(findings, validateUsage(t.Usage)...)
	findings = append(findings, validateEvents(t.Events)...)
	findings = append(findings, validateTerminal(t.Terminal)...)
	if len(t.Events) > 0 {
		last := t.Events[len(t.Events)-1]
		if last.Kind != KindRunFinished || last.Terminal == nil {
			findings = append(findings, "last event must be run_finished with a terminal payload")
		} else if !sameTerminal(t.Terminal, *last.Terminal) {
			findings = append(findings, "envelope terminal does not match run_finished terminal")
		}
	}
	findings = append(findings, validateRedaction(t.Redaction)...)
	report.Findings = findings
	report.Valid = len(findings) == 0
	return report
}

func validateEvents(events []Event) []string {
	findings := []string{}
	if len(events) < 2 {
		findings = append(findings, "events must contain run_started and run_finished")
		return findings
	}
	if events[0].Kind != KindRunStarted {
		findings = append(findings, "first event must be run_started")
	}
	seenFinished := false
	previous := 0
	for index, event := range events {
		label := fmt.Sprintf("event %d", index)
		if event.Sequence <= previous {
			findings = append(findings, label+" sequence must increase strictly")
		}
		previous = event.Sequence
		if !validTime(event.At) {
			findings = append(findings, label+" at must be an RFC3339 timestamp")
		}
		if !validEventStatus(event.Status) {
			findings = append(findings, label+" status is invalid")
		}
		payloads := 0
		if event.Todo != nil {
			payloads++
		}
		if event.Tool != nil {
			payloads++
		}
		if event.Validation != nil {
			payloads++
		}
		if event.Evidence != nil {
			payloads++
		}
		if event.Handoff != nil {
			payloads++
		}
		if event.Retry != nil {
			payloads++
		}
		if event.Terminal != nil {
			payloads++
		}
		if event.Kind == KindRunStarted && payloads != 0 {
			findings = append(findings, label+" run_started cannot have a payload")
		}
		if event.Kind != KindRunStarted && payloads != 1 {
			findings = append(findings, label+" must have exactly one payload")
		}
		if !validEventKind(event.Kind) {
			findings = append(findings, label+" kind is invalid")
		}
		switch event.Kind {
		case KindTodoTransition:
			if event.Todo != nil {
				findings = append(findings, validateTodo(*event.Todo)...)
			}
		case KindToolOutcome:
			if event.Tool != nil {
				findings = append(findings, validateTool(*event.Tool)...)
			}
		case KindValidationOutcome:
			if event.Validation != nil {
				findings = append(findings, validateValidation(*event.Validation)...)
			}
		case KindEvidence:
			if event.Evidence != nil {
				findings = append(findings, validateEvidence(*event.Evidence)...)
			}
		case KindHandoff:
			if event.Handoff != nil {
				findings = append(findings, validateHandoff(*event.Handoff)...)
			}
		case KindRetry:
			if event.Retry != nil {
				findings = append(findings, validateRetry(*event.Retry)...)
			}
		case KindRunFinished:
			if event.Terminal != nil {
				seenFinished = true
				findings = append(findings, validateTerminal(*event.Terminal)...)
				if !terminalEventStatusMatches(event.Status, event.Terminal.Status) {
					findings = append(findings, label+" status does not match terminal status")
				}
			}
		}
		if seenFinished && index < len(events)-1 {
			findings = append(findings, "run_finished must be the final event")
			break
		}
	}
	if !seenFinished {
		findings = append(findings, "events must contain exactly one run_finished event")
	}
	return findings
}

func validateTodo(todo TodoTransition) []string {
	findings := []string{}
	for name, value := range map[string]string{"item_id": todo.ItemID, "from": todo.From, "to": todo.To} {
		if !validIdentifier(value) {
			findings = append(findings, "todo."+name+" is invalid")
		}
	}
	if todo.EvidenceRef != "" && !validIdentifier(todo.EvidenceRef) {
		findings = append(findings, "todo.evidence_ref is invalid")
	}
	return findings
}

func validateTool(tool ToolOutcome) []string {
	findings := []string{}
	if !validIdentifier(tool.Name) {
		findings = append(findings, "tool.name is invalid")
	}
	if !validIdentifier(tool.Result) {
		findings = append(findings, "tool.result is invalid")
	}
	if tool.Classification != "" && !validClassification(tool.Classification) {
		findings = append(findings, "tool.classification is invalid")
	}
	return findings
}

func validateValidation(validation Validation) []string {
	findings := []string{}
	if !validIdentifier(validation.Name) {
		findings = append(findings, "validation.name is invalid")
	}
	if !validIdentifier(validation.Result) {
		findings = append(findings, "validation.result is invalid")
	}
	if validation.Classification != "" && !validClassification(validation.Classification) {
		findings = append(findings, "validation.classification is invalid")
	}
	return findings
}

func validateEvidence(evidence Evidence) []string {
	findings := []string{}
	if !oneOf(evidence.Kind, "path", "command", "issue", "pull_request", "commit", "validation") {
		findings = append(findings, "evidence.kind is invalid")
	}
	if !validEvidenceRef(evidence.Kind, evidence.Ref) {
		findings = append(findings, "evidence.ref is unsafe or invalid")
	}
	if evidence.Result != "" && !validIdentifier(evidence.Result) {
		findings = append(findings, "evidence.result is invalid")
	}
	return findings
}

func validateHandoff(handoff Handoff) []string {
	findings := []string{}
	if !validIdentifier(handoff.Destination) {
		findings = append(findings, "handoff.destination is invalid")
	}
	if !validIdentifier(handoff.Artifact) {
		findings = append(findings, "handoff.artifact is invalid")
	}
	if handoff.Outcome != "" && !validIdentifier(handoff.Outcome) {
		findings = append(findings, "handoff.outcome is invalid")
	}
	return findings
}

func validateRetry(retry Retry) []string {
	findings := []string{}
	if retry.Attempt < 1 || retry.MaxAttempts < 1 || retry.Attempt > retry.MaxAttempts {
		findings = append(findings, "retry attempt must be within a positive max_attempts bound")
	}
	if !validIdentifier(retry.Reason) {
		findings = append(findings, "retry.reason is invalid")
	}
	return findings
}

func validateTerminal(terminal Terminal) []string {
	findings := []string{}
	if !oneOf(terminal.Status, "success", "failed", "skipped", "interrupted", "infrastructure_error") {
		findings = append(findings, "terminal.status is invalid")
	}
	if !validClassification(terminal.Classification) {
		findings = append(findings, "terminal.classification is invalid")
	}
	if !validTime(terminal.At) {
		findings = append(findings, "terminal.at must be an RFC3339 timestamp")
	}
	want := map[string]string{"success": ClassificationNone, "failed": ClassificationDeterministic, "skipped": ClassificationSkip, "interrupted": ClassificationInterruption, "infrastructure_error": ClassificationInfrastructure}
	if expected, ok := want[terminal.Status]; ok && terminal.Classification != expected && !(terminal.Status == "failed" && terminal.Classification == ClassificationBehavioral) {
		findings = append(findings, "terminal classification does not match terminal status")
	}
	return findings
}

func validateUsage(usage *Usage) []string {
	if usage == nil {
		return nil
	}
	findings := []string{}
	if usage.ElapsedMillis < 0 {
		findings = append(findings, "usage.elapsed_millis must not be negative")
	}
	for name, value := range map[string]*int64{"input_tokens": usage.InputTokens, "output_tokens": usage.OutputTokens, "context_tokens": usage.ContextTokens, "cost_micros": usage.CostMicros} {
		if value != nil && *value < 0 {
			findings = append(findings, "usage."+name+" must not be negative")
		}
	}
	return findings
}

func validateRedaction(redaction RedactionSummary) []string {
	findings := []string{}
	if redaction.Mode != "allowlist" {
		findings = append(findings, "redaction.mode must be allowlist")
	}
	if redaction.RedactedCount < 0 {
		findings = append(findings, "redaction.redacted_count must not be negative")
	}
	if redaction.RetentionDays < 1 || redaction.RetentionDays > DefaultRetentionDays {
		findings = append(findings, "redaction.retention_days must be between 1 and 30")
	}
	for _, field := range redaction.OmittedFields {
		if !validIdentifier(field) {
			findings = append(findings, "redaction.omitted_fields contains an invalid field")
		}
	}
	return findings
}

// Sanitize returns an allowlisted copy safe for persistence.
func Sanitize(input Trace) (Trace, error) {
	output := input
	output.SchemaVersion = CurrentSchemaVersion
	output.Redaction = RedactionSummary{Mode: "allowlist", OmittedFields: []string{}, RetentionDays: DefaultRetentionDays}
	redact := func(value string) string {
		clean, changed := redactString(value)
		if changed {
			output.Redaction.RedactedCount++
		}
		return clean
	}
	output.RunID = redact(input.RunID)
	output.ScenarioID = redact(input.ScenarioID)
	output.SkillID = redact(input.SkillID)
	output.SkillVersion = redact(input.SkillVersion)
	output.Host = redact(input.Host)
	output.HostVersion = redact(input.HostVersion)
	output.Model = redact(input.Model)
	output.ModelTier = redact(input.ModelTier)
	if output.HostVersion != "" && !validShortString(output.HostVersion) {
		output.HostVersion = ""
		output.Redaction.OmittedFields = append(output.Redaction.OmittedFields, "host_version")
	}
	if input.Model != "" && !validShortString(output.Model) {
		output.Model = "redacted-model"
		output.Redaction.RedactedCount++
	}
	if output.ModelTier != "" && !validIdentifier(output.ModelTier) {
		output.ModelTier = ""
		output.Redaction.OmittedFields = append(output.Redaction.OmittedFields, "model_tier")
	}
	output.RepositoryRevision = strings.ToLower(input.RepositoryRevision)
	output.StartedAt = input.StartedAt
	output.Events = make([]Event, 0, len(input.Events))
	for _, event := range input.Events {
		copyEvent := event
		if copyEvent.Todo != nil {
			copyTodo := *copyEvent.Todo
			copyTodo.ItemID, copyTodo.From, copyTodo.To = redact(copyTodo.ItemID), redact(copyTodo.From), redact(copyTodo.To)
			copyTodo.EvidenceRef = redact(copyTodo.EvidenceRef)
			copyEvent.Todo = &copyTodo
		}
		if copyEvent.Tool != nil {
			copyTool := *copyEvent.Tool
			copyTool.Name, copyTool.Result, copyTool.Classification = redact(copyTool.Name), redact(copyTool.Result), redact(copyTool.Classification)
			copyEvent.Tool = &copyTool
		}
		if copyEvent.Validation != nil {
			copyValidation := *copyEvent.Validation
			copyValidation.Name, copyValidation.Result, copyValidation.Classification = redact(copyValidation.Name), redact(copyValidation.Result), redact(copyValidation.Classification)
			copyEvent.Validation = &copyValidation
		}
		if copyEvent.Evidence != nil {
			copyEvidence := *copyEvent.Evidence
			copyEvidence.Result = redact(copyEvidence.Result)
			clean, changed := redactString(copyEvidence.Ref)
			if changed {
				output.Redaction.RedactedCount++
				output.Redaction.OmittedFields = append(output.Redaction.OmittedFields, "evidence.ref")
				continue
			}
			copyEvidence.Ref = clean
			copyEvent.Evidence = &copyEvidence
		}
		if copyEvent.Handoff != nil {
			copyHandoff := *copyEvent.Handoff
			copyHandoff.Destination, copyHandoff.Artifact, copyHandoff.Outcome = redact(copyHandoff.Destination), redact(copyHandoff.Artifact), redact(copyHandoff.Outcome)
			copyEvent.Handoff = &copyHandoff
		}
		if copyEvent.Retry != nil {
			copyRetry := *copyEvent.Retry
			copyRetry.Reason = redact(copyRetry.Reason)
			copyEvent.Retry = &copyRetry
		}
		if copyEvent.Terminal != nil {
			copyTerminal := *copyEvent.Terminal
			copyTerminal.Status = redact(copyTerminal.Status)
			copyTerminal.Classification = redact(copyTerminal.Classification)
			copyEvent.Terminal = &copyTerminal
		}
		output.Events = append(output.Events, copyEvent)
	}
	for index := range output.Events {
		output.Events[index].Sequence = index + 1
	}
	output.Terminal.Status = redact(input.Terminal.Status)
	output.Terminal.Classification = redact(input.Terminal.Classification)
	output.Terminal.At = input.Terminal.At
	output.Redaction.OmittedFields = uniqueSorted(output.Redaction.OmittedFields)
	if report := Validate(output); !report.Valid {
		return Trace{}, errors.New(strings.Join(report.Findings, "; "))
	}
	return output, nil
}

// WriteJSONL validates, sanitizes, and appends traces to an explicitly chosen path.
func WriteJSONL(path string, traces []Trace) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("trace output path is required for opt-in persistence")
	}
	if len(traces) == 0 {
		return errors.New("at least one trace is required")
	}
	safe := make([]Trace, 0, len(traces))
	for _, input := range traces {
		trace, err := Sanitize(input)
		if err != nil {
			return fmt.Errorf("sanitize trace %q: %w", input.RunID, err)
		}
		safe = append(safe, trace)
	}
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil {
			return err
		}
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	for _, trace := range safe {
		if err := encoder.Encode(trace); err != nil {
			return err
		}
	}
	return nil
}

// ReadJSONL reads and validates one or more persisted traces.
func ReadJSONL(path string) ([]Trace, error) {
	report := ValidateJSONL(path, "")
	if !report.Valid {
		return nil, errors.New(strings.Join(report.Findings, "; "))
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	decoder := json.NewDecoder(bufio.NewReader(f))
	traces := []Trace{}
	for line := 1; ; line++ {
		var trace Trace
		decoder.DisallowUnknownFields()
		err := decoder.Decode(&trace)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("trace line %d: %w", line, err)
		}
		traces = append(traces, trace)
	}
	if len(traces) == 0 {
		return nil, errors.New("trace file contains no records")
	}
	return traces, nil
}

// ValidateJSONL validates every trace line. When root is non-empty it also
// checks the skill and graph versions against repository metadata.
func ValidateJSONL(path, root string) FileValidationReport {
	report := FileValidationReport{Valid: true}
	f, err := os.Open(path)
	if err != nil {
		report.Valid = false
		report.Findings = []string{fmt.Sprintf("open trace file: %v", err)}
		return report
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	line := 0
	for scanner.Scan() {
		line++
		content := strings.TrimSpace(scanner.Text())
		if content == "" {
			continue
		}
		var item Trace
		if err := decodeStrict([]byte(content), &item); err != nil {
			report.Valid = false
			report.Findings = append(report.Findings, fmt.Sprintf("line %d: decode trace: %v", line, err))
			continue
		}
		report.TraceCount++
		if validation := Validate(item); !validation.Valid {
			report.Valid = false
			for _, finding := range validation.Findings {
				report.Findings = append(report.Findings, fmt.Sprintf("line %d: %s", line, finding))
			}
		}
		if root != "" {
			for _, finding := range validateRepositoryMetadata(root, item) {
				report.Valid = false
				report.Findings = append(report.Findings, fmt.Sprintf("line %d: %s", line, finding))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		report.Valid = false
		report.Findings = append(report.Findings, fmt.Sprintf("read trace file: %v", err))
	}
	if report.TraceCount == 0 {
		report.Valid = false
		report.Findings = append(report.Findings, "trace file contains no records")
	}
	return report
}

func validateRepositoryMetadata(root string, item Trace) []string {
	findings := []string{}
	graphVersion, err := GraphVersion(root)
	if err != nil {
		return []string{fmt.Sprintf("load graph: %v", err)}
	}
	if item.GraphVersion != graphVersion {
		findings = append(findings, fmt.Sprintf("graph_version %d does not match repository graph version %d", item.GraphVersion, graphVersion))
	}
	loaded, err := graph.Load(root)
	if err != nil {
		return append(findings, fmt.Sprintf("load graph: %v", err))
	}
	if _, ok := loaded.Skill(item.SkillID); !ok {
		findings = append(findings, fmt.Sprintf("skill_id %q is not in the repository graph", item.SkillID))
	}
	skillVersion, err := SkillVersion(root, item.SkillID)
	if err != nil {
		findings = append(findings, fmt.Sprintf("read skill version: %v", err))
	} else if item.SkillVersion != skillVersion {
		findings = append(findings, fmt.Sprintf("skill_version %q does not match catalog version %q", item.SkillVersion, skillVersion))
	}
	return findings
}

// CheckFixtures validates committed trace fixtures and their repository links.
func CheckFixtures(root string, out, errOut io.Writer) int {
	directory := filepath.Join(root, "workflow", "trace-fixtures")
	entries, err := os.ReadDir(directory)
	if os.IsNotExist(err) {
		fmt.Fprintln(out, "trace fixture check skipped: no workflow/trace-fixtures directory")
		return 0
	}
	if err != nil {
		fmt.Fprintf(errOut, "trace fixture check failed: %v\n", err)
		return 1
	}
	count := 0
	failed := false
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		count++
		path := filepath.Join(directory, entry.Name())
		fileReport := ValidateJSONL(path, root)
		if !fileReport.Valid {
			failed = true
			for _, finding := range fileReport.Findings {
				fmt.Fprintf(errOut, "%s: %s\n", entry.Name(), finding)
			}
			continue
		}
		fmt.Fprintf(out, "%s: %d trace(s) valid\n", entry.Name(), fileReport.TraceCount)
	}
	if count == 0 {
		fmt.Fprintln(out, "trace fixture check skipped: no JSONL fixtures found")
		return 0
	}
	if failed {
		return 1
	}
	fmt.Fprintf(out, "trace fixture check passed: %d file(s).\n", count)
	return 0
}

// SkillVersion reads one cataloged skill version without reading skill prose.
func SkillVersion(root, skillID string) (string, error) {
	content, err := os.ReadFile(filepath.Join(root, "CATALOG.yml"))
	if err != nil {
		return "", err
	}
	var catalog struct {
		Skills []struct {
			Name    string `yaml:"name"`
			Version string `yaml:"version"`
		} `yaml:"skills"`
	}
	if err := yaml.Unmarshal(content, &catalog); err != nil {
		return "", err
	}
	for _, skill := range catalog.Skills {
		if skill.Name == skillID {
			return skill.Version, nil
		}
	}
	return "", fmt.Errorf("skill %q is not cataloged", skillID)
}

// GraphVersion reads the authoritative skill graph version.
func GraphVersion(root string) (int, error) {
	loaded, err := graph.Load(root)
	if err != nil {
		return 0, err
	}
	return loaded.SchemaVersion, nil
}

// RepositoryRevision returns the SHA-256 hash of a repository revision string.
// It is used only by fixture adapters that have no Git checkout.
func RepositoryRevision(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func validIdentifier(value string) bool {
	return value != "" && len(value) <= 128 && identifierPattern.MatchString(value)
}
func validRunID(value string) bool {
	return value != "" && len(value) <= 128 && runIDPattern.MatchString(value)
}
func validShortString(value string) bool {
	return value != "" && len(value) <= 128 && metadataPattern.MatchString(value)
}
func validTime(value string) bool { _, err := time.Parse(time.RFC3339Nano, value); return err == nil }
func validClassification(value string) bool {
	return oneOf(value, ClassificationNone, ClassificationDeterministic, ClassificationBehavioral, ClassificationInfrastructure, ClassificationInterruption, ClassificationSkip)
}
func validEventStatus(value string) bool {
	return oneOf(value, StatusStarted, StatusSuccess, StatusFailed, StatusSkipped, StatusInterrupted, StatusError)
}
func validEventKind(value string) bool {
	return oneOf(value, KindRunStarted, KindTodoTransition, KindToolOutcome, KindValidationOutcome, KindEvidence, KindHandoff, KindRetry, KindRunFinished)
}
func oneOf(value string, choices ...string) bool {
	for _, choice := range choices {
		if value == choice {
			return true
		}
	}
	return false
}
func sameTerminal(left, right Terminal) bool {
	return left.Status == right.Status && left.Classification == right.Classification && left.At == right.At
}
func terminalEventStatusMatches(eventStatus, terminalStatus string) bool {
	if terminalStatus == "infrastructure_error" {
		return eventStatus == StatusError
	}
	return eventStatus == terminalStatus
}

func validEvidenceRef(kind, reference string) bool {
	if !validShortString(reference) || credentialPattern.MatchString(reference) {
		return false
	}
	switch kind {
	case "path":
		return !filepath.IsAbs(reference) && reference != "." && !strings.HasPrefix(filepath.ToSlash(reference), "../") && !strings.Contains(filepath.ToSlash(reference), "/../")
	case "command", "validation":
		return validIdentifier(reference)
	case "commit":
		return shaPattern.MatchString(reference)
	case "issue", "pull_request":
		parsed, err := url.Parse(reference)
		if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Host != "github.com" {
			return false
		}
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		want := "issues"
		if kind == "pull_request" {
			want = "pull"
		}
		return len(parts) == 4 && parts[0] != "" && parts[1] != "" && parts[2] == want && parts[3] != ""
	default:
		return false
	}
}

func redactString(value string) (string, bool) {
	if value == "" {
		return value, false
	}
	changed := false
	clean := credentialPattern.ReplaceAllStringFunc(value, func(match string) string { changed = true; return "[REDACTED]" })
	clean = urlPattern.ReplaceAllStringFunc(clean, func(match string) string {
		parsed, err := url.Parse(match)
		if err == nil && parsed.Scheme == "https" && parsed.Host == "github.com" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == "" {
			return match
		}
		changed = true
		return "[REDACTED_URL]"
	})
	if strings.Contains(clean, "-----BEGIN") {
		changed = true
		clean = regexp.MustCompile(`-----BEGIN [^-]+-----[\s\S]*?-----END [^-]+-----`).ReplaceAllString(clean, "[REDACTED_KEY]")
	}
	return clean, changed
}

func uniqueSorted(values []string) []string {
	set := map[string]bool{}
	for _, value := range values {
		if value != "" {
			set[value] = true
		}
	}
	result := make([]string, 0, len(set))
	for value := range set {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

// IsPrivateAddress is exposed for adapters that must reject local evidence URLs.
func IsPrivateAddress(host string) bool {
	if parsed := net.ParseIP(host); parsed != nil {
		return parsed.IsPrivate() || parsed.IsLoopback()
	}
	return host == "localhost" || strings.HasSuffix(host, ".local")
}
