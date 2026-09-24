package check

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// cutoverParticipant is one row of the readiness and recovery matrix in
// workflow/cutover-record.yml.
type cutoverParticipant struct {
	ID             string   `yaml:"id"`
	Kind           string   `yaml:"kind"`
	Surface        string   `yaml:"surface"`
	Owner          string   `yaml:"owner"`
	Contracts      []string `yaml:"contracts"`
	Evidence       []string `yaml:"evidence"`
	Readiness      string   `yaml:"readiness"`
	ReadinessCheck string   `yaml:"readiness_check"`
	Recovery       string   `yaml:"recovery"`
}

// cutoverSlice is one Sub-issue of the migration order.
type cutoverSlice struct {
	Issue     int    `yaml:"issue"`
	Delivers  string `yaml:"delivers"`
	DependsOn []int  `yaml:"depends_on"`
}

// cutoverTrigger is one condition that starts the recovery procedure.
type cutoverTrigger struct {
	ID        string `yaml:"id"`
	Condition string `yaml:"condition"`
	Action    string `yaml:"action"`
}

// cutoverStep is one step of the recovery procedure.
type cutoverStep struct {
	ID          string `yaml:"id"`
	Action      string `yaml:"action"`
	Observation string `yaml:"observation"`
}

// cutoverRehearsal is one command the recovery rehearsal ran.
type cutoverRehearsal struct {
	Command     string `yaml:"command"`
	Observation string `yaml:"observation"`
}

// cutoverRecovery is the tested recovery procedure and the conditions that
// start it.
type cutoverRecovery struct {
	SafeState string             `yaml:"safe_state"`
	Method    string             `yaml:"method"`
	Triggers  []cutoverTrigger   `yaml:"triggers"`
	Steps     []cutoverStep      `yaml:"steps"`
	Rehearsal []cutoverRehearsal `yaml:"rehearsal"`
}

// cutoverCheck is one post-cutover observation.
type cutoverCheck struct {
	ID          string `yaml:"id"`
	Aspect      string `yaml:"aspect"`
	Command     string `yaml:"command"`
	Observation string `yaml:"observation"`
}

// cutoverRecord is workflow/cutover-record.yml.
type cutoverRecord struct {
	SchemaVersion int                  `yaml:"schema_version"`
	Baseline      string               `yaml:"baseline"`
	CutoverPoint  string               `yaml:"cutover_point"`
	Order         []cutoverSlice       `yaml:"order"`
	Participants  []cutoverParticipant `yaml:"participants"`
	Recovery      cutoverRecovery      `yaml:"recovery"`
	PostCutover   []cutoverCheck       `yaml:"post_cutover"`
}

// commitRE is the shape of the baseline and cutover point commit identifiers.
var commitRE = regexp.MustCompile(`^[0-9a-f]{40}$`)

// participantKinds are the kinds a matrix row may declare.
var participantKinds = map[string]bool{
	"module":   true,
	"command":  true,
	"workflow": true,
	"schema":   true,
	"fixture":  true,
	"consumer": true,
	"document": true,
}

// requiredAspects are the post-cutover aspects Issue #332 requires. Each one
// must be covered by at least one recorded check.
var requiredAspects = []string{
	"public-behavior",
	"privacy",
	"authority",
	"provider-failure",
	"terminal-outcome",
	"deterministic-replay",
	"compatibility",
}

// readCutoverRecord parses workflow/cutover-record.yml.
func readCutoverRecord(path string) (*cutoverRecord, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var record cutoverRecord
	// A misspelled key would otherwise be dropped in silence, and a row that
	// records nothing would read as one that records everything.
	decoder := yaml.NewDecoder(bytes.NewReader(content))
	decoder.KnownFields(true)
	if err := decoder.Decode(&record); err != nil {
		return nil, fmt.Errorf("cannot parse cutover-record.yml: %w", err)
	}
	if record.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported schema_version %d, want 1", record.SchemaVersion)
	}
	if len(record.Participants) == 0 {
		return nil, fmt.Errorf("cutover-record.yml lists no participant")
	}
	return &record, nil
}

