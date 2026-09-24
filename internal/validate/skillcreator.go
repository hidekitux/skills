package validate

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"

	"github.com/hidekitux/skills/internal/support"
)

// skillDirsUnder returns the sorted publishable skill directories under
// root/skills, one per SKILL.md.
func skillDirsUnder(root string) []string {
	dirs := map[string]bool{}
	_ = filepath.WalkDir(filepath.Join(root, "skills"), func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "SKILL.md" {
			return nil
		}
		dirs[filepath.Dir(path)] = true
		return nil
	})
	var list []string
	for dir := range dirs {
		list = append(list, dir)
	}
	sort.Strings(list)
	return list
}

// uvCacheDir returns a temporary UV cache root outside the repository.
func uvCacheDir() string {
	if t := os.Getenv("RUNNER_TEMP"); t != "" {
		return t + "/skills-validate-skill-creator-uv-cache"
	}
	if t := os.Getenv("TMPDIR"); t != "" {
		return t + "/skills-validate-skill-creator-uv-cache"
	}
	return "/tmp/skills-validate-skill-creator-uv-cache"
}

// ValidateSkillCreator validates every published skill with the external
// Codex skill-creator validator, returning the process exit code. This is a
// Go adapter around an external validator that retains its pinned Python
// execution without representing a repository-owned Python implementation.
func ValidateSkillCreator(root string, out, errOut io.Writer) int {
	skillCreatorRoot := os.Getenv("SKILL_CREATOR_ROOT")
	if skillCreatorRoot == "" {
		codexHome := os.Getenv("CODEX_HOME")
		if codexHome == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				home = "."
			}
			codexHome = filepath.Join(home, ".codex")
		}
		skillCreatorRoot = filepath.Join(codexHome, "skills", ".system", "skill-creator")
	}
	quickValidate := filepath.Join(skillCreatorRoot, "scripts", "quick_validate.py")
	if _, err := os.Stat(quickValidate); err != nil {
		fmt.Fprintln(errOut, "skill-creator validator not found; set SKILL_CREATOR_ROOT to its skill directory.")
		return 2
	}

	cacheDir := uvCacheDir()
	for _, skillDir := range skillDirsUnder(root) {
		cmd := exec.Command("uv", "run", "--with", "pyyaml==6.0.3", "python", quickValidate, skillDir)
		cmd.Stdout = out
		cmd.Stderr = errOut
		cmd.Env = append(os.Environ(), "UV_CACHE_DIR="+cacheDir)
		if code := support.ExitError(cmd.Run()); code != 0 {
			return code
		}
	}
	return 0
}
