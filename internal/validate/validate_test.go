package validate

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func runValidateCheck(t *testing.T, fn func(root string, out, errOut io.Writer) int, root string) int {
	t.Helper()
	var out, errOut bytes.Buffer
	return fn(root, &out, &errOut)
}

func writeVFile(t *testing.T, root, name, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestOpenCodeConfigParses(t *testing.T) {
	root := t.TempDir()
	writeVFile(t, root, "opencode.json", `{"agent":{}}`)
	content, err := os.ReadFile(filepath.Join(root, "opencode.json"))
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(content, &config); err != nil {
		t.Fatalf("opencode.json must parse: %v", err)
	}
}

func TestScriptTestsRequiresEveryScript(t *testing.T) {
	root := t.TempDir()
	writeVFile(t, root, "scripts/new.py", "print('ok')\n")
	writeVFile(t, root, "SCRIPT_TESTS.toml", "[cmds]\n[scripts]\n")
	if code := runValidateCheck(t, CheckScriptTests, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func TestScriptTestsRequiresNestedScriptCoverage(t *testing.T) {
	root := t.TempDir()
	writeVFile(t, root, "scripts/validate/new.py", "print('ok')\n")
	writeVFile(t, root, "SCRIPT_TESTS.toml", "[cmds]\n[scripts]\n")
	if code := runValidateCheck(t, CheckScriptTests, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func TestScriptTestsRequiresEveryCmdEntrypoint(t *testing.T) {
	root := t.TempDir()
	writeVFile(t, root, "cmd/sample/main.go", "package main\n")
	writeVFile(t, root, "SCRIPT_TESTS.toml", "[cmds]\n[scripts]\n")
	if code := runValidateCheck(t, CheckScriptTests, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	}
}

func TestScriptTestsRejectsOrphanCmdMapping(t *testing.T) {
	root := t.TempDir()
	writeVFile(t, root, "cmd/sample/main.go", "package main\n")
	writeVFile(t, root, "internal/sample/sample_test.go", "package sample\n")
	// scripts/ directory is empty so no script mappings are required.
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeVFile(t, root, "SCRIPT_TESTS.toml", `[cmds]
"cmd/sample" = "./internal/sample"
"cmd/ghost" = "./internal/sample"
[scripts]
`)
	if code := runValidateCheck(t, CheckScriptTests, root); code != 1 {
		t.Fatalf("expected failure for an orphan cmd mapping, got exit %d", code)
	}
}

func TestScriptTestsPassesWithCompleteCoverage(t *testing.T) {
	root := t.TempDir()
	writeVFile(t, root, "cmd/sample/main.go", "package main\n")
	writeVFile(t, root, "internal/sample/sample_test.go", "package sample\n")
	if err := os.MkdirAll(filepath.Join(root, "scripts"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeVFile(t, root, "SCRIPT_TESTS.toml", `[cmds]
"cmd/sample" = "./internal/sample"
[scripts]
`)
	if code := runValidateCheck(t, CheckScriptTests, root); code != 0 {
		t.Fatalf("expected pass, got exit %d", code)
	}
}
