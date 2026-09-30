package check

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const pinSHA = "c2a87611a18de5b3828c5652fe268e992400cb5c"

func runActionPinCheck(t *testing.T, files map[string]string) (int, string) {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var out, errOut bytes.Buffer
	code := CheckActionPins(root, &out, &errOut)
	return code, out.String() + errOut.String()
}

func TestActionPinsPassesWithTrailingVersionComment(t *testing.T) {
	code, output := runActionPinCheck(t, map[string]string{
		".github/workflows/ci.yml":            "steps:\n      - name: Set up mise\n        uses: jdx/mise-action@" + pinSHA + " # v4.3.0\n",
		".github/actions/setup-go/action.yml": "runs:\n  steps:\n    - uses: actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff # v5.6.0\n",
	})
	if code != 0 {
		t.Fatalf("expected 0, got %d:\n%s", code, output)
	}
	if !strings.Contains(output, "2 pinned action reference(s)") {
		t.Fatalf("output missing pin count:\n%s", output)
	}
}

func TestActionPinsFailsWithoutVersionComment(t *testing.T) {
	code, output := runActionPinCheck(t, map[string]string{
		".github/workflows/ci.yml": "steps:\n      - uses: jdx/mise-action@" + pinSHA + "\n",
	})
	if code != 1 {
		t.Fatalf("expected 1, got %d:\n%s", code, output)
	}
	if !strings.Contains(output, ".github/workflows/ci.yml:2: pinned uses: needs a trailing") {
		t.Fatalf("output missing file and line:\n%s", output)
	}
}

func TestActionPinsFailsOnSeparateLineSHAComment(t *testing.T) {
	code, output := runActionPinCheck(t, map[string]string{
		".github/workflows/ci.yml": "steps:\n      # v3 -> 3c2e0cf82a5b2e5249f0d3635a4d83d0ae861518\n      - name: Set up mise\n        uses: jdx/mise-action@" + pinSHA + " # v4.3.0\n",
	})
	if code != 1 {
		t.Fatalf("expected 1, got %d:\n%s", code, output)
	}
	if !strings.Contains(output, ".github/workflows/ci.yml:2: comment repeats a commit SHA") {
		t.Fatalf("output missing file and line:\n%s", output)
	}
}

func TestActionPinsFailsOnSameLineSHAComment(t *testing.T) {
	code, output := runActionPinCheck(t, map[string]string{
		".github/workflows/ci.yml": "steps:\n      - uses: jdx/mise-action@" + pinSHA + " # v3 -> 3c2e0cf82a5b2e5249f0d3635a4d83d0ae861518\n",
	})
	if code != 1 {
		t.Fatalf("expected 1, got %d:\n%s", code, output)
	}
	if !strings.Contains(output, ".github/workflows/ci.yml:2: pinned uses: needs a trailing") {
		t.Fatalf("output missing file and line:\n%s", output)
	}
}

func TestActionPinsReadsYAMLExtension(t *testing.T) {
	stale := "steps:\n  # v3 -> 3c2e0cf82a5b2e5249f0d3635a4d83d0ae861518\n  - uses: jdx/mise-action@" + pinSHA + "\n"
	code, output := runActionPinCheck(t, map[string]string{
		".github/workflows/ci.yaml":     stale,
		".github/actions/x/action.yaml": stale,
	})
	if code != 1 {
		t.Fatalf("expected 1, got %d:\n%s", code, output)
	}
	for _, want := range []string{
		".github/workflows/ci.yaml:2: comment repeats a commit SHA",
		".github/workflows/ci.yaml:3: pinned uses: needs a trailing",
		".github/actions/x/action.yaml:2: comment repeats a commit SHA",
		".github/actions/x/action.yaml:3: pinned uses: needs a trailing",
	} {
		if !strings.Contains(output, want) {
			t.Fatalf("output missing %q:\n%s", want, output)
		}
	}
}

func TestActionPinsSkipsLocalAndUnpinnedReferences(t *testing.T) {
	code, output := runActionPinCheck(t, map[string]string{
		".github/workflows/ci.yml": "steps:\n      - uses: $/.github/actions/setup-go\n      - uses: ./.github/actions/setup-go\n      - uses: actions/checkout@v4\n",
	})
	if code != 0 {
		t.Fatalf("expected 0, got %d:\n%s", code, output)
	}
	if !strings.Contains(output, "0 pinned action reference(s)") {
		t.Fatalf("output missing pin count:\n%s", output)
	}
}
