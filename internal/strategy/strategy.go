// Package strategy selects a bounded execution strategy from observable task
// signals and existing repository capability contracts.
package strategy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hidekitux/skills/internal/environment"
	"github.com/hidekitux/skills/internal/graph"
	"gopkg.in/yaml.v3"
)

const (
	Path                 = "workflow/execution-strategy-policy.yml"
	CurrentSchemaVersion = 1
)

type Level string

const (
	Low        Level = "low"
	Medium     Level = "medium"
	High       Level = "high"
	Unknown    Level = "unknown"
	None       Level = "none"
	Repository Level = "repository"
	External   Level = "external"
	Complete   Level = "complete"
	Partial    Level = "partial"
	Missing    Level = "missing"
)

// SignalDefinition declares the finite values accepted for one task signal.
type SignalDefinition struct {
	ID          string   `yaml:"id" json:"id"`
	Levels      []string `yaml:"levels" json:"levels"`
	Description string   `yaml:"description" json:"description"`
}

// Defaults bounds the safe fallback route.
type Defaults struct {
	Strategy                 string `yaml:"strategy" json:"strategy"`
	DeterministicChecksFirst bool   `yaml:"deterministic_checks_first" json:"deterministic_checks_first"`
	MaxElapsedMillis         int    `yaml:"max_elapsed_millis" json:"max_elapsed_millis"`
	MaxInputTokens           int    `yaml:"max_input_tokens" json:"max_input_tokens"`
	MaxOutputTokens          int    `yaml:"max_output_tokens" json:"max_output_tokens"`
	MaxCostMicros            int    `yaml:"max_cost_micros" json:"max_cost_micros"`
}

// Fallback records the named behavior for unavailable evidence or capacity.
type Fallback struct {
	ModelUnavailable   string `yaml:"model_unavailable" json:"model_unavailable"`
	IncompleteEvidence string `yaml:"incomplete_evidence" json:"incomplete_evidence"`
	FailedValidation   string `yaml:"failed_validation" json:"failed_validation"`
	ExternalMutation   string `yaml:"external_mutation" json:"external_mutation"`
}

// Strategy is a reusable execution profile. It references existing contracts
// and does not contain provider or model names.
type Strategy struct {
	ID               string `yaml:"id" json:"id"`
	ModelTier        string `yaml:"model_tier" json:"model_tier"`
	ContextProfile   string `yaml:"context_profile" json:"context_profile"`
	ValidationTier   string `yaml:"validation_tier" json:"validation_tier"`
	Parallelism      string `yaml:"parallelism" json:"parallelism"`
	MaxRetries       int    `yaml:"max_retries" json:"max_retries"`
	MaxElapsedMillis int    `yaml:"max_elapsed_millis" json:"max_elapsed_millis"`
	Escalation       string `yaml:"escalation" json:"escalation"`
	AuthorityCeiling string `yaml:"authority_ceiling" json:"authority_ceiling"`
}

// Rule is an ordered conjunction of signal predicates.
type Rule struct {
	ID       string              `yaml:"id" json:"id"`
	When     map[string][]string `yaml:"when" json:"when"`
	Strategy string              `yaml:"strategy" json:"strategy"`
	Reason   string              `yaml:"reason" json:"reason"`
}

// Policy is the versioned execution-strategy contract.
type Policy struct {
	SchemaVersion int                `yaml:"schema_version" json:"schema_version"`
	Signals       []SignalDefinition `yaml:"signals" json:"signals"`
	Defaults      Defaults           `yaml:"defaults" json:"defaults"`
	Fallback      Fallback           `yaml:"fallback" json:"fallback"`
	Strategies    []Strategy         `yaml:"strategies" json:"strategies"`
	Rules         []Rule             `yaml:"rules" json:"rules"`
}

// Evidence is a safe reference used to justify a decision or reclassification.
type Evidence struct {
	Kind   string `json:"kind" yaml:"kind"`
	Ref    string `json:"ref" yaml:"ref"`
	Result string `json:"result,omitempty" yaml:"result,omitempty"`
}

// Overrides records explicit choices or safe fallback actions.
type Overrides struct {
	ModelTier      string   `json:"model_tier,omitempty"`
	ValidationTier string   `json:"validation_tier,omitempty"`
	Values         []string `json:"values,omitempty"`
}

