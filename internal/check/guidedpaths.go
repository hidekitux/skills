// Package check implement the guided-path contract for the repository's
// published guidance.
package check

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// guidedPathAllowlist records repository-relative path references in published
// guidance that intentionally do not resolve to a tracked repository file.
// Each entry states why it is legitimate so a future migration that removes the
// reason can drop the entry. The check reports any other candidate that does not
// resolve, so a removed command or template referenced as current fails the
// repository validation instead of silently drifting.
var guidedPathAllowlist = map[string]string{
	// bootstrap-project documents the generated project's committed validator
	// location; the shipped template source resolves, this destination does
	// not exist in this repository by design.
	"scripts/lint/validate-commit-message.py": "generated destination in a boostrapped project",
}

var (
	// goRunCmdRE matches the supported mise/go run invocation style and yields
	// the top-level command directory under cmd/.
	goRunCmdRE = regexp.MustCompile(`go run \./cmd/([A-Za-z0-9_.-]+)`)
	// guidedPathRE matches repository-owned executable/template path tokens under
	// cmd/, scripts/, or templates/. The leading [^...] guard prevents matching
	// the "cmd" inside "./cmd/", which goRunCmdRE already handles.
	guidedPathRE = regexp.MustCompile(`(?:^|[^A-Za-z0-9_.-])((?:cmd|scripts|templates)/[A-Za-z0-9_./-]+)`)
)

// guidedPathFile returns true when the repository-relative path should be
// scanned for embedded command/template references: published skill guidance
// (SKILL.md and references/), repository-owned specifications (specs/), and the
// root current-state guidance.
func guidedPathFile(rel string) bool {
	switch {
	case rel == "AGENTS.md" || rel == "README.md":
		return true
	case strings.HasPrefix(rel, "specs/") && strings.HasSuffix(rel, ".fsl") && !strings.Contains(rel, "/skills/"):
		return true
	case strings.HasPrefix(rel, "skills/") && isPublishedGuidance(rel):
		return true
	default:
		return false
	}
}

func isPublishedGuidance(rel string) bool {
	if strings.Contains(rel, "/agents/") {
		return false
	}
	if strings.HasSuffix(rel, "/SKILL.md") {
		return true
	}
	if strings.Contains(rel, "/references/") && strings.HasSuffix(rel, ".md") {
		return true
	}
	if strings.Contains(rel, "/specs/") && strings.HasSuffix(rel, ".fsl") {
		return true
	}
	return false
}

// skillRoot returns the directory, under root, that owns the scanned file: the
// nearest ancestor that contains a SKILL.md. It returns "" when the file is not
// inside a published skill, in which case only repository-root resolution
// applies.
func skillRoot(root, rel string) string {
	dir := filepath.Dir(filepath.Join(root, rel))
	for {
		if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err == nil {
			relDir, rerr := filepath.Rel(root, dir)
			if rerr == nil {
				return relDir
			}
			return ""
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// guidedPathFiles returns the repository-relative paths to scan.
func guidedPathFiles(root string) []string {
	var files []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil || !guidedPathFile(rel) {
			return nil
		}
		files = append(files, rel)
		return nil
	})
	return files
}

// resolveGuidedPath reports whether the candidate resolves to an existing file
// or directory under the skill root (when inside a skill) or the repository
// root.
func resolveGuidedPath(root, rel, candidate string) bool {
	c := strings.TrimPrefix(candidate, "./")
	if s := skillRoot(root, rel); s != "" {
		if _, err := os.Stat(filepath.Join(root, s, c)); err == nil {
			return true
		}
	}
	if _, err := os.Stat(filepath.Join(root, c)); err == nil {
		return true
	}
	return false
}

// CheckGuidedPaths scans published guidance and specifications for references to
// repository-owned command and template paths and fails when one does not
// resolve to a tracked repository file. It returns 0 on success or 1 when a
// stale reference is found.
func CheckGuidedPaths(root string, out, errOut io.Writer) int {
	var findings []string
	for _, rel := range guidedPathFiles(root) {
		content, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			continue
		}
		text := string(content)
		seen := map[string]bool{}
		for _, m := range goRunCmdRE.FindAllStringSubmatch(text, -1) {
			seen["cmd/"+m[1]] = true
		}
		for _, m := range guidedPathRE.FindAllStringSubmatch(text, -1) {
			seen[m[1]] = true
		}
		for candidate := range seen {
			if resolveGuidedPath(root, rel, candidate) {
				continue
			}
			if reason, ok := guidedPathAllowlist[candidate]; ok {
				_ = reason
				continue
			}
			findings = append(findings, fmt.Sprintf("%s: %q does not resolve to a tracked repository file", rel, candidate))
		}
	}
	if len(findings) > 0 {
		fmt.Fprintln(errOut, "Guided-path check failed:")
		for _, finding := range findings {
			fmt.Fprintf(errOut, "- %s\n", finding)
		}
		return 1
	}
	fmt.Fprintln(out, "Guided-path check passed: published guidance references resolve to tracked repository files.")
	return 0
}
