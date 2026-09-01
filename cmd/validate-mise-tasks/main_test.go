package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, mise string, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "mise.toml"), []byte(mise), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestValidateAcceptsCanonicalTasks(t *testing.T) {
	root := writeFixture(t, "[tasks.\"check:repository\"]\nrun = \"true\"\n", map[string]string{"README.md": "mise run check:repository\n"})
	if errs := validate(root, filepath.Join(root, "mise.toml")); len(errs) != 0 {
		t.Fatalf("expected canonical fixture to pass, got %v", errs)
	}
}

func TestValidateRejectsUnnamespacedAndMultiWordCategories(t *testing.T) {
	root := writeFixture(t, "[tasks.verify-fsl]\nrun = \"true\"\n[tasks.\"worktree:diagnose\"]\nrun = \"true\"\n", nil)
	errs := validate(root, filepath.Join(root, "mise.toml"))
	joined := strings.Join(errorStrings(errs), "\n")
	if !strings.Contains(joined, "verify-fsl") || !strings.Contains(joined, "worktree:diagnose") {
		t.Fatalf("expected both invalid task names, got %v", errs)
	}
}

func TestValidateRejectsRetiredReference(t *testing.T) {
	root := writeFixture(t, "[tasks.\"verify:fsl\"]\nrun = \"true\"\n", map[string]string{"README.md": "mise run verify-fsl\n"})
	errs := validate(root, filepath.Join(root, "mise.toml"))
	joined := strings.Join(errorStrings(errs), "\n")
	if !strings.Contains(joined, "retired task \"verify-fsl\"") {
		t.Fatalf("expected retired reference failure, got %v", errs)
	}
}

func TestValidateRejectsRetiredDependency(t *testing.T) {
	root := writeFixture(t, "[tasks.\"verify:fsl\"]\nrun = \"true\"\n[tasks.\"validate:all\"]\ndepends = [\"verify-fsl\"]\n", nil)
	errs := validate(root, filepath.Join(root, "mise.toml"))
	joined := strings.Join(errorStrings(errs), "\n")
	if !strings.Contains(joined, "retired task \"verify-fsl\"") {
		t.Fatalf("expected retired dependency failure, got %v", errs)
	}
}

func TestValidateRejectsRetiredDiagnoseWorktree(t *testing.T) {
	root := writeFixture(t, "[tasks.\"check:local\"]\nrun = \"true\"\n", map[string]string{
		"README.md": "mise run diagnose:worktree -- --branch issue/123\n",
	})
	errs := validate(root, filepath.Join(root, "mise.toml"))
	joined := strings.Join(errorStrings(errs), "\n")
	if !strings.Contains(joined, "retired task \"diagnose:worktree\"") {
		t.Fatalf("expected retired diagnose:worktree reference failure, got %v", errs)
	}
}

func TestValidateRejectsRedeclaredDiagnoseWorktree(t *testing.T) {
	root := writeFixture(t, "[tasks.\"diagnose:worktree\"]\nrun = \"true\"\n", nil)
	errs := validate(root, filepath.Join(root, "mise.toml"))
	joined := strings.Join(errorStrings(errs), "\n")
	if !strings.Contains(joined, "retired task \"diagnose:worktree\"") {
		t.Fatalf("expected retired declaration failure, got %v", errs)
	}
	if !strings.Contains(joined, "one-word verb category") {
		t.Fatalf("expected the retired diagnose verb to leave the approved vocabulary, got %v", errs)
	}
}

func TestValidateRejectsRetiredDependencyInAnyPosition(t *testing.T) {
	mise := "[tasks.\"check:all\"]\nrun = \"true\"\n[tasks.\"validate:all\"]\ndepends = [\"check:all\", \"diagnose:worktree\"]\n"
	root := writeFixture(t, mise, nil)
	errs := validate(root, filepath.Join(root, "mise.toml"))
	joined := strings.Join(errorStrings(errs), "\n")
	if !strings.Contains(joined, "retired task \"diagnose:worktree\"") {
		t.Fatalf("expected a retired dependency past first position to fail, got %v", errs)
	}
}

func TestValidateRejectsWrappedRetiredInvocation(t *testing.T) {
	root := writeFixture(t, "[tasks.\"check:local\"]\nrun = \"true\"\n", map[string]string{
		"README.md": "Inspect the worktree with `mise run\ndiagnose:worktree -- --branch issue/1` first.\n",
	})
	errs := validate(root, filepath.Join(root, "mise.toml"))
	joined := strings.Join(errorStrings(errs), "\n")
	if !strings.Contains(joined, "retired task \"diagnose:worktree\"") {
		t.Fatalf("expected a line-wrapped invocation to fail, got %v", errs)
	}
}

// A wrapped reference must not swallow the namespaced successor of a retired
// bare name: `mise run` followed by `setup:all` is current guidance.
func TestValidateAllowsWrappedNamespacedSuccessor(t *testing.T) {
	root := writeFixture(t, "[tasks.\"setup:all\"]\nrun = \"true\"\n", map[string]string{
		"README.md": "The hook reruns `mise run\nsetup:all` on checkout.\n",
	})
	if errs := validate(root, filepath.Join(root, "mise.toml")); len(errs) != 0 {
		t.Fatalf("expected the wrapped namespaced task to pass, got %v", errs)
	}
}

func TestValidateAllowsRetiredTaskInProse(t *testing.T) {
	root := writeFixture(t, "[tasks.\"check:local\"]\nrun = \"true\"\n", map[string]string{
		"docs/worktrees.md": "The `diagnose:worktree` task was removed in favor of `wt list`.\n",
	})
	if errs := validate(root, filepath.Join(root, "mise.toml")); len(errs) != 0 {
		t.Fatalf("expected a historical prose mention to pass, got %v", errs)
	}
}

func errorStrings(errs []error) []string {
	result := make([]string, len(errs))
	for i, err := range errs {
		result[i] = err.Error()
	}
	return result
}
