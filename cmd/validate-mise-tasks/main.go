// Command validate-mise-tasks validates the repository's canonical mise task
// names, rejects references to retired task names, and requires the canonical
// task inventory in docs/mise-tasks.md to list exactly the declared tasks.
package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hidekitux/skills/internal/provider"
)

var gitPort = provider.NewGit(provider.OSRunner{})

var taskHeader = regexp.MustCompile(`^\[tasks\.(?:"([^"]+)"|([^]]+))\]$`)
var taskName = regexp.MustCompile(`^[a-z][a-z0-9]*:[a-z][a-z0-9-]*$`)
var inventoryTaskName = regexp.MustCompile("`([^`]+)`")

// inventoryPath is the document, relative to the repository root, whose
// canonical task inventory must list every task that mise.toml declares.
const inventoryPath = "docs/mise-tasks.md"

// inventoryHeading starts the canonical task inventory table in inventoryPath.
const inventoryHeading = "## Canonical task inventory"

var allowedVerbs = map[string]bool{
	"check": true, "evaluate": true, "format": true, "generate": true,
	"install": true, "lint": true, "mutate": true, "publish": true,
	"setup": true, "test": true, "validate": true, "verify": true,
}

// retiredTasks are task names that must not be declared or invoked again. A
// prose mention stays legal so a migration note can name what was removed;
// only an executable reference or a redeclaration fails.
var retiredTasks = []string{
	"diagnose:worktree", "evaluate", "fsl:install", "lint", "mutate-fsl",
	"mutate-fsl:changed", "release:publish", "setup", "test", "validate",
	"validate-skill-creator", "verify-fsl", "verify-release", "worktree:diagnose",
}

func main() {
	root := flag.String("root", ".", "repository root")
	tasks := flag.String("tasks", "mise.toml", "mise configuration path relative to root")
	flag.Parse()

	errs := validate(*root, filepath.Join(*root, *tasks))
	for _, err := range errs {
		fmt.Fprintln(os.Stderr, "error:", err)
	}
	if len(errs) > 0 {
		os.Exit(1)
	}
	fmt.Println("mise task names, references, and the canonical task inventory are valid")
}

func validate(root, misePath string) []error {
	var errs []error
	tasks, err := declaredTasks(misePath)
	if err != nil {
		return []error{err}
	}
	for _, task := range tasks {
		parts := strings.SplitN(task, ":", 2)
		if !taskName.MatchString(task) || !allowedVerbs[parts[0]] {
			errs = append(errs, fmt.Errorf("task %q must use a one-word verb category and hyphenated task name", task))
		}
	}
	errs = append(errs, inventoryErrors(root, tasks)...)
	candidates := candidateFiles(root)
	for _, retired := range retiredTasks {
		if found, path := findReference(root, candidates, retired); found {
			errs = append(errs, fmt.Errorf("retired task %q is referenced by %s", retired, path))
		}
	}
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
	return errs
}

// inventoryTasks reads the canonical task inventory table and returns the
// category of each listed task. It fails when the document or its inventory
// heading is missing, so the comparison cannot pass by skipping the table.
func inventoryTasks(root string) (map[string]string, error) {
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(inventoryPath)))
	if err != nil {
		return nil, fmt.Errorf("read task inventory %s: %w", inventoryPath, err)
	}
	listed := map[string]string{}
	inSection := false
	found := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			inSection = trimmed == inventoryHeading
			found = found || inSection
			continue
		}
		if !inSection || !strings.HasPrefix(trimmed, "|") {
			continue
		}
		cells := strings.Split(strings.Trim(trimmed, "|"), "|")
		if len(cells) < 2 {
			continue
		}
		category := strings.TrimSpace(cells[0])
		if category == "Category" || strings.Trim(category, "-: ") == "" {
			continue
		}
		for _, match := range inventoryTaskName.FindAllStringSubmatch(cells[1], -1) {
			listed[match[1]] = category
		}
	}
	if !found {
		return nil, fmt.Errorf("%s has no %q section", inventoryPath, inventoryHeading)
	}
	return listed, nil
}

