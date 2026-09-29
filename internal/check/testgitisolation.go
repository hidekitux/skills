package check

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// CheckTestGitIsolation rejects a Go test that starts git without an isolated
// environment. A test run from inside a Git hook inherits GIT_DIR and similar
// variables, so an unisolated git init or git commit writes to the calling
// repository instead of the test's temporary one (Issue #372). The check walks
// every _test.go file below root, skipping hidden directories and testdata. It
// reports each exec.Command or exec.CommandContext call whose command argument
// is the literal "git" unless the call is assigned to a variable and a later
// statement in the same block, before any reassignment, sets that variable's
// Env to a config-isolated environment. That value calls support.GitEnv or
// support.WithoutGitEnvironment and also sets GIT_CONFIG_GLOBAL and
// GIT_CONFIG_SYSTEM, directly or through a function in the same file that
// returns such a value, so a fixture commit reads neither the developer's
// global signing configuration nor the system one (Issue #401). It returns 0
// on success and 1 when findings exist.
//
// The rule is syntactic: a git binary passed through a variable or started by
// a script is not observed. test:go runs the suite against a sentinel
// repository for those cases (see RunGoTestsWithSentinel).
func CheckTestGitIsolation(root string, out, errOut io.Writer) int {
	findings := []string{}
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if path != root && (strings.HasPrefix(name, ".") || name == "testdata") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		fileFindings, err := unisolatedGitCalls(path)
		if err != nil {
			findings = append(findings, fmt.Sprintf("%s: cannot parse: %v", relPath(root, path), err))
			return nil
		}
		for _, line := range fileFindings {
			findings = append(findings, fmt.Sprintf("%s:%d: git command without an Env that calls support.GitEnv or support.WithoutGitEnvironment and sets GIT_CONFIG_GLOBAL and GIT_CONFIG_SYSTEM, directly or through a helper in the same file, in the same block before any reassignment", relPath(root, path), line))
		}
		return nil
	})
	if walkErr != nil {
		fmt.Fprintf(errOut, "test-git-isolation check failed: %v\n", walkErr)
		return 1
	}
	if len(findings) > 0 {
		sort.Strings(findings)
		for _, finding := range findings {
			fmt.Fprintln(errOut, finding)
		}
		return 1
	}
	fmt.Fprintln(out, "Test Git-isolation check passed: every literal git command in a Go test sets a config-isolated Env from support.GitEnv().")
	return 0
}

// isolatingFunctions names the support functions whose result is an
// environment without GIT_* variables.
var isolatingFunctions = map[string]bool{"GitEnv": true, "WithoutGitEnvironment": true}

// unisolatedGitCalls returns the line of every unisolated git call in the Go
// file at path.
func unisolatedGitCalls(path string) ([]int, error) {
	fileSet := token.NewFileSet()
	file, err := parser.ParseFile(fileSet, path, nil, 0)
	if err != nil {
		return nil, err
	}
	execName := execImportName(file)
	if execName == "" {
		return nil, nil
	}
	helpers := isolatingHelpers(file)
	isolated := map[*ast.CallExpr]bool{}
	ast.Inspect(file, func(node ast.Node) bool {
		switch block := node.(type) {
		case *ast.BlockStmt:
			markIsolated(block.List, execName, helpers, isolated)
		case *ast.CaseClause:
			markIsolated(block.Body, execName, helpers, isolated)
		case *ast.CommClause:
			markIsolated(block.Body, execName, helpers, isolated)
		}
		return true
	})
	lines := []int{}
	ast.Inspect(file, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok && isLiteralGitCommand(call, execName) && !isolated[call] {
			lines = append(lines, fileSet.Position(call.Pos()).Line)
		}
		return true
	})
	return lines, nil
}

