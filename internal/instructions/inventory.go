// Package instructions measures and validates the always-loaded instruction
// inventory for published skills.
package instructions

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hidekitux/skills/internal/discover"
	"github.com/pkoukk/tiktoken-go"
	"gopkg.in/yaml.v3"
)

const (
	EncodingName     = "cl100k_base"
	TokenizerModule  = "github.com/pkoukk/tiktoken-go"
	TokenizerVersion = "v0.1.6"
)

// Classification is the allowed classification for one SKILL.md section.
type Classification string

const (
	Invariant            Classification = "invariant"
	TaskProcedure        Classification = "task-procedure"
	ConditionalReference Classification = "conditional-reference"
	MechanicallyEnforced Classification = "mechanically-enforced"
	RemovableDuplication Classification = "removable-duplication"
)

var validClassifications = map[Classification]bool{
	Invariant:            true,
	TaskProcedure:        true,
	ConditionalReference: true,
	MechanicallyEnforced: true,
	RemovableDuplication: true,
}

// Section records the classification and loading rule for one Markdown
// heading in a published skill.
type Section struct {
	Heading        string         `yaml:"heading"`
	Classification Classification `yaml:"classification"`
	LoadCondition  string         `yaml:"load_condition,omitempty"`
	Reference      string         `yaml:"reference,omitempty"`
}

// Skill records the measured source and section inventory for one skill.
type Skill struct {
	Name         string    `yaml:"name"`
	Path         string    `yaml:"path"`
	BeforeTokens int       `yaml:"before_tokens"`
	AfterTokens  int       `yaml:"after_tokens"`
	BeforeCommit string    `yaml:"before_commit"`
	AfterCommit  string    `yaml:"after_commit"`
	Sections     []Section `yaml:"sections"`
}

// Inventory is the repository-owned provenance record for instruction
// reduction. BeforeTokens and AfterTokens may be equal when a skill was
// audited and retained without compaction.
type Inventory struct {
	Version   int `yaml:"version"`
	Tokenizer struct {
		Encoding string `yaml:"encoding"`
		Module   string `yaml:"module"`
		Version  string `yaml:"version"`
	} `yaml:"tokenizer"`
	Skills []Skill `yaml:"skills"`
}

// Counter counts tokens in one text using the fixed repository encoding.
type Counter interface {
	Count(text string) (int, error)
}

type cl100kCounter struct{ encoding *tiktoken.Tiktoken }

func (c cl100kCounter) Count(text string) (int, error) {
	return len(c.encoding.EncodeOrdinary(text)), nil
}

// NewCounter loads the fixed tokenizer. The tokenizer package downloads its
// pinned encoding data on first use, so callers must treat network failure as
// measurement infrastructure failure rather than inventing a count.
func NewCounter() (Counter, error) {
	encoding, err := tiktoken.GetEncoding(EncodingName)
	if err != nil {
		return nil, fmt.Errorf("load %s tokenizer: %w", EncodingName, err)
	}
	return cl100kCounter{encoding: encoding}, nil
}

// Load reads and validates a machine-readable inventory.
func Load(path string) (Inventory, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Inventory{}, err
	}
	var inventory Inventory
	if err := yaml.Unmarshal(content, &inventory); err != nil {
		return Inventory{}, fmt.Errorf("decode instruction inventory: %w", err)
	}
	if inventory.Version != 1 {
		return Inventory{}, fmt.Errorf("instruction inventory version must be 1")
	}
	if inventory.Tokenizer.Encoding != EncodingName ||
		inventory.Tokenizer.Module != TokenizerModule ||
		inventory.Tokenizer.Version != TokenizerVersion {
		return Inventory{}, fmt.Errorf("instruction inventory tokenizer must be %s %s %s", TokenizerModule, TokenizerVersion, EncodingName)
	}
	for _, skill := range inventory.Skills {
		if skill.Name == "" || skill.Path == "" || skill.BeforeCommit == "" || skill.AfterCommit == "" {
			return Inventory{}, fmt.Errorf("every inventory skill needs name, path, and source commits")
		}
		seen := map[string]bool{}
		for _, section := range skill.Sections {
			if section.Heading == "" || !validClassifications[section.Classification] {
				return Inventory{}, fmt.Errorf("%s has an invalid section classification", skill.Name)
			}
			if seen[section.Heading] {
				return Inventory{}, fmt.Errorf("%s lists section %q more than once", skill.Name, section.Heading)
			}
			seen[section.Heading] = true
			if section.Classification == ConditionalReference && (section.LoadCondition == "" || section.Reference == "") {
				return Inventory{}, fmt.Errorf("%s section %q needs a load condition and reference", skill.Name, section.Heading)
			}
		}
	}
	return inventory, nil
}

