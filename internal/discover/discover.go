// Package discover implements the repository's canonical skill discovery and
// path-identity contract. Every publishable skill lives under skills/ in either
// a flat layout (skills/<name>/SKILL.md) or a namespaced layout
// (skills/<namespace>/<name>/SKILL.md). Discovery is recursive, and each
// skill's identity is its bare name (the directory holding SKILL.md) while its
// canonical path identity is its repository-relative directory. Consumers that
// resolve a catalog entry to one skill file must agree on this contract so that
// repository validation, host validation, and local registration stay aligned.
package discover

import (
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// Skill is a discovered publishable skill.
type Skill struct {
	// Name is the bare skill name: the directory that holds SKILL.md.
	Name string
	// Dir is the repository-relative, slash-separated directory of the skill,
	// for example "skills/fix/refactor-code" for a category skill or
	// "skills/vendor/refactor-code" for an arbitrary namespaced skill. This is the
	// canonical path identity used to disambiguate duplicate bare names.
	Dir string
}

// All walks root/skills recursively and returns every discovered skill in
// deterministic order (lexical by Dir).
func All(root string) []Skill {
	var skills []Skill
	_ = filepath.WalkDir(filepath.Join(root, "skills"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}
		dir := filepath.Dir(path)
		relDir, err := filepath.Rel(root, dir)
		if err != nil {
			return nil
		}
		skills = append(skills, Skill{Name: filepath.Base(dir), Dir: filepath.ToSlash(relDir)})
		return nil
	})
	sort.Slice(skills, func(i, j int) bool { return skills[i].Dir < skills[j].Dir })
	return skills
}

// ByName groups every discovered skill under root/skills by bare name. The
// skills within each group are deterministic (lexical by Dir), so duplicate
// bare names can be resolved to a single canonical path.
func ByName(root string) map[string][]Skill {
	byName := map[string][]Skill{}
	for _, skill := range All(root) {
		byName[skill.Name] = append(byName[skill.Name], skill)
	}
	return byName
}

// Names returns the sorted, de-duplicated bare skill names under root/skills.
func Names(root string) []string {
	seen := map[string]bool{}
	for _, skill := range All(root) {
		seen[skill.Name] = true
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Catalog kinds. A catalog entry without a kind is a workflow skill.
const (
	// KindWorkflow marks a workflow skill, which hands artifacts to other
	// skills through workflow/skill-graph.yml.
	KindWorkflow = "workflow"
	// KindStack marks a technology skill, which gives the conventions, build,
	// and test commands for one technology and hands off to no skill.
	KindStack = "stack"
)

// TechnologyCategories lists the first directories below skills/ that hold
// technology skills, in lexical order.
var TechnologyCategories = []string{"game", "language", "mobile", "web"}

// Category returns the first directory below skills/ for a namespaced skill
// directory such as "skills/language/develop-go", or "" for a flat skill.
func Category(dir string) string {
	parts := strings.Split(filepath.ToSlash(dir), "/")
	if len(parts) < 3 || parts[0] != "skills" {
		return ""
	}
	return parts[1]
}

// IsTechnologyCategory reports whether category is one of
// TechnologyCategories.
func IsTechnologyCategory(category string) bool {
	for _, candidate := range TechnologyCategories {
		if candidate == category {
			return true
		}
	}
	return false
}
