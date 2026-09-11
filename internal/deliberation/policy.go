// Package deliberation loads and validates the repository's bounded
// multi-agent deliberation policy.
package deliberation

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	Path                 = "workflow/deliberation-policy.yml"
	CurrentSchemaVersion = 1
)

var expectedSignals = map[string]bool{
	"architectural-ambiguity":           true,
	"security-sensitivity":              true,
	"high-risk-migration":               true,
	"conflicting-hypotheses":            true,
	"independently-reviewable-evidence": true,
}

var expectedPatterns = map[string]bool{
	"independent-candidates": true,
	"fan-out-investigation":  true,
	"judge":                  true,
}

// Bounds limit every resource that a pattern may consume.
type Bounds struct {
	MaxAgents        int `yaml:"max_agents" json:"max_agents"`
	MaxRetries       int `yaml:"max_retries" json:"max_retries"`
	MaxElapsedMillis int `yaml:"max_elapsed_millis" json:"max_elapsed_millis"`
	MaxInputTokens   int `yaml:"max_input_tokens" json:"max_input_tokens"`
	MaxOutputTokens  int `yaml:"max_output_tokens" json:"max_output_tokens"`
	MaxCostMicros    int `yaml:"max_cost_micros" json:"max_cost_micros"`
}

// Defaults is the safe path used when no deliberation trigger is present.
type Defaults struct {
	DeterministicChecksFirst bool   `yaml:"deterministic_checks_first" json:"deterministic_checks_first"`
	Pattern                  string `yaml:"pattern" json:"pattern"`
	MaxAgents                int    `yaml:"max_agents" json:"max_agents"`
	MaxRetries               int    `yaml:"max_retries" json:"max_retries"`
	MaxElapsedMillis         int    `yaml:"max_elapsed_millis" json:"max_elapsed_millis"`
	MaxInputTokens           int    `yaml:"max_input_tokens" json:"max_input_tokens"`
	MaxOutputTokens          int    `yaml:"max_output_tokens" json:"max_output_tokens"`
	MaxCostMicros            int    `yaml:"max_cost_micros" json:"max_cost_micros"`
	Authority                string `yaml:"authority" json:"authority"`
	Concurrency              string `yaml:"concurrency" json:"concurrency"`
	Mutation                 string `yaml:"mutation" json:"mutation"`
}

// Signal describes one observable reason that independent reasoning may help.
type Signal struct {
	ID          string `yaml:"id" json:"id"`
	Description string `yaml:"description" json:"description"`
}

// Pattern describes one bounded deliberation pattern.
type Pattern struct {
	ID               string   `yaml:"id" json:"id"`
	Purpose          string   `yaml:"purpose" json:"purpose"`
	RequiredSignals  []string `yaml:"required_signals" json:"required_signals"`
	Independence     string   `yaml:"independence" json:"independence"`
	Authority        string   `yaml:"authority" json:"authority"`
	Concurrency      string   `yaml:"concurrency" json:"concurrency"`
	Mutation         string   `yaml:"mutation" json:"mutation"`
	Bounds           Bounds   `yaml:"bounds" json:"bounds"`
	CandidateContext string   `yaml:"candidate_context" json:"candidate_context"`
	Termination      string   `yaml:"termination" json:"termination"`
	Judge            string   `yaml:"judge" json:"judge"`
}

// Policy is the versioned machine-readable deliberation contract.
type Policy struct {
	SchemaVersion int       `yaml:"schema_version" json:"schema_version"`
	Defaults      Defaults  `yaml:"defaults" json:"defaults"`
	Signals       []Signal  `yaml:"signals" json:"signals"`
	Patterns      []Pattern `yaml:"patterns" json:"patterns"`
}

// Load reads the policy with strict YAML field checking.
func Load(root string) (*Policy, error) {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(Path)))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", Path, err)
	}
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	var policy Policy
	if err := decoder.Decode(&policy); err != nil {
		return nil, fmt.Errorf("decode %s: %w", Path, err)
	}
	return &policy, nil
}

