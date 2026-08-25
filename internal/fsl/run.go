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

// specFiles returns the repository-relative display paths of every unique
// logical FSL specification, deduplicated by canonical physical identity and
// ordered specs/ before skills/**/specs/ (both in lexical order).
//
// Skill-owned specs under skills/<name>/specs/ are also exposed through
// repository-level symbolic links under specs/<name>/. Both paths resolve to
// the same physical file, so each logical source is returned exactly once and
// the first-seen path is used as the stable display path that verification and
// mutation output reports.
//
// A candidate that cannot be resolved (a broken symlink) or that resolves
// outside the repository (an incorrectly targeted link) is returned as an
// error so broken FSL links still fail repository validation.
func specFiles(root string) ([]string, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	var candidates []string
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
				candidates = append(candidates, filepath.ToSlash(rel))
			}
			return nil
		})
	}
	collect("specs", func(rel string) bool { return true })
	collect("skills", func(rel string) bool {
		return strings.Contains(rel, string(filepath.Separator)+"specs"+string(filepath.Separator))
	})

	var specs []string
	seen := make(map[string]bool, len(candidates))
	for _, rel := range candidates {
		canon, err := filepath.EvalSymlinks(filepath.Join(absRoot, filepath.FromSlash(rel)))
		if err != nil {
			return nil, fmt.Errorf("resolve FSL spec %q: %w", rel, err)
		}
		if !pathWithin(absRoot, canon) {
			return nil, fmt.Errorf("FSL spec %q resolves outside the repository at %q", rel, canon)
		}
		if seen[canon] {
			continue
		}
		seen[canon] = true
		specs = append(specs, rel)
	}
	return specs, nil
}

// pathWithin reports whether path is root or a descendant of root.
func pathWithin(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
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
	specs, err := specFiles(root)
	if err != nil {
		fmt.Fprintf(errOut, "error: %v\n", err)
		return 1
	}
	depthValue := depth()
	if len(specs) == 0 {
		fmt.Fprintln(out, "No repository-owned or skill-owned FSL specs found.")
		return 0
	}
	for _, spec := range specs {
		fmt.Fprintf(out, "Checking %s\n", spec)
		if code := runFslc(out, errOut, "check", spec); code != 0 {
			return code
		}
		fmt.Fprintf(out, "Verifying %s at depth %s\n", spec, depthValue)
		if code := runFslc(out, errOut, "verify", spec, "--depth", depthValue); code != 0 {
			return code
		}
	}
	fmt.Fprintf(out, "Verified %d FSL spec(s).\n", len(specs))
	return 0
}

// MutateFSL measures mutation detection for every FSL spec.
func MutateFSL(root string, out, errOut io.Writer) int {
	specs, err := specFiles(root)
	if err != nil {
		fmt.Fprintf(errOut, "error: %v\n", err)
		return 1
	}
	depthValue := depth()
	if len(specs) == 0 {
		fmt.Fprintln(out, "No repository-owned or skill-owned FSL specs found.")
		return 0
	}
	for _, spec := range specs {
		fmt.Fprintf(out, "Mutating %s at depth %s\n", spec, depthValue)
		if code := runFslc(out, errOut, "mutate", spec, "--depth", depthValue); code != 0 {
			return code
		}
	}
	fmt.Fprintf(out, "Mutated %d FSL spec(s).\n", len(specs))
	return 0
}
