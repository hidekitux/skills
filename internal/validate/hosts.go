package validate

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/hidekitux/skills/internal/discover"
	"github.com/hidekitux/skills/internal/support"
)

// skillNames returns the sorted, de-duplicated publishable skill directory
// names under root/skills using the canonical recursive discovery contract, so
// host validation matches repository validation and local registration.
func skillNames(root string) []string {
	return discover.Names(root)
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

		if _, err := support.GitOutputIn(hostRoot, "init", "--quiet"); err != nil {
			fmt.Fprintf(errOut, "cannot initialize host test repository: %v\n", err)
			return 1
		}
		ghCmd := exec.Command("gh", "skill", "install", root, "--from-local", "--all", "--agent", host, "--scope", "project")
		ghCmd.Dir = hostRoot
		ghCmd.Env = support.GitEnv()
		ghCmd.Stdout = out
		ghCmd.Stderr = errOut
		if err := ghCmd.Run(); err != nil {
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
