package fsl

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

var mutatingHeaderRE = regexp.MustCompile(`(?m)^Mutating \S+ at depth \d+\r?$`)

// Survivor records one mutant that fslc could not kill, so a retained report
// and the triage register can point at a concrete mutant instead of a count.
type Survivor struct {
	Op     string `json:"op"`
	Target string `json:"target"`
	Line   int    `json:"line"`
	Column int    `json:"column"`
}

// SpecReport summarizes one spec's mutation run. Status is "mutated" on
// success or "error" when fslc itself failed (an infrastructure error rather
// than a surviving mutant).
type SpecReport struct {
	Spec      string     `json:"spec"`
	Status    string     `json:"status"`
	Total     int        `json:"total"`
	Killed    int        `json:"killed"`
	Survived  int        `json:"survived"`
	Invalid   int        `json:"invalid"`
	Survivors []Survivor `json:"survivors,omitempty"`
	Error     string     `json:"error,omitempty"`
}

// MutationReport is the retained, machine-readable output of a mutation run.
// It distinguishes killed mutants, surviving mutants, invalid (skipped)
// mutants, and infrastructure errors so a green aggregate never hides a
// survivor.
type MutationReport struct {
	Depth  string       `json:"depth"`
	Specs  []SpecReport `json:"specs"`
	Totals struct {
		Specs               int `json:"specs"`
		Killed              int `json:"killed"`
		Survived            int `json:"survived"`
		Invalid             int `json:"invalid"`
		InfrastructureError int `json:"infrastructure_errors"`
	} `json:"totals"`
}

// parseMutationDocument extracts the totals and survivor list from one parsed
// fslc mutation document.
func parseMutationDocument(doc map[string]any) (SpecReport, error) {
	report := SpecReport{Status: "mutated"}
	if summary, ok := doc["summary"].(map[string]any); ok {
		var err error
		if report.Total, err = intField(summary, "total"); err != nil {
			return report, err
		}
		if report.Killed, err = intField(summary, "killed"); err != nil {
			return report, err
		}
		if report.Survived, err = intField(summary, "survived"); err != nil {
			return report, err
		}
		if report.Invalid, err = intField(summary, "invalid"); err != nil {
			return report, err
		}
	}
	mutants, _ := doc["mutants"].([]any)
	for _, m := range mutants {
		mutant, ok := m.(map[string]any)
		if !ok {
			continue
		}
		status, _ := mutant["status"].(string)
		if status != "survived" {
			continue
		}
		op, _ := mutant["op"].(string)
		target, _ := mutant["target"].(string)
		survivor := Survivor{Op: op, Target: target}
		if loc, ok := mutant["loc"].(map[string]any); ok {
			survivor.Line, _ = intField(loc, "line")
			survivor.Column, _ = intField(loc, "column")
		}
		report.Survivors = append(report.Survivors, survivor)
	}
	return report, nil
}

func intField(m map[string]any, key string) (int, error) {
	switch v := m[key].(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	case json.Number:
		n, err := v.Int64()
		return int(n), err
	case nil:
		return 0, nil
	default:
		return 0, fmt.Errorf("expected an integer for %q, got %T", key, v)
	}
}

// parseMutationDocuments parses the fslc mutation JSON documents interleaved
// with the "Mutating <spec> at depth N" headers in text. Each header must be
// followed by exactly one complete JSON document; a header without a document
// is an error so truncated output fails loudly.
func parseMutationDocuments(text string) ([]SpecReport, error) {
	specs := []SpecReport{}
	scanner := bufio.NewScanner(strings.NewReader(text))
	var document bytes.Buffer
	inDocument := false
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		if !inDocument {
			if trimmed == "" || !strings.HasPrefix(trimmed, "{") {
				continue
			}
			document.Reset()
			inDocument = true
		}
		document.WriteString(line)
		document.WriteByte('\n')
		if !json.Valid(document.Bytes()) {
			continue
		}
		var doc map[string]any
		if err := json.Unmarshal(document.Bytes(), &doc); err != nil {
			return nil, err
		}
		report, err := parseMutationDocument(doc)
		if err != nil {
			return nil, err
		}
		specs = append(specs, report)
		document.Reset()
		inDocument = false
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	mutationRuns := len(mutatingHeaderRE.FindAllString(text, -1))
	if mutationRuns != len(specs) {
		return nil, fmt.Errorf(
			"mutation ran for %d spec(s) but produced %d document(s); output is truncated or a document is missing",
			mutationRuns, len(specs),
		)
	}
	return specs, nil
}

// WriteMutationReport marshals the report to path and returns an error if the
// write fails.
func WriteMutationReport(path string, report MutationReport) error {
	encoded, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(encoded, '\n'), 0o644)
}
