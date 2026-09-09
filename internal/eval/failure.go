package eval

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const failureRecordDir = "workflow/failure-records"

var (
	failureIdentifierRE = regexp.MustCompile(`^[a-z0-9][a-z0-9._/-]*$`)
	failureRevisionRE   = regexp.MustCompile(`^[0-9a-f]{40}$`)
)

// failureRecord is the semantic subset of workflow/failure-record.schema.json
// needed by the repository check. The decoder rejects unknown fields so the
// checked JSONL stays aligned with the versioned contract.
type failureRecord struct {
	SchemaVersion     int                  `json:"schema_version"`
	ID                string               `json:"id"`
	Classification    string               `json:"classification"`
	Status            string               `json:"status"`
	Summary           string               `json:"summary"`
	Reproduction      failureReproduction  `json:"reproduction"`
	ExpectedOutcome   string               `json:"expected_outcome"`
	Owner             failureOwner         `json:"owner"`
	Evidence          []failureEvidence    `json:"evidence"`
	RegressionAsset   string               `json:"regression_asset"`
	InstructionAction string               `json:"instruction_action"`
	InstructionRef    string               `json:"instruction_ref"`
	DecisionReason    string               `json:"decision_reason"`
	Observations      []failureObservation `json:"observations"`
	RecurrenceCount   int                  `json:"recurrence_count"`
}

type failureReproduction struct {
	Kind      string   `json:"kind"`
	Fixture   string   `json:"fixture"`
	Steps     []string `json:"steps"`
	Sanitized bool     `json:"sanitized"`
}

type failureOwner struct {
	Layer string `json:"layer"`
	Name  string `json:"name"`
}

type failureEvidence struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

type failureObservation struct {
	Host               string `json:"host"`
	Model              string `json:"model"`
	Skill              string `json:"skill"`
	RepositoryRevision string `json:"repository_revision"`
	RunID              string `json:"run_id"`
	Outcome            string `json:"outcome"`
}

var validFailureClassifications = map[string]bool{
	"instruction":       true,
	"context_selection": true,
	"tool_use":          true,
	"model_behavior":    true,
	"repository_design": true,
	"external_state":    true,
	"infrastructure":    true,
}

var validFailureStatuses = map[string]bool{
	"candidate":    true,
	"promoted":     true,
	"not_promoted": true,
	"retired":      true,
}

var validFailureKinds = map[string]bool{
	"evaluation_fixture": true,
	"test_fixture":       true,
	"command":            true,
	"trace":              true,
}

var validFailureLayers = map[string]bool{
	"evaluation_fixture": true,
	"static_check":       true,
	"type_constraint":    true,
	"test":               true,
	"property_test":      true,
	"fsl":                true,
	"runtime_isolation":  true,
	"guidance":           true,
	"none":               true,
}

var validFailureEvidenceKinds = map[string]bool{
	"path":         true,
	"command":      true,
	"issue":        true,
	"pull_request": true,
	"commit":       true,
	"evaluation":   true,
	"trace":        true,
}

var validFailureOutcomes = map[string]bool{
	"deterministic_failure": true,
	"behavioral_failure":    true,
	"skipped":               true,
	"infrastructure_error":  true,
	"resolved":              true,
}

var validInstructionActions = map[string]bool{
	"replace":  true,
	"compress": true,
	"retain":   true,
	"none":     true,
}

// checkFailureRecords validates the durable failure corpus. An absent corpus
// is allowed for repositories that have not adopted the contract yet; once a
// JSONL record exists, the schema and every record are required.
func checkFailureRecords(root string, findings *[]string) {
	dir := filepath.Join(root, failureRecordDir)
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return
	}
	if err != nil {
		*findings = append(*findings, fmt.Sprintf("failure records: cannot read %s: %v", failureRecordDir, err))
		return
	}

	schemaPath := filepath.Join(root, "workflow", "failure-record.schema.json")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		*findings = append(*findings, fmt.Sprintf("failure records: schema is unreadable: %v", err))
	} else if !json.Valid(schema) {
		*findings = append(*findings, "failure records: workflow/failure-record.schema.json is not valid JSON")
	}

	seen := map[string]bool{}
	found := false
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".jsonl" {
			continue
		}
		found = true
		validateFailureFile(root, filepath.Join(dir, entry.Name()), seen, findings)
	}
	if !found {
		*findings = append(*findings, fmt.Sprintf("failure records: no JSONL records found under %s", failureRecordDir))
	}
}