// Input is the privacy-safe selector input. It contains no prompt, source,
// command output, credential, or user content.
type Input struct {
	Skill               string      `json:"skill"`
	Impact              Level       `json:"impact"`
	Reversibility       Level       `json:"reversibility"`
	Ambiguity           Level       `json:"ambiguity"`
	SecuritySensitivity Level       `json:"security_sensitivity"`
	StateMutation       Level       `json:"state_mutation"`
	EvidenceQuality     Level       `json:"evidence_quality"`
	ValidationCost      Level       `json:"validation_cost"`
	Overrides           UserChoices `json:"overrides,omitempty"`
	Evidence            []Evidence  `json:"evidence,omitempty"`
}

// UserChoices contains explicit user selections that the policy may preserve.
type UserChoices struct {
	ModelTier      string `json:"model_tier,omitempty"`
	ValidationTier string `json:"validation_tier,omitempty"`
}

// Decision is the stable machine-readable selection result.
type Decision struct {
	SchemaVersion    int                     `json:"schema_version"`
	PolicyVersion    int                     `json:"policy_version"`
	Rule             string                  `json:"rule"`
	Strategy         string                  `json:"strategy"`
	Skill            string                  `json:"skill"`
	ModelTier        string                  `json:"model_tier"`
	ContextProfile   string                  `json:"context_profile"`
	ValidationTier   string                  `json:"validation_tier"`
	Parallelism      string                  `json:"parallelism"`
	MaxRetries       int                     `json:"max_retries"`
	MaxElapsedMillis int                     `json:"max_elapsed_millis"`
	Escalation       string                  `json:"escalation"`
	Authority        environment.Permissions `json:"authority"`
	Signals          InputSignals            `json:"signals"`
	Reasons          []string                `json:"reasons"`
	Overrides        Overrides               `json:"overrides,omitempty"`
	Evidence         []Evidence              `json:"evidence,omitempty"`
	Outcome          string                  `json:"outcome"`
}

// InputSignals is the normalized, explicit input subset persisted in a decision.
type InputSignals struct {
	Impact              Level `json:"impact"`
	Reversibility       Level `json:"reversibility"`
	Ambiguity           Level `json:"ambiguity"`
	SecuritySensitivity Level `json:"security_sensitivity"`
	StateMutation       Level `json:"state_mutation"`
	EvidenceQuality     Level `json:"evidence_quality"`
	ValidationCost      Level `json:"validation_cost"`
}

