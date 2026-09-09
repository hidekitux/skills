package release

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// PublishRelease runs the repository's complete release gates, then publishes
// a verified release tag. It is the Go port of publish-release.sh.
func PublishRelease(args []string, out, errOut io.Writer) int {
	return publishRelease(args, out, errOut, osReleaseCommandRunner{})
}

func publishRelease(args []string, out, errOut io.Writer, runner releaseCommandRunner) int {
	if len(args) != 1 {
		fmt.Fprintln(errOut, "Usage: mise run publish:release -- vX.Y.Z")
		return 2
	}
	tag := args[0]

	if code := runner.stream("mise", out, errOut, "run", "validate:all"); code != 0 {
		return code
	}

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
	if _, err := os.Stat(filepath.Join(skillCreatorRoot, "scripts", "quick_validate.py")); err == nil {
		if code := runner.stream("mise", out, errOut, "run", "validate:skill-creator"); code != 0 {
			return code
		}
	} else {
		fmt.Fprintln(errOut, "skill-creator validator unavailable; skipping Codex-specific evidence.")
	}

	if code := runner.stream("mise", out, errOut, "run", "verify:release", "--", tag); code != 0 {
		return code
	}
	return runner.stream("gh", out, errOut, "skill", "publish", "--tag", tag)
}