// execImportName returns the name the file uses for os/exec, or "" when the
// file does not import it.
func execImportName(file *ast.File) string {
	for _, spec := range file.Imports {
		if importPath, err := strconv.Unquote(spec.Path.Value); err != nil || importPath != "os/exec" {
			continue
		}
		if spec.Name != nil {
			if spec.Name.Name == "_" {
				return ""
			}
			return spec.Name.Name
		}
		return "exec"
	}
	return ""
}

// Helper levels record what a top-level function in a test file returns.
const (
	helperStripsGitEnvironment = 1 // an environment without GIT_* variables
	helperIsolatesConfig       = 2 // a config-isolated environment
)

// isolatingHelpers returns the level of each top-level function in file that
// returns an environment without GIT_* variables or a config-isolated one,
// such as fixtureGitEnvironment in internal/environment/provision_test.go. It
// repeats until no level rises, so a helper that builds on another helper's
// result also counts.
func isolatingHelpers(file *ast.File) map[string]int {
	helpers := map[string]int{}
	for raised := true; raised; {
		raised = false
		for _, decl := range file.Decls {
			function, ok := decl.(*ast.FuncDecl)
			if !ok || function.Recv != nil || function.Body == nil || helpers[function.Name.Name] == helperIsolatesConfig {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				if _, ok := node.(*ast.FuncLit); ok {
					return false
				}
				statement, ok := node.(*ast.ReturnStmt)
				if !ok {
					return true
				}
				for _, result := range statement.Results {
					level := 0
					switch {
					case isIsolatingValue(result, helpers):
						level = helperIsolatesConfig
					case stripsGitEnvironment(result, helpers):
						level = helperStripsGitEnvironment
					}
					if level > helpers[function.Name.Name] {
						helpers[function.Name.Name] = level
						raised = true
					}
				}
				return true
			})
		}
	}
	return helpers
}

// configIsolationPrefixes are the variable assignments a config-isolated
// environment must contain, so Git reads no global or system configuration.
var configIsolationPrefixes = []string{"GIT_CONFIG_GLOBAL=", "GIT_CONFIG_SYSTEM="}

// isIsolatingValue reports whether expr is a config-isolated environment: it
// calls an isolating helper, or it calls support.GitEnv or
// support.WithoutGitEnvironment and contains a string literal for each of
// configIsolationPrefixes. append(os.Environ(), ...) and nil are not, because
// both keep an inherited GIT_DIR, and support.GitEnv() alone is not, because
// it keeps the developer's global configuration.
func isIsolatingValue(expr ast.Expr, helpers map[string]int) bool {
	if callsIsolatingHelper(expr, helpers) {
		return true
	}
	return stripsGitEnvironment(expr, helpers) && setsConfigIsolation(expr)
}

// callsIsolatingHelper reports whether expr calls a same-file helper that
// already returns a config-isolated environment.
func callsIsolatingHelper(expr ast.Expr, helpers map[string]int) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		if call, ok := node.(*ast.CallExpr); ok {
			if function, ok := call.Fun.(*ast.Ident); ok && helpers[function.Name] == helperIsolatesConfig {
				found = true
			}
		}
		return !found
	})
	return found
}

// setsConfigIsolation reports whether expr contains a string literal that
// starts with each of configIsolationPrefixes.
func setsConfigIsolation(expr ast.Expr) bool {
	seen := map[string]bool{}
	ast.Inspect(expr, func(node ast.Node) bool {
		literal, ok := node.(*ast.BasicLit)
		if !ok || literal.Kind != token.STRING {
			return true
		}
		value, err := strconv.Unquote(literal.Value)
		if err != nil {
			return true
		}
		for _, prefix := range configIsolationPrefixes {
			if strings.HasPrefix(value, prefix) {
				seen[prefix] = true
			}
		}
		return true
	})
	return len(seen) == len(configIsolationPrefixes)
}

