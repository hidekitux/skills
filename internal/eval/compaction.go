package eval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PairResult records the comparison of one scenario and host across two
// instruction sources.
type PairResult struct {
	Scenario              string         `json:"scenario"`
	Skill                 string         `json:"skill"`
	Host                  string         `json:"host"`
	FullVerdict           string         `json:"full_verdict"`
	CompactVerdict        string         `json:"compact_verdict"`
	Status                string         `json:"status"`
	Reasons               []string       `json:"reasons,omitempty"`
	FullScores            map[string]int `json:"full_scores,omitempty"`
	CompactScores         map[string]int `json:"compact_scores,omitempty"`
	FullContextTokens     int            `json:"full_context_tokens,omitempty"`
	CompactContextTokens  int            `json:"compact_context_tokens,omitempty"`
	ContextTokenReduction int            `json:"context_token_reduction,omitempty"`
}

// PairReport is the machine-readable comparison artifact for Issue #197.
type PairReport struct {
	Encoding string       `json:"encoding"`
	Full     string       `json:"full_report"`
	Compact  string       `json:"compact_report"`
	Results  []PairResult `json:"results"`
}

// LoadRecords reads one JSONL evaluation report.
func LoadRecords(path string) ([]Record, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	var records []Record
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record Record
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			return nil, fmt.Errorf("decode %s: %w", path, err)
		}
		records = append(records, record)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

// LatestReport returns the newest JSONL report in a directory.
func LatestReport(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	var paths []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".jsonl") {
			paths = append(paths, filepath.Join(dir, entry.Name()))
		}
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("no JSONL evaluation report in %s", dir)
	}
	sort.Strings(paths)
	return paths[len(paths)-1], nil
}

