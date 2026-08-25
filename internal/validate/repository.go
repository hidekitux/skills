// Package validate implements deterministic repository metadata validators
// ported from the former Python scripts under scripts/validate/.
package validate

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hidekitux/skills/internal/discover"
	"gopkg.in/yaml.v3"
)

const (
	expectedLicense       = "Apache-2.0"
	expectedLicenseSHA256 = "c71d239df91726fc519c6eb72d318ec65820627232b2f796219e87dcf35d0ab4"
	expectedNotice        = `hidekitux/skills
Copyright 2026 Hideki Tanaka

This repository and its published skills are licensed under the Apache License,
Version 2.0. See the LICENSE file for the complete license text.
`
)

var (
	validStatuses      = map[string]bool{"experimental": true, "stable": true, "deprecated": true}
	validLayers        = map[string]bool{"process": true, "analyze": true, "fix": true, "govern": true}
	versionPattern     = regexp.MustCompile(`^\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?$`)
	skillNamePattern   = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	frontmatterPattern = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n`)
	todoHeadingPattern = regexp.MustCompile(`(?im)^#{1,6}\s+.*todo list.*$`)
)

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func valueString(value any) (string, bool) {
	s, ok := value.(string)
	return s, ok
}

func truthy(value any) bool {
	switch v := value.(type) {
	case nil:
		return false
	case string:
		return v != ""
	default:
		return true
	}
}

func parseFrontmatter(text string) (map[string]any, string, error) {
	match := frontmatterPattern.FindStringSubmatchIndex(text)
	if match == nil || match[0] < 0 {
		return nil, "", errors.New("missing YAML frontmatter")
	}
	yamlText := text[match[2]:match[3]]
	rest := text[match[1]:]
	var decoded map[string]any
	if err := yaml.Unmarshal([]byte(yamlText), &decoded); err != nil {
		return nil, "", err
	}
	if decoded == nil {
		return nil, "", errors.New("frontmatter must be a YAML mapping")
	}
	return decoded, rest, nil
}

func validateLicense(path string, errors *[]string) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		*errors = append(*errors, "LICENSE is missing")
		return
	}
	if err != nil {
		*errors = append(*errors, fmt.Sprintf("LICENSE cannot be read: %v", err))
		return
	}
	digest := sha256.Sum256(content)
	if hex.EncodeToString(digest[:]) != expectedLicenseSHA256 {
		*errors = append(*errors, "LICENSE must contain the unmodified Apache-2.0 text")
	}
}

func validateNotice(path string, errors *[]string) {
	content, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		*errors = append(*errors, "NOTICE is missing")
		return
	}
	if err != nil {
		*errors = append(*errors, fmt.Sprintf("NOTICE cannot be read: %v", err))
		return
	}
	if string(content) != expectedNotice {
		*errors = append(*errors, "NOTICE must retain the confirmed repository attribution and Apache-2.0 notice")
	}
}

func validateFSLLayout(root string, skillFiles []string, errors *[]string) {
	rootSpecs := filepath.Join(root, "specs")
	type skillSpec struct{ skillDir, source string }
	var skillSpecs []skillSpec
	for _, skillFile := range skillFiles {
		skillDir := filepath.Dir(skillFile)
		specDir := filepath.Join(skillDir, "specs")
		if info, statErr := os.Stat(specDir); statErr == nil && info.IsDir() {
			_ = filepath.WalkDir(specDir, func(path string, d fs.DirEntry, err error) error {
				if err != nil || d.IsDir() || !strings.HasSuffix(path, ".fsl") {
					return nil
				}
				skillSpecs = append(skillSpecs, skillSpec{skillDir, path})
				return nil
			})
		}
	}

	rel := func(path string) string {
		if r, err := filepath.Rel(root, path); err == nil {
			return r
		}
		return path
	}

	for _, spec := range skillSpecs {
		skillRelative, err := filepath.Rel(filepath.Join(root, "skills"), spec.skillDir)
		if err != nil {
			skillRelative = spec.skillDir
		}
		specRel, err := filepath.Rel(filepath.Join(spec.skillDir, "specs"), spec.source)
		if err != nil {
			specRel = spec.source
		}
		expectedLink := filepath.Join(rootSpecs, skillRelative, specRel)
		linkInfo, linkErr := os.Lstat(expectedLink)
		if linkErr != nil || linkInfo.Mode()&os.ModeSymlink == 0 {
			*errors = append(*errors, fmt.Sprintf("%s: missing repository link %s", rel(spec.source), rel(expectedLink)))
			continue
		}
		target, resolveErr := filepath.EvalSymlinks(expectedLink)
		if resolveErr != nil {
			*errors = append(*errors, fmt.Sprintf("%s: broken FSL link: %v", rel(expectedLink), resolveErr))
			continue
		}
		sourceResolved, sourceErr := filepath.EvalSymlinks(spec.source)
		if sourceErr != nil {
			continue
		}
		if target != sourceResolved {
			*errors = append(*errors, fmt.Sprintf("%s: must link to %s", rel(expectedLink), rel(spec.source)))
		}
	}

	if info, err := os.Stat(rootSpecs); err != nil || !info.IsDir() {
		*errors = append(*errors, "specs/ is missing")
		return
	}
	_ = filepath.WalkDir(rootSpecs, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".fsl") {
			return nil
		}
		if filepath.Dir(path) != rootSpecs && !isSymlink(path) {
			*errors = append(*errors, fmt.Sprintf("%s: nested repository FSL entries must be relative symbolic links to skill-owned specs", rel(path)))
		}
		if isSymlink(path) {
			target, resolveErr := filepath.EvalSymlinks(path)
			if resolveErr != nil {
				*errors = append(*errors, fmt.Sprintf("%s: broken FSL link: %v", rel(path), resolveErr))
				return nil
			}
			info, statErr := os.Stat(target)
			if statErr != nil || info.IsDir() || !strings.HasSuffix(target, ".fsl") {
				*errors = append(*errors, fmt.Sprintf("%s: FSL link must resolve to a .fsl file", rel(path)))
			}
		}
		return nil
	})
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

