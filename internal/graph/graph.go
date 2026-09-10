// Package graph loads and validates the repository-owned skill graph.
package graph

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hidekitux/skills/internal/discover"
	"gopkg.in/yaml.v3"
)

// Path is the repository-relative location of the authoritative graph.
const Path = "workflow/skill-graph.yml"

// CurrentSchemaVersion is the graph schema version implemented by this package.
const CurrentSchemaVersion = 1

// Context schema values are versioned independently from the workflow graph
// because context packages evolve without changing handoff semantics.
const CurrentContextSchemaVersion = 1

// ContextCategory identifies one budgeted part of a compiled package.
type ContextCategory string

const (
	ContextCoreInstructions   ContextCategory = "core_instructions"
	ContextConditionalRefs    ContextCategory = "conditional_references"
	ContextRepositoryEvidence ContextCategory = "repository_evidence"
	ContextValidatorFeedback  ContextCategory = "validator_feedback"
)

var validContextCategories = map[ContextCategory]bool{
	ContextCoreInstructions:   true,
	ContextConditionalRefs:    true,
	ContextRepositoryEvidence: true,
	ContextValidatorFeedback:  true,
}

// Definition is a named graph registry entry.
type Definition struct {
	ID          string `yaml:"id" json:"id"`
	Description string `yaml:"description" json:"description"`
}

// ContextBudget declares the maximum number of fixed-encoding tokens allowed
// for each part of one skill's context package.
type ContextBudget struct {
	CoreInstructions   int `yaml:"core_instructions" json:"core_instructions"`
	ConditionalRefs    int `yaml:"conditional_references" json:"conditional_references"`
	RepositoryEvidence int `yaml:"repository_evidence" json:"repository_evidence"`
	ValidatorFeedback  int `yaml:"validator_feedback" json:"validator_feedback"`
}

// ContextActivation describes observable signals that activate a module. All
// non-empty signal groups must match; an empty activation matches every use of
// the profile that declares the module. Signal-backed modules additionally
// require at least one corresponding signal value.
type ContextActivation struct {
	TaskKinds       []string `yaml:"task_kinds,omitempty" json:"task_kinds,omitempty"`
	PathPatterns    []string `yaml:"path_patterns,omitempty" json:"path_patterns,omitempty"`
	DiagnosticCodes []string `yaml:"diagnostic_codes,omitempty" json:"diagnostic_codes,omitempty"`
	FSLResults      []string `yaml:"fsl_results,omitempty" json:"fsl_results,omitempty"`
	EvidenceKinds   []string `yaml:"evidence_kinds,omitempty" json:"evidence_kinds,omitempty"`
}

// ContextModule declares a file-backed or signal-backed context module.
type ContextModule struct {
	ID         string            `yaml:"id" json:"id"`
	Category   ContextCategory   `yaml:"category" json:"category"`
	Source     string            `yaml:"source,omitempty" json:"source,omitempty"`
	Signal     string            `yaml:"signal,omitempty" json:"signal,omitempty"`
	Required   bool              `yaml:"required" json:"required"`
	Priority   int               `yaml:"priority" json:"priority"`
	Activation ContextActivation `yaml:"activation,omitempty" json:"activation,omitempty"`
}

// ContextProfile declares the package contract for one skill.
type ContextProfile struct {
	Budgets            ContextBudget `yaml:"budgets" json:"budgets"`
	CriticalInvariants []string      `yaml:"critical_invariants" json:"critical_invariants"`
	Modules            []string      `yaml:"modules" json:"modules"`
}

// ContextConfig is the machine-readable context compiler contract.
type ContextConfig struct {
	SchemaVersion int                       `yaml:"schema_version" json:"schema_version"`
	Invariants    []Definition              `yaml:"invariants" json:"invariants"`
	Modules       []ContextModule           `yaml:"modules" json:"modules"`
	GlobalModules []string                  `yaml:"global_modules" json:"global_modules"`
	Profiles      map[string]ContextProfile `yaml:"profiles" json:"profiles"`
}

// Authority declares the permissions a skill requires. These values describe
// authority; they do not grant it to a caller.
type Authority struct {
	Repository       string `yaml:"repository" json:"repository"`
	Git              string `yaml:"git" json:"git"`
	GitHub           string `yaml:"github" json:"github"`
	ExternalMutation string `yaml:"external_mutation" json:"external_mutation"`
}