// CompareReports applies the Issue #173 deterministic and rubric thresholds
// to one full and one compact report for the same scenario set.
func CompareReports(fullPath, compactPath string) (PairReport, error) {
	full, err := LoadRecords(fullPath)
	if err != nil {
		return PairReport{}, err
	}
	compact, err := LoadRecords(compactPath)
	if err != nil {
		return PairReport{}, err
	}
	fullByKey := recordsByKey(full)
	compactByKey := recordsByKey(compact)
	if len(fullByKey) != len(compactByKey) {
		return PairReport{}, fmt.Errorf("paired reports contain different record counts: full=%d compact=%d", len(fullByKey), len(compactByKey))
	}
	keys := make([]string, 0, len(fullByKey))
	for key := range fullByKey {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	report := PairReport{Encoding: "cl100k_base", Full: fullPath, Compact: compactPath}
	for _, key := range keys {
		left := fullByKey[key]
		right, ok := compactByKey[key]
		if !ok {
			return PairReport{}, fmt.Errorf("compact report is missing %s", key)
		}
		result := PairResult{
			Scenario:             left.Scenario,
			Skill:                left.Skill,
			Host:                 left.Host,
			FullVerdict:          left.Verdict,
			CompactVerdict:       right.Verdict,
			FullScores:           left.RubricScores,
			CompactScores:        right.RubricScores,
			FullContextTokens:    contextTokens(left),
			CompactContextTokens: contextTokens(right),
		}
		result.ContextTokenReduction = result.FullContextTokens - result.CompactContextTokens
		if left.PromptSHA != right.PromptSHA {
			result.Reasons = append(result.Reasons, "prompt hash changed")
		}
		if left.Verdict == VerdictInterrupted {
			result.Reasons = append(result.Reasons, "full source was interrupted")
		}
		if right.Verdict == VerdictInterrupted {
			result.Reasons = append(result.Reasons, "compact source was interrupted")
		}
		switch {
		case left.Verdict == VerdictPass && right.Verdict == VerdictFail:
			result.Reasons = append(result.Reasons, "compact source lost a deterministic pass")
		case left.Verdict == VerdictInfra || right.Verdict == VerdictInfra:
			result.Reasons = append(result.Reasons, "infrastructure evidence is incomplete")
		case left.Verdict == VerdictSkipped || right.Verdict == VerdictSkipped:
			result.Reasons = append(result.Reasons, "host or scenario was skipped")
		}
		switch {
		case left.RubricReview == RubricComplete && right.RubricReview == RubricComplete:
			result.Reasons = append(result.Reasons, compareScores(left.RubricScores, right.RubricScores)...)
		case left.RubricReview == RubricNA && right.RubricReview == RubricNA:
			// Deterministic evidence is sufficient when both runs explicitly opt
			// out of rubric scoring.
		default:
			result.Reasons = append(result.Reasons, "rubric evidence pending")
		}
		switch {
		case containsReason(result.Reasons, "source was interrupted"):
			result.Status = "inconclusive"
		case containsReason(result.Reasons, "compact source lost a deterministic pass"):
			result.Status = "fail"
		case len(result.Reasons) == 0:
			result.Status = "pass"
		case containsReason(result.Reasons, "infrastructure"), containsReason(result.Reasons, "skipped"), containsReason(result.Reasons, "pending"):
			result.Status = "inconclusive"
		default:
			result.Status = "fail"
		}
		report.Results = append(report.Results, result)
	}
	return report, nil
}

func contextTokens(record Record) int {
	if record.Context == nil {
		return 0
	}
	return record.Context.TotalTokens
}

func recordsByKey(records []Record) map[string]Record {
	result := make(map[string]Record, len(records))
	for _, record := range records {
		result[record.Scenario+"\x00"+record.Host] = record
	}
	return result
}

func compareScores(full, compact map[string]int) []string {
	var reasons []string
	fullMean, compactMean := scoreMean(full), scoreMean(compact)
	if compactMean < fullMean-1.0 {
		reasons = append(reasons, fmt.Sprintf("compact rubric mean %.2f is more than 1.0 below full mean %.2f", compactMean, fullMean))
	}
	if compactMean < 4.0 {
		reasons = append(reasons, fmt.Sprintf("compact rubric mean %.2f is below 4.0", compactMean))
	}
	for _, dimension := range scoreOrder {
		left, leftOK := full[dimension]
		right, rightOK := compact[dimension]
		if !leftOK || !rightOK {
			reasons = append(reasons, "rubric dimension is missing")
			continue
		}
		if right < 3 {
			reasons = append(reasons, fmt.Sprintf("compact %s score %d is below 3", dimension, right))
		}
		if difference(left, right) > 1 {
			reasons = append(reasons, fmt.Sprintf("%s score variance exceeds 1", dimension))
		}
	}
	return reasons
}

func scoreMean(scores map[string]int) float64 {
	if len(scoreOrder) == 0 {
		return 0
	}
	total := 0
	for _, dimension := range scoreOrder {
		total += scores[dimension]
	}
	return float64(total) / float64(len(scoreOrder))
}

func difference(left, right int) int {
	if left > right {
		return left - right
	}
	return right - left
}

func containsReason(reasons []string, fragment string) bool {
	for _, reason := range reasons {
		if strings.Contains(reason, fragment) {
			return true
		}
	}
	return false
}

// WritePairReport writes both JSON and Markdown evidence for a comparison.
func WritePairReport(dir string, report PairReport, out io.Writer) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	jsonPath := filepath.Join(dir, "comparison.json")
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(jsonPath, append(data, '\n'), 0o644); err != nil {
		return err
	}
	markdownPath := filepath.Join(dir, "comparison.md")
	file, err := os.Create(markdownPath)
	if err != nil {
		return err
	}
	defer file.Close()
	fmt.Fprintln(file, "# Instruction compaction comparison")
	fmt.Fprintln(file)
	fmt.Fprintf(file, "Encoding: `%s`.\n\n", report.Encoding)
	fmt.Fprintln(file, "| scenario | skill | host | full | compact | status | reason |")
	fmt.Fprintln(file, "| --- | --- | --- | --- | --- | --- | --- |")
	for _, result := range report.Results {
		fmt.Fprintf(file, "| %s | %s | %s | %s | %s | %s | %s |\n", result.Scenario, result.Skill, result.Host, result.FullVerdict, result.CompactVerdict, result.Status, strings.Join(result.Reasons, "; "))
	}
	fmt.Fprintf(out, "pair report written to %s and %s\n", jsonPath, markdownPath)
	return nil
}
