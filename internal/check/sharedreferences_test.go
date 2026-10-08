package check

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sharedReferenceSource = "# Persistent prose\n\n- Use plain prose.\n"

func writeSharedReferenceRepo(t *testing.T, copies map[string]string) string {
	t.Helper()
	root := t.TempDir()
	write := func(rel, content string) {
		if err := writeNestedFile(filepath.Join(root, filepath.FromSlash(rel)), []byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	write("shared/references/persistent-prose.md", sharedReferenceSource)
	write("skills/process/create-pr/SKILL.md", "Read [persistent prose](references/persistent-prose.md).\n")
	write("skills/process/merge-pr/SKILL.md", "No shared reference here.\n")
	for rel, content := range copies {
		write(rel, content)
	}
	return root
}

func runSharedReferences(t *testing.T, root string) (int, string, string) {
	t.Helper()
	var out, errOut bytes.Buffer
	code := CheckSharedReferences(root, &out, &errOut)
	return code, out.String(), errOut.String()
}

func TestSharedReferencesAcceptsMatchingCopy(t *testing.T) {
	root := writeSharedReferenceRepo(t, map[string]string{"skills/process/create-pr/references/persistent-prose.md": sharedReferenceSource})
	if code, out, errOut := runSharedReferences(t, root); code != 0 || !strings.Contains(out, "1 copy(ies) of 1 shared reference(s)") {
		t.Fatalf("expected pass with one copy, got %d: %s%s", code, out, errOut)
	}
}

func TestSharedReferencesRejectsDriftedCopy(t *testing.T) {
	root := writeSharedReferenceRepo(t, map[string]string{"skills/process/create-pr/references/persistent-prose.md": sharedReferenceSource + "- An extra local rule.\n"})
	code, _, errOut := runSharedReferences(t, root)
	if code != 1 || !strings.Contains(errOut, "skills/process/create-pr/references/persistent-prose.md differs from shared/references/persistent-prose.md; run cp shared/references/persistent-prose.md skills/process/create-pr/references/persistent-prose.md") {
		t.Fatalf("expected drift finding with the repair command, got %d: %s", code, errOut)
	}
}

func TestSharedReferencesRejectsMissingCopy(t *testing.T) {
	root := writeSharedReferenceRepo(t, nil)
	code, _, errOut := runSharedReferences(t, root)
	if code != 1 || !strings.Contains(errOut, "skills/process/create-pr/references/persistent-prose.md is missing") {
		t.Fatalf("expected missing-copy finding, got %d: %s", code, errOut)
	}
	if strings.Contains(errOut, "merge-pr") {
		t.Fatalf("a skill that does not link the reference must not need a copy: %s", errOut)
	}
}

func TestSharedReferencesSkipsWithoutSourceDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "skills"), 0o755); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := runSharedReferences(t, root); code != 0 || !strings.Contains(out, "skipped") {
		t.Fatalf("expected skip without shared/references, got %d: %s", code, out)
	}
}

func TestSharedReferencesRecognizesLinkForms(t *testing.T) {
	cases := map[string]bool{
		"Read [p](references/persistent-prose.md).":                   true,
		"Read [p](./references/persistent-prose.md).":                 true,
		"Read [p](references/persistent-prose.md#rules).":             true,
		`Read [p](references/persistent-prose.md "Rules").`:           true,
		"Read [p](references/persistent-prose.md 'Rules').":           true,
		"Read [p](references/persistent-prose.md (Rules)).":           true,
		"Read [p][rules].\n\n[rules]: references/persistent-prose.md": true,
		"Read <references/persistent-prose.md>.":                      true,
		"Read references/persistent-prose.md.":                        true,
		"Read [p](references/persistent-prose.md.bak).":               false,
		"Read [p](other/references/persistent-prose.md).":             false,
		"Read [p](references/persistent-prose.mdx).":                  false,
	}
	for text, want := range cases {
		if got := linksReference(text, "persistent-prose.md"); got != want {
			t.Errorf("linksReference(%q) = %t, want %t", text, got, want)
		}
	}
}

func TestSharedReferencesRejectsUnlinkedCopy(t *testing.T) {
	root := writeSharedReferenceRepo(t, map[string]string{
		"skills/process/create-pr/references/persistent-prose.md": sharedReferenceSource,
		"skills/process/merge-pr/references/persistent-prose.md":  sharedReferenceSource,
	})
	code, _, errOut := runSharedReferences(t, root)
	if code != 1 || !strings.Contains(errOut, "skills/process/merge-pr/references/persistent-prose.md is not named in skills/process/merge-pr/SKILL.md") {
		t.Fatalf("expected unlinked-copy finding, got %d: %s", code, errOut)
	}
}
