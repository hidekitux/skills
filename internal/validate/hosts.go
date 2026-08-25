package validate

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"github.com/hidekitux/skills/internal/support"
)

// skillNames returns the sorted publishable skill directory names under
// root/skills.
func skillNames(root string) []string {
	var names []string
	_ = filepath.WalkDir(filepath.Join(root, "skills"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}
		names = append(names, filepath.Base(filepath.Dir(path)))
		return nil
	})
	sort.Strings(names)
	return names
}

// CheckHosts validates that every published skill installs for each
// supported host using the gh CLI, returning the process exit code.
func CheckHosts(root string, out, errOut io.Writer) int {
	names := skillNames(root)
	if len(names) == 0 {
		fmt.Fprintln(errOut, "No publishable skills found.")
		return 1
	}
	for _, host := range []string{"codex", "claude-code"} {
		installRoot := ".agents/skills"
		if host == "claude-code" {
			installRoot = ".claude/skills"
		}
		hostRoot, err := os.MkdirTemp("", "skills-host-validation.")
		if err != nil {
			fmt.Fprintf(errOut, "cannot create host validation directory: %v\n", err)
			return 1
		}
		defer os.RemoveAll(hostRoot)

		if _, err := support.OutputIn(hostRoot, "git", "init", "--quiet"); err != nil {
			fmt.Fprintf(errOut, "cannot initialize host test repository: %v\n", err)
			return 1
		}
		if _, err := support.OutputIn(hostRoot, "gh", "skill", "install", root, "--from-local", "--all", "--agent", host, "--scope", "project"); err != nil {
			fmt.Fprintf(errOut, "gh skill install failed for %s: %v\n", host, err)
			return 1
		}

		for _, skillName := range names {
			skillFile := filepath.Join(hostRoot, installRoot, skillName, "SKILL.md")
			if info, statErr := os.Stat(skillFile); statErr != nil || info.IsDir() {
				fmt.Fprintf(errOut, "%s did not install %s at %s.\n", host, skillName, installRoot)
				return 1
			}
		}
		fmt.Fprintf(out, "%s installation validated: %d skill(s) in %s.\n", host, len(names), installRoot)
	}
	return 0
}
