package check

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// catalogSkill is one authoritative entry in CATALOG.yml.
type catalogSkill struct {
	Name string
}

// docClaim is one documented inventory claim parsed from a table row in a
// contributor-facing document.
type docClaim struct {
	line    int
	skill   string
	status  string
	planned bool
}

var (
	// plannedCountRE matches the README's tracked planned-skill count, e.g.
	// "and tracks 0 planned next-generation skills".
	plannedCountRE = regexp.MustCompile(`tracks?\s+(\d+)\s+planned`)
	// plannedBulletRE matches one subject entry of a "- Planned:" bullet: a
	// backticked skill name optionally followed by its feature-Issue link,
	// e.g. "`write-tests` ([#69](...))".
	plannedBulletRE = regexp.MustCompile("^`([a-z0-9]+(?:-[a-z0-9]+)*)`(?:\\s*\\(\\[#[0-9]+\\]\\([^)]*\\)\\))?")
	// plannedMarkerRE matches a skill name immediately followed by a
	// "(planned)" marker, tolerating the backticks the docs use around names.
	plannedMarkerRE = regexp.MustCompile("`?([a-z0-9]+(?:-[a-z0-9]+)*)`?\\s*\\(\\s*planned\\s*\\)")
)

// readCatalog loads the authoritative skill entries from CATALOG.yml, keyed by
// skill name.
func readCatalog(path string) (map[string]catalogSkill, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Skills []struct {
			Name string `yaml:"name"`
		} `yaml:"skills"`
	}
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, err
	}
	catalog := map[string]catalogSkill{}
	for _, entry := range doc.Skills {
		if entry.Name == "" {
			continue
		}
		catalog[entry.Name] = catalogSkill{Name: entry.Name}
	}
	return catalog, nil
}

// splitRow splits one markdown pipe-table row into trimmed cells.
func splitRow(line string) []string {
	line = strings.TrimSpace(line)
	line = strings.TrimPrefix(line, "|")
	line = strings.TrimSuffix(line, "|")
	raw := strings.Split(line, "|")
	cells := make([]string, 0, len(raw))
	for _, cell := range raw {
		cells = append(cells, strings.TrimSpace(cell))
	}
	return cells
}

// normalizeCell lowers a header cell and strips formatting so "Skill" and
// "`Skill`" both map to "skill".
func normalizeCell(cell string) string {
	return strings.ToLower(strings.Trim(strings.TrimSpace(cell), "`"))
}

// columnIndex returns the index of the header cell matching want, or -1.
func columnIndex(header []string, want string) int {
	for i, cell := range header {
		if normalizeCell(cell) == want {
			return i
		}
	}
	return -1
}

// isTableRow reports whether the line starts a markdown table row.
func isTableRow(line string) bool {
	return strings.HasPrefix(strings.TrimSpace(line), "|")
}

// isSeparator reports whether the row is a markdown header separator such as
// "| --- | --- |".
func isSeparator(cells []string) bool {
	for _, cell := range cells {
		if strings.Trim(cell, "-: ") != "" {
			return false
		}
	}
	return true
}

// parseSkillCell extracts the skill name from a table cell, recognizing a
// trailing "(planned)" or "(published)" status marker.
func parseSkillCell(cell string) (string, bool) {
	name := strings.TrimSpace(cell)
	planned := false
	switch lower := strings.ToLower(name); {
	case strings.HasSuffix(lower, "(planned)"):
		planned = true
		name = strings.TrimSpace(name[:len(name)-len("(planned)")])
	case strings.HasSuffix(lower, "(published)"):
		name = strings.TrimSpace(name[:len(name)-len("(published)")])
	}
	return strings.Trim(name, "`"), planned
}

// tableClaims parses every inventory table in the document lines: a table
// whose header has a "skill" column. It returns one claim per data row.
func tableClaims(lines []string) []docClaim {
	var claims []docClaim
	for i := 0; i < len(lines); i++ {
		if !isTableRow(lines[i]) {
			continue
		}
		header := splitRow(lines[i])
		skillIdx := columnIndex(header, "skill")
		if skillIdx < 0 {
			continue
		}
		statusIdx := columnIndex(header, "status")
		i++
		for i < len(lines) && isTableRow(lines[i]) {
			cells := splitRow(lines[i])
			if isSeparator(cells) {
				i++
				continue
			}
			if skillIdx < len(cells) {
				skill, planned := parseSkillCell(cells[skillIdx])
				if skill != "" {
					claim := docClaim{line: i + 1, skill: skill, planned: planned}
					if statusIdx >= 0 && statusIdx < len(cells) {
						claim.status = cells[statusIdx]
						if strings.EqualFold(cells[statusIdx], "planned") {
							claim.planned = true
						}
					}
					claims = append(claims, claim)
				}
			}
			i++
		}
		i--
	}
	return claims
}

// plannedBulletSubjects returns the skill names listed as subjects of a
// "- Planned:" bullet: comma-separated backticked names, each optionally
// followed by a parenthesized feature-Issue link. Parsing stops at the first
// text that is not a list entry, so context prose such as "related to
// `plan-issue`" does not count as a planned subject.
func plannedBulletSubjects(line string) []string {
	content := strings.TrimSpace(line)
	if idx := strings.Index(strings.ToLower(content), ":"); idx >= 0 {
		content = strings.TrimSpace(content[idx+1:])
	}
	var subjects []string
	for content != "" {
		m := plannedBulletRE.FindStringSubmatchIndex(content)
		if m == nil || m[0] != 0 {
			return subjects
		}
		subjects = append(subjects, content[m[2]:m[3]])
		content = content[m[1]:]
		content = strings.TrimLeft(content, " \t")
		if !strings.HasPrefix(content, ",") {
			return subjects
		}
		content = strings.TrimLeft(content[1:], " \t")
	}
	return subjects
}