// CheckRepository validates repository-wide metadata and skill authoring
// conventions, returning 0 on success or 1 when validation fails.
func CheckRepository(root string, out, errOut io.Writer) int {
	errors := []string{}

	catalogPath := filepath.Join(root, "CATALOG.yml")
	validateLicense(filepath.Join(root, "LICENSE"), &errors)
	validateNotice(filepath.Join(root, "NOTICE"), &errors)
	catalogContent, err := os.ReadFile(catalogPath)
	if os.IsNotExist(err) {
		errors = append(errors, "CATALOG.yml is missing")
		printRepositoryErrors(errors, errOut)
		return 1
	}
	if err != nil {
		errors = append(errors, fmt.Sprintf("CATALOG.yml cannot be parsed: %v", err))
		printRepositoryErrors(errors, errOut)
		return 1
	}
	var catalog map[string]any
	if err := yaml.Unmarshal(catalogContent, &catalog); err != nil {
		errors = append(errors, fmt.Sprintf("CATALOG.yml cannot be parsed: %v", err))
		printRepositoryErrors(errors, errOut)
		return 1
	}
	if catalog == nil {
		errors = append(errors, "CATALOG.yml must contain a YAML mapping")
		printRepositoryErrors(errors, errOut)
		return 1
	}
	if license, _ := valueString(catalog["license"]); license != expectedLicense {
		errors = append(errors, "CATALOG.yml must declare license: Apache-2.0")
	}
	entries, ok := catalog["skills"].([]any)
	if !ok {
		errors = append(errors, "CATALOG.yml must contain a skills list")
		printRepositoryErrors(errors, errOut)
		return 1
	}

	skillFiles := []string{}
	skillByName := map[string][]string{}
	for _, skill := range discover.All(root) {
		skillPath := filepath.Join(root, skill.Dir, "SKILL.md")
		skillFiles = append(skillFiles, skillPath)
		skillByName[skill.Name] = append(skillByName[skill.Name], skill.Dir)

		content, readErr := os.ReadFile(skillPath)
		if readErr != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", rel(root, skillPath), readErr))
			continue
		}
		metadata, body, parseErr := parseFrontmatter(string(content))
		if parseErr != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", rel(root, skillPath), parseErr))
			continue
		}
		relativePath := rel(root, skillPath)
		if name, _ := valueString(metadata["name"]); name != skill.Name {
			errors = append(errors, fmt.Sprintf("%s: frontmatter name must match its directory", relativePath))
		}
		if description, ok := valueString(metadata["description"]); !ok || strings.TrimSpace(description) == "" {
			errors = append(errors, fmt.Sprintf("%s: description is required", relativePath))
		}
		if license, _ := valueString(metadata["license"]); license != expectedLicense {
			errors = append(errors, fmt.Sprintf("%s: license must be Apache-2.0", relativePath))
		}
		if !todoHeadingPattern.MatchString(body) {
			errors = append(errors, fmt.Sprintf("%s: a Todo List heading is required", relativePath))
		}
		bodyLower := strings.ToLower(body)
		for _, requiredTerm := range []string{"complete", "handoff"} {
			if !strings.Contains(bodyLower, requiredTerm) {
				errors = append(errors, fmt.Sprintf("%s: Todo List guidance must explain %q", relativePath, requiredTerm))
			}
		}
	}

	validateFSLLayout(root, skillFiles, &errors)

	catalogNames := map[string]bool{}
	for index, rawEntry := range entries {
		prefix := fmt.Sprintf("CATALOG.yml skills[%d]", index+1)
		entry, isMap := rawEntry.(map[string]any)
		if !isMap {
			errors = append(errors, prefix+" must be a mapping")
			continue
		}
		name, _ := valueString(entry["name"])
		if name == "" {
			errors = append(errors, prefix+".name is required")
			continue
		}
		if catalogNames[name] {
			errors = append(errors, fmt.Sprintf("%s: duplicate skill name %q", prefix, name))
		}
		catalogNames[name] = true
		matches := skillByName[name]
		resolvedDir := ""
		switch {
		case len(matches) == 0:
			errors = append(errors, fmt.Sprintf("%s: no skills/**/%s/SKILL.md found", prefix, name))
		case len(matches) == 1:
			if path, has := valueString(entry["path"]); has && path != "" {
				errors = append(errors, fmt.Sprintf("%s: catalog path %q is unnecessary for an unambiguous skill name", prefix, path))
			} else {
				resolvedDir = matches[0]
			}
		default:
			path, has := valueString(entry["path"])
			if !has || path == "" {
				errors = append(errors, fmt.Sprintf("%s: skill name is ambiguous; add a unique catalog path", prefix))
				break
			}
			for _, dir := range matches {
				if dir == path {
					resolvedDir = dir
					break
				}
			}
			if resolvedDir == "" {
				errors = append(errors, fmt.Sprintf("%s: catalog path %q does not resolve to a discovered %s skill under skills/", prefix, path, name))
			}
		}

		for _, field := range []string{"summary", "owner", "status", "license", "version"} {
			if !truthy(entry[field]) {
				errors = append(errors, fmt.Sprintf("%s.%s is required", prefix, field))
			}
		}
		if status, _ := valueString(entry["status"]); !validStatuses[status] {
			errors = append(errors, fmt.Sprintf("%s.status must be one of %v", prefix, sortedKeys(validStatuses)))
		}
		if license, _ := valueString(entry["license"]); license != expectedLicense {
			errors = append(errors, prefix+".license must be Apache-2.0")
		}
		if version, ok := valueString(entry["version"]); !ok || !versionPattern.MatchString(version) {
			errors = append(errors, prefix+".version must be semantic version text")
		}
		if layer, _ := valueString(entry["layer"]); !validLayers[layer] {
			errors = append(errors, fmt.Sprintf("%s.layer must be one of %v", prefix, sortedKeys(validLayers)))
		}

		related, hasRelated := entry["related"].([]any)
		if !hasRelated && entry["related"] == nil {
			related = []any{}
			hasRelated = true
		}
		if !hasRelated {
			errors = append(errors, prefix+".related must be a list")
		} else {
			seen := map[string]bool{}
			for _, rawRelated := range related {
				relatedName, isStr := rawRelated.(string)
				if !isStr || !skillNamePattern.MatchString(relatedName) {
					errors = append(errors, fmt.Sprintf("%s.related contains an invalid skill name %q", prefix, relatedName))
					continue
				}
				if relatedName == name {
					errors = append(errors, prefix+".related must not include the skill itself")
				}
				if seen[relatedName] {
					errors = append(errors, fmt.Sprintf("%s.related contains a duplicate skill name %q", prefix, relatedName))
				}
				seen[relatedName] = true
			}
		}

		adapters, hasAdapters := entry["host_adapters"].([]any)
		if !hasAdapters && entry["host_adapters"] == nil {
			adapters = []any{}
			hasAdapters = true
		}
		if !hasAdapters {
			errors = append(errors, prefix+".host_adapters must be a list")
		} else if resolvedDir != "" {
			for _, rawHost := range adapters {
				host, isStr := rawHost.(string)
				if !isStr || host == "" {
					errors = append(errors, prefix+".host_adapters contains an invalid host")
					continue
				}
				adapter := filepath.Join(root, resolvedDir, "references", "hosts", host+".md")
				if info, statErr := os.Stat(adapter); statErr != nil || info.IsDir() {
					errors = append(errors, fmt.Sprintf("%s: missing host adapter %s", prefix, rel(root, adapter)))
				}
			}
		}
	}

	missing := []string{}
	for name := range skillByName {
		if !catalogNames[name] {
			missing = append(missing, name)
		}
	}
	sort.Strings(missing)
	for _, name := range missing {
		errors = append(errors, fmt.Sprintf("skill %q is missing from CATALOG.yml", name))
	}

	if len(errors) > 0 {
		printRepositoryErrors(errors, errOut)
		return 1
	}
	entryNoun := "entries"
	if len(entries) == 1 {
		entryNoun = "entry"
	}
	fmt.Fprintf(out, "Repository metadata is valid: %d skill(s), %d catalog %s.\n", len(skillFiles), len(entries), entryNoun)
	return 0
}

func rel(root, path string) string {
	if r, err := filepath.Rel(root, path); err == nil {
		return r
	}
	return path
}

func printRepositoryErrors(errors []string, errOut io.Writer) {
	fmt.Fprintln(errOut, "Repository validation failed:")
	for _, error := range errors {
		fmt.Fprintf(errOut, "- %s\n", error)
	}
}
