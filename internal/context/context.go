// Package context compiles a bounded, task-aware skill context package.
package context

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hidekitux/skills/internal/graph"
	"github.com/hidekitux/skills/internal/instructions"
)

var ErrRequiredOverflow = errors.New("required context exceeds its category budget")

// Signals are observable task facts. Raw command output and private content
// are intentionally not fields in this input contract.
type Signals struct {
	TaskKind      string             `json:"task_kind,omitempty" yaml:"task_kind,omitempty"`
	Paths         []string           `json:"paths,omitempty" yaml:"paths,omitempty"`
	Diagnostics   []DiagnosticSignal `json:"diagnostics,omitempty" yaml:"diagnostics,omitempty"`
	FSL           *FSLSignal         `json:"fsl,omitempty" yaml:"fsl,omitempty"`
	PriorEvidence []EvidenceSignal   `json:"prior_evidence,omitempty" yaml:"prior_evidence,omitempty"`
}

type DiagnosticSignal struct {
	Producer string `json:"producer" yaml:"producer"`
	Code     string `json:"code" yaml:"code"`
	Result   string `json:"result,omitempty" yaml:"result,omitempty"`
}

type FSLSignal struct {
	Spec   string `json:"spec,omitempty" yaml:"spec,omitempty"`
	Result string `json:"result" yaml:"result"`
}

type EvidenceSignal struct {
	Kind   string `json:"kind" yaml:"kind"`
	Ref    string `json:"ref" yaml:"ref"`
	Result string `json:"result,omitempty" yaml:"result,omitempty"`
}

// Manifest is the privacy-safe explanation of one compilation. It records
// module identity and measurements, never module content.
type Manifest struct {
	SchemaVersion    int                 `json:"schema_version"`
	Skill            string              `json:"skill"`
	Encoding         string              `json:"encoding"`
	TokenizerModule  string              `json:"tokenizer_module"`
	TokenizerVersion string              `json:"tokenizer_version"`
	Budgets          graph.ContextBudget `json:"budgets"`
	Measured         map[string]int      `json:"measured"`
	TotalTokens      int                 `json:"total_tokens"`
	TotalBudget      int                 `json:"total_budget"`
	Decisions        []Decision          `json:"decisions"`
	Overflow         *Overflow           `json:"overflow,omitempty"`
}

type Decision struct {
	ID       string                `json:"id"`
	Category graph.ContextCategory `json:"category"`
	Source   string                `json:"source,omitempty"`
	Reason   string                `json:"reason"`
	Required bool                  `json:"required"`
	Included bool                  `json:"included"`
	Tokens   int                   `json:"tokens,omitempty"`
}

type Overflow struct {
	Category       graph.ContextCategory `json:"category"`
	RequiredTokens int                   `json:"required_tokens"`
	Budget         int                   `json:"budget"`
	Action         string                `json:"action"`
}

// Module is one selected context payload. It is returned to the caller but is
// deliberately excluded from Manifest so traces can persist only provenance.
type Module struct {
	ID       string                `json:"id"`
	Category graph.ContextCategory `json:"category"`
	Source   string                `json:"source,omitempty"`
	Content  string                `json:"content"`
}

type Package struct {
	Manifest Manifest `json:"manifest"`
	Modules  []Module `json:"modules"`
}

type Compiler struct {
	Root    string
	Counter instructions.Counter
}

