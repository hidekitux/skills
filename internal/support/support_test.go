package support

import (
	"os"
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
