package check

import (
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

// packageImports returns the repository packages imported by the Go files
// directly under dir, split into the internal packages and the cmd packages it
// imports. Both lists carry the path below their prefix. It reads imports with
// go/parser and ignores test files, so a package that only a test imports is
// not treated as a module edge.
func packageImports(dir string) (internal, command []string, err error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, nil, err
	}
	fileSet := token.NewFileSet()
	seenInternal := map[string]bool{}
	seenCommand := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fileSet, filepath.Join(dir, name), nil, parser.ImportsOnly)
		if err != nil {
			return nil, nil, err
		}
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return nil, nil, err
			}
			switch {
			case strings.HasPrefix(path, internalImportPrefix):
				seenInternal[strings.TrimPrefix(path, internalImportPrefix)] = true
			case strings.HasPrefix(path, commandImportPrefix):
				seenCommand[strings.TrimPrefix(path, commandImportPrefix)] = true
			}
		}
	}
	return sortedKeys(seenInternal), sortedKeys(seenCommand), nil
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
// below a top-level directory. A test-only package is included so the
// ownership file must still name its top-level directory, and its test imports
// stay outside the module-edge rule.
func internalPackages(root string) ([]string, error) {
	base := filepath.Join(root, "internal")
	packages := []string{}
	err := filepath.WalkDir(base, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() || path == base {
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

// CheckModuleBoundaries enforces the internal module ownership recorded in
// workflow/module-ownership.yml and documented in docs/architecture.md. It
// fails when an internal package belongs to no module, when the ownership file
// names a package that no longer exists, when a package imports a package
// whose module the importing module may not import, and when any internal
// package imports the composition root under cmd/. It returns 0 on success and
// 1 on any violation.
//
// The check reads imports with go/parser rather than building the packages, so
// an import behind a build tag the parser skips is not observed.
func CheckModuleBoundaries(root string, out, errOut io.Writer) int {
	model, err := readOwnershipModel(filepath.Join(root, "workflow", "module-ownership.yml"))
	if err != nil {
		fmt.Fprintf(errOut, "module-boundaries check failed: %v\n", err)
		return 1
	}
	packages, err := internalPackages(root)
	if err != nil {
		fmt.Fprintf(errOut, "module-boundaries check failed: %v\n", err)
		return 1
	}
	findings := []string{}
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
		imports, commands, err := packageImports(filepath.Join(root, "internal", filepath.FromSlash(pkg)))
		if err != nil {
			fmt.Fprintf(errOut, "module-boundaries check failed: %v\n", err)
			return 1
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
	if len(findings) > 0 {
		sort.Strings(findings)
		for _, finding := range findings {
			fmt.Fprintln(errOut, finding)
		}
		fmt.Fprintf(errOut, "module-boundaries check failed: %d violation(s).\n", len(findings))
		return 1
	}
	fmt.Fprintf(out, "module boundaries valid: %d packages in %d modules, %d allowed module edges.\n",
		len(packages), len(model.order), edges)
	return 0
}