// Destination identifies either a skill node or a terminal outcome.
type Destination struct {
	Skill           string `yaml:"skill,omitempty" json:"skill,omitempty"`
	TerminalOutcome string `yaml:"terminal_outcome,omitempty" json:"terminal_outcome,omitempty"`
}

// Retry describes the finite retry or termination rule for a transition.
type Retry struct {
	MaxAttempts          int    `yaml:"max_attempts,omitempty" json:"max_attempts,omitempty"`
	TerminationCondition string `yaml:"termination_condition,omitempty" json:"termination_condition,omitempty"`
}

// Transition moves one skill outcome to a skill or terminal outcome.
type Transition struct {
	ID          string      `yaml:"id" json:"id"`
	Outcome     string      `yaml:"outcome" json:"outcome"`
	Artifact    string      `yaml:"artifact" json:"artifact"`
	Condition   string      `yaml:"condition,omitempty" json:"condition,omitempty"`
	Destination Destination `yaml:"destination" json:"destination"`
	Retry       *Retry      `yaml:"retry,omitempty" json:"retry,omitempty"`
}

// Skill is one graph node for one cataloged skill.
type Skill struct {
	ID            string       `yaml:"id" json:"id"`
	Path          string       `yaml:"path" json:"path"`
	Layer         string       `yaml:"layer" json:"layer"`
	Capabilities  []string     `yaml:"capabilities" json:"capabilities"`
	Inputs        []string     `yaml:"inputs" json:"inputs"`
	Outputs       []string     `yaml:"outputs" json:"outputs"`
	Mutability    string       `yaml:"mutability" json:"mutability"`
	Authority     Authority    `yaml:"authority" json:"authority"`
	Prerequisites []string     `yaml:"prerequisites" json:"prerequisites"`
	Outcomes      []string     `yaml:"outcomes" json:"outcomes"`
	Transitions   []Transition `yaml:"transitions" json:"transitions"`
}

// Graph is the versioned machine-readable skill graph.
type Graph struct {
	SchemaVersion         int           `yaml:"schema_version" json:"schema_version"`
	Artifacts             []Definition  `yaml:"artifacts" json:"artifacts"`
	Prerequisites         []Definition  `yaml:"prerequisites" json:"prerequisites"`
	TerminalOutcomes      []string      `yaml:"terminal_outcomes" json:"terminal_outcomes"`
	TerminationConditions []Definition  `yaml:"termination_conditions" json:"termination_conditions"`
	Conditions            []Definition  `yaml:"conditions" json:"conditions"`
	Context               ContextConfig `yaml:"context" json:"context"`
	Skills                []Skill       `yaml:"skills" json:"skills"`
}

// ValidationReport is the stable machine-readable result of graph validation.
type ValidationReport struct {
	Valid         bool     `json:"valid"`
	GraphPath     string   `json:"graph_path"`
	SchemaVersion int      `json:"schema_version"`
	SkillCount    int      `json:"skill_count"`
	Findings      []string `json:"findings,omitempty"`
}

type catalogEntry struct {
	Layer string
}

type catalogDocument struct {
	Skills []struct {
		Name  string `yaml:"name"`
		Layer string `yaml:"layer"`
	} `yaml:"skills"`
}

var graphDocEntry = regexp.MustCompile("^- \\x60([a-z0-9][a-z0-9-]*)\\x60$")

// Load reads the authoritative graph from root.
func Load(root string) (*Graph, error) {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(Path)))
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", Path, err)
	}
	var graph Graph
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&graph); err != nil {
		return nil, fmt.Errorf("decode %s: %w", Path, err)
	}
	return &graph, nil
}

// Skill returns the named skill node.
func (g *Graph) Skill(id string) (*Skill, bool) {
	for index := range g.Skills {
		if g.Skills[index].ID == id {
			return &g.Skills[index], true
		}
	}
	return nil, false
}

// Validate loads and validates the graph, catalog, discovered skill paths, and
// documented graph view rooted at root.
func Validate(root string) ValidationReport {
	report := ValidationReport{GraphPath: Path}
	graph, err := Load(root)
	if err != nil {
		report.Findings = []string{err.Error()}
		return report
	}
	report.SchemaVersion = graph.SchemaVersion
	report.SkillCount = len(graph.Skills)
	report.Findings = validateGraph(root, graph)
	sort.Strings(report.Findings)
	report.Valid = len(report.Findings) == 0
	return report
}

