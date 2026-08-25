package check

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// catalogDocCatalog is a three-skill catalog used by the fixture documents.
const catalogDocCatalog = `catalog_version: 1
license: Apache-2.0
skills:
  - name: plan-issue
    summary: Plan a change issue.
    owner: hidekitux
    status: experimental
    agents: [codex, claude-code]
    license: Apache-2.0
    version: 0.1.0
    layer: process
    related: [implement-issue]
  - name: write-tests
    summary: Write focused tests.
    owner: hidekitux
    status: experimental
    agents: [codex, claude-code]
    license: Apache-2.0
    version: 0.1.0
    layer: fix
    related: [debug-code]
  - name: bootstrap-project
    summary: Bootstrap a project.
    owner: hidekitux
    status: experimental
    agents: [codex, claude-code]
    license: Apache-2.0
    version: 0.1.0
    layer: govern
    related: [audit-workflow-enforcement]
`

const catalogDocReadme = `# Fixture repository

The repository publishes 3 skills today and tracks 0 planned next-generation skills.

## Skill-set map

| Skill | Layer | Status |
| --- | --- | --- |
| plan-issue | process | experimental |
| write-tests | fix | experimental |
| bootstrap-project | govern | experimental |
`

const catalogDocLayers = `# Skill layers

## Skill-set mapping

| Layer | Skill | Status |
| --- | --- | --- |
| process | plan-issue | experimental |
| fix | write-tests | experimental |
| govern | bootstrap-project | experimental |
`

// catalogDocContract returns the ownership-boundary document with backticked
// skill names and no status markers.
func catalogDocContract() string {
	return "# Skill contract\n\n" +
		"## Ownership boundary\n\n" +
		"| Skill | Produces | Handoff target | Ownership boundary |\n" +
		"| --- | --- | --- | --- |\n" +
		"| `plan-issue` | Verified plan | `implement-issue` | Plans only |\n" +
		"| `write-tests` | Focused test cases | `implement-issue` | Tests only |\n" +
		"| `bootstrap-project` | Runnable foundation | change flow | Initialization only |\n"
}

// writeCatalogDocRepo scaffolds a repository with CATALOG.yml and the three
// contributor-facing documents the catalog-docs check scans.
func writeCatalogDocRepo(t *testing.T, readme, layers, contract string) string {
	t.Helper()
	root := t.TempDir()
	if err := writeNestedFile(filepath.Join(root, "CATALOG.yml"), []byte(catalogDocCatalog)); err != nil {
		t.Fatal(err)
	}
	if err := writeNestedFile(filepath.Join(root, "README.md"), []byte(readme)); err != nil {
		t.Fatal(err)
	}
	if err := writeNestedFile(filepath.Join(root, "docs", "skill-layers.md"), []byte(layers)); err != nil {
		t.Fatal(err)
	}
	if err := writeNestedFile(filepath.Join(root, "docs", "skill-contract.md"), []byte(contract)); err != nil {
		t.Fatal(err)
	}
	return root
}

func runCatalogDocs(t *testing.T, root string) (int, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := CheckCatalogDocs(root, &out, &errOut)
	return code, errOut.String()
}

func TestCatalogDocsAcceptsCurrentInventoryDocs(t *testing.T) {
	root := writeCatalogDocRepo(t, catalogDocReadme, catalogDocLayers, catalogDocContract())
	if code, errOut := runCatalogDocs(t, root); code != 0 {
		t.Fatalf("expected pass, got exit %d: %s", code, errOut)
	}
}

func TestCatalogDocsRejectsStaleCount(t *testing.T) {
	// The README still claims the old inventory count while the catalog has
	// more entries.
	stale := strings.Replace(catalogDocReadme, "publishes 3 skills today", "publishes 2 skills today", 1)
	root := writeCatalogDocRepo(t, stale, catalogDocLayers, catalogDocContract())
	if code, errOut := runCatalogDocs(t, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	} else if !strings.Contains(errOut, "README.md states 2 published skills, but CATALOG.yml lists 3") {
		t.Fatalf("missing count-drift finding:\n%s", errOut)
	}
}

func TestCatalogDocsRejectsStaleLayer(t *testing.T) {
	// The README maps write-tests to the process layer, but the catalog
	// declares it in the fix layer.
	stale := strings.Replace(catalogDocReadme, "| write-tests | fix | experimental |", "| write-tests | process | experimental |", 1)
	root := writeCatalogDocRepo(t, stale, catalogDocLayers, catalogDocContract())
	if code, errOut := runCatalogDocs(t, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	} else if !strings.Contains(errOut, `write-tests is listed in layer "process", but CATALOG.yml declares layer "fix"`) {
		t.Fatalf("missing layer-drift finding:\n%s", errOut)
	}
}

func TestCatalogDocsRejectsStaleStatus(t *testing.T) {
	// The ownership table still marks write-tests as planned although the
	// catalog publishes it.
	stale := strings.Replace(catalogDocContract(), "| `write-tests` |", "| `write-tests` (planned) |", 1)
	root := writeCatalogDocRepo(t, catalogDocReadme, catalogDocLayers, stale)
	if code, errOut := runCatalogDocs(t, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	} else if !strings.Contains(errOut, "write-tests is described as planned, but it exists in CATALOG.yml") {
		t.Fatalf("missing status-drift finding:\n%s", errOut)
	}
}

func TestCatalogDocsRejectsMissingCatalogSkill(t *testing.T) {
	// The README drops a current skill from the skill-set map, so the
	// documented inventory no longer covers the catalog.
	stale := strings.Replace(catalogDocReadme, "| bootstrap-project | govern | experimental |\n", "", 1)
	root := writeCatalogDocRepo(t, stale, catalogDocLayers, catalogDocContract())
	if code, errOut := runCatalogDocs(t, root); code != 1 {
		t.Fatalf("expected failure, got exit %d", code)
	} else if !strings.Contains(errOut, "bootstrap-project is missing from the skill-set map table") {
		t.Fatalf("missing inventory-drift finding:\n%s", errOut)
	}
}
