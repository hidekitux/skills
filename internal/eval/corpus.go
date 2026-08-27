package eval

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// catalogSkill is the subset of a CATALOG.yml entry the corpus check needs.
type catalogSkill struct {
	Name   string
	Status string
}

// loadCatalogSkills decodes the skills list of root/CATALOG.yml.
func loadCatalogSkills(root string) ([]catalogSkill, error) {
	content, err := os.ReadFile(filepath.Join(root, "CATALOG.yml"))
	if err != nil {
		return nil, err
	}
	var catalog map[string]any
	if err := yaml.Unmarshal(content, &catalog); err != nil {
		return nil, err
	}
	raw, ok := catalog["skills"].([]any)
	if !ok {
		return nil, fmt.Errorf("CATALOG.yml must contain a skills list")
	}
	var skills []catalogSkill
	for index, entry := range raw {
		mapping, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("CATALOG.yml skills[%d] must be a mapping", index+1)
		}
		name, _ := mapping["name"].(string)
		status, _ := mapping["status"].(string)
		if name == "" {
			return nil, fmt.Errorf("CATALOG.yml skills[%d].name is required", index+1)
		}
		skills = append(skills, catalogSkill{Name: name, Status: status})
	}
	return skills, nil
}

// fixtureExists reports whether evaluations/fixtures/<key> is a directory.
func fixtureExists(root, key string) bool {
	info, err := os.Stat(filepath.Join(root, fixtureBase, key))
	return err == nil && info.IsDir()
}

// validScenarioKind reports whether kind is a supported scenario kind.
func validScenarioKind(kind string) bool {
	switch kind {
	case KindPositive, KindNegative, KindBoundary, KindSafety:
		return true
	}
	return false
}

// failureKind reports whether a kind counts as the failure/boundary half of
// Acceptance criterion 1.
func failureKind(kind string) bool {
	return kind == KindNegative || kind == KindBoundary
}

// validateScenario applies the scenario schema rules and returns findings.
func validateScenario(sc *Scenario, catalogNames map[string]bool, seen map[string]bool, root string, findings *[]string) {
	if sc.ID == "" {
		*findings = append(*findings, "scenario is missing id")
		return
	}
	if seen[sc.ID] {
		*findings = append(*findings, fmt.Sprintf("scenario id %q is duplicated", sc.ID))
	}
	seen[sc.ID] = true
	if sc.Title == "" {
		*findings = append(*findings, fmt.Sprintf("%s: title is required", sc.ID))
	}
	if !catalogNames[sc.Skill] && sc.Skill != E2ESkill {
		*findings = append(*findings, fmt.Sprintf("%s: skill %q is not a cataloged skill or e2e", sc.ID, sc.Skill))
	}
	if !validScenarioKind(sc.Kind) {
		*findings = append(*findings, fmt.Sprintf("%s: kind %q must be positive, negative, boundary, or safety", sc.ID, sc.Kind))
	}
	if sc.Prompt != "" && len(sc.Stages) > 0 {
		*findings = append(*findings, fmt.Sprintf("%s: use either prompt or stages, not both", sc.ID))
	}
	if sc.Prompt == "" && len(sc.Stages) == 0 {
		*findings = append(*findings, fmt.Sprintf("%s: prompt or stages is required", sc.ID))
	}
	if len(sc.Stages) > 0 && sc.Skill != E2ESkill {
		*findings = append(*findings, fmt.Sprintf("%s: stages are only allowed for skill: e2e", sc.ID))
	}
	for index, stage := range sc.Stages {
		if !catalogNames[stage.Skill] {
			*findings = append(*findings, fmt.Sprintf("%s: stage %d skill %q is not a cataloged skill", sc.ID, index+1, stage.Skill))
		}
		if stage.Prompt == "" {
			*findings = append(*findings, fmt.Sprintf("%s: stage %d prompt is required", sc.ID, index+1))
		}
	}
	if sc.Fixture != "" && !fixtureExists(root, sc.Fixture) {
		*findings = append(*findings, fmt.Sprintf("%s: fixture %q does not resolve under evaluations/fixtures/", sc.ID, sc.Fixture))
	}
	if sc.Expectations.Handoff == "" {
		*findings = append(*findings, fmt.Sprintf("%s: expectations.handoff is required", sc.ID))
	}
	validateRubric(sc, findings)
	validatePromptLeaks(sc, catalogNames, findings)
}

// validateRubric requires guidance for every scoring dimension.
func validateRubric(sc *Scenario, findings *[]string) {
	rubric := sc.Rubric
	if rubric.TriggerSelection == "" || rubric.TaskCompletion == "" || rubric.EvidenceQuality == "" ||
		rubric.ScopeControl == "" || rubric.Safety == "" || rubric.UserCorrectionCount == "" ||
		rubric.HandoffQuality == "" {
		*findings = append(*findings, fmt.Sprintf("%s: rubric must define all seven dimensions", sc.ID))
	}
}

// validatePromptLeaks enforces that prompts never reveal the expected answer:
// the named handoff and every cataloged skill name (trigger selection is
// measured, not assumed) must be absent from all stage prompts.
func validatePromptLeaks(sc *Scenario, catalogNames map[string]bool, findings *[]string) {
	for _, prompt := range sc.prompts() {
		if sc.Expectations.Handoff != "" && strings.Contains(prompt, sc.Expectations.Handoff) {
			*findings = append(*findings, fmt.Sprintf("%s: prompt reveals the expected handoff %q", sc.ID, sc.Expectations.Handoff))
		}
		for name := range catalogNames {
			if strings.Contains(prompt, name) {
				*findings = append(*findings, fmt.Sprintf("%s: prompt names the skill %q, revealing trigger selection", sc.ID, name))
			}
		}
	}
}

