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
// Env from support.GitEnv, support.WithoutGitEnvironment, or a function in the
// same file that returns one of them. It returns 0 on success and 1 when
// findings exist.
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
			findings = append(findings, fmt.Sprintf("%s:%d: git command without an Env set from support.GitEnv, support.WithoutGitEnvironment, or a helper in the same file, in the same block before any reassignment", relPath(root, path), line))
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
	fmt.Fprintln(out, "Test Git-isolation check passed: every literal git command in a Go test sets Env from support.GitEnv().")
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

// isolatingHelpers returns the top-level functions in file that return an
// isolating value, such as fixtureGitEnvironment in
// internal/environment/provision_test.go. It repeats until no helper is added,
// so a helper that returns another helper's result also counts.
func isolatingHelpers(file *ast.File) map[string]bool {
	helpers := map[string]bool{}
	for added := true; added; {
		added = false
		for _, decl := range file.Decls {
			function, ok := decl.(*ast.FuncDecl)
			if !ok || function.Recv != nil || function.Body == nil || helpers[function.Name.Name] {
				continue
			}
			ast.Inspect(function.Body, func(node ast.Node) bool {
				if _, ok := node.(*ast.FuncLit); ok {
					return false
				}
				if statement, ok := node.(*ast.ReturnStmt); ok && !helpers[function.Name.Name] {
					for _, result := range statement.Results {
						if isIsolatingValue(result, helpers) {
							helpers[function.Name.Name] = true
							added = true
						}
					}
				}
				return true
			})
		}
	}
	return helpers
}

// isIsolatingValue reports whether expr calls support.GitEnv,
// support.WithoutGitEnvironment, or an isolating helper. append(os.Environ(),
// ...) and nil do not, because both keep an inherited GIT_DIR.
func isIsolatingValue(expr ast.Expr, helpers map[string]bool) bool {
	found := false
	ast.Inspect(expr, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return !found
		}
		switch function := call.Fun.(type) {
		case *ast.Ident:
			found = found || isolatingFunctions[function.Name] || helpers[function.Name]
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
func markIsolated(statements []ast.Stmt, execName string, helpers map[string]bool, isolated map[*ast.CallExpr]bool) {
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
