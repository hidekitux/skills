package check

import (
	"errors"
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// internalImportPrefix is the import-path prefix every internal package shares.
const internalImportPrefix = "github.com/hidekitux/skills/internal/"

// commandImportPrefix is the import-path prefix of the composition root. No
// internal package may import it.
const commandImportPrefix = "github.com/hidekitux/skills/cmd/"

// ownershipModule is one module entry of workflow/module-ownership.yml.
type ownershipModule struct {
	ID        string   `yaml:"id"`
	Owns      string   `yaml:"owns"`
	Packages  []string `yaml:"packages"`
	MayImport []string `yaml:"may_import"`
}

// ownershipModel is the parsed ownership file: the module of each internal
// package and the module edges the model allows.
type ownershipModel struct {
	moduleOf  map[string]string
	mayImport map[string]map[string]bool
	order     []string
}

// readOwnershipModel parses workflow/module-ownership.yml into the module map
// and the allowed-edge map. It rejects a package listed by more than one
// module and a may_import entry that names no module.
func readOwnershipModel(path string) (*ownershipModel, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc struct {
		SchemaVersion int               `yaml:"schema_version"`
		Modules       []ownershipModule `yaml:"modules"`
	}
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("cannot parse module-ownership.yml: %w", err)
	}
	if doc.SchemaVersion != 1 {
		return nil, fmt.Errorf("unsupported schema_version %d, want 1", doc.SchemaVersion)
	}
	if len(doc.Modules) == 0 {
		return nil, fmt.Errorf("module-ownership.yml lists no module")
	}
	model := &ownershipModel{
		moduleOf:  map[string]string{},
		mayImport: map[string]map[string]bool{},
	}
	known := map[string]bool{}
	for _, module := range doc.Modules {
		if module.ID == "" {
			return nil, fmt.Errorf("module-ownership.yml has a module without an id")
		}
		if known[module.ID] {
			return nil, fmt.Errorf("module %q is declared more than once", module.ID)
		}
		known[module.ID] = true
		model.order = append(model.order, module.ID)
		for _, pkg := range module.Packages {
			if owner, ok := model.moduleOf[pkg]; ok {
				return nil, fmt.Errorf("package %q is owned by both %q and %q", pkg, owner, module.ID)
			}
			model.moduleOf[pkg] = module.ID
		}
		allowed := map[string]bool{}
		for _, target := range module.MayImport {
			allowed[target] = true
		}
		model.mayImport[module.ID] = allowed
	}
	for _, module := range doc.Modules {
		for _, target := range module.MayImport {
			if !known[target] {
				return nil, fmt.Errorf("module %q may_import names unknown module %q", module.ID, target)
			}
			if target == module.ID {
				return nil, fmt.Errorf("module %q lists itself in may_import", module.ID)
			}
		}
	}
	return model, nil
}

// providerStandardImports names the standard-library packages that start an
// external process or send a network request. Only the provider module may
// import them, so every external operation has one owner.
var providerStandardImports = []string{"os/exec", "net/http"}

// packageImports returns the repository packages imported by the Go files
// directly under dir, split into the internal packages, the cmd packages, and
// the external-operation standard packages it imports. The internal and
// command lists carry the path below their prefix. It reads imports with
// go/parser and ignores test files, so a package that only a test imports is
// not treated as a module edge.
func packageImports(dir string) (internal, command, external []string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, nil, err
	}
	fileSet := token.NewFileSet()
	seenInternal := map[string]bool{}
	seenCommand := map[string]bool{}
	seenExternal := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fileSet, filepath.Join(dir, name), nil, parser.ImportsOnly)
		if err != nil {
			return nil, nil, nil, err
		}
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return nil, nil, nil, err
			}
			switch {
			case strings.HasPrefix(path, internalImportPrefix):
				seenInternal[strings.TrimPrefix(path, internalImportPrefix)] = true
			case strings.HasPrefix(path, commandImportPrefix):
				seenCommand[strings.TrimPrefix(path, commandImportPrefix)] = true
			}
			for _, external := range providerStandardImports {
				if path == external {
					seenExternal[path] = true
				}
			}
		}
	}
	return sortedKeys(seenInternal), sortedKeys(seenCommand), sortedKeys(seenExternal), nil
}

