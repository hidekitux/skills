package validate

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hidekitux/skills/internal/support"
)

// scriptTestMapping is the decoded form of SCRIPT_TESTS.toml. The cmds table
// maps a cmd/ entrypoint, and the scripts table maps a retained executable
// script, to the Go test package and the named tests that exercise it.
type scriptTestMapping struct {
	Cmds    map[string]testEvidence `toml:"cmds"`
	Scripts map[string]testEvidence `toml:"scripts"`
}

// testEvidence names a Go test package and the test functions in it that
// exercise one command or script.
type testEvidence struct {
	Package string   `toml:"package"`
	Tests   []string `toml:"tests"`
}

var testFunctionPattern = regexp.MustCompile(`(?m)^func (Test\w+)\(t \*testing\.T\)`)

// testFunctions returns the names of the test functions declared in the
// _test.go files of dir, or false when dir holds no test file.
func testFunctions(dir string) (map[string]bool, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, false
	}
	names := map[string]bool{}
	found := false
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		found = true
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		for _, match := range testFunctionPattern.FindAllStringSubmatch(string(data), -1) {
			names[match[1]] = true
		}
	}
	return names, found
}

// evidenceErrors reports why entry does not show a test that exercises name:
// no named test, a package without test files, or a named test the package
// does not declare.
func evidenceErrors(root, name string, entry testEvidence) []string {
	if len(entry.Tests) == 0 {
		return []string{fmt.Sprintf("%s: names no test", name)}
	}
	functions, ok := testFunctions(filepath.Join(root, filepath.FromSlash(entry.Package)))
	if !ok {
		return []string{fmt.Sprintf("%s: package %s has no test file", name, entry.Package)}
	}
	var errors []string
	for _, test := range entry.Tests {
		if !functions[test] {
			errors = append(errors, fmt.Sprintf("%s: test %s is not declared in %s", name, test, entry.Package))
		}
	}
	return errors
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
		mapping.Cmds = map[string]testEvidence{}
	}
	if mapping.Scripts == nil {
		mapping.Scripts = map[string]testEvidence{}
	}

	errors := []string{}
	commands := cmdEntrypoints(root)
	for _, command := range commands {
		entry, ok := mapping.Cmds[command]
		if !ok {
			errors = append(errors, fmt.Sprintf("%s: missing representative Go test package", command))
			continue
		}
		errors = append(errors, evidenceErrors(root, command, entry)...)
	}
	for _, command := range sortedEntryKeys(mapping.Cmds) {
		if !containsCommand(commands, command) {
			errors = append(errors, fmt.Sprintf("%s: mapping has no repository command", command))
		}
	}

	scripts := scriptFiles(root)
	for _, script := range scripts {
		entry, ok := mapping.Scripts[script]
		if !ok {
			errors = append(errors, fmt.Sprintf("%s: missing representative test", script))
			continue
		}
		errors = append(errors, evidenceErrors(root, script, entry)...)
	}
	for _, script := range sortedEntryKeys(mapping.Scripts) {
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

func sortedEntryKeys(m map[string]testEvidence) []string {
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
