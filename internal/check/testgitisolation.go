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

// CheckTestGitIsolation rejects a Go test that starts git without setting the
// command's Env. A test run from inside a Git hook inherits GIT_DIR and
// similar variables, so an unisolated git init or git commit writes to the
// calling repository instead of the test's temporary one (Issue #372). The
// check walks every _test.go file below root, skipping hidden directories and
// testdata. It reports each exec.Command or exec.CommandContext call whose
// command argument is the literal "git" unless the call result is assigned to
// a variable whose Env field the same top-level function assigns. It returns 0
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
			findings = append(findings, fmt.Sprintf("%s:%d: git command started without Env; set it from support.GitEnv() (internal/support/support.go)", relPath(root, path), line))
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
	fmt.Fprintln(out, "Test Git-isolation check passed: every literal git command in a Go test sets Env.")
	return 0
}

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
	lines := []int{}
	for _, decl := range file.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok || function.Body == nil {
			continue
		}
		for _, call := range unisolatedCallsIn(function.Body, execName) {
			lines = append(lines, fileSet.Position(call.Pos()).Line)
		}
	}
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

// unisolatedCallsIn returns the literal git calls in body that no Env
// assignment in body isolates.
func unisolatedCallsIn(body *ast.BlockStmt, execName string) []*ast.CallExpr {
	assignedTo := map[*ast.CallExpr]string{}
	withEnv := map[string]bool{}
	ast.Inspect(body, func(node ast.Node) bool {
		switch statement := node.(type) {
		case *ast.AssignStmt:
			for index, left := range statement.Lhs {
				if selector, ok := left.(*ast.SelectorExpr); ok && selector.Sel.Name == "Env" {
					if target, ok := selector.X.(*ast.Ident); ok {
						withEnv[target.Name] = true
					}
				}
				if len(statement.Lhs) != len(statement.Rhs) {
					continue
				}
				if call, ok := statement.Rhs[index].(*ast.CallExpr); ok {
					if target, ok := left.(*ast.Ident); ok {
						assignedTo[call] = target.Name
					}
				}
			}
		case *ast.ValueSpec:
			for index, name := range statement.Names {
				if index < len(statement.Values) {
					if call, ok := statement.Values[index].(*ast.CallExpr); ok {
						assignedTo[call] = name.Name
					}
				}
			}
		}
		return true
	})
	calls := []*ast.CallExpr{}
	ast.Inspect(body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok || !isLiteralGitCommand(call, execName) {
			return true
		}
		if name, ok := assignedTo[call]; ok && withEnv[name] {
			return true
		}
		calls = append(calls, call)
		return true
	})
	return calls
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
