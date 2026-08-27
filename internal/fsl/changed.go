package fsl

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hidekitux/skills/internal/support"
)

// isScopedFSLPath reports whether rel is a repository-owned spec under specs/
// or a skill-owned spec under skills/<name>/specs/, mirroring the scoping used
// by specFiles.
func isScopedFSLPath(rel string) bool {
	if !strings.HasSuffix(rel, ".fsl") {
		return false
	}
	if strings.HasPrefix(rel, "specs/") {
		return true
	}
	return strings.Contains(rel, "/specs/")
}

// gitDiffNames returns the paths that differ between rev and the working
// tree, from git diff --name-only run in root, plus untracked (not ignored)
// files so a newly added spec is detected before it is committed.
func gitDiffNames(root, rev string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", rev)
	cmd.Dir = root
	cmd.Env = support.GitEnv()
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff --name-only %s: %w", rev, err)
	}
	names := splitLines(string(out))

	untracked := exec.Command("git", "ls-files", "--others", "--exclude-standard")
	untracked.Dir = root
	untracked.Env = support.GitEnv()
	out, err = untracked.Output()
	if err != nil {
		return nil, fmt.Errorf("git ls-files --others: %w", err)
	}
	names = append(names, splitLines(string(out))...)
	return names, nil
}

func splitLines(text string) []string {
	var names []string
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			names = append(names, line)
		}
	}
	return names
}

// ChangedSpecs returns the repository-relative display paths of every FSL
// specification changed since the git revision baseRev, scoped to specs/ and
// skills/**/specs/ and deduplicated by canonical physical identity (a changed
// symlink exposure and its resolved source collapse to one entry). A broken
// symlink or a path resolving outside the repository is an error, matching
// specFiles. The result is sorted for deterministic output.
func ChangedSpecs(root, baseRev string) ([]string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	absRoot, err = filepath.EvalSymlinks(absRoot)
	if err != nil {
		return nil, err
	}
	names, err := gitDiffNames(absRoot, baseRev)
	if err != nil {
		return nil, err
	}

	var specs []string
	seen := make(map[string]bool, len(names))
	for _, name := range names {
		rel := filepath.ToSlash(name)
		if !isScopedFSLPath(rel) {
			continue
		}
		canon, err := filepath.EvalSymlinks(filepath.Join(absRoot, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("resolve changed FSL spec %q: %w", rel, err)
		}
		if !pathWithin(absRoot, canon) {
			return nil, fmt.Errorf("changed FSL spec %q resolves outside the repository at %q", rel, canon)
		}
		if seen[canon] {
			continue
		}
		seen[canon] = true
		specs = append(specs, rel)
	}
	sort.Strings(specs)
	return specs, nil
}