func validateFailureFile(root, path string, seen map[string]bool, findings *[]string) {
	f, err := os.Open(path)
	if err != nil {
		*findings = append(*findings, fmt.Sprintf("failure records: cannot open %s: %v", filepath.Base(path), err))
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	line := 0
	for scanner.Scan() {
		line++
		raw := scanner.Bytes()
		if strings.TrimSpace(string(raw)) == "" {
			continue
		}
		if containsFailureSecret(string(raw)) {
			*findings = append(*findings, fmt.Sprintf("failure records: %s:%d contains credential-like or private content", filepath.Base(path), line))
		}
		var record failureRecord
		decoder := json.NewDecoder(strings.NewReader(string(raw)))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&record); err != nil {
			*findings = append(*findings, fmt.Sprintf("failure records: %s:%d is invalid JSONL: %v", filepath.Base(path), line, err))
			continue
		}
		validateFailureRecord(root, filepath.Base(path), line, &record, seen, findings)
	}
	if err := scanner.Err(); err != nil {
		*findings = append(*findings, fmt.Sprintf("failure records: %s cannot be scanned: %v", filepath.Base(path), err))
	}
}

func validateFailureRecord(root, file string, line int, record *failureRecord, seen map[string]bool, findings *[]string) {
	where := fmt.Sprintf("failure records: %s:%d", file, line)
	if record.SchemaVersion != 1 {
		*findings = append(*findings, fmt.Sprintf("%s schema_version must be 1", where))
	}
	if !failureIdentifierRE.MatchString(record.ID) {
		*findings = append(*findings, fmt.Sprintf("%s id %q is invalid", where, record.ID))
	}
	if seen[record.ID] {
		*findings = append(*findings, fmt.Sprintf("%s id %q is duplicated", where, record.ID))
	}
	seen[record.ID] = true
	if !validFailureClassifications[record.Classification] {
		*findings = append(*findings, fmt.Sprintf("%s classification %q is invalid", where, record.Classification))
	}
	if !validFailureStatuses[record.Status] {
		*findings = append(*findings, fmt.Sprintf("%s status %q is invalid", where, record.Status))
	}
	if record.Summary == "" || record.ExpectedOutcome == "" {
		*findings = append(*findings, fmt.Sprintf("%s summary and expected_outcome are required", where))
	}
	if !validFailureKinds[record.Reproduction.Kind] || record.Reproduction.Fixture == "" || len(record.Reproduction.Steps) == 0 || !record.Reproduction.Sanitized {
		*findings = append(*findings, fmt.Sprintf("%s reproduction must name a sanitized fixture and at least one step", where))
	} else if !relativePath(record.Reproduction.Fixture) || !pathExists(root, record.Reproduction.Fixture) {
		*findings = append(*findings, fmt.Sprintf("%s reproduction fixture %q does not resolve under the repository", where, record.Reproduction.Fixture))
	}
	if !validFailureLayers[record.Owner.Layer] || record.Owner.Name == "" {
		*findings = append(*findings, fmt.Sprintf("%s owner must name a valid enforcement layer and owner", where))
	}
	if len(record.Evidence) == 0 {
		*findings = append(*findings, fmt.Sprintf("%s evidence must not be empty", where))
	}
	for _, evidence := range record.Evidence {
		if !validFailureEvidenceKinds[evidence.Kind] || evidence.Ref == "" {
			*findings = append(*findings, fmt.Sprintf("%s evidence entries need a valid kind and ref", where))
		}
	}
	if !validInstructionActions[record.InstructionAction] {
		*findings = append(*findings, fmt.Sprintf("%s instruction_action %q is invalid", where, record.InstructionAction))
	}
	if record.InstructionAction != "none" && record.InstructionRef == "" {
		*findings = append(*findings, fmt.Sprintf("%s instruction_ref is required when instruction_action is %q", where, record.InstructionAction))
	}
	if record.InstructionRef != "" {
		instructionPath := strings.SplitN(record.InstructionRef, "#", 2)[0]
		if !relativePath(instructionPath) || !pathExists(root, instructionPath) {
			*findings = append(*findings, fmt.Sprintf("%s instruction_ref %q does not resolve under the repository", where, record.InstructionRef))
		}
	}
	if len(record.Observations) == 0 || record.RecurrenceCount != len(record.Observations) {
		*findings = append(*findings, fmt.Sprintf("%s recurrence_count must equal the non-empty observations count", where))
	}
	for _, observation := range record.Observations {
		if !failureIdentifierRE.MatchString(observation.Host) || !failureIdentifierRE.MatchString(observation.Model) ||
			!failureIdentifierRE.MatchString(observation.Skill) || !failureIdentifierRE.MatchString(observation.RunID) ||
			!failureRevisionRE.MatchString(observation.RepositoryRevision) || !validFailureOutcomes[observation.Outcome] {
			*findings = append(*findings, fmt.Sprintf("%s observation has invalid provenance", where))
		}
	}

	if record.Status == "promoted" {
		if record.Classification == "infrastructure" || record.Classification == "external_state" {
			*findings = append(*findings, fmt.Sprintf("%s infrastructure and external-state records cannot be promoted", where))
		}
		if record.Owner.Layer == "none" || record.RegressionAsset == "" || record.DecisionReason == "" {
			*findings = append(*findings, fmt.Sprintf("%s promoted records need an owner, regression_asset, and decision_reason", where))
		}
		if len(record.Observations) < 2 {
			*findings = append(*findings, fmt.Sprintf("%s promoted records need at least two observations", where))
		}
		if record.Owner.Layer != "evaluation_fixture" && record.Owner.Layer != "guidance" && record.InstructionAction == "none" {
			*findings = append(*findings, fmt.Sprintf("%s mechanical owners need an instruction_action", where))
		}
		if !validRegressionAsset(root, record.RegressionAsset) {
			*findings = append(*findings, fmt.Sprintf("%s regression_asset %q does not resolve", where, record.RegressionAsset))
		}
	} else if record.Status == "not_promoted" {
		if record.Owner.Layer != "none" || record.DecisionReason == "" {
			*findings = append(*findings, fmt.Sprintf("%s not_promoted records need owner layer none and decision_reason", where))
		}
	}
}

