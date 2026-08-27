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

func TestCheckRejectsStaleEntryPointStatus(t *testing.T) {
	// The committed block still shows improve-project as experimental, but the
	// catalog marks it stable: the derived documentation is stale and must
	// fail validation (Issue 177 validation, acceptance criterion 2).
	block := renderFixture(t, fixtureCatalog, fixtureEvidence)
	catalog := strings.Replace(fixtureCatalog, "    status: experimental", "    status: stable", 1)
	files := baseFiles()
	files["CATALOG.yml"] = catalog
	files["README.md"] = readmeWithBlock(block)
	root := writeRepo(t, files)
	if code, errOut := runCheck(t, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	} else if !strings.Contains(errOut, "public status section is stale") {
		t.Fatalf("missing stale finding:\n%s", errOut)
	}
}

func TestCheckRejectsStaleReleaseEvidence(t *testing.T) {
	// The committed block documents that no release exists, but the retained
	// evidence now records one: stale derived documentation must fail.
	block := renderFixture(t, fixtureCatalog, fixtureEvidence)
	evidence := "released: true\ntag: v0.1.0\nrelease_url: https://github.com/hidekitux/skills/releases/tag/v0.1.0\ncommit: abc123\n"
	files := baseFiles()
	files["docs/release-evidence.yml"] = evidence
	files["README.md"] = readmeWithBlock(block)
	root := writeRepo(t, files)
	if code, errOut := runCheck(t, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	} else if !strings.Contains(errOut, "public status section is stale") {
		t.Fatalf("missing stale finding:\n%s", errOut)
	}
}

func TestCheckRejectsHandEditedBlock(t *testing.T) {
	// A manual edit inside the generated block must be rejected instead of
	// silently repaired.
	block := renderFixture(t, fixtureCatalog, fixtureEvidence)
	edited := strings.Replace(block, "experimental", "stable", 1)
	files := baseFiles()
	files["README.md"] = readmeWithBlock(edited)
	root := writeRepo(t, files)
	if code, errOut := runCheck(t, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	} else if !strings.Contains(errOut, "public status section is stale") {
		t.Fatalf("missing stale finding:\n%s", errOut)
	}
}

func TestCheckRejectsCatalogVersionReleaseMismatch(t *testing.T) {
	// The evidence records a verified v0.2.0 release while every catalog
	// version is 0.1.0: the authorities cannot support the documented claim.
	evidence := "released: true\ntag: v0.2.0\nrelease_url: https://github.com/hidekitux/skills/releases/tag/v0.2.0\ncommit: abc123\n"
	files := baseFiles()
	files["docs/release-evidence.yml"] = evidence
	files["README.md"] = readmeWithBlock(beginningMarker + "\n" + endingMarker)
	root := writeRepo(t, files)
	if code, errOut := runCheck(t, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	} else if !strings.Contains(errOut, "does not match catalog version") {
		t.Fatalf("missing version-mismatch finding:\n%s", errOut)
	}
}

func TestCheckRejectsPartialReleasedEvidence(t *testing.T) {
	// A released record missing its commit cannot back a pinned-installation
	// claim.
	evidence := "released: true\ntag: v0.1.0\nrelease_url: https://github.com/hidekitux/skills/releases/tag/v0.1.0\ncommit: \"\"\n"
	files := baseFiles()
	files["docs/release-evidence.yml"] = evidence
	root := writeRepo(t, files)
	if code, errOut := runCheck(t, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	} else if !strings.Contains(errOut, "tag, release_url, or commit is empty") {
		t.Fatalf("missing partial-evidence finding:\n%s", errOut)
	}
}

func TestRenderRejectsMissingEntryPoint(t *testing.T) {
	// Removing an entry point from the catalog must stop generation rather
	// than silently dropping it from the public documentation.
	removed := "  - name: resolve-defect\n" +
		"    summary: Resolve a verified defect end to end.\n" +
		"    owner: hidekitux\n" +
		"    status: experimental\n" +
		"    agents: [codex, claude-code]\n" +
		"    license: Apache-2.0\n" +
		"    version: 0.1.0\n" +
		"    layer: fix\n" +
		"    related: [debug-code, create-issue]\n"
	catalog := strings.Replace(fixtureCatalog, removed, "", 1)
	if _, err := Render([]byte(catalog), []byte(fixtureEvidence)); err == nil || !strings.Contains(err.Error(), "resolve-defect is missing") {
		t.Fatalf("expected missing entry-point error, got %v", err)
	}
}

func TestRenderReleasedIncludesPinnedInstallationCommands(t *testing.T) {
	// The released branch documents the pinned installation commands from the
	// retained tag for both hosts, and never claims the absence of a release.
	evidence := "released: true\ntag: v0.1.0\nrelease_url: https://github.com/hidekitux/skills/releases/tag/v0.1.0\ncommit: abc123\n"
	block := renderFixture(t, fixtureCatalog, evidence)
	for _, want := range []string{
		"Verified release: `v0.1.0`",
		"gh skill install hidekitux/skills <skill>@v0.1.0 --agent codex --scope user",
		"gh skill install hidekitux/skills <skill>@v0.1.0 --agent claude-code --scope user",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("released block missing %q", want)
		}
	}
	if strings.Contains(block, "No verified release exists yet") {
		t.Errorf("released block must not claim the absence of a release")
	}
}