// WriteJSON writes a stable indented JSON representation of value.
func WriteJSON(w io.Writer, value any) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func validateGraph(root string, graph *Graph) []string {
	findings := []string{}
	if graph.SchemaVersion != CurrentSchemaVersion {
		findings = append(findings, fmt.Sprintf("schema_version must be %d, got %d", CurrentSchemaVersion, graph.SchemaVersion))
	}
	artifacts := validateDefinitions("artifact", graph.Artifacts, &findings)
	prerequisites := validateDefinitions("prerequisite", graph.Prerequisites, &findings)
	terminationConditions := validateDefinitions("termination condition", graph.TerminationConditions, &findings)
	conditions := validateDefinitions("condition", graph.Conditions, &findings)
	inputs := map[string]bool{}
	for name := range artifacts {
		inputs[name] = true
	}
	for name := range prerequisites {
		inputs[name] = true
	}
	terminalOutcomes := map[string]bool{}
	for index, outcome := range graph.TerminalOutcomes {
		if outcome == "" {
			findings = append(findings, fmt.Sprintf("terminal_outcomes[%d] is empty", index))
			continue
		}
		if terminalOutcomes[outcome] {
			findings = append(findings, fmt.Sprintf("terminal outcome %q is duplicated", outcome))
		}
		terminalOutcomes[outcome] = true
	}

	catalog, catalogFindings := loadCatalog(root)
	findings = append(findings, catalogFindings...)
	nodes := map[string]bool{}
	for index := range graph.Skills {
		skill := &graph.Skills[index]
		if skill.ID == "" {
			findings = append(findings, fmt.Sprintf("skills[%d].id is required", index))
			continue
		}
		if nodes[skill.ID] {
			findings = append(findings, fmt.Sprintf("skill %q is duplicated", skill.ID))
		}
		nodes[skill.ID] = true
		if entry, ok := catalog[skill.ID]; !ok {
			findings = append(findings, fmt.Sprintf("skill %q is missing from CATALOG.yml", skill.ID))
		} else if entry.Layer != skill.Layer {
			findings = append(findings, fmt.Sprintf("skill %q layer %q differs from CATALOG.yml layer %q", skill.ID, skill.Layer, entry.Layer))
		}
	}
	for index := range graph.Skills {
		findings = append(findings, validateSkill(&graph.Skills[index], artifacts, inputs, prerequisites, terminalOutcomes, terminationConditions, conditions, nodes)...)
	}
	for name := range catalog {
		if !nodes[name] {
			findings = append(findings, fmt.Sprintf("cataloged skill %q is missing from the graph", name))
		}
	}
	findings = append(findings, validateDiscoveredPaths(root, graph)...)
	findings = append(findings, validateCycles(graph, nodes, terminationConditions)...)
	findings = append(findings, validateDocumentation(root, graph)...)
	findings = append(findings, validateContext(root, graph)...)
	return findings
}