// checkCutoverPoint returns the findings for the two commit identifiers.
func checkCutoverPoint(record *cutoverRecord, architecture string) []string {
	findings := []string{}
	for field, commit := range map[string]string{"baseline": record.Baseline, "cutover_point": record.CutoverPoint} {
		if !commitRE.MatchString(commit) {
			findings = append(findings, fmt.Sprintf("%s %q is not a full 40-character commit identifier", field, commit))
			continue
		}
		if !strings.Contains(architecture, commit) {
			findings = append(findings, fmt.Sprintf(
				"%s %s does not appear in docs/architecture.md; the record and the file must name the same commit", field, commit))
		}
	}
	if record.Baseline != "" && record.Baseline == record.CutoverPoint {
		findings = append(findings, "baseline and cutover_point name the same commit, so the record describes no cutover")
	}
	return findings
}

// checkCutoverOrder returns the findings for the migration order. A depends_on
// entry must name an Issue that appears earlier in the list, which is how the
// file states a dependency order rather than a set.
func checkCutoverOrder(order []cutoverSlice) []string {
	findings := []string{}
	if len(order) == 0 {
		findings = append(findings, "cutover-record.yml records no migration order")
		return findings
	}
	placed := map[int]bool{}
	for _, slice := range order {
		label := fmt.Sprintf("Issue #%d", slice.Issue)
		if slice.Issue <= 0 {
			findings = append(findings, "an order entry has no Issue number")
			continue
		}
		if placed[slice.Issue] {
			findings = append(findings, fmt.Sprintf("%s appears more than once in the migration order", label))
		}
		if strings.TrimSpace(slice.Delivers) == "" {
			findings = append(findings, fmt.Sprintf("%s states nothing it delivers", label))
		}
		for _, dependency := range slice.DependsOn {
			if !placed[dependency] {
				findings = append(findings, fmt.Sprintf(
					"%s depends on Issue #%d, which does not appear earlier in the migration order", label, dependency))
			}
		}
		placed[slice.Issue] = true
	}
	return findings
}

// checkCutoverParticipant returns the findings for one matrix row.
func checkCutoverParticipant(root string, index int, row cutoverParticipant,
	seen map[string]bool, owners map[string]bool, contracts map[string]bool, architecture string) []string {
	findings := []string{}
	label := row.ID
	if label == "" {
		label = fmt.Sprintf("participant %d", index+1)
		findings = append(findings, fmt.Sprintf("%s has no id", label))
	} else if !contractIDRE.MatchString(label) {
		findings = append(findings, fmt.Sprintf("participant %s has an id that is not lower-case words joined by hyphens", label))
	}
	if row.ID != "" && seen[row.ID] {
		findings = append(findings, fmt.Sprintf("participant %s is listed more than once", label))
	}
	seen[row.ID] = true
	if !participantKinds[row.Kind] {
		findings = append(findings, fmt.Sprintf("participant %s has kind %q; want one of %s",
			label, row.Kind, strings.Join(sortedKeys(participantKinds), ", ")))
	}
	required := map[string]string{
		"surface":         row.Surface,
		"readiness":       row.Readiness,
		"readiness_check": row.ReadinessCheck,
		"recovery":        row.Recovery,
	}
	for _, field := range []string{"surface", "readiness", "readiness_check", "recovery"} {
		if strings.TrimSpace(required[field]) == "" {
			findings = append(findings, fmt.Sprintf("participant %s records no %s", label, field))
		}
	}
	if !owners[row.Owner] {
		findings = append(findings, fmt.Sprintf(
			"participant %s names the owner %q, which is neither a module in workflow/module-ownership.yml nor composition", label, row.Owner))
	}
	if len(row.Evidence) == 0 {
		findings = append(findings, fmt.Sprintf("participant %s cites no evidence path", label))
	}
	for _, path := range row.Evidence {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			findings = append(findings, fmt.Sprintf("participant %s cites %s, which does not exist", label, path))
		}
	}
	for _, contract := range row.Contracts {
		if !contracts[contract] {
			findings = append(findings, fmt.Sprintf(
				"participant %s names the contract %s, which workflow/contract-decisions.yml does not record", label, contract))
		}
	}
	if row.ID != "" && !strings.Contains(architecture, row.ID) {
		findings = append(findings, fmt.Sprintf(
			"participant %s does not appear in docs/architecture.md; the record and the file must describe the same matrix", label))
	}
	return findings
}

