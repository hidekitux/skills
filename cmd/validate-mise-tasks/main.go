// Command validate-mise-tasks validates the repository's canonical mise task
// names and rejects references to retired task names.
package main

import (
	"bufio"

	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var taskHeader = regexp.MustCompile(`^\[tasks\.(?:"([^"]+)"|([^]]+))\]$`)
var taskName = regexp.MustCompile(`^[a-z][a-z0-9]*:[a-z][a-z0-9-]*$`)

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
	fmt.Println("mise task names and references are valid")
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
	for _, retired := range retiredTasks {
		if found, path := findReference(root, retired); found {
			errs = append(errs, fmt.Errorf("retired task %q is referenced by %s", retired, path))
		}
	}
	sort.Slice(errs, func(i, j int) bool { return errs[i].Error() < errs[j].Error() })
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

func findReference(root, retired string) (bool, string) {
	var found string
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" || entry.Name() == ".mise" || entry.Name() == "__pycache__" {
				return filepath.SkipDir
			}
			return nil
		}
		validatorDir := filepath.Join(root, "cmd", "validate-mise-tasks")
		if filepath.Dir(path) == validatorDir {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr == nil && taskReference(string(data), retired) {
			found = path
		}
		return nil
	})
	return found != "", found
}
