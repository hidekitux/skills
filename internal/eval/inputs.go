package eval

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// skillInputPaths are read from the skill root: `gh skill install --all`
// installs every skill into the sandbox, so any skill can change the
// evaluated behavior.
var skillInputPaths = []string{"skills"}

// harnessInputPaths are the repository-root inputs every evaluation shares:
// the rubric the reviewer scores against and the Go harness with its pinned
// toolchain.
var harnessInputPaths = []string{"evaluations/rubric.md", "cmd", "internal", "go.mod", "go.sum", "mise.toml"}

// InputDigest returns the SHA-256 over the files that can change the
// evaluated behavior of skill: every skill under skillRoot, the skill's
// scenarios and the fixtures they stage, and the shared harness inputs
// under root. Promotion compares a record's digest with the digest at the
// checked-out revision, so a commit outside these files keeps the record
// fresh (docs/evaluation.md).
func InputDigest(root, skillRoot, skill string) (string, error) {
	paths, err := skillScenarioInputPaths(root, skill)
	if err != nil {
		return "", err
	}
	hash := sha256.New()
	for _, source := range []struct {
		root  string
		paths []string
	}{
		{skillRoot, skillInputPaths},
		{root, append(paths, harnessInputPaths...)},
	} {
		files, err := listInputFiles(source.root, source.paths)
		if err != nil {
			return "", err
		}
		for _, file := range files {
			content, err := os.ReadFile(filepath.Join(source.root, filepath.FromSlash(file)))
			if err != nil {
				return "", err
			}
			hash.Write([]byte(file))
			hash.Write([]byte{0})
			hash.Write(content)
			hash.Write([]byte{0})
		}
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

// skillScenarioInputPaths returns the skill's scenario directory and the
// fixture directories its scenarios stage.
func skillScenarioInputPaths(root, skill string) ([]string, error) {
	scenarios, err := LoadAllScenarios(root)
	if err != nil {
		return nil, err
	}
	paths := []string{scenarioBase + "/" + skill}
	seen := map[string]bool{}
	for _, sc := range scenarios {
		if sc.Skill != skill || sc.Fixture == "" || seen[sc.Fixture] {
			continue
		}
		seen[sc.Fixture] = true
		paths = append(paths, fixtureBase+"/"+sc.Fixture)
	}
	return paths, nil
}

// listInputFiles returns the sorted slash-separated files below paths. In a
// Git worktree it lists tracked and untracked files that are not ignored, so
// local build output does not change the digest; outside one it walks the
// file system.
func listInputFiles(root string, paths []string) ([]string, error) {
	args := append([]string{"ls-files", "-z", "--cached", "--others", "--exclude-standard", "--"}, paths...)
	out, err := gitPort.Output(context.Background(), root, args...)
	var files []string
	if err == nil {
		for _, file := range strings.Split(out.Stdout, "\x00") {
			if file == "" {
				continue
			}
			// A tracked file deleted in the working tree is not an input.
			if info, statErr := os.Stat(filepath.Join(root, filepath.FromSlash(file))); statErr != nil || info.IsDir() {
				continue
			}
			files = append(files, file)
		}
	} else {
		for _, path := range paths {
			walkErr := filepath.WalkDir(filepath.Join(root, filepath.FromSlash(path)), func(current string, entry fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				if entry.IsDir() {
					return nil
				}
				relative, err := filepath.Rel(root, current)
				if err != nil {
					return err
				}
				files = append(files, filepath.ToSlash(relative))
				return nil
			})
			if walkErr != nil && !os.IsNotExist(walkErr) {
				return nil, fmt.Errorf("cannot list evaluation inputs under %s: %w", path, walkErr)
			}
		}
	}
	sort.Strings(files)
	return files, nil
}