func validateContext(root string, graph *Graph) []string {
	findings := []string{}
	contextConfig := graph.Context
	if contextConfig.SchemaVersion != CurrentContextSchemaVersion {
		findings = append(findings, fmt.Sprintf("context.schema_version must be %d, got %d", CurrentContextSchemaVersion, contextConfig.SchemaVersion))
	}

	invariants := map[string]bool{}
	for index, invariant := range contextConfig.Invariants {
		if invariant.ID == "" {
			findings = append(findings, fmt.Sprintf("context.invariants[%d].id is required", index))
			continue
		}
		if invariants[invariant.ID] {
			findings = append(findings, fmt.Sprintf("context invariant %q is duplicated", invariant.ID))
		}
		invariants[invariant.ID] = true
		if strings.TrimSpace(invariant.Description) == "" {
			findings = append(findings, fmt.Sprintf("context invariant %q description is required", invariant.ID))
		}
	}

	modules := map[string]ContextModule{}
	for index, module := range contextConfig.Modules {
		if module.ID == "" {
			findings = append(findings, fmt.Sprintf("context.modules[%d].id is required", index))
			continue
		}
		if _, exists := modules[module.ID]; exists {
			findings = append(findings, fmt.Sprintf("context module %q is duplicated", module.ID))
		}
		modules[module.ID] = module
		if !validContextCategories[module.Category] {
			findings = append(findings, fmt.Sprintf("context module %q has invalid category %q", module.ID, module.Category))
		}
		if module.Priority < 0 {
			findings = append(findings, fmt.Sprintf("context module %q priority must not be negative", module.ID))
		}
		if (module.Source == "") == (module.Signal == "") {
			findings = append(findings, fmt.Sprintf("context module %q must define exactly one source or signal", module.ID))
		}
		if module.Source != "" {
			findings = append(findings, validateContextSource(root, module)...)
		}
		if module.Signal != "" && !oneOf(module.Signal, "validator_feedback", "fsl_result", "prior_evidence") {
			findings = append(findings, fmt.Sprintf("context module %q signal %q is invalid", module.ID, module.Signal))
		}
		findings = append(findings, validateContextActivation(module)...)
	}

	findings = append(findings, validateContextModuleRefs("context.global_modules", contextConfig.GlobalModules, modules)...)
	for skillID, profile := range contextConfig.Profiles {
		if !skillExists(graph.Skills, skillID) {
			findings = append(findings, fmt.Sprintf("context profile %q names unknown skill", skillID))
		}
		if profile.Budgets.CoreInstructions <= 0 || profile.Budgets.ConditionalRefs <= 0 || profile.Budgets.RepositoryEvidence <= 0 || profile.Budgets.ValidatorFeedback <= 0 {
			findings = append(findings, fmt.Sprintf("context profile %q must define positive budgets for all categories", skillID))
		}
		for _, invariantID := range profile.CriticalInvariants {
			if !invariants[invariantID] {
				findings = append(findings, fmt.Sprintf("context profile %q references unknown invariant %q", skillID, invariantID))
			}
		}
		findings = append(findings, validateContextModuleRefs(fmt.Sprintf("context profile %q modules", skillID), profile.Modules, modules)...)
	}
	for _, skill := range graph.Skills {
		if _, exists := contextConfig.Profiles[skill.ID]; !exists {
			findings = append(findings, fmt.Sprintf("context profile is missing for skill %q", skill.ID))
		}
	}
	return findings
}

func validateContextSource(root string, module ContextModule) []string {
	findings := []string{}
	clean := filepath.Clean(filepath.FromSlash(module.Source))
	if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return []string{fmt.Sprintf("context module %q source must be a repository-relative path", module.ID)}
	}
	info, err := os.Stat(filepath.Join(root, clean))
	if err != nil {
		findings = append(findings, fmt.Sprintf("context module %q source %q is not readable: %v", module.ID, module.Source, err))
	} else if !info.Mode().IsRegular() {
		findings = append(findings, fmt.Sprintf("context module %q source %q is not a regular file", module.ID, module.Source))
	}
	return findings
}

func validateContextActivation(module ContextModule) []string {
	findings := []string{}
	groups := map[string][]string{
		"task_kinds":       module.Activation.TaskKinds,
		"path_patterns":    module.Activation.PathPatterns,
		"diagnostic_codes": module.Activation.DiagnosticCodes,
		"fsl_results":      module.Activation.FSLResults,
		"evidence_kinds":   module.Activation.EvidenceKinds,
	}
	for name, values := range groups {
		seen := map[string]bool{}
		for _, value := range values {
			if strings.TrimSpace(value) == "" {
				findings = append(findings, fmt.Sprintf("context module %q has an empty %s activation", module.ID, name))
			}
			if seen[value] {
				findings = append(findings, fmt.Sprintf("context module %q duplicates %s activation %q", module.ID, name, value))
			}
			seen[value] = true
		}
	}
	return findings
}

func validateContextModuleRefs(label string, references []string, modules map[string]ContextModule) []string {
	findings := []string{}
	seen := map[string]bool{}
	for _, reference := range references {
		if _, exists := modules[reference]; !exists {
			findings = append(findings, fmt.Sprintf("%s references unknown module %q", label, reference))
		}
		if seen[reference] {
			findings = append(findings, fmt.Sprintf("%s duplicates module %q", label, reference))
		}
		seen[reference] = true
	}
	return findings
}