type ValidationReport struct {
	Valid         bool     `json:"valid"`
	PolicyPath    string   `json:"policy_path"`
	SchemaVersion int      `json:"schema_version"`
	StrategyCount int      `json:"strategy_count"`
	RuleCount     int      `json:"rule_count"`
	Findings      []string `json:"findings,omitempty"`
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

// Validate checks policy structure and references to existing contracts.
func Validate(root string) ValidationReport {
	report := ValidationReport{PolicyPath: Path}
	policy, err := Load(root)
	if err != nil {
		report.Findings = []string{err.Error()}
		return report
	}
	report.SchemaVersion = policy.SchemaVersion
	report.StrategyCount = len(policy.Strategies)
	report.RuleCount = len(policy.Rules)
	graphDocument, graphErr := graph.Load(root)
	if graphErr != nil {
		report.Findings = []string{graphErr.Error()}
		return report
	}
	findings := ValidatePolicy(policy, graphDocument)
	sort.Strings(findings)
	report.Findings = findings
	report.Valid = len(findings) == 0
	return report
}

// ValidatePolicy checks a policy without reading the repository twice.
func ValidatePolicy(policy *Policy, graphDocument *graph.Graph) []string {
	if policy == nil {
		return []string{"policy is missing"}
	}
	findings := []string{}
	if policy.SchemaVersion != CurrentSchemaVersion {
		findings = append(findings, fmt.Sprintf("schema_version must be %d, got %d", CurrentSchemaVersion, policy.SchemaVersion))
	}
	if !policy.Defaults.DeterministicChecksFirst {
		findings = append(findings, "defaults must run deterministic checks first")
	}
	if policy.Defaults.MaxElapsedMillis <= 0 || policy.Defaults.MaxInputTokens <= 0 || policy.Defaults.MaxOutputTokens <= 0 || policy.Defaults.MaxCostMicros <= 0 {
		findings = append(findings, "defaults must define positive resource bounds")
	}
	validSignals := map[string]map[string]bool{}
	seenSignals := map[string]bool{}
	for index, signal := range policy.Signals {
		if signal.ID == "" || seenSignals[signal.ID] {
			findings = append(findings, fmt.Sprintf("signals[%d].id is missing or duplicated", index))
		}
		seenSignals[signal.ID] = true
		if len(signal.Levels) == 0 || strings.TrimSpace(signal.Description) == "" {
			findings = append(findings, fmt.Sprintf("signal %q must define levels and a description", signal.ID))
		}
		levels := map[string]bool{}
		for _, level := range signal.Levels {
			if level == "" || levels[level] {
				findings = append(findings, fmt.Sprintf("signal %q has an empty or duplicate level", signal.ID))
			}
			levels[level] = true
		}
		validSignals[signal.ID] = levels
	}
	for _, required := range []string{"impact", "reversibility", "ambiguity", "security_sensitivity", "state_mutation", "evidence_quality", "validation_cost"} {
		if !seenSignals[required] {
			findings = append(findings, fmt.Sprintf("required signal %q is missing", required))
		}
	}
	strategies := map[string]Strategy{}
	for index, item := range policy.Strategies {
		if item.ID == "" || strategies[item.ID].ID != "" {
			findings = append(findings, fmt.Sprintf("strategies[%d].id is missing or duplicated", index))
		}
		strategies[item.ID] = item
		if item.ModelTier != "high" && item.ModelTier != "mid" && item.ModelTier != "low" {
			findings = append(findings, fmt.Sprintf("strategy %q has invalid model tier %q", item.ID, item.ModelTier))
		}
		if item.ContextProfile != "selected-skill" {
			findings = append(findings, fmt.Sprintf("strategy %q must use selected-skill context profile", item.ID))
		}
		if item.ValidationTier != "tier-1" && item.ValidationTier != "tier-2" && item.ValidationTier != "tier-3" && item.ValidationTier != "tier-4" {
			findings = append(findings, fmt.Sprintf("strategy %q has invalid validation tier %q", item.ID, item.ValidationTier))
		}
		if item.Parallelism != "single-agent" && item.Parallelism != "independent-candidates" && item.Parallelism != "fan-out-investigation" {
			findings = append(findings, fmt.Sprintf("strategy %q has invalid parallelism %q", item.ID, item.Parallelism))
		}
		if item.MaxRetries < 0 || item.MaxRetries > 1 || item.MaxElapsedMillis <= 0 {
			findings = append(findings, fmt.Sprintf("strategy %q has invalid retry or elapsed bound", item.ID))
		}
		if item.AuthorityCeiling != "graph" {
			findings = append(findings, fmt.Sprintf("strategy %q must use graph authority ceiling", item.ID))
		}
	}
	if _, ok := strategies[policy.Defaults.Strategy]; !ok {
		findings = append(findings, fmt.Sprintf("defaults.strategy %q is not defined", policy.Defaults.Strategy))
	}
	if len(policy.Rules) == 0 {
		findings = append(findings, "rules must not be empty")
	}
	seenRules := map[string]bool{}
	defaultCount := 0
	for index, rule := range policy.Rules {
		if rule.ID == "" || seenRules[rule.ID] {
			findings = append(findings, fmt.Sprintf("rules[%d].id is missing or duplicated", index))
		}
		seenRules[rule.ID] = true
		if _, ok := strategies[rule.Strategy]; !ok {
			findings = append(findings, fmt.Sprintf("rule %q references unknown strategy %q", rule.ID, rule.Strategy))
		}
		if strings.TrimSpace(rule.Reason) == "" {
			findings = append(findings, fmt.Sprintf("rule %q has no reason", rule.ID))
		}
		if len(rule.When) == 0 {
			defaultCount++
		}
		for field, values := range rule.When {
			levels, ok := validSignals[field]
			if !ok {
				findings = append(findings, fmt.Sprintf("rule %q references unknown signal %q", rule.ID, field))
			}
			if len(values) == 0 {
				findings = append(findings, fmt.Sprintf("rule %q has no values for signal %q", rule.ID, field))
			}
			for _, value := range values {
				if !levels[value] {
					findings = append(findings, fmt.Sprintf("rule %q uses invalid value %q for signal %q", rule.ID, value, field))
				}
			}
		}
	}
	if defaultCount != 1 {
		findings = append(findings, fmt.Sprintf("rules must contain exactly one default rule, got %d", defaultCount))
	}
	if graphDocument == nil {
		findings = append(findings, "graph is missing")
	}
	return findings
}

// Select loads the policy and graph and returns one deterministic decision.
func Select(root string, input Input) (Decision, error) {
	policy, err := Load(root)
	if err != nil {
		return Decision{}, err
	}
	graphDocument, err := graph.Load(root)
	if err != nil {
		return Decision{}, err
	}
	if findings := ValidatePolicy(policy, graphDocument); len(findings) > 0 {
		return Decision{}, fmt.Errorf("execution strategy policy is invalid: %s", strings.Join(findings, "; "))
	}
	return SelectWithPolicy(policy, graphDocument, input)
}

// SelectWithPolicy returns a decision from already loaded policy data.
func SelectWithPolicy(policy *Policy, graphDocument *graph.Graph, input Input) (Decision, error) {
	if policy == nil || graphDocument == nil {
		return Decision{}, fmt.Errorf("policy and graph are required")
	}
	if input.Skill == "" {
		return Decision{}, fmt.Errorf("skill is required")
	}
	skill, ok := graphDocument.Skill(input.Skill)
	if !ok {
		return Decision{}, fmt.Errorf("skill %q is not in the graph", input.Skill)
	}
	if findings := validateInput(policy, input); len(findings) > 0 {
		return Decision{}, fmt.Errorf("strategy input is invalid: %s", strings.Join(findings, "; "))
	}
	if signal := firstUnknownSignal(input); signal != "" {
		profile := findStrategy(policy, "blocked-evidence")
		_, permissions, err := environment.Derive(*skill)
		if err != nil {
			return Decision{}, err
		}
		return decisionFromProfile(profile, policy, input, permissions, "unknown-signal", "Required signal "+signal+" is unknown.", "ask_user"), nil
	}
	var selected Rule
	for _, rule := range policy.Rules {
		if matches(rule, input) {
			selected = rule
			break
		}
	}
	if selected.ID == "" {
		return Decision{}, fmt.Errorf("no execution strategy rule matched")
	}
	profiles := map[string]Strategy{}
	for _, profile := range policy.Strategies {
		profiles[profile.ID] = profile
	}
	profile := profiles[selected.Strategy]
	_, permissions, err := environment.Derive(*skill)
	if err != nil {
		return Decision{}, err
	}
	decision := Decision{
		SchemaVersion:    CurrentSchemaVersion,
		PolicyVersion:    policy.SchemaVersion,
		Rule:             selected.ID,
		Strategy:         selected.Strategy,
		Skill:            input.Skill,
		ModelTier:        profile.ModelTier,
		ContextProfile:   profile.ContextProfile,
		ValidationTier:   profile.ValidationTier,
		Parallelism:      profile.Parallelism,
		MaxRetries:       profile.MaxRetries,
		MaxElapsedMillis: profile.MaxElapsedMillis,
		Escalation:       profile.Escalation,
		Authority:        permissions,
		Signals: InputSignals{
			Impact: input.Impact, Reversibility: input.Reversibility, Ambiguity: input.Ambiguity,
			SecuritySensitivity: input.SecuritySensitivity, StateMutation: input.StateMutation,
			EvidenceQuality: input.EvidenceQuality, ValidationCost: input.ValidationCost,
		},
		Reasons:  []string{selected.Reason},
		Evidence: append([]Evidence(nil), input.Evidence...),
		Outcome:  "selected",
	}
	if input.EvidenceQuality == Missing || input.EvidenceQuality == Unknown {
		decision.Outcome = "ask_user"
	}
	if input.StateMutation == External && permissions.ExternalMutation == "none" {
		decision.Outcome = "blocked"
		decision.Rule = "authority-boundary"
		decision.ModelTier = profiles["blocked-authority"].ModelTier
		decision.ValidationTier = profiles["blocked-authority"].ValidationTier
		decision.Parallelism = profiles["blocked-authority"].Parallelism
		decision.MaxRetries = profiles["blocked-authority"].MaxRetries
		decision.Escalation = profiles["blocked-authority"].Escalation
		decision.Reasons = append(decision.Reasons, "selected skill does not declare external mutation authority")
	}
	decision = applyOverrides(decision, input.Overrides)
	return decision, nil
}

func decisionFromProfile(profile Strategy, policy *Policy, input Input, permissions environment.Permissions, rule, reason, outcome string) Decision {
	return Decision{
		SchemaVersion: CurrentSchemaVersion, PolicyVersion: policy.SchemaVersion, Rule: rule,
		Strategy: profile.ID,
		Skill:    input.Skill, ModelTier: profile.ModelTier, ContextProfile: profile.ContextProfile,
		ValidationTier: profile.ValidationTier, Parallelism: profile.Parallelism,
		MaxRetries: profile.MaxRetries, MaxElapsedMillis: profile.MaxElapsedMillis,
		Escalation: profile.Escalation, Authority: permissions,
		Signals: InputSignals{Impact: input.Impact, Reversibility: input.Reversibility, Ambiguity: input.Ambiguity, SecuritySensitivity: input.SecuritySensitivity, StateMutation: input.StateMutation, EvidenceQuality: input.EvidenceQuality, ValidationCost: input.ValidationCost},
		Reasons: []string{reason}, Evidence: append([]Evidence(nil), input.Evidence...), Outcome: outcome,
	}
}

func findStrategy(policy *Policy, id string) Strategy {
	for _, profile := range policy.Strategies {
		if profile.ID == id {
			return profile
		}
	}
	return Strategy{}
}

func matches(rule Rule, input Input) bool {
	values := map[string]string{
		"impact": string(input.Impact), "reversibility": string(input.Reversibility),
		"ambiguity": string(input.Ambiguity), "security_sensitivity": string(input.SecuritySensitivity),
		"state_mutation": string(input.StateMutation), "evidence_quality": string(input.EvidenceQuality),
		"validation_cost": string(input.ValidationCost),
	}
	for field, expected := range rule.When {
		matched := false
		for _, value := range expected {
			if values[field] == value {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func applyOverrides(decision Decision, choices UserChoices) Decision {
	if choices.ModelTier != "" {
		decision.Overrides.ModelTier = choices.ModelTier
		decision.Overrides.Values = append(decision.Overrides.Values, "explicit-model-tier")
		decision.ModelTier = choices.ModelTier
	}
	if choices.ValidationTier != "" && tierRank(choices.ValidationTier) >= tierRank(decision.ValidationTier) {
		decision.Overrides.ValidationTier = choices.ValidationTier
		decision.Overrides.Values = append(decision.Overrides.Values, "explicit-validation-tier")
		decision.ValidationTier = choices.ValidationTier
	} else if choices.ValidationTier != "" {
		if decision.Outcome != "blocked" {
			decision.Outcome = "ask_user"
		}
		decision.Overrides.Values = append(decision.Overrides.Values, "unsafe-validation-tier-rejected")
		decision.Reasons = append(decision.Reasons, "explicit validation tier would reduce required verification")
	}
	return decision
}

func tierRank(value string) int {
	switch value {
	case "tier-1":
		return 1
	case "tier-2":
		return 2
	case "tier-3":
		return 3
	case "tier-4":
		return 4
	default:
		return 0
	}
}

func validateInput(policy *Policy, input Input) []string {
	findings := []string{}
	checks := map[string]Level{
		"impact": input.Impact, "reversibility": input.Reversibility, "ambiguity": input.Ambiguity,
		"security_sensitivity": input.SecuritySensitivity, "state_mutation": input.StateMutation,
		"evidence_quality": input.EvidenceQuality, "validation_cost": input.ValidationCost,
	}
	for name, value := range checks {
		if value == "" {
			findings = append(findings, name+" is required")
		}
	}
	valid := map[string]map[string]bool{}
	for _, signal := range policy.Signals {
		valid[signal.ID] = map[string]bool{}
		for _, level := range signal.Levels {
			valid[signal.ID][level] = true
		}
	}
	for name, value := range checks {
		if value != "" && !valid[name][string(value)] {
			findings = append(findings, fmt.Sprintf("%s value %q is not allowed", name, value))
		}
	}
	if input.Overrides.ModelTier != "" && !oneOf(input.Overrides.ModelTier, "high", "mid", "low") {
		findings = append(findings, "overrides.model_tier is invalid")
	}
	if input.Overrides.ValidationTier != "" && tierRank(input.Overrides.ValidationTier) == 0 {
		findings = append(findings, "overrides.validation_tier is invalid")
	}
	for index, item := range input.Evidence {
		if strings.TrimSpace(item.Kind) == "" || strings.TrimSpace(item.Ref) == "" {
			findings = append(findings, fmt.Sprintf("evidence[%d] requires kind and ref", index))
		}
	}
	return findings
}

func firstUnknownSignal(input Input) string {
	checks := []struct {
		name  string
		value Level
	}{
		{"impact", input.Impact}, {"reversibility", input.Reversibility}, {"ambiguity", input.Ambiguity},
		{"security_sensitivity", input.SecuritySensitivity}, {"state_mutation", input.StateMutation},
		{"evidence_quality", input.EvidenceQuality}, {"validation_cost", input.ValidationCost},
	}
	for _, item := range checks {
		if item.value == Unknown {
			return item.name
		}
	}
	return ""
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

// Check validates the policy and prints a stable result.
func Check(root string, out, errOut io.Writer) int {
	report := Validate(root)
	if report.Valid {
		fmt.Fprintf(out, "execution strategy policy valid: %d strategies, %d rules, schema version %d.\n", report.StrategyCount, report.RuleCount, report.SchemaVersion)
		return 0
	}
	for _, finding := range report.Findings {
		fmt.Fprintln(errOut, finding)
	}
	return 1
}

// WriteJSON writes stable indented JSON.
func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