func (c Compiler) Compile(skillID string, signals Signals) (Package, error) {
	graphDocument, err := graph.Load(c.Root)
	if err != nil {
		return Package{}, err
	}
	skill, ok := graphDocument.Skill(skillID)
	if !ok {
		return Package{}, fmt.Errorf("skill %q is not in the graph", skillID)
	}
	profile, ok := graphDocument.Context.Profiles[skillID]
	if !ok {
		return Package{}, fmt.Errorf("context profile is missing for skill %q", skillID)
	}
	if c.Counter == nil {
		return Package{}, errors.New("context compiler requires a token counter")
	}

	manifest := Manifest{
		SchemaVersion:    graph.CurrentContextSchemaVersion,
		Skill:            skillID,
		Encoding:         instructions.EncodingName,
		TokenizerModule:  instructions.TokenizerModule,
		TokenizerVersion: instructions.TokenizerVersion,
		Budgets:          profile.Budgets,
		Measured:         map[string]int{},
	}
	for _, entry := range budgetEntries(profile.Budgets) {
		manifest.TotalBudget += entry.budget
	}

	modulesByID := make(map[string]graph.ContextModule, len(graphDocument.Context.Modules))
	for _, module := range graphDocument.Context.Modules {
		modulesByID[module.ID] = module
	}
	moduleIDs := append([]string{}, graphDocument.Context.GlobalModules...)
	moduleIDs = append(moduleIDs, profile.Modules...)
	moduleIDs = unique(moduleIDs)
	var packageModules []Module
	var required []candidate
	var optional []candidate

	corePath := filepath.Join(c.Root, filepath.FromSlash(skill.Path))
	coreContent, err := os.ReadFile(corePath)
	if err != nil {
		return Package{}, fmt.Errorf("read %s: %w", skill.Path, err)
	}
	coreTokens, err := c.Counter.Count(string(coreContent))
	if err != nil {
		return Package{}, fmt.Errorf("count %s: %w", skill.Path, err)
	}
	coreCandidate := candidate{decision: Decision{ID: "core." + skillID, Category: graph.ContextCoreInstructions, Source: skill.Path, Reason: "core-skill", Required: true, Included: true, Tokens: coreTokens}, module: Module{ID: "core." + skillID, Category: graph.ContextCoreInstructions, Source: skill.Path, Content: string(coreContent)}}
	required = append(required, coreCandidate)
	for _, invariantID := range profile.CriticalInvariants {
		for _, invariant := range graphDocument.Context.Invariants {
			if invariant.ID != invariantID {
				continue
			}
			content := invariant.ID + ": " + invariant.Description
			tokens, countErr := c.Counter.Count(content)
			if countErr != nil {
				return Package{}, fmt.Errorf("count invariant %s: %w", invariant.ID, countErr)
			}
			required = append(required, candidate{decision: Decision{ID: "invariant." + invariant.ID, Category: graph.ContextCoreInstructions, Source: "graph.context.invariants", Reason: "critical-invariant", Required: true, Included: true, Tokens: tokens}, module: Module{ID: "invariant." + invariant.ID, Category: graph.ContextCoreInstructions, Source: "graph.context.invariants", Content: content}})
		}
	}

	for _, moduleID := range moduleIDs {
		module, exists := modulesByID[moduleID]
		if !exists {
			return Package{}, fmt.Errorf("context module %q is not defined", moduleID)
		}
		active, reason := activation(module, signals)
		if !active {
			manifest.Decisions = append(manifest.Decisions, Decision{ID: module.ID, Category: module.Category, Source: module.Source, Reason: reason, Required: module.Required})
			continue
		}
		content, err := c.moduleContent(module, signals)
		if err != nil {
			return Package{}, err
		}
		tokens, err := c.Counter.Count(content)
		if err != nil {
			return Package{}, fmt.Errorf("count context module %s: %w", module.ID, err)
		}
		candidate := candidate{decision: Decision{ID: module.ID, Category: module.Category, Source: module.Source, Required: module.Required, Tokens: tokens}, module: Module{ID: module.ID, Category: module.Category, Source: module.Source, Content: content}, priority: module.Priority}
		if module.Required {
			candidate.decision.Reason = "required-module"
			candidate.decision.Included = true
			required = append(required, candidate)
		} else {
			candidate.decision.Reason = "selected-by-signal"
			optional = append(optional, candidate)
		}
	}

	for _, item := range required {
		manifest.Measured[string(item.decision.Category)] += item.decision.Tokens
	}
	for _, entry := range budgetEntries(profile.Budgets) {
		if manifest.Measured[string(entry.category)] > entry.budget {
			manifest.TotalTokens = 0
			for _, measured := range manifest.Measured {
				manifest.TotalTokens += measured
			}
			manifest.Overflow = &Overflow{Category: entry.category, RequiredTokens: manifest.Measured[string(entry.category)], Budget: entry.budget, Action: "stop-and-escalate"}
			manifest.Decisions = append(manifest.Decisions, decisions(required)...)
			sortDecisions(manifest.Decisions)
			return Package{Manifest: manifest}, fmt.Errorf("%w: %s requires %d tokens but budget is %d", ErrRequiredOverflow, entry.category, manifest.Measured[string(entry.category)], entry.budget)
		}
	}
	sort.Slice(optional, func(i, j int) bool {
		if optional[i].module.Category != optional[j].module.Category {
			return optional[i].module.Category < optional[j].module.Category
		}
		if optional[i].priority != optional[j].priority {
			return optional[i].priority > optional[j].priority
		}
		return optional[i].module.ID < optional[j].module.ID
	})
	for _, item := range optional {
		category := string(item.decision.Category)
		budget := budgetFor(profile.Budgets, item.decision.Category)
		if manifest.Measured[category]+item.decision.Tokens <= budget {
			item.decision.Included = true
			manifest.Measured[category] += item.decision.Tokens
			packageModules = append(packageModules, item.module)
		} else {
			item.decision.Reason = "budget-evicted"
		}
		manifest.Decisions = append(manifest.Decisions, item.decision)
	}
	for _, item := range required {
		packageModules = append(packageModules, item.module)
		decision := item.decision
		manifest.Decisions = append(manifest.Decisions, decision)
	}
	sortDecisions(manifest.Decisions)
	sort.Slice(packageModules, func(i, j int) bool { return packageModules[i].ID < packageModules[j].ID })
	manifest.TotalTokens = 0
	for _, measured := range manifest.Measured {
		manifest.TotalTokens += measured
	}
	return Package{Manifest: manifest, Modules: packageModules}, nil
}

