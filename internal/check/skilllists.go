package check

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/hidekitux/skills/internal/discover"
	"gopkg.in/yaml.v3"
)

const (
	skillListBegin = "<!-- BEGIN generated: skill-list -->"
	skillListEnd   = "<!-- END generated: skill-list -->"
)

// skillListDocuments are the contributor documents that carry the generated
// skill list.
var skillListDocuments = []string{"README.md", filepath.Join("docs", "skill-layers.md"), filepath.Join("skills", "README.md")}

// skillListGroups is the order of the groups in the generated table: the
// workflow layers, then the technology categories.
var skillListGroups = append([]string{"process", "analyze", "fix", "govern"}, discover.TechnologyCategories...)

type skillListEntry struct {
	name   string
	group  string
	kind   string
	status string
}

// readSkillListEntries reads CATALOG.yml in order and resolves each
// technology skill's category from its directory.
func readSkillListEntries(root string) ([]skillListEntry, error) {
	content, err := os.ReadFile(filepath.Join(root, "CATALOG.yml"))
	if err != nil {
		return nil, err
	}
	var doc struct {
		Skills []struct {
			Name   string `yaml:"name"`
			Layer  string `yaml:"layer"`
			Kind   string `yaml:"kind"`
			Status string `yaml:"status"`
		} `yaml:"skills"`
	}
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, err
	}
	byName := discover.ByName(root)
	entries := make([]skillListEntry, 0, len(doc.Skills))
	for _, skill := range doc.Skills {
		entry := skillListEntry{name: skill.Name, group: skill.Layer, kind: discover.KindWorkflow, status: skill.Status}
		if skill.Kind == discover.KindStack {
			entry.kind = discover.KindStack
			entry.group = ""
			if matches := byName[skill.Name]; len(matches) == 1 {
				entry.group = discover.Category(matches[0].Dir)
			}
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func plural(count int, noun string) string {
	if count == 1 {
		return fmt.Sprintf("%d %s", count, noun)
	}
	return fmt.Sprintf("%d %ss", count, noun)
}

// renderSkillList renders the generated skill-list block, markers included.
func renderSkillList(entries []skillListEntry) (string, error) {
	rank := map[string]int{}
	for index, group := range skillListGroups {
		rank[group] = index
	}
	workflow, technology := 0, 0
	grouped := make([][]skillListEntry, len(skillListGroups))
	for _, entry := range entries {
		index, ok := rank[entry.group]
		if !ok {
			return "", fmt.Errorf("skill %q has no known layer or technology category", entry.name)
		}
		grouped[index] = append(grouped[index], entry)
		if entry.kind == discover.KindStack {
			technology++
		} else {
			workflow++
		}
	}
	var b strings.Builder
	b.WriteString(skillListBegin + "\n\n")
	b.WriteString("This list is generated from `CATALOG.yml` by `mise run generate:skill-lists` and checked by `check:repository`. Do not edit it by hand.\n\n")
	fmt.Fprintf(&b, "The repository publishes %s: %s and %s.\n\n", plural(len(entries), "skill"), plural(workflow, "workflow skill"), plural(technology, "technology skill"))
	b.WriteString("| Skill | Layer or technology category | Status |\n| --- | --- | --- |\n")
	for index, group := range grouped {
		for _, entry := range group {
			fmt.Fprintf(&b, "| `%s` | %s | %s |\n", entry.name, skillListGroups[index], entry.status)
		}
	}
	b.WriteString("\n" + skillListEnd)
	return b.String(), nil
}

// skillListBlock returns the marker-bounded block in text, markers included,
// and its byte offsets, or ok=false when either marker is missing.
func skillListBlock(text string) (block string, start, stop int, ok bool) {
	start = strings.Index(text, skillListBegin)
	if start < 0 {
		return "", 0, 0, false
	}
	end := strings.Index(text[start:], skillListEnd)
	if end < 0 {
		return "", 0, 0, false
	}
	stop = start + end + len(skillListEnd)
	return text[start:stop], start, stop, true
}

func renderSkillListForRoot(root string) (string, error) {
	entries, err := readSkillListEntries(root)
	if err != nil {
		return "", fmt.Errorf("cannot read CATALOG.yml: %w", err)
	}
	return renderSkillList(entries)
}

// CheckSkillLists fails when a document's generated skill list is missing or
// differs from a fresh render of CATALOG.yml. It returns 0 on success.
func CheckSkillLists(root string, out, errOut io.Writer) int {
	rendered, err := renderSkillListForRoot(root)
	if err != nil {
		fmt.Fprintf(errOut, "Skill-list check failed: %v\n", err)
		return 1
	}
	findings := []string{}
	for _, document := range skillListDocuments {
		content, err := os.ReadFile(filepath.Join(root, document))
		if err != nil {
			findings = append(findings, fmt.Sprintf("%s cannot be read: %v", document, err))
			continue
		}
		block, _, _, ok := skillListBlock(string(content))
		switch {
		case !ok:
			findings = append(findings, fmt.Sprintf("%s is missing the skill-list markers (BEGIN/END generated: skill-list)", document))
		case block != rendered:
			findings = append(findings, fmt.Sprintf("%s skill list is stale; run mise run generate:skill-lists", document))
		}
	}
	if len(findings) > 0 {
		fmt.Fprintln(errOut, "Skill-list check failed:")
		for _, finding := range findings {
			fmt.Fprintf(errOut, "- %s\n", finding)
		}
		return 1
	}
	fmt.Fprintf(out, "Skill-list check passed: %d document(s) match CATALOG.yml.\n", len(skillListDocuments))
	return 0
}

// WriteSkillLists rewrites the generated skill list in every document. A
// document must already contain the markers, so the list stays where an
// author placed it. Every document is read and checked before any is written.
// It returns 0 on success.
func WriteSkillLists(root string, out, errOut io.Writer) int {
	rendered, err := renderSkillListForRoot(root)
	if err != nil {
		fmt.Fprintf(errOut, "generate-skill-lists: %v\n", err)
		return 1
	}
	updates := map[string]string{}
	for _, document := range skillListDocuments {
		content, err := os.ReadFile(filepath.Join(root, document))
		if err != nil {
			fmt.Fprintf(errOut, "generate-skill-lists: cannot read %s: %v\n", document, err)
			return 1
		}
		_, start, stop, ok := skillListBlock(string(content))
		if !ok {
			fmt.Fprintf(errOut, "generate-skill-lists: %s is missing the skill-list markers (BEGIN/END generated: skill-list)\n", document)
			return 1
		}
		if updated := string(content[:start]) + rendered + string(content[stop:]); updated != string(content) {
			updates[document] = updated
		}
	}
	for _, document := range skillListDocuments {
		updated, ok := updates[document]
		if !ok {
			continue
		}
		if err := os.WriteFile(filepath.Join(root, document), []byte(updated), 0o644); err != nil {
			fmt.Fprintf(errOut, "generate-skill-lists: cannot write %s: %v\n", document, err)
			return 1
		}
	}
	fmt.Fprintf(out, "Regenerated the skill list in %d document(s).\n", len(updates))
	return 0
}
