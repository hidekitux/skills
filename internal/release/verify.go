// Package release verifies and publishes repository release tags, ported from
// scripts/release/verify-release.py and scripts/release/publish-release.sh.
package release

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hidekitux/skills/internal/discover"
	"github.com/hidekitux/skills/internal/eval"
	"github.com/hidekitux/skills/internal/provider"
	"gopkg.in/yaml.v3"
)

var tagPattern = regexp.MustCompile(`^v(?P<version>\d+\.\d+\.\d+(?:[-+][0-9A-Za-z.-]+)?)$`)

// releasePorts holds the external systems a release touches. Building them
// from one Runner lets a test substitute every release command at once.
type releasePorts struct {
	git    provider.Git
	github provider.GitHub
	mise   provider.Mise
	tool   provider.Tool
}

func newReleasePorts(runner provider.Runner) releasePorts {
	return releasePorts{
		git:    provider.NewGit(runner),
		github: provider.NewGitHub(runner),
		mise:   provider.NewMise(runner),
		tool:   provider.NewTool(runner),
	}
}

func osReleasePorts() releasePorts { return newReleasePorts(provider.OSRunner{}) }

// gitOutput returns the trimmed standard output of a git command in root.
func (p releasePorts) gitOutput(root string, args ...string) (string, error) {
	result, err := p.git.Output(context.Background(), root, args...)
	return strings.TrimSpace(result.Stdout), err
}

// execIn runs a command in root and returns its exit code.
func (p releasePorts) execIn(root, name string, args ...string) int {
	result, _ := p.tool.Invoke(context.Background(), provider.Command{Name: name, Args: args, Dir: root})
	return result.ExitCode
}

// stream runs a release command and writes both streams to out and errOut.
func (p releasePorts) stream(name string, out, errOut io.Writer, args ...string) int {
	ctx := context.Background()
	switch name {
	case "mise":
		result, _ := p.mise.Task(ctx, "", out, errOut, args...)
		return result.ExitCode
	case "gh":
		result, _ := p.github.Stream(ctx, "", out, errOut, args...)
		return result.ExitCode
	default:
		result, _ := p.tool.Invoke(ctx, provider.Command{Name: name, Args: args, Stdout: out, Stderr: errOut})
		return result.ExitCode
	}
}

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
	return verifyRelease(tag, root, out, errOut, osReleasePorts())
}

func verifyRelease(tag, root string, out, errOut io.Writer, runner releasePorts) int {
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
	errors = append(errors, eval.PromotionFindings(root)...)

	if _, err := runner.gitOutput(root, "diff", "--quiet"); err != nil {
		errors = append(errors, "working tree has unstaged changes")
	}
	if _, err := runner.gitOutput(root, "diff", "--cached", "--quiet"); err != nil {
		errors = append(errors, "index has staged changes not committed")
	}
	if untracked, err := runner.gitOutput(root, "ls-files", "--others", "--exclude-standard"); err == nil {
		if lines := strings.Fields(untracked); len(lines) > 0 {
			errors = append(errors, "working tree has untracked files: "+strings.Join(lines, ", "))
		}
	} else {
		errors = append(errors, fmt.Sprintf("could not inspect Git state: %v", err))
	}

	if _, err := runner.gitOutput(root, "rev-parse", "--verify", "--quiet", "refs/tags/"+tag); err == nil {
		errors = append(errors, fmt.Sprintf("tag %s already exists locally", tag))
	}
	remote, remoteErr := runner.gitOutput(root, "remote", "get-url", "origin")
	if remoteErr != nil {
		errors = append(errors, "remote origin is required to verify the published tag")
	} else {
		lsRemote := runner.execIn(root, "git", "ls-remote", "--exit-code", "--refs", remote, "refs/tags/"+tag)
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