// checkCutoverRecovery returns the findings for the recovery procedure.
func checkCutoverRecovery(recovery cutoverRecovery) []string {
	findings := []string{}
	if strings.TrimSpace(recovery.SafeState) == "" {
		findings = append(findings, "the recovery procedure names no safe state to return to")
	}
	if strings.TrimSpace(recovery.Method) == "" {
		findings = append(findings, "the recovery procedure names no method")
	}
	if len(recovery.Triggers) == 0 {
		findings = append(findings, "the recovery procedure records no trigger condition")
	}
	for index, trigger := range recovery.Triggers {
		label := trigger.ID
		if label == "" {
			label = fmt.Sprintf("trigger %d", index+1)
		}
		if strings.TrimSpace(trigger.Condition) == "" {
			findings = append(findings, fmt.Sprintf("recovery trigger %s states no condition", label))
		}
		if strings.TrimSpace(trigger.Action) == "" {
			findings = append(findings, fmt.Sprintf("recovery trigger %s states no action", label))
		}
	}
	if len(recovery.Steps) == 0 {
		findings = append(findings, "the recovery procedure records no step")
	}
	for index, step := range recovery.Steps {
		label := step.ID
		if label == "" {
			label = fmt.Sprintf("step %d", index+1)
		}
		if strings.TrimSpace(step.Action) == "" {
			findings = append(findings, fmt.Sprintf("recovery step %s states no action", label))
		}
		if strings.TrimSpace(step.Observation) == "" {
			findings = append(findings, fmt.Sprintf("recovery step %s states no observation, so running it proves nothing", label))
		}
	}
	if len(recovery.Rehearsal) == 0 {
		findings = append(findings, "the recovery procedure records no rehearsal, so it is untested")
	}
	for index, run := range recovery.Rehearsal {
		label := run.Command
		if label == "" {
			label = fmt.Sprintf("rehearsal entry %d", index+1)
			findings = append(findings, fmt.Sprintf("%s names no command", label))
		}
		if strings.TrimSpace(run.Observation) == "" {
			findings = append(findings, fmt.Sprintf("rehearsal entry %s records no observation", label))
		}
	}
	return findings
}

// checkPostCutover returns the findings for the post-cutover checks, including
// the aspects Issue #332 requires.
func checkPostCutover(checks []cutoverCheck) []string {
	findings := []string{}
	covered := map[string]bool{}
	seen := map[string]bool{}
	for index, entry := range checks {
		label := entry.ID
		if label == "" {
			label = fmt.Sprintf("post-cutover check %d", index+1)
			findings = append(findings, fmt.Sprintf("%s has no id", label))
		} else if seen[entry.ID] {
			findings = append(findings, fmt.Sprintf("post-cutover check %s is listed more than once", label))
		}
		seen[entry.ID] = true
		if strings.TrimSpace(entry.Command) == "" {
			findings = append(findings, fmt.Sprintf("post-cutover check %s names no command", label))
		}
		if strings.TrimSpace(entry.Observation) == "" {
			findings = append(findings, fmt.Sprintf("post-cutover check %s records no observation", label))
		}
		if strings.TrimSpace(entry.Aspect) == "" {
			findings = append(findings, fmt.Sprintf("post-cutover check %s names no aspect", label))
			continue
		}
		covered[entry.Aspect] = true
	}
	for _, aspect := range requiredAspects {
		if !covered[aspect] {
			findings = append(findings, fmt.Sprintf("no post-cutover check covers the %s aspect", aspect))
		}
	}
	return findings
}

// cutoverTextValues returns every prose value of the record, so the version
// identifier scan reads the whole cutover record rather than one field.
func cutoverTextValues(record *cutoverRecord) []string {
	values := []string{record.Recovery.SafeState, record.Recovery.Method}
	for _, slice := range record.Order {
		values = append(values, slice.Delivers)
	}
	for _, row := range record.Participants {
		values = append(values, row.Surface, row.Readiness, row.ReadinessCheck, row.Recovery)
	}
	for _, trigger := range record.Recovery.Triggers {
		values = append(values, trigger.Condition, trigger.Action)
	}
	for _, step := range record.Recovery.Steps {
		values = append(values, step.Action, step.Observation)
	}
	for _, run := range record.Recovery.Rehearsal {
		values = append(values, run.Command, run.Observation)
	}
	for _, entry := range record.PostCutover {
		values = append(values, entry.Command, entry.Observation)
	}
	return values
}

