package support

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func initTestRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if _, err := GitOutputIn(root, "init", "--quiet"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	return root
}

func TestResolveRoot(t *testing.T) {
	root := initTestRepository(t)
	nested := filepath.Join(root, "nested", "directory")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatal(err)
	}
	expectedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("evaluate repository root symlinks: %v", err)
	}
	expectedRoot = filepath.Clean(expectedRoot)

	t.Run("repository root", func(t *testing.T) {
		resolved, err := ResolveRoot(root)
		if err != nil {
			t.Fatalf("ResolveRoot(%q) failed: %v", root, err)
		}
		if resolved != expectedRoot {
			t.Fatalf("ResolveRoot(%q) = %q, want %q", root, resolved, expectedRoot)
		}
	})

	t.Run("nested directory", func(t *testing.T) {
		resolved, err := ResolveRoot(nested)
		if err != nil {
			t.Fatalf("ResolveRoot(%q) failed: %v", nested, err)
		}
		if resolved != expectedRoot {
			t.Fatalf("ResolveRoot(%q) = %q, want %q", nested, resolved, expectedRoot)
		}
	})

	t.Run("outside repository", func(t *testing.T) {
		outside := t.TempDir()
		_, err := ResolveRoot(outside)
		if err == nil {
			t.Fatalf("ResolveRoot(%q) succeeded outside a repository", outside)
		}
		if !strings.Contains(err.Error(), outside) {
			t.Fatalf("ResolveRoot(%q) error = %q, want path in error", outside, err)
		}
	})
}

func TestResolveRootPreservesTrailingSpace(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "repo-with-trailing-space ")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := GitOutputIn(root, "init", "--quiet"); err != nil {
		t.Fatalf("git init failed: %v", err)
	}
	expectedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("evaluate repository root symlinks: %v", err)
	}

	resolved, err := ResolveRoot(root)
	if err != nil {
		t.Fatalf("ResolveRoot(%q) failed: %v", root, err)
	}
	if resolved != filepath.Clean(expectedRoot) {
		t.Fatalf("ResolveRoot(%q) = %q, want %q", root, resolved, expectedRoot)
	}
}

func TestLoadTOMLFile(t *testing.T) {
	root := t.TempDir()
	validPath := filepath.Join(root, "valid.toml")
	if err := os.WriteFile(validPath, []byte("name = \"skills\"\ncount = 3\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var got struct {
		Name  string `toml:"name"`
		Count int    `toml:"count"`
	}
	if err := LoadTOMLFile(validPath, &got); err != nil {
		t.Fatalf("LoadTOMLFile(%q) failed: %v", validPath, err)
	}
	if got.Name != "skills" || got.Count != 3 {
		t.Fatalf("LoadTOMLFile(%q) = %#v, want name=skills and count=3", validPath, got)
	}

	t.Run("malformed input names path", func(t *testing.T) {
		path := filepath.Join(root, "malformed.toml")
		if err := os.WriteFile(path, []byte("name = \"unterminated\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := LoadTOMLFile(path, &struct{}{}); err == nil {
			t.Fatalf("LoadTOMLFile(%q) succeeded for malformed input", path)
		} else if !strings.Contains(err.Error(), path) {
			t.Fatalf("LoadTOMLFile(%q) error = %q, want path in error", path, err)
		}
	})

	t.Run("missing file names path", func(t *testing.T) {
		path := filepath.Join(root, "missing.toml")
		if err := LoadTOMLFile(path, &struct{}{}); err == nil {
			t.Fatalf("LoadTOMLFile(%q) succeeded for missing input", path)
		} else if !strings.Contains(err.Error(), path) {
			t.Fatalf("LoadTOMLFile(%q) error = %q, want path in error", path, err)
		}
	})
}

func TestExitErrorHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	os.Exit(7)
}

func TestExitError(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=TestExitErrorHelperProcess")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	exitErr := cmd.Run()
	if exitErr == nil {
		t.Fatal("helper process unexpectedly succeeded")
	}

	tests := map[string]struct {
		err  error
		want int
	}{
		"nil":     {want: 0},
		"plain":   {err: errors.New("plain error"), want: 1},
		"exit":    {err: exitErr, want: 7},
		"wrapped": {err: fmt.Errorf("wrapped: %w", exitErr), want: 7},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := ExitError(test.err); got != test.want {
				t.Fatalf("ExitError(%v) = %d, want %d", test.err, got, test.want)
			}
		})
	}
}

func TestIsZeroSHA(t *testing.T) {
	for name, test := range map[string]struct {
		sha  string
		want bool
	}{
		"empty":               {sha: "", want: true},
		"single zero":         {sha: "0", want: true},
		"git zero SHA":        {sha: strings.Repeat("0", 40), want: true},
		"last digit non-zero": {sha: strings.Repeat("0", 39) + "1", want: false},
		"non-zero text":       {sha: "not-a-sha", want: false},
	} {
		t.Run(name, func(t *testing.T) {
			if got := IsZeroSHA(test.sha); got != test.want {
				t.Fatalf("IsZeroSHA(%q) = %t, want %t", test.sha, got, test.want)
			}
		})
	}
}
