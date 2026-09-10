package context

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

type Case struct {
	ID            string   `yaml:"id"`
	Skill         string   `yaml:"skill"`
	Signals       Signals  `yaml:"signals"`
	Include       []string `yaml:"include"`
	Exclude       []string `yaml:"exclude"`
	ForceOverflow bool     `yaml:"force_overflow"`
}

type fixtureCounter func(string) (int, error)

func (c fixtureCounter) Count(value string) (int, error) { return c(value) }

// CheckCases validates representative selection and overflow fixtures without
// loading the network-backed tokenizer. Exact token counts are validated by
// compile-context; this check covers deterministic routing and stop behavior.
func CheckCases(root string, out, errOut io.Writer) int {
	base := filepath.Join(root, "workflow", "context-fixtures")
	paths := []string{}
	_ = filepath.WalkDir(base, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".yaml") {
			paths = append(paths, path)
		}
		return nil
	})
	sort.Strings(paths)
	if len(paths) == 0 {
		fmt.Fprintln(errOut, "context fixture check failed: no YAML cases")
		return 1
	}
	failures := []string{}
	for _, fixturePath := range paths {
		data, err := os.ReadFile(fixturePath)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", fixturePath, err))
			continue
		}
		var item Case
		if err := yaml.Unmarshal(data, &item); err != nil {
			failures = append(failures, fmt.Sprintf("%s: decode: %v", fixturePath, err))
			continue
		}
		counter := fixtureCounter(func(string) (int, error) {
			if item.ForceOverflow {
				return 100000, nil
			}
			return 1, nil
		})
		compiled, compileErr := (Compiler{Root: root, Counter: counter}).Compile(item.Skill, item.Signals)
		if item.ForceOverflow {
			if !isRequiredOverflow(compileErr) || compiled.Manifest.Overflow == nil {
				failures = append(failures, fmt.Sprintf("%s: expected required overflow, got %v", item.ID, compileErr))
			}
			continue
		}
		if compileErr != nil {
			failures = append(failures, fmt.Sprintf("%s: compile: %v", item.ID, compileErr))
			continue
		}
		selected := map[string]bool{}
		for _, module := range compiled.Modules {
			selected[module.ID] = true
		}
		for _, expected := range item.Include {
			if !selected[expected] {
				failures = append(failures, fmt.Sprintf("%s: expected %s", item.ID, expected))
			}
		}
		for _, excluded := range item.Exclude {
			if selected[excluded] {
				failures = append(failures, fmt.Sprintf("%s: did not expect %s", item.ID, excluded))
			}
		}
	}
	if len(failures) > 0 {
		for _, failure := range failures {
			fmt.Fprintln(errOut, failure)
		}
		return 1
	}
	fmt.Fprintf(out, "context fixture check passed: %d case(s).\n", len(paths))
	return 0
}

func isRequiredOverflow(err error) bool {
	return err != nil && strings.Contains(err.Error(), ErrRequiredOverflow.Error())
}
