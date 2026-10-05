package eval

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// retainedFields lists the only fields a retained promotion record may carry.
// They are the fields the promotion checks read; failures, transcripts,
// context, and comparison detail stay in the local evaluations/reports/.
var retainedFields = []string{
	"run_id", "scenario", "skill", "kind", "host", "model",
	"repo_commit", "skill_source_commit", "input_digest", "prompt_sha256",
	"verdict", "rubric_review", "rubric_scores", "started_at", "finished_at",
}

// requiredRetainedFields lists the fields validatePromotionRun reads, so a
// retained record without one of them can never count as evidence.
var requiredRetainedFields = []string{
	"run_id", "scenario", "skill", "host", "input_digest", "prompt_sha256",
	"verdict", "rubric_review", "finished_at",
}

// evidenceDirs returns the directories that hold promotion evidence: the
// ignored local reports and the tracked retained evidence.
func evidenceDirs(root string) []string {
	return []string{
		filepath.Join(root, "evaluations", "reports"),
		filepath.Join(root, "evaluations", "evidence"),
	}
}

// retainedRecord reduces a report record to the retained fields, dropping
// empty values.
func retainedRecord(record Record) (map[string]any, error) {
	content, err := json.Marshal(record)
	if err != nil {
		return nil, err
	}
	var full map[string]any
	if err := json.Unmarshal(content, &full); err != nil {
		return nil, err
	}
	kept := make(map[string]any, len(retainedFields))
	for _, field := range retainedFields {
		value, ok := full[field]
		if !ok || value == nil || value == "" {
			continue
		}
		kept[field] = value
	}
	return kept, nil
}

// writeRetainedEvidence writes the retained records of one run to
// evaluations/evidence/<skill>/<runID>.jsonl, one file per evaluated skill.
func writeRetainedEvidence(root, runID string, records []Record) error {
	bySkill := map[string][]map[string]any{}
	for _, record := range records {
		if record.Skill == "" {
			continue
		}
		kept, err := retainedRecord(record)
		if err != nil {
			return err
		}
		bySkill[record.Skill] = append(bySkill[record.Skill], kept)
	}
	skills := make([]string, 0, len(bySkill))
	for skill := range bySkill {
		skills = append(skills, skill)
	}
	sort.Strings(skills)
	for _, skill := range skills {
		dir := filepath.Join(root, "evaluations", "evidence", skill)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		var buffer bytes.Buffer
		encoder := json.NewEncoder(&buffer)
		for _, kept := range bySkill[skill] {
			if err := encoder.Encode(kept); err != nil {
				return err
			}
		}
		if err := os.WriteFile(filepath.Join(dir, runID+".jsonl"), buffer.Bytes(), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// checkRetainedEvidence rejects a retained evidence file that is not JSONL,
// a malformed line, a field outside retainedFields, a missing required field,
// and a record whose skill differs from its directory under
// evaluations/evidence/.
func checkRetainedEvidence(root string, findings *[]string) {
	evidenceRoot := filepath.Join(root, "evaluations", "evidence")
	allowed := make(map[string]bool, len(retainedFields))
	for _, field := range retainedFields {
		allowed[field] = true
	}
	err := filepath.WalkDir(evidenceRoot, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			relative = path
		}
		relative = filepath.ToSlash(relative)
		if filepath.Ext(path) != ".jsonl" {
			*findings = append(*findings, fmt.Sprintf("retained evidence %s is not a .jsonl file", relative))
			return nil
		}
		parts := strings.Split(strings.TrimPrefix(relative, "evaluations/evidence/"), "/")
		if len(parts) != 2 {
			*findings = append(*findings, fmt.Sprintf("retained evidence %s is not at evaluations/evidence/<skill>/<run_id>.jsonl", relative))
			return nil
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			*findings = append(*findings, fmt.Sprintf("retained evidence %s cannot be read: %v", relative, readErr))
			return nil
		}
		for index, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			var record map[string]any
			if err := json.Unmarshal([]byte(line), &record); err != nil {
				*findings = append(*findings, fmt.Sprintf("retained evidence %s line %d is not a JSON object", relative, index+1))
				continue
			}
			var unknown []string
			for field := range record {
				if !allowed[field] {
					unknown = append(unknown, field)
				}
			}
			sort.Strings(unknown)
			for _, field := range unknown {
				*findings = append(*findings, fmt.Sprintf("retained evidence %s line %d carries field %q outside the retained set", relative, index+1, field))
			}
			for _, field := range requiredRetainedFields {
				if value, ok := record[field]; !ok || value == nil || value == "" {
					*findings = append(*findings, fmt.Sprintf("retained evidence %s line %d is missing required field %q", relative, index+1, field))
				}
			}
			if skill, _ := record["skill"].(string); skill != "" && skill != parts[0] {
				*findings = append(*findings, fmt.Sprintf("retained evidence %s line %d records skill %q under directory %q", relative, index+1, skill, parts[0]))
			}
			if runID, _ := record["run_id"].(string); runID != "" && runID+".jsonl" != parts[1] {
				*findings = append(*findings, fmt.Sprintf("retained evidence %s line %d records run %q in file %q", relative, index+1, runID, parts[1]))
			}
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		*findings = append(*findings, fmt.Sprintf("cannot read retained evidence: %v", err))
	}
}
