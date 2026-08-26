// Package eval implements the repository's outcome-based behavioral
// evaluation system: scenario corpus loading and static validation,
// deterministic assertions, host execution, rubric review, and run reporting.
//
// Behavioral evaluation is deliberately separate from metadata validation and
// `gh skill publish --dry-run`: it measures whether a skill selects the right
// workflow and produces a correct, safe, useful result for a realistic
// request (Issue 173).
package eval

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Scenario kinds and the e2e marker skill used for cross-handoff flows.
const (
	KindPositive = "positive"
	KindNegative = "negative"
	KindBoundary = "boundary"
	KindSafety   = "safety"
	E2ESkill     = "e2e"
)

// Scenario verdicts recorded for every evaluated scenario (Acceptance
// criterion 3: the output distinguishes deterministic failures, rubric
// scores, skipped cases, and infrastructure errors).
const (
	VerdictPass    = "pass"
	VerdictFail    = "fail"
	VerdictSkipped = "skipped"
	VerdictInfra   = "infrastructure_error"
	RubricPending  = "pending"
	RubricComplete = "complete"
	RubricNA       = "not-applicable"
)

// Stage is one handoff step of an e2e flow. Each stage names the skill under
// evaluation and an isolated prompt that must not reveal the expected answer.
type Stage struct {
	Skill  string `yaml:"skill"`
	Prompt string `yaml:"prompt"`
}

// FileContains is a deterministic assertion that a file in the sandbox
// contains a literal substring.
type FileContains struct {
	Path    string `yaml:"path"`
	Pattern string `yaml:"pattern"`
}

// CommandRun is a deterministic assertion that a command run in the sandbox
// exits with the expected code (default 0).
type CommandRun struct {
	Run  string `yaml:"run"`
	Dir  string `yaml:"dir"`
	Exit int    `yaml:"exit"`
}

// Expectations are the deterministic, machine-checked outcome assertions for
// a scenario. They gate the verdict; rubric dimensions never override them.
type Expectations struct {
	Handoff           string         `yaml:"handoff"`
	TranscriptMust    []string       `yaml:"transcript_must"`
	TranscriptMustAny []string       `yaml:"transcript_must_any"`
	TranscriptMustNot []string       `yaml:"transcript_must_not"`
	FilesMustExist    []string       `yaml:"files_must_exist"`
	FilesMustNotExist []string       `yaml:"files_must_not_exist"`
	FileContains      []FileContains `yaml:"file_contains"`
	UnchangedFiles    []string       `yaml:"unchanged_files"`
	CommandRun        []CommandRun   `yaml:"command_run"`
}

// Rubric is the reviewer guidance for the seven observable scoring
// dimensions. See evaluations/rubric.md for the 1-5 anchors.
type Rubric struct {
	TriggerSelection    string `yaml:"trigger_selection"`
	TaskCompletion      string `yaml:"task_completion"`
	EvidenceQuality     string `yaml:"evidence_quality"`
	ScopeControl        string `yaml:"scope_control"`
	Safety              string `yaml:"safety"`
	UserCorrectionCount string `yaml:"user_correction_count"`
	HandoffQuality      string `yaml:"handoff_quality"`
}

// Scenario is one behavioral evaluation scenario: representative positive,
// negative, boundary, or safety request with deterministic expectations and
// rubric guidance.
type Scenario struct {
	ID            string       `yaml:"id"`
	Skill         string       `yaml:"skill"`
	Kind          string       `yaml:"kind"`
	Smoke         bool         `yaml:"smoke"`
	Title         string       `yaml:"title"`
	GithubSandbox bool         `yaml:"github_sandbox"`
	Fixture       string       `yaml:"fixture"`
	Prompt        string       `yaml:"prompt"`
	Stages        []Stage      `yaml:"stages"`
	Expectations  Expectations `yaml:"expectations"`
	Rubric        Rubric       `yaml:"rubric"`
	Corrections   []string     `yaml:"corrections"`
}

// prompts returns the stage prompts of the scenario. A single-skill scenario
// has one implicit stage derived from its top-level prompt.
func (s *Scenario) prompts() []string {
	if len(s.Stages) > 0 {
		prompts := make([]string, 0, len(s.Stages))
		for _, stage := range s.Stages {
			prompts = append(prompts, stage.Prompt)
		}
		return prompts
	}
	if s.Prompt != "" {
		return []string{s.Prompt}
	}
	return nil
}

// LoadScenario reads and decodes one scenario file.
func LoadScenario(path string) (*Scenario, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var sc Scenario
	if err := yaml.Unmarshal(content, &sc); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if sc.ID == "" {
		return nil, fmt.Errorf("parse %s: id is required", path)
	}
	return &sc, nil
}

// scenarioBase is the repository-relative scenario directory base.
const scenarioBase = "evaluations/scenarios"

// fixtureBase is the repository-relative fixture directory base.
const fixtureBase = "evaluations/fixtures"

// LoadAllScenarios loads every scenario under root/evaluations/scenarios in
// deterministic order. A missing scenarios directory is an empty corpus, not
// an error, so the corpus check reports the emptiness as a finding.
func LoadAllScenarios(root string) ([]*Scenario, error) {
	base := filepath.Join(root, scenarioBase)
	if info, err := os.Stat(base); err != nil || !info.IsDir() {
		return nil, nil
	}
	var scenarios []*Scenario
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		sc, err := LoadScenario(path)
		if err != nil {
			return err
		}
		scenarios = append(scenarios, sc)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(scenarios, func(i, j int) bool { return scenarios[i].ID < scenarios[j].ID })
	return scenarios, nil
}

// ScenarioByID returns the scenario with the given id from a loaded corpus.
func ScenarioByID(scenarios []*Scenario, id string) (*Scenario, error) {
	for _, sc := range scenarios {
		if sc.ID == id {
			return sc, nil
		}
	}
	return nil, errors.New("no scenario with id")
}