// CheckCutoverRecord validates workflow/cutover-record.yml against the
// repository tree, the contract decisions, the module ownership model, and the
// architecture record that explains it. It fails on a malformed file, an
// unknown key, a missing field, a participant that leaves a recorded contract
// uncovered, an evidence path that does not exist, an untested recovery
// procedure, a missing post-cutover aspect, and a version identifier Issue
// #326 has not separately approved.
func CheckCutoverRecord(root string, out, errOut io.Writer) int {
	record, err := readCutoverRecord(filepath.Join(root, "workflow", "cutover-record.yml"))
	if err != nil {
		fmt.Fprintf(errOut, "cutover-record check failed: %v\n", err)
		return 1
	}
	model, err := readOwnershipModel(filepath.Join(root, "workflow", "module-ownership.yml"))
	if err != nil {
		fmt.Fprintf(errOut, "cutover-record check failed: %v\n", err)
		return 1
	}
	decisions, err := readContractDecisions(filepath.Join(root, "workflow", "contract-decisions.yml"))
	if err != nil {
		fmt.Fprintf(errOut, "cutover-record check failed: %v\n", err)
		return 1
	}
	document, err := os.ReadFile(filepath.Join(root, "docs", "architecture.md"))
	if err != nil {
		fmt.Fprintf(errOut, "cutover-record check failed: %v\n", err)
		return 1
	}
	architecture := string(document)

	// composition owns the surfaces that live in cmd/ and in the repository
	// configuration rather than inside one internal module.
	owners := map[string]bool{"composition": true}
	for _, module := range model.order {
		owners[module] = true
	}
	contracts := map[string]bool{}
	for _, decision := range decisions {
		contracts[decision.ID] = true
	}

	findings := checkCutoverPoint(record, architecture)
	findings = append(findings, checkCutoverOrder(record.Order)...)
	seen := map[string]bool{}
	claimed := map[string]bool{}
	for index, row := range record.Participants {
		findings = append(findings, checkCutoverParticipant(root, index, row, seen, owners, contracts, architecture)...)
		for _, contract := range row.Contracts {
			claimed[contract] = true
		}
	}
	for _, decision := range decisions {
		if !claimed[decision.ID] {
			findings = append(findings, fmt.Sprintf(
				"contract %s appears in no participant, so the matrix does not cover every contract of the cutover", decision.ID))
		}
	}
	// A module with no row would be a component of the cutover the matrix does
	// not carry, which is the omission Issue #332 rejects.
	owned := map[string]bool{}
	for _, row := range record.Participants {
		owned[row.Owner] = true
	}
	for _, module := range model.order {
		if !owned[module] {
			findings = append(findings, fmt.Sprintf(
				"module %s owns no participant, so the matrix does not cover every module of the cutover", module))
		}
	}
	findings = append(findings, checkCutoverRecovery(record.Recovery)...)
	findings = append(findings, checkPostCutover(record.PostCutover)...)

	identifiers := map[string]bool{}
	for _, value := range cutoverTextValues(record) {
		for _, identifier := range findVersionIdentifiers(value) {
			identifiers[identifier] = true
		}
	}
	if len(identifiers) > 0 {
		findings = append(findings, fmt.Sprintf(
			"the cutover record names the version identifier %s; Issue #326 requires a separate decision before one appears",
			strings.Join(sortedKeys(identifiers), ", ")))
	}

	if len(findings) > 0 {
		sort.Strings(findings)
		for _, finding := range findings {
			fmt.Fprintln(errOut, finding)
		}
		fmt.Fprintf(errOut, "cutover-record check failed: %d finding(s).\n", len(findings))
		return 1
	}
	fmt.Fprintf(out, "cutover record valid: %d participant(s), %d recovery step(s), %d post-cutover check(s).\n",
		len(record.Participants), len(record.Recovery.Steps), len(record.PostCutover))
	return 0
}
