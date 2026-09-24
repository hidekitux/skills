package check

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// contractDecision is one entry of workflow/contract-decisions.yml. A preserved
// entry carries reason and check; a changed entry carries the full change
// record instead.
type contractDecision struct {
	ID             string   `yaml:"id"`
	Surface        string   `yaml:"surface"`
	Classification string   `yaml:"classification"`
	Owner          string   `yaml:"owner"`
	Evidence       []string `yaml:"evidence"`
	Reason         string   `yaml:"reason"`
	Check          string   `yaml:"check"`
	Old            string   `yaml:"old"`
	New            string   `yaml:"new"`
	Confirmation   string   `yaml:"confirmation"`
	Migration      string   `yaml:"migration"`
	Compatibility  string   `yaml:"compatibility"`
	Validation     string   `yaml:"validation"`
	// VersionDecision records the separate decision Issue #326 requires before
	// any entry may name a version identifier.
	VersionDecision string `yaml:"version_decision"`
}

// contractIDRE is the identifier shape every entry id must have.
var contractIDRE = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// versionIdentifierRE finds a candidate version identifier such as v1 or v2.
// A tool version such as v1.7.0 is not one; findVersionIdentifiers drops a
// match a dot or a hyphen continues.
var versionIdentifierRE = regexp.MustCompile(`\bv[0-9]+`)

// changeFields are the fields only a changed entry may carry, in report order.
var changeFields = []string{"old", "new", "confirmation", "migration", "compatibility", "validation"}

// changeFieldValues returns the change-record fields of one entry keyed by the
// field name the file uses.
func (d contractDecision) changeFieldValues() map[string]string {
	return map[string]string{
		"old":           d.Old,
		"new":           d.New,
		"confirmation":  d.Confirmation,
		"migration":     d.Migration,
		"compatibility": d.Compatibility,
		"validation":    d.Validation,
	}
}

// textValues returns every prose value of one entry, so a scan reads the whole
// recorded decision rather than one field.
func (d contractDecision) textValues() []string {
	values := []string{d.Surface, d.Owner, d.Reason, d.Check}
	for _, field := range changeFields {
		values = append(values, d.changeFieldValues()[field])
	}
	return values
}

// findVersionIdentifiers returns the version identifiers in text. A match that
// a dot or a hyphen continues is a pinned tool or module version, not a
// compatibility identifier, so it is not returned.
func findVersionIdentifiers(text string) []string {
	found := []string{}
	for _, span := range versionIdentifierRE.FindAllStringIndex(text, -1) {
		end := span[1]
		if end < len(text) && (text[end] == '.' || text[end] == '-') {
			continue
		}
		found = append(found, text[span[0]:end])
	}
	return found
}

// readContractDecisions parses workflow/contract-decisions.yml.
func readContractDecisions(path string) ([]contractDecision, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		SchemaVersion int                `yaml:"schema_version"`
		Contracts     []contractDecision `yaml:"contracts"`
	}
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("cannot parse contract-decisions.yml: %w", err)
	}
	if doc.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported schema_version %d, want 1", doc.SchemaVersion)
	}
	if len(doc.Contracts) == 0 {
		return nil, fmt.Errorf("contract-decisions.yml lists no contract")
	}
	return doc.Contracts, nil
}

