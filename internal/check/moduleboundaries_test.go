package check

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureOwnership is the ownership file used by the temporary-tree tests. It
// keeps two modules so one allowed edge and one forbidden edge both exist.
const fixtureOwnership = `schema_version: 1
modules:
  - id: foundation
    owns: Shared primitives.
    packages:
      - support
    may_import: []
  - id: policy
    owns: Policy decisions.
    packages:
      - strategy
    may_import:
      - foundation
`

// writeModuleTree builds a repository root whose internal packages contain
// only the given import lines, so a test states exactly the edges it exercises.
func writeModuleTree(t *testing.T, ownership string, imports map[string][]string) string {
	t.Helper()
	root := t.TempDir()
	workflowDir := filepath.Join(root, "workflow")
	if err := os.MkdirAll(workflowDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(workflowDir, "module-ownership.yml"), []byte(ownership), 0o644); err != nil {
		t.Fatal(err)
	}
	for pkg, paths := range imports {
		dir := filepath.Join(root, "internal", filepath.FromSlash(pkg))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		name := pkg
		if index := strings.LastIndex(pkg, "/"); index >= 0 {
			name = pkg[index+1:]
		}
		source := "package " + name + "\n"
		if len(paths) > 0 {
			source += "\nimport (\n"
			for _, path := range paths {
				prefix := internalImportPrefix
				if strings.HasPrefix(path, "cmd/") {
					prefix, path = commandImportPrefix, strings.TrimPrefix(path, "cmd/")
				}
				source += "\t_ \"" + prefix + path + "\"\n"
			}
			source += ")\n"
		}
		if err := os.WriteFile(filepath.Join(dir, name+".go"), []byte(source), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func runModuleBoundaries(t *testing.T, root string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := CheckModuleBoundaries(root, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestCheckModuleBoundaries(t *testing.T) {
	cases := []struct {
		name      string
		ownership string
		imports   map[string][]string
		wantCode  int
		wantText  string
	}{
		{
			name:      "allowed edge passes",
			ownership: fixtureOwnership,
			imports:   map[string][]string{"support": nil, "strategy": {"support"}},
			wantCode:  0,
			wantText:  "module boundaries valid: 2 packages in 2 modules, 1 allowed module edges.",
		},
		{
			name:      "forbidden reverse dependency fails",
			ownership: fixtureOwnership,
			imports:   map[string][]string{"support": {"strategy"}, "strategy": nil},
			wantCode:  1,
			wantText:  "forbidden reverse dependency: internal/support (foundation) imports internal/strategy (policy)",
		},
		{
			name:      "forbidden reverse dependency from a nested package fails",
			ownership: fixtureOwnership,
			imports:   map[string][]string{"support": nil, "support/util": {"strategy"}, "strategy": nil},
			wantCode:  1,
			wantText:  "forbidden reverse dependency: internal/support/util (foundation) imports internal/strategy (policy)",
		},
		{
			name:      "nested package on an allowed edge passes",
			ownership: fixtureOwnership,
			imports:   map[string][]string{"support": nil, "strategy/rule": {"support"}, "strategy": nil},
			wantCode:  0,
			wantText:  "module boundaries valid: 3 packages in 2 modules, 1 allowed module edges.",
		},
		{
			name:      "import of the composition root fails",
			ownership: fixtureOwnership,
			imports:   map[string][]string{"support": nil, "strategy": {"cmd/check-repository"}},
			wantCode:  1,
			wantText:  "forbidden reverse dependency: internal/strategy (policy) imports cmd/check-repository (composition); no module may import composition",
		},
		{
			name:      "unowned package fails",
			ownership: fixtureOwnership,
			imports:   map[string][]string{"support": nil, "strategy": nil, "trace": nil},
			wantCode:  1,
			wantText:  "package internal/trace belongs to no module",
		},
		{
			name:      "missing package fails",
			ownership: fixtureOwnership,
			imports:   map[string][]string{"support": nil},
			wantCode:  1,
			wantText:  "workflow/module-ownership.yml lists internal/strategy, which does not exist",
		},
		{
			name: "may_import naming an unknown module fails",
			ownership: `schema_version: 1
modules:
  - id: policy
    owns: Policy decisions.
    packages:
      - strategy
    may_import:
      - foundation
`,
			imports:  map[string][]string{"strategy": nil},
			wantCode: 1,
			wantText: `may_import names unknown module "foundation"`,
		},
		{
			name: "duplicate package ownership fails",
			ownership: `schema_version: 1
modules:
  - id: foundation
    owns: Shared primitives.
    packages:
      - support
    may_import: []
  - id: policy
    owns: Policy decisions.
    packages:
      - support
    may_import: []
`,
			imports:  map[string][]string{"support": nil},
			wantCode: 1,
			wantText: `package "support" is owned by both "foundation" and "policy"`,
		},
		{
			name: "unsupported schema version fails",
			ownership: `schema_version: 2
modules:
  - id: foundation
    owns: Shared primitives.
    packages:
      - support
    may_import: []
`,
			imports:  map[string][]string{"support": nil},
			wantCode: 1,
			wantText: "unsupported schema_version 2, want 1",
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			root := writeModuleTree(t, testCase.ownership, testCase.imports)
			code, out, errOut := runModuleBoundaries(t, root)
			if code != testCase.wantCode {
				t.Fatalf("exit code = %d, want %d (stdout %q, stderr %q)", code, testCase.wantCode, out, errOut)
			}
			text := out + errOut
			if !strings.Contains(text, testCase.wantText) {
				t.Fatalf("output %q does not contain %q", text, testCase.wantText)
			}
		})
	}
}

// TestCheckModuleBoundariesRepository runs the check against the repository
// itself, so the recorded ownership stays true for the committed tree.
func TestCheckModuleBoundariesRepository(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	code, out, errOut := runModuleBoundaries(t, root)
	if code != 0 {
		t.Fatalf("repository module boundaries failed: exit %d\nstdout: %s\nstderr: %s", code, out, errOut)
	}
	if !strings.Contains(out, "module boundaries valid:") {
		t.Fatalf("stdout %q does not report a valid result", out)
	}
}

// TestCheckModuleBoundariesMissingFile records the failure when the ownership
// file is absent, so the check cannot pass by skipping its input.
func TestCheckModuleBoundariesMissingFile(t *testing.T) {
	code, _, errOut := runModuleBoundaries(t, t.TempDir())
	if code != 1 {
		t.Fatalf("exit code = %d, want 1", code)
	}
	if !strings.Contains(errOut, "module-boundaries check failed") {
		t.Fatalf("stderr %q does not report the failure", errOut)
	}
}