// Validate returns deterministic findings for a policy document.
func Validate(policy *Policy) []string {
	findings := []string{}
	if policy == nil {
		return []string{"policy is missing"}
	}
	if policy.SchemaVersion != CurrentSchemaVersion {
		findings = append(findings, fmt.Sprintf("schema_version must be %d, got %d", CurrentSchemaVersion, policy.SchemaVersion))
	}
	if !policy.Defaults.DeterministicChecksFirst {
		findings = append(findings, "defaults must run deterministic checks first")
	}
	if policy.Defaults.Pattern != "single-agent" {
		findings = append(findings, "defaults.pattern must be single-agent")
	}
	findings = append(findings, validateBounds("defaults", Bounds{
		MaxAgents: policy.Defaults.MaxAgents, MaxRetries: policy.Defaults.MaxRetries,
		MaxElapsedMillis: policy.Defaults.MaxElapsedMillis, MaxInputTokens: policy.Defaults.MaxInputTokens,
		MaxOutputTokens: policy.Defaults.MaxOutputTokens, MaxCostMicros: policy.Defaults.MaxCostMicros,
	})...)
	if policy.Defaults.Authority != "read_only" || policy.Defaults.Concurrency != "serial" || policy.Defaults.Mutation != "parent_serialized" {
		findings = append(findings, "defaults must keep read-only serial parent mutation")
	}
	seenSignals := map[string]bool{}
	for index, signal := range policy.Signals {
		if signal.ID == "" || !expectedSignals[signal.ID] {
			findings = append(findings, fmt.Sprintf("signals[%d].id is unknown or empty", index))
		}
		if seenSignals[signal.ID] {
			findings = append(findings, fmt.Sprintf("signal %q is duplicated", signal.ID))
		}
		seenSignals[signal.ID] = true
		if strings.TrimSpace(signal.Description) == "" {
			findings = append(findings, fmt.Sprintf("signal %q has no description", signal.ID))
		}
	}
	for signal := range expectedSignals {
		if !seenSignals[signal] {
			findings = append(findings, fmt.Sprintf("required signal %q is missing", signal))
		}
	}
	seenPatterns := map[string]bool{}
	for index, pattern := range policy.Patterns {
		prefix := fmt.Sprintf("patterns[%d]", index)
		if pattern.ID == "" || !expectedPatterns[pattern.ID] {
			findings = append(findings, prefix+".id is unknown or empty")
		}
		if seenPatterns[pattern.ID] {
			findings = append(findings, fmt.Sprintf("pattern %q is duplicated", pattern.ID))
		}
		seenPatterns[pattern.ID] = true
		if strings.TrimSpace(pattern.Purpose) == "" {
			findings = append(findings, prefix+".purpose is empty")
		}
		if len(pattern.RequiredSignals) == 0 {
			findings = append(findings, prefix+".required_signals must not be empty")
		}
		for _, signal := range pattern.RequiredSignals {
			if !seenSignals[signal] {
				findings = append(findings, fmt.Sprintf("%s references unknown signal %q", prefix, signal))
			}
		}
		findings = append(findings, validatePattern(prefix, pattern)...)
	}
	for pattern := range expectedPatterns {
		if !seenPatterns[pattern] {
			findings = append(findings, fmt.Sprintf("required pattern %q is missing", pattern))
		}
	}
	sort.Strings(findings)
	return findings
}

func validatePattern(prefix string, pattern Pattern) []string {
	findings := validateBounds(prefix+".bounds", pattern.Bounds)
	if pattern.Authority != "read_only" {
		findings = append(findings, prefix+".authority must be read_only")
	}
	if pattern.Mutation != "parent_serialized" {
		findings = append(findings, prefix+".mutation must be parent_serialized")
	}
	if pattern.Independence == "" || pattern.Concurrency == "" || pattern.CandidateContext == "" || pattern.Termination == "" || pattern.Judge == "" {
		findings = append(findings, prefix+" must declare independence, concurrency, candidate_context, termination, and judge")
	}
	if pattern.Judge != "evidence_required_not_majority" {
		findings = append(findings, prefix+".judge must require evidence and reject majority-only selection")
	}
	switch pattern.ID {
	case "independent-candidates", "fan-out-investigation":
		if pattern.Bounds.MaxAgents < 2 {
			findings = append(findings, prefix+".bounds.max_agents must allow independent candidates")
		}
		if pattern.Independence != "isolated_context" || pattern.Concurrency != "parallel_read_only" {
			findings = append(findings, prefix+" must use isolated parallel read-only candidates")
		}
	case "judge":
		if pattern.Bounds.MaxAgents != 1 || pattern.Concurrency != "serial_after_candidates" {
			findings = append(findings, prefix+" must use one serial judge after candidates")
		}
	}
	return findings
}

func validateBounds(prefix string, bounds Bounds) []string {
	findings := []string{}
	if bounds.MaxAgents < 1 {
		findings = append(findings, prefix+".max_agents must be positive")
	}
	if bounds.MaxRetries < 0 {
		findings = append(findings, prefix+".max_retries must not be negative")
	}
	if bounds.MaxElapsedMillis < 1 {
		findings = append(findings, prefix+".max_elapsed_millis must be positive")
	}
	if bounds.MaxInputTokens < 1 || bounds.MaxOutputTokens < 1 || bounds.MaxCostMicros < 1 {
		findings = append(findings, prefix+" must declare positive token and cost bounds")
	}
	return findings
}

// Check validates the repository policy and writes a concise result.
func Check(root string, out, errOut io.Writer) int {
	policy, err := Load(root)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 1
	}
	findings := Validate(policy)
	if len(findings) > 0 {
		for _, finding := range findings {
			fmt.Fprintln(errOut, finding)
		}
		return 1
	}
	fmt.Fprintf(out, "deliberation policy valid: schema version %d, %d signals, %d patterns.\n", policy.SchemaVersion, len(policy.Signals), len(policy.Patterns))
	return 0
}