// inventoryErrors reports each declared task missing from the canonical task
// inventory, each listed task that mise.toml does not declare, and each listed
// task whose row category differs from its verb.
func inventoryErrors(root string, tasks []string) []error {
	listed, err := inventoryTasks(root)
	if err != nil {
		return []error{err}
	}
	var errs []error
	declared := map[string]bool{}
	for _, task := range tasks {
		declared[task] = true
		category, ok := listed[task]
		if !ok {
			errs = append(errs, fmt.Errorf("task %q is declared in mise.toml but missing from the canonical task inventory in %s", task, inventoryPath))
			continue
		}
		if verb, _, _ := strings.Cut(task, ":"); category != verb {
			errs = append(errs, fmt.Errorf("task %q is listed under category %q in %s instead of %q", task, category, inventoryPath, verb))
		}
	}
	for task := range listed {
		if !declared[task] {
			errs = append(errs, fmt.Errorf("task %q is listed in the canonical task inventory in %s but not declared in mise.toml", task, inventoryPath))
		}
	}
	return errs
}

func declaredTasks(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read mise configuration: %w", err)
	}
	defer file.Close()
	var tasks []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		match := taskHeader.FindStringSubmatch(strings.TrimSpace(scanner.Text()))
		if match != nil {
			if match[1] != "" {
				tasks = append(tasks, match[1])
			} else {
				tasks = append(tasks, strings.TrimSpace(match[2]))
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read mise configuration: %w", err)
	}
	return tasks, nil
}

func taskReference(content, retired string) bool {
	// Match against whitespace-normalized text so an invocation that Markdown
	// wrapped across two lines is caught like an inline one. Documentation in
	// this repository wraps prose, so `mise run` and the task name regularly
	// land on different lines.
	normalized := strings.Join(strings.Fields(content), " ")
	if invocationReference(normalized, retired) || dependsReference(normalized, retired) {
		return true
	}
	if strings.Contains(normalized, "[tasks."+retired+"]") || strings.Contains(normalized, "[tasks.\""+retired+"\"]") {
		return true
	}
	return false
}

// invocationReference reports whether the retired task is invoked through
// `mise run`. The trailing character must not extend the task name, so
// `mise run setup:all` is not a reference to the retired `setup`.
func invocationReference(content, retired string) bool {
	const marker = "mise run "
	for start := 0; ; {
		relative := strings.Index(content[start:], marker+retired)
		if relative < 0 {
			return false
		}
		index := start + relative + len(marker) + len(retired)
		if index == len(content) || !isTaskCharacter(content[index]) {
			return true
		}
		start = index
	}
}

// dependsReference reports whether the retired task appears as any element of a
// depends array. Matching each element rather than the text right after the
// opening bracket keeps a reintroduced dependency detectable wherever it is
// listed, not only in first position.
func dependsReference(content, retired string) bool {
	for _, marker := range []string{"depends = [", "depends=["} {
		for start := 0; ; {
			relative := strings.Index(content[start:], marker)
			if relative < 0 {
				break
			}
			open := start + relative + len(marker)
			end := strings.Index(content[open:], "]")
			if end < 0 {
				break
			}
			for _, element := range strings.Split(content[open:open+end], ",") {
				if strings.Trim(strings.TrimSpace(element), "\"'") == retired {
					return true
				}
			}
			start = open + end
		}
	}
	return false
}

func isTaskCharacter(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9' || value == '-' || value == ':'
}

func findReference(root string, candidates []string, retired string) (bool, string) {
	validatorDir := filepath.Join(root, "cmd", "validate-mise-tasks")
	for _, path := range candidates {
		if filepath.Dir(path) == validatorDir {
			continue
		}
		data, err := os.ReadFile(path)
		if err == nil && taskReference(string(data), retired) {
			return true, path
		}
	}
	return false, ""
}

// candidateFiles returns the tracked and untracked non-ignored files under
// root, so a Git-ignored copy of the repository, such as an agent worktree,
// cannot fail the check. It falls back to a plain walk when root is not a Git
// work tree.
func candidateFiles(root string) []string {
	result, err := gitPort.Output(context.Background(), root, "ls-files", "--cached", "--others", "--exclude-standard", "-z")
	if err == nil {
		var files []string
		for _, item := range strings.Split(result.Stdout, "\x00") {
			if item != "" {
				files = append(files, filepath.Join(root, item))
			}
		}
		sort.Strings(files)
		return files
	}
	var files []string
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".mise" || entry.Name() == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}
		files = append(files, path)
		return nil
	})
	return files
}
