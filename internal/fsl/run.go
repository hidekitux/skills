// Package fsl wraps the checksum-pinned fslc verifier, ported from
// scripts/fsl/verify-fsl.sh and scripts/fsl/mutate-fsl.sh.
package fsl

import (
	"bytes"
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
	// Resolve the root so the within-repository check compares resolved paths
	// against resolved paths. Otherwise a root reached through a symlink (common
	// on macOS, where /var and /tmp point into /private) would compare an
	// unresolved root against a resolved canonical path and falsely reject every
	// in-repo spec as outside the repository.
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	absRoot, err = filepath.EvalSymlinks(absRoot)
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
	cmd.Env = support.GitEnv()
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

// MutateOptions configures a mutation run.
type MutateOptions struct {
	// ChangedBase, when non-empty, restricts mutation to specs changed since
	// this git revision (ChangedSpecs); a run with no changed specs is a
	// successful no-op so the tied workflow job never goes pending.
	ChangedBase string
	// ReportPath, when non-empty, writes the retained MutationReport here.
	ReportPath string
}

// MutateFSL measures mutation detection for every FSL spec (or only the specs
// changed since opts.ChangedBase). An fslc failure for one spec is recorded as
// an infrastructure error in the report and the remaining specs still run; the
// command still exits non-zero so failures stay visible. On success it exits 0
// and writes the report when opts.ReportPath is set.
func MutateFSL(root string, out, errOut io.Writer, opts MutateOptions) int {
	var specs []string
	var err error
	if opts.ChangedBase != "" {
		specs, err = ChangedSpecs(root, opts.ChangedBase)
	} else {
		specs, err = specFiles(root)
	}
	if err != nil {
		fmt.Fprintf(errOut, "error: %v\n", err)
		return 1
	}
	depthValue := depth()
	if len(specs) == 0 {
		fmt.Fprintln(out, "No FSL specifications selected for mutation.")
		if opts.ReportPath != "" {
			report := MutationReport{Depth: depthValue}
			if err := WriteMutationReport(opts.ReportPath, report); err != nil {
				fmt.Fprintf(errOut, "error: write mutation report: %v\n", err)
				return 1
			}
		}
		return 0
	}

	report := MutationReport{Depth: depthValue}
	failed := 0
	for _, spec := range specs {
		fmt.Fprintf(out, "Mutating %s at depth %s\n", spec, depthValue)
		var buffer bytes.Buffer
		fmt.Fprintf(&buffer, "Mutating %s at depth %s\n", spec, depthValue)
		code := runFslc(io.MultiWriter(out, &buffer), errOut, "mutate", spec, "--depth", depthValue)
		if code != 0 {
			failed++
			if opts.ReportPath != "" {
				report.Specs = append(report.Specs, SpecReport{
					Spec:   spec,
					Status: "error",
					Error:  fmt.Sprintf("fslc exited with status %d", code),
				})
			}
			continue
		}
		if opts.ReportPath != "" {
			parsed, err := parseMutationDocuments(buffer.String())
			if err != nil {
				fmt.Fprintf(errOut, "error: parse mutation output for %s: %v\n", spec, err)
				return 1
			}
			if len(parsed) != 1 {
				fmt.Fprintf(errOut, "error: expected one mutation document for %s, got %d\n", spec, len(parsed))
				return 1
			}
			specReport := parsed[0]
			specReport.Spec = spec
			report.Specs = append(report.Specs, specReport)
		}
	}

	if opts.ReportPath != "" {
		report.Totals.Specs = len(report.Specs)
		for _, spec := range report.Specs {
			report.Totals.Killed += spec.Killed
			report.Totals.Survived += spec.Survived
			report.Totals.Invalid += spec.Invalid
			if spec.Status == "error" {
				report.Totals.InfrastructureError++
			}
		}
		if err := WriteMutationReport(opts.ReportPath, report); err != nil {
			fmt.Fprintf(errOut, "error: write mutation report: %v\n", err)
			return 1
		}
		fmt.Fprintf(out, "Mutation report written to %s.\n", opts.ReportPath)
	}
	if failed > 0 {
		fmt.Fprintf(errOut, "Mutation infrastructure error: %d spec(s) failed to mutate.\n", failed)
		return 1
	}
	fmt.Fprintf(out, "Mutated %d FSL spec(s).\n", len(specs))
	return 0
}