func skillExists(skills []Skill, id string) bool {
	for _, skill := range skills {
		if skill.ID == id {
			return true
		}
	}
	return false
}

func validateDefinitions(kind string, definitions []Definition, findings *[]string) map[string]bool {
	seen := map[string]bool{}
	for index, definition := range definitions {
		if definition.ID == "" {
			*findings = append(*findings, fmt.Sprintf("%s[%d].id is required", kind, index))
			continue
		}
		if seen[definition.ID] {
			*findings = append(*findings, fmt.Sprintf("%s %q is duplicated", kind, definition.ID))
		}
		seen[definition.ID] = true
		if strings.TrimSpace(definition.Description) == "" {
			*findings = append(*findings, fmt.Sprintf("%s %q description is required", kind, definition.ID))
		}
	}
	return seen
}

func loadCatalog(root string) (map[string]catalogEntry, []string) {
	content, err := os.ReadFile(filepath.Join(root, "CATALOG.yml"))
	if err != nil {
		return nil, []string{fmt.Sprintf("read CATALOG.yml: %v", err)}
	}
	var document catalogDocument
	if err := yaml.Unmarshal(content, &document); err != nil {
		return nil, []string{fmt.Sprintf("decode CATALOG.yml: %v", err)}
	}
	catalog := map[string]catalogEntry{}
	findings := []string{}
	for index, skill := range document.Skills {
		if skill.Name == "" {
			findings = append(findings, fmt.Sprintf("CATALOG.yml skills[%d].name is required", index))
			continue
		}
		if _, exists := catalog[skill.Name]; exists {
			findings = append(findings, fmt.Sprintf("CATALOG.yml skill %q is duplicated", skill.Name))
		}
		catalog[skill.Name] = catalogEntry{Layer: skill.Layer}
	}
	return catalog, findings
}

func validateSkill(skill *Skill, artifacts, inputs, prerequisites, terminalOutcomes, terminationConditions, conditions, nodes map[string]bool) []string {
	findings := []string{}
	if skill.Path == "" {
		findings = append(findings, fmt.Sprintf("skill %q path is required", skill.ID))
	}
	if len(skill.Capabilities) == 0 {
		findings = append(findings, fmt.Sprintf("skill %q must declare capabilities", skill.ID))
	}
	if !validMutability(skill.Mutability) {
		findings = append(findings, fmt.Sprintf("skill %q mutability %q is invalid", skill.ID, skill.Mutability))
	}
	findings = append(findings, validateAuthority(skill)...)
	findings = append(findings, validateReferences(fmt.Sprintf("skill %q input", skill.ID), skill.Inputs, inputs)...)
	findings = append(findings, validateReferences(fmt.Sprintf("skill %q output", skill.ID), skill.Outputs, artifacts)...)
	findings = append(findings, validateReferences(fmt.Sprintf("skill %q prerequisite", skill.ID), skill.Prerequisites, prerequisites)...)
	outcomes := map[string]bool{}
	for _, outcome := range skill.Outcomes {
		if outcome == "" {
			findings = append(findings, fmt.Sprintf("skill %q has an empty outcome", skill.ID))
		}
		if outcomes[outcome] {
			findings = append(findings, fmt.Sprintf("skill %q outcome %q is duplicated", skill.ID, outcome))
		}
		outcomes[outcome] = true
	}
	transitionIDs := map[string]bool{}
	for _, transition := range skill.Transitions {
		if transition.ID == "" {
			findings = append(findings, fmt.Sprintf("skill %q has a transition without an id", skill.ID))
		} else if transitionIDs[transition.ID] {
			findings = append(findings, fmt.Sprintf("skill %q transition %q is duplicated", skill.ID, transition.ID))
		}
		transitionIDs[transition.ID] = true
		if !outcomes[transition.Outcome] {
			findings = append(findings, fmt.Sprintf("skill %q transition %q names unknown outcome %q", skill.ID, transition.ID, transition.Outcome))
		}
		if transition.Condition != "" && !conditions[transition.Condition] {
			findings = append(findings, fmt.Sprintf("skill %q transition %q names unknown condition %q", skill.ID, transition.ID, transition.Condition))
		}
		if !artifacts[transition.Artifact] {
			findings = append(findings, fmt.Sprintf("skill %q transition %q names unknown artifact %q", skill.ID, transition.ID, transition.Artifact))
		}
		if (transition.Destination.Skill == "") == (transition.Destination.TerminalOutcome == "") {
			findings = append(findings, fmt.Sprintf("skill %q transition %q must name exactly one destination kind", skill.ID, transition.ID))
		} else if transition.Destination.Skill != "" && !nodes[transition.Destination.Skill] {
			findings = append(findings, fmt.Sprintf("skill %q transition %q names unknown skill destination %q", skill.ID, transition.ID, transition.Destination.Skill))
		} else if transition.Destination.TerminalOutcome != "" && !terminalOutcomes[transition.Destination.TerminalOutcome] {
			findings = append(findings, fmt.Sprintf("skill %q transition %q names unknown terminal outcome %q", skill.ID, transition.ID, transition.Destination.TerminalOutcome))
		}
		if transition.Retry != nil {
			if transition.Retry.MaxAttempts < 0 {
				findings = append(findings, fmt.Sprintf("skill %q transition %q has a negative max_attempts", skill.ID, transition.ID))
			}
			if transition.Retry.TerminationCondition != "" && !terminationConditions[transition.Retry.TerminationCondition] {
				findings = append(findings, fmt.Sprintf("skill %q transition %q names unknown termination condition %q", skill.ID, transition.ID, transition.Retry.TerminationCondition))
			}
		}
	}
	return findings
}

