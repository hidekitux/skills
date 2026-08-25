// Package fsl wraps the checksum-pinned fslc verifier, ported from
// scripts/fsl/verify-fsl.sh and scripts/fsl/mutate-fsl.sh.
package fsl

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hidekitux/skills/internal/support"
)

// cacheRoot returns the shared fslc cache directory outside the repository.
func cacheRoot() string {
	if t := os.Getenv("RUNNER_TEMP"); t != "" {
		return t + "/skills-fslc"
	}
	if t := os.Getenv("TMPDIR"); t != "" {
		return t + "/skills-fslc"
	}
	return "/tmp/skills-fslc"
}

func binPath() string {
	if b := os.Getenv("FSLC_BIN_DIR"); b != "" {
		return b
	}
	return cacheRoot() + "/bin"
}

func depth() string {
	return support.EnvOr("FSL_DEPTH", "8")
}

// specFiles returns the .fsl specs under specs/ then skills/**/specs/ in
// repository-relative lexical order, matching the two find commands of the
// former shell wrappers.
func specFiles(root string) []string {
	var specs []string
	collect := func(base string, match func(rel string) bool) {
		_ = filepath.WalkDir(filepath.Join(root, base), func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".fsl") {
				return nil
			}
			rel, _ := filepath.Rel(root, path)
			if match(rel) {
				specs = append(specs, filepath.ToSlash(rel))
			}
			return nil
		})
	}
	collect("specs", func(rel string) bool { return true })
	collect("skills", func(rel string) bool {
		return strings.Contains(rel, string(filepath.Separator)+"specs"+string(filepath.Separator))
	})
	return specs
}

// runFslc invokes the pinned fslc binary directly at its install path. The
// binary is located by binPath rather than by name so resolution does not
// depend on the parent process's inherited PATH (Go resolves an unqualified
// executable name via the parent env, not cmd.Env).
func runFslc(out, errOut io.Writer, args ...string) int {
	cmd := exec.Command(filepath.Join(binPath(), "fslc"), args...)
	cmd.Stdout = out
	cmd.Stderr = errOut
	cmd.Env = os.Environ()
	return support.ExitError(cmd.Run())
}

// VerifyFSL checks and verifies every FSL spec with the pinned fslc binary.
func VerifyFSL(root string, out, errOut io.Writer) int {
	specs := specFiles(root)
	depthValue := depth()
	found := 0
	for _, spec := range specs {
		found = 1
		fmt.Fprintf(out, "Checking %s\n", spec)
		if code := runFslc(out, errOut, "check", spec); code != 0 {
			return code
		}
		fmt.Fprintf(out, "Verifying %s at depth %s\n", spec, depthValue)
		if code := runFslc(out, errOut, "verify", spec, "--depth", depthValue); code != 0 {
			return code
		}
	}
	if found == 0 {
		fmt.Fprintln(out, "No repository-owned or skill-owned FSL specs found.")
	}
	return 0
}

// MutateFSL measures mutation detection for every FSL spec.
func MutateFSL(root string, out, errOut io.Writer) int {
	specs := specFiles(root)
	depthValue := depth()
	found := 0
	for _, spec := range specs {
		found = 1
		fmt.Fprintf(out, "Mutating %s at depth %s\n", spec, depthValue)
		if code := runFslc(out, errOut, "mutate", spec, "--depth", depthValue); code != 0 {
			return code
		}
	}
	if found == 0 {
		fmt.Fprintln(out, "No repository-owned or skill-owned FSL specs found.")
	}
	return 0
}
