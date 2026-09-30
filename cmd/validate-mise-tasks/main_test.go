package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFixture writes mise.toml and files below a temporary root. Unless files
// supplies docs/mise-tasks.md, it also writes a canonical task inventory that
// lists every declared task under its verb, so a test sees only the errors it
// provokes.
func writeFixture(t *testing.T, mise string, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "mise.toml"), []byte(mise), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, ok := files[inventoryPath]; !ok {
		tasks, err := declaredTasks(filepath.Join(root, "mise.toml"))
		if err != nil {
			t.Fatal(err)
		}
		files = withFile(files, inventoryPath, inventoryFor(tasks))
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

// withFile returns a copy of files with name set to content.
func withFile(files map[string]string, name, content string) map[string]string {
	result := map[string]string{name: content}
	for key, value := range files {
		if key != name {
			result[key] = value
		}
	}
	return result
}

// inventoryFor renders a canonical task inventory that lists tasks under their
// verbs.
func inventoryFor(tasks []string) string {
	rows := map[string][]string{}
	var verbs []string
	for _, task := range tasks {
		verb, _, _ := strings.Cut(task, ":")
		if _, ok := rows[verb]; !ok {
			verbs = append(verbs, verb)
		}
		rows[verb] = append(rows[verb], "`"+task+"`")
	}
	var b strings.Builder
	b.WriteString("# Mise task naming\n\n" + inventoryHeading + "\n\n| Category | Task names |\n| --- | --- |\n")
	for _, verb := range verbs {
		b.WriteString("| " + verb + " | " + strings.Join(rows[verb], ", ") + " |\n")
	}
	return b.String()
}

const inventoryMise = "[tasks.\"check:all\"]\nrun = \"true\"\n[tasks.\"lint:go\"]\nrun = \"true\"\n"

func TestValidateAcceptsMatchingTaskInventory(t *testing.T) {
	root := writeFixture(t, inventoryMise, map[string]string{
		inventoryPath: "# Mise task naming\n\n" + inventoryHeading + "\n\n| Category | Task names |\n| --- | --- |\n| check | `check:all` |\n| lint | `lint:go` |\n\n## Other section\n\n| Category | Task names |\n| --- | --- |\n| test | `test:ignored` |\n",
	})
	if errs := validate(root, filepath.Join(root, "mise.toml")); len(errs) != 0 {
		t.Fatalf("expected a matching inventory to pass, got %v", errs)
	}
}

func TestValidateRejectsTaskMissingFromInventory(t *testing.T) {
	root := writeFixture(t, inventoryMise, map[string]string{inventoryPath: inventoryFor([]string{"check:all"})})
	joined := strings.Join(errorStrings(validate(root, filepath.Join(root, "mise.toml"))), "\n")
	if !strings.Contains(joined, `task "lint:go" is declared in mise.toml but missing from the canonical task inventory in docs/mise-tasks.md`) {
		t.Fatalf("expected the missing task to be named, got %s", joined)
	}
}

func TestValidateRejectsInventoryTaskThatMiseDoesNotDeclare(t *testing.T) {
	root := writeFixture(t, inventoryMise, map[string]string{inventoryPath: inventoryFor([]string{"check:all", "lint:go", "check:gone"})})
	joined := strings.Join(errorStrings(validate(root, filepath.Join(root, "mise.toml"))), "\n")
	if !strings.Contains(joined, `task "check:gone" is listed in the canonical task inventory in docs/mise-tasks.md but not declared in mise.toml`) {
		t.Fatalf("expected the extra task to be named, got %s", joined)
	}
}

func TestValidateRejectsInventoryTaskUnderAnotherCategory(t *testing.T) {
	root := writeFixture(t, inventoryMise, map[string]string{
		inventoryPath: inventoryHeading + "\n\n| Category | Task names |\n| --- | --- |\n| check | `check:all`, `lint:go` |\n",
	})
	joined := strings.Join(errorStrings(validate(root, filepath.Join(root, "mise.toml"))), "\n")
	if !strings.Contains(joined, `task "lint:go" is listed under category "check" in docs/mise-tasks.md instead of "lint"`) {
		t.Fatalf("expected the misplaced task to be named, got %s", joined)
	}
}

func TestValidateRejectsMissingTaskInventory(t *testing.T) {
	for name, files := range map[string]map[string]string{
		"missing file":    {"README.md": "mise run check:all\n"},
		"missing heading": {inventoryPath: "# Mise task naming\n\n| Category | Task names |\n| --- | --- |\n| check | `check:all` |\n"},
	} {
		root := writeFixture(t, inventoryMise, files)
		if name == "missing file" {
			if err := os.Remove(filepath.Join(root, inventoryPath)); err != nil {
				t.Fatal(err)
			}
		}
		joined := strings.Join(errorStrings(validate(root, filepath.Join(root, "mise.toml"))), "\n")
		if !strings.Contains(joined, inventoryPath) {
			t.Errorf("%s: expected an error that names %s, got %s", name, inventoryPath, joined)
		}
	}
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

// gitFixture turns root into a Git work tree, writes .gitignore, and stages
// the tracked paths. It stages without committing, so no commit-signing
// configuration is involved.
func gitFixture(t *testing.T, root, gitignore string, tracked ...string) {
	t.Helper()
	ctx := context.Background()
	if _, err := gitPort.Output(ctx, root, "init", "-q"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := gitPort.Output(ctx, root, append([]string{"add", "--"}, tracked...)...); err != nil {
		t.Fatal(err)
	}
}

const canonicalMise = "[tasks.\"verify:fsl\"]\nrun = \"true\"\n"

func TestValidateIgnoresRetiredReferenceInGitIgnoredFile(t *testing.T) {
	root := writeFixture(t, canonicalMise, map[string]string{".claude/worktrees/copy/README.md": "mise run verify-fsl\n"})
	gitFixture(t, root, ".claude/\n", "mise.toml")
	if errs := validate(root, filepath.Join(root, "mise.toml")); len(errs) != 0 {
		t.Fatalf("expected a Git-ignored reference to be skipped, got %v", errs)
	}
}

func TestValidateRejectsRetiredReferenceInTrackedFile(t *testing.T) {
	root := writeFixture(t, canonicalMise, map[string]string{"docs/guide.md": "mise run verify-fsl\n"})
	gitFixture(t, root, ".claude/\n", "mise.toml", "docs/guide.md")
	joined := strings.Join(errorStrings(validate(root, filepath.Join(root, "mise.toml"))), "\n")
	if !strings.Contains(joined, "retired task \"verify-fsl\" is referenced by "+filepath.Join(root, "docs", "guide.md")) {
		t.Fatalf("expected the tracked reference to fail, got %s", joined)
	}
}

func TestValidateRejectsRetiredReferenceInUntrackedNonIgnoredFile(t *testing.T) {
	root := writeFixture(t, canonicalMise, map[string]string{"notes/draft.md": "mise run verify-fsl\n"})
	gitFixture(t, root, ".claude/\n", "mise.toml")
	joined := strings.Join(errorStrings(validate(root, filepath.Join(root, "mise.toml"))), "\n")
	if !strings.Contains(joined, "retired task \"verify-fsl\" is referenced by "+filepath.Join(root, "notes", "draft.md")) {
		t.Fatalf("expected the untracked, non-ignored reference to fail, got %s", joined)
	}
}

func TestValidateWalksEveryFileOutsideAGitWorkTree(t *testing.T) {
	root := writeFixture(t, canonicalMise, map[string]string{".claude/worktrees/copy/README.md": "mise run verify-fsl\n"})
	joined := strings.Join(errorStrings(validate(root, filepath.Join(root, "mise.toml"))), "\n")
	if !strings.Contains(joined, "retired task \"verify-fsl\"") {
		t.Fatalf("expected the fallback walk to read every file, got %s", joined)
	}
}
