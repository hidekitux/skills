package check

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hidekitux/skills/internal/support"
)

// attestation is a reviewed license entry in TOOL_LICENSES.toml.
type attestation struct {
	License string `toml:"license"`
	Source  string `toml:"source"`
}

// directGoModules parses the direct (non-indirect) require entries out of a
// go.mod file without invoking the Go toolchain.
func directGoModules(goModPath string) ([]string, error) {
	text, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}
	var modules []string
	inBlock := false
	for _, line := range strings.Split(string(text), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		switch {
		case line == "require (":
			inBlock = true
			continue
		case line == ")":
			inBlock = false
			continue
		case strings.HasPrefix(line, "require "):
			fields := strings.Fields(strings.TrimPrefix(line, "require "))
			if len(fields) >= 1 {
				modules = append(modules, fields[0])
			}
			continue
		case inBlock && !strings.Contains(line, "// indirect"):
			fields := strings.Fields(line)
			if len(fields) >= 1 {
				modules = append(modules, fields[0])
			}
		}
	}
	return modules, nil
}

// CheckToolLicenses requires a reviewed license attestation for every
// mise-managed tool and every direct go.mod dependency, returning 0 on
// success or 1 when an attestation is missing or invalid.
func CheckToolLicenses(root string, out, errOut io.Writer) int {
	var mise struct {
		Tools map[string]string `toml:"tools"`
	}
	var registry struct {
		Tools     map[string]attestation `toml:"tools"`
		GoModules map[string]attestation `toml:"go_modules"`
	}
	misePath := filepath.Join(root, "mise.toml")
	registryPath := filepath.Join(root, "TOOL_LICENSES.toml")
	if err := support.LoadTOMLFile(misePath, &mise); err != nil {
		fmt.Fprintf(errOut, "Tool-license check failed: %v\n", err)
		return 1
	}
	if err := support.LoadTOMLFile(registryPath, &registry); err != nil {
		fmt.Fprintf(errOut, "Tool-license check failed: %v\n", err)
		return 1
	}
	if mise.Tools == nil {
		mise.Tools = map[string]string{}
	}

	errors := []string{}
	checkAttestations := func(attestations map[string]attestation, required map[string]bool, missing, nonEmpty, https, orphan string) {
		for tool := range required {
			entry, ok := attestations[tool]
			if !ok {
				errors = append(errors, fmt.Sprintf("%s: %s", tool, missing))
				continue
			}
			if strings.TrimSpace(entry.License) == "" {
				errors = append(errors, fmt.Sprintf("%s: %s", tool, nonEmpty))
			}
			if !strings.HasPrefix(entry.Source, "https://") {
				errors = append(errors, fmt.Sprintf("%s: %s", tool, https))
			}
		}
		extra := make([]string, 0, len(attestations))
		for tool := range attestations {
			if !required[tool] {
				extra = append(extra, tool)
			}
		}
		sort.Strings(extra)
		for _, tool := range extra {
			errors = append(errors, fmt.Sprintf("%s: %s", tool, orphan))
		}
	}

	tools := make(map[string]bool, len(mise.Tools))
	for tool := range mise.Tools {
		tools[tool] = true
	}
	checkAttestations(
		registry.Tools,
		tools,
		"missing license attestation",
		"license must be non-empty",
		"source must be a public https URL",
		"attestation has no mise tool",
	)

	modules := []string{}
	goModPath := filepath.Join(root, "go.mod")
	if _, err := os.Stat(goModPath); err == nil {
		parsed, err := directGoModules(goModPath)
		if err != nil {
			fmt.Fprintf(errOut, "Tool-license check failed: %v\n", err)
			return 1
		}
		modules = parsed
	}
	requiredModules := make(map[string]bool, len(modules))
	for _, module := range modules {
		requiredModules[module] = true
	}
	checkAttestations(
		registry.GoModules,
		requiredModules,
		"missing Go module license attestation",
		"license must be non-empty",
		"source must be a public https URL",
		"attestation has no go.mod direct dependency",
	)

	if len(errors) > 0 {
		fmt.Fprintln(errOut, "Tool-license check failed:")
		for _, error := range errors {
			fmt.Fprintf(errOut, "- %s\n", error)
		}
		return 1
	}
	fmt.Fprintf(out, "Tool-license check passed: %d tool(s) and %d Go module(s) attested.\n", len(tools), len(modules))
	return 0
}