// catalogHandoffNames returns the cataloged skill names used to decide which
// scenario handoffs are deterministic transcript assertions: only a handoff
// that names a cataloged skill is the next-owner contract in
// docs/skill-contract.md. A missing or unreadable catalog disables the
// assertion without failing the run.
func catalogHandoffNames(root string) map[string]bool {
	catalog, err := loadCatalogSkills(root)
	if err != nil {
		return nil
	}
	names := make(map[string]bool, len(catalog))
	for _, skill := range catalog {
		names[skill.Name] = true
	}
	return names
}

// qualifyingRecord reports whether a decoded report record satisfies the
// documented promotion evidence (docs/evaluation.md): a pass verdict for the
// skill with a completed seven-dimension rubric review. A passing verdict for
// another skill in the same file, or a pass without rubric evidence, does not
// count.
func qualifyingRecord(skill string, record map[string]any) bool {
	if record["verdict"] != VerdictPass || record["skill"] != skill {
		return false
	}
	if record["rubric_review"] != RubricComplete {
		return false
	}
	scores, ok := record["rubric_scores"].(map[string]any)
	if !ok {
		return false
	}
	for _, dimension := range scoreOrder {
		value, ok := scores[dimension].(float64)
		if !ok || value < 1 || value > 5 {
			return false
		}
	}
	return true
}

// reportRecords decodes a machine-readable report file into records. JSONL
// reports carry one record per line; .json reports may be a single record or
// an array of records. Unreadable or malformed files yield no records so a
// broken report cannot satisfy the promotion gate.
func reportRecords(path string) []map[string]any {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	if filepath.Ext(path) == ".jsonl" {
		var records []map[string]any
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var record map[string]any
			if err := json.Unmarshal([]byte(line), &record); err != nil {
				return nil
			}
			records = append(records, record)
		}
		return records
	}
	var records []map[string]any
	if err := json.Unmarshal(content, &records); err == nil {
		return records
	}
	var record map[string]any
	if err := json.Unmarshal(content, &record); err != nil {
		return nil
	}
	return []map[string]any{record}
}

// hasStableEvidence reports whether a machine-readable evaluation report
// under evaluations/reports/ records a qualifying pass for the skill,
// satisfying the documented promotion gate for status: stable entries. The
// skill and the pass verdict must belong to the same record.
func hasStableEvidence(root, skill string) bool {
	reportsDir := filepath.Join(root, "evaluations", "reports")
	found := false
	_ = filepath.WalkDir(reportsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		if ext != ".jsonl" && ext != ".json" {
			return nil
		}
		for _, record := range reportRecords(path) {
			if qualifyingRecord(skill, record) {
				found = true
				return filepath.SkipAll
			}
		}
		return nil
	})
	return found
}

// CheckCorpus validates the evaluation corpus against the repository
// requirements: schema, expected-answer leakage, Acceptance criterion 1
// coverage for every cataloged skill, and the promotion gate for
// status: stable entries. It returns 0 on success or 1 when findings exist.
func CheckCorpus(root string, out, errOut io.Writer) int {
	var findings []string

	catalog, err := loadCatalogSkills(root)
	if err != nil {
		fmt.Fprintf(errOut, "Evaluation corpus check failed: %v\n", err)
		return 1
	}
	catalogNames := make(map[string]bool, len(catalog))
	for _, skill := range catalog {
		catalogNames[skill.Name] = true
	}

	scenarios, err := LoadAllScenarios(root)
	if err != nil {
		fmt.Fprintf(errOut, "Evaluation corpus check failed: %v\n", err)
		return 1
	}
	if len(scenarios) == 0 {
		fmt.Fprintln(errOut, "Evaluation corpus check failed: no scenarios found under evaluations/scenarios/")
		return 1
	}

	seen := make(map[string]bool, len(scenarios))
	bySkill := make(map[string]map[string]bool, len(catalog))
	for _, sc := range scenarios {
		validateScenario(sc, catalogNames, seen, root, &findings)
		perSkill := bySkill[sc.Skill]
		if perSkill == nil {
			perSkill = make(map[string]bool)
			bySkill[sc.Skill] = perSkill
		}
		perSkill[sc.Kind] = true
	}

	for _, skill := range catalog {
		kinds := bySkill[skill.Name]
		if !kinds[KindPositive] {
			findings = append(findings, fmt.Sprintf("skill %q has no positive success scenario", skill.Name))
		}
		if !kinds[KindNegative] && !kinds[KindBoundary] {
			findings = append(findings, fmt.Sprintf("skill %q has no failure or boundary scenario", skill.Name))
		}
		if skill.Status == "stable" && !hasStableEvidence(root, skill.Name) {
			findings = append(findings, fmt.Sprintf("skill %q is stable but has no passing evaluation evidence under evaluations/reports/", skill.Name))
		}
	}

	if len(findings) > 0 {
		fmt.Fprintln(errOut, "Evaluation corpus check failed:")
		sort.Strings(findings)
		for _, finding := range findings {
			fmt.Fprintf(errOut, "- %s\n", finding)
		}
		return 1
	}
	fmt.Fprintf(out, "Evaluation corpus check passed: %d scenario(s) for %d cataloged skill(s).\n", len(scenarios), len(catalog))
	return 0
}
