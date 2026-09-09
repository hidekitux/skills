// Package replay validates structured skill traces against the cross-skill
// workflow contract and prepares deterministic observations for FSL replay.
package replay

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/hidekitux/skills/internal/trace"
)

// Outcome is the stable top-level result of a replay attempt.
type Outcome string

const (
	OutcomeValid          Outcome = "valid"
	OutcomeViolation      Outcome = "violation"
	OutcomeIncomplete     Outcome = "incomplete_evidence"
	OutcomeInterrupted    Outcome = "interrupted"
	OutcomeRetryExhausted Outcome = "retry_exhausted"
	OutcomeBlocked        Outcome = "blocked"
	OutcomeFailed         Outcome = "failed"
	OutcomeSkipped        Outcome = "skipped"
	OutcomeInvalidInput   Outcome = "invalid_input"
)

// Finding identifies one replay problem without copying raw trace content.
type Finding struct {
	Invariant     string `json:"invariant"`
	Category      string `json:"category"`
	Message       string `json:"message"`
	TraceIndex    int    `json:"trace_index"`
	Line          int    `json:"line"`
	EventSequence int    `json:"event_sequence"`
}

// Observation is the public FSL replay action and its source boundary. The
// source fields are safe numeric and identifier values; raw event payloads do
// not enter the observation.
type Observation struct {
	Action        string `json:"action"`
	SkillID       string `json:"skill_id,omitempty"`
	TraceIndex    int    `json:"trace_index"`
	Line          int    `json:"line"`
	EventSequence int    `json:"event_sequence"`
}

// Report is the stable replay result. FSL model checking and observed-run
// conformance are separate claims, so the report records observations and
// findings without presenting either as implementation-quality evidence.
type Report struct {
	Valid        bool          `json:"valid"`
	Outcome      Outcome       `json:"outcome"`
	Spec         string        `json:"spec"`
	StepsChecked int           `json:"steps_checked"`
	Observations []Observation `json:"observations,omitempty"`
	Findings     []Finding     `json:"findings,omitempty"`
}

// PublicTrace is the minimal host-neutral trace accepted by fslc replay. It
// carries only normalized actions; source locations remain in Report and raw
// trace payloads never cross this boundary.
type PublicTrace struct {
	Events []PublicEvent `json:"events"`
}

// PublicEvent is one normalized FSL action.
type PublicEvent struct {
	Action string `json:"action"`
}

// PublicTraceForReport converts a report into the stable fslc replay input.
func PublicTraceForReport(report Report) (PublicTrace, error) {
	if len(report.Observations) == 0 {
		return PublicTrace{}, errors.New("replay report contains no FSL observations")
	}
	result := PublicTrace{Events: make([]PublicEvent, 0, len(report.Observations))}
	for _, observation := range report.Observations {
		if observation.Action == "" {
			return PublicTrace{}, errors.New("replay report contains an empty FSL action")
		}
		result.Events = append(result.Events, PublicEvent{Action: observation.Action})
	}
	return result, nil
}

// TraceRecord retains a trace's input line and semantic validation result.
// A record with an invalid terminal or missing required terminal event remains
// available so Replay can return incomplete_evidence instead of false success.
type TraceRecord struct {
	Trace      trace.Trace
	Line       int
	Validation trace.ValidationReport
}

// TraceSet is the ordered JSONL input to one cross-skill replay.
type TraceSet struct {
	Records []TraceRecord
}

// InputError identifies malformed JSONL without exposing the malformed value.
type InputError struct {
	Line    int
	Message string
}

func (e *InputError) Error() string {
	if e.Line <= 0 {
		return "replay input is invalid: " + e.Message
	}
	return fmt.Sprintf("replay input is invalid at line %d: %s", e.Line, e.Message)
}

// ReadJSONL decodes one or more strict structured traces. Schema or terminal
// findings are retained for Replay to classify; malformed JSON and unknown
// fields are input errors because no safe semantic observation can be made.
func ReadJSONL(path string) (TraceSet, error) {
	if path == "" {
		return TraceSet{}, errors.New("replay input path is required")
	}
	f, err := os.Open(path)
	if err != nil {
		return TraceSet{}, err
	}
	defer f.Close()

	set := TraceSet{}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		data := scanner.Bytes()
		if len(data) == 0 {
			continue
		}
		var item trace.Trace
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&item); err != nil {
			return TraceSet{}, &InputError{Line: line, Message: "cannot decode trace JSON"}
		}
		var trailing json.RawMessage
		if err := decoder.Decode(&trailing); err != io.EOF {
			return TraceSet{}, &InputError{Line: line, Message: "trace line contains trailing JSON"}
		}
		set.Records = append(set.Records, TraceRecord{
			Trace:      item,
			Line:       line,
			Validation: trace.Validate(item),
		})
	}
	if err := scanner.Err(); err != nil {
		return TraceSet{}, err
	}
	if len(set.Records) == 0 {
		return TraceSet{}, errors.New("replay input contains no trace records")
	}
	return set, nil
}

// NewReport returns a deterministic empty report for the given specification.
func NewReport(spec string) Report {
	return Report{
		Valid:   false,
		Outcome: OutcomeInvalidInput,
		Spec:    spec,
	}
}