type candidate struct {
	decision Decision
	module   Module
	priority int
}

type budgetEntry struct {
	category graph.ContextCategory
	budget   int
}

func budgetEntries(b graph.ContextBudget) []budgetEntry {
	return []budgetEntry{
		{category: graph.ContextCoreInstructions, budget: b.CoreInstructions},
		{category: graph.ContextConditionalRefs, budget: b.ConditionalRefs},
		{category: graph.ContextRepositoryEvidence, budget: b.RepositoryEvidence},
		{category: graph.ContextValidatorFeedback, budget: b.ValidatorFeedback},
	}
}

func budgetFor(b graph.ContextBudget, category graph.ContextCategory) int {
	for _, entry := range budgetEntries(b) {
		if entry.category == category {
			return entry.budget
		}
	}
	return 0
}

func sortDecisions(items []Decision) {
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
}

func decisions(items []candidate) []Decision {
	result := make([]Decision, 0, len(items))
	for _, item := range items {
		result = append(result, item.decision)
	}
	return result
}

func unique(values []string) []string {
	seen := map[string]bool{}
	result := []string{}
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func activation(module graph.ContextModule, signals Signals) (bool, string) {
	rule := module.Activation
	switch module.Signal {
	case "validator_feedback":
		if len(signals.Diagnostics) == 0 {
			return false, "not-applicable-no-validator-feedback"
		}
	case "fsl_result":
		if signals.FSL == nil {
			return false, "not-applicable-no-fsl-result"
		}
	case "prior_evidence":
		if len(signals.PriorEvidence) == 0 {
			return false, "not-applicable-no-prior-evidence"
		}
	}
	if len(rule.TaskKinds) > 0 && !contains(rule.TaskKinds, signals.TaskKind) {
		return false, "not-applicable-task-kind"
	}
	if len(rule.PathPatterns) > 0 && !anyPathMatches(rule.PathPatterns, signals.Paths) {
		return false, "not-applicable-path"
	}
	if len(rule.DiagnosticCodes) > 0 && !diagnosticMatches(rule.DiagnosticCodes, signals.Diagnostics) {
		return false, "not-applicable-diagnostic"
	}
	if len(rule.FSLResults) > 0 && (signals.FSL == nil || !contains(rule.FSLResults, signals.FSL.Result)) {
		return false, "not-applicable-fsl-result"
	}
	if len(rule.EvidenceKinds) > 0 && !evidenceMatches(rule.EvidenceKinds, signals.PriorEvidence) {
		return false, "not-applicable-evidence"
	}
	return true, "selected-by-signal"
}

func contains(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

func anyPathMatches(patterns, paths []string) bool {
	for _, candidate := range paths {
		for _, pattern := range patterns {
			if pattern == candidate || strings.HasSuffix(pattern, "/**") && strings.HasPrefix(candidate, strings.TrimSuffix(pattern, "**")) {
				return true
			}
			if matched, _ := path.Match(pattern, candidate); matched {
				return true
			}
		}
	}
	return false
}

func diagnosticMatches(codes []string, diagnostics []DiagnosticSignal) bool {
	for _, diagnostic := range diagnostics {
		if contains(codes, diagnostic.Code) || contains(codes, diagnostic.Producer+":"+diagnostic.Code) {
			return true
		}
	}
	return false
}

func evidenceMatches(kinds []string, evidence []EvidenceSignal) bool {
	for _, item := range evidence {
		if contains(kinds, item.Kind) {
			return true
		}
	}
	return false
}

func (c Compiler) moduleContent(module graph.ContextModule, signals Signals) (string, error) {
	if module.Source != "" {
		content, err := os.ReadFile(filepath.Join(c.Root, filepath.FromSlash(module.Source)))
		if err != nil {
			return "", fmt.Errorf("read context module %s: %w", module.ID, err)
		}
		return string(content), nil
	}
	switch module.Signal {
	case "validator_feedback":
		var lines []string
		for _, diagnostic := range signals.Diagnostics {
			lines = append(lines, "validator feedback: "+diagnostic.Producer+"/"+diagnostic.Code+" result="+diagnostic.Result)
		}
		return strings.Join(lines, "\n"), nil
	case "fsl_result":
		if signals.FSL == nil {
			return "", errors.New("fsl_result signal selected without fsl input")
		}
		return "FSL result: spec=" + signals.FSL.Spec + " result=" + signals.FSL.Result, nil
	case "prior_evidence":
		var lines []string
		for _, item := range signals.PriorEvidence {
			lines = append(lines, "prior evidence: kind="+item.Kind+" ref="+item.Ref+" result="+item.Result)
		}
		return strings.Join(lines, "\n"), nil
	default:
		return "", fmt.Errorf("unsupported context signal %q", module.Signal)
	}
}

// ValidateManifest enforces the trace-safe subset of a manifest.
func ValidateManifest(manifest Manifest) []string {
	findings := []string{}
	if manifest.SchemaVersion != graph.CurrentContextSchemaVersion {
		findings = append(findings, "context schema version is unsupported")
	}
	if manifest.Skill == "" || strings.ContainsAny(manifest.Skill, " \t\r\n") {
		findings = append(findings, "context skill is missing or invalid")
	}
	if manifest.Encoding != instructions.EncodingName || manifest.TokenizerModule != instructions.TokenizerModule || manifest.TokenizerVersion != instructions.TokenizerVersion {
		findings = append(findings, "context tokenizer metadata is invalid")
	}
	totalTokens := 0
	for category, tokens := range manifest.Measured {
		if tokens < 0 {
			findings = append(findings, "context measured token count is negative for "+category)
		}
		totalTokens += tokens
	}
	if manifest.TotalTokens != totalTokens {
		findings = append(findings, "context total_tokens does not match measured counts")
	}
	totalBudget := 0
	for _, entry := range budgetEntries(manifest.Budgets) {
		totalBudget += entry.budget
		if manifest.Overflow == nil && manifest.Measured[string(entry.category)] > entry.budget {
			findings = append(findings, "context measured tokens exceed budget for "+string(entry.category))
		}
	}
	if manifest.TotalBudget != totalBudget {
		findings = append(findings, "context total_budget does not match budgets")
	}
	for _, decision := range manifest.Decisions {
		if decision.ID == "" || strings.ContainsAny(decision.ID, " \t\r\n") {
			findings = append(findings, "context decision id is missing or invalid")
		}
		if decision.Tokens < 0 {
			findings = append(findings, "context decision token count is negative")
		}
		if strings.ContainsAny(decision.Reason, " \t\r\n") {
			findings = append(findings, "context decision reason is not a safe identifier")
		}
		if strings.ContainsAny(decision.Source, " \t\r\n") || strings.Contains(strings.ToLower(decision.Source), "content") {
			findings = append(findings, "context decision source is not a safe path")
		}
	}
	if manifest.Overflow != nil && manifest.Overflow.Action != "stop-and-escalate" {
		findings = append(findings, "context overflow action is invalid")
	}
	return findings
}

// DecodeSignals decodes only the public signal contract and rejects trailing
// JSON values so a malformed input cannot silently change selection.
func DecodeSignals(data []byte) (Signals, error) {
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	var signals Signals
	if err := decoder.Decode(&signals); err != nil {
		return Signals{}, err
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return Signals{}, errors.New("signals contain trailing JSON")
	} else if !errors.Is(err, io.EOF) {
		return Signals{}, fmt.Errorf("decode trailing signals: %w", err)
	}
	return signals, nil
}