// stripsGitEnvironment reports whether expr calls support.GitEnv,
// support.WithoutGitEnvironment, or a same-file helper that returns their
// result, all of which remove the inherited GIT_* variables.
func stripsGitEnvironment(expr ast.Expr, helpers map[string]int) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return !found
		}
		switch function := call.Fun.(type) {
		case *ast.Ident:
			found = found || isolatingFunctions[function.Name] || helpers[function.Name] >= helperStripsGitEnvironment
		case *ast.SelectorExpr:
			found = found || isolatingFunctions[function.Sel.Name]
		}
		return !found
	})
	return found
}

// markIsolated records each literal git command assigned to a variable in
// statements when a later statement of the same list assigns that variable's
// Env an isolating value before any statement reassigns the variable.
func markIsolated(statements []ast.Stmt, execName string, helpers map[string]int, isolated map[*ast.CallExpr]bool) {
	for index, statement := range statements {
		name, call := gitCommandAssignment(statement, execName)
		if call == nil {
			continue
		}
		for _, later := range statements[index+1:] {
			assignment, ok := later.(*ast.AssignStmt)
			if !ok {
				continue
			}
			if isEnvAssignment(assignment, name) {
				if isIsolatingValue(assignment.Rhs[0], helpers) {
					isolated[call] = true
				}
				break
			}
			if assignsName(assignment, name) {
				break
			}
		}
	}
}

// gitCommandAssignment returns the variable and the call when statement
// assigns exactly one literal git command to exactly one variable.
func gitCommandAssignment(statement ast.Stmt, execName string) (string, *ast.CallExpr) {
	switch statement := statement.(type) {
	case *ast.AssignStmt:
		if len(statement.Lhs) != 1 || len(statement.Rhs) != 1 {
			return "", nil
		}
		target, ok := statement.Lhs[0].(*ast.Ident)
		call, isCall := statement.Rhs[0].(*ast.CallExpr)
		if ok && isCall && isLiteralGitCommand(call, execName) {
			return target.Name, call
		}
	case *ast.DeclStmt:
		declaration, ok := statement.Decl.(*ast.GenDecl)
		if !ok || len(declaration.Specs) != 1 {
			return "", nil
		}
		spec, ok := declaration.Specs[0].(*ast.ValueSpec)
		if !ok || len(spec.Names) != 1 || len(spec.Values) != 1 {
			return "", nil
		}
		if call, ok := spec.Values[0].(*ast.CallExpr); ok && isLiteralGitCommand(call, execName) {
			return spec.Names[0].Name, call
		}
	}
	return "", nil
}

// isEnvAssignment reports whether assignment sets name.Env alone.
func isEnvAssignment(assignment *ast.AssignStmt, name string) bool {
	if len(assignment.Lhs) != 1 || len(assignment.Rhs) != 1 {
		return false
	}
	selector, ok := assignment.Lhs[0].(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != "Env" {
		return false
	}
	target, ok := selector.X.(*ast.Ident)
	return ok && target.Name == name
}

// assignsName reports whether assignment writes the variable name itself.
func assignsName(assignment *ast.AssignStmt, name string) bool {
	for _, left := range assignment.Lhs {
		if target, ok := left.(*ast.Ident); ok && target.Name == name {
			return true
		}
	}
	return false
}

// isLiteralGitCommand reports whether call is exec.Command("git", ...) or
// exec.CommandContext(ctx, "git", ...).
func isLiteralGitCommand(call *ast.CallExpr, execName string) bool {
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	if !ok || pkg.Name != execName {
		return false
	}
	nameIndex := -1
	switch selector.Sel.Name {
	case "Command":
		nameIndex = 0
	case "CommandContext":
		nameIndex = 1
	default:
		return false
	}
	if len(call.Args) <= nameIndex {
		return false
	}
	literal, ok := call.Args[nameIndex].(*ast.BasicLit)
	if !ok || literal.Kind != token.STRING {
		return false
	}
	value, err := strconv.Unquote(literal.Value)
	return err == nil && value == "git"
}
