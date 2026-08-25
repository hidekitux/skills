// Package release verifies and publishes repository release tags, ported from
// scripts/release/verify-release.py and scripts/release/publish-release.sh.
package release

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hidekitux/skills/internal/discover"
	"github.com/hidekitux/skills/internal/support"
	"gopkg.in/yaml.v3"
)

var tagPattern = regexp.MustCompile(`^v(?P<version>\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?)$`)

// findSkillDirectories maps every discovered publishable skill name to its
// repository-relative directory using the canonical recursive discovery
// contract, so release checks agree with repository and host validation.
func findSkillDirectories(root string) map[string]string {
	discovered := map[string]string{}
	for _, skill := range discover.All(root) {
		discovered[skill.Name] = skill.Dir
	}
	return discovered
}

// FindCrossSkillReferences returns errors when a released skill references a
// skill outside the catalog, so a release never ships a broken pointer.
func FindCrossSkillReferences(root string, catalogNames map[string]bool) []string {
	skillDirs := findSkillDirectories(root)
	names := make([]string, 0, len(catalogNames))
	for name := range catalogNames {
		names = append(names, name)
	}
	sortStrings(names)
	errors := []string{}
	for _, name := range names {
		skillDir, ok := skillDirs[name]
		if !ok {
			continue
		}
		content, err := os.ReadFile(filepath.Join(root, skillDir, "SKILL.md"))
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: cannot read SKILL.md: %v", name, err))
			continue
		}
		body := string(content)
		for other := range skillDirs {
			if other == name {
				continue
			}
			if !catalogNames[other] && referenceRE(other).MatchString(body) {
				errors = append(errors, fmt.Sprintf(
					"%s references skill %q which is not listed in the release catalog", name, other,
				))
			}
		}
	}
	return errors
}

func referenceRE(name string) *regexp.Regexp {
	return regexp.MustCompile(`(?i)(?:^|[^A-Za-z0-9-])` + regexp.QuoteMeta(name) + `(?:$|[^A-Za-z0-9-])`)
}

func sortStrings(values []string) {
	for i := 1; i < len(values); i++ {
		for j := i; j > 0 && values[j] < values[j-1]; j-- {
			values[j], values[j-1] = values[j-1], values[j]
		}
	}
}

// VerifyRelease checks that a release tag matches the skills catalog and the
// committed Git state, returning the process exit code.
func VerifyRelease(tag, root string, out, errOut io.Writer) int {
	errors := []string{}
	match := tagPattern.FindStringSubmatch(tag)
	version := ""
	if match == nil {
		errors = append(errors, "release tag must match vX.Y.Z (with an optional semver suffix)")
	} else {
		version = match[tagPattern.SubexpIndex("version")]
	}

	catalog := map[string]any{}
	catalogPath := filepath.Join(root, "CATALOG.yml")
	if catalogFile, err := os.ReadFile(catalogPath); err != nil {
		errors = append(errors, fmt.Sprintf("CATALOG.yml cannot be parsed: %v", err))
	} else if err := yaml.Unmarshal(catalogFile, &catalog); err != nil {
		errors = append(errors, fmt.Sprintf("CATALOG.yml cannot be parsed: %v", err))
	}

	catalogNames := map[string]bool{}
	entries, _ := catalog["skills"].([]any)
	if entries != nil {
		for _, rawEntry := range entries {
			if entry, ok := rawEntry.(map[string]any); ok {
				if name, ok := entry["name"].(string); ok {
					catalogNames[name] = true
				}
			}
		}
	}
	errors = append(errors, FindCrossSkillReferences(root, catalogNames)...)
	if len(entries) == 0 {
		errors = append(errors, "CATALOG.yml must contain at least one skill for a release")
	} else {
		for index, rawEntry := range entries {
			entry, ok := rawEntry.(map[string]any)
			if !ok {
				errors = append(errors, fmt.Sprintf("CATALOG.yml skills[%d] must be a mapping", index+1))
				continue
			}
			if version != "" {
				entryVersion, _ := entry["version"].(string)
				if entryVersion != version {
					errors = append(errors, fmt.Sprintf(
						"CATALOG.yml skills[%d] version %q does not match %q", index+1, entryVersion, version,
					))
				}
			}
		}
	}

	if _, err := gitOutput(root, "diff", "--quiet"); err != nil {
		errors = append(errors, "working tree has unstaged changes")
	}
	if _, err := gitOutput(root, "diff", "--cached", "--quiet"); err != nil {
		errors = append(errors, "index has staged changes not committed")
	}
	if untracked, err := gitOutput(root, "ls-files", "--others", "--exclude-standard"); err == nil {
		if lines := strings.Fields(untracked); len(lines) > 0 {
			errors = append(errors, "working tree has untracked files: "+strings.Join(lines, ", "))
		}
	} else {
		errors = append(errors, fmt.Sprintf("could not inspect Git state: %v", err))
	}

	if _, err := gitOutput(root, "rev-parse", "--verify", "--quiet", "refs/tags/"+tag); err == nil {
		errors = append(errors, fmt.Sprintf("tag %s already exists locally", tag))
	}
	remote, remoteErr := gitOutput(root, "remote", "get-url", "origin")
	if remoteErr != nil {
		errors = append(errors, "remote origin is required to verify the published tag")
	} else {
		lsRemote := execIn(root, "git", "ls-remote", "--exit-code", "--refs", remote, "refs/tags/"+tag)
		switch {
		case lsRemote == 0:
			errors = append(errors, fmt.Sprintf("tag %s already exists on the origin remote", tag))
		case lsRemote != 2:
			errors = append(errors, fmt.Sprintf("could not inspect tag %s on the origin remote", tag))
		}
	}

	if len(errors) > 0 {
		fmt.Fprintln(errOut, "Release verification failed:")
		for _, error := range errors {
			fmt.Fprintf(errOut, "- %s\n", error)
		}
		return 1
	}
	fmt.Fprintf(out, "Release contract is valid for %s: %d skill(s) at version %s.\n", tag, len(entries), version)
	return 0
}

func gitOutput(root string, args ...string) (string, error) {
	stdout, err := support.GitOutputIn(root, args...)
	return strings.TrimSpace(stdout), err
}

func execIn(root string, name string, args ...string) int {
	cmd := exec.Command(name, args...)
	cmd.Dir = root
	if name == "git" {
		cmd.Env = support.GitEnv()
	}
	return support.ExitError(cmd.Run())
}
