package check

import (
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// internalImportPrefix is the import-path prefix every internal package shares.
const internalImportPrefix = "github.com/hidekitux/skills/internal/"

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

// packageImports returns the internal packages imported by the Go files
// directly under dir. It reads imports with go/parser and ignores test files,
// so a package that only a test imports is not treated as a module edge.
func packageImports(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	fileSet := token.NewFileSet()
	seen := map[string]bool{}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(fileSet, filepath.Join(dir, name), nil, parser.ImportsOnly)
		if err != nil {
			return nil, err
		}
		for _, spec := range file.Imports {
			path, err := strconv.Unquote(spec.Path.Value)
			if err != nil {
				return nil, err
			}
			if strings.HasPrefix(path, internalImportPrefix) {
				seen[strings.TrimPrefix(path, internalImportPrefix)] = true
			}
		}
	}
	imports := make([]string, 0, len(seen))
	for path := range seen {
		imports = append(imports, path)
	}
	sort.Strings(imports)
	return imports, nil
}

// internalPackages returns the directory names directly below internal/ that
// contain at least one Go file. A test-only package is included so the
// ownership file must still name it, and its test imports stay outside the
// module-edge rule.
func internalPackages(root string) ([]string, error) {
	base := filepath.Join(root, "internal")
	entries, err := os.ReadDir(base)
	if err != nil {
		return nil, err
	}
	packages := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(base, entry.Name()))
		if err != nil {
			return nil, err
		}
		for _, file := range files {
			name := file.Name()
			if !file.IsDir() && strings.HasSuffix(name, ".go") {
				packages = append(packages, entry.Name())
				break
			}
		}
	}
	sort.Strings(packages)
	return packages, nil
}

// CheckModuleBoundaries enforces the internal module ownership recorded in
// workflow/module-ownership.yml and documented in docs/architecture.md. It
// fails when an internal package belongs to no module, when the ownership file
// names a package that no longer exists, or when a package imports a package
// whose module the importing module may not import. It returns 0 on success
// and 1 on any violation.
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
	for _, pkg := range packages {
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
		source, ok := model.moduleOf[pkg]
		if !ok {
			continue
		}
		imports, err := packageImports(filepath.Join(root, "internal", pkg))
		if err != nil {
			fmt.Fprintf(errOut, "module-boundaries check failed: %v\n", err)
			return 1
		}
		for _, imported := range imports {
			target, ok := model.moduleOf[imported]
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
