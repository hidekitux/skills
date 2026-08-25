package validate

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// repoRoot locates the repository root (the directory containing go.mod) from
// this package's file location so tests can reuse the canonical LICENSE and
// NOTICE fixtures.
func repoRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func readRootFile(t *testing.T, rel string) string {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(repoRoot(), rel))
	if err != nil {
		t.Fatalf("cannot read %s: %v", rel, err)
	}
	return string(content)
}

// validSkillMarkdown returns SKILL.md content that satisfies the repository
// authoring checks (frontmatter name, description, license, and a Todo List
// with completion and handoff guidance).
func validSkillMarkdown(name string) string {
	return "---\n" +
		"name: " + name + "\n" +
		"description: A representative test skill that completes its task and handoff deterministically.\n" +
		"license: Apache-2.0\n" +
		"---\n\n" +
		"# " + name + "\n\n" +
		"## Todo List\n\n" +
		"1. **in progress:** complete the task.\n" +
		"2. handoff when complete.\n"
}

func catalogEntry(name, path string, hasPath bool) map[string]any {
	entry := map[string]any{
		"name":    name,
		"summary": "summary",
		"owner":   "hidekitux",
		"status":  "experimental",
		"license": "Apache-2.0",
		"version": "0.1.0",
		"layer":   "process",
	}
	if hasPath {
		entry["path"] = path
	}
	return entry
}

func renderCatalog(t *testing.T, entries []map[string]any) string {
	t.Helper()
	doc := map[string]any{
		"catalog_version": 1,
		"license":         "Apache-2.0",
		"skills":          entries,
	}
	b, err := yaml.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// scaffoldRepo builds a minimal repository shape valid under CheckRepository:
// a flat and/or namespaced skill set under skills/, a matching CATALOG.yml, the
// canonical LICENSE/NOTICE, and a specs/ directory.
func scaffoldRepo(t *testing.T, skillDirs []string, entries []map[string]any) string {
	t.Helper()
	root := t.TempDir()
	writeVFile(t, root, "LICENSE", readRootFile(t, "LICENSE"))
	writeVFile(t, root, "NOTICE", readRootFile(t, "NOTICE"))
	if err := os.MkdirAll(filepath.Join(root, "specs"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, dir := range skillDirs {
		rel := filepath.Join("skills", filepath.FromSlash(dir), "SKILL.md")
		writeVFile(t, root, rel, validSkillMarkdown(filepath.Base(filepath.FromSlash(dir))))
	}
	writeVFile(t, root, "CATALOG.yml", renderCatalog(t, entries))
	return root
}

func runRepoCheck(t *testing.T, root string) (int, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := CheckRepository(root, &out, &errOut)
	return code, errOut.String()
}

func TestCheckRepositoryAcceptsFlatAndNestedSkills(t *testing.T) {
	root := scaffoldRepo(t,
		[]string{"plan-issue", "skills/refactor-code"},
		[]map[string]any{catalogEntry("plan-issue", "", false), catalogEntry("refactor-code", "", false)},
	)
	if code, errOut := runRepoCheck(t, root); code != 0 {
		t.Fatalf("expected pass, got exit %d: %s", code, errOut)
	}
}

func TestCheckRepositoryRequiresCatalogPathForDuplicateName(t *testing.T) {
	root := scaffoldRepo(t,
		[]string{"a/dup", "b/dup"},
		[]map[string]any{catalogEntry("dup", "", false)},
	)
	code, errOut := runRepoCheck(t, root)
	if code != 1 || !strings.Contains(errOut, "ambiguous") {
		t.Fatalf("expected ambiguous-name failure, got exit %d: %s", code, errOut)
	}
}

func TestCheckRepositoryResolvesDuplicateNameWithCatalogPath(t *testing.T) {
	root := scaffoldRepo(t,
		[]string{"a/dup", "b/dup"},
		[]map[string]any{catalogEntry("dup", "skills/a/dup", true)},
	)
	if code, errOut := runRepoCheck(t, root); code != 0 {
		t.Fatalf("expected pass with a resolved path, got exit %d: %s", code, errOut)
	}
}

func TestCheckRepositoryRejectsInvalidCatalogPath(t *testing.T) {
	root := scaffoldRepo(t,
		[]string{"a/dup", "b/dup"},
		[]map[string]any{catalogEntry("dup", "skills/does-not-exist", true)},
	)
	code, errOut := runRepoCheck(t, root)
	if code != 1 || !strings.Contains(errOut, "does not resolve") {
		t.Fatalf("expected invalid-path failure, got exit %d: %s", code, errOut)
	}
}

func TestCheckRepositoryRejectsUnnecessaryCatalogPath(t *testing.T) {
	root := scaffoldRepo(t,
		[]string{"plan-issue"},
		[]map[string]any{catalogEntry("plan-issue", "skills/plan-issue", true)},
	)
	code, errOut := runRepoCheck(t, root)
	if code != 1 || !strings.Contains(errOut, "unnecessary") {
		t.Fatalf("expected unnecessary-path failure, got exit %d: %s", code, errOut)
	}
}

func TestSkillNamesIncludesNestedSkill(t *testing.T) {
	root := scaffoldRepo(t,
		[]string{"plan-issue", "skills/refactor-code"},
		[]map[string]any{catalogEntry("plan-issue", "", false), catalogEntry("refactor-code", "", false)},
	)
	names := skillNames(root)
	if len(names) != 2 {
		t.Fatalf("expected 2 names, got %v", names)
	}
	if names[0] != "plan-issue" || names[1] != "refactor-code" {
		t.Fatalf("unexpected names: %v", names)
	}
}
