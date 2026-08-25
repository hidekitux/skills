package validate

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hidekitux/skills/internal/support"
)

// scriptTestMapping is the decoded form of SCRIPT_TESTS.toml. The cmds table
// maps a cmd/ entrypoint to a Go test package; the scripts table maps a
// retained executable script to an existing test home (a test file or a Go
// test package).
type scriptTestMapping struct {
	Cmds    map[string]string `toml:"cmds"`
	Scripts map[string]string `toml:"scripts"`
}

// testHomeExists reports whether value names an existing file or a directory
// containing at least one _test.go file.
func testHomeExists(root, value string) bool {
	path := filepath.Join(root, value)
	if info, err := os.Stat(path); err == nil && !info.IsDir() {
		return true
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), "_test.go") {
			return true
		}
	}
	return false
}

// cmdEntrypoints returns every cmd/<name> directory that contains a main.go
// file, relative to root.
func cmdEntrypoints(root string) []string {
	cmdRoot := filepath.Join(root, "cmd")
	var commands []string
	_ = filepath.WalkDir(cmdRoot, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "main.go" {
			return nil
		}
		name := filepath.Base(filepath.Dir(path))
		if filepath.Dir(filepath.Dir(path)) != cmdRoot {
			return nil
		}
		commands = append(commands, "cmd/"+name)
		return nil
	})
	sort.Strings(commands)
	return commands
}

// scriptFiles returns every file under root/scripts, excluding bytecode
// caches, relative to root.
func scriptFiles(root string) []string {
	var files []string
	_ = filepath.WalkDir(filepath.Join(root, "scripts"), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if d.Name() == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr == nil {
			files = append(files, rel)
		}
		return nil
	})
	sort.Strings(files)
	return files
}

// CheckScriptTests requires each cmd/ entrypoint and each executable script
// under scripts/ to name an existing representative test, returning 0 on
// success or 1 when the registry has gaps or orphans.
func CheckScriptTests(root string, out, errOut io.Writer) int {
	var mapping scriptTestMapping
	if err := support.LoadTOMLFile(filepath.Join(root, "SCRIPT_TESTS.toml"), &mapping); err != nil {
		fmt.Fprintf(errOut, "Script-test mapping check failed: %v\n", err)
		return 1
	}
	if mapping.Cmds == nil {
		mapping.Cmds = map[string]string{}
	}
	if mapping.Scripts == nil {
		mapping.Scripts = map[string]string{}
	}

	errors := []string{}
	commands := cmdEntrypoints(root)
	for _, command := range commands {
		value, ok := mapping.Cmds[command]
		if !ok || !testHomeExists(root, value) {
			errors = append(errors, fmt.Sprintf("%s: missing representative Go test package", command))
		}
	}
	for _, command := range sortedStringKeys(mapping.Cmds) {
		if !containsCommand(commands, command) {
			errors = append(errors, fmt.Sprintf("%s: mapping has no repository command", command))
		}
	}

	scripts := scriptFiles(root)
	for _, script := range scripts {
		value, ok := mapping.Scripts[script]
		if !ok || !testHomeExists(root, value) {
			errors = append(errors, fmt.Sprintf("%s: missing representative test", script))
		}
	}
	for _, script := range sortedStringKeys(mapping.Scripts) {
		if !containsCommand(scripts, script) {
			errors = append(errors, fmt.Sprintf("%s: mapping has no repository script", script))
		}
	}

	if len(errors) > 0 {
		fmt.Fprintln(errOut, "Script-test mapping check failed:")
		for _, error := range errors {
			fmt.Fprintf(errOut, "- %s\n", error)
		}
		return 1
	}
	fmt.Fprintf(out, "Script-test mapping check passed: %d command(s) and %d script(s) mapped.\n", len(commands), len(scripts))
	return 0
}

func sortedStringKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func containsCommand(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}
