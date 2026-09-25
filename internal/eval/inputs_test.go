package eval

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/hidekitux/skills/internal/support"
)

// writeInputFile writes content to a slash-separated path below root.
func writeInputFile(t *testing.T, root, path, content string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// commitAll commits every file in root without signing or user config.
func commitAll(t *testing.T, root, message string) {
	t.Helper()
	for _, args := range [][]string{
		{"add", "--all"},
		{"-c", "user.name=Test", "-c", "user.email=test" + "@" + "example.invalid", "-c", "commit.gpgsign=false", "commit", "--quiet", "--message", message},
	} {
		command := exec.Command("git", args...)
		command.Dir = root
		command.Env = support.GitEnv()
		if out, err := command.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
}

// newInputRepository creates a Git repository holding one skill "demo", its
// scenario with a fixture, a scenario for another skill, the shared harness
// inputs, and a document outside the input set.
func newInputRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	command := exec.Command("git", "init", "--quiet")
	command.Dir = root
	command.Env = support.GitEnv()
	if out, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	for path, content := range map[string]string{
		"skills/process/demo/SKILL.md":                 "---\nname: demo\n---\n",
		"skills/process/other/SKILL.md":                "---\nname: other\n---\n",
		"evaluations/scenarios/demo/demo-success.yaml": "id: demo-success\nskill: demo\nkind: positive\nfixture: demo-fixture\n",
		"evaluations/scenarios/other/other.yaml":       "id: other-success\nskill: other\nkind: positive\nfixture: other-fixture\n",
		"evaluations/fixtures/demo-fixture/input.md":   "demo input\n",
		"evaluations/fixtures/other-fixture/input.md":  "other input\n",
		"evaluations/rubric.md":                        "rubric\n",
		"internal/eval/harness.go":                     "package eval\n",
		"go.mod":                                       "module demo\n",
		"mise.toml":                                    "[tools]\n",
		"docs/evaluation.md":                           "docs\n",
		".github/workflows/ci.yml":                     "on: push\n",
	} {
		writeInputFile(t, root, path, content)
	}
	commitAll(t, root, "initial")
	return root
}

func TestInputDigestIgnoresUnrelatedChanges(t *testing.T) {
	root := newInputRepository(t)
	before, err := InputDigest(root, root, "demo")
	if err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		"docs/evaluation.md":                          "changed docs\n",
		".github/workflows/ci.yml":                    "on: pull_request\n",
		"evaluations/scenarios/other/other.yaml":      "id: other-success\nskill: other\nkind: negative\nfixture: other-fixture\n",
		"evaluations/fixtures/other-fixture/input.md": "changed other input\n",
	} {
		writeInputFile(t, root, path, content)
	}
	commitAll(t, root, "unrelated")
	after, err := InputDigest(root, root, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("digest changed after an unrelated commit: %s -> %s", before, after)
	}
}

func TestInputDigestChangesWithRelevantInputs(t *testing.T) {
	for _, path := range []string{
		"skills/process/demo/SKILL.md",
		"skills/process/other/SKILL.md",
		"evaluations/scenarios/demo/demo-success.yaml",
		"evaluations/fixtures/demo-fixture/input.md",
		"evaluations/rubric.md",
		"internal/eval/harness.go",
		"go.mod",
		"mise.toml",
	} {
		t.Run(path, func(t *testing.T) {
			root := newInputRepository(t)
			before, err := InputDigest(root, root, "demo")
			if err != nil {
				t.Fatal(err)
			}
			content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
			if err != nil {
				t.Fatal(err)
			}
			writeInputFile(t, root, path, string(content)+"# changed\n")
			commitAll(t, root, "relevant")
			after, err := InputDigest(root, root, "demo")
			if err != nil {
				t.Fatal(err)
			}
			if before == after {
				t.Fatalf("digest did not change after a commit to %s", path)
			}
		})
	}
}

func TestInputDigestIgnoresIgnoredFiles(t *testing.T) {
	root := newInputRepository(t)
	before, err := InputDigest(root, root, "demo")
	if err != nil {
		t.Fatal(err)
	}
	writeInputFile(t, root, ".gitignore", "*.out\n")
	writeInputFile(t, root, "internal/eval/build.out", "local build output\n")
	after, err := InputDigest(root, root, "demo")
	if err != nil {
		t.Fatal(err)
	}
	if before != after {
		t.Fatalf("digest changed after adding an ignored file: %s -> %s", before, after)
	}
}