func pathExists(root, rel string) bool {
	if !relativePath(rel) {
		return false
	}
	_, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel)))
	return err == nil
}

func relativePath(value string) bool {
	return value != "" && !filepath.IsAbs(value) && filepath.Clean(value) == value && value != "." && !strings.HasPrefix(value, "../") && !strings.Contains(value, "/../")
}

func validRegressionAsset(root, asset string) bool {
	switch {
	case strings.HasPrefix(asset, "scenario:"):
		return pathExists(root, filepath.Join("evaluations", "scenarios", strings.TrimPrefix(asset, "scenario:")))
	case strings.HasPrefix(asset, "test:"):
		value := strings.TrimPrefix(asset, "test:")
		if pathExists(root, value) {
			return true
		}
		return pathExists(root, filepath.Dir(value))
	case strings.HasPrefix(asset, "fixture:"):
		return pathExists(root, strings.TrimPrefix(asset, "fixture:"))
	default:
		return false
	}
}

func containsFailureSecret(raw string) bool {
	for _, marker := range []string{"ghp_", "github_pat_", "sk-", "BEGIN PRIVATE KEY", "/Users/", "password=", "token="} {
		if strings.Contains(raw, marker) {
			return true
		}
	}
	return false
}

// CheckFailureRecords is exposed for focused tests while the aggregate corpus
// check remains the normal repository entry point.
func CheckFailureRecords(root string, out, errOut io.Writer) int {
	var findings []string
	checkFailureRecords(root, &findings)
	if len(findings) > 0 {
		sort.Strings(findings)
		for _, finding := range findings {
			fmt.Fprintln(errOut, finding)
		}
		return 1
	}
	fmt.Fprintln(out, "Failure record check passed.")
	return 0
}