// HeadingNames returns level-one through level-three Markdown headings in
// source order. It ignores headings inside fenced code blocks.
func HeadingNames(content string) []string {
	var headings []string
	fenced := false
	for _, line := range strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			fenced = !fenced
			continue
		}
		if fenced {
			continue
		}
		hashCount := len(trimmed) - len(strings.TrimLeft(trimmed, "#"))
		if hashCount < 2 || hashCount > 3 {
			continue
		}
		heading := strings.TrimSpace(strings.TrimLeft(trimmed, "#"))
		if heading != "" {
			headings = append(headings, heading)
		}
	}
	return headings
}

// ValidateCoverage ensures that the inventory names every discovered skill
// and every heading in each skill exactly once.
func ValidateCoverage(root string, inventory Inventory) error {
	byName := map[string]Skill{}
	for _, skill := range inventory.Skills {
		if _, exists := byName[skill.Name]; exists {
			return fmt.Errorf("inventory contains duplicate skill %q", skill.Name)
		}
		byName[skill.Name] = skill
	}
	discovered := discover.All(root)
	if len(discovered) != len(inventory.Skills) {
		return fmt.Errorf("inventory has %d skills but discovery found %d", len(inventory.Skills), len(discovered))
	}
	for _, discoveredSkill := range discovered {
		skill, ok := byName[discoveredSkill.Name]
		if !ok {
			return fmt.Errorf("inventory is missing %s", discoveredSkill.Name)
		}
		if filepath.ToSlash(skill.Path) != discoveredSkill.Dir+"/SKILL.md" {
			return fmt.Errorf("%s path must be %s", discoveredSkill.Name, discoveredSkill.Dir+"/SKILL.md")
		}
		content, err := os.ReadFile(filepath.Join(root, skill.Path))
		if err != nil {
			return fmt.Errorf("read %s: %w", skill.Path, err)
		}
		actual := HeadingNames(string(content))
		listed := make([]string, 0, len(skill.Sections))
		for _, section := range skill.Sections {
			if section.Classification == ConditionalReference {
				referencePath := filepath.Clean(section.Reference)
				relative, err := filepath.Rel(filepath.Dir(skill.Path), referencePath)
				if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
					return fmt.Errorf("%s section %q references a path outside its skill: %s", skill.Name, section.Heading, section.Reference)
				}
				info, err := os.Stat(filepath.Join(root, referencePath))
				if err != nil || !info.Mode().IsRegular() {
					return fmt.Errorf("%s section %q references missing file %s", skill.Name, section.Heading, section.Reference)
				}
			}
			listed = append(listed, section.Heading)
		}
		if !sameStrings(actual, listed) {
			return fmt.Errorf("%s sections do not match SKILL.md headings: actual=%v listed=%v", skill.Name, actual, listed)
		}
	}
	return nil
}

// Measure reads every cataloged skill and returns token counts using counter.
func Measure(root string, inventory Inventory, counter Counter) (map[string]int, error) {
	if err := ValidateCoverage(root, inventory); err != nil {
		return nil, err
	}
	counts := make(map[string]int, len(inventory.Skills))
	for _, skill := range inventory.Skills {
		content, err := os.ReadFile(filepath.Join(root, skill.Path))
		if err != nil {
			return nil, err
		}
		count, err := counter.Count(string(content))
		if err != nil {
			return nil, fmt.Errorf("count %s: %w", skill.Path, err)
		}
		counts[skill.Name] = count
	}
	return counts, nil
}

func sameStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

// SkillNames returns inventory skill names in deterministic order.
func SkillNames(inventory Inventory) []string {
	names := make([]string, 0, len(inventory.Skills))
	for _, skill := range inventory.Skills {
		names = append(names, skill.Name)
	}
	sort.Strings(names)
	return names
}
