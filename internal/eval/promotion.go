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
	"time"
)

const (
	minimumPromotionScore = 3
	minimumPromotionMean  = 4.0
)

type promotionRun struct {
	id       string
	records  []Record
	latestAt time.Time
}

// PromotionFindings checks the complete evidence contract for every stable
// catalog entry. It runs at release verification time, after the repository
// has a committed revision to compare with the retained reports.
func PromotionFindings(root string) []string {
	catalog, err := loadCatalogSkills(root)
	if err != nil {
		return []string{fmt.Sprintf("cannot read catalog for promotion: %v", err)}
	}
	stable := make([]string, 0)
	for _, skill := range catalog {
		if skill.Status == "stable" {
			stable = append(stable, skill.Name)
		}
	}
	if len(stable) == 0 {
		return nil
	}

	revision := repoCommit(root)
	if !validRevision(revision) {
		return []string{fmt.Sprintf("stable promotion requires a committed repository revision, got %q", revision)}
	}
	scenarios, err := LoadAllScenarios(root)
	if err != nil {
		return []string{fmt.Sprintf("cannot load evaluation scenarios for promotion: %v", err)}
	}

	reports, reportFindings := loadPromotionReports(root)
	findings := append([]string{}, reportFindings...)
	sort.Strings(stable)
	for _, skill := range stable {
		findings = append(findings, checkStableSkillPromotion(skill, revision, scenarios, reports)...)
	}
	sort.Strings(findings)
	return findings
}

// CheckPromotion is the command-facing wrapper for PromotionFindings.
func CheckPromotion(root string, out, errOut io.Writer) int {
	findings := PromotionFindings(root)
	if len(findings) > 0 {
		fmt.Fprintln(errOut, "Stable promotion verification failed:")
		for _, finding := range findings {
			fmt.Fprintf(errOut, "- %s\n", finding)
		}
		return 1
	}
	catalog, err := loadCatalogSkills(root)
	if err != nil {
		fmt.Fprintf(errOut, "Stable promotion verification failed: %v\n", err)
		return 1
	}
	stable := 0
	for _, skill := range catalog {
		if skill.Status == "stable" {
			stable++
		}
	}
	fmt.Fprintf(out, "Stable promotion verification passed: %d stable skill(s).\n", stable)
	return 0
}

func loadPromotionReports(root string) (map[string][]Record, []string) {
	runs := map[string][]Record{}
	reportsDir := filepath.Join(root, "evaluations", "reports")
	var findings []string
	err := filepath.WalkDir(reportsDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || (filepath.Ext(entry.Name()) != ".jsonl" && filepath.Ext(entry.Name()) != ".json") {
			return nil
		}
		records, err := decodePromotionReport(path)
		if err != nil {
			relativePath, relativeErr := filepath.Rel(root, path)
			if relativeErr != nil {
				relativePath = path
			}
			findings = append(findings, fmt.Sprintf("evaluation report %s is invalid: %v", filepath.ToSlash(relativePath), err))
			return nil
		}
		for _, record := range records {
			if record.Skill != "" {
				runs[record.Skill] = append(runs[record.Skill], record)
			}
		}
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return runs, findings
		}
		findings = append(findings, fmt.Sprintf("cannot read evaluation reports: %v", err))
	}
	return runs, findings
}

func decodePromotionReport(path string) ([]Record, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if filepath.Ext(path) == ".jsonl" {
		var records []Record
		for lineNumber, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var record Record
			if err := json.Unmarshal([]byte(line), &record); err != nil {
				return nil, fmt.Errorf("line %d: %w", lineNumber+1, err)
			}
			records = append(records, record)
		}
		return records, nil
	}
	var records []Record
	if err := json.Unmarshal(content, &records); err == nil {
		return records, nil
	}
	var record Record
	if err := json.Unmarshal(content, &record); err != nil {
		return nil, err
	}
	return []Record{record}, nil
}

