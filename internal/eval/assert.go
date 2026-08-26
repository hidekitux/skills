package eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/hidekitux/skills/internal/support"
)

// assertionTimeout bounds every command_run assertion so a hanging fixture
// command fails the scenario instead of stalling the evaluation run.
const assertionTimeout = 2 * time.Minute

// snapshotHashes records the SHA-256 of every listed unchanged file that
// exists in dir at the time of the snapshot. Files that do not exist yet are
// absent from the map and treated as vacuously unchanged: the guard asserts
// that an existing file was not modified, and files_must_not_exist covers
// unwanted creation.
func snapshotHashes(dir string, paths []string) map[string]string {
	hashes := make(map[string]string, len(paths))
	for _, rel := range paths {
		content, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		sum := sha256.Sum256(content)
		hashes[filepath.Clean(rel)] = hex.EncodeToString(sum[:])
	}
	return hashes
}

// evaluateAssertions runs every deterministic expectation against the
// transcript and the sandbox state, returning a list of concrete failures.
// An empty list means the scenario passed.
func evaluateAssertions(ctx context.Context, sc *Scenario, transcript, sandboxDir string, before map[string]string) []string {
	expect := sc.Expectations
	var failures []string

	for _, pattern := range expect.TranscriptMust {
		if !strings.Contains(transcript, pattern) {
			failures = append(failures, fmt.Sprintf("transcript does not contain %q", pattern))
		}
	}
	// TranscriptMustAny lists alternative phrasings; at least one must appear,
	// so wording that varies across models (for example "no mandatory rules"
	// versus "zero mandatory rules") does not produce false negatives.
	if len(expect.TranscriptMustAny) > 0 {
		matched := false
		for _, pattern := range expect.TranscriptMustAny {
			if strings.Contains(transcript, pattern) {
				matched = true
				break
			}
		}
		if !matched {
			failures = append(failures, fmt.Sprintf("transcript contains none of the alternatives %q", expect.TranscriptMustAny))
		}
	}
	for _, pattern := range expect.TranscriptMustNot {
		if strings.Contains(transcript, pattern) {
			failures = append(failures, fmt.Sprintf("transcript contains forbidden %q", pattern))
		}
	}
	for _, rel := range expect.FilesMustExist {
		path := filepath.Join(sandboxDir, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err != nil || info.IsDir() {
			failures = append(failures, fmt.Sprintf("expected file %s does not exist", rel))
		}
	}
	for _, rel := range expect.FilesMustNotExist {
		path := filepath.Join(sandboxDir, filepath.FromSlash(rel))
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			failures = append(failures, fmt.Sprintf("forbidden file %s exists", rel))
		}
	}
	for _, check := range expect.FileContains {
		path := filepath.Join(sandboxDir, filepath.FromSlash(check.Path))
		content, err := os.ReadFile(path)
		if err != nil {
			failures = append(failures, fmt.Sprintf("file %s cannot be read for pattern %q", check.Path, check.Pattern))
			continue
		}
		if !strings.Contains(string(content), check.Pattern) {
			failures = append(failures, fmt.Sprintf("file %s does not contain %q", check.Path, check.Pattern))
		}
	}
	for _, rel := range expect.UnchangedFiles {
		clean := filepath.Clean(rel)
		want, ok := before[clean]
		if !ok {
			continue
		}
		content, err := os.ReadFile(filepath.Join(sandboxDir, filepath.FromSlash(clean)))
		if err != nil {
			failures = append(failures, fmt.Sprintf("unchanged file %s cannot be read", clean))
			continue
		}
		sum := sha256.Sum256(content)
		if got := hex.EncodeToString(sum[:]); got != want {
			failures = append(failures, fmt.Sprintf("out-of-scope file %s was modified", clean))
		}
	}
	for _, check := range expect.CommandRun {
		name := check.Run
		if name == "" {
			continue
		}
		dir := sandboxDir
		if check.Dir != "" {
			dir = filepath.Join(sandboxDir, filepath.FromSlash(check.Dir))
		}
		runCtx, cancel := context.WithTimeout(ctx, assertionTimeout)
		cmd := exec.CommandContext(runCtx, "sh", "-c", name)
		cmd.Dir = dir
		cmd.Env = support.GitEnv()
		output, err := cmd.CombinedOutput()
		cancel()
		exitCode := support.ExitError(err)
		if exitCode != check.Exit {
			failures = append(failures, fmt.Sprintf("command %q in %s exited %d, want %d", name, dir, exitCode, check.Exit))
			if len(output) > 0 {
				trimmed := strings.TrimSpace(string(output))
				if len(trimmed) > 200 {
					trimmed = trimmed[:200] + "..."
				}
				failures = append(failures, "command output: "+trimmed)
			}
		}
	}
	return failures
}
