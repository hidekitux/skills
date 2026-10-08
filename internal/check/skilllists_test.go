package check

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const skillListCatalog = `skills:
  - name: plan-issue
    status: experimental
    layer: process
  - name: write-tests
    status: stable
    layer: fix
  - name: analyze-project
    status: experimental
    layer: analyze
`

func writeSkillListRepo(t *testing.T, catalog string, skillDirs []string) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		if err := writeNestedFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	write("CATALOG.yml", catalog)
	for _, dir := range skillDirs {
		write("skills/"+dir+"/SKILL.md", "---\nname: "+filepath.Base(dir)+"\n---\n")
	}
	for _, document := range skillListDocuments {
		write(filepath.ToSlash(document), "# Document\n\n"+skillListBegin+"\n"+skillListEnd+"\n\nTrailing prose.\n")
	}
	return root
}

func runSkillLists(t *testing.T, root string, write bool) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	var code int
	if write {
		code = WriteSkillLists(root, &out, &errOut)
	} else {
		code = CheckSkillLists(root, &out, &errOut)
	}
	return code, out.String(), errOut.String()
}

func TestSkillListsWriteThenCheckIsCurrentAndOrdered(t *testing.T) {
	root := writeSkillListRepo(t, skillListCatalog, []string{"process/plan-issue", "fix/write-tests", "analyze/analyze-project"})
	if code, _, errOut := runSkillLists(t, root, false); code != 1 || !strings.Contains(errOut, "README.md skill list is stale") {
		t.Fatalf("expected stale finding before generation, got %d: %s", code, errOut)
	}
	if code, _, errOut := runSkillLists(t, root, true); code != 0 {
		t.Fatalf("write failed: %d %s", code, errOut)
	}
	if code, _, errOut := runSkillLists(t, root, false); code != 0 {
		t.Fatalf("expected current lists after generation, got %d: %s", code, errOut)
	}
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(readme)
	if !strings.Contains(text, "The repository publishes 3 skills: 3 workflow skills and 0 technology skills.") {
		t.Fatalf("missing count sentence:\n%s", text)
	}
	process, analyze, fix := strings.Index(text, "`plan-issue`"), strings.Index(text, "`analyze-project`"), strings.Index(text, "`write-tests`")
	if !(process < analyze && analyze < fix) {
		t.Fatalf("expected process, analyze, fix order:\n%s", text)
	}
	if !strings.HasSuffix(text, skillListEnd+"\n\nTrailing prose.\n") {
		t.Fatalf("prose after the block changed:\n%s", text)
	}
}

func TestSkillListsRejectsDriftAfterNewWorkflowSkill(t *testing.T) {
	root := writeSkillListRepo(t, skillListCatalog, []string{"process/plan-issue", "fix/write-tests", "analyze/analyze-project"})
	runSkillLists(t, root, true)
	catalog := skillListCatalog + "  - name: create-pr\n    status: experimental\n    layer: process\n"
	if err := writeNestedFile(filepath.Join(root, "CATALOG.yml"), []byte(catalog)); err != nil {
		t.Fatal(err)
	}
	code, _, errOut := runSkillLists(t, root, false)
	if code != 1 || !strings.Contains(errOut, "docs/skill-layers.md skill list is stale") || !strings.Contains(errOut, "skills/README.md skill list is stale") {
		t.Fatalf("expected drift findings for every document, got %d: %s", code, errOut)
	}
}

func TestSkillListsPlacesTechnologySkillUnderItsCategory(t *testing.T) {
	catalog := skillListCatalog + "  - name: develop-go\n    status: experimental\n    kind: stack\n"
	root := writeSkillListRepo(t, catalog, []string{"process/plan-issue", "fix/write-tests", "analyze/analyze-project", "language/develop-go"})
	if code, _, errOut := runSkillLists(t, root, true); code != 0 {
		t.Fatalf("write failed: %d %s", code, errOut)
	}
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(readme)
	if !strings.Contains(text, "| `develop-go` | language | experimental |") || !strings.Contains(text, "4 skills: 3 workflow skills and 1 technology skill.") {
		t.Fatalf("missing technology skill row or count:\n%s", text)
	}
	if strings.Index(text, "`write-tests`") > strings.Index(text, "`develop-go`") {
		t.Fatalf("technology skills must follow workflow layers:\n%s", text)
	}
}

func TestSkillListsRejectsMissingMarkers(t *testing.T) {
	root := writeSkillListRepo(t, skillListCatalog, nil)
	if err := writeNestedFile(filepath.Join(root, "skills", "README.md"), []byte("# No markers\n")); err != nil {
		t.Fatal(err)
	}
	if code, _, errOut := runSkillLists(t, root, false); code != 1 || !strings.Contains(errOut, "skills/README.md is missing the skill-list markers") {
		t.Fatalf("expected missing-marker finding, got %d: %s", code, errOut)
	}
	if code, _, errOut := runSkillLists(t, root, true); code != 1 || !strings.Contains(errOut, "missing the skill-list markers") {
		t.Fatalf("expected write to refuse missing markers, got %d: %s", code, errOut)
	}
	readme, err := os.ReadFile(filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(readme), "publishes") {
		t.Fatalf("write must not change any document when one lacks markers:\n%s", readme)
	}
}
