package fsl

import (
	"os"
	"os/exec"
	"reflect"
	"testing"
)

func TestIsScopedFSLPath(t *testing.T) {
	cases := map[string]bool{
		"specs/branch-flow.fsl":          true,
		"specs/nested/deep.fsl":          true,
		"skills/a-skill/specs/flow.fsl":  true,
		"skills/a-skill/specs/x/y.fsl":   true,
		"specs/README.md":                false,
		"skills/a-skill/SKILL.md":        false,
		"skills/a-skill/not-specs/x.fsl": false,
		"cmd/mutate-fsl/main.go":         false,
		"docs/validation-tiers.md":       false,
		"specs":                          false,
		"skills/a-skill/specs":           false,
	}
	for path, want := range cases {
		if got := isScopedFSLPath(path); got != want {
			t.Errorf("isScopedFSLPath(%q) = %v, want %v", path, got, want)
		}
	}
}

// gitTest runs git in a throwaway repository with hermetic identity so the
// user's global git configuration never affects the test.
func gitTest(t *testing.T, root string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = root
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test"+"@"+"example.invalid",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test"+"@"+"example.invalid",
		"GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// changeGitfiles commits the current worktree as a new commit and returns its
// short SHA, leaving the following edits uncommitted so ChangedSpecs sees them
// against the recorded base.
func baseCommit(t *testing.T, root string) string {
	t.Helper()
	gitTest(t, root, "add", "-A")
	gitTest(t, root, "commit", "-m", "base")
	out, err := exec.Command("git", "-C", root, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatalf("rev-parse HEAD: %v", err)
	}
	return string(out)[:len(out)-1]
}

func TestChangedSpecsScopesToFSLPaths(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init")
	write(t, root, "specs/branch-flow.fsl", "x")
	write(t, root, "skills/example/specs/flow.fsl", "x")
	write(t, root, "README.md", "readme")
	base := baseCommit(t, root)

	write(t, root, "specs/branch-flow.fsl", "changed")
	write(t, root, "skills/example/specs/flow.fsl", "changed")
	write(t, root, "README.md", "readme changed")
	write(t, root, "docs/validation-tiers.md", "new doc")

	specs, err := ChangedSpecs(root, base)
	if err != nil {
		t.Fatal(err)
	}
	expected := []string{"skills/example/specs/flow.fsl", "specs/branch-flow.fsl"}
	if !reflect.DeepEqual(specs, expected) {
		t.Fatalf("unexpected specs %v", specs)
	}
}

func TestChangedSpecsEmptyWhenNoScopedChanges(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init")
	write(t, root, "README.md", "readme")
	base := baseCommit(t, root)

	write(t, root, "README.md", "readme changed")
	write(t, root, "cmd/mutate-fsl/main.go", "package main")

	specs, err := ChangedSpecs(root, base)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 0 {
		t.Fatalf("expected no specs, got %v", specs)
	}
}

func TestChangedSpecsDedupesSymlinkedExposure(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init")
	write(t, root, "skills/example/specs/flow.fsl", "x")
	writeLink(t, root, "specs/example/flow.fsl", "../../skills/example/specs/flow.fsl")
	base := baseCommit(t, root)

	// The physical source changed; the symlink exposure is unchanged.
	write(t, root, "skills/example/specs/flow.fsl", "changed")

	specs, err := ChangedSpecs(root, base)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(specs, []string{"skills/example/specs/flow.fsl"}) {
		t.Fatalf("expected one deduplicated spec, got %v", specs)
	}
}

func TestChangedSpecsRejectsBrokenSymlink(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init")
	write(t, root, "specs/ok.fsl", "x")
	base := baseCommit(t, root)

	writeLink(t, root, "specs/broken.fsl", "missing-target.fsl")

	if _, err := ChangedSpecs(root, base); err == nil {
		t.Fatal("expected an error for a changed broken symlink")
	}
}

func TestChangedSpecsSortsDeterministically(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init")
	write(t, root, "specs/z.fsl", "x")
	write(t, root, "specs/a.fsl", "x")
	base := baseCommit(t, root)

	write(t, root, "specs/m.fsl", "new middle spec")
	write(t, root, "specs/z.fsl", "changed")
	write(t, root, "specs/a.fsl", "changed")

	want, err := ChangedSpecs(root, base)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		got, err := ChangedSpecs(root, base)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("unstable ordering: got %v, want %v", got, want)
		}
	}
	if !reflect.DeepEqual(want, []string{"specs/a.fsl", "specs/m.fsl", "specs/z.fsl"}) {
		t.Fatalf("unexpected order %v", want)
	}
}

func TestGitDiffNamesErrorsOnUnknownRevision(t *testing.T) {
	root := t.TempDir()
	gitTest(t, root, "init")
	if _, err := gitDiffNames(root, "does-not-exist"); err == nil {
		t.Fatal("expected an error for an unknown revision")
	}
}
