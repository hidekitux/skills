package publicstatus

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixtureCatalog is a three-entry-point catalog used by the fixture documents.
const fixtureCatalog = `catalog_version: 1
license: Apache-2.0
skills:
  - name: improve-project
    summary: Improve a project end to end.
    owner: hidekitux
    status: experimental
    agents: [codex, claude-code]
    license: Apache-2.0
    version: 0.1.0
    layer: process
    related: [analyze-project, create-issue]
  - name: deliver-change
    summary: Deliver a governed Change Issue end to end.
    owner: hidekitux
    status: experimental
    agents: [codex, claude-code]
    license: Apache-2.0
    version: 0.1.0
    layer: process
    related: [plan-issue, implement-issue]
  - name: resolve-defect
    summary: Resolve a verified defect end to end.
    owner: hidekitux
    status: experimental
    agents: [codex, claude-code]
    license: Apache-2.0
    version: 0.1.0
    layer: fix
    related: [debug-code, create-issue]
`

// fixtureEvidence is the retained release record for the unreleased fixture
// repository.
const fixtureEvidence = `released: false
tag: ""
release_url: ""
commit: ""
`

const fixtureAuthority = "# Public skill status\n\nFixture authority and evidence contract.\n"

// baseFiles returns the fixture repository files with a README that has no
// generated section yet.
func baseFiles() map[string]string {
	return map[string]string{
		"CATALOG.yml":                 fixtureCatalog,
		"docs/release-evidence.yml":   fixtureEvidence,
		"docs/public-skill-status.md": fixtureAuthority,
		"README.md":                   "# Fixture README\n\n## Skill-set map\n\n| Skill | Status |\n| --- | --- |\n",
	}
}

// readmeWithBlock returns a README whose generated section is the given block.
func readmeWithBlock(block string) string {
	return "# Fixture README\n\n## Skill-set map\n\n| Skill | Status |\n| --- | --- |\n" + block + "\n\n## Development workflow\n\nFixture steps.\n"
}

// writeRepo scaffolds a fixture repository from a relative-path file map.
func writeRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		path := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// runCheck runs Check against a fixture repository and returns the exit code
// and the captured error output.
func runCheck(t *testing.T, root string) (int, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := Check(root, &out, &errOut)
	return code, errOut.String()
}

// renderFixture renders the fixture block once so tests can share it.
func renderFixture(t *testing.T, catalog, evidence string) string {
	t.Helper()
	block, err := Render([]byte(catalog), []byte(evidence))
	if err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	return block
}

func TestRenderIncludesEntryPointsAndStability(t *testing.T) {
	block := renderFixture(t, fixtureCatalog, fixtureEvidence)
	for _, want := range []string{
		beginningMarker,
		endingMarker,
		"## Public status",
		"`improve-project`",
		"`deliver-change`",
		"`resolve-defect`",
		"experimental",
		"0.1.0",
		"No verified release exists yet",
		"Pinned installation is documented from retained release evidence only",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("rendered block missing %q", want)
		}
	}
	if !strings.HasPrefix(block, beginningMarker) || !strings.HasSuffix(block, endingMarker) {
		t.Fatalf("block must be bounded by the generation markers")
	}
}

func TestCheckAcceptsCommittedBlock(t *testing.T) {
	block := renderFixture(t, fixtureCatalog, fixtureEvidence)
	files := baseFiles()
	files["README.md"] = readmeWithBlock(block)
	root := writeRepo(t, files)
	if code, errOut := runCheck(t, root); code != 0 {
		t.Fatalf("expected pass, got exit %d: %s", code, errOut)
	}
}

func TestCheckRejectsMissingMarkers(t *testing.T) {
	root := writeRepo(t, baseFiles())
	if code, errOut := runCheck(t, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	} else if !strings.Contains(errOut, "missing the public-status markers") {
		t.Fatalf("missing marker finding:\n%s", errOut)
	}
}

func TestWriteInsertsBlockAndIsIdempotent(t *testing.T) {
	root := writeRepo(t, baseFiles())
	var out, errOut bytes.Buffer
	if code := Write(root, &out, &errOut); code != 0 {
		t.Fatalf("expected Write success, got %d: %s", code, errOut.String())
	}
	content, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), beginningMarker) {
		t.Fatalf("README.md missing generated section after Write")
	}
	once := string(content)
	out.Reset()
	errOut.Reset()
	if code := Write(root, &out, &errOut); code != 0 {
		t.Fatalf("expected idempotent Write, got %d: %s", code, errOut.String())
	} else if !strings.Contains(out.String(), "already current") {
		t.Fatalf("second Write should report the section is current")
	}
	content, err = os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != once {
		t.Fatalf("second Write changed the file")
	}
	if code, errOut := runCheck(t, root); code != 0 {
		t.Fatalf("expected pass after Write, got %d: %s", code, errOut)
	}
}