func validateReferences(label string, references []string, registry map[string]bool) []string {
	findings := []string{}
	seen := map[string]bool{}
	for _, reference := range references {
		if !registry[reference] {
			findings = append(findings, fmt.Sprintf("%s references unknown %q", label, reference))
		}
		if seen[reference] {
			findings = append(findings, fmt.Sprintf("%s reference %q is duplicated", label, reference))
		}
		seen[reference] = true
	}
	return findings
}

func validateAuthority(skill *Skill) []string {
	findings := []string{}
	if !oneOf(skill.Authority.Repository, "none", "read", "write") {
		findings = append(findings, fmt.Sprintf("skill %q authority.repository %q is invalid", skill.ID, skill.Authority.Repository))
	}
	if !oneOf(skill.Authority.Git, "none", "read", "write") {
		findings = append(findings, fmt.Sprintf("skill %q authority.git %q is invalid", skill.ID, skill.Authority.Git))
	}
	if !oneOf(skill.Authority.GitHub, "none", "read", "write") {
		findings = append(findings, fmt.Sprintf("skill %q authority.github %q is invalid", skill.ID, skill.Authority.GitHub))
	}
	if !oneOf(skill.Authority.ExternalMutation, "none", "issue", "pull_request", "repository_configuration") {
		findings = append(findings, fmt.Sprintf("skill %q authority.external_mutation %q is invalid", skill.ID, skill.Authority.ExternalMutation))
	}
	if skill.Mutability == "read_only" && (skill.Authority.Repository == "write" || skill.Authority.Git == "write" || skill.Authority.GitHub == "write" || skill.Authority.ExternalMutation != "none") {
		findings = append(findings, fmt.Sprintf("read-only skill %q declares write authority", skill.ID))
	}
	return findings
}

func validateDiscoveredPaths(root string, graph *Graph) []string {
	findings := []string{}
	byName := discover.ByName(root)
	for _, skill := range graph.Skills {
		matches := byName[skill.ID]
		if len(matches) == 0 {
			findings = append(findings, fmt.Sprintf("skill %q has no discovered SKILL.md", skill.ID))
			continue
		}
		if len(matches) > 1 {
			findings = append(findings, fmt.Sprintf("skill %q has %d discovered SKILL.md paths", skill.ID, len(matches)))
			continue
		}
		expected := filepath.ToSlash(filepath.Join(matches[0].Dir, "SKILL.md"))
		if skill.Path != expected {
			findings = append(findings, fmt.Sprintf("skill %q path %q differs from discovered path %q", skill.ID, skill.Path, expected))
		}
	}
	return findings
}