// sortedKeys returns the keys of set in a stable order.
func sortedKeys(set map[string]bool) []string {
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// internalPackages returns every directory below internal/ that contains at
// least one Go file, as a slash-separated path relative to internal/. A nested
// package such as support/util is included, so a forbidden import cannot hide
// below a top-level directory. A directory the Go build ignores is skipped
// with its whole subtree. A test-only package is included so the
// ownership file must still name its top-level directory, and its test imports
// stay outside the module-edge rule.
func internalPackages(root string) ([]string, error) {
	return goPackagesBelow(filepath.Join(root, "internal"))
}

// goPackagesBelow returns every directory below base that contains at least one
// Go file, as a slash-separated path relative to base. A directory the Go build
// ignores is skipped with its whole subtree.
func goPackagesBelow(base string) ([]string, error) {
	packages := []string{}
	err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			return nil
		}
		if path != base && !buildableDirectory(entry.Name()) {
			return fs.SkipDir
		}
		if path == base {
			return nil
		}
		files, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, file := range files {
			if !file.IsDir() && strings.HasSuffix(file.Name(), ".go") {
				relative, err := filepath.Rel(base, path)
				if err != nil {
					return err
				}
				packages = append(packages, filepath.ToSlash(relative))
				break
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(packages)
	return packages, nil
}

// commandPackages returns every directory below cmd/ that contains at least
// one Go file, as a slash-separated path relative to cmd/. It uses the same
// walk rule as internalPackages, so a nested command package is included and a
// directory the Go build ignores is skipped with its whole subtree. A
// repository without a cmd/ directory yields no package rather than an error.
func commandPackages(root string) ([]string, error) {
	base := filepath.Join(root, "cmd")
	if _, err := os.Stat(base); errors.Is(err, fs.ErrNotExist) {
		return []string{}, nil
	}
	return goPackagesBelow(base)
}

// buildableDirectory reports whether the Go build considers a directory with
// this name. The build ignores testdata and any name beginning with an
// underscore or a dot, so a Go file below one of them is test data rather than
// a package and carries no module edge.
func buildableDirectory(name string) bool {
	return name != "testdata" && !strings.HasPrefix(name, "_") && !strings.HasPrefix(name, ".")
}

// owningModule returns the module that owns pkg. A nested package belongs to
// the module that owns its top-level directory, so the ownership file lists
// only top-level directories.
func owningModule(model *ownershipModel, pkg string) (string, bool) {
	top := pkg
	if index := strings.Index(pkg, "/"); index >= 0 {
		top = pkg[:index]
	}
	module, ok := model.moduleOf[top]
	return module, ok
}

// topLevelPackages returns the top-level directory of every package in
// packages, without repetition.
func topLevelPackages(packages []string) []string {
	seen := map[string]bool{}
	for _, pkg := range packages {
		top := pkg
		if index := strings.Index(pkg, "/"); index >= 0 {
			top = pkg[:index]
		}
		seen[top] = true
	}
	return sortedKeys(seen)
}

// recordPath is the architecture record that explains the ownership file. The
// two must describe the same modules and the same packages.
const recordPath = "docs/architecture.md"

// recordSection is the heading of the table in the architecture record that
// carries the module identifiers and their packages.
const recordSection = "## Module ownership"

// recordOnlyModule is the one module the architecture record names and the
// ownership file does not. cmd/** is the composition root: it is not a
// directory below internal/, so no ownership entry can own it, and the check
// forbids the reverse import instead.
const recordOnlyModule = "composition"

// readRecordedModules parses the Module ownership table of the architecture
// record into the packages each module row lists. It returns an error when the
// section or its table is absent, so a rewrite of the record fails the check
// rather than silently disabling the comparison.
func readRecordedModules(path string) (map[string][]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(string(content), "\n")
	start := -1
	for index, line := range lines {
		if strings.TrimSpace(line) == recordSection {
			start = index + 1
			break
		}
	}
	if start < 0 {
		return nil, fmt.Errorf("%s has no %q section", recordPath, recordSection)
	}
	recorded := map[string][]string{}
	for _, line := range lines[start:] {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "## ") {
			break
		}
		if !strings.HasPrefix(trimmed, "|") {
			continue
		}
		cells := tableCells(trimmed)
		if len(cells) < 3 {
			continue
		}
		module := unquoteCell(cells[0])
		if module == "" || module == "Module" || strings.HasPrefix(module, "---") {
			continue
		}
		packages := []string{}
		for _, item := range strings.Split(cells[2], ",") {
			if name := unquoteCell(item); name != "" {
				packages = append(packages, name)
			}
		}
		recorded[module] = packages
	}
	if len(recorded) == 0 {
		return nil, fmt.Errorf("the %q section of %s records no module", recordSection, recordPath)
	}
	return recorded, nil
}

// tableCells splits one Markdown table row into its cells, without the
// leading and trailing pipe.
func tableCells(row string) []string {
	cells := strings.Split(strings.Trim(row, "|"), "|")
	for index, cell := range cells {
		cells[index] = strings.TrimSpace(cell)
	}
	return cells
}

// unquoteCell returns the cell text without its surrounding backticks and
// spaces, so `foundation` and foundation read the same.
func unquoteCell(cell string) string {
	return strings.Trim(strings.TrimSpace(cell), "` ")
}

// recordFindings compares the ownership model with the architecture record. It
// reports a module or a package that one of the two carries and the other does
// not, so a module split recorded in only one place fails the check.
func recordFindings(model *ownershipModel, recorded map[string][]string) []string {
	findings := []string{}
	ownedBy := map[string][]string{}
	for pkg, module := range model.moduleOf {
		ownedBy[module] = append(ownedBy[module], pkg)
	}
	for _, module := range model.order {
		packages, ok := recorded[module]
		if !ok {
			findings = append(findings, fmt.Sprintf(
				"module %s is declared in workflow/module-ownership.yml and is not recorded in the %q table of %s",
				module, recordSection, recordPath))
			continue
		}
		present := map[string]bool{}
		for _, name := range packages {
			present[name] = true
		}
		names := ownedBy[module]
		sort.Strings(names)
		for _, pkg := range names {
			if !present[pkg] {
				findings = append(findings, fmt.Sprintf(
					"workflow/module-ownership.yml gives internal/%s to module %s, whose row in the %q table of %s does not list it",
					pkg, module, recordSection, recordPath))
			}
		}
	}
	for _, module := range sortedKeys(boolSet(recorded)) {
		if module == recordOnlyModule {
			continue
		}
		if _, ok := model.mayImport[module]; !ok {
			findings = append(findings, fmt.Sprintf(
				"the %q table of %s records module %s, which workflow/module-ownership.yml does not declare",
				recordSection, recordPath, module))
		}
	}
	return findings
}

// boolSet returns the keys of recorded as a set, so sortedKeys can order them.
func boolSet(recorded map[string][]string) map[string]bool {
	set := map[string]bool{}
	for key := range recorded {
		set[key] = true
	}
	return set
}

// externalOperationExempt names the packages that may import an
// external-operation standard package. internal/provider owns every port and
// adapter. internal/support keeps one git invocation for repository-root
// resolution, because the foundation module cannot import the provider module,
// which imports foundation.
var externalOperationExempt = map[string]bool{"provider": true, "support": true}

// CheckModuleBoundaries enforces the internal module ownership recorded in
// workflow/module-ownership.yml and documented in docs/architecture.md. It
// fails when an internal package belongs to no module, when the ownership file
// names a package that no longer exists, when the ownership file and the
// Module ownership table of docs/architecture.md carry different modules or
// different packages, when a package imports a package whose module the
// importing module may not import, when any internal package imports the
// composition root under cmd/, and when a package outside the provider module,
// including a package under cmd/, imports os/exec or net/http. It returns 0 on
// success and 1 on any violation.
//
// The check reads imports with go/parser rather than building the packages, so
// an import behind a build tag the parser skips is not observed.
func CheckModuleBoundaries(root string, out, errOut io.Writer) int {
	model, err := readOwnershipModel(filepath.Join(root, "workflow", "module-ownership.yml"))
	if err != nil {
		fmt.Fprintf(errOut, "module-boundaries check failed: %v\n", err)
		return 1
	}
	recorded, err := readRecordedModules(filepath.Join(root, filepath.FromSlash(recordPath)))
	if err != nil {
		fmt.Fprintf(errOut, "module-boundaries check failed: %v\n", err)
		return 1
	}
	packages, err := internalPackages(root)
	if err != nil {
		fmt.Fprintf(errOut, "module-boundaries check failed: %v\n", err)
		return 1
	}
	commands, err := commandPackages(root)
	if err != nil {
		fmt.Fprintf(errOut, "module-boundaries check failed: %v\n", err)
		return 1
	}
	findings := recordFindings(model, recorded)
	present := map[string]bool{}
	for _, pkg := range topLevelPackages(packages) {
		present[pkg] = true
		if _, ok := model.moduleOf[pkg]; !ok {
			findings = append(findings, fmt.Sprintf("package internal/%s belongs to no module in workflow/module-ownership.yml", pkg))
		}
	}
	listed := make([]string, 0, len(model.moduleOf))
	for pkg := range model.moduleOf {
		listed = append(listed, pkg)
	}
	sort.Strings(listed)
	for _, pkg := range listed {
		if !present[pkg] {
			findings = append(findings, fmt.Sprintf("workflow/module-ownership.yml lists internal/%s, which does not exist", pkg))
		}
	}
	edges := 0
	for _, pkg := range packages {
		source, ok := owningModule(model, pkg)
		if !ok {
			continue
		}
		imports, commands, externals, err := packageImports(filepath.Join(root, "internal", filepath.FromSlash(pkg)))
		if err != nil {
			fmt.Fprintf(errOut, "module-boundaries check failed: %v\n", err)
			return 1
		}
		for _, external := range externals {
			if externalOperationExempt[pkg] {
				continue
			}
			findings = append(findings, fmt.Sprintf(
				"internal/%s (%s) imports %s; only the provider module starts a process or sends a request",
				pkg, source, external))
		}
		for _, command := range commands {
			findings = append(findings, fmt.Sprintf(
				"forbidden reverse dependency: internal/%s (%s) imports cmd/%s (composition); no module may import composition",
				pkg, source, command))
		}
		for _, imported := range imports {
			target, ok := owningModule(model, imported)
			if !ok {
				findings = append(findings, fmt.Sprintf("internal/%s imports internal/%s, which belongs to no module", pkg, imported))
				continue
			}
			if target == source {
				continue
			}
			edges++
			if !model.mayImport[source][target] {
				findings = append(findings, fmt.Sprintf(
					"forbidden reverse dependency: internal/%s (%s) imports internal/%s (%s); %s may not import %s",
					pkg, source, imported, target, source, target))
			}
		}
	}
	for _, command := range commands {
		_, _, externals, err := packageImports(filepath.Join(root, "cmd", filepath.FromSlash(command)))
		if err != nil {
			fmt.Fprintf(errOut, "module-boundaries check failed: %v\n", err)
			return 1
		}
		for _, external := range externals {
			findings = append(findings, fmt.Sprintf(
				"cmd/%s (composition) imports %s; only the provider module starts a process or sends a request",
				command, external))
		}
	}
	if len(findings) > 0 {
		sort.Strings(findings)
		for _, finding := range findings {
			fmt.Fprintln(errOut, finding)
		}
		fmt.Fprintf(errOut, "module-boundaries check failed: %d violation(s).\n", len(findings))
		return 1
	}
	fmt.Fprintf(out, "module boundaries valid: %d packages in %d modules, %d allowed module edges, %d command packages scanned.\n",
		len(packages), len(model.order), edges, len(commands))
	return 0
}
