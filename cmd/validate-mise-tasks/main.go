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
	"check": true, "diagnose": true, "evaluate": true, "generate": true,
	"install": true, "lint": true, "mutate": true, "publish": true,
	"setup": true, "test": true, "validate": true, "verify": true,
}

var retiredTasks = []string{
	"evaluate", "fsl:install", "lint", "mutate-fsl", "mutate-fsl:changed",
	"release:publish", "setup", "test", "validate", "validate-skill-creator",
	"verify-fsl", "verify-release", "worktree:diagnose",
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
	for _, marker := range []string{"mise run ", "depends = [", "depends = [\""} {
		for start := 0; ; {
			relative := strings.Index(content[start:], marker+retired)
			if relative < 0 {
				break
			}
			index := start + relative + len(marker) + len(retired)
			if index == len(content) || !isTaskCharacter(content[index]) {
				return true
			}
			start = index
		}
	}
	if strings.Contains(content, "[tasks."+retired+"]") || strings.Contains(content, "[tasks.\""+retired+"\"]") {
		return true
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