func checkStableSkillPromotion(skill, revision string, scenarios []*Scenario, recordsBySkill map[string][]Record) []string {
	required := map[string]bool{}
	for _, scenario := range scenarios {
		if scenario.Skill != skill || (scenario.Kind != KindPositive && scenario.Kind != KindNegative && scenario.Kind != KindBoundary) {
			continue
		}
		required[scenario.ID] = true
	}
	if len(required) == 0 {
		return []string{fmt.Sprintf("stable skill %q has no direct positive, negative, or boundary scenarios", skill)}
	}

	runsByID := map[string]*promotionRun{}
	for _, record := range recordsBySkill[skill] {
		if record.RunID == "" {
			continue
		}
		run := runsByID[record.RunID]
		if run == nil {
			run = &promotionRun{id: record.RunID}
			runsByID[record.RunID] = run
		}
		run.records = append(run.records, record)
		finishedAt, err := recordFinishedAt(record)
		if err == nil && finishedAt.After(run.latestAt) {
			run.latestAt = finishedAt
		}
	}
	if len(runsByID) < 2 {
		return []string{fmt.Sprintf("stable skill %q needs two complete evaluation runs at revision %s; found %d run(s)", skill, revision, len(runsByID))}
	}

	runs := make([]*promotionRun, 0, len(runsByID))
	for _, run := range runsByID {
		runs = append(runs, run)
	}
	sort.Slice(runs, func(i, j int) bool {
		if runs[i].latestAt.Equal(runs[j].latestAt) {
			return runs[i].id > runs[j].id
		}
		return runs[i].latestAt.After(runs[j].latestAt)
	})
	latest := runs[:2]
	findings := []string{}
	validated := make([]map[string]Record, 0, len(latest))
	for _, run := range latest {
		valid, runFindings := validatePromotionRun(skill, revision, required, run)
		findings = append(findings, runFindings...)
		validated = append(validated, valid)
	}
	if len(findings) > 0 {
		return findings
	}

	first, second := validated[1], validated[0]
	if len(first) != len(second) {
		return []string{fmt.Sprintf("stable skill %q has different evaluated scenario/host sets across its two most recent runs", skill)}
	}
	for key, older := range first {
		newer, ok := second[key]
		if !ok {
			findings = append(findings, fmt.Sprintf("stable skill %q is missing %s in its older or newer comparison run", skill, key))
			continue
		}
		if older.PromptSHA == "" || older.PromptSHA != newer.PromptSHA {
			findings = append(findings, fmt.Sprintf("stable skill %q changed the prompt for %s between its two most recent runs", skill, key))
		}
		for _, dimension := range scoreOrder {
			if scoreDifference(older.RubricScores[dimension], newer.RubricScores[dimension]) > 1 {
				findings = append(findings, fmt.Sprintf("stable skill %q exceeds bounded rubric variance for %s dimension %q", skill, key, dimension))
			}
		}
	}
	return findings
}

func validatePromotionRun(skill, revision string, required map[string]bool, run *promotionRun) (map[string]Record, []string) {
	validated := map[string]Record{}
	findings := []string{}
	seenScenarios := map[string]bool{}
	for _, record := range run.records {
		key := record.Scenario + "@" + record.Host
		if _, exists := validated[key]; exists {
			findings = append(findings, fmt.Sprintf("stable skill %q has duplicate scenario/host evidence %s in run %s", skill, key, run.id))
			continue
		}
		validated[key] = record
		seenScenarios[record.Scenario] = true
		if record.Commit != revision || record.SkillSourceCommit != revision {
			findings = append(findings, fmt.Sprintf("stable skill %q has stale or incomplete revision evidence in run %s for %s", skill, run.id, key))
		}
		if record.Skill != skill || record.Host == "" || record.PromptSHA == "" {
			findings = append(findings, fmt.Sprintf("stable skill %q has incomplete identity evidence in run %s for %s", skill, run.id, key))
		}
		if record.Verdict != VerdictPass {
			findings = append(findings, fmt.Sprintf("stable skill %q has %s evidence for %s in newer or comparison run %s", skill, record.Verdict, key, run.id))
		}
		if record.RubricReview != RubricComplete {
			findings = append(findings, fmt.Sprintf("stable skill %q has incomplete rubric review for %s in run %s", skill, key, run.id))
			continue
		}
		total := 0
		for _, dimension := range scoreOrder {
			score, ok := record.RubricScores[dimension]
			if !ok {
				findings = append(findings, fmt.Sprintf("stable skill %q is missing rubric score %q for %s in run %s", skill, dimension, key, run.id))
				continue
			}
			if score < minimumPromotionScore || score > 5 {
				findings = append(findings, fmt.Sprintf("stable skill %q has rubric score %d for %s dimension %q in run %s; want 3-5", skill, score, key, dimension, run.id))
			}
			total += score
		}
		if len(record.RubricScores) < len(scoreOrder) {
			continue
		}
		mean := float64(total) / float64(len(scoreOrder))
		if mean < minimumPromotionMean {
			findings = append(findings, fmt.Sprintf("stable skill %q has rubric mean %.2f for %s in run %s; want at least %.1f", skill, mean, key, run.id, minimumPromotionMean))
		}
		if _, err := recordFinishedAt(record); err != nil {
			findings = append(findings, fmt.Sprintf("stable skill %q has invalid completion time for %s in run %s", skill, key, run.id))
		}
	}
	for scenario := range required {
		if !seenScenarios[scenario] {
			findings = append(findings, fmt.Sprintf("stable skill %q is missing required scenario %q in run %s", skill, scenario, run.id))
		}
	}
	return validated, findings
}

func recordFinishedAt(record Record) (time.Time, error) {
	if record.FinishedAt == "" {
		return time.Time{}, fmt.Errorf("finished_at is required")
	}
	return time.Parse(time.RFC3339Nano, record.FinishedAt)
}

func validRevision(revision string) bool {
	if len(revision) != 40 {
		return false
	}
	for _, char := range revision {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func scoreDifference(left, right int) int {
	if left > right {
		return left - right
	}
	return right - left
}