// checkContractEntry returns the findings for one entry: its required fields,
// the fields its classification allows, and the evidence paths it cites.
func checkContractEntry(root string, index int, entry contractDecision, seen map[string]bool) []string {
	findings := []string{}
	label := entry.ID
	if label == "" {
		label = fmt.Sprintf("entry %d", index+1)
		findings = append(findings, fmt.Sprintf("%s has no id", label))
	} else if !contractIDRE.MatchString(label) {
		findings = append(findings, fmt.Sprintf("contract %s has an id that is not lower-case words joined by hyphens", label))
	}
	if seen[entry.ID] && entry.ID != "" {
		findings = append(findings, fmt.Sprintf("contract %s is listed more than once", label))
	}
	seen[entry.ID] = true
	if strings.TrimSpace(entry.Surface) == "" {
		findings = append(findings, fmt.Sprintf("contract %s does not state the surface it covers", label))
	}
	if strings.TrimSpace(entry.Owner) == "" {
		findings = append(findings, fmt.Sprintf("contract %s names no owning module", label))
	}
	if len(entry.Evidence) == 0 {
		findings = append(findings, fmt.Sprintf("contract %s cites no evidence path", label))
	}
	for _, path := range entry.Evidence {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
			findings = append(findings, fmt.Sprintf("contract %s cites %s, which does not exist", label, path))
		}
	}
	values := entry.changeFieldValues()
	switch entry.Classification {
	case "preserved":
		if strings.TrimSpace(entry.Reason) == "" {
			findings = append(findings, fmt.Sprintf("preserved contract %s does not state why it remains preserved", label))
		}
		if strings.TrimSpace(entry.Check) == "" {
			findings = append(findings, fmt.Sprintf("preserved contract %s names no check that observes it", label))
		}
		for _, field := range changeFields {
			if strings.TrimSpace(values[field]) != "" {
				findings = append(findings, fmt.Sprintf("preserved contract %s carries the change field %s; classify it as changed instead", label, field))
			}
		}
	case "changed":
		if strings.TrimSpace(entry.Reason) == "" {
			findings = append(findings, fmt.Sprintf("changed contract %s states no reason", label))
		}
		for _, field := range changeFields {
			if strings.TrimSpace(values[field]) == "" {
				findings = append(findings, fmt.Sprintf("changed contract %s records no %s", label, field))
			}
		}
	case "":
		findings = append(findings, fmt.Sprintf("contract %s has no classification; want preserved or changed", label))
	default:
		findings = append(findings, fmt.Sprintf("contract %s has classification %q; want preserved or changed", label, entry.Classification))
	}
	if strings.TrimSpace(entry.VersionDecision) == "" {
		identifiers := map[string]bool{}
		for _, value := range entry.textValues() {
			for _, identifier := range findVersionIdentifiers(value) {
				identifiers[identifier] = true
			}
		}
		named := make([]string, 0, len(identifiers))
		for identifier := range identifiers {
			named = append(named, identifier)
		}
		sort.Strings(named)
		if len(named) > 0 {
			findings = append(findings, fmt.Sprintf(
				"contract %s names the version identifier %s without a version_decision; Issue #326 requires a separate decision",
				label, strings.Join(named, ", ")))
		}
	}
	return findings
}

// CheckContractDecisions validates workflow/contract-decisions.yml against the
// repository tree and against the architecture record that explains it. It
// fails on a malformed file, an incomplete decision, an evidence path that does
// not exist, an entry the architecture record does not describe, and a version
// identifier Issue #326 has not separately approved.
func CheckContractDecisions(root string, out, errOut io.Writer) int {
	entries, err := readContractDecisions(filepath.Join(root, "workflow", "contract-decisions.yml"))
	if err != nil {
		fmt.Fprintf(errOut, "contract-decisions check failed: %v\n", err)
		return 1
	}
	record, err := os.ReadFile(filepath.Join(root, "docs", "architecture.md"))
	if err != nil {
		fmt.Fprintf(errOut, "contract-decisions check failed: %v\n", err)
		return 1
	}
	findings := []string{}
	seen := map[string]bool{}
	preserved, changed := 0, 0
	for index, entry := range entries {
		findings = append(findings, checkContractEntry(root, index, entry, seen)...)
		switch entry.Classification {
		case "preserved":
			preserved++
		case "changed":
			changed++
		}
		if entry.ID != "" && !strings.Contains(string(record), entry.ID) {
			findings = append(findings, fmt.Sprintf(
				"contract %s does not appear in docs/architecture.md; the record and the file must describe the same decisions", entry.ID))
		}
	}
	if len(findings) > 0 {
		sort.Strings(findings)
		for _, finding := range findings {
			fmt.Fprintln(errOut, finding)
		}
		fmt.Fprintf(errOut, "contract-decisions check failed: %d finding(s).\n", len(findings))
		return 1
	}
	fmt.Fprintf(out, "contract decisions valid: %d contract(s), %d preserved, %d changed.\n",
		len(entries), preserved, changed)
	return 0
}
