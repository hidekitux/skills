package eval

import (
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

// hasStableEvidence reports whether a machine-readable evaluation report
// under evaluations/reports/ records a passing verdict for the skill,
// satisfying the documented promotion gate for status: stable entries.
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
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		text := string(content)
		if strings.Contains(text, `"skill":"`+skill+`"`) && strings.Contains(text, `"verdict":"pass"`) {
			found = true
			return filepath.SkipAll
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
		if !kinds[KindNegative] && !kinds[KindBoundary] && !kinds[KindSafety] {
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