// checkDocTables checks the planned markers in the skill tables of one
// contributor-facing document: a cataloged skill must not be marked planned,
// and every planned skill outside the catalog is recorded for the README
// planned count. The generated skill list (check-skill-lists) owns the
// current inventory, layers, and statuses.
func checkDocTables(docName, text string, catalog map[string]catalogSkill, findings *[]string, plannedAbsent map[string]bool) {
	for _, claim := range tableClaims(strings.Split(text, "\n")) {
		if !claim.planned {
			continue
		}
		if _, ok := catalog[claim.skill]; ok {
			*findings = append(*findings, fmt.Sprintf("%s:%d: %s is described as planned, but it exists in CATALOG.yml", docName, claim.line, claim.skill))
		} else {
			plannedAbsent[claim.skill] = true
		}
	}
}

// checkPlannedText rejects any catalog skill that is described as planned in
// the document and records every planned skill the document names, whether in
// a "Planned:" bullet or in a "(planned)" marker, so the README planned count
// is compared against the same set.
func checkPlannedText(docName, text string, catalog map[string]catalogSkill, findings *[]string, plannedAbsent map[string]bool) {
	lines := strings.Split(text, "\n")
	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])
		if !strings.HasPrefix(strings.ToLower(trimmed), "- planned:") {
			continue
		}
		bullet := trimmed
		for i+1 < len(lines) && lines[i+1] != "" && (lines[i+1][0] == ' ' || lines[i+1][0] == '\t') {
			bullet += " " + strings.TrimSpace(lines[i+1])
			i++
		}
		for _, subject := range plannedBulletSubjects(bullet) {
			if _, ok := catalog[subject]; ok {
				*findings = append(*findings, fmt.Sprintf("%s: %s is listed as planned, but it exists in CATALOG.yml", docName, subject))
			} else {
				plannedAbsent[subject] = true
			}
		}
	}
	for _, m := range plannedMarkerRE.FindAllStringSubmatch(text, -1) {
		subject := m[1]
		if _, ok := catalog[subject]; ok {
			*findings = append(*findings, fmt.Sprintf("%s: %s is described as planned, but it exists in CATALOG.yml", docName, subject))
		} else {
			plannedAbsent[subject] = true
		}
	}
}

// checkReadmeCounts verifies the README planned count against the distinct
// non-catalog skills the documentation marks planned. The generated skill
// list owns the published count.
func checkReadmeCounts(readme string, plannedAbsent map[string]bool, findings *[]string) {
	if p := plannedCountRE.FindStringSubmatch(readme); p == nil {
		*findings = append(*findings, `README.md must state the tracked planned count as "tracks N planned"`)
	} else if n, err := strconv.Atoi(p[1]); err != nil || n != len(plannedAbsent) {
		*findings = append(*findings, fmt.Sprintf("README.md states %s planned skills, but the documents list %d", p[1], len(plannedAbsent)))
	}
}

// CheckCatalogDocs checks the planned-skill claims in README.md,
// docs/skill-layers.md, and docs/skill-contract.md against CATALOG.yml: a
// cataloged skill must not be described as planned, and the README planned
// count must match the planned skills the documents name. It returns 0 on
// success or 1 when documentation drifts.
func CheckCatalogDocs(root string, out, errOut io.Writer) int {
	catalog, err := readCatalog(filepath.Join(root, "CATALOG.yml"))
	if err != nil {
		fmt.Fprintf(errOut, "Catalog-docs check failed: cannot read CATALOG.yml: %v\n", err)
		return 1
	}

	readDoc := func(rel string) string {
		content, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			return ""
		}
		return string(content)
	}
	readme := readDoc("README.md")
	layers := readDoc(filepath.Join("docs", "skill-layers.md"))
	contract := readDoc(filepath.Join("docs", "skill-contract.md"))

	findings := []string{}
	if readme == "" {
		findings = append(findings, "README.md is missing")
	}
	if layers == "" {
		findings = append(findings, "docs/skill-layers.md is missing")
	}
	if contract == "" {
		findings = append(findings, "docs/skill-contract.md is missing")
	}

	plannedAbsent := map[string]bool{}
	checkDocTables("README.md", readme, catalog, &findings, plannedAbsent)
	checkDocTables("docs/skill-layers.md", layers, catalog, &findings, plannedAbsent)
	checkDocTables("docs/skill-contract.md", contract, catalog, &findings, plannedAbsent)
	checkPlannedText("README.md", readme, catalog, &findings, plannedAbsent)
	checkPlannedText("docs/skill-layers.md", layers, catalog, &findings, plannedAbsent)
	checkPlannedText("docs/skill-contract.md", contract, catalog, &findings, plannedAbsent)
	checkReadmeCounts(readme, plannedAbsent, &findings)

	sort.Strings(findings)
	if len(findings) > 0 {
		fmt.Fprintln(errOut, "Catalog-docs check failed:")
		for _, finding := range findings {
			fmt.Fprintf(errOut, "- %s\n", finding)
		}
		return 1
	}
	fmt.Fprintf(out, "Catalog-docs check passed: contributor documentation matches CATALOG.yml for %d skills.\n", len(catalog))
	return 0
}