func validateDocumentation(root string, graph *Graph) []string {
	findings := []string{}
	contract, err := os.ReadFile(filepath.Join(root, "docs", "skill-contract.md"))
	if err != nil {
		findings = append(findings, fmt.Sprintf("read docs/skill-contract.md: %v", err))
	} else {
		text := string(contract)
		for _, required := range []string{Path, "cmd/read-skill-graph", "cmd/validate-skill-graph"} {
			if !strings.Contains(text, required) {
				findings = append(findings, fmt.Sprintf("docs/skill-contract.md does not reference %q", required))
			}
		}
	}
	document, err := os.ReadFile(filepath.Join(root, "docs", "skill-graph.md"))
	if err != nil {
		return append(findings, fmt.Sprintf("read docs/skill-graph.md: %v", err))
	}
	section := graphInventorySection(string(document))
	if section == "" {
		return append(findings, "docs/skill-graph.md has no graph inventory markers")
	}
	documented := map[string]bool{}
	for _, line := range strings.Split(section, "\n") {
		match := graphDocEntry.FindStringSubmatch(strings.TrimSpace(line))
		if len(match) == 2 {
			if documented[match[1]] {
				findings = append(findings, fmt.Sprintf("docs/skill-graph.md lists skill %q more than once", match[1]))
			}
			documented[match[1]] = true
		}
	}
	for _, skill := range graph.Skills {
		if !documented[skill.ID] {
			findings = append(findings, fmt.Sprintf("docs/skill-graph.md omits skill %q", skill.ID))
		}
	}
	for skill := range documented {
		if _, ok := graph.Skill(skill); !ok {
			findings = append(findings, fmt.Sprintf("docs/skill-graph.md lists unknown skill %q", skill))
		}
	}
	return findings
}

func graphInventorySection(document string) string {
	const start = "<!-- skills:graph-inventory:start -->"
	const end = "<!-- skills:graph-inventory:end -->"
	startIndex := strings.Index(document, start)
	endIndex := strings.Index(document, end)
	if startIndex < 0 || endIndex < startIndex {
		return ""
	}
	return document[startIndex+len(start) : endIndex]
}

func validateCycles(graph *Graph, nodes map[string]bool, terminationConditions map[string]bool) []string {
	findings := []string{}
	adjacency := map[string][]Transition{}
	for _, skill := range graph.Skills {
		for _, transition := range skill.Transitions {
			if transition.Destination.Skill != "" && nodes[transition.Destination.Skill] {
				adjacency[skill.ID] = append(adjacency[skill.ID], transition)
			}
		}
	}
	index := 0
	indices := map[string]int{}
	lowlink := map[string]int{}
	onStack := map[string]bool{}
	stack := []string{}
	var visit func(string)
	visit = func(node string) {
		indices[node] = index
		lowlink[node] = index
		index++
		stack = append(stack, node)
		onStack[node] = true
		for _, transition := range adjacency[node] {
			next := transition.Destination.Skill
			if _, seen := indices[next]; !seen {
				visit(next)
				if lowlink[next] < lowlink[node] {
					lowlink[node] = lowlink[next]
				}
			} else if onStack[next] && indices[next] < lowlink[node] {
				lowlink[node] = indices[next]
			}
		}
		if lowlink[node] != indices[node] {
			return
		}
		component := []string{}
		for {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component = append(component, last)
			if last == node {
				break
			}
		}
		cyclic := len(component) > 1
		if !cyclic && len(adjacency[node]) > 0 {
			for _, transition := range adjacency[node] {
				if transition.Destination.Skill == node {
					cyclic = true
				}
			}
		}
		if !cyclic {
			return
		}
		members := map[string]bool{}
		for _, member := range component {
			members[member] = true
		}
		for _, member := range component {
			for _, transition := range adjacency[member] {
				if !members[transition.Destination.Skill] {
					continue
				}
				bounded := transition.Retry != nil && transition.Retry.MaxAttempts > 0
				terminated := transition.Retry != nil && transition.Retry.TerminationCondition != "" && terminationConditions[transition.Retry.TerminationCondition]
				if !bounded && !terminated {
					findings = append(findings, fmt.Sprintf("cycle transition %s.%s has no retry bound or termination condition", member, transition.ID))
				}
			}
		}
	}
	for _, skill := range graph.Skills {
		if _, seen := indices[skill.ID]; !seen {
			visit(skill.ID)
		}
	}
	return findings
}

func validMutability(value string) bool {
	return oneOf(value, "read_only", "repository_write", "external_mutation")
}

func oneOf(value string, values ...string) bool {
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}
